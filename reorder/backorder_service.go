// C:\Users\wasab\OneDrive\デスクトップ\TKR\reorder\backorder_service.go
package reorder

import (
	"fmt"
	"strings"
	"time"
	"tkr/database"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

// RegisterBackorders は発注データのリストを受け取り、共通のルールでDB（発注残）に登録します。
// DAT発注とCSV発注の両方から利用されます。
func RegisterBackorders(tx *sqlx.Tx, items []model.Backorder) error {
	if len(items) == 0 {
		return nil
	}

	today := time.Now().Format("20060102150405")
	var backordersToInsert []model.Backorder

	for _, item := range items {
		// 日付の正規化と予約判定
		isReservation := false
		orderDate := item.OrderDate

		if orderDate != "" {
			normalized := strings.ReplaceAll(orderDate, "-", "")
			normalized = strings.ReplaceAll(normalized, ":", "")
			normalized = strings.ReplaceAll(normalized, " ", "")
			normalized = strings.ReplaceAll(normalized, "T", "")

			if len(normalized) == 12 {
				normalized += "00"
			}
			orderDate = normalized

			if orderDate > today {
				isReservation = true
			}
		} else {
			orderDate = today
		}

		// 予約の場合、卸コードにサフィックスを追加
		wholesalerCode := item.WholesalerCode
		if isReservation && !strings.HasSuffix(wholesalerCode, "_RSV") {
			wholesalerCode += "_RSV"
		}

		// 登録用データの構築
		bo := item
		bo.OrderDate = orderDate
		bo.WholesalerCode = wholesalerCode

		// 発注数と残数をセット (フロントエンドから送られてきたYJ数量を使用)
		bo.OrderQuantity = bo.YjQuantity
		bo.RemainingQuantity = bo.YjQuantity

		backordersToInsert = append(backordersToInsert, bo)
	}

	if err := database.InsertBackordersInTx(tx, backordersToInsert); err != nil {
		return fmt.Errorf("failed to insert backorders: %w", err)
	}

	return nil
}
