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
	default: // 棚卸(0), その他
		return 0
	}
}

func GetStockLedger(conn *sqlx.DB, filters model.AggregationFilters) ([]model.StockLedgerYJGroup, error) {
	// 1. マスタの取得
	mastersByYjCode, yjCodes, err := GetFilteredMastersAndYjCodes(conn, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered masters: %w", err)
	}
	if len(yjCodes) == 0 {
		return []model.StockLedgerYJGroup{}, nil
	}

	// 2. トランザクション取得
	allProductCodes, err := getAllProductCodesForYjCodes(conn, yjCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to get all product codes: %w", err)
	}
	transactionsByProductCode, err := getTransactionsByProductCodes(conn, allProductCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	precompTotals, err := database.GetPreCompoundingTotals(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to get pre-compounding totals: %w", err)
	}

	backordersMap, err := database.GetAllBackordersMap(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to get backorders: %w", err)
	}

	// 3. PackageKey の生成と PackageStock の一括取得
	var targetPackageKeys []string
	mastersByPackageKeyGlobal := make(map[string][]*model.ProductMaster)

	for _, yjCode := range yjCodes {
		masters := mastersByYjCode[yjCode]
		for _, m := range masters {
			key := database.GeneratePackageKey(m)
			if _, exists := mastersByPackageKeyGlobal[key]; !exists {
				targetPackageKeys = append(targetPackageKeys, key)
			}
			mastersByPackageKeyGlobal[key] = append(mastersByPackageKeyGlobal[key], m)
		}
	}

	packageStockMap, err := database.GetPackageStocksByKeys(conn, targetPackageKeys)
	if err != nil {
		return nil, fmt.Errorf("failed to get package stocks by keys: %w", err)
	}

	// 4. 計算処理
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

		localMastersByPackageKey := make(map[string][]*model.ProductMaster)
		for _, m := range mastersInYjGroup {
			key := database.GeneratePackageKey(m)
			localMastersByPackageKey[key] = append(localMastersByPackageKey[key], m)
		}

		var allPackageLedgers []model.StockLedgerPackageGroup
		for key, mastersInPackageGroup := range localMastersByPackageKey {

			// A. 起点(StartingBalance)の決定
			var startingBalance float64
			var stockStartDate string

			if stockInfo, ok := packageStockMap[key]; ok {
				startingBalance = stockInfo.StockQuantityYj
				stockStartDate = stockInfo.LastInventoryDate
			} else {
				startingBalance = 0.0
				stockStartDate = "00000000"
			}

			// B. 関連トランザクションの収集
			// 「期間表示」のためには StartDate 以降のデータが必要。
			// 「在庫計算（逆算）」のためには StartDate ～ stockStartDate のデータが必要。
			// よって、filters.StartDate より未来の全データを対象とする。
			var relevantTxs []*model.TransactionRecord
			for _, m := range mastersInPackageGroup {
				txs, ok := transactionsByProductCode[m.ProductCode]
				if !ok {
					continue
				}
				for i := range txs {
					// ▼▼▼【修正】stockStartDate ではなく filters.StartDate を基準にする ▼▼▼
					if txs[i].TransactionDate >= filters.StartDate {
						relevantTxs = append(relevantTxs, &txs[i])
					}
					// ▲▲▲【修正ここまで】▲▲▲
				}
			}

			sort.Slice(relevantTxs, func(i, j int) bool {
				if relevantTxs[i].TransactionDate != relevantTxs[j].TransactionDate {
					return relevantTxs[i].TransactionDate < relevantTxs[j].TransactionDate
				}
				return relevantTxs[i].ID < relevantTxs[j].ID
			})

			// C. 逆算による期間開始在庫の算出
			// stockStartDate(確定在庫日) の在庫は startingBalance である。
			// ここから「逆算」して、filters.StartDate(90日前) 時点の在庫を推定する。
			netChangeForBackCalc := 0.0

			for _, t := range relevantTxs {
				qty := signedYjQty(t.Flag, t.YjQuantity)

				// ケースA: トランザクションが [StartDate ... stockStartDate] の範囲にある場合
				// -> この変動は startingBalance に含まれているので、引くことで過去に戻る
				if t.TransactionDate >= filters.StartDate && t.TransactionDate <= stockStartDate {
					netChangeForBackCalc -= qty
				}

				// ケースB: トランザクションが [stockStartDate ... StartDate] の範囲にある場合
				// (StartDateが未来の場合など。今回は稀だがロジックとして保持)
				if t.TransactionDate > stockStartDate && t.TransactionDate < filters.StartDate {
					netChangeForBackCalc += qty
				}
			}

			// 期間開始時点の在庫
			periodStartingBalance := startingBalance + netChangeForBackCalc

			// D. 積み上げ計算 (順算) で帳簿を作成
			runningBalance := periodStartingBalance
			var transactionsInPeriod []model.LedgerTransaction
			var maxUsage float64

			for _, t := range relevantTxs {
				qty := 0.0
				if t.Flag != 0 {
					qty = signedYjQty(t.Flag, t.YjQuantity)
				}

				runningBalance += qty

				if t.Flag == 3 && t.YjQuantity > maxUsage {
					maxUsage = t.YjQuantity
				}

				// 指定期間内のデータであれば表示用リストに追加
				if t.TransactionDate >= filters.StartDate && t.TransactionDate <= filters.EndDate {
					transactionsInPeriod = append(transactionsInPeriod, model.LedgerTransaction{
						TransactionRecord: *t,
						RunningBalance:    runningBalance,
					})
				}
			}

			// E. 最終在庫の確定
			// runningBalance は、論理的には「現在在庫」と一致するはず
			// (startingBalance - backCalc + forwardCalc = startingBalance + forward_from_stock_date)

			backorderQty := backordersMap[key]
			effectiveEndingBalance := runningBalance + backorderQty

			var precompTotalForPackage float64
			for _, master := range mastersInPackageGroup {
				if total, ok := precompTotals[master.ProductCode]; ok {
					precompTotalForPackage += total
				}
			}

			pkg := model.StockLedgerPackageGroup{
				PackageKey:             key,
				StartingBalance:        periodStartingBalance,
				EndingBalance:          runningBalance,
				Transactions:           transactionsInPeriod,
				NetChange:              runningBalance - periodStartingBalance,
				Masters:                mastersInPackageGroup,
				EffectiveEndingBalance: effectiveEndingBalance,
				MaxUsage:               maxUsage,
			}

			pkg.BaseReorderPoint = maxUsage * filters.Coefficient
			pkg.PrecompoundedTotal = precompTotalForPackage
			pkg.ReorderPoint = pkg.BaseReorderPoint + pkg.PrecompoundedTotal
			pkg.IsReorderNeeded = effectiveEndingBalance < pkg.ReorderPoint && pkg.MaxUsage > 0

			allPackageLedgers = append(allPackageLedgers, pkg)
		}

		if len(allPackageLedgers) > 0 {
			var yjTotalEnding, yjTotalNetChange, yjTotalStarting float64
			var yjTotalReorderPoint, yjTotalBaseReorderPoint, yjTotalPrecompounded float64
			var yjTotalEffectiveEnding float64
			isYjReorderNeeded := false

			for _, pkg := range allPackageLedgers {
				if start, ok := pkg.StartingBalance.(float64); ok {
					yjTotalStarting += start
				}
				if end, ok := pkg.EndingBalance.(float64); ok {
					yjTotalEnding += end
				}
				yjTotalEffectiveEnding += pkg.EffectiveEndingBalance
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
			yjGroup.NetChange = yjTotalNetChange
			yjGroup.PackageLedgers = allPackageLedgers
			yjGroup.TotalReorderPoint = yjTotalReorderPoint
			yjGroup.TotalBaseReorderPoint = yjTotalBaseReorderPoint
			yjGroup.TotalPrecompounded = yjTotalPrecompounded
			yjGroup.IsReorderNeeded = isYjReorderNeeded
			result = append(result, yjGroup)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		prio := map[string]int{"内": 1, "外": 2, "歯": 3, "注": 4, "機": 5, "他": 6}
		masterI := mastersByYjCode[result[i].YjCode][0]
		masterJ := mastersByYjCode[result[j].YjCode][0]
		prioI, okI := prio[strings.TrimSpace(masterI.UsageClassification)]
		if !okI {
			prioI = 7
		}
		prioJ, okJ := prio[strings.TrimSpace(masterJ.UsageClassification)]
		if !okJ {
			prioJ = 7
		}
		if prioI != prioJ {
			return prioI < prioJ
		}
		return masterI.KanaName < masterJ.KanaName
	})
	return result, nil
}
