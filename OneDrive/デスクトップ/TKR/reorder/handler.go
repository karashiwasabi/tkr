// C:\Users\wasab\OneDrive\デスクトップ\TKR\reorder\handler.go
package reorder

import (
	"encoding/json"
	"fmt" // TKRはlogを使う
	"net/http"
	"strconv"
	"time"
	"tkr/aggregation" // TKRのaggregation
	"tkr/config"
	"tkr/database" // TKRのdatabase
	"tkr/mappers"  // TKRのmappers
	"tkr/model"
	"tkr/units" // ★★★ units をインポート

	"github.com/jmoiron/sqlx"
)

// OrderCandidatesResponse は発注候補画面のレスポンスです。
// (WASABI: orders/handlee.go より)
type OrderCandidatesResponse struct {
	Candidates  []OrderCandidateYJGroup `json:"candidates"`
	Wholesalers []model.Wholesaler      `json:"wholesalers"`
}

// OrderCandidateYJGroup は発注候補のYJグループです。
// (WASABI: orders/handlee.go より)
type OrderCandidateYJGroup struct {
	model.StockLedgerYJGroup
	PackageLedgers []OrderCandidatePackageGroup `json:"packageLedgers"`
}

// OrderCandidatePackageGroup は発注候補の包装グループです。
// (WASABI: orders/handlee.go より)
type OrderCandidatePackageGroup struct {
	model.StockLedgerPackageGroup
	Masters            []model.ProductMasterView `json:"masters"`
	ExistingBackorders []model.Backorder         `json:"existingBackorders"`
}

// GenerateOrderCandidatesHandler は発注候補リストを生成します。
// (WASABI: orders/handlee.go を TKR 用に修正)
func GenerateOrderCandidatesHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		kanaName := q.Get("kanaName")
		dosageForm := q.Get("dosageForm")
		shelfNumber := q.Get("shelfNumber")
		coefficientStr := q.Get("coefficient")
		coefficient, err := strconv.ParseFloat(coefficientStr, 64)
		if err != nil {
			coefficient = 1.3 // TKRのデフォルト
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			http.Error(w, "設定ファイルの読み込みに失敗しました: "+err.Error(), http.StatusInternalServerError)
			return
		}

		now := time.Now()
		// ▼▼▼【ここを修正】TKRの集計ロジックのEndDateを「本日」から「未来」に変更 ▼▼▼
		endDate := "99991231" //
		startDate := now.AddDate(0, 0, -cfg.CalculationPeriodDays)

		filters := model.AggregationFilters{
			StartDate:   startDate.Format("20060102"),
			EndDate:     endDate, //
			KanaName:    kanaName,
			DosageForm:  dosageForm,
			ShelfNumber: shelfNumber,
			Coefficient: coefficient,
		}

		// TKRの集計関数を呼ぶ
		// ★★★ 注意: TKRの aggregation.GetStockLedger は発注残を考慮するよう修正が必要
		yjGroups, err := aggregation.GetStockLedger(conn, filters)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// TKRのDB関数を呼ぶ
		allBackorders, err := database.GetAllBackordersList(conn)
		if err != nil {
			http.Error(w, "Failed to get backorder list for candidates", http.StatusInternalServerError)
			return
		}

		// TKRの aggregation.go のキー生成ロジック (units.ResolveName) に合わせる
		backordersByPackageKey := make(map[string][]model.Backorder)
		for _, bo := range allBackorders {
			// units.ResolveName を使って単位名を解決する
			resolvedKey :=
				fmt.Sprintf("%s|%s|%g|%s", bo.YjCode, bo.PackageForm, bo.JanPackInnerQty, units.ResolveName(bo.YjUnitName))
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
						// TKRの aggregation.go は Resolved なキーを使っているので、
						// ここでも Resolved なキー (pkg.PackageKey) でマップを引く
						ExistingBackorders: backordersByPackageKey[pkg.PackageKey],
					}

					// TKRの mappers を使って View を生成
					for _, master := range pkg.Masters {
						masterView := mappers.ToProductMasterView(master)
						newPkgGroup.Masters = append(newPkgGroup.Masters, masterView)
					}
					newYjGroup.PackageLedgers = append(newYjGroup.PackageLedgers, newPkgGroup)
				}
				candidates = append(candidates, newYjGroup)
			}
		}

		// TKRのDB関数を呼ぶ
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

// PlaceOrderHandler は発注内容を発注残として登録します。
// (WASABI: orders/handlee.go を TKR 用に修正)
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

		// ▼▼▼【ここを修正】日付フォーマットを YYYYMMDDHHMMSS に変更 ▼▼▼
		today := time.Now().Format("20060102150405") // YYYYMMDDHHMMSS
		// ▲▲▲【修正ここまで】▲▲▲
		for i := range payload {
			if payload[i].OrderDate == "" {
				payload[i].OrderDate = today
			}
			// YjQuantity (フロントから来る発注単位数 * 包装単位量) を
			// OrderQuantity (発注数量) と RemainingQuantity (残数量) にコピー
			payload[i].OrderQuantity = payload[i].YjQuantity
			payload[i].RemainingQuantity = payload[i].YjQuantity
		}

		// TKRのDB関数を呼ぶ
		if err := database.InsertBackordersInTx(tx, payload); err != nil {
			http.Error(w, "Failed to save backorders", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "発注内容を発注残として登録しました。"})
	}
}
