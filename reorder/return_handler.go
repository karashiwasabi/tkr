// C:\Users\wasab\OneDrive\デスクトップ\TKR\reorder\return_handler.go
package reorder

import (
	"encoding/json"
	"math"
	"net/http"
	"strconv"
	"time"
	"tkr/aggregation"
	"tkr/config"
	"tkr/model"
	"tkr/units"

	"github.com/jmoiron/sqlx"
)

type ReturnCandidate struct {
	YjCode           string                  `json:"yjCode"`
	ProductName      string                  `json:"productName"`
	PackageKey       string                  `json:"packageKey"`
	Representative   model.ProductMasterView `json:"representative"`
	TheoreticalStock float64                 `json:"theoreticalStock"`
	Threshold        float64                 `json:"threshold"`
	ExcessQuantity   float64                 `json:"excessQuantity"`
	MinJanPackQty    float64                 `json:"minJanPackQty"`
	UnitName         string                  `json:"unitName"`
	ReturnableBoxes  int                     `json:"returnableBoxes"`
}

func GenerateReturnCandidatesHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		kanaName := q.Get("kanaName")
		dosageForm := q.Get("dosageForm")
		shelfNumber := q.Get("shelfNumber")

		// 返品係数
		coefficientStr := q.Get("coefficient")
		coefficient, err := strconv.ParseFloat(coefficientStr, 64)
		if err != nil {
			coefficient = 1.5
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			http.Error(w, "設定ファイルの読み込みに失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}

		now := time.Now()
		endDate := "99991231"
		startDate := now.AddDate(0, 0, -cfg.CalculationPeriodDays)

		filters := model.AggregationFilters{
			StartDate:   startDate.Format("20060102"),
			EndDate:     endDate,
			KanaName:    kanaName,
			DosageForm:  dosageForm,
			ShelfNumber: shelfNumber,
			Coefficient: coefficient,
		}

		yjGroups, err := aggregation.GetStockLedger(conn, filters)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		var candidates []ReturnCandidate

		for _, group := range yjGroups {
			for _, pkg := range group.PackageLedgers {

				// 理論在庫
				currentStock, ok := pkg.EndingBalance.(float64)
				if !ok {
					currentStock = 0
				}

				// 最小の包装単位数量（1箱のバラ数）を探す
				minPackUnitQty := math.MaxFloat64
				var repMaster *model.ProductMaster

				for _, m := range pkg.Masters {
					if repMaster == nil || (repMaster.Origin != "JCSHMS" && m.Origin == "JCSHMS") {
						repMaster = m
					}

					var packQty float64
					// YJ包装単位数量を優先
					if m.YjPackUnitQty > 0 {
						packQty = m.YjPackUnitQty
					} else if m.JanPackUnitQty > 0 {
						packQty = m.JanPackUnitQty
					}

					if packQty > 0 {
						if packQty < minPackUnitQty {
							minPackUnitQty = packQty
						}
					}
				}

				if minPackUnitQty == math.MaxFloat64 {
					minPackUnitQty = 0
				}

				// 1. 保持すべき在庫 (KeepStock) = (係数×発注点 + 予製)
				keepStock := pkg.ReorderPoint

				// 2. リストアップ判定用閾値 = KeepStock + 最小包装単位(1箱)
				// ※「1箱以上余っているか」を判定するため、ここでは +1箱 する
				thresholdForListing := keepStock + minPackUnitQty

				// 判定: 理論在庫 >= 判定用閾値
				if currentStock >= thresholdForListing {

					// 3. 返品可能総数 = 理論在庫 - 保持すべき在庫
					// ※返すときは「KeepStock」を超えた分をすべて対象にする（+1箱を残す必要はない）
					rawExcess := currentStock - keepStock

					if minPackUnitQty > 0 {
						// 箱数計算 (切り捨て)
						returnableBoxes := math.Floor(rawExcess / minPackUnitQty)

						// 返品推奨数 = 箱数 × 入数
						excessQuantity := returnableBoxes * minPackUnitQty

						if excessQuantity > 0 {
							var repView model.ProductMasterView
							if repMaster != nil {
								repView = model.ProductMasterView{
									ProductMaster: *repMaster,
									JanUnitName:   units.ResolveName(repMaster.YjUnitName),
								}
							}

							candidates = append(candidates, ReturnCandidate{
								YjCode:           group.YjCode,
								ProductName:      group.ProductName,
								PackageKey:       pkg.PackageKey,
								Representative:   repView,
								TheoreticalStock: currentStock,
								Threshold:        thresholdForListing, // 画面には「リストに載る基準」を表示
								ExcessQuantity:   excessQuantity,
								MinJanPackQty:    minPackUnitQty,
								UnitName:         group.YjUnitName,
								ReturnableBoxes:  int(returnableBoxes),
							})
						}
					}
				}
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(candidates)
	}
}
