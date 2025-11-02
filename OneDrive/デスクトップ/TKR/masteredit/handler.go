// C:\Users\wasab\OneDrive\デスクトップ\TKR\masteredit\handler.go
package masteredit

import (
	"encoding/json"
	"fmt" // ★ fmt をインポート
	"log"
	"net/http"
	"strings" // ★ strings をインポート
	"tkr/database"
	"tkr/mastermanager"
	"tkr/model"

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
			sb.WriteString(fmt.Sprintf(`<tr data-product-code="%s">`, master.ProductCode))
			sb.WriteString(fmt.Sprintf(`<td class="center col-action"><button class="edit-master-btn btn" data-code="%s">編集</button></td>`, master.ProductCode))
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
