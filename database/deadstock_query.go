// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\deadstock_query.go
package database

import (
	"fmt"
	"log"
	"time"
	"tkr/model"
	"tkr/units"

	"github.com/jmoiron/sqlx"
)

// GetDeadStockList は、指定された期間に処方(flag=3)されていない在庫品目（不動在庫）のリストを取得します。
func GetDeadStockList(db *sqlx.DB, startDate, endDate string, excludeZeroStock bool) ([]model.DeadStockItem, error) {

	// B: 期間内に処方(flag=3)された package_key のリストを作成
	const movedKeysQuery = `
		SELECT DISTINCT 
			T.yj_code || '|' || 
			COALESCE(T.package_form, '不明') || '|' || 
			PRINTF('%g', COALESCE(T.jan_pack_inner_qty, 0)) || '|' ||
			COALESCE(U.name, T.yj_unit_name, '不明') AS package_key
		FROM transaction_records AS T
		LEFT JOIN units AS U ON T.yj_unit_name = U.code
		WHERE T.flag = 3 
		  AND T.transaction_date BETWEEN ? AND ?
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
			MIN(P.jan_pack_inner_qty) as jan_pack_inner_qty
		FROM product_master AS P
		LEFT JOIN units AS U ON P.yj_unit_name = U.code
		WHERE P.yj_code != "" 
		GROUP BY package_key, P.yj_code
	`

	// メインクエリ
	// StockQuantityYj には「前回棚卸数(package_stock)」を取得する
	query := `
		SELECT 
			A.package_key, 
			A.yj_code, 
			COALESCE(PS.stock_quantity_yj, 0) AS stock_quantity_yj, -- 前回棚卸数
			A.kana_name,
			A.usage_classification,
			A.jan_pack_inner_qty
		FROM (
			` + allMasterKeysQuery + `
		) AS A
		LEFT JOIN (
			` + movedKeysQuery + `
		) AS B ON A.package_key = B.package_key
		LEFT JOIN package_stock AS PS ON A.package_key = PS.package_key
		WHERE 
			B.package_key IS NULL -- 期間内に動きがなかったもの
	`

	// SQL段階での除外判定には前回棚卸数を使用（参考程度）
	// 厳密な excludeZeroStock 判定は後続のGoロジックで現在在庫を計算してから行う
	query += `
		ORDER BY 
			CASE COALESCE(A.usage_classification, '他')
				WHEN '内' THEN 1
				WHEN '外' THEN 2
				WHEN '歯' THEN 3
				WHEN '注' THEN 4
				WHEN '機' THEN 5
				WHEN '他' THEN 6
				ELSE 7
			END,
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

	// 2. 詳細情報取得のための準備
	yjCodesMap := make(map[string]bool)
	for _, item := range items {
		if item.YjCode != "" {
			yjCodesMap[item.YjCode] = true
		}
	}
	var yjCodes []string
	for yj := range yjCodesMap {
		yjCodes = append(yjCodes, yj)
	}

	// 関連マスタの取得
	productCodes, err := GetProductCodesByYjCodes(db, yjCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to get product codes for dead stock details: %w", err)
	}
	mastersMap, err := GetProductMastersByCodesMap(db, productCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to get product masters map for dead stock details: %w", err)
	}
	var masters []*model.ProductMaster
	for _, m := range mastersMap {
		masters = append(masters, m)
	}

	// マスタを PackageKey ごとにグルーピング
	janCodesByPackageKey := make(map[string][]string)
	masterInfoByPackageKey := make(map[string]*model.ProductMaster)
	representativeJanByPackageKey := make(map[string]string)

	for _, m := range masters {
		key := GeneratePackageKey(m)
		janCodesByPackageKey[key] = append(janCodesByPackageKey[key], m.ProductCode)

		// 表示用・計算用の代表マスタを選定
		if current, exists := masterInfoByPackageKey[key]; !exists {
			masterInfoByPackageKey[key] = m
			representativeJanByPackageKey[key] = m.ProductCode
		} else if current.Origin != "JCSHMS" && m.Origin == "JCSHMS" {
			masterInfoByPackageKey[key] = m
			representativeJanByPackageKey[key] = m.ProductCode
		}
	}

	today := time.Now().Format("20060102")
	var resultItems []model.DeadStockItem

	// 3. 各項目の詳細設定と現在在庫計算
	for i := range items {
		item := &items[i]

		// 代表JANを使って「現在の正確な理論在庫」を計算
		repJan, hasRep := representativeJanByPackageKey[item.PackageKey]
		if hasRep {
			currentStock, err := CalculateStockOnDate(db, repJan, today)
			if err != nil {
				log.Printf("WARN: Failed to calculate stock for %s: %v", repJan, err)
				item.CurrentStockYj = item.StockQuantityYj // エラー時は前回棚卸数を入れる
			} else {
				item.CurrentStockYj = currentStock
			}
		} else {
			item.CurrentStockYj = 0
		}

		// 「在庫0を除外」オプションの適用（現在在庫で判定）
		if excludeZeroStock && item.CurrentStockYj <= 0 {
			continue
		}

		// 代表品名などの設定
		if master, ok := masterInfoByPackageKey[item.PackageKey]; ok {
			item.ProductName = master.ProductName
			item.PackageSpec = fmt.Sprintf("%s %g%s", master.PackageForm, master.YjPackUnitQty, units.ResolveName(master.YjUnitName))
			if master.JanPackInnerQty > 0 {
				item.PackageSpec += fmt.Sprintf(" (%g%s×%g%s)",
					master.JanPackInnerQty, units.ResolveName(master.YjUnitName), master.JanPackUnitQty, units.ResolveName(fmt.Sprintf("%d", master.JanUnitCode)))
			}
		}

		// ロット詳細（前回棚卸時の明細）を取得
		targetJanCodes := janCodesByPackageKey[item.PackageKey]
		if len(targetJanCodes) > 0 {
			q := `
				SELECT 
					T.jan_code, 
					COALESCE(P.gs1_code, '') AS gs1_code, 
					T.package_spec, 
					T.expiry_date, 
					T.lot_number, 
					T.jan_quantity, 
					T.jan_unit_name
				FROM transaction_records AS T
				LEFT JOIN product_master AS P ON T.jan_code = P.product_code
				WHERE T.jan_code IN (?) 
				  AND T.flag = 0 
				  AND T.transaction_date = (
					  SELECT MAX(last_inventory_date) 
					  FROM package_stock 
					  WHERE package_key = ?
				)
				ORDER BY T.expiry_date, T.lot_number
			`
			query, args, err := sqlx.In(q, targetJanCodes, item.PackageKey)
			if err != nil {
				log.Printf("WARN: Failed to build IN query for dead stock details: %v", err)
				item.LotDetails = []model.LotDetail{}
			} else {
				query = db.Rebind(query)
				err = db.Select(&item.LotDetails, query, args...)
				if err != nil {
					item.LotDetails = []model.LotDetail{}
				}
			}
		} else {
			item.LotDetails = []model.LotDetail{}
		}

		resultItems = append(resultItems, *item)
	}

	return resultItems, nil
}
