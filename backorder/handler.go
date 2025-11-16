// C:\Users\wasab\OneDrive\デスクトップ\TKR\backorder\handler.go
package backorder

import (
	"encoding/json"
	"fmt" // TKRはlogの代わりにfmtを使うことが多い
	"net/http"
	"tkr/database" // TKRのdatabase
	"tkr/mappers"  // TKRのmappers
	"tkr/model"    // TKRのmodel

	"github.com/jmoiron/sqlx"
)

// BackorderView は発注残データを画面表示用に整形します。
// (WASABI: backorder/handler.go より)
type BackorderView struct {
	model.Backorder
	FormattedPackageSpec string `json:"formattedPackageSpec"`
}

/**
 * @brief 全ての発注残リストを取得し、画面表示用に整形して返すためのHTTPハンドラです。
 * (WASABI: backorder/handler.go を TKR 用に修正)
 */
func GetBackordersHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// TKRのDB関数を呼ぶ
		backorders, err := database.GetAllBackordersList(conn)
		if err != nil {
			http.Error(w, "Failed to get backorder list", http.StatusInternalServerError)
			return
		}

		backorderViews := make([]BackorderView, 0, len(backorders))
		for _, bo := range backorders {
			// TKRの JcshmsInfo 構造体（units.FormatPackageSpec が要求する型）に合わせる
			// TKRの mappers.ToProductMasterView が TKRの JcshmsInfo 形式のラッパーを使っている
			// TKRの mappers/view.go を参考に、model.ProductMaster を一時的に作成
			tempMaster := model.ProductMaster{
				PackageForm:     bo.PackageForm,
				YjUnitName:      bo.YjUnitName,
				YjPackUnitQty:   bo.YjPackUnitQty,
				JanPackInnerQty: bo.JanPackInnerQty,
				JanPackUnitQty:  bo.JanPackUnitQty,
				JanUnitCode:     bo.JanUnitCode,
			}

			// TKRの ToProductMasterView を使って包装仕様を取得
			// (この関数は内部で TKRの units.FormatPackageSpec を呼んでいる)
			view := mappers.ToProductMasterView(&tempMaster)

			backorderViews = append(backorderViews, BackorderView{
				Backorder:            bo,
				FormattedPackageSpec: view.FormattedPackageSpec, // TKRのマッパーが生成した包装仕様
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(backorderViews)
	}
}

/**
 * @brief 単一の発注残レコードを削除するためのHTTPハンドラです。
 * (WASABI: backorder/handler.go を TKR 用に修正)
 */
func DeleteBackorderHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			ID int `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		tx, err := conn.Beginx()
		if err != nil {
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// TKRのDB関数を呼ぶ
		if err := database.DeleteBackorderInTx(tx, payload.ID); err != nil {
			http.Error(w, "Failed to delete backorder: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "発注残を削除しました。"})
	}
}

/**
 * @brief 複数の発注残レコード（ID指定）を一括で削除するためのHTTPハンドラです。
 * (旧 BulkDeleteBackordersHandler)
 */
func BulkDeleteBackordersByIDHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload []struct {
			ID int `json:"id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if len(payload) == 0 {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]string{"message": "削除する項目がありません。"})
			return
		}

		tx, err := conn.Beginx()
		if err != nil {
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		for _, bo := range payload {
			// TKRのDB関数を呼ぶ
			if err := database.DeleteBackorderInTx(tx, bo.ID); err != nil {
				// 1件でも失敗したらエラーを返し、トランザクション全体をロールバック
				http.Error(w, fmt.Sprintf("Failed to delete backorder (ID: %d): %s", bo.ID, err.Error()), http.StatusInternalServerError)
				return
			}
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "選択された発注残を削除しました。"})
	}
}

/**
 * @brief 指定された発注日時(order_date)の発注残レコードをすべて削除するためのHTTPハンドラです。
 */
func BulkDeleteBackordersByDateHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			OrderDate string `json:"orderDate"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if payload.OrderDate == "" {
			http.Error(w, "orderDate is required", http.StatusBadRequest)
			return
		}

		tx, err := conn.Beginx()
		if err != nil {
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// TKRのDB関数を呼ぶ
		rowsAffected, err := database.DeleteBackordersByOrderDateInTx(tx, payload.OrderDate)
		if err != nil {
			http.Error(w, "Failed to delete backorders: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": fmt.Sprintf("発注 (日時: %s) に関連する %d 件の品目を削除しました。", payload.OrderDate, rowsAffected),
		})
	}
}
