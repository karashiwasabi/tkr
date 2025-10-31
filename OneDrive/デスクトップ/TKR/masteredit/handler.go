// C:\Users\wasab\OneDrive\デスクトップ\TKR\masteredit\handler.go (全体)
package masteredit

import (
	"encoding/json"
	"log"
	"net/http"
	"tkr/database"
	"tkr/mastermanager"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

func ListMastersHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		queryParams := r.URL.Query()
		usageClass := queryParams.Get("usage_class")
		productName := queryParams.Get("product_name")
		kanaName := queryParams.Get("kana_name")
		genericName := queryParams.Get("generic_name")

		shelfNumber := queryParams.Get("shelf_number")

		masters, err := database.GetFilteredProductMasters(db, usageClass, productName, kanaName, genericName, shelfNumber)

		if err != nil {
			log.Printf("Error fetching filtered product masters: %v", err)
			http.Error(w, "Failed to retrieve product masters", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(masters); err != nil {
			log.Printf("Error encoding product masters to JSON: %v", err)
		}
	}
}

func UpdateMasterHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		var input model.ProductMasterInput

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			log.Printf("UpdateMasterHandler: Invalid request body: %v", err)
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if input.ProductCode == "" {
			log.Println("UpdateMasterHandler: Product Code (JAN) cannot be empty.")
			http.Error(w, "Product Code (JAN) cannot be empty.", http.StatusBadRequest)
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			log.Printf("UpdateMasterHandler: Failed to start transaction: %v", err)
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		if _, err := mastermanager.UpsertProductMasterSqlx(tx, input); err != nil {
			log.Printf("UpdateMasterHandler: Failed to upsert product master (JAN: %s): %v", input.ProductCode, err)
			http.Error(w, "Failed to upsert product master", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			log.Printf("UpdateMasterHandler: Failed to commit transaction (JAN: %s): %v", input.ProductCode, err)
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		log.Printf("UpdateMasterHandler: Successfully saved master (JAN: %s)", input.ProductCode)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "Saved successfully."})
	}
}
