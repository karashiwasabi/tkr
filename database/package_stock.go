// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\package_stock.go
package database

import (
	"fmt"
	"strconv"
	"strings"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

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

// GetPackageStockByYjCode は単一のYJコードで検索します（既存互換用）
func GetPackageStockByYjCode(dbtx DBTX, yjCode string) (map[string]model.PackageStock, error) {
	var stocks []model.PackageStock
	const q = `
		SELECT package_key, yj_code, stock_quantity_yj, last_inventory_date
		FROM package_stock
		WHERE yj_code = ?`
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

// ▼▼▼【追加】複数のPackageKeyで一括取得する関数 ▼▼▼
func GetPackageStocksByKeys(dbtx DBTX, keys []string) (map[string]model.PackageStock, error) {
	stockMap := make(map[string]model.PackageStock)
	if len(keys) == 0 {
		return stockMap, nil
	}

	// sqlx.In を使用して IN 句を構築
	query, args, err := sqlx.In(`
		SELECT package_key, yj_code, stock_quantity_yj, last_inventory_date
		FROM package_stock
		WHERE package_key IN (?)`, keys)
	if err != nil {
		return nil, fmt.Errorf("failed to construct IN query for package_stock keys: %w", err)
	}

	if rebinder, ok := dbtx.(interface{ Rebind(string) string }); ok {
		query = rebinder.Rebind(query)
	}

	var stocks []model.PackageStock
	err = dbtx.Select(&stocks, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get package_stocks by keys: %w", err)
	}

	for _, s := range stocks {
		stockMap[s.PackageKey] = s
	}
	return stockMap, nil
}

// ▲▲▲【追加ここまで】▲▲▲

type ParsedPackageKey struct {
	YjCode          string
	PackageForm     string
	JanPackInnerQty float64
	YjUnitName      string
}

func ParsePackageKey(key string) (*ParsedPackageKey, error) {
	parts := strings.Split(key, "|")
	if len(parts) != 4 {
		return nil, fmt.Errorf("invalid package key format: %s", key)
	}

	innerQty, err := strconv.ParseFloat(parts[2], 64)
	if err != nil {
		return nil, fmt.Errorf("invalid JanPackInnerQty in package key (%s): %w", parts[2], err)
	}

	return &ParsedPackageKey{
		YjCode:          parts[0],
		PackageForm:     parts[1],
		JanPackInnerQty: innerQty,
		YjUnitName:      parts[3],
	}, nil
}
