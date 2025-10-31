// C:\Users\wasab\OneDrive\デスクトップ\TKR\dat\handler.go
package dat

import (
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

// ▼▼▼ エラーレスポンスをJSONで返すヘルパー ▼▼▼
func respondJSONError(w http.ResponseWriter, message string, statusCode int) {
	log.Println("Error response:", message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   message,
		"results":   []interface{}{},
		"tableHTML": renderTransactionTableHTML(nil), // 空のテーブルを返す
	})
}

// ▲▲▲ 追加 ▲▲▲

// UploadDatHandler handles DAT file uploads, checks/creates masters, and inserts transaction records.
func UploadDatHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received DAT upload request...")

		err := r.ParseMultipartForm(32 << 20) // 32MB limit
		if err != nil {
			// ▼▼▼ JSONエラーレスポンスに変更 ▼▼▼
			respondJSONError(w, "File upload error: "+err.Error(), http.StatusBadRequest)
			// ▲▲▲ 変更 ▲▲▲
			return
		}
		defer r.MultipartForm.RemoveAll()

		var processedFiles []string
		// ▼▼▼ 全ファイルを通して成功した TransactionRecord を格納するスライス ▼▼▼
		var successfullyInsertedTransactions []model.TransactionRecord
		// ▲▲▲ 追加 ▲▲▲
		var allResults []map[string]interface{} // Results for each processed file

		for _, fileHeader := range r.MultipartForm.File["file"] {
			log.Printf("Processing file: %s", fileHeader.Filename)
			processedFiles = append(processedFiles, fileHeader.Filename)
			fileResult := map[string]interface{}{"filename": fileHeader.Filename}
			// ▼▼▼ このファイルで成功した TransactionRecord を一時的に格納 ▼▼▼
			var currentFileTransactions []model.TransactionRecord
			// ▲▲▲ 追加 ▲▲▲

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
			// ... (中略: for _, rec := range parsedRecords ループ ... 変更なし) ...
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
					tx.Rollback() // Rollback transaction for this file
					goto nextFileLoop
				}

				transaction := MapDatToTransaction(rec, master)
				if err := database.InsertTransactionRecord(tx, transaction); err != nil {
					log.Printf("Failed to insert transaction record for key %s (Product: %s) in file %s: %v", key, rec.ProductName, fileHeader.Filename, err)
					fileResult["error"] = fmt.Sprintf("Transaction insert failed for key %s: %v", key, err)
					allResults = append(allResults, fileResult)
					tx.Rollback() // Rollback transaction for this file
					goto nextFileLoop
				}

				// ... (中略: 成功した TransactionRecord を currentFileTransactions に追加する部分 ... 変更なし) ...
				currentFileTransactions = append(currentFileTransactions, transaction)
				// ▲▲▲ 追加 ▲▲▲
				insertedCount++
			} // End record loop

			if commitErr := tx.Commit(); commitErr != nil {
				log.Printf("Failed to commit transaction for %s: %v", fileHeader.Filename, commitErr)
				fileResult["error"] = fmt.Sprintf("Failed to commit transaction: %v", commitErr)
				allResults = append(allResults, fileResult)
				goto nextFileLoop // Rollback is handled by defer
			}

			// ▼▼▼【ここから追加】コミット成功時、このファイルのトランザクションを全体リストに追加 ▼▼▼
			successfullyInsertedTransactions = append(successfullyInsertedTransactions, currentFileTransactions...)
			// ▲▲▲【追加ここまで】▲▲▲

			log.Printf("Successfully inserted %d records from %s", insertedCount, fileHeader.Filename)
			fileResult["success"] = true
			fileResult["records_inserted"] = insertedCount
			allResults = append(allResults, fileResult)

		nextFileLoop:
			continue
		} // End file loop

		// --- Final response ---
		w.Header().Set("Content-Type", "application/json")
		// ▼▼▼ レスポンスにサマリーと「HTML文字列」を含める ▼▼▼
		htmlString := renderTransactionTableHTML(successfullyInsertedTransactions)

		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   fmt.Sprintf("Processed %d DAT file(s). See results for details.", len(processedFiles)),
			"results":   allResults, // 各ファイルごとの処理結果サマリー
			"tableHTML": htmlString, // ★ サーバーで生成したHTML文字列
		})
		// ▲▲▲ 変更 ▲▲▲
		log.Println("Finished DAT upload request.")
	}
}

// MapDatToTransaction (変更なし)
func MapDatToTransaction(dat model.DatRecord, master *model.ProductMaster) model.TransactionRecord {
	// ... (中略: 関数全体 ... 変更なし) ...
	// PackageSpec の計算 (例)
	packageSpec := fmt.Sprintf("%s %v%s", master.PackageForm, master.YjPackUnitQty, master.YjUnitName)

	return model.TransactionRecord{
		TransactionDate:     dat.Date,
		ClientCode:          dat.ClientCode,
		ReceiptNumber:       dat.ReceiptNumber,
		LineNumber:          dat.LineNumber,
		Flag:                dat.Flag,
		JanCode:             master.ProductCode, // Use code from master
		YjCode:              master.YjCode,
		ProductName:         master.ProductName,
		KanaName:            master.KanaName,
		UsageClassification: master.UsageClassification,
		PackageForm:         master.PackageForm,
		PackageSpec:         packageSpec, // Calculated spec
		MakerName:           master.MakerName,
		DatQuantity:         dat.DatQuantity,
		JanPackInnerQty:     master.JanPackInnerQty,
		JanQuantity:         0, // Requires calculation
		JanPackUnitQty:      master.JanPackUnitQty,
		JanUnitName:         "", // Requires mapping from JanUnitCode if needed
		JanUnitCode:         fmt.Sprintf("%d", master.JanUnitCode),
		YjQuantity:          0, // Requires calculation
		YjPackUnitQty:       master.YjPackUnitQty,
		YjUnitName:          master.YjUnitName,
		UnitPrice:           dat.UnitPrice,
		PurchasePrice:       master.PurchasePrice, // From master
		SupplierWholesale:   master.SupplierWholesale,
		Subtotal:            dat.Subtotal,
		TaxAmount:           0, // DAT file doesn't seem to have tax
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

// OpenFileHeader (変更なし)
func OpenFileHeader(fh *multipart.FileHeader) (multipart.File, error) {
	// ... (以下略) ...
	return fh.Open()
}

// ▼▼▼【ここから追加】トランザクションリストからHTMLテーブル文字列を生成する関数 ▼▼▼

func renderTransactionTableHTML(transactions []model.TransactionRecord) string {
	var sb strings.Builder

	// 1. テーブルヘッダー (thead)
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

	// 2. テーブルボディ (tbody)
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

			// 1行目
			sb.WriteString(`<tr>`)
			sb.WriteString(`<td class="center col-action">－</td>`)
			sb.WriteString(fmt.Sprintf(`<td class="center col-date">%s</td>`, formattedDate))
			sb.WriteString(fmt.Sprintf(`<td class="col-yj">%s</td>`, tx.YjCode))
			sb.WriteString(fmt.Sprintf(`<td colspan="2" class="col-product">%s</td>`, tx.ProductName))
			sb.WriteString(fmt.Sprintf(`<td class="right col-count">%.0f</td>`, tx.DatQuantity)) // DatQuantityは整数で表示
			sb.WriteString(fmt.Sprintf(`<td class="right col-yjqty">%s</td>`, formattedYjQty))
			sb.WriteString(fmt.Sprintf(`<td class="right col-yjpackqty">%.0f</td>`, tx.YjPackUnitQty)) // YjPackUnitQtyは整数で表示
			sb.WriteString(fmt.Sprintf(`<td class="center col-yjunit">%s</td>`, tx.YjUnitName))
			sb.WriteString(fmt.Sprintf(`<td class="right col-unitprice">%s</td>`, formattedUnitPrice))
			sb.WriteString(fmt.Sprintf(`<td class="center col-expiry">%s</td>`, formattedExpiry))
			sb.WriteString(fmt.Sprintf(`<td class="center col-wholesaler">%s</td>`, tx.ClientCode))
			sb.WriteString(fmt.Sprintf(`<td class="center col-line">%s</td>`, tx.LineNumber))
			sb.WriteString(`</tr>`)

			// 2行目
			sb.WriteString(`<tr>`)
			sb.WriteString(`<td></td>`)
			sb.WriteString(fmt.Sprintf(`<td class="center col-flag">%s</td>`, flagText))
			sb.WriteString(fmt.Sprintf(`<td class="col-jan">%s</td>`, tx.JanCode))
			sb.WriteString(fmt.Sprintf(`<td class="col-package">%s</td>`, tx.PackageForm))
			sb.WriteString(fmt.Sprintf(`<td class="col-maker">%s</td>`, tx.MakerName))
			sb.WriteString(`<td class="col-form"></td>`)                                                 // 剤型 (空)
			sb.WriteString(`<td class="right col-janqty"></td>`)                                         // JAN数量 (空)
			sb.WriteString(fmt.Sprintf(`<td class="right col-janpackqty">%.0f</td>`, tx.JanPackUnitQty)) // JanPackUnitQtyは整数で表示
			sb.WriteString(fmt.Sprintf(`<td class="center col-janunit">%s</td>`, tx.JanUnitName))
			sb.WriteString(fmt.Sprintf(`<td class="right col-amount">%s</td>`, formattedSubtotal))
			sb.WriteString(fmt.Sprintf(`<td classcol-lot">%s</td>`, tx.LotNumber))
			sb.WriteString(fmt.Sprintf(`<td class="col-receipt">%s</td>`, tx.ReceiptNumber))
			sb.WriteString(fmt.Sprintf(`<td class="center col-ma">%s</td>`, tx.ProcessFlagMA))
			sb.WriteString(`</tr>`)
		}
	}
	sb.WriteString(`</tbody>`)

	return sb.String()
}

// ▲▲▲【追加ここまで】▲▲▲
