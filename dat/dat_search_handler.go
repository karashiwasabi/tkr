// C:\Users\wasab\OneDrive\デスクトップ\TKR\dat\dat_search_handler.go
package dat

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"tkr/barcode"
	"tkr/database"

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

		var expiryYYYYMM, lotNumber string
		if len(barcodeStr) > 14 {
			gs1Result, parseErr := barcode.Parse(barcodeStr)
			if parseErr == nil && gs1Result != nil {
				expiryYYYYMM = gs1Result.ExpiryDate
				lotNumber = gs1Result.LotNumber
			}
		}

		log.Printf("Search criteria: ProductCode(JAN)='%s', Expiry(YYYYMM)='%s', Lot='%s'",
			productCode, expiryYYYYMM, lotNumber)

		transactions, err := database.SearchTransactions(db, productCode, expiryYYYYMM, lotNumber)

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
