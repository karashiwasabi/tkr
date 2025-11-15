// C:\Users\wasab\OneDrive\デスクトップ\TKR\pricing\handler.go
package pricing

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"tkr/database"
	"tkr/mappers"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

// writeJsonError はエラーレスポンスを返します
func writeJsonError(w http.ResponseWriter, message string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]string{"message": message})
}

// QuoteDataWithSpec は価格比較画面用のデータ構造です
type QuoteDataWithSpec struct {
	model.ProductMaster
	FormattedPackageSpec string             `json:"formattedPackageSpec"`
	Quotes               map[string]float64 `json:"quotes"`
}

// UploadResponse はアップロード結果のデータ構造です
type UploadResponse struct {
	ProductData     []QuoteDataWithSpec `json:"productData"`
	WholesalerOrder []string            `json:"wholesalerOrder"`
}

// BulkUpdateHandler は価格比較画面からのマスター更新を処理します
func BulkUpdateHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload []model.PriceUpdate
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			writeJsonError(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			writeJsonError(w,
				"Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		if err := database.UpdatePricesAndSuppliersInTx(tx, payload); err != nil {
			writeJsonError(w, "Failed to update prices: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			writeJsonError(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("%d件の医薬品マスターを更新しました。", len(payload)),
		})
	}
}

// GetAllMastersForPricingHandler は価格比較画面の初期表示用データを返します
func GetAllMastersForPricingHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allMasters, err := database.GetAllProductMasters(db)
		if err != nil {
			writeJsonError(w, "Failed to get all product masters for pricing", http.StatusInternalServerError)
			return
		}
		allQuotes, err := database.GetAllProductQuotes(db)
		if err != nil {
			writeJsonError(w, "Failed to get product quotes", http.StatusInternalServerError)
			return
		}
		wholesalerMasterMap, err := database.GetWholesalerMap(db)
		if err != nil {
			writeJsonError(w, "卸マスタの取得に失敗しました: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var wholesalerOrder []string
		for _, name := range wholesalerMasterMap {
			wholesalerOrder = append(wholesalerOrder, name)
		}
		sort.Strings(wholesalerOrder)

		responseData := make([]QuoteDataWithSpec, 0, len(allMasters))
		for _, master := range allMasters {
			view := mappers.ToProductMasterView(master)
			formattedSpec := view.FormattedPackageSpec

			quotesForThisProduct := make(map[string]float64)
			if quotes, ok :=
				allQuotes[master.ProductCode]; ok {
				for wCode, price := range quotes {
					if wName, ok := wholesalerMasterMap[wCode]; ok {
						quotesForThisProduct[wName] = price
					}
				}
			}

			responseData = append(responseData, QuoteDataWithSpec{
				ProductMaster:        *master,
				FormattedPackageSpec: formattedSpec,
				Quotes:               quotesForThisProduct,
			})
		}

		finalResponse := UploadResponse{
			ProductData:     responseData,
			WholesalerOrder: wholesalerOrder,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(finalResponse)
	}
}
