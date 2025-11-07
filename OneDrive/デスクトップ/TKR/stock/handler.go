// C:\Users\wasab\OneDrive\デスクトップ\TKR\stock\handler.go
package stock

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
	"tkr/database"
	"tkr/mappers"
	"tkr/mastermanager"
	"tkr/model"
	"tkr/parsers"
	"tkr/units"

	"github.com/jmoiron/sqlx"
)

func quoteAll(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func ExportCurrentStockHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// 1. 在庫起点テーブル (package_stock) から全在庫データを取得
		stocks, err := database.GetAllPackageStock(db)
		if err != nil {
			http.Error(w, "Failed to get all package stock: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 2. YJコードと代表品名のマップを取得
		nameMap, err := database.GetRepresentativeProductNameMap(db)
		if err != nil {
			http.Error(w, "Failed to get representative product names: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var buf bytes.Buffer
		buf.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM

		// 3. ヘッダーを PackageKey, ProductName, JAN数量 に変更
		header := []string{
			"PackageKey",
			"ProductName",
			"JAN数量",
		}
		buf.WriteString(strings.Join(header, ",") + "\r\n")

		// 4. データをCSV行に変換
		for _, pkg := range stocks {
			// PackageKey を解析して JanPackInnerQty を取得
			parsedKey, err := database.ParsePackageKey(pkg.PackageKey)
			if err != nil {
				log.Printf("WARN: Skipping invalid PackageKey in export: %s", pkg.PackageKey)
				continue
			}

			// YJ数量 を JAN数量 に換算
			var janQty float64
			if parsedKey.JanPackInnerQty > 0 {
				janQty = pkg.StockQuantityYj / parsedKey.JanPackInnerQty
			} else {
				janQty = 0 // 内入数が0ならJAN数量も0
			}

			// 代表品名を取得
			productName := nameMap[pkg.YjCode]

			record := []string{
				quoteAll(pkg.PackageKey),
				quoteAll(productName),
				quoteAll(fmt.Sprintf("%.2f", janQty)),
			}
			buf.WriteString(strings.Join(record, ",") + "\r\n")
		}

		today := time.Now().Format("20060102")
		filename := fmt.Sprintf("TKR在庫データ_%s.csv", today) // ファイル名を変更

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(filename))

		w.Write(buf.Bytes())
	}
}

func ImportExternalStockCSVHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "CSVファイルの読み取りに失敗: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// ▼▼▼【ロジック変更】先にCSVをパースする ▼▼▼
		records, err := parsers.ParseExternalStockCSV(file)
		if err != nil {
			http.Error(w, "CSVファイルの解析に失敗: "+err.Error(), http.StatusBadRequest)
			return
		}
		if len(records) == 0 {
			http.Error(w, "読み込むデータがありません（JAN数量が0以下のデータは無視されます）。", http.StatusBadRequest)
			return
		}

		date := r.FormValue("date") // YYYYMMDD形式
		if date == "" || len(date) != 8 {
			http.Error(w, "日付(YYYYMMDD)が不正です。", http.StatusBadRequest)
			return
		}

		var dateYYMMDD string
		if len(date) >= 8 {
			dateYYMMDD = date[2:8] // "20251107" -> "251107"
		} else {
			dateYYMMDD = date
		}
		receiptNumber := "MIG_EXT_" + dateYYMMDD

		tx, err := db.Beginx()
		if err != nil {
			http.Error(w, "データベーストランザクションの開始に失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// ▼▼▼【ロジック変更】洗い替えロジック ▼▼▼

		// 1. CSVのデータをループし、先に（仮）マスタを作成する
		//    (この時点ではDBに永続化しない)
		//    同時に、後の処理のためにCSVデータを集計する
		packageStockTotalsYj := make(map[string]float64)
		var transactionsToInsert []model.TransactionRecord
		csvMasters := make(map[string]*model.ProductMaster) // CSVに出てきたマスタを格納

		for i, rec := range records {
			master, err := mastermanager.FindOrCreateMaster(tx, rec.JanCode, "")
			if err != nil {
				http.Error(w, fmt.Sprintf("マスターの特定/作成に失敗 (JAN: %s): %v", rec.JanCode, err), http.StatusInternalServerError)
				return
			}
			csvMasters[master.ProductCode] = master // マスタをキャッシュ

			janQty := rec.JanQuantity
			yjQty := janQty * master.JanPackInnerQty

			packageKey := fmt.Sprintf("%s|%s|%g|%s", master.YjCode, master.PackageForm, master.JanPackInnerQty, units.ResolveName(master.YjUnitName))
			packageStockTotalsYj[packageKey] += yjQty

			tr := model.TransactionRecord{
				TransactionDate: date,
				Flag:            0,
				ReceiptNumber:   receiptNumber,
				LineNumber:      fmt.Sprintf("%d", i+1),
				JanCode:         master.ProductCode,
				JanQuantity:     janQty,
				YjQuantity:      yjQty,
				ExpiryDate:      rec.ExpiryDate,
				LotNumber:       rec.LotNumber,
				UnitPrice:       master.NhiPrice,
				Subtotal:        yjQty * master.NhiPrice,
			}
			mappers.MapMasterToTransaction(&tr, master)
			transactionsToInsert = append(transactionsToInsert, tr)
		}

		// 2. 全ての棚卸明細(flag=0)を削除 (既存の棚卸履歴をクリア)
		if _, err := tx.Exec("DELETE FROM transaction_records WHERE flag = 0"); err != nil {
			http.Error(w, "既存の棚卸明細(flag=0)の削除に失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 3. product_master に存在する全品目（CSVから仮作成されたものも含む）の
		//    package_key を取得し、package_stock の在庫を 0 でリセットする
		var allMasterKeys []struct {
			PackageKey string `db:"package_key"`
			YjCode     string `db:"yj_code"`
		}
		const allMasterKeysQuery = `
			SELECT
				P.yj_code || '|' || 
				COALESCE(P.package_form, '不明') || '|' ||
				PRINTF('%g', COALESCE(P.jan_pack_inner_qty, 0)) || '|' || 
				COALESCE(U.name, P.yj_unit_name, '不明') AS package_key,
				P.yj_code
			FROM product_master AS P
			LEFT JOIN units AS U ON P.yj_unit_name = U.code
			WHERE P.yj_code != "" 
			GROUP BY package_key, P.yj_code
		`
		if err := tx.Select(&allMasterKeys, allMasterKeysQuery); err != nil {
			http.Error(w, "全マスタの包装キー取得に失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("[ImportExternalStock] Resetting %d master package keys to 0...", len(allMasterKeys))
		// 全マスタ品目の在庫を 0 で Upsert (洗い替えの準備)
		for _, m := range allMasterKeys {
			if err := database.UpsertPackageStockInTx(tx, m.PackageKey, m.YjCode, 0, date); err != nil {
				http.Error(w, "全在庫の0リセットに失敗: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// 4. 棚卸明細(flag=0)を挿入 (CSVに存在する分のみ)
		if err := database.PersistTransactionRecordsInTx(tx, transactionsToInsert); err != nil {
			http.Error(w, "棚卸明細の挿入に失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 5. package_stock を CSV の在庫データで上書き更新
		log.Printf("[ImportExternalStock] Updating %d package keys from CSV...", len(packageStockTotalsYj))
		for key, totalYjQty := range packageStockTotalsYj {
			yjCode := strings.Split(key, "|")[0]
			// 既に 0 で Upsert されているレコードを、CSVの在庫数で再度 Upsert
			if err := database.UpsertPackageStockInTx(tx, key, yjCode, totalYjQty, date); err != nil {
				http.Error(w, "在庫起点の上書き更新に失敗: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		// ▲▲▲【修正ここまで】▲▲▲

		if err := tx.Commit(); err != nil {
			http.Error(w, "データベースのコミットに失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": fmt.Sprintf("在庫の洗い替えが完了しました。%d件の明細が登録されました。", len(transactionsToInsert)),
		})
	}
}

func ImportTKRStockCSVHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "CSVファイルの読み取りに失敗: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		records, err := parsers.ParseTKRStockCSV(file)
		if err != nil {
			http.Error(w, "CSVファイルの解析に失敗: "+err.Error(), http.StatusBadRequest)
			return
		}
		if len(records) == 0 {
			http.Error(w, "読み込むデータがありません。", http.StatusBadRequest)
			return
		}

		date := time.Now().Format("20060102")
		dateYYMMDD := time.Now().Format("060102")
		receiptNumber := "MIG_TKR_" + dateYYMMDD

		tx, err := db.Beginx()
		if err != nil {
			http.Error(w, "データベーストランザクションの開始に失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		for i, rec := range records {
			parsedKey, err := database.ParsePackageKey(rec.PackageKey)
			if err != nil {
				log.Printf("WARN: Skipping invalid PackageKey: %s", rec.PackageKey)
				continue
			}

			janQty := rec.JanQuantity
			yjQty := janQty * parsedKey.JanPackInnerQty

			if err := database.UpsertPackageStockInTx(tx, rec.PackageKey, parsedKey.YjCode, yjQty, date); err != nil {
				http.Error(w, "在庫起点の差分更新に失敗: "+err.Error(), http.StatusInternalServerError)
				return
			}

			var master *model.ProductMaster
			err = tx.Get(&master, "SELECT * FROM product_master WHERE yj_code = ? AND package_form = ? ORDER BY origin = 'JCSHMS' DESC, product_code ASC LIMIT 1", parsedKey.YjCode, parsedKey.PackageForm)
			if err != nil {
				if err == sql.ErrNoRows {
					log.Printf("WARN: No master found for PackageKey %s, skipping transaction record.", rec.PackageKey)
					continue
				}
				http.Error(w, "代表マスターの検索に失敗: "+err.Error(), http.StatusInternalServerError)
				return
			}

			_, err = tx.Exec("DELETE FROM transaction_records WHERE flag = 0 AND transaction_date = ? AND yj_code = ?", date, parsedKey.YjCode)
			if err != nil {
				http.Error(w, "既存棚卸明細の削除に失敗: "+err.Error(), http.StatusInternalServerError)
				return
			}

			tr := model.TransactionRecord{
				TransactionDate: date,
				Flag:            0,
				ReceiptNumber:   fmt.Sprintf("%s-%d", receiptNumber, i+1),
				LineNumber:      "1",
				JanCode:         master.ProductCode,
				JanQuantity:     janQty,
				YjQuantity:      yjQty,
				UnitPrice:       master.NhiPrice,
				Subtotal:        yjQty * master.NhiPrice,
			}
			mappers.MapMasterToTransaction(&tr, master)

			if err := database.InsertTransactionRecord(tx, tr); err != nil {
				http.Error(w, "棚卸明細の挿入に失敗: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "データベースのコミットに失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": fmt.Sprintf("在庫の差分更新が完了しました。%d件のPackageKeyが処理されました。", len(records)),
		})
	}
}
