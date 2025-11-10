// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\guided_inventory.go
package database

import (
	"database/sql"
	"fmt"
	"strconv"
	"tkr/mappers"
	"tkr/model"
	"tkr/units" //

	"github.com/jmoiron/sqlx"
)

// DeleteTransactionsByFlagAndDateAndCodes は、指定されたフラグ、日付、製品コード群に一致する取引データを削除します。
// (WASABI: db/transaction_records.go  より移植)
// ▼▼▼【修正】[source]タグを文字列の外に移動 ▼▼▼
func DeleteTransactionsByFlagAndDateAndCodes(tx *sqlx.Tx, flag int, date string, productCodes []string) error {
	if len(productCodes) == 0 {
		return nil
	}

	query, args, err := sqlx.In(`DELETE FROM transaction_records WHERE flag = ?
 AND transaction_date = ? AND jan_code IN (?)`, flag, date, productCodes) //
	if err != nil {
		return fmt.Errorf("failed to create IN query for deleting transactions: %w", err)
	}
	query = tx.Rebind(query)

	_, err = tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete transactions by flag, date, and codes: %w", err)
	}
	return nil
}

// ▲▲▲【修正ここまで】▲▲▲

// SaveGuidedInventoryData は、棚卸調整画面からの入力データを棚卸レコード(flag=0)として保存します。
// (WASABI: db/guided_inventory.go  より移植・TKR用に修正)
func SaveGuidedInventoryData(tx *sqlx.Tx, date string, yjCode string, allPackagings []*model.ProductMaster, inventoryData map[string]float64, deadstockData []model.DeadStockRecord) error {

	var allProductCodes []string
	mastersMap := make(map[string]*model.ProductMaster)
	for _, pkg := range allPackagings {
		allProductCodes = append(allProductCodes, pkg.ProductCode)
		mastersMap[pkg.ProductCode] = pkg
	}

	// 1. 同日の既存棚卸データ(flag=0)を削除
	if len(allProductCodes) > 0 {
		if err := DeleteTransactionsByFlagAndDateAndCodes(tx, 0, date, allProductCodes); err != nil { //
			return fmt.Errorf("failed to delete old inventory records for the same day: %w", err)
		}
	}

	// ▼▼▼【ここから修正】伝票番号の採番ロジック (WASABI  を参考に変更) ▼▼▼
	var lastSeq int
	// ▼▼▼【修正】ADJ(14桁) -> AJ(13桁) ▼▼▼
	// AJyymmddnnnnn (AJ + 6桁 + 5桁 = 13桁)

	// YYYYMMDD (8桁) -> YYMMDD (6桁)
	var dateYYMMDD string
	if len(date) >= 8 {
		dateYYMMDD = date[2:8] // "20251031" -> "251031"
	} else if len(date) == 6 {
		dateYYMMDD = date // 既に6桁ならそのまま
	} else {
		return fmt.Errorf("invalid date format for receipt number: %s", date)
	}

	prefix := "AJ" + dateYYMMDD // "AJ251031"

	// データベースから 'AJ251031' で始まる最大の伝票番号を取得
	// ▲▲▲【修正ここまで】▲▲▲
	// ▼▼▼【修正】[source]タグを文字列の外に移動 ▼▼▼
	q := `SELECT receipt_number FROM transaction_records 
		  WHERE receipt_number LIKE ?
 ORDER BY receipt_number DESC LIMIT 1` //
	var lastReceiptNumber string
	err := tx.Get(&lastReceiptNumber, q, prefix+"%") //
	// ▲▲▲【修正ここまで】▲▲▲

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get last receipt number sequence for prefix %s: %w", prefix, err)
	}

	lastSeq = 0
	if lastReceiptNumber != "" {
		// ▼▼▼【修正】14桁 -> 13桁, 9文字目 -> 8文字目 ▼▼▼
		// "AJ25103100001" (13桁) から "00001" (5桁) の部分を取得
		if len(lastReceiptNumber) == 13 {
			seqStr := lastReceiptNumber[8:] // 8文字目以降 (AJ + 6桁 = 8桁)
			lastSeq, _ = strconv.Atoi(seqStr)
		}
	}

	newSeq := lastSeq + 1
	receiptNumber := fmt.Sprintf("%s%05d", prefix, newSeq) // 13桁 (AJ + 6 + 5)
	// ▲▲▲【修正ここまで】▲▲▲

	// ▼▼▼【削除】productCodesWithInventory (dead_stock_list 用) は不要 ▼▼▼
	// var productCodesWithInventory []string

	// ▼▼▼【ここから修正】packageStockTotalsYj（在庫起点用）の集計を deadStockData から inventoryData（合計マップ）基準に変更 ▼▼▼
	packageStockTotalsYj := make(map[string]float64) //

	for productCode, janQty := range inventoryData {
		master, ok := mastersMap[productCode]
		if !ok {
			continue // マスタ情報がない（YJグループ外の）データは無視
		}

		yjQty := janQty * master.JanPackInnerQty

		// 包装キー単位でYJ在庫量を集計
		packageKey := fmt.Sprintf("%s|%s|%g|%s", master.YjCode, master.PackageForm, master.JanPackInnerQty, units.ResolveName(master.YjUnitName)) //
		packageStockTotalsYj[packageKey] += yjQty                                                                                                 //
	}

	// ▼▼▼【ここから修正】2. deadStockData (明細) を基準に transaction_records(flag=0) を挿入 ▼▼▼
	// (JSが0の行も送信するようになったため、0の履歴も保存される)
	for i, ds := range deadstockData {
		master, ok := mastersMap[ds.ProductCode]
		if !ok {
			// allPackagings に含まれないマスタのデッドストックデータは無視
			continue
		}

		janQty := ds.StockQuantityJan
		yjQty := janQty * master.JanPackInnerQty //

		tr := model.TransactionRecord{
			TransactionDate: date,
			Flag:            0,             // 棚卸
			ReceiptNumber:   receiptNumber, // ★生成した伝票番号を使用
			LineNumber:      fmt.Sprintf("%d", i+1),
			JanCode:         master.ProductCode,
			YjCode:          master.YjCode,
			JanQuantity:     janQty,
			YjQuantity:      yjQty, // YJ単位に換算
			// ▼▼▼ 明細から期限とロットを設定 ▼▼▼
			ExpiryDate: ds.ExpiryDate,
			LotNumber:  ds.LotNumber,
		}

		// (棚卸の場合は薬価を単価とし、金額を計算)
		tr.UnitPrice = master.NhiPrice
		tr.Subtotal = tr.YjQuantity * tr.UnitPrice

		// 共通マッパー呼び出し
		mappers.MapMasterToTransaction(&tr, master)

		if err := InsertTransactionRecord(tx, tr); err != nil { //
			return fmt.Errorf("failed to insert inventory record for %s: %w", ds.ProductCode, err)
		}
	}
	// ▲▲▲【修正ここまで】▲▲▲

	// ▼▼▼【削除】3. ロット・期限情報を更新 (dead_stock_list 廃止のため不要) ▼▼▼
	//
	// ▲▲▲【削除ここまで】▲▲▲

	// ▼▼▼【ここから修正】4. package_stock テーブルを更新（在庫起点）▼▼▼
	// (集計済みの packageStockTotalsYj を使う)
	for key, totalYjQty := range packageStockTotalsYj {
		if err := UpsertPackageStockInTx(tx, key, yjCode, totalYjQty, date); err != nil { //
			return fmt.Errorf("failed to upsert package_stock for key %s: %w", key, err)
		}
	}
	// ▲▲▲【修正ここまで】▲▲▲

	return nil
}
