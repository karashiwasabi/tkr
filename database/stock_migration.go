// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\stock_migration.go
package database

import (
	"database/sql"
	"fmt"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

func ClearAllPackageStockInTx(tx *sqlx.Tx) error {
	_, err := tx.Exec("DELETE FROM package_stock")
	if err !=
		nil {
		return fmt.Errorf("failed to clear package_stock: %w", err)
	}
	_, err = tx.Exec("DELETE FROM transaction_records WHERE flag = 0")
	if err != nil {
		return fmt.Errorf("failed to clear old inventory transactions (flag=0): %w", err)
	}
	return nil
}

func GetAllPackageStock(db *sqlx.DB) ([]model.PackageStock, error) {
	var stocks []model.PackageStock
	const q = `
		SELECT package_key, yj_code, stock_quantity_yj, last_inventory_date
		FROM package_stock
		ORDER BY yj_code
	`
	err := db.Select(&stocks, q)
	if err != nil {
		return nil, fmt.Errorf("failed to get all package_stock: %w", err)
	}
	return stocks, nil
}

func GetRepresentativeProductNameMap(db *sqlx.DB) (map[string]string, error) {
	var results []struct {
		YjCode      string `db:"yj_code"`
		ProductName string `db:"product_name"`
	}

	const q = `
		SELECT yj_code, product_name
		FROM product_master
		WHERE (yj_code, origin = 'JCSHMS') IN (
			SELECT yj_code, MAX(origin = 'JCSHMS')
			FROM product_master
			WHERE yj_code 
!= ''
			GROUP BY yj_code
		)
	`
	err := db.Select(&results, q)
	if err != nil {
		return nil, fmt.Errorf("failed to get representative product names: %w", err)
	}

	nameMap := make(map[string]string)
	for _, r := range results {
		nameMap[r.YjCode] = r.ProductName
	}
	return nameMap, nil
}

type StockDetailItem struct {
	JanCode     string  `db:"jan_code"`
	Gs1Code     string  `db:"gs1_code"`
	ProductName string  `db:"product_name"`
	JanQuantity float64 `db:"jan_quantity"`
	ExpiryDate  string  `db:"expiry_date"`
	LotNumber   string  `db:"lot_number"`
}

func GetCurrentStockDetails(db *sqlx.DB) ([]StockDetailItem, error) {
	var items []StockDetailItem

	var latestInventoryDate string
	err := db.Get(&latestInventoryDate, `
		SELECT MAX(last_inventory_date) 
		FROM package_stock
	`)
	if err != nil {
		if err == sql.ErrNoRows {
			return items, nil
		}
		return nil, fmt.Errorf("failed to get latest inventory date from package_stock: %w", err)
	}

	if latestInventoryDate == "" {
		return items, nil
	}

	query := `
		SELECT 
			T.jan_code, 
			COALESCE(P.gs1_code, '') AS gs1_code, 
			T.product_name, 
			T.jan_quantity, 
			T.expiry_date, 
			T.lot_number
		FROM transaction_records AS T
		LEFT JOIN product_master AS P ON T.jan_code = P.product_code
		WHERE T.flag = 0 
		  AND T.transaction_date = ?
AND T.jan_quantity > 0
		ORDER BY P.kana_name, T.jan_code, T.expiry_date, T.lot_number
	`

	if err := db.Select(&items, query, latestInventoryDate); err != nil {
		return nil, fmt.Errorf("failed to select current stock details: %w", err)
	}

	return items, nil
}

// ▼▼▼【ここから修正】製品ごとの最新日以外を削除するロジックに変更 ▼▼▼
func DeleteOldInventoryTransactions(tx *sqlx.Tx) (int64, error) {
	// package_stock に (yj_code, last_inventory_date) のペアで存在する
	// 最新の棚卸履歴(flag=0) *以外* の、古い履歴を削除する
	const q = `
		DELETE FROM transaction_records
		WHERE flag = 0 AND id IN (
			SELECT T.id
			FROM transaction_records AS T
			LEFT JOIN package_stock AS P ON T.yj_code = P.yj_code AND T.transaction_date = P.last_inventory_date
			WHERE T.flag = 0 AND P.package_key IS NULL
		)
	`
	res, err := tx.Exec(q)
	if err != nil {
		return 0, fmt.Errorf("failed to delete old inventory transactions: %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get affected rows: %w", err)
	}

	return rowsAffected, nil
}

// ▲▲▲【修正ここまで】▲▲▲
