// C:\Users\wasab\OneDrive\デスクトップ\TKR\valuation\handler.go
package valuation

import (
	"bytes" // ▼▼▼【追加】CSV出力のため ▼▼▼
	"encoding/json"
	"fmt"
	"net/http"
	"net/url" // ▼▼▼【追加】CSV出力のため ▼▼▼

	// ▼▼▼【ここから追加】▼▼▼
	"database/sql"
	"sort"
	"strings" // ▼▼▼【追加】CSV出力のため ▼▼▼
	"time"
	"tkr/aggregation"
	"tkr/config"

	// ▲▲▲【追加ここまで】▲▲▲
	"tkr/database"
	"tkr/model"

	// ▼▼▼【ここに追加】▼▼▼
	"tkr/units"
	// ▲▲▲【追加ここまで】▲▲▲

	"github.com/jmoiron/sqlx"
	// (gofpdf と excelize への参照は削除)
)

// ▼▼▼【ここから修正】GetValuationHandler を aggregation.GetStockLedger を使うように修正 ▼▼▼
func GetValuationHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		filters := model.ValuationFilters{
			Date:                q.Get("date"),
			KanaName:            q.Get("kanaName"),
			UsageClassification: q.Get("dosageForm"), // TKRでは dosageForm
		}
		if filters.Date == "" {
			http.Error(w, "Date parameter is required", http.StatusBadRequest)
			return
		}

		// 1. 在庫評価（棚卸調整）と同じ集計ロジックを呼び出す
		results, err := runValuationAggregation(conn, filters) //
		if err != nil {
			http.Error(w, "Failed to get inventory valuation: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}

// ▲▲▲【修正ここまで】▲▲▲

// (ExportValuationPDFHandler (PDF) は削除)
// (formatCurrency は削除)

// ▼▼▼【ここから修正】TKR在庫CSV互換のCSVエクスポートハンドラ ▼▼▼
func quoteAll(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// ExportValuationCSVHandler は、在庫評価の結果を「TKR独自CSV」形式でエクスポートします。
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

		// 1. 在庫評価データをDBから取得 (修正された runValuationAggregation を呼ぶ)
		results, err := runValuationAggregation(conn, filters) //
		if err != nil {
			http.Error(w, "Failed to get inventory valuation for export: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var buf bytes.Buffer
		buf.Write([]byte{0xEF, 0xBB, 0xBF}) // UTF-8 BOM

		//2. TKR独自CSVインポート形式 (PackageKey, ProductName, JAN数量) に合わせる
		header := []string{
			"PackageKey",
			"ProductName",
			"JAN数量",
		}
		buf.WriteString(strings.Join(header, ",") + "\r\n")

		// 3. データをCSV行に変換
		for _, group := range results {
			for _, row := range group.DetailRows {

				// YJ単位の在庫 (row.Stock) を JAN単位 に変換
				var janQty float64
				if row.JanPackInnerQty > 0 {
					janQty = row.Stock / row.JanPackInnerQty
				} else {
					janQty = 0 // 内包装数量がなければ0
				}

				record := []string{
					quoteAll(row.PackageKey), // PackageKey (DBロジックで追加 )
					quoteAll(row.ProductName),
					quoteAll(fmt.Sprintf("%.2f", janQty)), // JAN数量
				}
				buf.WriteString(strings.Join(record, ",") + "\r\n")
			}
		}

		// 4. ファイル名を「TKR在庫データ...」形式 に合わせる
		filename := fmt.Sprintf("TKR在庫データ(評価日_%s).csv", filters.Date)

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(filename))

		w.Write(buf.Bytes())
	}
}

// ▲▲▲【修正ここまで】▲▲▲

// ▼▼▼【ここから追加】共通の集計実行関数 ▼▼▼
// runValuationAggregation は、
// 1. aggregation.GetStockLedger を呼び出し
// 2. 結果を database.ValuationGroup 形式に変換する
// (GetValuationHandler と ExportValuationCSVHandler の両方から呼び出される)
func runValuationAggregation(conn *sqlx.DB, filters model.ValuationFilters) ([]database.ValuationGroup, error) {

	// 1. 在庫評価（棚卸調整）と同じ集計ロジックを呼び出す
	cfg, err := config.LoadConfig() //
	if err != nil {
		return nil, fmt.Errorf("設定ファイルの読み込みに失敗: %w", err)
	}
	startDate := time.Now().AddDate(0, 0, -cfg.CalculationPeriodDays).Format("20060102")

	aggFilters := model.AggregationFilters{
		StartDate:   startDate,    //
		EndDate:     filters.Date, // 評価基準日
		KanaName:    filters.KanaName,
		DosageForm:  filters.UsageClassification,
		Coefficient: 1.0, // 在庫評価では発注点計算は不要
	}

	// aggregation.GetStockLedger は package_stock を起点とした理論在庫 (EndingBalance) を計算する
	yjGroups, err := aggregation.GetStockLedger(conn, aggFilters) //
	if err != nil {
		return nil, fmt.Errorf("failed to get stock ledger for valuation: %w", err)
	}

	if len(yjGroups) == 0 {
		return []database.ValuationGroup{}, nil
	}

	// 採用済みマスター（JCSHMS由来）を持つYJコードをマップに記録
	yjHasJcshmsMaster := make(map[string]bool)
	// JANコードをキーにしたマスターマップ (剤型グループ分け用)
	mastersByJanCode := make(map[string]*model.ProductMaster)

	for _, yjGroup := range yjGroups {
		for _, pkg := range yjGroup.PackageLedgers {
			for _, master := range pkg.Masters {
				if master.Origin == "JCSHMS" {
					yjHasJcshmsMaster[master.YjCode] = true
				}
				mastersByJanCode[master.ProductCode] = master //
			}
		}
	}

	var detailRows []model.ValuationDetailRow

	// 2. 包装キーごとにループ
	for _, yjGroup := range yjGroups {
		for _, pkg := range yjGroup.PackageLedgers {

			// 3. GetStockLedger が計算した理論在庫を取得
			totalStockForPackage, ok := pkg.EndingBalance.(float64) //
			if !ok {
				totalStockForPackage = 0
			}

			if totalStockForPackage == 0 {
				continue // 在庫ゼロの包装はスキップ
			}

			// 代表マスターを選定 (JCSHMS優先)
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
				continue // マスターがいないキーはスキップ
			}

			// 仮マスター（PROVISIONAL）しか存在しないYJコードの場合、警告フラグ
			showAlert := false
			if repMaster.Origin != "JCSHMS" && !yjHasJcshmsMaster[repMaster.YjCode] {
				showAlert = true
			}

			// 包装仕様 (TKRの units.FormatPackageSpec が要求する型に合わせる)
			tempJcshmsInfo := model.JcshmsInfo{ //
				PackageForm:     repMaster.PackageForm,
				YjUnitName:      repMaster.YjUnitName,
				YjPackUnitQty:   repMaster.YjPackUnitQty,
				JanPackInnerQty: sql.NullFloat64{Float64: repMaster.JanPackInnerQty, Valid: true},
				JanPackUnitQty:  sql.NullFloat64{Float64: repMaster.JanPackUnitQty, Valid: true},
				JanUnitCode:     sql.NullString{String: fmt.Sprintf("%d", repMaster.JanUnitCode), Valid: true},
			}
			spec := units.FormatPackageSpec(&tempJcshmsInfo) //

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

			detailRows = append(detailRows, model.ValuationDetailRow{ //
				YjCode:               repMaster.YjCode,
				ProductName:          repMaster.ProductName,
				ProductCode:          repMaster.ProductCode, // 代表JAN
				PackageSpec:          spec,
				Stock:                totalStockForPackage,                    // YJ単位
				YjUnitName:           units.ResolveName(repMaster.YjUnitName), //
				PackageNhiPrice:      packageNhiPrice,
				PackagePurchasePrice: repMaster.PurchasePrice,
				TotalNhiValue:        totalNhiValue,
				TotalPurchaseValue:   totalPurchaseValue,
				ShowAlert:            showAlert,
				PackageKey:           pkg.PackageKey,            //
				JanPackInnerQty:      repMaster.JanPackInnerQty, //
			})
		}
	}

	// 5. 剤型ごとにグループ化
	resultGroups := make(map[string]*database.ValuationGroup) //
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
			group = &database.ValuationGroup{UsageClassification: uc} //
			resultGroups[uc] = group
		}
		group.DetailRows = append(group.DetailRows, row)
		group.TotalNhiValue += row.TotalNhiValue
		group.TotalPurchaseValue += row.TotalPurchaseValue
	}

	// 6. 最終結果をソート
	order := map[string]int{"内": 1, "外": 2, "歯": 3, "注": 4, "機": 5, "他": 6}
	var finalResult []database.ValuationGroup
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

// ▲▲▲【追加ここまで】▲▲▲
