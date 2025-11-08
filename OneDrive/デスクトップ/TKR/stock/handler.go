// C:\Users\wasab\OneDrive\デスクトップ\TKR\stock\handler.go
package stock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	// "strconv" // (未使用)
	"strings"
	"time"
	"tkr/database"
	"tkr/mappers"

	// "tkr/mastermanager" // (未使用)
	"tkr/model"
	"tkr/parsers"

	// "tkr/units" // (未使用)

	"github.com/jmoiron/sqlx"
)

func quoteAll(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// (ExportCurrentStockHandler 関数は変更なし)
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

// ▼▼▼【ここから修正】ImportTKRStockCSVHandler を「完全洗い替え」ロジックに変更 ▼▼▼
func ImportTKRStockCSVHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		date := r.FormValue("date") // YYYYMMDD形式
		if date == "" || len(date) != 8 {
			http.Error(w, "日付(YYYYMMDD)が不正です。", http.StatusBadRequest)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "CSVファイルの読み取りに失敗: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// 1. TKR独自CSVパーサーを呼ぶ (CSVに記載のあるデータ)
		csvRecords, err := parsers.ParseTKRStockCSV(file)
		if err != nil {
			http.Error(w, "CSVファイルの解析に失敗: "+err.Error(), http.StatusBadRequest)
			return
		}

		// CSVの在庫データをマップ化 (Key: PackageKey, Value: JAN数量)
		csvStockMap := make(map[string]float64)
		for _, rec := range csvRecords {
			if rec.PackageKey != "" {
				csvStockMap[rec.PackageKey] = rec.JanQuantity
			}
		}

		// 2. 棚卸日（フォーム値）と伝票番号を生成
		var dateYYMMDD string
		if len(date) >= 8 {
			dateYYMMDD = date[2:8] // "20251107" -> "251107"
		} else {
			dateYYMMDD = date
		}
		receiptNumber := "MIG_TKR_" + dateYYMMDD

		tx, err := db.Beginx()
		if err != nil {
			http.Error(w, "データベーストランザクションの開始に失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// 3. 【洗い替え】既存の在庫データをすべて削除
		// (package_stock と flag=0 の transaction_records)
		if err := database.ClearAllPackageStockInTx(tx); err != nil {
			http.Error(w, "既存在庫の全削除（洗い替え）に失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 4. 【洗い替え】product_master から「すべての」PackageKey を取得
		allMasterKeysMap, err := database.GetAllPackageKeysFromMasters(tx)
		if err != nil {
			http.Error(w, "全マスタのPackageKey取得に失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 5. すべての PackageKey をループして登録処理
		lineCounter := 0
		for key, keyInfo := range allMasterKeysMap {
			lineCounter++

			janQty := 0.0
			// CSVマップに存在するかチェック
			if csvQty, ok := csvStockMap[key]; ok {
				janQty = csvQty // 存在すればCSVの値を使用
			}
			// 存在しなければ janQty は 0.0 のまま (洗い替え)

			master := keyInfo.Representative
			if master == nil {
				log.Printf("WARN: No representative master found for PackageKey %s, skipping.", key)
				continue
			}

			yjQty := janQty * master.JanPackInnerQty

			// 5a. package_stock を更新（これが在庫の起点となる）
			if err :=
				database.UpsertPackageStockInTx(tx, key, master.YjCode, yjQty, date); err != nil {
				http.Error(w, "在庫起点(package_stock)の更新に失敗: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// 5b. transaction_records に棚卸明細(flag=0)を登録
			// (期限・ロットは不明のため、代表マスターのJANで合計行を1行だけ登録)
			tr := model.TransactionRecord{
				TransactionDate: date,
				Flag:            0,
				ReceiptNumber:   fmt.Sprintf("%s-%d", receiptNumber, lineCounter),
				LineNumber:      "1",
				JanCode:         master.ProductCode,
				JanQuantity:     janQty,
				YjQuantity:      yjQty,
				UnitPrice:       master.NhiPrice,
				Subtotal:        yjQty * master.NhiPrice,
				ExpiryDate:      "", // 期限不明
				LotNumber:       "", // ロット不明
			}
			mappers.MapMasterToTransaction(&tr, master)

			if err := database.InsertTransactionRecord(tx, tr); err != nil {
				http.Error(w, "棚卸明細(flag=0)の挿入に失敗: "+err.Error(), http.StatusInternalServerError)
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
			"message": fmt.Sprintf("在庫の洗い替えが完了しました。%d件のPackageKeyが処理されました。", len(allMasterKeysMap)),
		})
	}
}
