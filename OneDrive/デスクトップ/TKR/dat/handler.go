// C:\Users\wasab\OneDrive\デスクトップ\TKR\dat\handler.go
package dat

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"tkr/database"
	"tkr/mastermanager"
	"tkr/model"
	"tkr/parsers"
	"tkr/units"

	"github.com/jmoiron/sqlx"
)

// ▼▼▼【修正】卸名変換マップを引数に追加 ▼▼▼
func respondJSONError(w http.ResponseWriter, message string, statusCode int) {
	log.Println("Error response:", message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   message,
		"results":   []interface{}{},
		"tableHTML": renderTransactionTableHTML(nil, nil), // ★ 卸名マップ(nil)を追加
	})
}

// ▲▲▲【修正ここまで】▲▲▲

// ▼▼▼【修正】UploadDatHandler (変更なし) ▼▼▼
func UploadDatHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received DAT upload request...")

		// ★ 先に卸マスターを取得・マップ化
		wholesalerMap, err := database.GetWholesalerMap(db)
		if err != nil {
			respondJSONError(w, "卸マスターの読み込みに失敗しました。", http.StatusInternalServerError)
			return
		}

		err = r.ParseMultipartForm(32 << 20)
		if err != nil {
			respondJSONError(w, "File upload error: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer r.MultipartForm.RemoveAll()

		var processedFiles []string
		var successfullyInsertedTransactions []model.TransactionRecord
		var allResults []map[string]interface{}

		for _, fileHeader := range r.MultipartForm.File["file"] {
			log.Printf("Processing file: %s", fileHeader.Filename)
			processedFiles = append(processedFiles, fileHeader.Filename)
			fileResult := map[string]interface{}{"filename": fileHeader.Filename}
			var currentFileTransactions []model.TransactionRecord

			file, openErr := fileHeader.Open()
			if openErr != nil {
				log.Printf("Failed to open uploaded file %s: %v", fileHeader.Filename, openErr)
				fileResult["error"] = fmt.Sprintf("Failed to open file: %v", openErr)
				allResults = append(allResults, fileResult)
				continue
			}

			parsedRecords, parseErr := parsers.ParseDat(file)
			file.Close()
			if parseErr != nil {
				log.Printf("Failed to parse DAT file %s: %v", fileHeader.Filename, parseErr)
				fileResult["error"] = fmt.Sprintf("Failed to parse DAT file: %v", parseErr)
				allResults = append(allResults, fileResult)
				continue
			}
			log.Printf("Parsed %d records from %s", len(parsedRecords), fileHeader.Filename)
			fileResult["records_parsed"] = len(parsedRecords)

			tx, txErr := db.Beginx()
			if txErr != nil {
				log.Printf("Failed to start transaction for %s: %v", fileHeader.Filename, txErr)
				fileResult["error"] = fmt.Sprintf("Failed to start transaction: %v", txErr)
				allResults = append(allResults, fileResult)
				continue
			}

			var insertedCount int = 0
			for _, rec := range parsedRecords {
				key := rec.JanCode
				if key == "" || key == "0000000000000" {
					key = fmt.Sprintf("9999999999999%s", rec.ProductName)
				}

				master, err := mastermanager.FindOrCreateMaster(tx, key, rec.ProductName)
				if err != nil {
					log.Printf("Failed to find or create master for key %s (Product: %s) in file %s: %v", key, rec.ProductName, fileHeader.Filename, err)
					fileResult["error"] = fmt.Sprintf("Master creation failed for key %s: %v", key, err)
					allResults = append(allResults, fileResult)
					tx.Rollback()
					goto nextFileLoop
				}

				transaction := MapDatToTransaction(rec, master)
				if err := database.InsertTransactionRecord(tx, transaction); err != nil {
					log.Printf("Failed to insert transaction record for key %s (Product: %s) in file %s: %v", key, rec.ProductName, fileHeader.Filename, err)
					fileResult["error"] = fmt.Sprintf("Transaction insert failed for key %s: %v", key, err)
					allResults = append(allResults, fileResult)
					tx.Rollback()
					goto nextFileLoop
				}

				currentFileTransactions = append(currentFileTransactions, transaction)
				insertedCount++
			}

			if commitErr := tx.Commit(); commitErr != nil {
				log.Printf("Failed to commit transaction for %s: %v", fileHeader.Filename, commitErr)
				fileResult["error"] = fmt.Sprintf("Failed to commit transaction: %v", commitErr)
				allResults = append(allResults, fileResult)
				goto nextFileLoop
			}

			successfullyInsertedTransactions = append(successfullyInsertedTransactions, currentFileTransactions...)

			log.Printf("Successfully inserted %d records from %s", insertedCount, fileHeader.Filename)
			fileResult["success"] = true
			fileResult["records_inserted"] = insertedCount
			allResults = append(allResults, fileResult)

		nextFileLoop:
			continue
		}

		w.Header().Set("Content-Type", "application/json")
		// ★ 卸名マップを渡す
		htmlString := renderTransactionTableHTML(successfullyInsertedTransactions, wholesalerMap)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   fmt.Sprintf("Processed %d DAT file(s). See results for details.", len(processedFiles)),
			"results":   allResults,
			"tableHTML": htmlString,
		})
		log.Println("Finished DAT upload request.")
	}
}

// ▲▲▲【修正ここまで】▲▲▲

// ▼▼▼【修正】SearchDatHandler (変更なし) ▼▼▼
func SearchDatHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondJSONError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		// ★ 先に卸マスターを取得・マップ化
		wholesalerMap, err := database.GetWholesalerMap(db)
		if err != nil {
			respondJSONError(w, "卸マスターの読み込みに失敗しました。", http.StatusInternalServerError)
			return
		}

		barcode := r.URL.Query().Get("barcode")
		log.Printf("Received DAT search request... Barcode: [%s]", barcode)

		if barcode == "" {
			respondJSONError(w, "バーコードを入力してください。", http.StatusBadRequest)
			return
		}

		gs1Result, parseErr := parsers.ParseGS1_128(barcode)
		if parseErr != nil {
			log.Printf("Error parsing GS1-128 barcode [%s]: %v", barcode, parseErr)
			respondJSONError(w, fmt.Sprintf("バーコード解析エラー: %v", parseErr), http.StatusBadRequest)
			return
		}

		gtin14 := gs1Result.JanCode
		if gtin14 == "" {
			respondJSONError(w, "バーコードから(01)GTINが抽出できませんでした。", http.StatusBadRequest)
			return
		}

		master, err := database.GetProductMasterByGs1Code(db, gtin14)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Printf("Product master not found for GS1_CODE: %s", gtin14)
				respondJSONError(w, "GS1コードに対応するマスターが見つかりません。", http.StatusNotFound)
			} else {
				log.Printf("Error searching product master by GS1_CODE %s: %v", gtin14, err)
				respondJSONError(w, "マスター検索中にデータベースエラーが発生しました。", http.StatusInternalServerError)
			}
			return
		}

		productCode := master.ProductCode

		expiryYYMMDD := gs1Result.ExpiryDate
		expiryYYMM := ""
		if len(expiryYYMMDD) == 6 {
			expiryYYMM = expiryYYMMDD[:4]
		}

		log.Printf("Parsed GS1: GTIN(14)='%s' -> ProductCode(JAN)='%s', Expiry(6)='%s', Expiry(4)='%s', Lot='%s'",
			gtin14, productCode, expiryYYMMDD, expiryYYMM, gs1Result.LotNumber)

		transactions, err := database.SearchTransactions(db, productCode, expiryYYMMDD, expiryYYMM, gs1Result.LotNumber)

		if err != nil {
			log.Printf("Error searching transactions: %v", err)
			respondJSONError(w, "トランザクション検索中にエラーが発生しました。", http.StatusInternalServerError)
			return
		}

		log.Printf("Found %d transactions matching criteria.", len(transactions))

		// ★ 卸名マップを渡す
		htmlString := renderTransactionTableHTML(transactions, wholesalerMap)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   fmt.Sprintf("%d 件のデータが見つかりました。", len(transactions)),
			"tableHTML": htmlString,
		})
	}
}

// ▲▲▲【修正ここまで】▲▲▲

func MapDatToTransaction(dat model.DatRecord, master *model.ProductMaster) model.TransactionRecord {
	// ▼▼▼【ここから修正】製品名と包装仕様の生成ロジックをWASABIに合わせる (units.ResolveName と 数量計算を追加) ▼▼▼

	// 1. 製品名と規格を連結
	productNameWithSpec := master.ProductName
	if master.Specification != "" {
		productNameWithSpec = master.ProductName + " " + master.Specification
	}

	// 2. 数量の計算 (DATの個数から計算)
	var yjQuantity, janQuantity float64
	if master.YjPackUnitQty > 0 {
		yjQuantity = dat.DatQuantity * master.YjPackUnitQty
	}
	if master.JanPackUnitQty > 0 {
		janQuantity = dat.DatQuantity * master.JanPackUnitQty
	}

	// 3. 詳細な包装仕様を生成 (units.ResolveName を使用)
	yjUnitName := units.ResolveName(master.YjUnitName)
	packageSpec := fmt.Sprintf("%s %g%s", master.PackageForm, master.YjPackUnitQty, yjUnitName)

	janUnitCodeStr := fmt.Sprintf("%d", master.JanUnitCode)
	var janUnitName string

	if master.JanUnitCode == 0 {
		janUnitName = yjUnitName
	} else {
		janUnitName = units.ResolveName(janUnitCodeStr)
	}

	if master.JanPackInnerQty > 0 && master.JanPackUnitQty > 0 {
		packageSpec += fmt.Sprintf(" (%g%s×%g%s)",
			master.JanPackInnerQty,
			yjUnitName,
			master.JanPackUnitQty,
			janUnitName,
		)
	}

	// 4. MAフラグの設定
	var processFlagMA string
	if master.Origin == "JCSHMS" {
		processFlagMA = "COMPLETE"
	} else {
		processFlagMA = "PROVISIONAL"
	}

	// ▲▲▲【修正ここまで】▲▲▲

	return model.TransactionRecord{
		TransactionDate:     dat.Date,
		ClientCode:          dat.ClientCode,
		ReceiptNumber:       dat.ReceiptNumber,
		LineNumber:          dat.LineNumber,
		Flag:                dat.Flag,
		JanCode:             master.ProductCode,
		YjCode:              master.YjCode,
		ProductName:         productNameWithSpec, // ★ 修正した製品名を使用
		KanaName:            master.KanaName,
		UsageClassification: master.UsageClassification,
		PackageForm:         master.PackageForm,
		PackageSpec:         packageSpec, // ★ 修正した包装仕様を使用
		MakerName:           master.MakerName,
		DatQuantity:         dat.DatQuantity, // DATファイルからの生データ
		JanPackInnerQty:     master.JanPackInnerQty,
		JanQuantity:         janQuantity, // ★ 数量計算を反映
		JanPackUnitQty:      master.JanPackUnitQty,
		JanUnitName:         janUnitName, // ★ 修正したJAN単位名を使用
		JanUnitCode:         janUnitCodeStr,
		YjQuantity:          yjQuantity, // ★ 数量計算を反映
		YjPackUnitQty:       master.YjPackUnitQty,
		YjUnitName:          yjUnitName, // ★ 解決済みのYJ単位名を使用
		UnitPrice:           dat.UnitPrice,
		PurchasePrice:       master.PurchasePrice,
		SupplierWholesale:   master.SupplierWholesale,
		Subtotal:            dat.Subtotal,
		TaxAmount:           0,
		TaxRate:             0,
		ExpiryDate:          dat.ExpiryDate,
		LotNumber:           dat.LotNumber,
		FlagPoison:          master.FlagPoison,
		FlagDeleterious:     master.FlagDeleterious,
		FlagNarcotic:        master.FlagNarcotic,
		FlagPsychotropic:    master.FlagPsychotropic,
		FlagStimulant:       master.FlagStimulant,
		FlagStimulantRaw:    master.FlagStimulantRaw,
		ProcessFlagMA:       processFlagMA, // ★ MAフラグを設定
	}
}

func OpenFileHeader(fh *multipart.FileHeader) (multipart.File, error) {
	return fh.Open()
}

// ▼▼▼【修正】renderTransactionTableHTML (変更なし) ▼▼▼
func renderTransactionTableHTML(transactions []model.TransactionRecord, wholesalerMap map[string]string) string {
	var sb strings.Builder

	sb.WriteString(`
    <thead>
        <tr>
            <th rowspan="2" class="col-action">－</th>
            <th class="col-date">日付</th>
            <th class="col-yj">YJ</th>
            <th colspan="2" class="col-product">製品名</th>
            <th class="col-count">個数</th>
            <th class="col-yjqty">YJ数量</th>
            <th class="col-yjpackqty">YJ包装数</th>
            <th class="col-yjunit">YJ単位</th>
            <th class="col-unitprice">単価</th>
            <th class="col-expiry">期限</th>
            <th class="col-wholesaler">卸</th>
            <th class="col-line">行</th>
        </tr>
        <tr>
            <th class="col-flag">種別</th>
            <th class="col-jan">JAN</th>
            <th class="col-package">包装</th>
            <th class="col-maker">メーカー</th>
            <th class="col-form">剤型</th>
            <th class="col-janqty">JAN数量</th>
            <th class="col-janpackqty">JAN包装数</th>
            <th class="col-janunit">JAN単位</th>
            <th class="col-amount">金額</th>
            <th class="col-lot">ロット</th>
            <th class="col-receipt">伝票番号</th>
            <th class="col-ma">MA</th>
        </tr>
    </thead>`)

	sb.WriteString(`<tbody>`)
	if len(transactions) == 0 {
		sb.WriteString(`<tr><td colspan="13">登録されたデータはありません。</td></tr>`)
	} else {
		for _, tx := range transactions {
			formattedDate := tx.TransactionDate
			formattedExpiry := tx.ExpiryDate
			formattedYjQty := strconv.FormatFloat(tx.YjQuantity, 'f', 2, 64)
			formattedUnitPrice := strconv.FormatFloat(tx.UnitPrice, 'f', 2, 64)
			formattedSubtotal := strconv.FormatFloat(tx.Subtotal, 'f', 2, 64)

			var flagText string
			switch tx.Flag {
			case 1:
				flagText = "納品"
			case 2:
				flagText = "返品"
			default:
				flagText = strconv.Itoa(tx.Flag)
			}

			// ★ 卸名をマップから取得
			wholesalerName := tx.ClientCode
			if wholesalerMap != nil {
				if name, ok := wholesalerMap[tx.ClientCode]; ok {
					wholesalerName = name
				}
			}

			sb.WriteString(`<tr>`)
			sb.WriteString(`<td class="center col-action">－</td>`)
			sb.WriteString(fmt.Sprintf(`<td class="center col-date">%s</td>`, formattedDate))
			sb.WriteString(fmt.Sprintf(`<td class="col-yj">%s</td>`, tx.YjCode))
			sb.WriteString(fmt.Sprintf(`<td colspan="2" class="col-product">%s</td>`, tx.ProductName))
			sb.WriteString(fmt.Sprintf(`<td class="right col-count">%.0f</td>`, tx.DatQuantity))
			sb.WriteString(fmt.Sprintf(`<td class="right col-yjqty">%s</td>`, formattedYjQty))
			sb.WriteString(fmt.Sprintf(`<td class="right col-yjpackqty">%.0f</td>`, tx.YjPackUnitQty))
			sb.WriteString(fmt.Sprintf(`<td classC="center col-yjunit">%s</td>`, tx.YjUnitName))
			sb.WriteString(fmt.Sprintf(`<td class="right col-unitprice">%s</td>`, formattedUnitPrice))
			sb.WriteString(fmt.Sprintf(`<td class="center col-expiry">%s</td>`, formattedExpiry))
			// ★ 卸コードではなく卸名を表示
			sb.WriteString(fmt.Sprintf(`<td class="center col-wholesaler">%s</td>`, wholesalerName))
			sb.WriteString(fmt.Sprintf(`<td class="center col-line">%s</td>`, tx.LineNumber))
			sb.WriteString(`</tr>`)

			sb.WriteString(`<tr>`)
			sb.WriteString(`<td></td>`)
			sb.WriteString(fmt.Sprintf(`<td class="center col-flag">%s</td>`, flagText))
			sb.WriteString(fmt.Sprintf(`<td class="col-jan">%s</td>`, tx.JanCode))
			// ▼▼▼【ここが修正箇所です】▼▼▼
			sb.WriteString(fmt.Sprintf(`<td class="col-package">%s</td>`, tx.PackageSpec))
			// ▲▲▲【修正ここまで】▲▲▲
			sb.WriteString(fmt.Sprintf(`<td class="col-maker">%s</td>`, tx.MakerName))
			sb.WriteString(`<td class="col-form"></td>`)
			sb.WriteString(fmt.Sprintf(`<td class="right col-janqty">%.2f</td>`, tx.JanQuantity)) // ★ JAN数量
			sb.WriteString(fmt.Sprintf(`<td class="right col-janpackqty">%.0f</td>`, tx.JanPackUnitQty))
			sb.WriteString(fmt.Sprintf(`<td class="center col-janunit">%s</td>`, tx.JanUnitName))
			sb.WriteString(fmt.Sprintf(`<td class="right col-amount">%s</td>`, formattedSubtotal))
			sb.WriteString(fmt.Sprintf(`<td class="col-lot">%s</td>`, tx.LotNumber))
			sb.WriteString(fmt.Sprintf(`<td class="col-receipt">%s</td>`, tx.ReceiptNumber))
			sb.WriteString(fmt.Sprintf(`<td class="center col-ma">%s</td>`, tx.ProcessFlagMA))
			sb.WriteString(`</tr>`)
		}
	}
	sb.WriteString(`</tbody>`)

	return sb.String()
}
