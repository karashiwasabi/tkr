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
	ProductName     string  `json:"productName"`
	MakerName       string  `json:"makerName"`
	PackageForm     string  `json:"packageForm"`
	LastInDate      string  `json:"lastInDate"`
	EstimatedExpiry string  `json:"estimatedExpiry"`
	StockQuantity   float64 `json:"stockQuantity"`
	NhiPrice        float64 `json:"nhiPrice"`
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
			// ★修正: 予約除外のコードを削除。これで予約分も「既存の発注残」として画面に表示され、
			// 集計(aggregation)側でも在庫としてカウントされるため、二重発注候補が出なくなります。
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

// GenerateReturnCandidatesHandler (期間指定なし版)
func GenerateReturnCandidatesHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, err := config.LoadConfig()
		if err != nil {
			http.Error(w, "設定の読み込みに失敗しました", http.StatusInternalServerError)
			return
		}
		days := cfg.CalculationPeriodDays
		deadline := time.Now().AddDate(0, 0, -days).Format("20060102")

		query := `
			SELECT 
				p.product_name,
				p.maker_name,
				p.package_form,
				p.nhi_price,
				IFNULL(MAX(CASE WHEN t.flag IN (1, 11) THEN t.transaction_date END), '') as last_in_date,
				IFNULL(MAX(CASE WHEN t.flag IN (3, 12) THEN t.transaction_date END), '') as last_out_date,
				SUM(CASE 
					WHEN t.flag IN (1, 11) THEN t.yj_quantity 
					WHEN t.flag IN (3, 2, 12) THEN -t.yj_quantity 
					ELSE 0 END
				) as stock_quantity
			FROM product_master p
			LEFT JOIN transaction_records t ON p.product_code = t.jan_code
			GROUP BY p.product_code
			HAVING 
				stock_quantity > 0 
				AND (last_out_date < ? OR last_out_date = '')
				AND (last_in_date < ? OR last_in_date = '')
			ORDER BY last_in_date ASC
		`

		rows, err := db.Queryx(query, deadline, deadline)
		if err != nil {
			http.Error(w, "返品候補の抽出に失敗しました: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var results []ReturnCandidate
		for rows.Next() {
			var item struct {
				ProductName   string  `db:"product_name"`
				MakerName     string  `db:"maker_name"`
				PackageForm   string  `db:"package_form"`
				NhiPrice      float64 `db:"nhi_price"`
				LastInDate    string  `db:"last_in_date"`
				StockQuantity float64 `db:"stock_quantity"`
			}
			if err := rows.StructScan(&item); err != nil {
				continue
			}

			expiry := ""
			if item.LastInDate != "" {
				t, _ := time.Parse("20060102", item.LastInDate)
				expiry = t.AddDate(3, 0, 0).Format("2006/01") + " (推定)"
			}

			results = append(results, ReturnCandidate{
				ProductName:     item.ProductName,
				MakerName:       item.MakerName,
				PackageForm:     item.PackageForm,
				LastInDate:      item.LastInDate,
				EstimatedExpiry: expiry,
				StockQuantity:   item.StockQuantity,
				NhiPrice:        item.NhiPrice,
			})
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	}
}
