// C:\Users\wasab\OneDrive\デスクトップ\TKR\dat\dat_upload_handler.go
package dat

import (
	"encoding/json"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"tkr/database"
	"tkr/mappers"
	"tkr/mastermanager"
	"tkr/model"
	"tkr/parsers"

	"github.com/jmoiron/sqlx"
)

// ▼▼▼【ここから削除】dat_utils.go に移管するため ▼▼▼
/*
func respondJSONError(w http.ResponseWriter, message string, statusCode int) {
	log.Println("Error response:", message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": message,
		"results": []interface{}{},
	})
}
*/
// ▲▲▲【削除ここまで】▲▲▲

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
			parsedRecords,
				parseErr := parsers.ParseDat(file)
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

			if len(insertedTransactions) > 0 {
				var deliveredItems []model.Backorder
				for _, rec := range insertedTransactions {
					if rec.Flag ==
						1 {
						deliveredItems = append(deliveredItems, model.Backorder{
							YjCode:          rec.YjCode,
							PackageForm:     rec.PackageForm,
							JanPackInnerQty: rec.JanPackInnerQty,
							YjUnitName:      rec.YjUnitName,
							YjQuantity:      rec.YjQuantity,
						})
					}
				}

				if len(deliveredItems) > 0 {
					if err := database.ReconcileBackorders(tx, deliveredItems); err != nil {
						log.Printf("WARN: Failed to reconcile backorders after DAT import: %v", err)
						fileResult["error"] = fmt.Sprintf("納品登録成功、発注残消込失敗: %v", err)
					} else {
						log.Printf("Successfully reconciled %d backorder items for %s.", len(deliveredItems), fileHeader.Filename)
					}
				}
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

func removeDatDuplicates(records []model.DatRecord) []model.DatRecord {
	seen := make(map[string]struct{})
	var result []model.DatRecord
	for _, r := range records {
		key := fmt.Sprintf("%s|%s|%s|%s|%s", r.Date, r.ClientCode, r.ReceiptNumber, r.JanCode, r.LineNumber)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, r)
	}
	return result
}

func ProcessDatRecords(tx *sqlx.Tx, parsedRecords []model.DatRecord) ([]model.TransactionRecord, error) {
	var insertedTransactions []model.TransactionRecord
	var insertedCount int = 0

	recordsToProcess := removeDatDuplicates(parsedRecords)

	if len(recordsToProcess) == 0 {
		return insertedTransactions, nil
	}

	receiptKeysToDelete := make(map[string]struct {
		Date       string
		ClientCode string
	})
	for _, rec := range recordsToProcess {
		if rec.Flag == 1 ||
			rec.Flag == 2 {
			receiptKeysToDelete[rec.ReceiptNumber] = struct {
				Date       string
				ClientCode string
			}{
				Date:       rec.Date,
				ClientCode: rec.ClientCode,
			}
		}
	}

	for receiptNumber, keyInfo := range receiptKeysToDelete {
		log.Printf("Deleting existing DAT records for Receipt: %s, Date: %s, Client: %s", receiptNumber, keyInfo.Date, keyInfo.ClientCode)
		const q = `DELETE FROM transaction_records WHERE receipt_number = ?
AND transaction_date = ? AND client_code = ? AND flag IN (1, 2)`
		_, err := tx.Exec(q, receiptNumber, keyInfo.Date, keyInfo.ClientCode)
		if err != nil {
			return nil, fmt.Errorf("failed to delete existing DAT records for receipt %s: %w", receiptNumber, err)
		}
	}

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
			input.KanaNameShort = rec.ProductName

			updatedMaster, upsertErr := mastermanager.UpsertProductMasterSqlx(tx, input)
			if upsertErr != nil {
				return nil, fmt.Errorf("failed to update kana_name_short for existing master %s: %w", master.ProductCode, upsertErr)
			}
			*master = *updatedMaster
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

		transaction.YjQuantity = rec.DatQuantity * master.YjPackUnitQty

		if master.JanPackInnerQty > 0 {
			transaction.JanQuantity = transaction.YjQuantity /
				master.JanPackInnerQty
		} else {
			transaction.JanQuantity = 0
		}

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

func OpenFileHeader(fh *multipart.FileHeader) (multipart.File, error) {
	return fh.Open()
}
