// C:\Users\wasab\OneDrive\デスクトップ\TKR\deadstock\handler.go
package deadstock

import (
	"encoding/json"
	"net/http"
	"tkr/database"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

type DeadStockListResponse struct {
	Items  []model.DeadStockItem `json:"items"`
	Errors []string              `json:"errors"`
}

func ListDeadStockHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startDate := r.URL.Query().Get("startDate") // YYYYMMDD
		endDate := r.URL.Query().Get("endDate")     // YYYYMMDD

		if startDate == "" || endDate == "" {
			http.Error(w, "startDate and endDate (YYYYMMDD) are required.", http.StatusBadRequest)
			return
		}

		items, err := database.GetDeadStockList(db, startDate, endDate)
		if err != nil {
			http.Error(w, "Failed to get dead stock list: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DeadStockListResponse{
			Items:  items,
			Errors: nil,
		})
	}
}
