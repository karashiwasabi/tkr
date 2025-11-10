// C:\Users\wasab\OneDrive\デスクトップ\TKR\stock\handler.go
package stock

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"

	"database/sql" // ▼▼▼【追加】sql パッケージをインポート ▼▼▼
	"strconv"      // ▼▼▼【追加】strconv パッケージをインポート ▼▼▼
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

// ▼▼▼【ここから修正】ExportCurrentStockHandler を product_master 基準に変更 ▼▼▼
func ExportCurrentStockHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		// 1. 在庫起点テーブル (package_stock) ではなく、
		//    product_master から全ての PackageKey 情報を取得
		allMasterKeysMap, err := database.GetAllPackageKeysFromMasters(db)
		if err != nil {
			http.Error(w, "Failed to get all package keys from masters: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// (GetRepresentativeProductNameMap は不要)

		var buf bytes.Buffer
		buf.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM

		// 3. ヘッダーを PackageKey, ProductName, JAN数量 に変更
		header := []string{
			"PackageKey",
			"ProductName",
			"JAN数量",
		}
		buf.WriteString(strings.Join(header, ",") + "\r\n")

		// 4. データをCSV行に変換 (allMasterKeysMap をループ)
		for key, keyInfo := range allMasterKeysMap {
			if keyInfo.Representative == nil {
				log.Printf("WARN: Skipping invalid PackageKey in export (no representative): %s", key)
				continue
			}

			// YJ数量 ではなく、JAN数量を 0.00 として出力
			janQty := 0.0

			// 代表品名を取得
			productName := keyInfo.Representative.ProductName

			record := []string{
				quoteAll(key), // マスタから取得した最新の PackageKey
				quoteAll(productName),
				quoteAll(fmt.Sprintf("%.2f", janQty)), // JAN数量は 0.00 固定
			}
			buf.WriteString(strings.Join(record, ",") + "\r\n")
		}
		// ▲▲▲【修正ここまで】▲▲▲

		today := time.Now().Format("20060102")
		filename := fmt.Sprintf("TKR在庫データ_%s.csv", today) // ファイル名を変更

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(filename))

		w.Write(buf.Bytes())
	}
}

// ▼▼▼ ImportTKRStockCSVHandler は変更なし (ご要望の「洗い替え」ロジックを実装済み) ▼▼▼
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
		// ▼▼▼【削除】古い伝票番号プレフィックス ▼▼▼
		// receiptNumber := "MIG_TKR_" + dateYYMMDD
		// ▲▲▲【削除ここまで】▲▲▲

		tx, err := db.Beginx()
		if err != nil {
			http.Error(w, "データベーストランザクションの開始に失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// ▼▼▼【ここから追加】伝票番号採番ロジック (AJ/IO/ADJ と同様) ▼▼▼
		var lastSeq int
		prefix := "MG" + dateYYMMDD // プレフィックスを "MG" (2桁) に変更 (Migrate)

		var lastReceiptNumber string
		// 'MG' + yymmdd で始まる最大の伝票番号を取得
		err = tx.Get(&lastReceiptNumber, `SELECT receipt_number FROM transaction_records WHERE receipt_number LIKE ? ORDER BY receipt_number DESC LIMIT 1`, prefix+"%")
		if err != nil && err != sql.ErrNoRows {
			http.Error(w, "伝票番号の採番に失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}

		lastSeq = 0
		if lastReceiptNumber != "" && len(lastReceiptNumber) == 13 { // 13桁チェック (MG+6+5)
			seqStr := lastReceiptNumber[8:] // 8文字目以降 (MG + 6桁 = 8桁)
			lastSeq, _ = strconv.Atoi(seqStr)
		}
		// ▲▲▲【追加ここまで】▲▲▲

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

			// 5a.package_stock を更新（これが在庫の起点となる）
			if err :=
				database.UpsertPackageStockInTx(tx, key, master.YjCode, yjQty, date); err != nil {
				http.Error(w, "在庫起点(package_stock)の更新に失敗: "+err.Error(), http.StatusInternalServerError)
				return
			}

			// ▼▼▼【ここから修正】13桁の伝票番号を生成 ▼▼▼
			newSeq := lastSeq + lineCounter // 連番を計算 (i ではなく lineCounter を使用)
			receiptNumber := fmt.Sprintf("%s%05d", prefix, newSeq)
			// ▲▲▲【修正ここまで】▲▲▲

			// 5b.transaction_records に棚卸明細(flag=0)を登録
			// (期限・ロットは不明のため、代表マスターのJANで合計行を1行だけ登録)
			tr := model.TransactionRecord{
				TransactionDate: date,
				Flag:            0,
				// ▼▼▼【修正】ハイフン連結をやめ、生成した13桁の番号を使用 ▼▼▼
				ReceiptNumber: receiptNumber,
				// ▲▲▲【修正ここまで】▲▲▲
				LineNumber:  "1",
				JanCode:     master.ProductCode,
				JanQuantity: janQty,
				YjQuantity:  yjQty,
				UnitPrice:   master.NhiPrice,
				Subtotal:    yjQty * master.NhiPrice,
				ExpiryDate:  "", // 期限不明
				LotNumber:   "", // ロット不明
			}
			mappers.MapMasterToTransaction(&tr, master)

			if err := database.InsertTransactionRecord(tx, tr); err != nil {
				http.Error(w, "棚卸明細(flag=0)の挿入に失敗: "+err.Error(), http.StatusInternalServerError)
				return
			}
		}

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
