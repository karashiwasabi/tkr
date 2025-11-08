// C:\Users\wasab\OneDrive\デスクトップ\TKR\stock\master_migration_handler.go
package stock

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"tkr/database"
	"tkr/mastermanager"
	"tkr/model"
	"tkr/parsers"

	"github.com/jmoiron/sqlx"
	// ★【残す】
	// ★【残す】
)

// ヘッダーの定義 (TKR product_master の全カラム)
var masterCSVHeader = []string{
	"product_code", "yj_code", "gs1_code", "product_name", "kana_name", "kana_name_short",
	"generic_name", "maker_name", "specification", "usage_classification", "package_form",
	"yj_unit_name", "yj_pack_unit_qty", "jan_pack_inner_qty", "jan_unit_code", "jan_pack_unit_qty",
	"origin", "nhi_price", "purchase_price",
	"flag_poison", "flag_deleterious", "flag_narcotic", "flag_psychotropic", "flag_stimulant", "flag_stimulant_raw",
	"is_order_stopped", "supplier_wholesale",
	"group_code", "shelf_number", "category", "user_notes",
}

// ExportAllMastersHandler は product_master の全データをCSVとしてエクスポートします。
func ExportAllMastersHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		masters, err := database.GetAllProductMasters(db)
		if err != nil {
			http.Error(w, "Failed to get all product masters: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var buf bytes.Buffer
		buf.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM

		// csv.Writer を使用して確実なCSVを生成
		writer := csv.NewWriter(&buf)

		// 1. ヘッダーを書き込む
		if err := writer.Write(masterCSVHeader); err != nil {
			http.Error(w, "Failed to write CSV header: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 2. データを書き込む
		for _, m := range masters {
			record := []string{
				m.ProductCode, m.YjCode, m.Gs1Code, m.ProductName, m.KanaName, m.KanaNameShort,
				m.GenericName, m.MakerName, m.Specification, m.UsageClassification, m.PackageForm,
				m.YjUnitName,
				strconv.FormatFloat(m.YjPackUnitQty, 'f', -1, 64),
				strconv.FormatFloat(m.JanPackInnerQty, 'f', -1, 64),
				strconv.Itoa(m.JanUnitCode),
				strconv.FormatFloat(m.JanPackUnitQty, 'f', -1, 64),
				m.Origin,
				strconv.FormatFloat(m.NhiPrice, 'f', -1, 64),
				strconv.FormatFloat(m.PurchasePrice, 'f', -1, 64),
				strconv.Itoa(m.FlagPoison),
				strconv.Itoa(m.FlagDeleterious),
				strconv.Itoa(m.FlagNarcotic),
				strconv.Itoa(m.FlagPsychotropic),
				strconv.Itoa(m.FlagStimulant),
				strconv.Itoa(m.FlagStimulantRaw),
				strconv.Itoa(m.IsOrderStopped),
				m.SupplierWholesale,
				m.GroupCode, m.ShelfNumber, m.Category, m.UserNotes,
			}
			if err := writer.Write(record); err != nil {
				log.Printf("WARN: Failed to write master row to CSV (Code: %s): %v", m.ProductCode, err)
			}
		}
		writer.Flush()

		if err := writer.Error(); err != nil {
			http.Error(w, "Failed to flush CSV writer: "+err.Error(), http.StatusInternalServerError)
			return
		}

		filename := fmt.Sprintf("TKRマスタバックアップ_%s.csv", time.Now().Format("20060102"))
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(filename))
		w.Write(buf.Bytes())
	}
}

// ImportAllMastersHandler はマスタCSVを読み込み、JCSHMSとマージしながらUpsertします。
func ImportAllMastersHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "CSVファイルの読み取りに失敗: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// ▼▼▼【修正】Shift-JISデコーダーを削除し、UTF-8 BOMスキップのみを行う ▼▼▼
		reader := csv.NewReader(parsers.SkipBOM(file))
		// ▲▲▲【修正ここまで】▲▲▲

		reader.LazyQuotes = true
		reader.FieldsPerRecord = -1 // ヘッダー数と一致かチェックするため

		header, err := reader.Read()
		if err == io.EOF {
			http.Error(w, "CSVファイルが空です。", http.StatusBadRequest)
			return
		}
		if err != nil {
			http.Error(w, "CSVヘッダーの読み取りに失敗: "+err.Error(), http.StatusBadRequest)
			return
		}

		// ヘッダーの列インデックスをマップ化
		colIndex := make(map[string]int, len(header))
		for i, h := range header {
			colIndex[strings.TrimSpace(h)] = i
		}

		// 必須列の確認
		if _, ok := colIndex["product_code"]; !ok {
			http.Error(w, "CSVヘッダーに 'product_code' が見つかりません。", http.StatusBadRequest)
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			http.Error(w, "データベーストランザクションの開始に失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		var importedCount int
		var errors []string

		for {
			row, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				log.Printf("WARN: CSV行の読み取りエラー (スキップ): %v", err)
				errors = append(errors, fmt.Sprintf("行スキップ: %v", err))
				continue
			}

			// CSVの値をマップに読み込むヘルパー
			get := func(key string) string {
				if idx, ok := colIndex[key]; ok && idx < len(row) {
					return strings.TrimSpace(row[idx])
				}
				return ""
			}
			getFloat := func(key string) float64 {
				f, _ := strconv.ParseFloat(get(key), 64)
				return f
			}
			getInt := func(key string) int {
				i, _ := strconv.Atoi(get(key))
				return i
			}

			productCode := get("product_code")
			if productCode == "" {
				errors = append(errors, "行スキップ: product_code が空です。")
				continue
			}

			// --- ご要望のマージロジック ---
			var input model.ProductMasterInput

			// 1. JCSHMSを検索 (JAN優先、なければGS1)
			jcshmsInfo, _ := database.GetJcshmsInfoByJan(tx, productCode)
			if jcshmsInfo == nil {
				gs1Code := get("gs1_code")
				if gs1Code != "" {
					jcshmsInfo, _ = database.GetJcshmsInfoByGs1Code(tx, gs1Code)
				}
			}

			if jcshmsInfo != nil {
				// 2a. JCSHMSに存在した場合: JCSHMSをベースにする
				input = mastermanager.JcshmsToProductMasterInput(jcshmsInfo)
			} else {
				// 2b. JCSHMSに存在しない場合: CSVのJCSHMS由来情報をベースにする
				input = model.ProductMasterInput{
					ProductCode:         productCode,
					YjCode:              get("yj_code"),
					Gs1Code:             get("gs1_code"),
					ProductName:         get("product_name"),
					KanaName:            get("kana_name"),
					KanaNameShort:       get("kana_name_short"),
					GenericName:         get("generic_name"),
					MakerName:           get("maker_name"),
					Specification:       get("specification"),
					UsageClassification: get("usage_classification"),
					PackageForm:         get("package_form"),
					YjUnitName:          get("yj_unit_name"),
					YjPackUnitQty:       getFloat("yj_pack_unit_qty"),
					JanPackInnerQty:     getFloat("jan_pack_inner_qty"),
					JanUnitCode:         getInt("jan_unit_code"),
					JanPackUnitQty:      getFloat("jan_pack_unit_qty"),
					Origin:              get("origin"),
					NhiPrice:            getFloat("nhi_price"),
					FlagPoison:          getInt("flag_poison"),
					FlagDeleterious:     getInt("flag_deleterious"),
					FlagNarcotic:        getInt("flag_narcotic"),
					FlagPsychotropic:    getInt("flag_psychotropic"),
					FlagStimulant:       getInt("flag_stimulant"),
					FlagStimulantRaw:    getInt("flag_stimulant_raw"),
				}
				// Originが空ならPROVISIONALに設定
				if input.Origin == "" {
					input.Origin = "PROVISIONAL"
				}
			}

			// 3. ユーザー登録部分をCSVの値で上書き
			input.PurchasePrice = getFloat("purchase_price")
			input.IsOrderStopped = getInt("is_order_stopped")
			input.SupplierWholesale = get("supplier_wholesale")
			input.GroupCode = get("group_code")
			input.ShelfNumber = get("shelf_number")
			input.Category = get("category")
			input.UserNotes = get("user_notes")

			// 4. YJコードの自動採番
			if input.YjCode == "" {
				newYj, seqErr := database.NextSequenceInTx(tx, "MA2Y", "MA2Y", 8)
				if seqErr != nil {
					errors = append(errors, fmt.Sprintf("行 %s: YJコード採番失敗: %v", productCode, seqErr))
					continue
				}
				input.YjCode = newYj
			}

			// 5. DBにUpsert
			if _, err := mastermanager.UpsertProductMasterSqlx(tx, input); err != nil {
				errors = append(errors, fmt.Sprintf("行 %s: DB登録失敗: %v", productCode, err))
				continue
			}
			importedCount++
		}

		// 6. シーケンスの更新 (インポート後に実行)
		if err := database.InitializeSequenceFromMaxYjCode(tx); err != nil {
			log.Printf("WARN: Failed to re-initialize YJ (MA2Y) sequence after import: %v", err)
		}
		if err := database.InitializeSequenceFromMaxProductCode(tx); err != nil {
			log.Printf("WARN: Failed to re-initialize JAN (MA2J) sequence after import: %v", err)
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "データベースのコミットに失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}

		message := fmt.Sprintf("%d件の製品マスターをインポート（上書き/新規登録）しました。", importedCount)
		if len(errors) > 0 {
			message += fmt.Sprintf("\n%d件のエラーが発生しました: %s", len(errors), strings.Join(errors[:3], ", ")) // エラーが多すぎないよう制限
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": message})
	}
}
