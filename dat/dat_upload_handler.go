// C:\Users\wasab\OneDrive\デスクトップ\TKR\dat\dat_upload_handler.go
package dat

import (
	"encoding/json"
	"fmt"
	"io"
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

// UploadDatHandler はブラウザからのアップロードを受け付けます
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
				allResults = append(allResults, fileResult)
				continue
			}

			// ▼▼▼ 共通関数を呼び出す ▼▼▼
			inserted, err := ImportDatStream(db, file, fileHeader.Filename)
			file.Close()
			// ▲▲▲ 共通関数呼び出し終了 ▲▲▲

			if err != nil {
				log.Printf("Failed to process DAT file %s: %v", fileHeader.Filename, err)
				fileResult["error"] = fmt.Sprintf("Failed to process: %v", err)
				allResults = append(allResults, fileResult)
				continue
			}

			successfullyInsertedTransactions = append(successfullyInsertedTransactions, inserted...)
			log.Printf("Successfully inserted %d records from %s", len(inserted), fileHeader.Filename)
			fileResult["success"] = true
			fileResult["records_inserted"] = len(inserted)
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

// ▼▼▼ 追加: 外部から呼び出せる共通関数 ▼▼▼
func ImportDatStream(db *sqlx.DB, r io.Reader, filename string) ([]model.TransactionRecord, error) {
	parsedRecords, parseErr := parsers.ParseDat(r)
	if parseErr != nil {
		return nil, fmt.Errorf("DAT parse error: %w", parseErr)
	}

	tx, txErr := db.Beginx()
	if txErr != nil {
		return nil, fmt.Errorf("failed to start transaction: %w", txErr)
	}
	defer tx.Rollback()

	insertedTransactions, processErr := ProcessDatRecords(tx, parsedRecords)
	if processErr != nil {
		return nil, fmt.Errorf("process records error: %w", processErr)
	}

	// 発注残消込処理
	if len(insertedTransactions) > 0 {
		var deliveredItems []model.Backorder
		for _, rec := range insertedTransactions {
			if rec.Flag == 1 { // 納品
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
				log.Printf("WARN: Failed to reconcile backorders after DAT import (%s): %v", filename, err)
			}
		}
	}

	if commitErr := tx.Commit(); commitErr != nil {
		return nil, fmt.Errorf("commit error: %w", commitErr)
	}

	return insertedTransactions, nil
}

// ▲▲▲ 追加ここまで ▲▲▲

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

	recordsToProcess := removeDatDuplicates(parsedRecords)

	if len(recordsToProcess) == 0 {
		return insertedTransactions, nil
	}

	receiptKeysToDelete := make(map[string]struct {
		Date       string
		ClientCode string
	})
	for _, rec := range recordsToProcess {
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

	for receiptNumber, keyInfo := range receiptKeysToDelete {
		log.Printf("Deleting existing DAT records for Receipt: %s, Date: %s, Client: %s", receiptNumber, keyInfo.Date, keyInfo.ClientCode)
		const q = `DELETE FROM transaction_records WHERE receipt_number = ? AND transaction_date = ? AND client_code = ? AND flag IN (1, 2)`
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

		mappers.MapMasterToTransaction(&transaction, master)

		if err := database.InsertTransactionRecord(tx, transaction); err != nil {
			return nil, fmt.Errorf("transaction insert failed for key %s: %w", key, err)
		}
		insertedTransactions = append(insertedTransactions, transaction)
	}
	return insertedTransactions, nil
}

func OpenFileHeader(fh *multipart.FileHeader) (multipart.File, error) {
	return fh.Open()
}
