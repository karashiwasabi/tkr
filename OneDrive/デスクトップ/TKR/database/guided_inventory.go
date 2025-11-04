// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\guided_inventory.go
package database

import (
	"database/sql" // ▼▼▼【ここに追加】▼▼▼
	"fmt"
	"strconv"     // ▼▼▼【ここに追加】▼▼▼
	"tkr/mappers" // ▼▼▼【ここに追加】▼▼▼
	"tkr/model"

	// "tkr/units" // 削除 (mappers が担当)

	"github.com/jmoiron/sqlx"
)

// DeleteTransactionsByFlagAndDateAndCodes は、指定されたフラグ、日付、製品コード群に一致する取引データを削除します。
// (WASABI: db/transaction_records.go  より移植)
func DeleteTransactionsByFlagAndDateAndCodes(tx *sqlx.Tx, flag int, date string, productCodes []string) error {
	if len(productCodes) == 0 {
		return nil
	}

	query, args, err := sqlx.In(`DELETE FROM transaction_records WHERE flag = ? AND transaction_date = ? AND jan_code IN (?)`, flag, date, productCodes)
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
		if err := DeleteTransactionsByFlagAndDateAndCodes(tx, 0, date, allProductCodes); err != nil {
			return fmt.Errorf("failed to delete old inventory records for the same day: %w", err)
		}
	}

	// ▼▼▼【ここから修正】伝票番号の採番ロジック (WASABI  を参考に変更) ▼▼▼
	var lastSeq int
	// ADJyymmddnnnnn (ADJ + 6桁 + 5桁 = 14桁)

	// YYYYMMDD (8桁) -> YYMMDD (6桁)
	var dateYYMMDD string
	if len(date) >= 8 {
		dateYYMMDD = date[2:8] // "20251031" -> "251031"
	} else if len(date) == 6 {
		dateYYMMDD = date // 既に6桁ならそのまま
	} else {
		return fmt.Errorf("invalid date format for receipt number: %s", date)
	}

	prefix := "ADJ" + dateYYMMDD // "ADJ251031"

	// データベースから 'ADJ251031' で始まる最大の伝票番号を取得
	q := `SELECT receipt_number FROM transaction_records 
		  WHERE receipt_number LIKE ? 
		  ORDER BY receipt_number DESC LIMIT 1`
	var lastReceiptNumber string
	err := tx.Get(&lastReceiptNumber, q, prefix+"%")

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get last receipt number sequence for prefix %s: %w", prefix, err)
	}

	lastSeq = 0
	if lastReceiptNumber != "" {
		// "ADJ25103100001" (14桁) から "00001" (5桁) の部分を取得
		if len(lastReceiptNumber) == 14 {
			seqStr := lastReceiptNumber[9:] // 9文字目以降 (ADJ + 6桁 = 9桁)
			lastSeq, _ = strconv.Atoi(seqStr)
		}
	}

	newSeq := lastSeq + 1
	receiptNumber := fmt.Sprintf("%s%05d", prefix, newSeq) // 14桁 (ADJ + 6 + 5)
	// ▲▲▲【修正ここまで】▲▲▲

	var productCodesWithInventory []string

	// 2. 新しい棚卸データを挿入
	for i, productCode := range allProductCodes {
		master, ok := mastersMap[productCode]
		if !ok {
			continue
		}

		janQty := inventoryData[productCode]
		// 在庫が0より大きい（＝入力があった）コードを記録
		if janQty > 0 {
			productCodesWithInventory = append(productCodesWithInventory, productCode)
		}

		tr := model.TransactionRecord{
			TransactionDate: date,
			Flag:            0,             // 棚卸
			ReceiptNumber:   receiptNumber, // ★生成した伝票番号を使用
			LineNumber:      fmt.Sprintf("%d", i+1),
			JanCode:         master.ProductCode,
			YjCode:          master.YjCode,
			JanQuantity:     janQty,
			YjQuantity:      janQty * master.JanPackInnerQty,
		}

		// (棚卸の場合は薬価を単価とし、金額を計算)
		tr.UnitPrice = master.NhiPrice
		tr.Subtotal = tr.YjQuantity * tr.UnitPrice

		// 共通マッパー呼び出し
		mappers.MapMasterToTransaction(&tr, master)

		if err := InsertTransactionRecord(tx, tr); err != nil {
			return fmt.Errorf("failed to insert inventory record for %s: %w", productCode, err)
		}
	}

	// 3. ロット・期限情報を更新 (入力があったもののみ)
	if len(productCodesWithInventory) > 0 {
		var relevantDeadstockData []model.DeadStockRecord
		for _, ds := range deadstockData {
			for _, pid := range productCodesWithInventory {
				if ds.ProductCode == pid {
					relevantDeadstockData = append(relevantDeadstockData, ds)
					break
				}
			}
		}

		// このYJコードに関連する既存のロット・期限情報を一度すべて削除
		if err := DeleteDeadStockByProductCodesInTx(tx, allProductCodes); err != nil {
			return fmt.Errorf("failed to delete old dead stock records: %w", err)
		}
		// 新しいロット・期限情報（在庫>0のもの）を保存
		if len(relevantDeadstockData) > 0 {
			if err := SaveDeadStockListInTx(tx, relevantDeadstockData); err != nil {
				return fmt.Errorf("failed to upsert new dead stock records: %w", err)
			}
		}
	}

	return nil
}
