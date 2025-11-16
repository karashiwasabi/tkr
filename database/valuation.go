// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\valuation.go
package database

import (
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"tkr/model"
	"tkr/units"

	"github.com/jmoiron/sqlx"
)

// ValuationGroup は剤型ごとの在庫評価額の集計結果を保持します。
// (WASABI: db/valuation.go より移植)
type ValuationGroup struct {
	UsageClassification string                     `json:"usageClassification"`
	DetailRows          []model.ValuationDetailRow `json:"detailRows"`
	TotalNhiValue       float64                    `json:"totalNhiValue"`
	TotalPurchaseValue  float64                    `json:"totalPurchaseValue"`
}

// GetInventoryValuation は、指定された基準日時点での在庫評価レポートを生成します。
// (WASABI: db/valuation.go を TKR 用に修正)
func GetInventoryValuation(conn *sqlx.DB, filters model.ValuationFilters) ([]ValuationGroup, error) {

	// 1. フィルタ条件に合致する製品マスターを取得
	masterQuery := `SELECT ` + SelectColumns + ` FROM product_master WHERE 1=1`
	var masterArgs []interface{}
	if filters.KanaName != "" {
		masterQuery += " AND (kana_name LIKE ? OR product_name LIKE ?)"
		masterArgs = append(masterArgs, "%"+filters.KanaName+"%", "%"+filters.KanaName+"%")
	}
	if filters.UsageClassification != "" && filters.UsageClassification != "all" {
		masterQuery += " AND usage_classification = ?"
		masterArgs = append(masterArgs, filters.UsageClassification)
	}

	allMasters, err := getAllProductMastersFiltered(conn, masterQuery, masterArgs...)
	if err != nil {
		return nil, fmt.Errorf("failed to get filtered product masters: %w", err)
	}
	if len(allMasters) == 0 {
		return []ValuationGroup{}, nil
	}

	// 採用済みマスター（JCSHMS由来）を持つYJコードをマップに記録
	yjHasJcshmsMaster := make(map[string]bool)
	// JANコードをキーにしたマスターマップ
	mastersByJanCode := make(map[string]*model.ProductMaster)
	for _, master := range allMasters {
		if master.Origin == "JCSHMS" {
			yjHasJcshmsMaster[master.YjCode] = true
		}
		mastersByJanCode[master.ProductCode] = master
	}

	// TKRの package_key 形式でマスターをグループ化
	mastersByPackageKey := make(map[string][]*model.ProductMaster)
	for _, master := range allMasters {
		key := fmt.Sprintf("%s|%s|%g|%s", master.YjCode, master.PackageForm, master.JanPackInnerQty, units.ResolveName(master.YjUnitName))
		mastersByPackageKey[key] = append(mastersByPackageKey[key], master)
	}

	var detailRows []model.ValuationDetailRow

	// 2. 包装キーごとにループ
	// ▼▼▼【ここから修正】 PackageKey もループ変数で取得 ▼▼▼
	for key, mastersInPackageGroup := range mastersByPackageKey {
		// ▲▲▲【修正ここまで】▲▲▲
		var totalStockForPackage float64

		// 3. 包装グループ内の全JANの在庫を指定日で計算
		for _, m := range mastersInPackageGroup {
			// TKRの CalculateStockOnDate を使用
			stock, err := CalculateStockOnDate(conn, m.ProductCode, filters.Date)
			if err != nil {
				return nil, fmt.Errorf("failed to calculate stock on date for product %s: %w", m.ProductCode, err)
			}
			totalStockForPackage += stock
		}

		if totalStockForPackage == 0 {
			continue // 在庫ゼロの包装はスキップ
		}

		// 代表マスターを選定 (JCSHMS優先)
		var repMaster *model.ProductMaster
		if len(mastersInPackageGroup) > 0 {
			repMaster = mastersInPackageGroup[0]
			for _, m := range mastersInPackageGroup {
				if m.Origin == "JCSHMS" {
					repMaster = m
					break
				}
			}
		} else {
			continue // マスターがいないキーはスキップ
		}

		// 仮マスター（PROVISIONAL）しか存在しないYJコードの場合、警告フラグ
		showAlert := false
		if repMaster.Origin != "JCSHMS" && !yjHasJcshmsMaster[repMaster.YjCode] {
			showAlert = true
		}

		// 包装仕様 (TKRの units.FormatPackageSpec が要求する型に合わせる)
		tempJcshmsInfo := model.JcshmsInfo{
			PackageForm:     repMaster.PackageForm,
			YjUnitName:      repMaster.YjUnitName,
			YjPackUnitQty:   repMaster.YjPackUnitQty,
			JanPackInnerQty: sql.NullFloat64{Float64: repMaster.JanPackInnerQty, Valid: true},
			JanPackUnitQty:  sql.NullFloat64{Float64: repMaster.JanPackUnitQty, Valid: true},
			JanUnitCode:     sql.NullString{String: fmt.Sprintf("%d", repMaster.JanUnitCode), Valid: true},
		}
		spec := units.FormatPackageSpec(&tempJcshmsInfo)

		// 4. 評価額を計算
		unitNhiPrice := repMaster.NhiPrice
		totalNhiValue := totalStockForPackage * unitNhiPrice
		packageNhiPrice := unitNhiPrice * repMaster.YjPackUnitQty

		var totalPurchaseValue float64
		// 仕入単価 (PurchasePrice) は包装単位あたりの価格
		if repMaster.YjPackUnitQty > 0 {
			// YJ単位あたりの仕入単価 = 包装単価 / YJ包装数
			unitPurchasePrice := repMaster.PurchasePrice / repMaster.YjPackUnitQty
			totalPurchaseValue = totalStockForPackage * unitPurchasePrice
		}

		detailRows = append(detailRows, model.ValuationDetailRow{
			YjCode:               repMaster.YjCode,
			ProductName:          repMaster.ProductName,
			ProductCode:          repMaster.ProductCode, // 代表JAN
			PackageSpec:          spec,
			Stock:                totalStockForPackage, // YJ単位
			YjUnitName:           units.ResolveName(repMaster.YjUnitName),
			PackageNhiPrice:      packageNhiPrice,
			PackagePurchasePrice: repMaster.PurchasePrice,
			TotalNhiValue:        totalNhiValue,
			TotalPurchaseValue:   totalPurchaseValue,
			ShowAlert:            showAlert,
			// ▼▼▼【ここから追加】CSV出力用の項目を設定 ▼▼▼
			PackageKey:      key, // ループ変数から PackageKey を設定
			JanPackInnerQty: repMaster.JanPackInnerQty,
			// ▲▲▲【追加ここまで】▲▲▲
		})
	}

	// 5. 剤型ごとにグループ化
	resultGroups := make(map[string]*ValuationGroup)
	for _, row := range detailRows {
		// 代表JANのマスター情報をマップから取得
		master, ok := mastersByJanCode[row.ProductCode]
		if !ok {
			continue
		}
		uc := master.UsageClassification
		if uc == "" {
			uc = "他"
		}
		group, ok := resultGroups[uc]
		if !ok {
			group = &ValuationGroup{UsageClassification: uc}
			resultGroups[uc] = group
		}
		group.DetailRows = append(group.DetailRows, row)
		group.TotalNhiValue += row.TotalNhiValue
		group.TotalPurchaseValue += row.TotalPurchaseValue
	}

	// 6. 最終結果をソート
	order := map[string]int{"内": 1, "外": 2, "歯": 3, "注": 4, "機": 5, "他": 6}
	var finalResult []ValuationGroup
	for _, group := range resultGroups {
		sort.Slice(group.DetailRows, func(i, j int) bool {
			// カナ名でソートするためにマスター情報を参照
			masterI, okI := mastersByJanCode[group.DetailRows[i].ProductCode]
			masterJ, okJ := mastersByJanCode[group.DetailRows[j].ProductCode]
			if !okI || !okJ {
				return group.DetailRows[i].ProductCode < group.DetailRows[j].ProductCode
			}
			return masterI.KanaName < masterJ.KanaName
		})
		finalResult = append(finalResult, *group)
	}
	sort.Slice(finalResult, func(i, j int) bool {
		prioI, okI := order[strings.TrimSpace(finalResult[i].UsageClassification)]
		if !okI {
			prioI = 7
		}
		prioJ, okJ := order[strings.TrimSpace(finalResult[j].UsageClassification)]
		if !okJ {
			prioJ = 7
		}
		return prioI < prioJ
	})

	return finalResult, nil
}

// getAllProductMastersFiltered はフィルタ条件に基づいて製品マスターを取得するヘルパー関数です。
// (WASABI: db/valuation.go より移植)
func getAllProductMastersFiltered(conn *sqlx.DB, query string, args ...interface{}) ([]*model.ProductMaster, error) {
	rows, err := conn.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("GetAllProductMastersFiltered query failed: %w", err)
	}
	defer rows.Close()

	var masters []*model.ProductMaster
	for rows.Next() {
		// TKRの ScanProductMaster を使用
		m, err := ScanProductMaster(rows)
		if err != nil {
			return nil, err
		}
		masters = append(masters, m)
	}
	return masters, nil
}
