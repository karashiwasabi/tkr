// C:\Users\wasab\OneDrive\デスクトップ\TKR\product\handler.go
package product

import (
	"encoding/json"
	"net/http"
	"strings"
	"tkr/aggregation" // 集計ロジック
	"tkr/database"
	"tkr/mappers" // Viewマッパー
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

// SearchProductsHandler は棚卸調整画面での品目検索（YJコード単位）を行います。
// (WASABI: product/handler.go  より移植・TKR用に修正)
func SearchProductsHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filters := model.AggregationFilters{
			KanaName:    q.Get("kanaName"),
			DosageForm:  q.Get("dosageForm"),
			ShelfNumber: q.Get("shelfNumber"),
			// TKRでは YjCode, DrugTypes, MovementOnly は検索条件として使用しない
		}

		var results []model.ProductMasterView

		// aggregation.go に移植したヘルパー関数を使用
		// ▼▼▼【修正】Gを大文字に ▼▼▼
		mastersByYjCode, yjCodes, err := aggregation.GetFilteredMastersAndYjCodes(conn, filters)
		// ▲▲▲【修正ここまで】▲▲▲
		if err != nil {
			http.Error(w, "Failed to search products: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// YJコードごとに代表マスターを1つだけ選んで返す
		seenYjCodes := make(map[string]bool)
		for _, yjCode := range yjCodes {
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

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}

// GetProductByGS1Handler はGS1コードで品目を検索します。
// (TKR: database/product_master_query.go の GetProductMasterByGs1Code を呼び出す)
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

// GetMasterByCodeHandler はJANコードで品目を検索します。
// (TKR: database/product_master_query.go の GetProductMasterByCode を呼び出す)
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
