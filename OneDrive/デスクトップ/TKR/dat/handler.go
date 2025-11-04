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
			// ▼▼▼【削除】使用しない変数を削除 ▼▼▼
			// var currentFileTransactions []model.TransactionRecord //
			// ▲▲▲【削除ここまで】▲▲▲
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

func ProcessDatRecords(tx *sqlx.Tx, parsedRecords []model.DatRecord) ([]model.TransactionRecord, error) {
	var insertedTransactions []model.TransactionRecord
	var insertedCount int = 0

	for _, rec := range parsedRecords {
		key := rec.JanCode
		if key == "" || key == "0000000000000" {
			key = fmt.Sprintf("9999999999999%s", rec.ProductName)
		}

		master, err := mastermanager.FindOrCreateMaster(tx, key, rec.ProductName)
		if err != nil {
			return nil, fmt.Errorf("master creation failed for key %s: %w", key, err)
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

		if master.YjPackUnitQty > 0 {
			transaction.YjQuantity = rec.DatQuantity * master.YjPackUnitQty
		}
		if master.JanPackUnitQty > 0 {
			transaction.JanQuantity = rec.DatQuantity * master.JanPackUnitQty
		}

		if transaction.UnitPrice == 0 {
			transaction.UnitPrice = master.NhiPrice
			transaction.Subtotal = transaction.YjQuantity * transaction.UnitPrice
		}

		mappers.MapMasterToTransaction(&transaction, master)

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
			gs1Result, parseErr := barcode.Parse(barcodeStr)
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
