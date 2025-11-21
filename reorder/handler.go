// C:\Users\wasab\OneDrive\デスクトップ\TKR\reorder\handler.go
package reorder

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
	"tkr/aggregation"
	"tkr/config"
	"tkr/database"
	"tkr/mappers"
	"tkr/model"
	"tkr/units"

	"github.com/jmoiron/sqlx"
)

type OrderCandidatesResponse struct {
	Candidates  []OrderCandidateYJGroup `json:"candidates"`
	Wholesalers []model.Wholesaler      `json:"wholesalers"`
}

type OrderCandidateYJGroup struct {
	model.StockLedgerYJGroup
	PackageLedgers []OrderCandidatePackageGroup `json:"packageLedgers"`
}

type OrderCandidatePackageGroup struct {
	model.StockLedgerPackageGroup
	Masters            []model.ProductMasterView `json:"masters"`
	ExistingBackorders []model.Backorder         `json:"existingBackorders"`
}

// 返品推奨リスト用の構造体
type ReturnCandidate struct {
	Representative   model.ProductMasterView `json:"representative"`
	ProductName      string                  `json:"productName"`
	MakerName        string                  `json:"makerName"`
	PackageForm      string                  `json:"packageForm"`
	PackageKey       string                  `json:"packageKey"`
	LastInDate       string                  `json:"lastInDate"`
	EstimatedExpiry  string                  `json:"estimatedExpiry"`
	StockQuantity    float64                 `json:"stockQuantity"`
	NhiPrice         float64                 `json:"nhiPrice"`
	MinJanPackQty    float64                 `json:"minJanPackQty"`
	Threshold        float64                 `json:"threshold"`
	ExcessQuantity   float64                 `json:"excessQuantity"`
	ReturnableBoxes  int                     `json:"returnableBoxes"`
	UnitName         string                  `json:"unitName"`
	TheoreticalStock float64                 `json:"theoreticalStock"`
}

func GenerateOrderCandidatesHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		kanaName := q.Get("kanaName")
		dosageForm := q.Get("dosageForm")
		shelfNumber := q.Get("shelfNumber")
		coefficientStr := q.Get("coefficient")
		coefficient, err := strconv.ParseFloat(coefficientStr, 64)
		if err != nil {
			coefficient = 1.5
		}

		// --- 予約解除ロジック ---
		func() {
			tx, err := conn.Beginx()
			if err != nil {
				fmt.Printf("WARN: Failed to begin transaction: %v\n", err)
				return
			}
			defer func() {
				if p := recover(); p != nil {
					tx.Rollback()
				}
			}()

			nowStr := time.Now().Format("20060102150405")
			expiredReservations, err := database.GetExpiredReservationsInTx(tx, nowStr)
			if err != nil {
				fmt.Printf("WARN: Failed to get expired reservations: %v\n", err)
				tx.Rollback()
				return
			}

			if len(expiredReservations) > 0 {
				for _, res := range expiredReservations {
					if err := database.DeleteBackorderInTx(tx, res.ID); err != nil {
						fmt.Printf("WARN: Failed to delete reservation ID %d: %v\n", res.ID, err)
						tx.Rollback()
						return
					}
				}
				if err := tx.Commit(); err != nil {
					fmt.Printf("WARN: Failed to commit reservation cleanup: %v\n", err)
				} else {
					fmt.Printf("INFO: Released %d expired reservations.\n", len(expiredReservations))
				}
			} else {
				tx.Rollback()
			}
		}()

		cfg, err := config.LoadConfig()
		if err != nil {
			http.Error(w, "設定ファイルの読み込みに失敗しました: "+err.Error(), http.StatusInternalServerError)
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

		allBackorders, err := database.GetAllBackordersList(conn)
		if err != nil {
			http.Error(w, "Failed to get backorder list", http.StatusInternalServerError)
			return
		}

		backordersByPackageKey := make(map[string][]model.Backorder)
		for _, bo := range allBackorders {
			resolvedKey := fmt.Sprintf("%s|%s|%g|%s", bo.YjCode, bo.PackageForm, bo.JanPackInnerQty, units.ResolveName(bo.YjUnitName))
			backordersByPackageKey[resolvedKey] = append(backordersByPackageKey[resolvedKey], bo)
		}

		var candidates []OrderCandidateYJGroup
		for _, group := range yjGroups {
			if group.IsReorderNeeded {
				newYjGroup := OrderCandidateYJGroup{
					StockLedgerYJGroup: group,
					PackageLedgers:     []OrderCandidatePackageGroup{},
				}

				for _, pkg := range group.PackageLedgers {
					newPkgGroup := OrderCandidatePackageGroup{
						StockLedgerPackageGroup: pkg,
						Masters:                 []model.ProductMasterView{},
						ExistingBackorders:      backordersByPackageKey[pkg.PackageKey],
					}

					for _, master := range pkg.Masters {
						masterView := mappers.ToProductMasterView(master)
						newPkgGroup.Masters = append(newPkgGroup.Masters, masterView)
					}
					newYjGroup.PackageLedgers = append(newYjGroup.PackageLedgers, newPkgGroup)
				}
				candidates = append(candidates, newYjGroup)
			}
		}

		wholesalers, err := database.GetAllWholesalers(conn)
		if err != nil {
			http.Error(w, "Failed to get wholesalers", http.StatusInternalServerError)
			return
		}

		response := OrderCandidatesResponse{
			Candidates:  candidates,
			Wholesalers: wholesalers,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

func PlaceOrderHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload []model.Backorder
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		tx, err := conn.Beginx()
		if err != nil {
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		today := time.Now().Format("20060102150405")

		isReservation := false
		if len(payload) > 0 && payload[0].OrderDate != "" {
			userDate := strings.ReplaceAll(payload[0].OrderDate, "-", "")
			userDate = strings.ReplaceAll(userDate, ":", "")
			userDate = strings.ReplaceAll(userDate, " ", "")
			userDate = strings.ReplaceAll(userDate, "T", "")

			if len(userDate) == 12 {
				userDate += "00"
			}

			if userDate > today {
				isReservation = true
			}
		}

		for i := range payload {
			if payload[i].OrderDate != "" {
				normalized := strings.ReplaceAll(payload[i].OrderDate, "-", "")
				normalized = strings.ReplaceAll(normalized, ":", "")
				normalized = strings.ReplaceAll(normalized, " ", "")
				normalized = strings.ReplaceAll(normalized, "T", "")

				if len(normalized) == 12 {
					normalized += "00"
				}
				payload[i].OrderDate = normalized
			} else {
				payload[i].OrderDate = today
			}

			if isReservation {
				payload[i].WholesalerCode += "_RSV"
			}

			payload[i].OrderQuantity = payload[i].YjQuantity
			payload[i].RemainingQuantity = payload[i].YjQuantity
		}

		if err := database.InsertBackordersInTx(tx, payload); err != nil {
			http.Error(w, "Failed to save backorders", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		msg := "発注内容を発注残として登録しました。"
		if isReservation {
			msg = "発注を予約しました。指定日時まで保留されます。"
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": msg})
	}
}

// GenerateReturnCandidatesHandler
// 在庫過多（不動含む）の品目を検出し、返品推奨リストを作成します。
func GenerateReturnCandidatesHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		kanaName := q.Get("kanaName")
		dosageForm := q.Get("dosageForm")
		shelfNumber := q.Get("shelfNumber")
		coefficientStr := q.Get("coefficient")
		coefficient, err := strconv.ParseFloat(coefficientStr, 64)
		if err != nil {
			coefficient = 1.5 // デフォルト
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			http.Error(w, "設定の読み込みに失敗しました", http.StatusInternalServerError)
			return
		}
		now := time.Now()
		startDate := now.AddDate(0, 0, -cfg.CalculationPeriodDays)

		// 1. 共通集計関数を呼び出す
		aggFilters := model.AggregationFilters{
			StartDate:   startDate.Format("20060102"),
			EndDate:     "99991231",
			KanaName:    kanaName,
			DosageForm:  dosageForm,
			ShelfNumber: shelfNumber,
			Coefficient: 1.0,
		}

		yjGroups, err := aggregation.GetStockLedger(db, aggFilters)
		if err != nil {
			http.Error(w, "在庫集計に失敗しました: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var results []ReturnCandidate

		for _, group := range yjGroups {
			for _, pkg := range group.PackageLedgers {
				// 閾値と過剰数の計算
				threshold := (pkg.BaseReorderPoint * coefficient) + pkg.PrecompoundedTotal
				excess := pkg.EffectiveEndingBalance - threshold

				var master *model.ProductMaster
				if len(pkg.Masters) > 0 {
					master = pkg.Masters[0]
				} else {
					continue
				}

				// 最小包装単位 (JAN包装) の確認
				minJanPackQty := 0.0
				if master.JanPackUnitQty > 0 {
					minJanPackQty = master.JanPackUnitQty
				}

				// ▼▼▼ 修正: 過剰数が「最小JAN包装数量」以上の場合のみリストアップする ▼▼▼
				// 最小包装が0の場合は、過剰があれば出す
				if excess > 0 && (minJanPackQty == 0 || excess >= minJanPackQty) {
					// 返品可能箱数
					returnableBoxes := 0
					if minJanPackQty > 0 {
						returnableBoxes = int(excess / minJanPackQty)
					}

					lastIn := ""
					for _, t := range pkg.Transactions {
						if t.Flag == 1 || t.Flag == 11 {
							if t.TransactionDate > lastIn {
								lastIn = t.TransactionDate
							}
						}
					}

					results = append(results, ReturnCandidate{
						Representative:   mappers.ToProductMasterView(master),
						ProductName:      master.ProductName,
						MakerName:        master.MakerName,
						PackageForm:      master.PackageForm,
						PackageKey:       pkg.PackageKey,
						LastInDate:       lastIn,
						StockQuantity:    pkg.EndingBalance.(float64), // 現在在庫(生)
						TheoreticalStock: pkg.EffectiveEndingBalance,  // 発注残・予製込みの理論在庫
						Threshold:        threshold,
						ExcessQuantity:   excess,
						MinJanPackQty:    minJanPackQty,
						ReturnableBoxes:  returnableBoxes,
						UnitName:         group.YjUnitName,
						NhiPrice:         master.NhiPrice,
					})
				}
				// ▲▲▲ 修正ここまで ▲▲▲
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}
