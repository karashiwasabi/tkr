// C:\Users\wasab\OneDrive\デスクトップ\TKR\valuation\handler.go
package valuation

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
	"tkr/aggregation"
	"tkr/config"
	"tkr/database"
	"tkr/model"
	"tkr/units"

	"github.com/jmoiron/sqlx"
)

// GetValuationHandler は在庫評価データをJSONで返します。
func GetValuationHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filters := model.ValuationFilters{
			Date:                q.Get("date"),
			KanaName:            q.Get("kanaName"),
			UsageClassification: q.Get("dosageForm"),
		}
		if filters.Date == "" {
			http.Error(w, "Date parameter is required", http.StatusBadRequest)
			return
		}

		results, err := runValuationAggregation(conn, filters)
		if err != nil {
			http.Error(w, "Failed to get inventory valuation: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}

func quoteAll(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// ExportValuationCSVHandler
func ExportValuationCSVHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filters := model.ValuationFilters{
			Date:                q.Get("date"),
			KanaName:            q.Get("kanaName"),
			UsageClassification: q.Get("dosageForm"),
		}
		if filters.Date == "" {
			http.Error(w, "Date parameter is required", http.StatusBadRequest)
			return
		}

		results, err := runValuationAggregation(conn, filters)
		if err != nil {
			http.Error(w, "Failed to get inventory valuation for export: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var buf bytes.Buffer
		buf.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM

		header := []string{"PackageKey", "ProductName", "JAN数量"}
		buf.WriteString(strings.Join(header, ",") + "\r\n")

		for _, group := range results {
			for _, row := range group.DetailRows {
				var janQty float64
				if row.JanPackInnerQty > 0 {
					janQty = row.Stock / row.JanPackInnerQty
				} else {
					janQty = 0
				}

				record := []string{
					quoteAll(row.PackageKey),
					quoteAll(row.ProductName),
					quoteAll(fmt.Sprintf("%.2f", janQty)),
				}
				buf.WriteString(strings.Join(record, ",") + "\r\n")
			}
		}

		filename := fmt.Sprintf("TKR在庫データ(評価日_%s).csv", filters.Date)
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(filename))
		w.Write(buf.Bytes())
	}
}

func runValuationAggregation(conn *sqlx.DB, filters model.ValuationFilters) ([]database.ValuationGroup, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("設定ファイルの読み込みに失敗: %w", err)
	}

	// 開始日は「90日前」とするが、実際の計算は PackageStock の日付に依存する
	startDate := time.Now().AddDate(0, 0, -cfg.CalculationPeriodDays).Format("20060102")

	aggFilters := model.AggregationFilters{
		StartDate:   startDate,
		EndDate:     filters.Date,
		KanaName:    filters.KanaName,
		DosageForm:  filters.UsageClassification,
		Coefficient: 1.0,
		YjCode:      "",
	}

	yjGroups, err := aggregation.GetStockLedger(conn, aggFilters)
	if err != nil {
		return nil, fmt.Errorf("failed to get stock ledger: %w", err)
	}

	if len(yjGroups) == 0 {
		return []database.ValuationGroup{}, nil
	}

	yjHasJcshmsMaster := make(map[string]bool)
	mastersByJanCode := make(map[string]*model.ProductMaster)

	for _, yjGroup := range yjGroups {
		for _, pkg := range yjGroup.PackageLedgers {
			for _, master := range pkg.Masters {
				if master.Origin == "JCSHMS" {
					yjHasJcshmsMaster[master.YjCode] = true
				}
				mastersByJanCode[master.ProductCode] = master
			}
		}
	}

	var detailRows []model.ValuationDetailRow

	for _, yjGroup := range yjGroups {
		for _, pkg := range yjGroup.PackageLedgers {
			totalStockForPackage, ok := pkg.EndingBalance.(float64)
			if !ok {
				totalStockForPackage = 0
			}

			// 在庫ゼロのスキップ処理は削除済み

			var repMaster *model.ProductMaster
			if len(pkg.Masters) > 0 {
				repMaster = pkg.Masters[0]
				for _, m := range pkg.Masters {
					if m.Origin == "JCSHMS" {
						repMaster = m
						break
					}
				}
			} else {
				continue
			}

			showAlert := false
			if repMaster.Origin != "JCSHMS" && !yjHasJcshmsMaster[repMaster.YjCode] {
				showAlert = true
			}

			tempJcshmsInfo := model.JcshmsInfo{
				PackageForm:     repMaster.PackageForm,
				YjUnitName:      repMaster.YjUnitName,
				YjPackUnitQty:   repMaster.YjPackUnitQty,
				JanPackInnerQty: sql.NullFloat64{Float64: repMaster.JanPackInnerQty, Valid: true},
				JanPackUnitQty:  sql.NullFloat64{Float64: repMaster.JanPackUnitQty, Valid: true},
				JanUnitCode:     sql.NullString{String: fmt.Sprintf("%d", repMaster.JanUnitCode), Valid: true},
			}
			spec := units.FormatPackageSpec(&tempJcshmsInfo)

			unitNhiPrice := repMaster.NhiPrice
			totalNhiValue := totalStockForPackage * unitNhiPrice
			packageNhiPrice := unitNhiPrice * repMaster.YjPackUnitQty

			var totalPurchaseValue float64
			if repMaster.YjPackUnitQty > 0 {
				unitPurchasePrice := repMaster.PurchasePrice / repMaster.YjPackUnitQty
				totalPurchaseValue = totalStockForPackage * unitPurchasePrice
			}

			detailRows = append(detailRows, model.ValuationDetailRow{
				YjCode:               repMaster.YjCode,
				ProductName:          repMaster.ProductName,
				ProductCode:          repMaster.ProductCode,
				PackageSpec:          spec,
				Stock:                totalStockForPackage,
				YjUnitName:           units.ResolveName(repMaster.YjUnitName),
				PackageNhiPrice:      packageNhiPrice,
				PackagePurchasePrice: repMaster.PurchasePrice,
				TotalNhiValue:        totalNhiValue,
				TotalPurchaseValue:   totalPurchaseValue,
				ShowAlert:            showAlert,
				PackageKey:           pkg.PackageKey,
				JanPackInnerQty:      repMaster.JanPackInnerQty,
			})
		}
	}

	resultGroups := make(map[string]*database.ValuationGroup)
	for _, row := range detailRows {
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
			group = &database.ValuationGroup{UsageClassification: uc}
			resultGroups[uc] = group
		}
		group.DetailRows = append(group.DetailRows, row)
		group.TotalNhiValue += row.TotalNhiValue
		group.TotalPurchaseValue += row.TotalPurchaseValue
	}

	order := map[string]int{"内": 1, "外": 2, "歯": 3, "注": 4, "機": 5, "他": 6}
	var finalResult []database.ValuationGroup
	for _, group := range resultGroups {
		sort.Slice(group.DetailRows, func(i, j int) bool {
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
