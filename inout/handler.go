// C:\Users\wasab\OneDrive\デスクトップ\TKR\inout\handler.go
package inout

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
	"tkr/database"
	"tkr/mappers"
	"tkr/mastermanager"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

// SaveRecordInput はフロントエンドから受け取る行データです。
type SaveRecordInput struct {
	ProductCode string  `json:"productCode"`
	ProductName string  `json:"productName"` // 仮マスター作成用
	JanQuantity float64 `json:"janQuantity"`
	DatQuantity float64 `json:"datQuantity"` // (TKRでは使わないがWASABI に合わせて残す)
	ExpiryDate  string  `json:"expiryDate"`
	LotNumber   string  `json:"lotNumber"`
}

// SavePayload はフロントエンドから受け取る全データです。
type SavePayload struct {
	IsNewClient           bool              `json:"isNewClient"`
	ClientCode            string            `json:"clientCode"`
	ClientName            string            `json:"clientName"`
	TransactionDate       string            `json:"transactionDate"`
	TransactionTypeFlag   int               `json:"transactionTypeFlag"` // "11" (入庫) or "12" (出庫)
	Records               []SaveRecordInput `json:"records"`
	OriginalReceiptNumber string            `json:"originalReceiptNumber"`
}

// SaveInOutHandler は入出庫伝票の保存を処理します。
func SaveInOutHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload SavePayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		clientCode := payload.ClientCode
		if payload.IsNewClient {
			exists, err := database.CheckClientExistsByName(tx, payload.ClientName)
			if err != nil {
				http.Error(w, "Failed to check client existence", http.StatusInternalServerError)
				return
			}
			if exists {
				http.Error(w, fmt.Sprintf("得意先名 '%s' は既に存在します。", payload.ClientName), http.StatusConflict)
				return
			}
			newCode, err := database.NextSequenceInTx(tx, "CL", "CL", 4)
			if err != nil {
				http.Error(w, "Failed to generate new client code", http.StatusInternalServerError)
				return
			}
			if err := database.CreateClientInTx(tx, newCode, payload.ClientName); err != nil {
				http.Error(w, "Failed to create new client", http.StatusInternalServerError)
				return
			}
			clientCode = newCode
		}

		var receiptNumber string
		dateStr := payload.TransactionDate
		if dateStr == "" {
			dateStr = time.Now().Format("20060102")
		}

		if payload.OriginalReceiptNumber != "" {
			receiptNumber = payload.OriginalReceiptNumber
			// ▼▼▼【修正】関数呼び出しを修正 ▼▼▼
			if err := database.DeleteTransactionsByReceiptNumberInTx(tx, receiptNumber); err != nil {
				// ▲▲▲【修正ここまで】▲▲▲
				http.Error(w, "Failed to delete old items from slip", http.StatusInternalServerError)
				return
			}
		} else {
			var lastSeq int
			prefix := "IO" + dateStr[2:8] // "IO" + YYMMDD
			q := `SELECT receipt_number FROM transaction_records 
				  WHERE receipt_number LIKE ? 
				  ORDER BY receipt_number DESC LIMIT 1`
			var lastReceiptNumber string
			err = tx.Get(&lastReceiptNumber, q, prefix+"%")
			if err != nil && err != sql.ErrNoRows {
				http.Error(w, "Failed to get last receipt number sequence", http.StatusInternalServerError)
				return
			}
			lastSeq = 0
			if lastReceiptNumber != "" && len(lastReceiptNumber) == 13 { // IO + 6 + 5 = 13
				seqStr := lastReceiptNumber[8:]
				lastSeq, _ = strconv.Atoi(seqStr)
			}
			newSeq := lastSeq + 1
			receiptNumber = fmt.Sprintf("%s%05d", prefix, newSeq) // 13桁
		}

		var finalRecords []model.TransactionRecord
		flag := payload.TransactionTypeFlag // 11 or 12

		for i, rec := range payload.Records {
			if rec.ProductCode == "" {
				continue
			}
			master, err := mastermanager.FindOrCreateMaster(tx, rec.ProductCode, rec.ProductName)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to resolve master for %s: %v", rec.ProductName, err), http.StatusInternalServerError)
				return
			}

			yjQuantity := rec.JanQuantity * master.JanPackInnerQty
			unitPrice := master.NhiPrice
			subtotal := yjQuantity * unitPrice

			tr := model.TransactionRecord{
				TransactionDate: dateStr,
				ClientCode:      clientCode,
				ReceiptNumber:   receiptNumber,
				LineNumber:      fmt.Sprintf("%d", i+1),
				Flag:            flag,
				JanCode:         master.ProductCode,
				JanQuantity:     rec.JanQuantity,
				YjQuantity:      yjQuantity,
				UnitPrice:       unitPrice,
				Subtotal:        subtotal,
				ExpiryDate:      rec.ExpiryDate,
				LotNumber:       rec.LotNumber,
			}

			mappers.MapMasterToTransaction(&tr, master)
			finalRecords = append(finalRecords, tr)
		}

		        if len(finalRecords) > 0 {
		            // ▼▼▼【修正】関数名を PersistTransactionRecordsInTx に変更 ▼▼▼
		            if err := database.PersistTransactionRecordsInTx(tx, finalRecords); err != nil {
		                // ▲▲▲【修正ここまで】▲▲▲
		                log.Printf("Failed to persist records: %v", err)
		                http.Error(w, "Failed to save records to database.", http.StatusInternalServerError)
		                return
		            }
		        }
		
		        		if err := tx.Commit(); err != nil {			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		response := map[string]interface{}{
			"message":       "保存しました",
			"receiptNumber": receiptNumber,
		}
		if payload.IsNewClient {
			response["newClient"] = map[string]string{
				"code": clientCode,
				"name": payload.ClientName,
			}
		}
		json.NewEncoder(w).Encode(response)
	}
}

// GetTransactionsByReceiptNumberHandler は伝票番号で明細を取得します
func GetTransactionsByReceiptNumberHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		receiptNumber := strings.TrimPrefix(r.URL.Path, "/api/transaction/")
		if receiptNumber == "" {
			http.Error(w, "Receipt number is required", http.StatusBadRequest)
			return
		}
		// ▼▼▼【修正】関数呼び出しを修正 ▼▼▼
		records, err := database.GetTransactionsByReceiptNumber(db, receiptNumber)
		// ▲▲▲【修正ここまで】▲▲▲
		if err != nil {
			http.Error(w, "Failed to get transaction details", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(records)
	}
}

// GetReceiptNumbersByDateHandler は日付で伝票番号リストを取得します
func GetReceiptNumbersByDateHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		date := r.URL.Query().Get("date")
		if date == "" {
			http.Error(w, "Date parameter is required", http.StatusBadRequest)
			return
		}
    if len(date) != 8 {
        http.Error(w, "Date must be in YYYYMMDD format", http.StatusBadRequest)
        return
    }
    clientCode := r.URL.Query().Get("client")
    prefix := "IO" + date[2:] // "IO" + YYMMDD
	receipts, err := database.GetReceiptNumbersByDate(db, date, prefix, clientCode)
		if err != nil {
			http.Error(w, "Failed to get receipt numbers", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(receipts)
	}
}

// DeleteTransactionHandler は伝票番号で伝票を削除します
func DeleteTransactionHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		receiptNumber := strings.TrimPrefix(r.URL.Path, "/api/transaction/delete/")
		if receiptNumber == "" {
			http.Error(w, "Receipt number is required", http.StatusBadRequest)
			return
		}
		tx, err := db.Beginx()
		if err != nil {
			http.Error(w, "Failed to start transaction", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// ▼▼▼【修正】関数呼び出しを修正 ▼▼▼
		if err := database.DeleteTransactionsByReceiptNumberInTx(tx, receiptNumber); err != nil {
			// ▲▲▲【修正ここまで】▲▲▲
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "Failed to commit transaction", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"message": "削除しました"})
	}
}
