// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\backorders.go
package database

import (
	"database/sql"
	"fmt"
	"tkr/model" // TKRのモデルを参照

	"github.com/jmoiron/sqlx"
)

/**
 * @brief 複数の発注残レコードをトランザクション内で登録します（INSERT）。
 * [cite_start](WASABI: db/backorders.go [cite: 2446] より移植)
 */
func InsertBackordersInTx(tx *sqlx.Tx, backorders []model.Backorder) error {
	const q = `
		INSERT INTO backorders (
			order_date, yj_code, product_name, package_form, jan_pack_inner_qty, 
			yj_unit_name, order_quantity, remaining_quantity, wholesaler_code,
			yj_pack_unit_qty, jan_pack_unit_qty, jan_unit_code
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	stmt, err := tx.Prepare(q)
	if err != nil {
		return fmt.Errorf("failed to prepare backorder insert statement: %w", err)
	}
	defer stmt.Close()

	for _, bo := range backorders {
		_, err := stmt.Exec(
			bo.OrderDate, bo.YjCode, bo.ProductName, bo.PackageForm, bo.JanPackInnerQty,
			bo.YjUnitName, bo.OrderQuantity, bo.RemainingQuantity, bo.WholesalerCode,
			bo.YjPackUnitQty, bo.JanPackUnitQty, bo.JanUnitCode,
		)
		if err != nil {
			return fmt.Errorf("failed to execute backorder insert for yj %s: %w", bo.YjCode, err)
		}
	}
	return nil
}

/**
 * @brief 納品データに基づいて発注残を消し込みます（FIFO: 先入れ先出し）。
 * [cite_start](WASABI: db/backorders.go [cite: 2447] より移植・TKR用に *sqlx.Tx を使用)
 */
func ReconcileBackorders(tx *sqlx.Tx, deliveredItems []model.Backorder) error {
	for _, item := range deliveredItems {
		deliveryQty := item.YjQuantity

		rows, err := tx.Query(`
			SELECT id, remaining_quantity FROM backorders 
			WHERE yj_code = ? AND package_form = ? AND jan_pack_inner_qty = ? AND yj_unit_name = ?
			ORDER BY order_date, id`,
			item.YjCode, item.PackageForm, item.JanPackInnerQty, item.YjUnitName,
		)
		if err != nil {
			return fmt.Errorf("failed to query backorders for reconciliation: %w", err)
		}

		// デッドロックを避けるため、更新/削除対象のIDを先に収集
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
				// 納品数で発注残が完全にカバーされる場合
				actions = append(actions, updateAction{id: id, deleteRecord: true})
				deliveryQty -= remainingQty
			} else {
				// 納品数の一部で発注残を減らす場合
				newRemaining := remainingQty - deliveryQty
				actions = append(actions, updateAction{id: id, deleteRecord: false, newRemaining: newRemaining})
				deliveryQty = 0
			}
		}
		rows.Close() // スキャンループの直後にClose

		// 収集したアクションを実行
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

/**
 * @brief 全ての発注残を、集計で高速に参照できるマップ形式で取得します。
 * [cite_start](WASABI: db/backorders.go [cite: 2452] より移植・TKR用に DBTX を使用)
 */
func GetAllBackordersMap(dbtx DBTX) (map[string]float64, error) {
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
		// TKRの package_key 形式 (YjUnitNameをResolveする)
		key := fmt.Sprintf("%s|%s|%g|%s", yjCode, packageForm, janPackInnerQty, yjUnitName)
		backordersMap[key] = totalRemaining
	}
	return backordersMap, nil
}

/**
 * @brief 全ての発注残を画面表示用のリスト形式で取得します。
 * [cite_start](WASABI: db/backorders.go [cite: 2453] より移植・TKR用に DBTX を使用)
 */
func GetAllBackordersList(dbtx DBTX) ([]model.Backorder, error) {
	const q = `
		SELECT
			id, order_date, yj_code, product_name, package_form, jan_pack_inner_qty, 
			yj_unit_name, order_quantity, remaining_quantity, wholesaler_code,
			yj_pack_unit_qty, jan_pack_unit_qty, jan_unit_code
		FROM backorders
		ORDER BY order_date, product_name, id
	`
	rows, err := dbtx.Query(q)
	if err != nil {
		return nil, fmt.Errorf("failed to query all backorders list: %w", err)
	}
	defer rows.Close()

	var backorders []model.Backorder
	for rows.Next() {
		var bo model.Backorder
		// wholesalerCode は sql.NullString 型
		var wholesalerCode sql.NullString

		if err := rows.Scan(
			&bo.ID, &bo.OrderDate, &bo.YjCode, &bo.ProductName, &bo.PackageForm, &bo.JanPackInnerQty,
			&bo.YjUnitName, &bo.OrderQuantity, &bo.RemainingQuantity, &wholesalerCode,
			&bo.YjPackUnitQty, &bo.JanPackUnitQty, &bo.JanUnitCode,
		); err != nil {
			return nil, err
		}
		// TKRのモデルは sql.NullString を持っていないため、TKRモデルに合わせて変換
		bo.WholesalerCode = wholesalerCode.String

		backorders = append(backorders, bo)
	}
	return backorders, nil
}

/**
 * @brief 指定されたIDの発注残レコードをトランザクション内で削除します。
 * [cite_start](WASABI: db/backorders.go [cite: 2454] より移植・TKR用に *sqlx.Tx を使用)
 */
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
