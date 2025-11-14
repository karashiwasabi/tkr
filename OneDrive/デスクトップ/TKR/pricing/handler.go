// C:\Users\wasab\OneDrive\デスクトップ\TKR\pricing\handler.go
package pricing

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"tkr/database"
	"tkr/mappers"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

// writeJsonError はエラーレスポンスを返します
func writeJsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}

// GetExportDataHandler は見積依頼CSVを作成します (WASABI
func GetExportDataHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wholesalerName := r.URL.Query().Get("wholesalerName")
		unregisteredOnlyStr := r.URL.Query().Get("unregisteredOnly")
		unregisteredOnly := unregisteredOnlyStr == "true"

		if wholesalerName == "" {
			writeJsonError(w, "Wholesaler name is required", http.StatusBadRequest)
			return
		}

		allMasters, err := database.GetAllProductMasters(db) // TKR
		if err != nil {
			writeJsonError(w, "Failed to get products for export", http.StatusInternalServerError)
			return
		}

		var mastersToProcess []*model.ProductMaster
		for _, p := range allMasters {
			// TKRでは仮マスター(MA2J)を除外
			if strings.HasPrefix(p.ProductCode, "MA2J") {
				continue
			}
			mastersToProcess = append(mastersToProcess, p)
		}

		var dataToExport []*model.ProductMaster
		if unregisteredOnly {
			for _, p := range mastersToProcess {
				if p.SupplierWholesale == "" { // 採用卸が未設定
					dataToExport = append(dataToExport, p)
				}
			}
			if len(dataToExport) == 0 {
				writeJsonError(w, "未登録の品目が見つかりませんでした。", http.StatusNotFound)
				return
			}
		} else {
			dataToExport = mastersToProcess
		}

		dateStr := r.URL.Query().Get("date")
		fileType := "ALL"
		if unregisteredOnly {
			fileType = "UNREGISTERED"
		}
		fileName := fmt.Sprintf("価格見積依頼_%s_%s_%s.csv", wholesalerName, fileType, dateStr)
		fileName = url.PathEscape(fileName) // ファイル名をURLエンコード

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+fileName)
		w.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM

		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()

		headers := []string{"product_code", "product_name", "maker_name", "package_spec", "purchase_price"}
		if err := csvWriter.Write(headers); err != nil {
			log.Printf("Failed to write CSV header: %v", err)
		}

		for _, m := range dataToExport {
			// mappers.ToProductMasterView を使って包装仕様を取得
			view := mappers.ToProductMasterView(m) // TKR
			formattedSpec := view.FormattedPackageSpec

			record := []string{
				// ▼▼▼【ここを修正】 "=%q" から "%q" に変更 ▼▼▼
				fmt.Sprintf("%q", m.ProductCode),
				// ▲▲▲【修正ここまで】▲▲▲
				m.ProductName,
				m.MakerName,
				formattedSpec,
				"", // purchase_price is left empty
			}
			if err := csvWriter.Write(record); err != nil {
				log.Printf("Failed to write product row to CSV (JAN: %s): %v", m.ProductCode, err)
			}
		}
	}
}

// QuoteDataWithSpec は価格比較画面用のデータ構造です (WASABI
type QuoteDataWithSpec struct {
	model.ProductMaster
	FormattedPackageSpec string             `json:"formattedPackageSpec"`
	Quotes               map[string]float64 `json:"quotes"`
}

// UploadResponse はアップロード結果のデータ構造です (WASABI
type UploadResponse struct {
	ProductData     []QuoteDataWithSpec `json:"productData"`
	WholesalerOrder []string            `json:"wholesalerOrder"`
}

// UploadQuotesHandler は見積CSVをDBに保存し、比較データを返します (WASABI 修正)
func UploadQuotesHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseMultipartForm(32 << 20); err != nil {
			writeJsonError(w, "File upload error", http.StatusBadRequest)
			return
		}

		wholesalerFiles := r.MultipartForm.File["files"]
		wholesalerNames := r.MultipartForm.Value["wholesalerNames"]

		if len(wholesalerFiles) != len(wholesalerNames) {
			writeJsonError(w, "File and wholesaler name mismatch", http.StatusBadRequest)
			return
		}

		// TKRの卸マップを取得
		wholesalerMasterMap, err := database.GetWholesalerMap(db)
		if err != nil {
			writeJsonError(w, "卸マスタの取得に失敗しました: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// 卸名 -> 卸コード の逆引きマップを作成
		wholesalerReverseMap := make(map[string]string)
		for code, name := range wholesalerMasterMap {
			wholesalerReverseMap[name] = code
		}

		tx, err := db.Beginx()
		if err != nil {
			writeJsonError(w, "トランザクションの開始に失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		quoteDate := time.Now().Format("20060102")
		var wholesalerOrder []string // 画面表示用の卸名リスト

		for i, fileHeader := range wholesalerFiles {
			wholesalerName := wholesalerNames[i]
			wholesalerOrder = append(wholesalerOrder, wholesalerName) // 順番を保持

			wholesalerCode, ok := wholesalerReverseMap[wholesalerName]
			if !ok {
				log.Printf("WARN: Wholesaler name '%s' not found in master. Skipping file.", wholesalerName)
				continue // マスタにない卸名はスキップ
			}

			if err := processQuoteFile(tx, fileHeader, wholesalerCode, quoteDate); err != nil {
				writeJsonError(w, fmt.Sprintf("ファイル '%s' の処理に失敗: %v", fileHeader.Filename, err), http.StatusInternalServerError)
				return
			}
		}

		// データベースから全マスターと保存された見積データを取得
		allMasters, err := database.GetAllProductMasters(db)
		if err != nil {
			writeJsonError(w, "Failed to get all product masters", http.StatusInternalServerError)
			return
		}
		allQuotes, err := database.GetAllProductQuotes(tx) // 新設
		if err != nil {
			writeJsonError(w, "Failed to get product quotes", http.StatusInternalServerError)
			return
		}

		var responseData []QuoteDataWithSpec
		for _, master := range allMasters {
			view := mappers.ToProductMasterView(master) // TKR
			formattedSpec := view.FormattedPackageSpec

			// 見積データをアタッチ
			quotesForThisProduct := make(map[string]float64)
			if quotes, ok := allQuotes[master.ProductCode]; ok {
				for wCode, price := range quotes {
					if wName, ok := wholesalerMasterMap[wCode]; ok {
						quotesForThisProduct[wName] = price // 卸コード -> 卸名 に変換して格納
					}
				}
			}

			responseData = append(responseData, QuoteDataWithSpec{
				ProductMaster:        *master,
				FormattedPackageSpec: formattedSpec,
				Quotes:               quotesForThisProduct,
			})
		}

		if err := tx.Commit(); err != nil {
			writeJsonError(w, "トランザクションのコミットに失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}

		finalResponse := UploadResponse{
			ProductData:     responseData,
			WholesalerOrder: wholesalerOrder,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(finalResponse)
	}
}

// processQuoteFile は単一の見積CSVファイルを解析しDBに保存します
func processQuoteFile(tx *sqlx.Tx, fileHeader *multipart.FileHeader, wholesalerCode string, quoteDate string) error {
	file, err := fileHeader.Open()
	if err != nil {
		return fmt.Errorf("could not open uploaded file: %w", err)
	}
	defer file.Close()

	// BOMスキップ
	br := bufio.NewReader(file)
	bom, err := br.Peek(3)
	if err == nil && bom[0] == 0xef && bom[1] == 0xbb && bom[2] == 0xbf {
		br.Discard(3)
	}
	csvReader := csv.NewReader(br)
	csvReader.LazyQuotes = true
	rows, err := csvReader.ReadAll()
	if err != nil || len(rows) < 1 {
		return fmt.Errorf("could not parse CSV: %w", err)
	}

	header := rows[0]
	codeIndex, priceIndex := -1, -1
	for i, h := range header {
		if h == "product_code" {
			codeIndex = i
		}
		if h == "purchase_price" {
			priceIndex = i
		}
	}
	if codeIndex == -1 || priceIndex == -1 {
		return fmt.Errorf("required columns (product_code, purchase_price) not found")
	}

	var quotes []model.ProductQuote
	for _, row := range rows[1:] {
		if len(row) <= codeIndex || len(row) <= priceIndex {
			continue
		}
		productCode := strings.Trim(strings.TrimSpace(row[codeIndex]), `="`)
		priceStr := row[priceIndex]
		if productCode == "" || priceStr == "" {
			continue
		}
		price, err := strconv.ParseFloat(priceStr, 64)
		if err != nil {
			continue
		}
		quotes = append(quotes, model.ProductQuote{
			ProductCode:    productCode,
			WholesalerCode: wholesalerCode,
			QuotePrice:     price,
			QuoteDate:      quoteDate,
		})
	}

	if len(quotes) > 0 {
		if err := database.UpsertProductQuotesInTx(tx, quotes); err != nil { // 新設
			return fmt.Errorf("failed to save quotes to DB: %w", err)
		}
	}
	return nil
}

// BulkUpdateHandler は価格比較画面からのマスター更新を処理します (WASABI
func BulkUpdateHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload []model.PriceUpdate
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJsonError(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			writeJsonError(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		if err := database.UpdatePricesAndSuppliersInTx(tx, payload); err != nil { // TKR
			writeJsonError(w, "Failed to update prices: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			writeJsonError(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("%d件の医薬品マスターを更新しました。", len(payload)),
		})
	}
}

// GetAllMastersForPricingHandler は価格比較画面の初期表示用データを返します (WASABI 修正)
func GetAllMastersForPricingHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TKRのDB関数を呼ぶ
		allMasters, err := database.GetAllProductMasters(db)
		if err != nil {
			writeJsonError(w, "Failed to get all product masters for pricing", http.StatusInternalServerError)
			return
		}
		// TKRの見積テーブルから全見積もりを取得
		allQuotes, err := database.GetAllProductQuotes(db)
		if err != nil {
			writeJsonError(w, "Failed to get product quotes", http.StatusInternalServerError)
			return
		}
		// TKRの卸マップを取得
		wholesalerMasterMap, err := database.GetWholesalerMap(db)
		if err != nil {
			writeJsonError(w, "卸マスタの取得に失敗しました: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var wholesalerOrder []string
		for _, name := range wholesalerMasterMap {
			wholesalerOrder = append(wholesalerOrder, name)
		}
		sort.Strings(wholesalerOrder)

		responseData := make([]QuoteDataWithSpec, 0, len(allMasters))
		for _, master := range allMasters {
			view := mappers.ToProductMasterView(master) // TKR
			formattedSpec := view.FormattedPackageSpec

			// 見積データをアタッチ
			quotesForThisProduct := make(map[string]float64)
			if quotes, ok := allQuotes[master.ProductCode]; ok {
				for wCode, price := range quotes {
					if wName, ok := wholesalerMasterMap[wCode]; ok {
						quotesForThisProduct[wName] = price // 卸コード -> 卸名 に変換
					}
				}
			}

			responseData = append(responseData, QuoteDataWithSpec{
				ProductMaster:        *master,
				FormattedPackageSpec: formattedSpec,
				Quotes:               quotesForThisProduct,
			})
		}

		finalResponse := UploadResponse{
			ProductData:     responseData,
			WholesalerOrder: wholesalerOrder,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(finalResponse)
	}
}

// DirectImportHandler はバックアップCSVから納入価・卸を一括更新します (WASABI
func DirectImportHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("file")
		if err != nil {
			writeJsonError(w, "No file uploaded", http.StatusBadRequest)
			return
		}
		defer file.Close()

		br := bufio.NewReader(file)
		bom, err := br.Peek(3)
		if err == nil && bom[0] == 0xef && bom[1] == 0xbb && bom[2] == 0xbf {
			br.Discard(3)
		}
		csvReader := csv.NewReader(br)
		csvReader.LazyQuotes = true
		rows, err := csvReader.ReadAll()
		if err != nil {
			writeJsonError(w, "Failed to parse CSV file: "+err.Error(), http.StatusBadRequest)
			return
		}

		var updates []model.PriceUpdate
		for i, row := range rows {
			if i == 0 {
				continue
			}
			if len(row) < 6 {
				continue
			}
			productCode := strings.Trim(strings.TrimSpace(row[0]), `="`)
			priceStr := strings.TrimSpace(row[4])
			supplierCode := strings.TrimSpace(row[5])
			if productCode == "" || priceStr == "" || supplierCode == "" {
				continue
			}
			price, err := strconv.ParseFloat(priceStr, 64)
			if err != nil {
				continue
			}
			updates = append(updates, model.PriceUpdate{
				ProductCode:      productCode,
				NewPurchasePrice: price,
				NewSupplier:      supplierCode,
			})
		}

		if len(updates) == 0 {
			writeJsonError(w, "No valid data to import found in the file.", http.StatusBadRequest)
			return
		}

		tx, err := db.Beginx() // TKR
		if err != nil {
			writeJsonError(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		if err := database.UpdatePricesAndSuppliersInTx(tx, updates); err != nil { // TKR
			writeJsonError(w, "Failed to update prices: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			writeJsonError(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("%d件の納入価と卸情報を更新しました。", len(updates)),
		})
	}
}

// BackupExportHandler は現在の納入価・卸をCSVにバックアップします (WASABI
func BackupExportHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allMasters, err := database.GetAllProductMasters(db) // TKR
		if err != nil {
			writeJsonError(w, "Failed to get products for backup export", http.StatusInternalServerError)
			return
		}

		now := time.Now()
		fileName := fmt.Sprintf("納入価・卸バックアップ_%s.csv", now.Format("20060102_150405"))
		fileName = url.PathEscape(fileName) // ファイル名をURLエンコード

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+fileName)
		w.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM

		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()

		headers := []string{"product_code", "product_name", "maker_name", "package_spec", "purchase_price", "supplier_wholesale"}
		if err := csvWriter.Write(headers); err != nil {
			log.Printf("Failed to write CSV header: %v", err)
		}

		for _, m := range allMasters {
			view := mappers.ToProductMasterView(m) // TKR
			formattedSpec := view.FormattedPackageSpec

			record := []string{
				// ▼▼▼【ここを修正】 "=%q" から "%q" に変更 ▼▼▼
				fmt.Sprintf("%q", m.ProductCode),
				// ▲▲▲【修正ここまで】▲▲▲
				m.ProductName,
				m.MakerName,
				formattedSpec,
				strconv.FormatFloat(m.PurchasePrice, 'f', 2, 64),
				m.SupplierWholesale,
			}
			if err := csvWriter.Write(record); err != nil {
				log.Printf("Failed to write product row to CSV (JAN: %s): %v", m.ProductCode, err)
			}
		}
	}
}
