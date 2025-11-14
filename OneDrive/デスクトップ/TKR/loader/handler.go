// C:\Users\wasab\OneDrive\デスクトップ\TKR\loader\handler.go
package loader

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"tkr/database"

	"github.com/jmoiron/sqlx"
)

// ReloadJCSHMSHandler は JCSHMS.CSV と JANCODE.CSV の再読み込みをトリガーします。
func ReloadJCSHMSHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("HTTP request received: Reloading JCSHMS and JANCODE...")

		jcshmsPath := "SOU/JCSHMS.CSV"
		jancodePath := "SOU/JANCODE.CSV"

		// JCSHMS.CSV のロード
		if _, err := os.Stat(jcshmsPath); os.IsNotExist(err) {
			log.Printf("WARN: %s not found, skipping.", jcshmsPath)
		} else {
			log.Printf("Reloading %s...", jcshmsPath)
			if err := LoadCSV(db, jcshmsPath, "jcshms", 125, false); err != nil {
				msg := fmt.Sprintf("failed to reload %s: %v", jcshmsPath, err)
				log.Println(msg)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}
			log.Printf("Reloaded %s successfully.", jcshmsPath)
		}

		// JANCODE.CSV のロード
		if _, err := os.Stat(jancodePath); os.IsNotExist(err) {
			log.Printf("WARN: %s not found, skipping.", jancodePath)
		} else {
			log.Printf("Reloading %s...", jancodePath)
			if err := LoadCSV(db, jancodePath, "jancode", 30, true); err != nil {
				msg := fmt.Sprintf("failed to reload %s: %v", jancodePath, err)
				log.Println(msg)
				http.Error(w, msg, http.StatusInternalServerError)
				return
			}
			log.Printf("Reloaded %s successfully.", jancodePath)
		}

		// シーケンスの再初期化
		tx, err := db.Beginx()
		if err != nil {
			msg := fmt.Sprintf("failed to begin transaction for sequence initialization: %v", err)
			log.Println(msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
		defer tx.Rollback() // エラー時

		if err := database.InitializeSequenceFromMaxYjCode(tx); err != nil {
			log.Printf("WARN: Failed to re-initialize MA2Y sequence: %v", err)
			// エラーでも続行
		}
		if err := database.InitializeSequenceFromMaxProductCode(tx); err != nil {
			log.Printf("WARN: Failed to re-initialize MA2J sequence: %v", err)
			// エラーでも続行
		}

		if err := tx.Commit(); err != nil {
			msg := fmt.Sprintf("failed to commit sequence initialization: %v", err)
			log.Println(msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}
		log.Println("Code sequences re-initialized.")

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "JCSHMS (SOU) マスターの更新が完了しました。",
		})
	}
}
