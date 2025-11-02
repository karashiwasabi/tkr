// C:\Users\wasab\OneDrive\デスクトップ\TKR\product\handler.go
package product

import (
	"encoding/json"
	"net/http"
	"sort" // ▼▼▼【ここに追加】ソート用 ▼▼▼
	"strings"
	"tkr/aggregation" // 集計ロジック
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
				for _, m := range mastersInGroup {
					if m.Origin == "JCSHMS" {
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

// (GetProductByGS1Handler, GetMasterByCodeHandler は変更なし)
// ...
func GetProductByGS1Handler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		gs1Code := r.URL.Query().Get("gs1_code")
		if gs1Code == "" {
			http.Error(w, "gs1_code is required", http.StatusBadRequest)
			return
		}
		master, err := database.GetProductMasterByGs1Code(conn, gs1Code)
		if err != nil {
			if strings.Contains(err.Error(), "no rows") {
				http.Error(w, "Product not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to get product by gs1 code: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}
		masterView := mappers.ToProductMasterView(master)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(masterView)
	}
}
func GetMasterByCodeHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		productCode := strings.TrimPrefix(r.URL.Path, "/api/master/by_code/")
		if productCode == "" {
			http.Error(w, "Product code is required", http.StatusBadRequest)
			return
		}
		master, err := database.GetProductMasterByCode(conn, productCode)
		if err != nil {
			if strings.Contains(err.Error(), "no rows") {
				http.Error(w, "Product not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to get product by code: "+err.Error(), http.StatusInternalServerError)
			}
			return
		}
		masterView := mappers.ToProductMasterView(master)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(masterView)
	}
}
