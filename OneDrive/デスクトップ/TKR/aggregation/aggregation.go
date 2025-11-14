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

func signedYjQty(flag int, yjQty float64) float64 {
	switch flag {
	// ▼▼▼【ここから修正】TKRの入出庫フラグ (11, 12) も考慮 ▼▼▼
	case 1: // 納品
		return yjQty
	case 11: // 入庫
		return yjQty
	case 3: // 処方
		return -yjQty
	case 2: // 返品
		return -yjQty
	case 12: // 出庫
		return -yjQty
	// ▲▲▲【修正ここまで】▲▲▲
	default: // 棚卸(0), その他
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
	transactionsByProductCode, err := getTransactionsByProductCodes(conn, allProductCodes)
	if err !=
		nil {
		return nil, fmt.Errorf("failed to get all transactions for product codes: %w", err)
	}

	precompTotals, err := database.GetPreCompoundingTotals(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to get pre-compounding totals for aggregation: %w", err)
	}

	// ▼▼▼【ここから追加】発注残マップの取得 (WASABI: db/aggregation.go より) ▼▼▼
	backordersMap, err := database.GetAllBackordersMap(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to get backorders for aggregation: %w", err)
	}
	// ▲▲▲【追加ここまで】▲▲▲

	packageStockMap, err := database.GetPackageStockByYjCode(conn, filters.YjCode) //
	if err != nil {
		return nil, fmt.Errorf("failed to get package stock map: %w", err)
	}

	var result []model.StockLedgerYJGroup
	for _, yjCode := range yjCodes {
		mastersInYjGroup, ok := mastersByYjCode[yjCode]
		if !ok ||
			len(mastersInYjGroup) == 0 { //
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

		mastersByPackageKey := make(map[string][]*model.ProductMaster)
		for _, m := range mastersInYjGroup { //
			key := fmt.Sprintf("%s|%s|%g|%s", m.YjCode, m.PackageForm, m.JanPackInnerQty, units.ResolveName(m.YjUnitName)) //
			mastersByPackageKey[key] = append(mastersByPackageKey[key], m)
		}
		var allPackageLedgers []model.StockLedgerPackageGroup
		for key, mastersInPackageGroup := range mastersByPackageKey {

			var startingBalance float64
			var stockStartDate string

			if stockInfo, ok := packageStockMap[key]; ok { //
				startingBalance = stockInfo.StockQuantityYj  //
				stockStartDate = stockInfo.LastInventoryDate //
			} else {
				startingBalance = 0.0       //
				stockStartDate = "00000000" //
			}

			netChangeAfterStockDate := 0.0
			for _, m := range mastersInPackageGroup {
				txs, ok := transactionsByProductCode[m.ProductCode]
				if !ok {
					continue
				}
				for _, t := range txs {
					// ▼▼▼【ここを修正】filters.EndDate の制限を削除し、未来日も集計対象に含める ▼▼▼
					if t.TransactionDate > stockStartDate { //
						netChangeAfterStockDate += signedYjQty(t.Flag, t.YjQuantity) //
					}
					// ▲▲▲【修正ここまで】▲▲▲
				}
			}

			currentTheoreticalStock := startingBalance + netChangeAfterStockDate //

			netChangeInPeriod := 0.0
			var txsForPackageInPeriod []*model.TransactionRecord
			for _, m := range mastersInPackageGroup { //
				txs, ok := transactionsByProductCode[m.ProductCode]
				if !ok {
					continue
				}
				for _, t := range txs {
					if t.TransactionDate >= filters.StartDate && t.TransactionDate <= filters.EndDate {
						netChangeInPeriod += signedYjQty(t.Flag, t.YjQuantity) //
						txCopy := t
						txsForPackageInPeriod = append(txsForPackageInPeriod, &txCopy)
						//
					}
				}
			}

			periodStartingBalance := currentTheoreticalStock - netChangeInPeriod //

			var maxUsage float64

			var transactionsInPeriod []model.LedgerTransaction
			runningBalance := periodStartingBalance //

			sort.Slice(txsForPackageInPeriod, func(i, j int) bool { //
				if txsForPackageInPeriod[i].TransactionDate != txsForPackageInPeriod[j].TransactionDate {
					return txsForPackageInPeriod[i].TransactionDate < txsForPackageInPeriod[j].TransactionDate
				}
				return txsForPackageInPeriod[i].ID < txsForPackageInPeriod[j].ID
			})

			periodInventorySums := make(map[string]float64) //
			for _, t := range txsForPackageInPeriod {
				if t.Flag == 0 {
					periodInventorySums[t.TransactionDate] += t.YjQuantity //
				}
				// ▼▼▼【ここを修正】TKRの処方フラグ(3)のみで最大使用量を計算 ▼▼▼
				if t.Flag == 3 && t.YjQuantity > maxUsage {
					maxUsage = t.YjQuantity
				}
				// ▲▲▲【修正ここまで】▲▲▲
			}

			lastProcessedDate := ""
			for _, t := range txsForPackageInPeriod { //
				if t.TransactionDate != lastProcessedDate && lastProcessedDate != "" { //
					if inventorySum, ok := periodInventorySums[lastProcessedDate]; ok { //
						runningBalance = inventorySum //
					}
				}

				if t.Flag == 0 {
					// 棚卸データ(flag=0)は runningBalance に影響を与えない (次のループの開始時に適用される)
				} else {
					runningBalance += signedYjQty(t.Flag, t.YjQuantity) //
				}

				transactionsInPeriod = append(transactionsInPeriod, model.LedgerTransaction{TransactionRecord: *t, RunningBalance: runningBalance}) //
				lastProcessedDate = t.TransactionDate
				//
			}

			if inventorySum, ok := periodInventorySums[lastProcessedDate]; ok { //
				runningBalance = inventorySum //
			}

			// ▼▼▼【ここから修正】発注残(backorderQty)を考慮 (WASABI: db/aggregation.go より) ▼▼▼
			backorderQty := backordersMap[key] //
			effectiveEndingBalance := runningBalance + backorderQty
			// ▲▲▲【修正ここまで】▲▲▲

			pkg := model.StockLedgerPackageGroup{
				PackageKey:             key,
				StartingBalance:        periodStartingBalance, //
				EndingBalance:          runningBalance,        //
				Transactions:           transactionsInPeriod,
				NetChange:              netChangeInPeriod, //
				Masters:                mastersInPackageGroup,
				EffectiveEndingBalance: effectiveEndingBalance, // 修正後の値
				MaxUsage:               maxUsage,
			}

			var precompTotalForPackage float64
			for _, master := range mastersInPackageGroup {
				if total, ok := precompTotals[master.ProductCode]; ok {
					precompTotalForPackage += total
				}
			}

			pkg.BaseReorderPoint = maxUsage * filters.Coefficient
			pkg.PrecompoundedTotal = precompTotalForPackage
			pkg.ReorderPoint = pkg.BaseReorderPoint + pkg.PrecompoundedTotal
			// ▼▼▼【ここを修正】発注判定を effectiveEndingBalance (発注残込み) で行う ▼▼▼
			pkg.IsReorderNeeded = effectiveEndingBalance < pkg.ReorderPoint && pkg.MaxUsage > 0
			// ▲▲▲【修正ここまで】▲▲▲

			allPackageLedgers = append(allPackageLedgers, pkg)
		}
		if len(allPackageLedgers) > 0 { //
			var yjTotalEnding, yjTotalNetChange, yjTotalStarting float64
			var yjTotalReorderPoint, yjTotalBaseReorderPoint, yjTotalPrecompounded float64
			// ▼▼▼【ここに追加】YJグループ全体の発注残込み在庫 (表示用) ▼▼▼
			var yjTotalEffectiveEnding float64
			// ▲▲▲【追加ここまで】▲▲▲
			isYjReorderNeeded := false

			for _, pkg := range allPackageLedgers {
				if start, ok := pkg.StartingBalance.(float64); ok { //
					yjTotalStarting += start
				}
				if end, ok := pkg.EndingBalance.(float64); ok { //
					yjTotalEnding += end
				}
				// ▼▼▼【ここに追加】YJグループ全体の発注残込み在庫 (表示用) ▼▼▼
				yjTotalEffectiveEnding += pkg.EffectiveEndingBalance
				// ▲▲▲【追加ここまで】▲▲▲
				yjTotalNetChange += pkg.NetChange
				yjTotalReorderPoint += pkg.ReorderPoint
				yjTotalBaseReorderPoint += pkg.BaseReorderPoint
				yjTotalPrecompounded += pkg.PrecompoundedTotal
				if pkg.IsReorderNeeded {
					isYjReorderNeeded = true
				}
			}
			yjGroup.StartingBalance = yjTotalStarting
			yjGroup.EndingBalance = yjTotalEnding
			// ▼▼▼【ここに追加】YJグループ全体の発注残込み在庫 (表示用) ▼▼▼
			// (model.StockLedgerYJGroup に EffectiveEndingBalance フィールドはないが、
			// TKRでは ReorderItemView にコピーされるため、計算だけはしておく)
			// yjGroup.EffectiveEndingBalance = yjTotalEffectiveEnding
			// ▲▲▲【追加ここまで】▲▲▲
			yjGroup.NetChange = yjTotalNetChange
			yjGroup.PackageLedgers = allPackageLedgers
			yjGroup.TotalReorderPoint = yjTotalReorderPoint
			yjGroup.TotalBaseReorderPoint = yjTotalBaseReorderPoint
			yjGroup.TotalPrecompounded = yjTotalPrecompounded
			yjGroup.IsReorderNeeded = isYjReorderNeeded
			result = append(result, yjGroup)
		}
	}
	sort.Slice(result, func(i, j int) bool { //
		// ▼▼▼【ここを修正】TKRの剤型順序に合わせる ▼▼▼
		prio := map[string]int{"内": 1, "外": 2, "歯": 3, "注": 4, "機": 5, "他": 6}
		// ▲▲▲【修正ここまで】▲▲▲
		masterI := mastersByYjCode[result[i].YjCode][0]
		masterJ := mastersByYjCode[result[j].YjCode][0]
		prioI, okI := prio[strings.TrimSpace(masterI.UsageClassification)]
		if !okI {
			prioI = 7 //
		}
		prioJ, okJ := prio[strings.TrimSpace(masterJ.UsageClassification)]
		if !okJ {
			prioJ = 7 //
		}
		if prioI != prioJ {
			return prioI < prioJ
		}
		return masterI.KanaName < masterJ.KanaName
	})
	return result, nil
}
