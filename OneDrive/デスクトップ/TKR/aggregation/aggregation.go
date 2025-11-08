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

// ... (signedYjQty 関数は変更なし) ...
func signedYjQty(flag int, yjQty float64) float64 { //
	switch flag {
	case 1: // 納品
		return yjQty
	// ▼▼▼【ここから修正】返品(2)もマイナスとして扱う ▼▼▼
	case 3: // 処方
		return -yjQty
	case 2: // 返品
		return -yjQty
	default: // 棚卸(0), その他
		return 0
	}
}

// ▼▼▼【ここから修正】GetStockLedger の在庫計算ロジックを変更 ▼▼▼
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
	if err != nil {
		return nil, fmt.Errorf("failed to get all transactions for product codes: %w", err)
	}

	// ▼▼▼【ここから追加】(WASABI: db/aggregation.go より) ▼▼▼
	// (TKRには発注残(backorders)はまだないため、予製のみ取得)
	precompTotals, err := database.GetPreCompoundingTotals(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to get pre-compounding totals for aggregation: %w", err)
	}
	// ▲▲▲【追加ここまで】▲▲▲

	// ▼▼▼【ここから追加】YJコードに紐づく包装在庫の起点マップを取得 ▼▼▼
	packageStockMap, err := database.GetPackageStockByYjCode(conn, filters.YjCode) //
	if err != nil {
		return nil, fmt.Errorf("failed to get package stock map: %w", err)
	}
	// ▲▲▲【追加ここまで】▲▲▲

	var result []model.StockLedgerYJGroup
	for _, yjCode := range yjCodes {
		mastersInYjGroup, ok := mastersByYjCode[yjCode]
		if !ok || len(mastersInYjGroup) == 0 { //
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
			// ▼▼▼【修正】units.ResolveName を使ってキーを生成 ▼▼▼
			key := fmt.Sprintf("%s|%s|%g|%s", m.YjCode, m.PackageForm, m.JanPackInnerQty, units.ResolveName(m.YjUnitName)) //
			mastersByPackageKey[key] = append(mastersByPackageKey[key], m)
		}
		var allPackageLedgers []model.StockLedgerPackageGroup
		for key, mastersInPackageGroup := range mastersByPackageKey {

			// ▼▼▼【ここから修正】在庫起点を package_stock テーブルから取得 ▼▼▼
			var startingBalance float64
			var stockStartDate string // 在庫計算の開始日 (この日を含まない)

			if stockInfo, ok := packageStockMap[key]; ok { //
				// 1. package_stock に記録がある場合
				startingBalance = stockInfo.StockQuantityYj  //
				stockStartDate = stockInfo.LastInventoryDate //
			} else {
				// 2. package_stock に記録がない場合 (初回起動時など)
				startingBalance = 0.0       //
				stockStartDate = "00000000" // 全期間を集計対象とする //
			}

			// 3. 起点日(stockStartDate)の *翌日* から、フィルタの終了日(EndDate)までの増減(flag=1, 3, 2)を計算
			netChangeAfterStockDate := 0.0
			for _, m := range mastersInPackageGroup {
				txs, ok := transactionsByProductCode[m.ProductCode]
				if !ok {
					continue
				}
				for _, t := range txs {
					// ▼▼▼【修正】stockStartDate より後 (>)、かつ EndDate 以前 (<=) の取引を集計 ▼▼▼
					if t.TransactionDate > stockStartDate && t.TransactionDate <= filters.EndDate { //
						netChangeAfterStockDate += signedYjQty(t.Flag, t.YjQuantity) //
					}
				}
			}

			// 4. 現在の理論在庫 = 起点在庫 + 起点日以降の増減
			currentTheoreticalStock := startingBalance + netChangeAfterStockDate //

			// 5. フィルタの開始日(StartDate)時点の期首在庫を逆算
			netChangeInPeriod := 0.0
			var txsForPackageInPeriod []*model.TransactionRecord
			for _, m := range mastersInPackageGroup { //
				txs, ok := transactionsByProductCode[m.ProductCode]
				if !ok {
					continue
				}
				for _, t := range txs {
					if t.TransactionDate >= filters.StartDate && t.TransactionDate <= filters.EndDate {
						// 期間内の取引 (flag=1, 3, 2) の増減
						netChangeInPeriod += signedYjQty(t.Flag, t.YjQuantity) //
						// 期間内の全取引 (flag=0 も含む) を台帳表示用にコピー
						txCopy := t
						txsForPackageInPeriod = append(txsForPackageInPeriod, &txCopy) //
					}
				}
			}

			// 期首在庫 = 現在の理論在庫 - 期間内の増減
			periodStartingBalance := currentTheoreticalStock - netChangeInPeriod //

			// ▼▼▼【ここから追加】maxUsage の計算 (WASABI: db/aggregation.go より) ▼▼▼
			var maxUsage float64
			// ▲▲▲【追加ここまで】▲▲▲

			// ▼▼▼【ここから修正】台帳表示ロジック (期首在庫を起点にする) ▼▼▼
			var transactionsInPeriod []model.LedgerTransaction
			runningBalance := periodStartingBalance // 起点を periodStartingBalance に変更 //

			sort.Slice(txsForPackageInPeriod, func(i, j int) bool { //
				if txsForPackageInPeriod[i].TransactionDate != txsForPackageInPeriod[j].TransactionDate {
					return txsForPackageInPeriod[i].TransactionDate < txsForPackageInPeriod[j].TransactionDate
				}
				return txsForPackageInPeriod[i].ID < txsForPackageInPeriod[j].ID
			})

			// 期間内の棚卸合計 (flag=0) を日付ごとに集計
			periodInventorySums := make(map[string]float64) //
			for _, t := range txsForPackageInPeriod {
				if t.Flag == 0 {
					periodInventorySums[t.TransactionDate] += t.YjQuantity //
				}
				// ▼▼▼【ここから追加】maxUsage の計算 (WASABI: db/aggregation.go より) ▼▼▼
				// TKRでは flag=3 (処方)
				if t.Flag == 3 && t.YjQuantity > maxUsage {
					maxUsage = t.YjQuantity
				}
				// ▲▲▲【追加ここまで】▲▲▲
			}

			lastProcessedDate := ""
			for _, t := range txsForPackageInPeriod { //
				// 日付が変わる瞬間に、前日の棚卸(flag=0)を反映させる
				if t.TransactionDate != lastProcessedDate && lastProcessedDate != "" { //
					if inventorySum, ok := periodInventorySums[lastProcessedDate]; ok { //
						runningBalance = inventorySum // 在庫を棚卸値で上書き //
					}
				}

				if t.Flag == 0 {
					// 棚卸当日。この時点ではまだ増減させない（次の日付の処理で上書きされる）
				} else {
					runningBalance += signedYjQty(t.Flag, t.YjQuantity) // 納品・処方・返品を加算 //
				}

				transactionsInPeriod = append(transactionsInPeriod, model.LedgerTransaction{TransactionRecord: *t, RunningBalance: runningBalance}) //
				lastProcessedDate = t.TransactionDate                                                                                               //
			}

			// 最終日の棚卸(flag=0)を反映
			if inventorySum, ok := periodInventorySums[lastProcessedDate]; ok { //
				runningBalance = inventorySum //
			}

			// ▼▼▼【ここから追加】予製と発注点の計算 (WASABI: db/aggregation.go  より) ▼▼▼
			// (TKRには発注残(backorderQty)はまだないため0とする)
			backorderQty := 0.0
			effectiveEndingBalance := runningBalance + backorderQty

			pkg := model.StockLedgerPackageGroup{
				PackageKey:             key,
				StartingBalance:        periodStartingBalance, // 期間の期首在庫 //
				EndingBalance:          runningBalance,        // 期間の期末在庫 //
				Transactions:           transactionsInPeriod,
				NetChange:              netChangeInPeriod, // 期間内の増減 //
				Masters:                mastersInPackageGroup,
				EffectiveEndingBalance: effectiveEndingBalance,
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
			pkg.IsReorderNeeded = effectiveEndingBalance < pkg.ReorderPoint && pkg.MaxUsage > 0
			// ▲▲▲【追加ここまで】▲▲▲

			allPackageLedgers = append(allPackageLedgers, pkg)
		}
		if len(allPackageLedgers) > 0 { //
			// ▼▼▼【修正】YJグループの合計に発注点関連を追加 ▼▼▼
			var yjTotalEnding, yjTotalNetChange, yjTotalStarting float64
			var yjTotalReorderPoint, yjTotalBaseReorderPoint, yjTotalPrecompounded float64
			isYjReorderNeeded := false
			// ▲▲▲【修正ここまで】▲▲▲

			for _, pkg := range allPackageLedgers {
				if start, ok := pkg.StartingBalance.(float64); ok { //
					yjTotalStarting += start
				}
				if end, ok := pkg.EndingBalance.(float64); ok { //
					yjTotalEnding += end
				}
				yjTotalNetChange += pkg.NetChange
				// ▼▼▼【ここから追加】(WASABI: db/aggregation.go  より) ▼▼▼
				yjTotalReorderPoint += pkg.ReorderPoint
				yjTotalBaseReorderPoint += pkg.BaseReorderPoint
				yjTotalPrecompounded += pkg.PrecompoundedTotal
				if pkg.IsReorderNeeded {
					isYjReorderNeeded = true
				}
				// ▲▲▲【追加ここまで】▲▲▲
			}
			yjGroup.StartingBalance = yjTotalStarting
			yjGroup.EndingBalance = yjTotalEnding
			yjGroup.NetChange = yjTotalNetChange
			yjGroup.PackageLedgers = allPackageLedgers
			// ▼▼▼【ここから追加】(WASABI: db/aggregation.go  より) ▼▼▼
			yjGroup.TotalReorderPoint = yjTotalReorderPoint
			yjGroup.TotalBaseReorderPoint = yjTotalBaseReorderPoint
			yjGroup.TotalPrecompounded = yjTotalPrecompounded
			yjGroup.IsReorderNeeded = isYjReorderNeeded
			// ▲▲▲【追加ここまで】▲▲▲
			result = append(result, yjGroup)
		}
	}
	sort.Slice(result, func(i, j int) bool { //
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

// ▼▼▼【ここから修正】GetFilteredMastersAndYjCodes の kanaName フィルタロジックを変更 ▼▼▼
func GetFilteredMastersAndYjCodes(conn *sqlx.DB, filters model.AggregationFilters) (map[string][]*model.ProductMaster, []string, error) {
	query := `SELECT * FROM product_master p WHERE 1=1 ` //
	var args []interface{}
	if filters.YjCode != "" { //
		query += " AND p.yj_code = ? " //
		args = append(args, filters.YjCode)
	}
	if filters.KanaName != "" {
		// ▼▼▼【修正】製品名(product_name)でのOR検索を削除 ▼▼▼
		query += " AND p.kana_name LIKE ? "       //
		args = append(args, filters.KanaName+"%") // カナ名は前方一致のみ //
		// ▲▲▲【修正ここまで】▲▲▲
	}
	// ★一般名フィルタを追加
	if filters.GenericName != "" {
		query += " AND p.generic_name LIKE ?" //
		args = append(args, "%"+filters.GenericName+"%")
	}
	if filters.DosageForm != "" && filters.DosageForm != "all" {
		query += " AND p.usage_classification = ?" //
		args = append(args, filters.DosageForm)
	}
	if filters.ShelfNumber != "" {
		query += " AND p.shelf_number LIKE ? " //
		args = append(args, "%"+filters.ShelfNumber+"%")
	}

	// ★ソート順を kana_name に変更
	query += " ORDER BY p.kana_name " //

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
			// ▼▼▼【削除】未使用の key 変数定義を削除 ▼▼▼
			// key := fmt.Sprintf("%s|%s|%g|%s", m.YjCode, m.PackageForm, m.JanPackInnerQty, units.ResolveName(m.YjUnitName)) // [ This is the unused 'key' at L287]
			// ▲▲▲【削除ここまで】▲▲▲
			mastersByYjCode[m.YjCode] = append(mastersByYjCode[m.YjCode], &m) //
			if !yjCodeMap[m.YjCode] {
				yjCodeMap[m.YjCode] = true
				yjCodes = append(yjCodes, m.YjCode) // 検索結果のソート順でYJコードを追加
			}
		}
	}

	return mastersByYjCode, yjCodes, nil
}

// ... (getAllProductCodesForYjCodes  は変更なし) ...
func getAllProductCodesForYjCodes(conn *sqlx.DB, yjCodes []string) ([]string, error) { //
	if len(yjCodes) == 0 {
		return []string{}, nil
	}
	query, args, err := sqlx.In(`
		SELECT DISTINCT product_code FROM product_master WHERE yj_code IN (?)
		UNION
		SELECT DISTINCT jan_code FROM transaction_records WHERE yj_code IN (?)`, yjCodes, yjCodes) //
	if err != nil {
		return nil, fmt.Errorf("failed to create IN query for all product codes: %w", err)
	}
	query = conn.Rebind(query)
	var codes []string
	if err := conn.Select(&codes, query, args...); err != nil { //
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

// ... (getTransactionsByProductCodes  は変更なし) ...
func getTransactionsByProductCodes(conn *sqlx.DB, productCodes []string) (map[string][]model.TransactionRecord, error) { //
	transactionsMap := make(map[string][]model.TransactionRecord)
	if len(productCodes) == 0 {
		return transactionsMap, nil
	}
	const batchSize = 500
	for i := 0; i < len(productCodes); i += batchSize { //
		end := i + batchSize
		if end > len(productCodes) {
			end = len(productCodes)
		}
		batch := productCodes[i:end]
		if len(batch) > 0 {
			query, args, err := sqlx.In("SELECT "+database.TransactionColumns+" FROM transaction_records WHERE jan_code IN (?) ORDER BY transaction_date, id", batch) //
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
