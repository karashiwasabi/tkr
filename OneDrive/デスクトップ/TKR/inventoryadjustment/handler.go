// C:\Users\wasab\OneDrive\デスクトップ\TKR\inventoryadjustment\handler.go
package inventoryadjustment

import (
	"encoding/json"
	"log"
	"net/http"
	"time"
	"tkr/aggregation" // TKR集計ロジック
	"tkr/config"
	"tkr/database"
	"tkr/mappers" // TKR Viewマッパー
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

// GetInventoryDataHandler は棚卸調整画面に必要な全データを取得します。
// (WASABI: guidedinventory/handler.go  より移植・TKR用に修正)
func GetInventoryDataHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		yjCode := q.Get("yjCode")
		if yjCode == "" {
			http.Error(w, "yjCode is a required parameter", http.StatusBadRequest)
			return
		}

		// 1. 集計期間の設定
		cfg, err := config.LoadConfig()
		if err != nil {
			http.Error(w, "設定ファイルの読み込みに失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}
		now := time.Now()
		endDate := now
		// TKRでは集計開始日を「(本日 - N日)」または「最新棚卸日」の *遅い方* にする
		// GetStockLedger がそのロジックを内包している
		startDate := now.AddDate(0, 0, -cfg.CalculationPeriodDays)
		yesterdayDate := now.AddDate(0, 0, -1)

		// 2. 本日時点の理論在庫を取得
		filtersToday := model.AggregationFilters{
			StartDate: startDate.Format("20060102"),
			EndDate:   endDate.Format("20060102"),
			YjCode:    yjCode,
		}
		ledgerToday, err := aggregation.GetStockLedger(conn, filtersToday)
		if err != nil {
			http.Error(w, "Failed to get today's stock ledger: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 3. 昨日時点の理論在庫を取得
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

		// 4. 表示用にモデルを変換
		transactionLedgerView := mappers.ConvertToView(ledgerToday)
		var yesterdaysStockView *mappers.StockLedgerYJGroupView
		if len(ledgerYesterday) > 0 {
			view := mappers.ConvertToView(ledgerYesterday)
			if len(view) > 0 {
				yesterdaysStockView = &view[0]
			}
		}

		// 5. ロット・期限情報を取得
		tx, err := conn.Beginx() // 読み取りだが念のため
		if err != nil {
			http.Error(w, "Failed to start transaction for dead stock details", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()
		deadStockDetails, err := database.GetDeadStockByYjCode(tx, yjCode)
		if err != nil {
			log.Printf("WARN: Failed to get dead stock details for inventory adjustment: %v", err)
			// エラーでも続行
		}

		// 6. レスポンスを構築
		response := mappers.ResponseDataView{
			TransactionLedger: transactionLedgerView,
			YesterdaysStock:   yesterdaysStockView,
			DeadStockDetails:  deadStockDetails,
			// TKRには予製(PrecompDetails)はない
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}
}

// SavePayload は棚卸保存時のペイロードです。
// (WASABI: guidedinventory/handler.go  より)
type SavePayload struct {
	Date          string                  `json:"date"`
	YjCode        string                  `json:"yjCode"`
	InventoryData map[string]float64      `json:"inventoryData"` // Key: ProductCode, Value: JanQuantity
	DeadStockData []model.DeadStockRecord `json:"deadStockData"`
}

// SaveInventoryDataHandler は棚卸データを保存します。
// (WASABI: guidedinventory/handler.go  より)
func SaveInventoryDataHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload SavePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
			return
		}

		tx, err := conn.Beginx()
		if err != nil {
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// YJコードに紐づく全てのマスター包装を取得
		masters, err := database.GetProductMastersByYjCode(tx, payload.YjCode)
		if err != nil {
			http.Error(w, "Failed to get product masters for yj: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := database.SaveGuidedInventoryData(tx, payload.Date, payload.YjCode, masters, payload.InventoryData, payload.DeadStockData); err != nil {
			http.Error(w, "Failed to save inventory data: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "棚卸データを保存しました。"})
	}
}
