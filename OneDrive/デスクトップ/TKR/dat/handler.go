package dat

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"tkr/barcode"
	"tkr/database"
	"tkr/mappers"
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
		"message": message,
		"results": []interface{}{},
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
			file, openErr := fileHeader.Open()
			if openErr != nil {
				log.Printf("Failed to open uploaded file %s: %v", fileHeader.Filename, openErr)
				fileResult["error"] = fmt.Sprintf("Failed to open file: %v", openErr)
				allResults =
					append(allResults,
						fileResult)
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

			insertedTransactions, processErr := ProcessDatRecords(tx, parsedRecords)
			if processErr != nil {
				log.Printf("Failed to process DAT records for %s: %v", fileHeader.Filename, processErr)
				fileResult["error"] = fmt.Sprintf("Failed to process records: %v", processErr)
				allResults = append(allResults, fileResult)
				tx.Rollback()
				continue
			}

			if commitErr := tx.Commit(); commitErr != nil {
				log.Printf("Failed to commit transaction for %s: %v", fileHeader.Filename, commitErr)
				fileResult["error"] = fmt.Sprintf("Failed to commit transaction: %v", commitErr)
				allResults = append(allResults, fileResult)
				continue
			}

			successfullyInsertedTransactions = append(successfullyInsertedTransactions, insertedTransactions...)
			log.Printf("Successfully inserted %d records from %s", len(insertedTransactions), fileHeader.Filename)
			fileResult["success"] = true
			fileResult["records_inserted"] = len(insertedTransactions)
			allResults = append(allResults, fileResult)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": fmt.Sprintf("Processed %d DAT file(s). See results for details.", len(processedFiles)),
			"results": allResults,
			"records": successfullyInsertedTransactions,
		})
		log.Println("Finished DAT upload request.")
	}
}

// ▼▼▼【ここを修正】キーから `LineNumber` を削除 ▼▼▼
func removeDatDuplicates(records []model.DatRecord) []model.DatRecord {
	seen := make(map[string]struct{})
	var result []model.DatRecord
	for _, r := range records {
		// キーを「日付・卸コード・伝票番号・行番号」から
		// 「日付・卸コード・伝票番号・JANコード・行番号」に変更 (より厳密な重複排除)
		key := fmt.Sprintf("%s|%s|%s|%s|%s", r.Date, r.ClientCode, r.ReceiptNumber, r.JanCode, r.LineNumber)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, r)
	}
	return result
}

// ▲▲▲【修正ここまで】▲▲▲

func ProcessDatRecords(tx *sqlx.Tx, parsedRecords []model.DatRecord) ([]model.TransactionRecord, error) {
	var insertedTransactions []model.TransactionRecord
	var insertedCount int = 0

	recordsToProcess := removeDatDuplicates(parsedRecords)

	// ▼▼▼【ここから追加】伝票単位での削除ロジック ▼▼▼
	if len(recordsToProcess) == 0 {
		return insertedTransactions, nil
	}

	// 処理対象の「伝票キー」を収集 (日付 + 卸コード + 伝票番号)
	receiptKeysToDelete := make(map[string]struct {
		Date       string
		ClientCode string
	})
	for _, rec := range recordsToProcess {
		// 納品(1)と返品(2)のみを洗い替え対象とする
		if rec.Flag == 1 || rec.Flag == 2 {
			receiptKeysToDelete[rec.ReceiptNumber] = struct {
				Date       string
				ClientCode string
			}{
				Date:       rec.Date,
				ClientCode: rec.ClientCode,
			}
		}
	}

	// 収集したキーで既存データを削除
	for receiptNumber, keyInfo := range receiptKeysToDelete {
		log.Printf("Deleting existing DAT records for Receipt: %s, Date: %s, Client: %s", receiptNumber, keyInfo.Date, keyInfo.ClientCode)
		// 伝票番号、日付、卸コード、フラグ(1,2)が一致するものを削除
		const q = `DELETE FROM transaction_records WHERE receipt_number = ? AND transaction_date = ? AND client_code = ? AND flag IN (1, 2)`
		_, err := tx.Exec(q, receiptNumber, keyInfo.Date, keyInfo.ClientCode)
		if err != nil {
			return nil, fmt.Errorf("failed to delete existing DAT records for receipt %s: %w", receiptNumber, err)
		}
	}
	// ▲▲▲【追加ここまで】▲▲▲

	for _, rec := range recordsToProcess {
		key := rec.JanCode

		master, err := mastermanager.FindOrCreateMaster(tx, key, rec.ProductName)
		if err != nil {
			return nil, fmt.Errorf("master creation failed for key %s: %w", key, err)
		}

		if master.Origin != "JCSHMS" && master.KanaNameShort != rec.ProductName {
			log.Printf("Updating non-JCSHMS master (ProductCode: %s) KanaNameShort from '%s' to '%s'",
				master.ProductCode, master.KanaNameShort, rec.ProductName)

			input := mastermanager.MasterToInput(master)
			input.KanaNameShort = rec.ProductName // DATの製品名で上書き

			updatedMaster, upsertErr := mastermanager.UpsertProductMasterSqlx(tx, input)
			if upsertErr != nil {
				return nil, fmt.Errorf("failed to update kana_name_short for existing master %s: %w", master.ProductCode, upsertErr)
			}
			*master = *updatedMaster // ポインタが指す中身を更新
		}

		transaction := model.TransactionRecord{
			TransactionDate: rec.Date,
			ClientCode:      rec.ClientCode,
			ReceiptNumber:   rec.ReceiptNumber,
			LineNumber:      rec.LineNumber,
			Flag:            rec.Flag,
			DatQuantity:     rec.DatQuantity,
			UnitPrice:       rec.UnitPrice,
			Subtotal:        rec.Subtotal,
			ExpiryDate:      rec.ExpiryDate,
			LotNumber:       rec.LotNumber,
		}

		transaction.JanQuantity = rec.DatQuantity
		transaction.YjQuantity = rec.DatQuantity * master.JanPackInnerQty

		if transaction.UnitPrice == 0 {
			transaction.UnitPrice = master.NhiPrice
			transaction.Subtotal = transaction.YjQuantity * transaction.UnitPrice
		}

		mappers.MapMasterToTransaction(&transaction,
			master)

		if err := database.InsertTransactionRecord(tx, transaction); err != nil {
			return nil, fmt.Errorf("transaction insert failed for key %s: %w", key, err)
		}
		insertedTransactions = append(insertedTransactions, transaction)
		insertedCount++
	}
	return insertedTransactions, nil
}

func SearchDatHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondJSONError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		barcodeStr := r.URL.Query().Get("barcode")
		log.Printf("Received DAT search request... Barcode: [%s]", barcodeStr)

		if barcodeStr == "" {
			respondJSONError(w, "バーコードを入力してください。", http.StatusBadRequest)
			return
		}

		master, err := database.GetProductMasterByBarcode(db, barcodeStr)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Printf("Product master not found for Barcode: %s", barcodeStr)
				respondJSONError(w, "バーコードに対応するマスターが見つかりません。", http.StatusNotFound)
			} else {
				log.Printf("Error searching product master by Barcode %s: %v", barcodeStr, err)
				respondJSONError(w, fmt.Sprintf("マスター検索エラー: %v", err), http.StatusInternalServerError)
			}
			return
		}

		productCode := master.ProductCode
		var expiryYYMMDD, expiryYYMM, lotNumber string
		if len(barcodeStr) > 14 {
			gs1Result,
				parseErr := barcode.Parse(barcodeStr)
			if parseErr == nil && gs1Result != nil {
				expiryYYMMDD = gs1Result.ExpiryDate
				if len(expiryYYMMDD) == 6 {
					expiryYYMM = expiryYYMMDD[:4]
				}
				lotNumber = gs1Result.LotNumber
			}
		}

		log.Printf("Search criteria: ProductCode(JAN)='%s', Expiry(6)='%s', Expiry(4)='%s', Lot='%s'",
			productCode, expiryYYMMDD, expiryYYMM, lotNumber)

		transactions, err := database.SearchTransactions(db, productCode, expiryYYMMDD, expiryYYMM, lotNumber)

		if err != nil {
			log.Printf("Error searching transactions: %v", err)
			respondJSONError(w, "トランザクション検索中にエラーが発生しました。", http.StatusInternalServerError)
			return
		}

		log.Printf("Found %d transactions matching criteria.", len(transactions))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":      fmt.Sprintf("%d 件のデータが見つかりました。", len(transactions)),
			"transactions": transactions,
		})
	}
}

func OpenFileHeader(fh *multipart.FileHeader) (multipart.File, error) {
	return fh.Open()
}
