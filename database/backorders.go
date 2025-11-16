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
 * (WASABI: db/backorders.go より移植)
 */
func InsertBackordersInTx(tx *sqlx.Tx, backorders []model.Backorder) error {
	// ▼▼▼【ここを修正】jan_code を追加 ▼▼▼
	const q = `
		INSERT INTO backorders (
			order_date, jan_code, yj_code, product_name, package_form, jan_pack_inner_qty, 
			yj_unit_name, order_quantity, remaining_quantity, wholesaler_code,
			yj_pack_unit_qty, jan_pack_unit_qty, jan_unit_code
		) 
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	// ▲▲▲【修正ここまで】▲▲▲

	stmt, err := tx.Prepare(q)
	if err != nil {
		return fmt.Errorf("failed to prepare backorder insert statement: %w", err)
	}
	defer stmt.Close()

	for _, bo := range backorders {
		// ▼▼▼【ここを修正】bo.JanCode を追加 ▼▼▼
		_, err := stmt.Exec(
			bo.OrderDate, bo.JanCode, bo.YjCode, bo.ProductName, bo.PackageForm, bo.JanPackInnerQty,
			bo.YjUnitName, bo.OrderQuantity, bo.RemainingQuantity, bo.WholesalerCode,
			bo.YjPackUnitQty, bo.JanPackUnitQty, bo.JanUnitCode,
		)
		// ▲▲▲【修正ここまで】▲▲▲
		if err != nil {
			return fmt.Errorf("failed to execute backorder insert for yj %s: %w", bo.YjCode, err)
		}
	}
	return nil
}

/**
 * @brief 納品データに基づいて発注残を消し込みます（FIFO: 先入れ先出し）。
 * (WASABI: db/backorders.go より移植・TKR用に *sqlx.Tx を使用)
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
 * (WASABI: db/backorders.go より移植・TKR用に DBTX を使用)
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
 * (WASABI: db/backorders.go より移植・TKR用に DBTX を使用)
 */
func GetAllBackordersList(dbtx DBTX) ([]model.Backorder, error) {
	// ▼▼▼【ここを修正】jan_code を SELECT に追加 ▼▼▼
	const q = `
		SELECT
			id, order_date, jan_code, yj_code, product_name, package_form, jan_pack_inner_qty, 
			yj_unit_name, order_quantity, remaining_quantity, wholesaler_code,
			yj_pack_unit_qty, jan_pack_unit_qty, jan_unit_code
		FROM backorders
		ORDER BY order_date, wholesaler_code, product_name, id
	`
	// ▲▲▲【修正ここまで】▲▲▲

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
		// ▼▼▼【ここを修正】bo.JanCode 用の変数を追加 ▼▼▼
		var janCode sql.NullString
		// ▲▲▲【修正ここまで】▲▲▲

		// ▼▼▼【ここを修正】Scan に &janCode を追加 ▼▼▼
		if err := rows.Scan(
			&bo.ID, &bo.OrderDate, &janCode, &bo.YjCode, &bo.ProductName, &bo.PackageForm, &bo.JanPackInnerQty,
			&bo.YjUnitName, &bo.OrderQuantity, &bo.RemainingQuantity, &wholesalerCode,
			&bo.YjPackUnitQty, &bo.JanPackUnitQty, &bo.JanUnitCode,
		); err != nil {
			return nil, err
		}
		// ▲▲▲【修正ここまで】▲▲▲

		// TKRのモデルは sql.NullString を持っていないため、TKRモデルに合わせて変換
		bo.WholesalerCode = wholesalerCode.String
		// ▼▼▼【ここを修正】JanCode をモデルにセット ▼▼▼
		bo.JanCode = janCode.String
		// ▲▲▲【修正ここまで】▲▲▲

		backorders = append(backorders, bo)
	}
	return backorders, nil
}

/**
 * @brief 指定されたIDの発注残レコードをトランザクション内で削除します。
 * (WASABI: db/backorders.go より移植・TKR用に *sqlx.Tx を使用)
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

/**
 * @brief 指定された発注日時(YYYYMMDDHHMMSS)の発注残レコードをすべてトランザクション内で削除します。
 */
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
