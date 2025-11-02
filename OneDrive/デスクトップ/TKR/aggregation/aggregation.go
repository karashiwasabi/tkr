// C:\Users\wasab\OneDrive\デスクトップ\TKR\aggregation\aggregation.go
package aggregation

import (
	"fmt"
	"sort"
	"strings"
	"tkr/database" // ▼▼▼【ここに追加】▼▼▼
	"tkr/model"
	"tkr/units"

	"github.com/jmoiron/sqlx"
)

// SignedYjQty はフラグに基づいて符号付きのYJ数量を返します (TKR版)。
func signedYjQty(flag int, yjQty float64) float64 {
	switch flag {
	case 1: // 納品
		return yjQty
	case 3: // 処方
		return -yjQty
	default: // 棚卸(0), 返品(2), その他
		return 0
	}
}

// GetStockLedger は在庫元帳レポートを生成します。
// (WASABI: db/aggregation.go  より移植・TKR用に修正)
func GetStockLedger(conn *sqlx.DB, filters model.AggregationFilters) ([]model.StockLedgerYJGroup, error) {

	// TKRには発注残と予製がないため、関連マップの取得は不要

	// ステップ1: フィルターに合致する製品マスターを取得し、対象YJコードを特定
	// ▼▼▼【修正】Gを大文字に ▼▼▼
	mastersByYjCode, yjCodes, err := GetFilteredMastersAndYjCodes(conn, filters)
	// ▲▲▲【修正ここまで】▲▲▲
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered masters: %w", err)
	}
	if len(yjCodes) == 0 {
		return []model.StockLedgerYJGroup{}, nil
	}

	// ステップ2 & 3: 関連する全ての製品コードと取引履歴を取得
	allProductCodes, err := getAllProductCodesForYjCodes(conn, yjCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to get all product codes for yj codes: %w", err)
	}

	allMasters, err := getMastersByProductCodes(conn, allProductCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to get all masters for product codes: %w", err)
	}

	transactionsByProductCode, err := getTransactionsByProductCodes(conn, allProductCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to get all transactions for product codes: %w", err)
	}

	// ステップ4: YJコードごとに集計処理
	var result []model.StockLedgerYJGroup
	for _, yjCode := range yjCodes {
		mastersInYjGroup, ok := mastersByYjCode[yjCode]
		if !ok || len(mastersInYjGroup) == 0 {
			continue
		}

		// YJグループの代表情報を設定
		var representativeProductName, representativeYjUnitName string
		representativeProductName = mastersInYjGroup[0].ProductName
		representativeYjUnitName = mastersInYjGroup[0].YjUnitName
		for _, m := range mastersInYjGroup {
			if m.Origin == "JCSHMS" {
				representativeProductName = m.ProductName
				representativeYjUnitName = m.YjUnitName
				break
			}
		}
		yjGroup := model.StockLedgerYJGroup{
			YjCode:      yjCode,
			ProductName: representativeProductName,
			YjUnitName:  units.ResolveName(representativeYjUnitName),
		}

		// YJグループに属する全ての取引を集める
		var allTxsForYjGroup []*model.TransactionRecord
		for _, m := range allMasters {
			if m.YjCode == yjCode {
				// TKRの GetTransactionsByProductCodes は *sqlx.DB を受け取るため、
				// 返り値は map[string][]model.TransactionRecord であると仮定
				// (実際は database/transaction_records_query.go でそのように定義した)
				txs, ok := transactionsByProductCode[m.ProductCode]
				if ok {
					for i := range txs {
						allTxsForYjGroup = append(allTxsForYjGroup, &txs[i])
					}
				}
			}
		}
		sort.Slice(allTxsForYjGroup, func(i, j int) bool {
			if allTxsForYjGroup[i].TransactionDate != allTxsForYjGroup[j].TransactionDate {
				return allTxsForYjGroup[i].TransactionDate < allTxsForYjGroup[j].TransactionDate
			}
			return allTxsForYjGroup[i].ID < allTxsForYjGroup[j].ID
		})

		// YJグループ全体の最新棚卸日を特定
		latestInventoryDateForGroup := ""
		for _, t := range allTxsForYjGroup {
			if t.Flag == 0 && t.TransactionDate > latestInventoryDateForGroup {
				latestInventoryDateForGroup = t.TransactionDate
			}
		}

		// 包装キーでマスターをグループ化
		mastersByPackageKey := make(map[string][]*model.ProductMaster)
		for _, m := range mastersInYjGroup {
			key := fmt.Sprintf("%s|%s|%g|%s", m.YjCode, m.PackageForm, m.JanPackInnerQty, m.YjUnitName)
			mastersByPackageKey[key] = append(mastersByPackageKey[key], m)
		}

		var allPackageLedgers []model.StockLedgerPackageGroup
		for key, mastersInPackageGroup := range mastersByPackageKey {
			var startingBalance float64
			// ステップ5: 期間前在庫を計算
			if latestInventoryDateForGroup != "" {
				baseStockOnDate := 0.0
				for _, m := range mastersInPackageGroup {
					txs, ok := transactionsByProductCode[m.ProductCode]
					if !ok {
						continue
					}
					for _, t := range txs {
						if t.Flag == 0 && t.TransactionDate == latestInventoryDateForGroup {
							baseStockOnDate += t.YjQuantity
						}
					}
				}

				netChangeAfterInv := 0.0
				for _, m := range mastersInPackageGroup {
					txs, ok := transactionsByProductCode[m.ProductCode]
					if !ok {
						continue
					}
					for _, t := range txs {
						if t.TransactionDate > latestInventoryDateForGroup && t.TransactionDate < filters.StartDate {
							netChangeAfterInv += signedYjQty(t.Flag, t.YjQuantity)
						}
					}
				}
				startingBalance = baseStockOnDate + netChangeAfterInv
			} else {
				// 棚卸履歴が全くない場合
				netChangeBeforePeriod := 0.0
				for _, m := range mastersInPackageGroup {
					txs, ok := transactionsByProductCode[m.ProductCode]
					if !ok {
						continue
					}
					for _, t := range txs {
						if t.TransactionDate < filters.StartDate {
							netChangeBeforePeriod += signedYjQty(t.Flag, t.YjQuantity)
						}
					}
				}
				startingBalance = netChangeBeforePeriod
			}

			// ステップ6: 期間内変動を計算
			var transactionsInPeriod []model.LedgerTransaction
			var netChange float64
			runningBalance := startingBalance

			var txsForPackageInPeriod []*model.TransactionRecord
			for _, m := range mastersInPackageGroup {
				txs, ok := transactionsByProductCode[m.ProductCode]
				if !ok {
					continue
				}
				for _, t := range txs {
					if t.TransactionDate >= filters.StartDate && t.TransactionDate <= filters.EndDate {
						// ポインタを取得する必要がある
						txCopy := t
						txsForPackageInPeriod = append(txsForPackageInPeriod, &txCopy)
					}
				}
			}
			sort.Slice(txsForPackageInPeriod, func(i, j int) bool {
				if txsForPackageInPeriod[i].TransactionDate != txsForPackageInPeriod[j].TransactionDate {
					return txsForPackageInPeriod[i].TransactionDate < txsForPackageInPeriod[j].TransactionDate
				}
				return txsForPackageInPeriod[i].ID < txsForPackageInPeriod[j].ID
			})

			periodInventorySums := make(map[string]float64)
			for _, t := range txsForPackageInPeriod {
				if t.Flag == 0 {
					periodInventorySums[t.TransactionDate] += t.YjQuantity
				}
			}

			lastProcessedDate := ""
			for _, t := range txsForPackageInPeriod {
				if t.TransactionDate != lastProcessedDate && lastProcessedDate != "" {
					if inventorySum, ok := periodInventorySums[lastProcessedDate]; ok {
						runningBalance = inventorySum
					}
				}

				if t.Flag == 0 {
					if inventorySum, ok := periodInventorySums[t.TransactionDate]; ok {
						runningBalance = inventorySum
					}
				} else {
					runningBalance += signedYjQty(t.Flag, t.YjQuantity)
				}
				transactionsInPeriod = append(transactionsInPeriod, model.LedgerTransaction{TransactionRecord: *t, RunningBalance: runningBalance})
				netChange += signedYjQty(t.Flag, t.YjQuantity)

				lastProcessedDate = t.TransactionDate
			}

			// ステップ7 (TKRでは発注点計算は不要)
			pkg := model.StockLedgerPackageGroup{
				PackageKey:      key,
				StartingBalance: startingBalance,
				EndingBalance:   runningBalance,
				Transactions:    transactionsInPeriod,
				NetChange:       netChange,
				Masters:         mastersInPackageGroup,
			}
			allPackageLedgers = append(allPackageLedgers, pkg)
		}

		// ステップ8: YJグループ全体で集計
		if len(allPackageLedgers) > 0 {
			var yjTotalEnding, yjTotalNetChange, yjTotalStarting float64
			for _, pkg := range allPackageLedgers {
				if start, ok := pkg.StartingBalance.(float64); ok {
					yjTotalStarting += start
				}
				if end, ok := pkg.EndingBalance.(float64); ok {
					yjTotalEnding += end
				}
				yjTotalNetChange += pkg.NetChange
			}
			yjGroup.StartingBalance = yjTotalStarting
			yjGroup.EndingBalance = yjTotalEnding
			yjGroup.NetChange = yjTotalNetChange
			yjGroup.PackageLedgers = allPackageLedgers
			result = append(result, yjGroup)
		}
	}

	// ステップ9: ソートと最終フィルタリング (TKRでは movementOnly は不要)
	sort.Slice(result, func(i, j int) bool {
		prio := map[string]int{"内": 1, "外": 2, "注": 3, "他": 4} // TKRの区分
		masterI := mastersByYjCode[result[i].YjCode][0]
		masterJ := mastersByYjCode[result[j].YjCode][0]
		prioI, okI := prio[strings.TrimSpace(masterI.UsageClassification)]
		if !okI {
			prioI = 5
		}
		prioJ, okJ := prio[strings.TrimSpace(masterJ.UsageClassification)]
		if !okJ {
			prioJ = 5
		}
		if prioI != prioJ {
			return prioI < prioJ
		}
		return masterI.KanaName < masterJ.KanaName
	})

	return result, nil
}

// getFilteredMastersAndYjCodes は フィルタ条件に合うマスターをYJコードごとにグループ化して返します。
// ▼▼▼【修正】Gを大文字に (エクスポート) ▼▼▼
func GetFilteredMastersAndYjCodes(conn *sqlx.DB, filters model.AggregationFilters) (map[string][]*model.ProductMaster, []string, error) {
	// ▲▲▲【修正ここまで】▲▲▲
	query := `SELECT * FROM product_master p WHERE 1=1 `
	var args []interface{}
	if filters.YjCode != "" {
		query += " AND p.yj_code = ? "
		args = append(args, filters.YjCode)
	}
	if filters.KanaName != "" {
		query += " AND (p.kana_name LIKE ? OR p.product_name LIKE ?) "
		args = append(args, "%"+filters.KanaName+"%", "%"+filters.KanaName+"%")
	}
	if filters.DosageForm != "" && filters.DosageForm != "all" {
		query += " AND p.usage_classification = ? "
		args = append(args, filters.DosageForm)
	}
	if filters.ShelfNumber != "" {
		query += " AND p.shelf_number LIKE ? "
		args = append(args, "%"+filters.ShelfNumber+"%")
	}
	// TKRには薬剤種別(DrugTypes)フィルタはないため省略

	rows, err := conn.Queryx(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	mastersByYjCode := make(map[string][]*model.ProductMaster)
	yjCodeMap := make(map[string]bool)
	for rows.Next() {
		var m model.ProductMaster
		if err := rows.StructScan(&m); err != nil {
			return nil, nil, err
		}
		if m.YjCode != "" {
			mastersByYjCode[m.YjCode] = append(mastersByYjCode[m.YjCode], &m)
			yjCodeMap[m.YjCode] = true
		}
	}
	if len(yjCodeMap) == 0 {
		return nil, []string{}, nil
	}
	var yjCodes []string
	for yj := range yjCodeMap {
		yjCodes = append(yjCodes, yj)
	}
	return mastersByYjCode, yjCodes, nil
}

// getAllProductCodesForYjCodes は YJコード群に関連する全JANコードを取得します。
func getAllProductCodesForYjCodes(conn *sqlx.DB, yjCodes []string) ([]string, error) {
	if len(yjCodes) == 0 {
		return []string{}, nil
	}

	query, args, err := sqlx.In(`
		SELECT DISTINCT product_code FROM product_master WHERE yj_code IN (?)
		UNION
		SELECT DISTINCT jan_code FROM transaction_records WHERE yj_code IN (?)`, yjCodes, yjCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to create IN query for all product codes: %w", err)
	}
	query = conn.Rebind(query)

	var codes []string
	if err := conn.Select(&codes, query, args...); err != nil {
		return nil, err
	}

	// 空文字を除外
	var validCodes []string
	for _, code := range codes {
		if code != "" {
			validCodes = append(validCodes, code)
		}
	}
	return validCodes, nil
}

// getMastersByProductCodes は JANコード群から全マスターをマップで取得します。
func getMastersByProductCodes(conn *sqlx.DB, productCodes []string) (map[string]*model.ProductMaster, error) {
	mastersMap := make(map[string]*model.ProductMaster)
	if len(productCodes) == 0 {
		return mastersMap, nil
	}

	const batchSize = 500
	for i := 0; i < len(productCodes); i += batchSize {
		end := i + batchSize
		if end > len(productCodes) {
			end = len(productCodes)
		}
		batch := productCodes[i:end]

		if len(batch) > 0 {
			query, args, err := sqlx.In("SELECT * FROM product_master WHERE product_code IN (?)", batch)
			if err != nil {
				return nil, err
			}
			query = conn.Rebind(query)
			rows, err := conn.Queryx(query, args...)
			if err != nil {
				return nil, err
			}
			for rows.Next() {
				var m model.ProductMaster
				if err := rows.StructScan(&m); err != nil {
					rows.Close()
					return nil, err
				}
				mastersMap[m.ProductCode] = &m
			}
			rows.Close()
		}
	}
	return mastersMap, nil
}

// getTransactionsByProductCodes は JANコード群から全取引履歴をマップで取得します。
func getTransactionsByProductCodes(conn *sqlx.DB, productCodes []string) (map[string][]model.TransactionRecord, error) {
	transactionsMap := make(map[string][]model.TransactionRecord)
	if len(productCodes) == 0 {
		return transactionsMap, nil
	}

	const batchSize = 500
	for i := 0; i < len(productCodes); i += batchSize {
		end := i + batchSize
		if end > len(productCodes) {
			end = len(productCodes)
		}
		batch := productCodes[i:end]

		if len(batch) > 0 {
			// ▼▼▼【修正】database.TransactionColumns を使用 ▼▼▼
			query, args, err := sqlx.In("SELECT "+database.TransactionColumns+" FROM transaction_records WHERE jan_code IN (?) ORDER BY transaction_date, id", batch)
			if err != nil {
				return nil, err
			}
			query = conn.Rebind(query)
			rows, err := conn.Query(query, args...)
			if err != nil {
				return nil, err
			}
			for rows.Next() {
				// ▼▼▼【修正】database.ScanTransactionRecord を使用 ▼▼▼
				t, err := database.ScanTransactionRecord(rows)
				if err != nil {
					rows.Close()
					return nil, err
				}
				transactionsMap[t.JanCode] = append(transactionsMap[t.JanCode], *t)
			}
			rows.Close()
		}
	}
	return transactionsMap, nil
}
