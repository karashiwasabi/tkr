// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\stock.go
package database

import (
	"database/sql"
	"fmt"

	// TKRのmodelを使用
	"github.com/jmoiron/sqlx"
)

// SignedYjQty はフラグに基づいて符号付きのYJ数量を返します。
// TKRの集計では 納品(1) と 処方(3) のみ考慮します。
func signedYjQty(flag int, yjQty float64) float64 {
	switch flag {
	case 1: // 納品
		return yjQty
	case 11: // 入庫
		return yjQty
	case 3: // 処方
		return -yjQty
	case 2: // 返品
		return -yjQty
	case 12: // 出庫
		return -yjQty
	default: // 棚卸(0), その他
		return 0
	}
}

// CalculateStockOnDate は指定された製品の、特定の日付時点での理論在庫を計算します。
// (WASABI: db/stock.go より移植・TKR用に修正)
func CalculateStockOnDate(dbtx *sqlx.DB, productCode string, targetDate string) (float64, error) {
	var latestInventoryDate sql.NullString
	// 1. 基準日以前の最新の棚卸日を取得
	err := dbtx.Get(&latestInventoryDate, `
		SELECT MAX(transaction_date) FROM transaction_records
		WHERE jan_code = ?
AND flag = 0 AND transaction_date <= ?`,
		productCode, targetDate)
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to get latest inventory date for %s on or before %s: %w", productCode, targetDate, err)
	}

	if latestInventoryDate.Valid && latestInventoryDate.String != "" {
		// --- 棚卸履歴がある場合の計算 ---
		var baseStock float64
		// 1a.棚卸日の在庫合計を基点とする
		err := dbtx.Get(&baseStock, `
			SELECT SUM(yj_quantity) FROM transaction_records
			WHERE jan_code = ? AND flag = 0 AND transaction_date = ?`,
			productCode, latestInventoryDate.String)
		if err != nil {
			return 0, fmt.Errorf("failed to sum inventory for %s on %s: %w", productCode, latestInventoryDate.String, err)
		}

		// 1b.もし基準日が棚卸日当日なら、棚卸数量のみを返す
		if latestInventoryDate.String == targetDate {
			return baseStock, nil
		}

		// 1c. 棚卸日の翌日から基準日までの変動を計算 (flag 1, 2, 3, 11, 12 を全て考慮)
		var netChangeAfterInvDate sql.NullFloat64
		// ▼▼▼【ここを修正】納品(1), 返品(2), 処方(3), 入庫(11), 出庫(12) を全て含める ▼▼▼
		err = dbtx.Get(&netChangeAfterInvDate, `
			SELECT SUM(CASE WHEN flag IN (1, 11) THEN yj_quantity WHEN flag IN (2, 3, 12) THEN -yj_quantity ELSE 0 END)
			FROM transaction_records
			WHERE jan_code = ? AND flag IN (1, 2, 3, 11, 12) AND transaction_date > ? AND transaction_date <= ?`,
			productCode, latestInventoryDate.String, targetDate)
		// ▲▲▲【修正ここまで】▲▲▲
		if err != nil && err != sql.ErrNoRows {
			return 0, fmt.Errorf("failed to calculate net change after inventory date for %s: %w", productCode, err)
		}

		return baseStock + netChangeAfterInvDate.Float64, nil

	} else {
		// --- 棚卸履歴がない場合の計算 ---
		var totalNetChange sql.NullFloat64
		// ▼▼▼【ここを修正】納品(1), 返品(2), 処方(3), 入庫(11), 出庫(12) を全て含める ▼▼▼
		err = dbtx.Get(&totalNetChange, `
			SELECT SUM(CASE WHEN flag IN (1, 11) THEN yj_quantity WHEN flag IN (2, 3, 12) THEN -yj_quantity ELSE 0 END)
			FROM transaction_records
			WHERE jan_code = ? AND flag IN (1, 2, 3, 11, 12) AND transaction_date <= ?`,
			productCode, targetDate)
		// ▲▲▲【修正ここまで】▲▲▲
		if err != nil && err != sql.ErrNoRows {
			return 0, fmt.Errorf("failed to calculate total net change for %s: %w", productCode, err)
		}
		return totalNetChange.Float64, nil
	}
}

// (GetAllCurrentStockMap は変更なし)
func GetAllCurrentStockMap(conn *sqlx.DB) (map[string]float64, error) {
	rows, err := conn.Query(`
		SELECT jan_code, transaction_date, flag, yj_quantity 
		FROM transaction_records 
		ORDER BY jan_code, transaction_date, id`)
	if err != nil {
		return nil, fmt.Errorf("failed to get all transactions for stock calculation: %w", err)
	}
	defer rows.Close()

	stockMap := make(map[string]float64)

	type txRecord struct {
		Date string
		Flag int
		Qty  float64
	}
	recordsByJan := make(map[string][]txRecord)

	for rows.Next() {
		var janCode, date string
		var flag int
		var qty float64
		if err := rows.Scan(&janCode, &date, &flag, &qty); err != nil {
			return nil, err
		}
		if janCode == "" {
			continue
		}
		recordsByJan[janCode] = append(recordsByJan[janCode], txRecord{Date: date, Flag: flag, Qty: qty})
	}

	for janCode, records := range recordsByJan {
		var latestInvDate string
		baseStock := 0.0

		invStocksOnDate := make(map[string]float64)
		for _, r := range records {
			if r.Flag == 0 {
				if r.Date > latestInvDate {
					latestInvDate = r.Date
				}
				invStocksOnDate[r.Date] += r.Qty
			}
		}
		if latestInvDate != "" {
			baseStock = invStocksOnDate[latestInvDate]
		}

		netChange := 0.0
		for _, r := range records {
			startDate := "00000000"
			if latestInvDate != "" {
				startDate = latestInvDate
			}

			if r.Date > startDate {
				// TKR用に signedYjQty を使う
				netChange += signedYjQty(r.Flag, r.Qty)
			}
		}
		stockMap[janCode] = baseStock + netChange
	}

	return stockMap, nil
}
