package reorder

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
	"tkr/aggregation"
	"tkr/config"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

// ReorderItemView は発注点リストの表示用構造体です
type ReorderItemView struct {
	YjCode                 string  `json:"yjCode"`
	ProductName            string  `json:"productName"`
	PackageKey             string  `json:"packageKey"`
	YjUnitName             string  `json:"yjUnitName"`
	EffectiveEndingBalance float64 `json:"effectiveEndingBalance"` // TKRでは YJ単位
	ReorderPoint           float64 `json:"reorderPoint"`           // TKRでは YJ単位
	MaxUsage               float64 `json:"maxUsage"`               // TKRでは YJ単位
	PrecompoundedTotal     float64 `json:"precompoundedTotal"`     // TKRでは YJ単位
}

// GetReorderListHandler は発注が必要な品目リストを返します
func GetReorderListHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		kanaName := q.Get("kanaName")
		dosageForm := q.Get("dosageForm") // "all", "内", "外", ...

		// 1. 集計期間の設定
		cfg, err := config.LoadConfig()
		if err != nil {
			http.Error(w, "設定ファイルの読み込みに失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}
		now := time.Now()
		endDate := now
		startDate := now.AddDate(0, 0, -cfg.CalculationPeriodDays)

		// 2. 在庫台帳（発注点計算済み）を取得
		filters := model.AggregationFilters{
			StartDate:   startDate.Format("20060102"),
			EndDate:     endDate.Format("20060102"),
			KanaName:    kanaName,
			DosageForm:  dosageForm,
			Coefficient: 1.5, // TKRでは安全係数を固定 (将来的にConfigから取得も可)
		}
		ledger, err := aggregation.GetStockLedger(conn, filters)
		if err != nil {
			log.Printf("ERROR: GetStockLedger failed: %v", err)
			http.Error(w, "Failed to get stock ledger: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 3. 発注が必要な品目 (PackageKey単位) のみを抽出
		var reorderList []ReorderItemView
		for _, yjGroup := range ledger {
			for _, pkg := range yjGroup.PackageLedgers {
				if pkg.IsReorderNeeded {
					reorderList = append(reorderList, ReorderItemView{
						YjCode:                 yjGroup.YjCode,
						ProductName:            yjGroup.ProductName,
						PackageKey:             pkg.PackageKey,
						YjUnitName:             yjGroup.YjUnitName,
						EffectiveEndingBalance: pkg.EffectiveEndingBalance,
						ReorderPoint:           pkg.ReorderPoint,
						MaxUsage:               pkg.MaxUsage,
						PrecompoundedTotal:     pkg.PrecompoundedTotal,
					})
				}
			}
		}

		log.Printf("Reorder list generated. Filters (Kana: %s, Dosage: %s), Found: %d items", kanaName, dosageForm, len(reorderList))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(reorderList)
	}
}
