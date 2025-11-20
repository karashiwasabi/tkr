// C:\Users\wasab\OneDrive\デスクトップ\TKR\reprocess\reprocess.go
package reprocess

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"tkr/database" // TKRのdb
	"tkr/mappers"  // TKRのmappers
	"tkr/model"    // TKRのmodel

	"github.com/jmoiron/sqlx" // TKRは sqlx を使用
)

// ExecuteReprocess は全ての取引データを最新のマスター情報で更新します。
func ExecuteReprocess(conn *sqlx.DB) error {
	// 全ての製品マスターをメモリにロード（高速化のため）
	allMasters, err := database.GetAllProductMasters(conn) // TKRの関数呼び出し
	if err != nil {
		return fmt.Errorf("ExecuteReprocess: Failed to fetch all product masters: %w", err)
	}
	mastersMap := make(map[string]*model.ProductMaster)
	for _, m := range allMasters {
		mastersMap[m.ProductCode] = m // TKRでは ProductCode が主キー
	}

	// 全ての取引レコードを取得
	rows, err := conn.Query("SELECT " + database.TransactionColumns + " FROM transaction_records")
	if err != nil {
		return fmt.Errorf("ExecuteReprocess: Failed to fetch all transaction records: %w", err)
	}
	defer rows.Close()

	var allRecords []model.TransactionRecord
	for rows.Next() {
		rec, err := database.ScanTransactionRecord(rows) // TKRの関数呼び出し
		if err != nil {
			return fmt.Errorf("ExecuteReprocess: Failed to scan transaction record: %w", err)
		}
		allRecords = append(allRecords, *rec)
	}

	if len(allRecords) == 0 {
		log.Println("ExecuteReprocess: No transaction data to reprocess.")
		return nil
	}

	// バッチ処理で更新
	const batchSize = 500
	updatedCount := 0
	for i := 0; i < len(allRecords); i += batchSize {
		end := i + batchSize
		if end > len(allRecords) {
			end = len(allRecords)
		}
		batch := allRecords[i:end]

		tx, err := conn.Beginx() // TKRは sqlx.Tx
		if err != nil {
			return fmt.Errorf("ExecuteReprocess: Failed to start transaction: %w", err)
		}

		for _, rec := range batch {
			master, ok := mastersMap[rec.JanCode]
			if !ok {
				continue // マスターが見つからないデータはスキップ
			}

			// 1. 最新のマスター情報をレコードにマッピング (TKRの関数名に変更)
			mappers.MapMasterToTransaction(&rec, master)

			// 2. 数量の再計算（マスターの包装規格変更への対応）
			// DAT数量(箱数)がある場合は、それを正としてYJ/JAN数量を再計算する
			if rec.DatQuantity > 0 && master.YjPackUnitQty > 0 {
				// (A) DAT数量(箱数)が存在する場合（DAT取込データ）
				// YJ数量 = DAT数量(箱数) * YJ包装数
				rec.YjQuantity = rec.DatQuantity * master.YjPackUnitQty

				// JAN数量 = YJ数量 / JAN包装内入数
				if master.JanPackInnerQty > 0 {
					rec.JanQuantity = rec.YjQuantity / master.JanPackInnerQty
				} else {
					rec.JanQuantity = 0 // 復旧不可
				}

			} else if rec.JanQuantity > 0 && master.JanPackInnerQty > 0 {
				// (B) JAN数量のみ存在する場合（入出庫データなど）
				rec.YjQuantity = rec.JanQuantity * master.JanPackInnerQty
			} else if rec.YjQuantity > 0 && rec.JanQuantity == 0 && master.JanPackInnerQty > 0 {
				// (C) YJ数量のみ存在する場合（処方データなど）
				rec.JanQuantity = rec.YjQuantity / master.JanPackInnerQty
			}
			// (D) すべて0の場合は、0のまま

			// 3. 金額を再計算 (TKRのFlag定義に合わせてロジックを分岐)
			switch rec.Flag {
			case 1, 2: // 納品(1), 返品(2) -> DAT由来
				// ▼▼▼【修正】DAT由来の納品・返品データは金額を再計算しない ▼▼▼
				// DATファイルの値が正であるため、マスタの薬価等を適用して書き換えることはしない。
				// 何もしない (break不要、処理をスキップ)

			case 0, 3: // 棚卸(0), 処方(3)
				rec.UnitPrice = master.NhiPrice // 薬価を単価とする
				rec.Subtotal = rec.YjQuantity * rec.UnitPrice

			default: // その他 (入庫 11, 出庫 12 など)
				// ユーザー指示：入出庫は薬価ベースで計算してOK
				rec.UnitPrice = master.NhiPrice
				rec.Subtotal = rec.YjQuantity * rec.UnitPrice
			}
			// ▲▲▲【修正ここまで】▲▲▲

			// 4. 処理ステータスを更新 (TKRの定義に合わせて修正)
			if rec.ProcessFlagMA == "PRO" && master.Origin == "JCSHMS" {
				rec.ProcessFlagMA = "COM"
			}

			// 5. データベースを更新 (TKRの関数呼び出し)
			if err := database.UpdateFullTransactionInTx(tx, &rec); err != nil {
				tx.Rollback()
				return fmt.Errorf("ExecuteReprocess: Failed to update record ID %d: %w", rec.ID, err)
			}
			updatedCount++
		}

		if err := tx.Commit(); err != nil {
			tx.Rollback()
			return fmt.Errorf("ExecuteReprocess: Failed to commit transaction: %w", err)
		}
		log.Printf("ExecuteReprocess: Processed %d/%d records...", updatedCount, len(allRecords))
	}

	log.Printf("ExecuteReprocess: Successfully updated %d transaction records.", updatedCount)
	return nil
}

// ProcessTransactionsHandler は ExecuteReprocess をラップするHTTPハンドラです。
func ProcessTransactionsHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("HTTP request received: Starting reprocessing of all transactions...")
		if err := ExecuteReprocess(conn); err != nil {
			log.Printf("HTTP Reprocess Error: %v", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"message": "全取引データの再計算が完了しました。",
		})
	}
}
