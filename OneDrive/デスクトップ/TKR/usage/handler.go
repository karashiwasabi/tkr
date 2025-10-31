// C:\Users\wasab\OneDrive\デスクトップ\TKR\usage\handler.go
package usage

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"tkr/config"
	"tkr/database"
	"tkr/mastermanager"
	"tkr/model"
	"tkr/parsers"

	"github.com/jmoiron/sqlx"
)

var (
	importMutex sync.Mutex
)

// GetUsageConfigHandler retrieves the current usage import configuration.
func GetUsageConfigHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := config.GetConfig()
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(cfg)
	}
}

// SaveUsageConfigHandler saves the usage import configuration.
func SaveUsageConfigHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var newCfg config.Config
		if err := json.NewDecoder(r.Body).Decode(&newCfg); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		// Basic validation: Check if path exists and is a directory (optional but recommended)
		if newCfg.UsageFolderPath != "" {
			info, err := os.Stat(newCfg.UsageFolderPath)
			if err != nil {
				if os.IsNotExist(err) {
					http.Error(w, fmt.Sprintf("フォルダが見つかりません: %s", newCfg.UsageFolderPath), http.StatusBadRequest)
					return
				}
				log.Printf("Error checking folder path '%s': %v", newCfg.UsageFolderPath, err)
				http.Error(w, "フォルダパスの確認中にエラーが発生しました", http.StatusInternalServerError)
				return
			}
			if !info.IsDir() {
				http.Error(w, fmt.Sprintf("指定されたパスはフォルダではありません: %s", newCfg.UsageFolderPath), http.StatusBadRequest)
				return
			}
		}

		if err := config.SaveConfig(newCfg); err != nil {
			log.Printf("Error saving config: %v", err)
			http.Error(w, "設定の保存に失敗しました", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "設定を保存しました。"})
	}
}

// ImportUsageHandler manually triggers the import of usage files from the configured folder.
func ImportUsageHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Prevent concurrent imports
		if !importMutex.TryLock() {
			http.Error(w, "他の取り込み処理が実行中です。しばらく待ってから再試行してください。", http.StatusConflict)
			return
		}
		defer importMutex.Unlock()

		cfg := config.GetConfig()
		folderPath := cfg.UsageFolderPath
		if folderPath == "" {
			http.Error(w, "処方ファイル取込フォルダが設定されていません。", http.StatusBadRequest)
			return
		}

		log.Printf("Starting manual usage import from: %s", folderPath)

		files, err := os.ReadDir(folderPath)
		if err != nil {
			log.Printf("Error reading usage directory '%s': %v", folderPath, err)
			http.Error(w, fmt.Sprintf("フォルダの読み取りに失敗しました: %s", folderPath), http.StatusInternalServerError)
			return
		}

		var importResults []map[string]interface{}
		var overallSuccess = true
		var processedFileCount int

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(strings.ToLower(file.Name()), ".csv") {
				continue // Skip directories and non-CSV files
			}

			processedFileCount++
			filePath := filepath.Join(folderPath, file.Name())
			log.Printf("Processing usage file: %s", filePath)
			fileResult := map[string]interface{}{
				"fileName":       file.Name(),
				"timestamp":      time.Now().Format(time.RFC3339),
				"success":        false,
				"processedCount": 0,
				"error":          "",
			}

			err := processSingleUsageFile(db, filePath)
			if err != nil {
				log.Printf("Error processing file %s: %v", file.Name(), err)
				fileResult["error"] = err.Error()
				overallSuccess = false
			} else {
				fileResult["success"] = true
				// TODO: Add processed count if needed by modifying processSingleUsageFile
			}
			importResults = append(importResults, fileResult)

			// Optionally move or delete processed file here
			// os.Rename(filePath, filePath+".processed") or os.Remove(filePath)
		}

		log.Printf("Finished manual usage import. Processed %d CSV files.", processedFileCount)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": fmt.Sprintf("%d 件のCSVファイルを処理しました。", processedFileCount),
			"success": overallSuccess, // Overall success flag
			"history": importResults,  // Results for each file
		})
	}
}

// processSingleUsageFile handles the parsing and DB insertion for one usage file.
func processSingleUsageFile(db *sqlx.DB, filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("ファイルを開けません: %w", err)
	}
	defer file.Close()

	parsedRecords, err := parsers.ParseUsage(file)
	if err != nil {
		return fmt.Errorf("ファイルのパースに失敗しました: %w", err)
	}

	if len(parsedRecords) == 0 {
		log.Printf("ファイル %s に有効なレコードがありません。", filepath.Base(filePath))
		return nil // Not an error, just empty
	}

	// Determine date range for deletion
	minDate, maxDate := parsedRecords[0].Date, parsedRecords[0].Date
	for _, rec := range parsedRecords {
		if rec.Date < minDate {
			minDate = rec.Date
		}
		if rec.Date > maxDate {
			maxDate = rec.Date
		}
	}

	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("トランザクションの開始に失敗しました: %w", err)
	}
	defer tx.Rollback() // Rollback on any error

	// Delete existing records in the date range
	if err := database.DeleteUsageTransactionsInDateRange(tx, minDate, maxDate); err != nil {
		return fmt.Errorf("既存レコードの削除に失敗しました (%s - %s): %w", minDate, maxDate, err)
	}

	// Assume a default client or derive from filename if needed. WASABI derived from filename.
	// For simplicity, let's try finding/creating a default client "CL0000" / "処方取込"
	defaultClientCode := "CL0000"
	defaultClientName := "処方取込"
	clientExists, err := database.CheckClientExistsByName(tx, defaultClientName)
	if err != nil {
		return fmt.Errorf("得意先 '%s' の存在確認に失敗しました: %w", defaultClientName, err)
	}
	if !clientExists {
		// Get next CL code if "CL0000" doesn't exist either
		var actualClientCode string
		err = tx.Get(&actualClientCode, "SELECT client_code FROM client_master WHERE client_code=?", defaultClientCode)
		if err != nil {
			if err == sql.ErrNoRows {
				// CL0000 doesn't exist, generate new one
				newCode, seqErr := database.NextSequenceInTx(tx, "CL", "CL", 4)
				if seqErr != nil {
					return fmt.Errorf("新規得意先コードの採番に失敗しました: %w", seqErr)
				}
				actualClientCode = newCode
				if err := database.CreateClientInTx(tx, actualClientCode, defaultClientName); err != nil {
					return fmt.Errorf("新規得意先 '%s' (%s) の作成に失敗しました: %w", defaultClientName, actualClientCode, err)
				}
			} else {
				return fmt.Errorf("得意先 '%s' の確認中にエラー: %w", defaultClientCode, err)
			}
		} else {
			actualClientCode = defaultClientCode // CL0000 exists, use it
		}
		defaultClientCode = actualClientCode
	} else {
		// Client exists by name, get its code
		err = tx.Get(&defaultClientCode, "SELECT client_code FROM client_master WHERE client_name=?", defaultClientName)
		if err != nil {
			return fmt.Errorf("得意先コード '%s' の取得に失敗しました: %w", defaultClientName, err)
		}
	}

	// Insert new records
	for i, rec := range parsedRecords {
		// Find or create master using YJ code primarily, then JAN code
		// ▼▼▼【ここから修正】キー選択ロジックを WASABI の YJ 優先に設定 ▼▼▼
		masterKey := rec.YjCode // WASABIのロジック: まず YjCode を使う
		if masterKey == "" {
			masterKey = rec.JanCode // YjCode が空なら JanCode を使う
		}
		// ▲▲▲【修正ここまで】▲▲▲

		// Skip if both are empty? Or create provisional based on product name?
		if masterKey == "" {
			log.Printf("WARN: Skipping usage record %d in %s - YJ and JAN codes are empty. Product: %s", i+1, filepath.Base(filePath), rec.ProductName)
			continue
		}

		master, err := mastermanager.FindOrCreateMaster(tx, masterKey, rec.ProductName)
		if err != nil {
			// Log error and fail the whole file for now.
			return fmt.Errorf("レコード %d のマスター処理に失敗しました (Key: %s, Name: %s): %w", i+1, masterKey, rec.ProductName, err)
		}

		// Map to TransactionRecord
		transaction := model.TransactionRecord{
			TransactionDate:     rec.Date,
			ClientCode:          defaultClientCode,      // Use the determined client code
			ReceiptNumber:       "",                     // Usage doesn't have receipt number
			LineNumber:          fmt.Sprintf("%d", i+1), // Use line number
			Flag:                3,                      // 3 for Usage
			JanCode:             master.ProductCode,     // Use code from master (should be JAN or synthetic JAN)
			YjCode:              master.YjCode,
			ProductName:         master.ProductName,
			KanaName:            master.KanaName,
			UsageClassification: master.UsageClassification,
			PackageForm:         master.PackageForm,
			// PackageSpec needs calculation like in DAT handler
			PackageSpec:       fmt.Sprintf("%s %v%s", master.PackageForm, master.YjPackUnitQty, master.YjUnitName), // Basic spec
			MakerName:         master.MakerName,
			DatQuantity:       0, // Not from DAT
			JanPackInnerQty:   master.JanPackInnerQty,
			JanQuantity:       0, // To be calculated if needed
			JanPackUnitQty:    master.JanPackUnitQty,
			JanUnitName:       "", // Determine if needed
			JanUnitCode:       fmt.Sprintf("%d", master.JanUnitCode),
			YjQuantity:        rec.YjQuantity, // From usage file
			YjPackUnitQty:     master.YjPackUnitQty,
			YjUnitName:        rec.YjUnitName, // Prefer unit name from usage file
			UnitPrice:         0,              // Usage doesn't have price
			PurchasePrice:     master.PurchasePrice,
			SupplierWholesale: master.SupplierWholesale,
			Subtotal:          0,
			// Fill flags from master
			FlagPoison:       master.FlagPoison,
			FlagDeleterious:  master.FlagDeleterious,
			FlagNarcotic:     master.FlagNarcotic,
			FlagPsychotropic: master.FlagPsychotropic,
			FlagStimulant:    master.FlagStimulant,
			FlagStimulantRaw: master.FlagStimulantRaw,
		}

		if err := database.InsertTransactionRecord(tx, transaction); err != nil {
			return fmt.Errorf("レコード %d の登録に失敗しました (YJ: %s, JAN: %s): %w", i+1, rec.YjCode, rec.JanCode, err)
		}
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("トランザクションのコミットに失敗しました: %w", err)
	}

	return nil
}
