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
	// 1. マスタとYJコードの取得
	mastersByYjCode, yjCodes, err := GetFilteredMastersAndYjCodes(conn, filters)
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered masters: %w", err)
	}
	if len(yjCodes) == 0 {
		return []model.StockLedgerYJGroup{}, nil
	}

	// 2. 全トランザクションの取得 (メモリ上でフィルタするため一旦全て取得)
	allProductCodes, err := getAllProductCodesForYjCodes(conn, yjCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to get all product codes for yj codes: %w", err)
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

	// 4. 計算処理 (PackageKey単位)
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

			// =================================================================================
			// ステップ A: 【計算（正解）】 現在在庫の確定 (PackageStock起点)
			// =================================================================================

			var startingBalance float64
			var stockStartDate string

			if stockInfo, ok := packageStockMap[key]; ok {
				startingBalance = stockInfo.StockQuantityYj
				stockStartDate = stockInfo.LastInventoryDate
			} else {
				startingBalance = 0.0
				stockStartDate = "00000000" // データなし＝通算
			}

			// 関連する全トランザクションをマスタグループから集約
			var allRelatedTxs []*model.TransactionRecord
			for _, m := range mastersInPackageGroup {
				txs, ok := transactionsByProductCode[m.ProductCode]
				if !ok {
					continue
				}
				for i := range txs {
					allRelatedTxs = append(allRelatedTxs, &txs[i])
				}
			}

			// 日付順ソート
			sort.Slice(allRelatedTxs, func(i, j int) bool {
				if allRelatedTxs[i].TransactionDate != allRelatedTxs[j].TransactionDate {
					return allRelatedTxs[i].TransactionDate < allRelatedTxs[j].TransactionDate
				}
				return allRelatedTxs[i].ID < allRelatedTxs[j].ID
			})

			// PackageStock日より未来の変動を積み上げる（これが現在在庫の正解）
			currentTheoreticalStock := startingBalance
			for _, t := range allRelatedTxs {
				if t.TransactionDate > stockStartDate {
					if t.Flag != 0 { // 棚卸レコードは無視
						currentTheoreticalStock += signedYjQty(t.Flag, t.YjQuantity)
					}
				}
			}

			// =================================================================================
			// ステップ B: 【表示（リスト）】 90日分のリスト作成
			// =================================================================================

			var transactionsInPeriod []model.LedgerTransaction
			var txsForDisplay []*model.TransactionRecord
			netChangeInPeriod := 0.0
			var maxUsage float64

			// 表示期間（90日前）以降のデータを収集
			for _, t := range allRelatedTxs {
				if t.TransactionDate >= filters.StartDate && t.TransactionDate <= filters.EndDate {
					txsForDisplay = append(txsForDisplay, t)

					qty := 0.0
					if t.Flag != 0 {
						qty = signedYjQty(t.Flag, t.YjQuantity)
					}
					netChangeInPeriod += qty

					if t.Flag == 3 && t.YjQuantity > maxUsage {
						maxUsage = t.YjQuantity
					}
				}
			}

			// =================================================================================
			// ステップ C: 【つじつま合わせ】 表示用開始在庫の逆算
			// =================================================================================

			// 「現在在庫(正解)」から「期間内の変動」を引けば、「期間開始時点の在庫」になる
			// ※filters.EndDate が未来(99991231)の場合はこれでOK
			// もしfilters.EndDateが過去なら、EndDate以降の変動も考慮が必要だが、
			// 今回の要件（現在在庫を知る、直近90日を見る）ではこれで十分整合する

			// 全期間の変動を考慮した現在在庫から、表示期間の変動を引く
			// (厳密には、表示期間より「後」のデータがある場合はそれも引く必要がある)
			netChangeAfterPeriod := 0.0
			for _, t := range allRelatedTxs {
				if t.TransactionDate > filters.EndDate {
					if t.Flag != 0 {
						netChangeAfterPeriod += signedYjQty(t.Flag, t.YjQuantity)
					}
				}
			}

			// 期間開始在庫 = 現在在庫 - (期間内変動 + 期間後変動)
			periodStartingBalance := currentTheoreticalStock - (netChangeInPeriod + netChangeAfterPeriod)

			// 帳簿の作成（RunningBalanceの計算）
			runningBalance := periodStartingBalance
			for _, t := range txsForDisplay {
				qty := 0.0
				if t.Flag != 0 {
					qty = signedYjQty(t.Flag, t.YjQuantity)
				}
				runningBalance += qty

				transactionsInPeriod = append(transactionsInPeriod, model.LedgerTransaction{
					TransactionRecord: *t,
					RunningBalance:    runningBalance,
				})
			}

			// =================================================================================
			// ステップ D: 最終結果の格納
			// =================================================================================

			backorderQty := backordersMap[key]
			effectiveEndingBalance := currentTheoreticalStock + backorderQty // 現在在庫 + 発注残

			var precompTotalForPackage float64
			for _, master := range mastersInPackageGroup {
				if total, ok := precompTotals[master.ProductCode]; ok {
					precompTotalForPackage += total
				}
			}

			pkg := model.StockLedgerPackageGroup{
				PackageKey:             key,
				StartingBalance:        periodStartingBalance,   // 画面表示用（90日前）
				EndingBalance:          currentTheoreticalStock, // 現在在庫（計算の正解）
				Transactions:           transactionsInPeriod,    // 表示用リスト
				NetChange:              netChangeInPeriod,
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
