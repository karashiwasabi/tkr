// C:\Users\wasab\OneDrive\デスクトップ\TKR\dat\dat_upload_handler.go
package dat

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
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

		// ▼▼▼ 追加: 修正モードかどうかの判定 ▼▼▼
		mode := r.FormValue("mode")
		skipReconcile := (mode == "fix_only")
		if skipReconcile {
			log.Println("Mode: fix_only (Skipping backorder reconciliation)")
		}
		// ▲▲▲ 追加ここまで ▲▲▲

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

			// ▼▼▼ 修正: アーカイブと共通関数呼び出し ▼▼▼
			// 1. ファイルをメモリに読み込む（アーカイブ用とパース用）
			fileBytes, readErr := io.ReadAll(file)
			file.Close()
			if readErr != nil {
				fileResult["error"] = fmt.Sprintf("Failed to read file: %v", readErr)
				allResults = append(allResults, fileResult)
				continue
			}

			// 2. アーカイブ保存 (Sレコード解析)
			archivePath, archiveErr := archiveDatFile(fileBytes)
			if archiveErr != nil {
				log.Printf("WARN: Failed to archive DAT file: %v", archiveErr)
			} else if archivePath != "" {
				log.Printf("Archived DAT file to: %s", archivePath)
			} else {
				log.Printf("File already archived (skipped save).")
			}

			// 3. 共通関数を呼び出す (バイト列からReaderを作成)
			inserted, err := ImportDatStream(db, bytes.NewReader(fileBytes), fileHeader.Filename, skipReconcile)
			// ▲▲▲ 修正ここまで ▲▲▲

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

// archiveDatFile はSレコードを解析してファイルを保存します
func archiveDatFile(data []byte) (string, error) {
	// 保存先ディレクトリ
	archiveDir := "archive/dat"
	if err := os.MkdirAll(archiveDir, 0755); err != nil {
		return "", err
	}

	// Sレコードを探す
	scanner := bufio.NewScanner(bytes.NewReader(data))
	var sRecord string
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "S") {
			sRecord = line
			break
		}
	}

	if sRecord == "" || len(sRecord) < 39 {
		// Sレコードがない、または短すぎる場合は現在時刻で保存
		fileName := fmt.Sprintf("UNKNOWN_%s.DAT", time.Now().Format("20060102_150405"))
		return saveFileIfNotExists(filepath.Join(archiveDir, fileName), data)
	}

	// 解析: S20902020014 05262978172736251112050241
	// 日付: 27-33文字目 (251112)
	// 時刻: 33-39文字目 (050241)
	datePart := sRecord[27:33] // YYMMDD
	timePart := sRecord[33:39] // HHMMSS

	// 西暦補完 (25 -> 2025)
	fullDate := "20" + datePart

	fileName := fmt.Sprintf("%s_%s.DAT", fullDate, timePart)
	savePath := filepath.Join(archiveDir, fileName)

	return saveFileIfNotExists(savePath, data)
}

func saveFileIfNotExists(path string, data []byte) (string, error) {
	if _, err := os.Stat(path); err == nil {
		// ファイルが既に存在する -> 内容を確認すべきだが、今回は名前重複＝保存済みとみなす
		return "", nil
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return "", err
	}
	return path, nil
}

// ▼▼▼ 修正: skipReconcile 引数を追加 ▼▼▼
func ImportDatStream(db *sqlx.DB, r io.Reader, filename string, skipReconcile bool) ([]model.TransactionRecord, error) {
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

	// 発注残消込処理 (skipReconcileがfalseの場合のみ実行)
	if !skipReconcile && len(insertedTransactions) > 0 {
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

// ▲▲▲ 修正ここまで ▲▲▲

// ... (removeDatDuplicates, ProcessDatRecords, OpenFileHeader は前回の回答と同じ内容のため省略可能ですが、全体を要求されている場合は含めます)
// 前回の回答で修正した「マスタからの補完ロジック削除」済みの ProcessDatRecords を使用します。

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
			transaction.JanQuantity = transaction.YjQuantity / master.JanPackInnerQty
		} else {
			transaction.JanQuantity = 0
		}

		// マスタからの補完ロジックは削除済み

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
