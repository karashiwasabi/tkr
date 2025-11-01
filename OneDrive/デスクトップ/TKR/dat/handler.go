// C:\Users\wasab\OneDrive\デスクトップ\TKR\dat\handler.go
package dat

import (
	"database/sql" // ★ sql.ErrNoRows のために追加
	"encoding/json"
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

	"github.com/jmoiron/sqlx"
)

func respondJSONError(w http.ResponseWriter, message string, statusCode int) {
	log.Println("Error response:", message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   message,
		"results":   []interface{}{},
		"tableHTML": renderTransactionTableHTML(nil),
	})
}

func UploadDatHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received DAT upload request...")

		err := r.ParseMultipartForm(32 << 20)
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
		htmlString := renderTransactionTableHTML(successfullyInsertedTransactions)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   fmt.Sprintf("Processed %d DAT file(s). See results for details.", len(processedFiles)),
			"results":   allResults,
			"tableHTML": htmlString,
		})
		log.Println("Finished DAT upload request.")
	}
}

// ▼▼▼【ここから追加】GS1-128検索ハンドラ (2段階検索ロジック) ▼▼▼
func SearchDatHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondJSONError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
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

		// 1. GS1コード(14桁)で product_master を検索
		master, err := database.GetProductMasterByGs1Code(db, gtin14)
		if err != nil {
			if err == sql.ErrNoRows {
				log.Printf("Product master not found for GS1_CODE: %s", gtin14)
				respondJSONError(w, "GS1コードに対応するマスターが見つかりません。", http.StatusNotFound)
			} else {
				log.Printf("Error searching product master by GS1_CODE %s: %v", gtin14, err)
				respondJSONError(w, "マスター検索中にエラーが発生しました。", http.StatusInternalServerError)
			}
			return
		}

		productCode := master.ProductCode // product_master.product_code (JAN)

		expiryYYMMDD := gs1Result.ExpiryDate
		expiryYYMM := ""
		if len(expiryYYMMDD) == 6 {
			expiryYYMM = expiryYYMMDD[:4]
		}

		log.Printf("Parsed GS1: GTIN(14)='%s' -> ProductCode(JAN)='%s', Expiry(6)='%s', Expiry(4)='%s', Lot='%s'",
			gtin14, productCode, expiryYYMMDD, expiryYYMM, gs1Result.LotNumber)

		// 2. ProductCode(JAN), 期限(4/6), ロットで transaction_records を検索
		transactions, err := database.SearchTransactions(db, productCode, expiryYYMMDD, expiryYYMM, gs1Result.LotNumber)

		if err != nil {
			log.Printf("Error searching transactions: %v", err)
			respondJSONError(w, "トランザクション検索中にエラーが発生しました。", http.StatusInternalServerError)
			return
		}

		log.Printf("Found %d transactions matching criteria.", len(transactions))

		htmlString := renderTransactionTableHTML(transactions)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   fmt.Sprintf("%d 件のデータが見つかりました。", len(transactions)),
			"tableHTML": htmlString,
		})
	}
}

// ▲▲▲【追加ここまで】▲▲▲

func MapDatToTransaction(dat model.DatRecord, master *model.ProductMaster) model.TransactionRecord {
	packageSpec := fmt.Sprintf("%s %v%s", master.PackageForm, master.YjPackUnitQty, master.YjUnitName)

	return model.TransactionRecord{
		TransactionDate:     dat.Date,
		ClientCode:          dat.ClientCode,
		ReceiptNumber:       dat.ReceiptNumber,
		LineNumber:          dat.LineNumber,
		Flag:                dat.Flag,
		JanCode:             master.ProductCode,
		YjCode:              master.YjCode,
		ProductName:         master.ProductName,
		KanaName:            master.KanaName,
		UsageClassification: master.UsageClassification,
		PackageForm:         master.PackageForm,
		PackageSpec:         packageSpec,
		MakerName:           master.MakerName,
		DatQuantity:         dat.DatQuantity,
		JanPackInnerQty:     master.JanPackInnerQty,
		JanQuantity:         0,
		JanPackUnitQty:      master.JanPackUnitQty,
		JanUnitName:         "",
		JanUnitCode:         fmt.Sprintf("%d", master.JanUnitCode),
		YjQuantity:          0,
		YjPackUnitQty:       master.YjPackUnitQty,
		YjUnitName:          master.YjUnitName,
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
		ProcessFlagMA:       "",
	}
}

func OpenFileHeader(fh *multipart.FileHeader) (multipart.File, error) {
	return fh.Open()
}

func renderTransactionTableHTML(transactions []model.TransactionRecord) string {
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
            <th classs="col-janunit">JAN単位</th>
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

			sb.WriteString(`<tr>`)
			sb.WriteString(`<td class="center col-action">－</td>`)
			sb.WriteString(fmt.Sprintf(`<td class="center col-date">%s</td>`, formattedDate))
			sb.WriteString(fmt.Sprintf(`<td class="col-yj">%s</td>`, tx.YjCode))
			sb.WriteString(fmt.Sprintf(`<td colspan="2" class="col-product">%s</td>`, tx.ProductName))
			sb.WriteString(fmt.Sprintf(`<td class="right col-count">%.0f</td>`, tx.DatQuantity))
			sb.WriteString(fmt.Sprintf(`<td class="right col-yjqty">%s</td>`, formattedYjQty))
			sb.WriteString(fmt.Sprintf(`<td class="right col-yjpackqty">%.0f</td>`, tx.YjPackUnitQty))
			sb.WriteString(fmt.Sprintf(`<td class="center col-yjunit">%s</td>`, tx.YjUnitName))
			sb.WriteString(fmt.Sprintf(`<td class="right col-unitprice">%s</td>`, formattedUnitPrice))
			sb.WriteString(fmt.Sprintf(`<td class="center col-expiry">%s</td>`, formattedExpiry))
			sb.WriteString(fmt.Sprintf(`<td class="center col-wholesaler">%s</td>`, tx.ClientCode))
			sb.WriteString(fmt.Sprintf(`<td class="center col-line">%s</td>`, tx.LineNumber))
			sb.WriteString(`</tr>`)

			sb.WriteString(`<tr>`)
			sb.WriteString(`<td></td>`)
			sb.WriteString(fmt.Sprintf(`<td class="center col-flag">%s</td>`, flagText))
			sb.WriteString(fmt.Sprintf(`<td class="col-jan">%s</td>`, tx.JanCode))
			sb.WriteString(fmt.Sprintf(`<td class="col-package">%s</td>`, tx.PackageForm))
			sb.WriteString(fmt.Sprintf(`<td class="col-maker">%s</td>`, tx.MakerName))
			sb.WriteString(`<td class="col-form"></td>`)
			sb.WriteString(`<td class="right col-janqty"></td>`)
			sb.WriteString(fmt.Sprintf(`<td class="right col-janpackqty">%.0f</td>`, tx.JanPackUnitQty))
			sb.WriteString(fmt.Sprintf(`<td class="center col-janunit">%s</td>`, tx.JanUnitName))
			sb.WriteString(fmt.Sprintf(`<td class="right col-amount">%s</td>`, formattedSubtotal))
			// ▼▼▼【修正】class 属性のタイプミスを修正 (classcol-lot -> class="col-lot") ▼▼▼
			sb.WriteString(fmt.Sprintf(`<td class="col-lot">%s</td>`, tx.LotNumber))
			// ▲▲▲【修正ここまで】▲▲▲
			sb.WriteString(fmt.Sprintf(`<td class="col-receipt">%s</td>`, tx.ReceiptNumber))
			sb.WriteString(fmt.Sprintf(`<td class="center col-ma">%s</td>`, tx.ProcessFlagMA))
			sb.WriteString(`</tr>`)
		}
	}
	sb.WriteString(`</tbody>`)

	return sb.String()
}
