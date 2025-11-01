// C:\Users\wasab\OneDrive\デスクトップ\TKR\wholesaler_handler.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"tkr/database"

	"github.com/jmoiron/sqlx"
)

// ListWholesalersHandler は卸一覧を返します
func ListWholesalersHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wholesalers, err := database.GetAllWholesalers(db)
		if err != nil {
			log.Printf("Error getting all wholesalers: %v", err)
			http.Error(w, "卸一覧の取得に失敗しました。", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(wholesalers)
	}
}

// CreateWholesalerHandler は新しい卸を作成します
func CreateWholesalerHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			Code string `json:"code"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "リクエストが不正です。", http.StatusBadRequest)
			return
		}
		if input.Code == "" || input.Name == "" {
			http.Error(w, "卸コードと卸名は必須です。", http.StatusBadRequest)
			return
		}

		if err := database.CreateWholesaler(db, input.Code, input.Name); err != nil {
			log.Printf("Error creating wholesaler (Code: %s): %v", input.Code, err)
			http.Error(w, "卸の作成に失敗しました。", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{"message": "作成しました。"})
	}
}

// DeleteWholesalerHandler は卸を削除します
func DeleteWholesalerHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// URLからコードを取得 (例: /api/wholesalers/DELETE/W001)
		code := strings.TrimPrefix(r.URL.Path, "/api/wholesalers/delete/")
		if code == "" {
			http.Error(w, "削除する卸コードが指定されていません。", http.StatusBadRequest)
			return
		}

		if err := database.DeleteWholesaler(db, code); err != nil {
			log.Printf("Error deleting wholesaler (Code: %s): %v", code, err)
			http.Error(w, "卸の削除に失敗しました。", http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"message": "削除しました。"})
	}
}
