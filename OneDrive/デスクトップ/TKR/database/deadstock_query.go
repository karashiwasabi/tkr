// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\deadstock_query.go
package database

import (
	"fmt"
	"log" // log パッケージをインポート
	"tkr/model"
	"tkr/units" // TKRのunitsパッケージをインポート

	"github.com/jmoiron/sqlx"
)

// ▼▼▼【ここを修正】シグネチャに excludeZeroStock を追加 ▼▼▼
// GetDeadStockList は、指定された期間に処方(flag=3)されていない在庫品目（不動在庫）のリストを取得します。
func GetDeadStockList(db *sqlx.DB, startDate, endDate string, excludeZeroStock bool) ([]model.DeadStockItem, error) {
	// ▲▲▲【修正ここまで】▲▲▲

	// B: 期間内に処方(flag=3)された package_key のリストを作成
	const movedKeysQuery = `
		SELECT DISTINCT 
			T.yj_code ||
'|' || 
			COALESCE(T.package_form, '不明') || '|' || 
			PRINTF('%g', COALESCE(T.jan_pack_inner_qty, 0)) || '|' ||
COALESCE(U.name, T.yj_unit_name, '不明') AS package_key
		FROM transaction_records AS T
		LEFT JOIN units AS U ON T.yj_unit_name = U.code
		WHERE T.flag = 3 
		  AND T.transaction_date BETWEEN ?
AND ?
	`

	// A: product_master から構築した全 package_key
	const allMasterKeysQuery = `
		SELECT
			P.yj_code || '|' || 
			COALESCE(P.package_form, '不明') || '|' ||
PRINTF('%g', COALESCE(P.jan_pack_inner_qty, 0)) || '|' || 
			COALESCE(U.name, P.yj_unit_name, '不明') AS package_key,
			P.yj_code,
			MIN(P.kana_name) as kana_name, 
			MIN(P.usage_classification) as usage_classification,
			-- ▼▼▼【ここに追加】代表の内包装数量を取得 ▼▼▼
			MIN(P.jan_pack_inner_qty) as jan_pack_inner_qty
			-- ▲▲▲【追加ここまで】▲▲▲
		FROM product_master AS P
		LEFT JOIN units AS U ON P.yj_unit_name = U.code
		WHERE P.yj_code != "" 
		GROUP BY package_key, P.yj_code
	`

	// C: 全期間の理論在庫（通算在庫）を集計
	const theoreticalStockQuery = `
		SELECT 
			T.yj_code ||
'|' || 
			COALESCE(T.package_form, '不明') || '|' || 
			PRINTF('%g', COALESCE(T.jan_pack_inner_qty, 0)) || '|' ||
COALESCE(U.name, T.yj_unit_name, '不明') AS package_key,
			SUM(CASE 
				WHEN T.flag = 1 THEN T.yj_quantity  -- 納品 (+)
				WHEN T.flag = 3 THEN -T.yj_quantity -- 処方 (-)
				WHEN T.flag = 2 THEN -T.yj_quantity -- 返品 (-)
				ELSE 0 
			END) AS theoretical_stock
		FROM transaction_records AS T
		LEFT JOIN units AS U ON T.yj_unit_name = U.code
		WHERE T.flag IN (1, 2, 3)
		GROUP BY package_key
	`

	// A(product_master) を基準にし、B(処方)、PS(棚卸在庫)、C(理論在庫) をJOINする
	query := `
		SELECT 
			A.package_key, 
			A.yj_code, 
			COALESCE(PS.stock_quantity_yj, C.theoretical_stock, 0) AS stock_quantity_yj,
			A.kana_name,             -- ソート用
			A.usage_classification,   -- ソート用
			-- ▼▼▼【ここに追加】内包装数量 ▼▼▼
			A.jan_pack_inner_qty
			-- ▲▲▲【追加ここまで】▲▲▲
		FROM (
			` + allMasterKeysQuery + `
		) AS A
		LEFT JOIN (
			` + movedKeysQuery + `
		) AS B ON A.package_key 
= B.package_key
		LEFT JOIN package_stock AS PS ON A.package_key = PS.package_key -- D: 棚卸在庫
		LEFT JOIN (
			` + theoreticalStockQuery + `
		) AS C ON A.package_key = C.package_key -- C: 理論在庫
		WHERE 
			B.package_key IS NULL -- 期間内に動きがなかったもの
	`
	// ▼▼▼【ここに追加】在庫ゼロ除外オプション ▼▼▼
	if excludeZeroStock {
		query += ` AND COALESCE(PS.stock_quantity_yj, C.theoretical_stock, 0) > 0`
	}
	// ▲▲▲【追加ここまで】▲▲▲

	query += `
		ORDER BY 
			-- ▼▼▼【ここから修正】「内外歯注機他」の順序に変更 ▼▼▼
			CASE COALESCE(A.usage_classification, '他')
				WHEN '内' THEN 1
				WHEN '外' THEN 2
				WHEN '歯' THEN 3
				WHEN '注' THEN 4
				WHEN '機' THEN 5
				WHEN '他' THEN 6
				ELSE 7
			END,
			-- ▲▲▲【修正ここまで】▲▲▲
			A.kana_name,
			A.package_key
	`

	var items []model.DeadStockItem
	if err := db.Select(&items, query, startDate, endDate); err != nil {
		return nil, fmt.Errorf("failed to select dead stock list (A-B): %w", err)
	}

	if len(items) == 0 {
		return []model.DeadStockItem{}, nil
	}

	// 各不動在庫品目の詳細（品名、ロット）を取得
	for i := range items {
		item := &items[i]

		// ▼▼▼【ここに追加】YJ在庫 -> JAN在庫 への換算 ▼▼▼
		if item.JanPackInnerQty > 0 {
			item.StockQuantityJan = item.StockQuantityYj / item.JanPackInnerQty
		} else {
			item.StockQuantityJan = 0 // 内包装数量がなければ 0
		}
		// ▲▲▲【追加ここまで】▲▲▲

		// 代表品名と包装仕様を取得
		var masterInfo struct {
			ProductName   string  `db:"product_name"`
			PackageForm   string  `db:"package_form"`
			YjUnitName    string  `db:"yj_unit_name"`
			YjPackUnitQty float64 `db:"yj_pack_unit_qty"`
		}
		err := db.Get(&masterInfo, `
			SELECT product_name, package_form, yj_unit_name, yj_pack_unit_qty 
			FROM product_master 
			WHERE yj_code = ? 
			ORDER BY origin = 'JCSHMS' DESC, product_code ASC LIMIT 1`,
			item.YjCode)

		if err == nil {
			item.ProductName = masterInfo.ProductName
			item.PackageSpec = fmt.Sprintf("%s %g%s",
				masterInfo.PackageForm,
				masterInfo.YjPackUnitQty,
				units.ResolveName(masterInfo.YjUnitName)) // ※表示用の仕様は units.ResolveName を使う
		}

		// ▼▼▼【ここを修正】YJ在庫 -> JAN在庫 で判定 ▼▼▼
		// 在庫が 0 より大きい品目のみロット・期限を取得
		if item.StockQuantityJan > 0 {
			// ▲▲▲【修正ここまで】▲▲▲
			err = db.Select(&item.LotDetails, `
				SELECT 
					T.jan_code, 
					COALESCE(P.gs1_code, '') AS 
gs1_code, 
					T.package_spec, 
					T.expiry_date, 
					T.lot_number, 
					T.jan_quantity, 
					T.jan_unit_name
				FROM transaction_records AS T
				LEFT JOIN product_master AS P ON T.jan_code = P.product_code
				WHERE T.yj_code = ?
AND T.flag = 0 
				  AND T.transaction_date = (
					  SELECT MAX(last_inventory_date) 
					  FROM package_stock 
					  WHERE yj_code = ?
				  )
				ORDER BY T.expiry_date, T.lot_number
			`, item.YjCode, item.YjCode)

			if err != nil {
				// 明細取得に失敗してもエラーにせず、リストは返す
				log.Printf("WARN: Failed to get lot details for dead stock YJ %s: %v", item.YjCode, err)
				item.LotDetails = []model.LotDetail{}
			}
		} else {
			// 在庫が0の品目はロット検索をスキップ
			item.LotDetails = []model.LotDetail{}
		}
	}

	return items, nil
}
