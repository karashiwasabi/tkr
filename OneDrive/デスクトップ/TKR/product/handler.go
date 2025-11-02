// C:\Users\wasab\OneDrive\デスクトップ\TKR\product\handler.go
package product

import (
	"database/sql"

	// ▼▼▼【ここに追加】▼▼▼
	"encoding/json"
	"errors" // ▼▼▼【ここに追加】▼▼▼
	"fmt"    // ▼▼▼【ここに追加】▼▼▼
	"log"    // ▼▼▼【ここに追加】▼▼▼
	"net/http"
	"sort"
	"strings"
	"tkr/aggregation" // 集計ロジック

	// ▼▼▼【ここに追加】▼▼▼
	"tkr/database"
	"tkr/mappers" // Viewマッパー
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

// SearchProductsHandler は棚卸調整画面での品目検索（YJコード単位）を行います。
func SearchProductsHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filters := model.AggregationFilters{
			KanaName:    q.Get("kanaName"),
			DosageForm:  q.Get("dosageForm"),
			ShelfNumber: q.Get("shelfNumber"),
			GenericName: q.Get("genericName"), // ▼▼▼【ここに追加】一般名 ▼▼▼
		}

		var results []model.ProductMasterView

		mastersByYjCode, yjCodes, err := aggregation.GetFilteredMastersAndYjCodes(conn, filters)
		if err != nil {
			http.Error(w, "Failed to search products: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// YJコードごとに代表マスターを1つだけ選んで返す
		seenYjCodes := make(map[string]bool)
		for _, yjCode := range yjCodes { // yjCodes は aggregation でソート済みの順序
			if seenYjCodes[yjCode] {
				continue
			}
			mastersInGroup := mastersByYjCode[yjCode]
			if len(mastersInGroup) > 0 {
				repMaster := mastersInGroup[0]
				// ▼▼▼【ここを修正】if と { を同じ行にする ▼▼▼
				for _, m := range mastersInGroup {
					if m.Origin == "JCSHMS" {
						// ▲▲▲【修正ここまで】▲▲▲
						repMaster = m // JCSHMSを優先
						break
					}
				}
				view := mappers.ToProductMasterView(repMaster)
				results = append(results, view)
				seenYjCodes[yjCode] = true
			}
		}

		// ▼▼▼【ここに追加】Go側でも最終結果をカナ名でソートし、ランダムな並びを防ぐ ▼▼▼
		sort.Slice(results, func(i, j int) bool {
			return results[i].KanaName < results[j].KanaName
		})
		// ▲▲▲【追加ここまで】▲▲▲

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}

// ▼▼▼【ここから修正】バーコード検索を単一APIに共通化（DB共通関数を使用） ▼▼▼
func GetProductByBarcodeHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		barcodeStr := strings.TrimPrefix(r.URL.Path, "/api/product/by_barcode/")
		if barcodeStr == "" {
			http.Error(w, "barcode is required", http.StatusBadRequest)
			return
		}

		log.Printf("API request received for barcode: %s", barcodeStr)

		// DB共通関数でマスターを検索
		master, err := database.GetProductMasterByBarcode(conn, barcodeStr)

		// 共通のエラーハンドリング
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Printf("Product master not found for barcode: %s", barcodeStr)
				http.Error(w, "Product not found", http.StatusNotFound)
			} else {
				log.Printf("Error searching product master by barcode %s: %v", barcodeStr, err)
				http.Error(w, fmt.Sprintf("マスター検索エラー: %v", err), http.StatusInternalServerError)
			}
			return
		}

		masterView := mappers.ToProductMasterView(master)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(masterView)
	}
}

// ▲▲▲【修正ここまで】▲▲▲

// ▼▼▼【ここから削除】古いハンドラを削除 ▼▼▼
/*
func GetProductByGS1Handler(conn *sqlx.DB) http.HandlerFunc
{
	return func(w http.ResponseWriter, r *http.Request) {
// ... (古いコード [cite: 804] 削除) ...
	}
}
func GetMasterByCodeHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
// ... (古いコード [cite: 804] 削除) ...
	}
}
*/
// ▲▲▲【削除ここまで】▲▲▲
