// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\deadstock.go
package database

import (
	"fmt"
	"time"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

// GetDeadStockByYjCode は指定されたYJコードに紐づくロット・期限情報を取得します。
// (WASABI: db/deadstock.go  より移植)
func GetDeadStockByYjCode(tx *sqlx.Tx, yjCode string) ([]model.DeadStockRecord, error) {
	const q = `
		SELECT id, product_code, stock_quantity_jan, expiry_date, lot_number,
		       yj_code, package_form, jan_pack_inner_qty, yj_unit_name
		FROM dead_stock_list 
		WHERE yj_code = ?
		ORDER BY product_code, expiry_date, lot_number`

	rows, err := tx.Queryx(q, yjCode)
	if err != nil {
		return nil, fmt.Errorf("failed to query dead stock by yj_code: %w", err)
	}
	defer rows.Close()

	var records []model.DeadStockRecord
	for rows.Next() {
		var r model.DeadStockRecord
		if err := rows.Scan(&r.ID, &r.ProductCode, &r.StockQuantityJan, &r.ExpiryDate, &r.LotNumber, &r.YjCode, &r.PackageForm, &r.JanPackInnerQty, &r.YjUnitName); err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

// DeleteDeadStockByProductCodesInTx は指定された製品コード群のロット・期限情報を削除します。
// (WASABI: db/deadstock.go  より移植)
func DeleteDeadStockByProductCodesInTx(tx *sqlx.Tx, productCodes []string) error {
	if len(productCodes) == 0 {
		return nil
	}
	query, args, err := sqlx.In("DELETE FROM dead_stock_list WHERE product_code IN (?)", productCodes)
	if err != nil {
		return fmt.Errorf("failed to create IN query for deleting dead stock: %w", err)
	}
	query = tx.Rebind(query)
	_, err = tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete dead stock records by product codes: %w", err)
	}
	return nil
}

// SaveDeadStockListInTx はロット・期限情報のリストを保存（UPSERT）します。
// (WASABI: db/deadstock.go  より移植)
func SaveDeadStockListInTx(tx *sqlx.Tx, records []model.DeadStockRecord) error {
	const q = `
        INSERT INTO dead_stock_list 
        (product_code, yj_code, package_form, jan_pack_inner_qty, yj_unit_name, 
        stock_quantity_jan, expiry_date, lot_number, created_at)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(product_code, expiry_date, lot_number) DO UPDATE SET
		stock_quantity_jan = excluded.stock_quantity_jan,
		yj_code = excluded.yj_code,
		package_form = excluded.package_form,
		jan_pack_inner_qty = excluded.jan_pack_inner_qty,
		yj_unit_name = excluded.yj_unit_name,
		created_at = excluded.created_at`

	stmt, err := tx.Prepare(q)
	if err != nil {
		return fmt.Errorf("failed to prepare statement for dead_stock_list: %w", err)
	}
	defer stmt.Close()

	createdAt := time.Now().Format("2006-01-02 15:04:05")

	for _, rec := range records {
		_, err := stmt.Exec(
			rec.ProductCode, rec.YjCode, rec.PackageForm, rec.JanPackInnerQty, rec.YjUnitName,
			rec.StockQuantityJan, rec.ExpiryDate, rec.LotNumber, createdAt,
		)
		if err != nil {
			return fmt.Errorf("failed to insert/replace dead_stock_list for product %s: %w", rec.ProductCode, err)
		}
	}
	return nil
}
