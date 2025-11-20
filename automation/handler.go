// C:\Users\wasab\OneDrive\デスクトップ\TKR\automation\handler.go
package automation

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"tkr/config"
	"tkr/dat" // ★ ここで正しく dat パッケージをインポート

	"github.com/jmoiron/sqlx"
)

func writeJSONError(w http.ResponseWriter, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}

func DownloadMedicodeDatHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeJSONError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			writeJSONError(w, "設定の読み込みに失敗しました: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if cfg.MedicodeUserID == "" || cfg.MedicodePassword == "" {
			writeJSONError(w, "MEDICODEのIDまたはパスワードが設定されていません。設定画面で入力してください。", http.StatusBadRequest)
			return
		}

		saveDir := cfg.DatFolderPath
		if saveDir == "" {
			saveDir = os.TempDir()
			log.Printf("DAT保存先設定がないため、一時フォルダを使用します: %s", saveDir)
		}

		log.Println("Starting MEDICODE automation...")
		filePath, err := DownloadDat(cfg.MedicodeUserID, cfg.MedicodePassword, saveDir)

		if err != nil {
			log.Printf("Automation Error: %v", err)
			writeJSONError(w, "自動受信エラー: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if filePath == "NO_DATA" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{
				"status":  "no_data",
				"message": "未受信のデータはありませんでした。",
			})
			return
		}

		// ▼▼▼ 共通関数 ImportDatStream を呼び出し ▼▼▼
		log.Printf("Importing downloaded file via dat.ImportDatStream: %s", filePath)
		file, err := os.Open(filePath)
		if err != nil {
			writeJSONError(w, "ダウンロードファイルのオープンに失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer file.Close()

		insertedTransactions, err := dat.ImportDatStream(db, file, filepath.Base(filePath))
		if err != nil {
			writeJSONError(w, "DAT取込処理(dat.ImportDatStream)でエラー: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// ▲▲▲ ここまで ▲▲▲

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "success",
			"message":  fmt.Sprintf("ダウンロード＆登録完了: %d件", len(insertedTransactions)),
			"filePath": filePath,
			"records":  insertedTransactions,
		})
	}
}
