// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\package_stock.go
package database

import (
	"fmt"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

// UpsertPackageStockInTx は、包装キー単位の在庫データを挿入または更新します。
func UpsertPackageStockInTx(tx *sqlx.Tx, packageKey string, yjCode string, quantityYj float64, inventoryDate string) error {
	const q = `
		INSERT INTO package_stock (package_key, yj_code, stock_quantity_yj, last_inventory_date)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(package_key) DO UPDATE SET
			stock_quantity_yj = excluded.stock_quantity_yj,
			last_inventory_date = excluded.last_inventory_date,
			yj_code = excluded.yj_code
	`
	_, err := tx.Exec(q, packageKey, yjCode, quantityYj, inventoryDate)
	if err != nil {
		return fmt.Errorf("failed to upsert package_stock for key %s: %w", packageKey, err)
	}
	return nil
}

// GetPackageStockByYjCode は、YJコードに紐づく全ての包装在庫レコードを取得します。
func GetPackageStockByYjCode(dbtx DBTX, yjCode string) (map[string]model.PackageStock, error) {
	var stocks []model.PackageStock
	const q = `
		SELECT package_key, yj_code, stock_quantity_yj, last_inventory_date
		FROM package_stock
		WHERE yj_code = ?
	`
	err := dbtx.Select(&stocks, q, yjCode)
	if err != nil {
		return nil, fmt.Errorf("failed to get package_stock by yj_code %s: %w", yjCode, err)
	}

	stockMap := make(map[string]model.PackageStock)
	for _, s := range stocks {
		stockMap[s.PackageKey] = s
	}
	return stockMap, nil
}
