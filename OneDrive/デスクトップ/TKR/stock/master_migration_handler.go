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

// ▼▼▼【ここを修正】masterCSVHeader を（再度）定義 ▼▼▼
var masterCSVHeader = []string{
	"product_code", "yj_code", "gs1_code", "product_name", "kana_name", "kana_name_short",
	"generic_name", "maker_name", "specification", "usage_classification", "package_form",
	"yj_unit_name", "yj_pack_unit_qty", "jan_pack_inner_qty", "jan_unit_code", "jan_pack_unit_qty",
	"origin", "nhi_price", "purchase_price",
	"flag_poison", "flag_deleterious", "flag_narcotic", "flag_psychotropic", "flag_stimulant", "flag_stimulant_raw",
	"is_order_stopped", "supplier_wholesale",
	"group_code", "shelf_number", "category", "user_notes",
}

// ▲▲▲【修正ここまで】▲▲▲

// ▼▼▼【ここから修正】ExportAllMastersHandler を「TKRマスタバックアップ」を出力する元のロジックに戻す ▼▼▼
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

		// 1. ヘッダーを書き込む (masterCSVHeader を使用)
		if err := writer.Write(masterCSVHeader); err != nil {
			http.Error(w, "Failed to write CSV header: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 2. データを書き込む (全カラム)
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

		// 3. ファイル名を「TKRマスタバックアップ...」に戻す
		filename := fmt.Sprintf("TKRマスタバックアップ_%s.csv", time.Now().Format("20060102"))
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(filename))
		w.Write(buf.Bytes())
	}
}

// ▲▲▲【修正ここまで】▲▲▲

// ImportAllMastersHandler はマスタCSVを読み込み、JCSHMSとマージしながらUpsertします。
func ImportAllMastersHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "CSVファイルの読み取りに失敗: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		reader := csv.NewReader(parsers.SkipBOM(file))

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

		colIndex := make(map[string]int, len(header))
		for i, h := range header {
			colIndex[strings.TrimSpace(h)] = i
		}

		if _, ok := colIndex["product_code"]; !ok {
			http.Error(w, "CSVヘッダーに 'product_code' が見つかりません。", http.StatusBadRequest)
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			http.Error(w, "データベーストランザクションの開始に失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback() // コミット成功時以外は自動でロールバックされる

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

			var input model.ProductMasterInput

			// 1. キーの特定: product_code (JAN) のみで JCSHMS を検索
			jcshmsInfo, _ := database.GetJcshmsInfoByJan(tx, productCode)
			// (gs1_code での再検索は行わない)

			// 2. ベースデータの決定
			if jcshmsInfo != nil {
				// 2A.JCSHMSに情報が見つかった場合
				input = mastermanager.JcshmsToProductMasterInput(jcshmsInfo)
				input.Origin = "JCSHMS" // JcshmsToProductMasterInput が設定するが、明示的に "JCSHMS" を維持

			} else {
				// 2B.JCSHMSに情報が見つからなかった場合
				csvProductName := get("product_name")
				dbProductName := csvProductName
				if !strings.HasPrefix(csvProductName, "◆") {
					dbProductName = "◆" + csvProductName
				}

				// ▼▼▼【ここに追加】CSVの剤型区分が空白なら「他」を設定 ▼▼▼
				csvUsageClass := get("usage_classification")
				if csvUsageClass == "" {
					csvUsageClass = "他"
				}
				// ▲▲▲【追加ここまで】▲▲▲

				input = model.ProductMasterInput{
					ProductCode: productCode,
					YjCode:      "", // 3B で設定するため、ここでは空
					Gs1Code:     get("gs1_code"),
					ProductName: dbProductName,
					KanaName:    get("kana_name"),
					// KanaNameShort: 4B で設定
					GenericName:         get("generic_name"),
					MakerName:           get("maker_name"),
					Specification:       get("specification"),
					UsageClassification: csvUsageClass, // ▼▼▼【修正】変更後の変数を設定
					PackageForm:         get("package_form"),
					YjUnitName:          get("yj_unit_name"),
					YjPackUnitQty:       getFloat("yj_pack_unit_qty"),
					JanPackInnerQty:     getFloat("jan_pack_inner_qty"),
					JanUnitCode:         getInt("jan_unit_code"),
					JanPackUnitQty:      getFloat("jan_pack_unit_qty"),
					Origin:              "PROVISIONAL",
					NhiPrice:            getFloat("nhi_price"),
					FlagPoison:          getInt("flag_poison"),
					FlagDeleterious:     getInt("flag_deleterious"),
					FlagNarcotic:        getInt("flag_narcotic"),
					FlagPsychotropic:    getInt("flag_psychotropic"),
					FlagStimulant:       getInt("flag_stimulant"),
					FlagStimulantRaw:    getInt("flag_stimulant_raw"),
				}
			}

			// 3. YJコードの処理 (CSVのYJは無視)
			if jcshmsInfo != nil {
				// 3A.JCSHMS由来の場合
				if jcshmsInfo.YjCode == "" {
					// JCSHMS由来だがYJなし (JC009が空)
					newYj, seqErr := database.NextSequenceInTx(tx, "MA2Y", "MA2Y", 8)
					if seqErr != nil {
						log.Printf("ERROR: YJコード自動採番失敗 (JCSHMS/YJ無/ProductCode: %s): %v. Rolling back...", productCode, seqErr)
						http.Error(w, "YJコードの自動採番に失敗しました: "+seqErr.Error(), http.StatusInternalServerError)
						return
					}
					input.YjCode = newYj
					// input.Origin は "JCSHMS" のまま
					log.Printf("INFO: Master import (JCSHMS item %s) had no YJ. Assigned MA2Y: %s. Origin remains JCSHMS.", productCode, newYj)
				}
				// else (jcshmsInfo.YjCode != ""):
				// JcshmsToProductMasterInput で input.YjCode に設定済みなので何もしない
			} else {
				// 3B.JCSHMSになかった場合(PROVISIONAL)
				// ▼▼▼【ここから修正】YJコードの採番ロジックを変更 ▼▼▼
				csvYjCode := get("yj_code")

				if csvYjCode == "" ||
					strings.HasPrefix(csvYjCode, "MA2Y") {
					// CSVのYJコードが空欄、または MA2Y で始まる場合のみ、自動採番
					newYj, seqErr := database.NextSequenceInTx(tx, "MA2Y", "MA2Y", 8)
					if seqErr != nil {
						log.Printf("ERROR: YJコード自動採番失敗 (PROVISIONAL/ProductCode: %s): %v. Rolling back...", productCode, seqErr)
						http.Error(w, "YJコードの自動採番に失敗しました: "+seqErr.Error(), http.StatusInternalServerError)
						return
					}
					input.YjCode = newYj
					if csvYjCode != "" {
						log.Printf("INFO: Master import (PROVISIONAL item %s) had MA2Y YJCode ('%s'). Re-assigned new YJ: %s.", productCode, csvYjCode, newYj)
					}
				} else {
					// CSVに MA2Y 以外（例: 610...）が記載されている場合は、その値を採用する
					input.YjCode = csvYjCode
					log.Printf("INFO: Master import (PROVISIONAL item %s) adopted YJCode from CSV: %s.", productCode, csvYjCode)
				}
				// ▲▲▲【修正ここまで】▲▲▲
			}

			// 4. カナ名短縮 (JC019) の処理
			csvKanaNameShort := get("kana_name_short")

			if jcshmsInfo != nil {
				// 4A.JCSHMS由来の場合
				// (input.KanaNameShort には JcshmsToProductMasterInput によって JC019 が入っている)
				if input.KanaNameShort == "" {
					// JCSHMS(JC019)が空だった場合のみ、CSVの値をフォールバックとして使用
					input.KanaNameShort = csvKanaNameShort
					// (CSVも空なら input.KanaNameShort は "" のまま)
				}
			} else {
				// 4B.JCSHMSになかった場合(PROVISIONAL)
				if csvKanaNameShort != "" {
					input.KanaNameShort = csvKanaNameShort
				} else {
					// CSVも空なら、CSVの製品名(◆なし)をフォールバック
					input.KanaNameShort = get("product_name")
				}
			}

			// 5. ユーザー設定項目の上書き (JCSHMS由来かどうかにかかわらず、CSVの値を優先)
			input.PurchasePrice = getFloat("purchase_price")
			input.IsOrderStopped = getInt("is_order_stopped")
			input.SupplierWholesale = get("supplier_wholesale")
			input.GroupCode = get("group_code")
			input.ShelfNumber = get("shelf_number")
			input.Category = get("category")
			input.UserNotes = get("user_notes")

			// 6. データベースへ保存
			if _, err := mastermanager.UpsertProductMasterSqlx(tx, input); err != nil {
				errors = append(errors, fmt.Sprintf("行 %s: DB登録失敗: %v", productCode, err))
				continue
			}
			importedCount++
		}

		// ▼▼▼【ここを修正】カウンターのリセット処理を「最後」に戻す ▼▼▼
		// ▼▼▼【修正】 seqErr -> err に変更 ▼▼▼
		if err := database.InitializeSequenceFromMaxYjCode(tx); err != nil {
			log.Printf("ERROR: Failed to re-initialize YJ (MA2Y) sequence: %v. Rolling back...", err)
			http.Error(w, "自動採番カウンター(MA2Y)のリセットに失敗しました: "+err.Error(), http.StatusInternalServerError)
			return
		}
		if err := database.InitializeSequenceFromMaxProductCode(tx); err != nil {
			log.Printf("ERROR: Failed to re-initialize JAN (MA2J) sequence: %v. Rolling back...", err)
			http.Error(w, "自動採番カウンター(MA2J)のリセットに失敗しました: "+err.Error(), http.StatusInternalServerError)
			return
		}
		// ▲▲▲【修正ここまで】▲▲▲
		// ▲▲▲【修正ここまで】▲▲▲

		if err := tx.Commit(); err != nil {
			log.Printf("ERROR: Failed to commit master import transaction: %v. Transaction was rolled back.", err)
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
