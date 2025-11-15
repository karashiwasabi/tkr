// C:\Users\wasab\OneDrive\デスクトップ\TKR\masteredit\handler.go
package masteredit

import (
	"database/sql" // sql をインポート
	"encoding/json"
	"errors"
	"fmt" // 変更
	"log"
	"net/http"
	"strings"
	"tkr/database"
	"tkr/mastermanager"
	"tkr/model"
	"tkr/units" // 変更

	"github.com/jmoiron/sqlx"
)

func ListMastersHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		queryParams := r.URL.Query()
		usageClass := queryParams.Get("usage_class")
		kanaName := queryParams.Get("kana_name")
		genericName := queryParams.Get("generic_name")
		shelfNumber := queryParams.Get("shelf_number")

		if usageClass == "" {
			log.Println("ListMastersHandler: usage_class is required, returning empty.")
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"tableHTML": renderMasterListHTML(nil, "内外注区分を選択してください。"),
				"masters":   []model.ProductMaster{},
			})
			return
		}

		masters, err := database.GetFilteredProductMasters(db,
			usageClass, kanaName, genericName, shelfNumber)

		if err != nil {
			log.Printf("Error fetching filtered product masters: %v", err)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"tableHTML": renderMasterListHTML(nil, "マスターの検索中にエラーが発生しました。"),
				"masters":   []model.ProductMaster{},
			})
			return
		}

		tableHTML := renderMasterListHTML(masters, "")

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]interface{}{
			"tableHTML": tableHTML,
			"masters":   masters,
		}); err != nil {
			log.Printf("Error encoding product masters to JSON: %v", err)
			http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		}
	}
}

func renderMasterListHTML(masters []model.ProductMaster, statusMessage string) string {
	var sb strings.Builder

	sb.WriteString(`
    <thead>
        <tr>
            <th class="col-action"></th>
            <th class="col-yj">YJコード</th>
            <th class="col-gs1">GS1コード</th>
            <th class="col-jan">JANコード</th>
          
   



            <th class="col-product">製品名</th>
            <th class="col-kana">カナ名</th>
            <th class="col-maker">メーカー</th>
            <th class="col-generic">一般名</th>
            <th class="col-shelf">棚番</th>
        </tr>
    </thead>`)

	sb.WriteString(`<tbody>`)
	if statusMessage != "" {
		sb.WriteString(fmt.Sprintf(`<tr><td colspan="9">%s</td></tr>`, statusMessage))
	} else if len(masters) == 0 {
		sb.WriteString(`<tr><td colspan="9">データがありません。</td></tr>`)
	} else {
		for _, master := range masters {
			sb.WriteString(fmt.Sprintf(`<tr data-product-code="%s">`,
				master.ProductCode))
			sb.WriteString(fmt.Sprintf(`<td 

class="center col-action"><button class="edit-master-btn 
 btn" data-code="%s">編集</button></td>`, master.ProductCode))
			sb.WriteString(fmt.Sprintf(`<td class="col-yj">%s</td>`, master.YjCode))
			sb.WriteString(fmt.Sprintf(`<td class="col-gs1">%s</td>`,
				master.Gs1Code))
			sb.WriteString(fmt.Sprintf(`<td class="col-jan">%s</td>`, master.ProductCode))
			sb.WriteString(fmt.Sprintf(`<td class="left col-product">%s</td>`, master.ProductName))
			sb.WriteString(fmt.Sprintf(`<td class="left col-kana">%s</td>`, master.KanaName))
			sb.WriteString(fmt.Sprintf(`<td class="left col-maker">%s</td>`, master.MakerName))
			sb.WriteString(fmt.Sprintf(`<td class="left col-generic">%s</td>`, master.GenericName))
			sb.WriteString(fmt.Sprintf(`<td class="col-shelf">%s</td>`, master.ShelfNumber))
			sb.WriteString(`</tr>`)
		}
	}
	sb.WriteString(`</tbody>`)

	return sb.String()
}

// ▼▼▼【ここを修正】UpdateMasterHandler に「新規作成」ロジックを追加 ▼▼▼
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

		// productCode が空でもエラーにしなくなった

		tx, err := db.Beginx()
		if err != nil {
			log.Printf("UpdateMasterHandler: Failed to start transaction: %v", err)
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		alertMessage := ""
		isNew := false

		// 1. 既存マスタ情報を取得
		oldMaster, err := database.GetProductMasterByCode(tx, input.ProductCode)

		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				// -------------------------------------------------
				// B. 新規作成の場合
				// -------------------------------------------------
				isNew = true
				log.Printf("UpdateMasterHandler: No existing master found for ProductCode [%s]. Treating as NEW.", input.ProductCode)

				// B-1. ProductCode (JAN) の採番
				if input.ProductCode == "" {
					newCode, seqErr := database.NextSequenceInTx(tx, "MA2J", "MA2J", 9)
					if seqErr != nil {
						http.Error(w, "JANコード(MA2J)の自動採番に失敗しました: "+seqErr.Error(), http.StatusInternalServerError)
						return
					}
					input.ProductCode = newCode
					log.Printf("UpdateMasterHandler: Assigned new ProductCode (MA2J): %s", newCode)
				}

				// B-2. YjCode の採番
				if input.YjCode == "" {
					newYj, seqErr := database.NextSequenceInTx(tx, "MA2Y", "MA2Y", 8)
					if seqErr != nil {
						http.Error(w, "YJコード(MA2Y)の自動採番に失敗しました: "+seqErr.Error(), http.StatusInternalServerError)
						return
					}
					input.YjCode = newYj
					log.Printf("UpdateMasterHandler: Assigned new YjCode (MA2Y): %s", newYj)
				}

				// B-3. Origin の設定
				if input.Origin != "JCSHMS" {
					input.Origin = "PROVISIONAL"
				}

				// B-4. 剤型のフォールバック
				if input.UsageClassification == "" {
					input.UsageClassification = "他"
				}

			} else {
				// -------------------------------------------------
				// C. 既存マスタ取得時のDBエラー
				// -------------------------------------------------
				log.Printf("UpdateMasterHandler: Failed to get old master (JAN: %s): %v", input.ProductCode, err)
				http.Error(w, "Failed to retrieve master before edit", http.StatusInternalServerError)
				return
			}
		}

		// -------------------------------------------------
		// A. 更新の場合
		// -------------------------------------------------
		if !isNew {
			log.Printf("UpdateMasterHandler: Found existing master for ProductCode [%s]. Treating as UPDATE.", input.ProductCode)
			// 2. 編集前後の PackageKey を計算
			oldYjUnitName := units.ResolveName(oldMaster.YjUnitName)
			oldKey := fmt.Sprintf("%s|%s|%g|%s", oldMaster.YjCode, oldMaster.PackageForm, oldMaster.JanPackInnerQty, oldYjUnitName)

			newYjUnitName := units.ResolveName(input.YjUnitName)
			newKey := fmt.Sprintf("%s|%s|%g|%s", input.YjCode, input.PackageForm, input.JanPackInnerQty, newYjUnitName)

			if oldKey != newKey {
				alertMessage = "PackageKeyが変更されました。棚卸（在庫振替）を実施してください。"

				input.UserNotes = fmt.Sprintf("(自動記録: 旧Key [%s] から在庫振替要) %s", oldKey, input.UserNotes)

				log.Printf("UpdateMasterHandler: PackageKey changed for %s.OldKey [%s] NewKey [%s]. Alert set.",
					input.ProductCode, oldKey, newKey)
			}
		}
		// ▲▲▲【修正ここまで】▲▲▲

		// 6. マスタをDBに保存
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

		// 7. レスポンスにアラートメッセージを含める
		response := map[string]string{
			"message": "Saved successfully.",
		}
		if alertMessage != "" {
			response["alert"] = alertMessage
		}
		json.NewEncoder(w).Encode(response)
	}
}

// ▲▲▲【修正ここまで】▲▲▲

// ▼▼▼【ここから追加】(WASABI: masteredit/handler.go より移植) ▼▼▼

type SetOrderStoppedRequest struct {
	ProductCode string `json:"productCode"`
	Status      int    `json:"status"` // 0: 発注可, 1: 発注不可
}

func SetOrderStoppedHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SetOrderStoppedRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		if req.ProductCode == "" {
			http.Error(w, "productCode is required", http.StatusBadRequest)
			return
		}

		tx, err := conn.Beginx() // TKR用に .Beginx() を使用
		if err != nil {
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// TKRの GetProductMasterByCode を使用
		master, err := database.GetProductMasterByCode(tx, req.ProductCode)
		if err != nil {
			// ▼▼▼【修正】エラーハンドリング (sql.ErrNoRows を考慮) ▼▼▼
			if err == sql.ErrNoRows {
				http.Error(w, "Product not found", http.StatusNotFound)
			} else {
				http.Error(w, "Failed to get product master: "+err.Error(), http.StatusInternalServerError)
			}
			// ▲▲▲【修正ここまで】▲▲▲
			return
		}
		if master == nil {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}

		// ステータスを更新
		master.IsOrderStopped = req.Status

		// TKRの mastermanager.MasterToInput を使用
		input := mastermanager.MasterToInput(master)

		// TKRの UpsertProductMasterSqlx を使用
		if _, err := mastermanager.UpsertProductMasterSqlx(tx, input); err != nil {
			http.Error(w, "Failed to update product master", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "更新しました。"})
	}
}

// ▲▲▲【追加ここまで】▲▲▲
