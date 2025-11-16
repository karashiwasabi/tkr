// C:\Users\wasab\OneDrive\デスクトップ\TKR\inventoryadjustment\handler.go
package inventoryadjustment

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
	"tkr/aggregation"
	"tkr/config"
	"tkr/database"
	"tkr/mappers"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

func GetInventoryDataHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		yjCode := q.Get("yjCode")
		if yjCode == "" {
			http.Error(w, "yjCode is a required parameter", http.StatusBadRequest)
			return
		}

		cfg, err := config.LoadConfig()
		if err != nil {
			http.Error(w, "設定ファイルの読み込みに失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}
		now := time.Now()
		endDate := "99991231"
		startDate := now.AddDate(0, 0, -cfg.CalculationPeriodDays)
		yesterdayDate := now.AddDate(0, 0, -1)

		filtersToday := model.AggregationFilters{
			StartDate: startDate.Format("20060102"),
			EndDate:   endDate,
			YjCode:    yjCode,
		}
		ledgerToday, err := aggregation.GetStockLedger(conn, filtersToday)
		if err !=
			nil {
			http.Error(w, "Failed to get today's stock ledger: "+err.Error(), http.StatusInternalServerError)
			return
		}

		filtersYesterday := model.AggregationFilters{
			StartDate: startDate.Format("20060102"),
			EndDate:   yesterdayDate.Format("20060102"),
			YjCode:    yjCode,
		}
		ledgerYesterday, err := aggregation.GetStockLedger(conn, filtersYesterday)
		if err != nil {
			http.Error(w, "Failed to get yesterday's stock ledger: "+err.Error(), http.StatusInternalServerError)
			return
		}

		transactionLedgerView := mappers.ConvertToView(ledgerToday)
		var yesterdaysStockView *mappers.StockLedgerYJGroupView
		if len(ledgerYesterday) > 0 {
			view := mappers.ConvertToView(ledgerYesterday)
			if len(view) > 0 {
				yesterdaysStockView = &view[0]
			}
		}

		var productCodes []string
		if len(ledgerToday) > 0 {
			for _, pkg := range ledgerToday[0].PackageLedgers {
				for _, master := range pkg.Masters {
					productCodes = append(productCodes, master.ProductCode)
				}
			}
		}

		precompDetails, err := database.GetPreCompoundingDetailsByProductCodes(conn, productCodes)
		if err != nil {
			log.Printf("WARN: Failed to get pre-compounding details: %v",
				err)
		}

		tx, err := conn.Beginx()
		if err != nil {
			http.Error(w, "Failed to start transaction for latest inventory details", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		latestInventoryTxs, err := database.GetLatestInventoryDetailsByYjCode(tx, yjCode)
		if err != nil {
			log.Printf("WARN: Failed to get latest inventory details (flag=0 txs) for adjustment: %v", err)
		}

		deadStockDetails := make([]model.DeadStockRecord, len(latestInventoryTxs))
		for i, tx := range latestInventoryTxs {
			deadStockDetails[i] = model.DeadStockRecord{
				ProductCode:      tx.JanCode,
				YjCode:           tx.YjCode,
				PackageForm:      tx.PackageForm,
				JanPackInnerQty:  tx.JanPackInnerQty,
				YjUnitName:       tx.YjUnitName,
				StockQuantityJan: tx.JanQuantity,
				ExpiryDate:       tx.ExpiryDate,
				LotNumber:        tx.LotNumber,
			}
		}

		response := mappers.ResponseDataView{
			TransactionLedger: transactionLedgerView,
			YesterdaysStock:   yesterdaysStockView,
			DeadStockDetails:  deadStockDetails,
			PrecompDetails:    precompDetails,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

type SavePayload struct {
	Date          string                  `json:"date"`
	YjCode        string                  `json:"yjCode"`
	InventoryData map[string]float64      `json:"inventoryData"`
	DeadStockData []model.DeadStockRecord `json:"deadStockData"`
}

func SaveInventoryDataHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload SavePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			log.Printf("[SaveInventoryDataHandler] ERROR: Invalid request body: %v", err)
			http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		log.Printf("[SaveInventoryDataHandler] Received payload for YJ: %s, Date: %s. DeadStock items: %d", payload.YjCode, payload.Date, len(payload.DeadStockData))

		tx, err := conn.Beginx()
		if err != nil {
			log.Printf("[SaveInventoryDataHandler] ERROR: Failed to start transaction: %v", err)
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		masters, err := database.GetProductMastersByYjCode(tx, payload.YjCode)
		if err != nil {
			log.Printf("[SaveInventoryDataHandler] ERROR: Failed to get product masters for yj %s: %v", payload.YjCode, err)
			http.Error(w, "Failed to get product masters for yj: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("[SaveInventoryDataHandler] Calling SaveGuidedInventoryData with Date: %s, YjCode: %s, Masters found: %d, DeadStock items: %d",
			payload.Date,
			payload.YjCode, len(masters), len(payload.DeadStockData))

		if err := database.SaveGuidedInventoryData(tx, payload.Date, payload.YjCode, masters, payload.InventoryData, payload.DeadStockData); err != nil {
			log.Printf("[SaveInventoryDataHandler] ERROR: Failed to save inventory data in SaveGuidedInventoryData: %v", err)
			http.Error(w, "Failed to save inventory data: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			log.Printf("[SaveInventoryDataHandler] ERROR: Failed to commit transaction: %v", err)
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		log.Printf("[SaveInventoryDataHandler] Successfully saved inventory data for YJ: %s, Date: %s", payload.YjCode, payload.Date)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "棚卸データを保存しました。"})
	}
}

// ▼▼▼【ここから修正】戻り値とメッセージを変更 ▼▼▼
func ClearOldInventoryHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		tx, err := conn.Beginx()
		if err != nil {
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		rowsAffected, err := database.DeleteOldInventoryTransactions(tx)
		if err != nil {
			http.Error(w, "Failed to delete old inventory records: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		var message string
		if rowsAffected == 0 {
			message = "削除対象の古い棚卸履歴はありませんでした。"
		} else {
			message = fmt.Sprintf("最新ではない古い棚卸履歴(flag=0) %d 件を削除しました。", rowsAffected)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": message})
	}
}

// ▲▲▲【修正ここまで】▲▲▲
