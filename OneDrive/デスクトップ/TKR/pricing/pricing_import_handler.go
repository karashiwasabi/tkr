// C:\Users\wasab\OneDrive\デスクトップ\TKR\pricing\pricing_import_handler.go
package pricing

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"
	"tkr/database"
	"tkr/mappers"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

// UploadQuotesHandler は見積CSVをDBに保存し、比較データを返します
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

		wholesalerMasterMap, err := database.GetWholesalerMap(db)
		if err != nil {
			writeJsonError(w, "卸マスタの取得に失敗しました: "+err.Error(), http.StatusInternalServerError)
			return
		}
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
		var wholesalerOrder []string

		for i, fileHeader := range wholesalerFiles {
			wholesalerName := wholesalerNames[i]
			wholesalerOrder = append(wholesalerOrder, wholesalerName)

			wholesalerCode, ok := wholesalerReverseMap[wholesalerName]
			if !ok {
				log.Printf("WARN: Wholesaler name '%s' not found in master.Skipping file.", wholesalerName)
				continue
			}

			if err := processQuoteFile(tx, fileHeader, wholesalerCode, quoteDate); err != nil {
				writeJsonError(w, fmt.Sprintf("ファイル '%s' の処理に失敗: %v", fileHeader.Filename, err), http.StatusInternalServerError)
				return
			}
		}

		allMasters, err := database.GetAllProductMasters(db)
		if err != nil {
			writeJsonError(w, "Failed to get all product masters", http.StatusInternalServerError)
			return
		}
		allQuotes, err := database.GetAllProductQuotes(tx)
		if err != nil {
			writeJsonError(w, "Failed to get product quotes", http.StatusInternalServerError)
			return
		}

		var responseData []QuoteDataWithSpec
		for _, master := range allMasters {
			view := mappers.ToProductMasterView(master)
			formattedSpec := view.FormattedPackageSpec

			quotesForThisProduct := make(map[string]float64)
			if quotes, ok := allQuotes[master.ProductCode]; ok {
				for wCode, price := range quotes {
					if wName, ok := wholesalerMasterMap[wCode]; ok {
						quotesForThisProduct[wName] = price
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

	br := bufio.NewReader(file)
	bom, err := br.Peek(3)
	if err == nil && bom[0] == 0xef && bom[1] == 0xbb && bom[2] == 0xbf {
		br.Discard(3)
	}
	csvReader := csv.NewReader(br)
	csvReader.LazyQuotes = true
	rows, err := csvReader.ReadAll()
	if err != nil ||
		len(rows) < 1 {
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
	if codeIndex == -1 ||
		priceIndex == -1 {
		return fmt.Errorf("required columns (product_code, purchase_price) not found")
	}

	var quotes []model.ProductQuote
	for _, row := range rows[1:] {
		if len(row) <= codeIndex ||
			len(row) <= priceIndex {
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
		if err := database.UpsertProductQuotesInTx(tx, quotes); err != nil {
			return fmt.Errorf("failed to save quotes to DB: %w", err)
		}
	}
	return nil
}

// DirectImportHandler はバックアップCSVから納入価・卸を一括更新します
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
		if err !=
			nil {
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
			if productCode == "" ||
				priceStr == "" || supplierCode == "" {
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

		tx, err := db.Beginx()
		if err != nil {
			writeJsonError(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		if err := database.UpdatePricesAndSuppliersInTx(tx, updates); err != nil {
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
