// C:\Users\wasab\OneDrive\デスクトップ\TKR\usage\handler.go
package usage

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
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
		var file io.Reader
		var err error

		if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			log.Println("Processing manual USAGE file upload...")
			var f multipart.File
			f, _, err = r.FormFile("file")
			if err != nil {
				respondJSONError(w, "ファイルの取得に失敗しました: "+err.Error(), http.StatusBadRequest)
				return
			}
			defer f.Close()
			file = f
		} else {
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
			file = f
		}

		processedRecords, procErr := processUsageFile(db, file)
		if procErr != nil {
			respondJSONError(w, procErr.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": fmt.Sprintf("%d件の処方データを処理しました。", len(processedRecords)),
			"results": []interface{}{},
			"records": processedRecords,
		})
	}
}

// ▼▼▼【ここから修正】dat/handler.go のロジック (  ) に合わせて書き換え ▼▼▼
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
		if key == "" || key == "0000000000000" {
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

// ▲▲▲【修正ここまで】▲▲▲

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
