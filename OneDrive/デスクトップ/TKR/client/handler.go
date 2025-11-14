// C:\Users\wasab\OneDrive\デスクトップ\TKR\client\handler.go
package client

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"  // ▼▼▼【追加】正規表現パッケージをインポート ▼▼▼
	"strings" // ▼▼▼【追加】strings パッケージをインポート ▼▼▼
	"tkr/database"
	"tkr/parsers"

	"github.com/jmoiron/sqlx"
)

// ▼▼▼【ここに追加】コード形式を判定するための正規表現 ▼▼▼
var (
	// 「CL」で始まるコード（得意先コード）
	clientCodeRegex = regexp.MustCompile(`^CL`)
	// 9桁の数字（卸コード）
	wholesalerCodeRegex = regexp.MustCompile(`^[0-9]{9}$`)
)

// ▲▲▲【追加ここまで】▲▲▲

// ImportClientsHandler は得意先マスタCSVのインポートを処理します。
func ImportClientsHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "CSVファイルの読み取りに失敗: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		records, err := parsers.ParseClientCSV(file)
		if err != nil {
			http.Error(w, "CSVファイルの解析に失敗: "+err.Error(), http.StatusBadRequest)
			return
		}

		if len(records) == 0 {
			http.Error(w, "CSVから読み込むデータがありません。", http.StatusBadRequest)
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			http.Error(w, "データベーストランザクションの開始に失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		var importedClients int
		var importedWholesalers int
		var errors []string

		for _, rec := range records {
			// ▼▼▼【ここから修正】振り分けロジック ▼▼▼
			clientCode := strings.TrimSpace(rec.ClientCode)
			clientName := strings.TrimSpace(rec.ClientName)

			if clientCodeRegex.MatchString(clientCode) {
				// 1. 得意先コードの条件 ("CL"で始まる) に一致
				if err := database.UpsertClientInTx(tx, clientCode, clientName); err != nil {
					log.Printf("ERROR: Failed to upsert client %s (Name: %s): %v", clientCode, clientName, err)
					errors = append(errors, fmt.Sprintf("得意先 コード %s (名称: %s): %v", clientCode, clientName, err))
				} else {
					importedClients++
				}
			} else if wholesalerCodeRegex.MatchString(clientCode) {
				// 2. 卸コードの条件 (9桁の数字) に一致
				if err := database.UpsertWholesalerInTx(tx, clientCode, clientName); err != nil {
					log.Printf("ERROR: Failed to upsert wholesaler %s (Name: %s): %v", clientCode, clientName, err)
					errors = append(errors, fmt.Sprintf("卸 コード %s (名称: %s): %v", clientCode, clientName, err))
				} else {
					importedWholesalers++
				}
			} else {
				// 3. どちらの条件にも一致しない (スキップ)
				log.Printf("WARN: Skipped CSV row. Code '%s' is neither a client code (CL...) nor a wholesaler code (9 digits).", clientCode)
				errors = append(errors, fmt.Sprintf("スキップ: コード %s (形式不正)", clientCode))
			}
			// ▲▲▲【修正ここまで】▲▲▲
		}

		if err := tx.Commit(); err != nil { //
			http.Error(w, "データベースのコミットに失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// ▼▼▼【ここから修正】メッセージを変更 ▼▼▼
		message := fmt.Sprintf("インポート完了。\n得意先: %d件\n卸: %d件", importedClients, importedWholesalers)
		// ▲▲▲【修正ここまで】▲▲▲
		if len(errors) > 0 {
			message += fmt.Sprintf("\n%d件のエラーまたはスキップが発生しました。", len(errors))
		}

		// (JSの handleFileUpload は results 配列を期待するため、空でも追加)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": message,
			"results": []interface{}{}, // 空の results を返す
		})
	}
}
