// C:\Users\wasab\OneDrive\デスクトップ\TKR\aggregation\aggregation.go
package aggregation

import (
	"fmt"
	"sort"
	"strings"
	"tkr/database"
	"tkr/model"
	"tkr/units"

	"github.com/jmoiron/sqlx"
)

// ... (signedYjQty, GetStockLedger 関数は変更なし) ...
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
func GetStockLedger(conn *sqlx.DB, filters model.AggregationFilters) ([]model.StockLedgerYJGroup, error) {
	mastersByYjCode, yjCodes, err := GetFilteredMastersAndYjCodes(conn, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered masters: %w", err)
	}
	if len(yjCodes) == 0 {
		return []model.StockLedgerYJGroup{}, nil
	}
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
	var result []model.StockLedgerYJGroup
	for _, yjCode := range yjCodes {
		mastersInYjGroup, ok := mastersByYjCode[yjCode]
		if !ok || len(mastersInYjGroup) == 0 {
			continue
		}
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
		var allTxsForYjGroup []*model.TransactionRecord
		for _, m := range allMasters {
			if m.YjCode == yjCode {
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
		latestInventoryDateForGroup := ""
		for _, t := range allTxsForYjGroup {
			if t.Flag == 0 && t.TransactionDate > latestInventoryDateForGroup {
				latestInventoryDateForGroup = t.TransactionDate
			}
		}
		mastersByPackageKey := make(map[string][]*model.ProductMaster)
		for _, m := range mastersInYjGroup {
			key := fmt.Sprintf("%s|%s|%g|%s", m.YjCode, m.PackageForm, m.JanPackInnerQty, m.YjUnitName)
			mastersByPackageKey[key] = append(mastersByPackageKey[key], m)
		}
		var allPackageLedgers []model.StockLedgerPackageGroup
		for key, mastersInPackageGroup := range mastersByPackageKey {
			var startingBalance float64
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
	sort.Slice(result, func(i, j int) bool {
		prio := map[string]int{"内": 1, "外": 2, "注": 3, "他": 4}
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

// ▼▼▼【ここから修正】GetFilteredMastersAndYjCodes に 'generalName' フィルタを追加 ▼▼▼
func GetFilteredMastersAndYjCodes(conn *sqlx.DB, filters model.AggregationFilters) (map[string][]*model.ProductMaster, []string, error) {
	query := `SELECT * FROM product_master p WHERE 1=1 `
	var args []interface{}
	if filters.YjCode != "" {
		query += " AND p.yj_code = ? "
		args = append(args, filters.YjCode)
	}
	if filters.KanaName != "" {
		query += " AND (p.kana_name LIKE ? OR p.product_name LIKE ?) "
		args = append(args, filters.KanaName+"%", "%"+filters.KanaName+"%") // カナ名は前方一致、製品名は部分一致
	}
	// ★一般名フィルタを追加
	if filters.GenericName != "" {
		query += " AND p.generic_name LIKE ? "
		args = append(args, "%"+filters.GenericName+"%")
	}
	if filters.DosageForm != "" && filters.DosageForm != "all" {
		query += " AND p.usage_classification = ? "
		args = append(args, filters.DosageForm)
	}
	if filters.ShelfNumber != "" {
		query += " AND p.shelf_number LIKE ? "
		args = append(args, "%"+filters.ShelfNumber+"%")
	}

	// ★ソート順を kana_name に変更
	query += " ORDER BY p.kana_name "

	rows, err := conn.Queryx(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	mastersByYjCode := make(map[string][]*model.ProductMaster)
	yjCodeMap := make(map[string]bool)
	var yjCodes []string // ★ソート順を維持するためにマップではなくスライスを使う

	for rows.Next() {
		var m model.ProductMaster
		if err := rows.StructScan(&m); err != nil {
			return nil, nil, err
		}
		if m.YjCode != "" {
			mastersByYjCode[m.YjCode] = append(mastersByYjCode[m.YjCode], &m)
			if !yjCodeMap[m.YjCode] {
				yjCodeMap[m.YjCode] = true
				yjCodes = append(yjCodes, m.YjCode) // 検索結果のソート順でYJコードを追加
			}
		}
	}

	return mastersByYjCode, yjCodes, nil
}

// ▲▲▲【修正ここまで】▲▲▲

// ... (getAllProductCodesForYjCodes, getMastersByProductCodes, getTransactionsByProductCodes 関数は変更なし) ...
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
	var validCodes []string
	for _, code := range codes {
		if code != "" {
			validCodes = append(validCodes, code)
		}
	}
	return validCodes, nil
}
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
