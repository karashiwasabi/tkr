// C:\Users\wasab\OneDrive\デスクトップ\TKR\usage\handler.go
package usage

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"tkr/config"
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

func UploadUsageHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var processedFiles []string
		var successfullyInsertedTransactions []model.TransactionRecord
		var allResults []map[string]interface{}

		if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			// --- A. 手動アップロード (複数ファイル対応) ---
			log.Println("Processing manual USAGE file upload...")
			err := r.ParseMultipartForm(32 << 20) // 32MB
			if err != nil {
				respondJSONError(w, "File upload error: "+err.Error(), http.StatusBadRequest)
				return
			}
			defer r.MultipartForm.RemoveAll()

			for _, fileHeader := range r.MultipartForm.File["file"] {
				log.Printf("Processing file: %s", fileHeader.Filename)
				// ▼▼▼【SA4010 修正】appendの結果を再代入する ▼▼▼
				processedFiles = append(processedFiles, fileHeader.Filename)
				// ▲▲▲【修正ここまで】▲▲▲
				fileResult := map[string]interface{}{"filename": fileHeader.Filename}

				file, openErr := fileHeader.Open()
				if openErr != nil {
					log.Printf("Failed to open uploaded file %s: %v", fileHeader.Filename, openErr)
					fileResult["error"] = fmt.Sprintf("Failed to open file: %v", openErr)
					allResults = append(allResults, fileResult)
					continue
				}

				processedRecords, procErr := processUsageFile(db, file)
				file.Close() // ファイルごとに閉じる

				if procErr != nil {
					log.Printf("Failed to process USAGE file %s: %v", fileHeader.Filename, procErr)
					fileResult["error"] = fmt.Sprintf("Failed to process USAGE file: %v", procErr)
					allResults = append(allResults, fileResult)
					continue
				}

				log.Printf("Successfully processed %d records from %s", len(processedRecords), fileHeader.Filename)
				fileResult["success"] = true
				fileResult["records_parsed"] = len(processedRecords) // USAGEではパース=登録
				fileResult["records_inserted"] = len(processedRecords)
				allResults = append(allResults, fileResult)
				successfullyInsertedTransactions = append(successfullyInsertedTransactions, processedRecords...)
			}

		} else {
			// --- B. 自動アップロード (単一ファイル) ---
			log.Println("Processing automatic USAGE file import...")
			cfg, cfgErr := config.LoadConfig()
			if cfgErr != nil {
				respondJSONError(w, "設定ファイルの読み込みに失敗: "+cfgErr.Error(), http.StatusInternalServerError)
				return
			}
			if cfg.UsageFolderPath == "" {
				respondJSONError(w, "処方取込フォルダパス(usageFolderPath)が設定されていません。", http.StatusBadRequest)
				return
			}

			rawPath := cfg.UsageFolderPath
			unquotedPath := strings.Trim(strings.TrimSpace(rawPath), "\"")
			filePath := strings.ReplaceAll(unquotedPath, "\\", "/")

			log.Printf("Opening specified USAGE file: %s", filePath)
			f, fErr := os.Open(filePath)
			if fErr != nil {
				displayError := fmt.Sprintf("設定されたパスのファイルを開けませんでした。\nパス: %s\nエラー: %v", filePath, fErr)
				respondJSONError(w, displayError, http.StatusInternalServerError)
				return
			}
			defer f.Close()

			fileResult := map[string]interface{}{"filename": filePath}
			processedRecords, procErr := processUsageFile(db, f)
			if procErr != nil {
				fileResult["error"] = procErr.Error()
				allResults = append(allResults, fileResult)
			} else {
				fileResult["success"] = true
				fileResult["records_parsed"] = len(processedRecords)
				fileResult["records_inserted"] = len(processedRecords)
				allResults = append(allResults, fileResult)
				successfullyInsertedTransactions = processedRecords
			}
		}

		// --- 共通レスポンス ---
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": fmt.Sprintf("Processed %d USAGE file(s). See results for details.", len(allResults)),
			"results": allResults,
			"records": successfullyInsertedTransactions,
		})
	}
}

func processUsageFile(db *sqlx.DB, file io.Reader) ([]model.TransactionRecord, error) {
	parsed, err := parsers.ParseUsage(file)
	if err != nil {
		return nil, fmt.Errorf("USAGEファイルの解析に失敗しました: %w", err)
	}

	filtered := removeUsageDuplicates(parsed)
	if len(filtered) == 0 {
		return []model.TransactionRecord{}, nil
	}

	tx, err := db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer tx.Rollback()

	minDate, maxDate := "99999999", "00000000"
	for _, rec := range filtered {
		if rec.Date < minDate {
			minDate = rec.Date
		}
		if rec.Date > maxDate {
			maxDate = rec.Date
		}
	}

	if err := database.DeleteUsageTransactionsInDateRange(tx, minDate, maxDate); err != nil {
		return nil, fmt.Errorf("既存の処方データ削除に失敗: %w", err)
	}

	var finalRecords []model.TransactionRecord
	for i, rec := range filtered {
		key := rec.JanCode
		if key == "" {
			key = rec.YjCode
			if key == "" {
				key = fmt.Sprintf("9999999999999%s", rec.ProductName)
			}
		}

		master, err := mastermanager.FindOrCreateMaster(tx, key, rec.ProductName)
		if err != nil {
			return nil, fmt.Errorf("マスターの特定/作成に失敗 (Key: %s, Name: %s): %w", key, rec.ProductName, err)
		}

		transaction := model.TransactionRecord{
			TransactionDate: rec.Date,
			ClientCode:      "",
			ReceiptNumber:   fmt.Sprintf("USAGE-%s", rec.Date),
			LineNumber:      strconv.Itoa(i + 1),
			Flag:            3, // 処方
			YjQuantity:      rec.YjQuantity,
		}

		if master.JanPackInnerQty > 0 {
			transaction.JanQuantity = rec.YjQuantity / master.JanPackInnerQty
		}

		transaction.UnitPrice = master.NhiPrice
		transaction.Subtotal = rec.YjQuantity * master.NhiPrice

		mappers.MapMasterToTransaction(&transaction, master)

		if err := database.InsertTransactionRecord(tx, transaction); err != nil {
			return nil, fmt.Errorf("処方レコードの挿入に失敗 (JAN: %s): %w", transaction.JanCode, err)
		}

		finalRecords = append(finalRecords, transaction)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}

	return finalRecords, nil
}

func removeUsageDuplicates(records []model.UnifiedInputRecord) []model.UnifiedInputRecord {
	seen := make(map[string]struct{})
	var result []model.UnifiedInputRecord
	for _, r := range records {
		key := fmt.Sprintf("%s|%s|%s|%s", r.Date, r.JanCode, r.YjCode, r.ProductName)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, r)
	}
	return result
}
