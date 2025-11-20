// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\backorders.go
package database

import (
	"database/sql"
	"fmt"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

func InsertBackordersInTx(tx *sqlx.Tx, backorders []model.Backorder) error {
	const q = `
		INSERT INTO backorders (
			order_date, jan_code, yj_code, product_name, package_form, jan_pack_inner_qty, 
			yj_unit_name, order_quantity, remaining_quantity, wholesaler_code,
			yj_pack_unit_qty, jan_pack_unit_qty, jan_unit_code
		) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	stmt, err := tx.Prepare(q)
	if err != nil {
		return fmt.Errorf("failed to prepare backorder insert statement: %w", err)
	}
	defer stmt.Close()

	for _, bo := range backorders {
		_, err := stmt.Exec(
			bo.OrderDate, bo.JanCode, bo.YjCode, bo.ProductName, bo.PackageForm, bo.JanPackInnerQty,
			bo.YjUnitName, bo.OrderQuantity, bo.RemainingQuantity, bo.WholesalerCode,
			bo.YjPackUnitQty, bo.JanPackUnitQty, bo.JanUnitCode,
		)
		if err != nil {
			return fmt.Errorf("failed to execute backorder insert for yj %s: %w", bo.YjCode, err)
		}
	}
	return nil
}

func ReconcileBackorders(tx *sqlx.Tx, deliveredItems []model.Backorder) error {
	for _, item := range deliveredItems {
		deliveryQty := item.YjQuantity

		// 消込処理では予約分(_RSV)を対象外にする（勝手に消されないように保護）
		rows, err := tx.Query(`
			SELECT id, remaining_quantity FROM backorders 
			WHERE yj_code = ? AND package_form = ? AND jan_pack_inner_qty = ? AND yj_unit_name = ?
			AND wholesaler_code NOT LIKE '%_RSV'
			ORDER BY order_date, id`,
			item.YjCode, item.PackageForm, item.JanPackInnerQty, item.YjUnitName,
		)
		if err != nil {
			return fmt.Errorf("failed to query backorders for reconciliation: %w", err)
		}

		type updateAction struct {
			id           int
			deleteRecord bool
			newRemaining float64
		}
		var actions []updateAction

		for rows.Next() {
			if deliveryQty <= 0 {
				break
			}
			var id int
			var remainingQty float64
			if err := rows.Scan(&id, &remainingQty); err != nil {
				rows.Close()
				return fmt.Errorf("failed to scan backorder row: %w", err)
			}

			if deliveryQty >= remainingQty {
				actions = append(actions, updateAction{id: id, deleteRecord: true})
				deliveryQty -= remainingQty
			} else {
				newRemaining := remainingQty - deliveryQty
				actions = append(actions, updateAction{id: id, deleteRecord: false, newRemaining: newRemaining})
				deliveryQty = 0
			}
		}
		rows.Close()

		for _, action := range actions {
			if action.deleteRecord {
				if _, err := tx.Exec(`DELETE FROM backorders WHERE id = ?`, action.id); err != nil {
					return fmt.Errorf("failed to delete reconciled backorder id %d: %w", action.id, err)
				}
			} else {
				if _, err := tx.Exec(`UPDATE backorders SET remaining_quantity = ? WHERE id = ?`, action.newRemaining, action.id); err != nil {
					return fmt.Errorf("failed to update partially reconciled backorder id %d: %w", action.id, err)
				}
			}
		}
	}
	return nil
}

func GetAllBackordersMap(dbtx DBTX) (map[string]float64, error) {
	// ★修正: WHERE句を削除しました。これで予約分も集計に含まれます。
	const q = `
		SELECT yj_code, package_form, jan_pack_inner_qty, yj_unit_name, SUM(remaining_quantity)
		FROM backorders
		GROUP BY yj_code, package_form, jan_pack_inner_qty, yj_unit_name`

	rows, err := dbtx.Query(q)
	if err != nil {
		return nil, fmt.Errorf("failed to query all backorders map: %w", err)
	}
	defer rows.Close()

	backordersMap := make(map[string]float64)
	for rows.Next() {
		var yjCode, packageForm, yjUnitName string
		var janPackInnerQty, totalRemaining float64
		if err := rows.Scan(&yjCode, &packageForm, &janPackInnerQty, &yjUnitName, &totalRemaining); err != nil {
			return nil, err
		}
		key := fmt.Sprintf("%s|%s|%g|%s", yjCode, packageForm, janPackInnerQty, yjUnitName)
		backordersMap[key] = totalRemaining
	}
	return backordersMap, nil
}

func GetAllBackordersList(dbtx DBTX) ([]model.Backorder, error) {
	const q = `
		SELECT
			id, order_date, jan_code, yj_code, product_name, package_form, jan_pack_inner_qty, 
			yj_unit_name, order_quantity, remaining_quantity, wholesaler_code,
			yj_pack_unit_qty, jan_pack_unit_qty, jan_unit_code
		FROM backorders
		ORDER BY order_date, wholesaler_code, product_name, id
	`

	rows, err := dbtx.Query(q)
	if err != nil {
		return nil, fmt.Errorf("failed to query all backorders list: %w", err)
	}
	defer rows.Close()

	var backorders []model.Backorder
	for rows.Next() {
		var bo model.Backorder
		var wholesalerCode sql.NullString
		var janCode sql.NullString

		if err := rows.Scan(
			&bo.ID, &bo.OrderDate, &janCode, &bo.YjCode, &bo.ProductName, &bo.PackageForm, &bo.JanPackInnerQty,
			&bo.YjUnitName, &bo.OrderQuantity, &bo.RemainingQuantity, &wholesalerCode,
			&bo.YjPackUnitQty, &bo.JanPackUnitQty, &bo.JanUnitCode,
		); err != nil {
			return nil, err
		}

		bo.WholesalerCode = wholesalerCode.String
		bo.JanCode = janCode.String

		backorders = append(backorders, bo)
	}
	return backorders, nil
}

func DeleteBackorderInTx(tx *sqlx.Tx, id int) error {
	const q = `DELETE FROM backorders WHERE id = ?`
	res, err := tx.Exec(q, id)
	if err != nil {
		return fmt.Errorf("failed to delete backorder for id %d: %w", id, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected for backorder id %d: %w", id, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("no backorder found to delete for id %d", id)
	}
	return nil
}

func DeleteBackordersByOrderDateInTx(tx *sqlx.Tx, orderDate string) (int64, error) {
	const q = `DELETE FROM backorders WHERE order_date = ?`
	res, err := tx.Exec(q, orderDate)
	if err != nil {
		return 0, fmt.Errorf("failed to delete backorders for orderDate %s: %w", orderDate, err)
	}
	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected for orderDate %s: %w", orderDate, err)
	}
	if rowsAffected == 0 {
		return 0, fmt.Errorf("no backorder found to delete for orderDate %s", orderDate)
	}
	return rowsAffected, nil
}

// 予約解除判定用の関数
func GetExpiredReservationsInTx(tx *sqlx.Tx, nowStr string) ([]model.Backorder, error) {
	const q = `
		SELECT
			id, order_date, jan_code, yj_code, product_name, package_form, jan_pack_inner_qty, 
			yj_unit_name, order_quantity, remaining_quantity, wholesaler_code,
			yj_pack_unit_qty, jan_pack_unit_qty, jan_unit_code
		FROM backorders
		WHERE REPLACE(order_date, 'T', '') <= ? AND wholesaler_code LIKE '%_RSV'
	`
	var records []model.Backorder
	rows, err := tx.Query(q, nowStr)
	if err != nil {
		return nil, fmt.Errorf("failed to query expired reservations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var bo model.Backorder
		var wholesalerCode sql.NullString
		var janCode sql.NullString

		if err := rows.Scan(
			&bo.ID, &bo.OrderDate, &janCode, &bo.YjCode, &bo.ProductName, &bo.PackageForm, &bo.JanPackInnerQty,
			&bo.YjUnitName, &bo.OrderQuantity, &bo.RemainingQuantity, &wholesalerCode,
			&bo.YjPackUnitQty, &bo.JanPackUnitQty, &bo.JanUnitCode,
		); err != nil {
			return nil, fmt.Errorf("failed to scan expired reservation: %w", err)
		}
		bo.WholesalerCode = wholesalerCode.String
		bo.JanCode = janCode.String
		records = append(records, bo)
	}

	return records, nil
}
