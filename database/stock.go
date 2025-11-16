// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\stock.go
package database

import (
	"database/sql"
	"fmt"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

func CalculateStockOnDate(dbtx *sqlx.DB, productCode string, targetDate string) (float64, error) {

	master, err := GetProductMasterByCode(dbtx, productCode)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get master for stock calculation (ProductCode: %s): %w", productCode, err)
	}

	packageKey := fmt.Sprintf("%s|%s|%g|%s", master.YjCode, master.PackageForm, master.JanPackInnerQty, master.YjUnitName)

	var latestInvDate string
	var baseStock float64

	var stockInfo model.PackageStock
	err = dbtx.Get(&stockInfo, `
		SELECT package_key, stock_quantity_yj, last_inventory_date 
		FROM package_stock 
		WHERE package_key = ? AND last_inventory_date <= ?
		ORDER BY last_inventory_date DESC 
LIMIT 1`,
		packageKey, targetDate)

	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("failed to get package_stock for %s on or before %s: %w", packageKey, targetDate, err)
	}

	if err == nil {
		baseStock = stockInfo.StockQuantityYj
		latestInvDate = stockInfo.LastInventoryDate

		if latestInvDate == targetDate {
			return baseStock, nil
		}

		var netChangeAfterInvDate sql.NullFloat64
		err = dbtx.Get(&netChangeAfterInvDate, `
			SELECT SUM(CASE WHEN flag IN (1, 11) THEN yj_quantity WHEN flag IN (2, 3, 12) THEN -yj_quantity ELSE 0 END)
			FROM transaction_records
			WHERE jan_code = ? AND flag IN (1, 2, 3, 11, 12) AND transaction_date > ? AND transaction_date <= ?`,
			productCode, latestInvDate, targetDate)

		if err != nil && err != sql.ErrNoRows {
			return 0, fmt.Errorf("failed to calculate net change after inventory date for %s: %w", productCode, err)
		}

		return baseStock + netChangeAfterInvDate.Float64, nil

	} else {
		var totalNetChange sql.NullFloat64

		err = dbtx.Get(&totalNetChange, `
			SELECT SUM(CASE WHEN flag IN (1, 11) THEN yj_quantity WHEN flag IN (2, 3, 12) THEN -yj_quantity ELSE 0 END)
			FROM transaction_records
			WHERE jan_code = ?
AND flag IN (1, 2, 3, 11, 12) AND transaction_date <= ?`,
			productCode, targetDate)

		if err != nil && err != sql.ErrNoRows {
			return 0, fmt.Errorf("failed to calculate total net change for %s: %w", productCode, err)
		}
		return totalNetChange.Float64, nil
	}
}
