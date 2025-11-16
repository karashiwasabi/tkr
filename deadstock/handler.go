// C:\Users\wasab\OneDrive\デスクトップ\TKR\deadstock\handler.go
package deadstock

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"tkr/database"
	"tkr/mappers"
	"tkr/mastermanager"
	"tkr/model"
	"tkr/parsers"

	"github.com/jmoiron/sqlx"
)

type DeadStockListResponse struct {
	Items  []model.DeadStockItem `json:"items"`
	Errors []string              `json:"errors"`
}

func ListDeadStockHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startDate := r.URL.Query().Get("startDate")
		endDate := r.URL.Query().Get("endDate")
		// ▼▼▼【ここに追加】excludeZeroStock を取得 ▼▼▼
		excludeZeroStock := r.URL.Query().Get("excludeZeroStock") == "true"
		// ▲▲▲【追加ここまで】▲▲▲

		if startDate == "" ||
			endDate == "" {
			http.Error(w, "startDate and endDate (YYYYMMDD) are required.", http.StatusBadRequest)
			return
		}

		// ▼▼▼【ここを修正】excludeZeroStock を渡す ▼▼▼
		items, err := database.GetDeadStockList(db, startDate, endDate, excludeZeroStock)
		// ▲▲▲【修正ここまで】▲▲▲
		if err != nil {
			http.Error(w, "Failed to get dead stock list: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(DeadStockListResponse{
			Items:  items,
			Errors: nil,
		})
	}
}

func UploadDeadStockCSVHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "CSVファイルの読み取りに失敗: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		records, err := parsers.ParseDeadStockCSV(file)
		if err != nil {
			http.Error(w, "CSVファイルの解析に失敗: "+err.Error(), http.StatusBadRequest)
			return
		}

		if len(records) == 0 {
			http.Error(w, "読み込むデータがありません（0件、または JAN数量 が 0 です）。", http.StatusBadRequest)
			return
		}

		date := r.FormValue("date")
		if date == "" ||
			len(date) != 8 {
			http.Error(w, "日付(YYYYMMDD)が不正です。", http.StatusBadRequest)
			return
		}

		tx, err := db.Beginx()
		if err != nil {
			http.Error(w, "データベーストランザクションの開始に失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		insertedCount, err := registerDeadStockCSVAsInventory(tx, records, date)
		if err != nil {
			log.Printf("ERROR: registerDeadStockCSVAsInventory: %v", err)
			http.Error(w, "棚卸データの登録に失敗しました: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			http.Error(w, "データベースのコミットに失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully imported dead stock CSV for date %s, inserted %d records.", date, insertedCount)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": fmt.Sprintf("%d件の棚卸データを登録しました。", insertedCount),
		})
	}
}

func registerDeadStockCSVAsInventory(tx *sqlx.Tx, records []parsers.ParsedDeadStockCSVRecord, date string) (int, error) {

	packageStockTotalsYj := make(map[string]float64)
	yjCodeMap := make(map[string]bool)
	var transactionsToInsert []model.TransactionRecord

	var lastSeq int
	var dateYYMMDD string
	if len(date) >= 8 {
		dateYYMMDD = date[2:8]
	} else {
		dateYYMMDD = date
	}
	// ▼▼▼【修正】ADJ -> AJ ▼▼▼
	prefix := "AJ" + dateYYMMDD
	// ▲▲▲【修正ここまで】▲▲▲

	var lastReceiptNumber string
	err := tx.Get(&lastReceiptNumber, `SELECT receipt_number FROM transaction_records WHERE receipt_number LIKE ?
ORDER BY receipt_number DESC LIMIT 1`, prefix+"%")
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("伝票番号の採番に失敗: %w", err)
	}
	// ▼▼▼【修正】14桁 -> 13桁, 9文字目 -> 8文字目 ▼▼▼
	if lastReceiptNumber != "" && len(lastReceiptNumber) == 13 {
		seqStr := lastReceiptNumber[8:]
		lastSeq, _ = strconv.Atoi(seqStr)
	}
	// ▲▲▲【修正ここまで】▲▲▲

	for i, rec := range records {

		var master *model.ProductMaster
		var err error
		var keyForLog string

		if rec.Gs1Code != "" {
			keyForLog = "GS1:" + rec.Gs1Code
			master, err = database.GetProductMasterByGs1Code(tx, rec.Gs1Code)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return 0, fmt.Errorf("GS1コードでのマスター検索に失敗 (GS1: %s): %w", rec.Gs1Code, err)
			}
		}

		if master == nil && rec.ProductCode != "" {
			keyForLog = "JAN:" + rec.ProductCode
			master, err = database.GetProductMasterByCode(tx, rec.ProductCode)
			if err != nil && !errors.Is(err, sql.ErrNoRows) {
				return 0, fmt.Errorf("JANコードでのマスター検索に失敗 (JAN: %s): %w", rec.ProductCode, err)
			}
		}

		if master == nil {
			keyForCreate := rec.Gs1Code
			if keyForCreate == "" {
				keyForCreate = rec.ProductCode
			}

			if keyForCreate == "" {
				log.Printf("WARN: CSVレコードに JANコード も GS1コード もないためスキップ (品名: %s)", rec.ProductName)
				continue
			}
			keyForLog = "Create:" + keyForCreate

			master, err = mastermanager.FindOrCreateMaster(tx, keyForCreate, rec.ProductName)
			if err != nil {
				return 0, fmt.Errorf("マスターの検索/作成に失敗 (Key: %s, Name: %s): %w", keyForCreate, rec.ProductName, err)
			}

			if master.Origin == "PROVISIONAL" && master.Gs1Code == "" && rec.Gs1Code != "" {
				master.Gs1Code = rec.Gs1Code
				input := mastermanager.MasterToInput(master)
				if _, err := mastermanager.UpsertProductMasterSqlx(tx, input); err != nil {
					return 0, fmt.Errorf("仮マスターへのGS1コード更新に失敗 (JAN: %s): %w", master.ProductCode, err)
				}
			}
		}

		log.Printf("CSV Import: Matched (Key: %s) -> Master (JAN: %s, YJ: %s)", keyForLog, master.ProductCode, master.YjCode)

		janQty := rec.JanQuantity
		yjQty := janQty * master.JanPackInnerQty

		newSeq := lastSeq + 1 + i
		receiptNumber := fmt.Sprintf("%s%05d", prefix, newSeq)

		tr := model.TransactionRecord{
			TransactionDate: date,
			Flag:            0,
			ReceiptNumber:   receiptNumber,
			LineNumber:      "1",
			JanCode:         master.ProductCode,
			YjCode:          master.YjCode,
			JanQuantity:     janQty,
			YjQuantity:      yjQty,
			ExpiryDate:      rec.ExpiryDate,
			LotNumber:       rec.LotNumber,
			UnitPrice:       master.NhiPrice,
			Subtotal:        yjQty * master.NhiPrice,
		}
		mappers.MapMasterToTransaction(&tr, master)
		transactionsToInsert = append(transactionsToInsert, tr)

		packageKey := fmt.Sprintf("%s|%s|%g|%s", master.YjCode, master.PackageForm, master.JanPackInnerQty, master.YjUnitName)
		packageStockTotalsYj[packageKey] += yjQty
		yjCodeMap[master.YjCode] = true
	}

	if len(transactionsToInsert) == 0 {
		return 0, nil
	}

	var yjCodes []string
	for yj := range yjCodeMap {
		yjCodes = append(yjCodes, yj)
	}

	productCodes, err := database.GetProductCodesByYjCodes(tx, yjCodes)
	if err != nil {
		return 0, fmt.Errorf("棚卸削除対象の製品コード取得に失敗: %w", err)
	}

	if len(productCodes) > 0 {
		if err := database.DeleteTransactionsByFlagAndDateAndCodes(tx, 0, date, productCodes); err != nil {
			return 0, fmt.Errorf("既存の棚卸データ(flag=0)の削除に失敗: %w", err)
		}
	}

	for _, tr := range transactionsToInsert {
		if err := database.InsertTransactionRecord(tx, tr); err != nil {
			return 0, fmt.Errorf("新しい棚卸データ(flag=0)の挿入に失敗 (JAN: %s): %w", tr.JanCode, err)
		}
	}

	for key, totalYjQty := range packageStockTotalsYj {
		yjCode := strings.Split(key, "|")[0]
		if err := database.UpsertPackageStockInTx(tx, key, yjCode, totalYjQty, date); err != nil {
			return 0, fmt.Errorf("package_stock の更新に失敗 (Key: %s): %w", key, err)
		}
	}

	return len(transactionsToInsert), nil
}

func quoteAll(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func ExportDeadStockHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startDate := r.URL.Query().Get("startDate")
		endDate := r.URL.Query().Get("endDate")
		// ▼▼▼【ここに追加】excludeZeroStock を取得（CSVエクスポートは常に在庫ゼロを含む）▼▼▼
		excludeZeroStock := false //
		// ▲▲▲【追加ここまで】▲▲▲

		if startDate == "" || endDate == "" {
			http.Error(w, "startDate and endDate (YYYYMMDD) are required.", http.StatusBadRequest)
			return
		}

		// ▼▼▼【ここを修正】excludeZeroStock (false) を渡す ▼▼▼
		items, err := database.GetDeadStockList(db, startDate, endDate, excludeZeroStock)
		// ▲▲▲【修正ここまで】▲▲▲
		if err != nil {
			http.Error(w, "Failed to get dead stock list: "+err.Error(), http.StatusInternalServerError)
			return
		}

		var buf bytes.Buffer
		buf.Write([]byte{0xEF, 0xBB, 0xBF})

		header := []string{
			"JANコード",
			"GS1コード",
			"品名",
			"JAN数量",
			"期限",
			"ロット",
		}
		buf.WriteString(strings.Join(header, ",") + "\r\n")

		for _, item := range items {
			if len(item.LotDetails) > 0 {
				for _, lot := range item.LotDetails {
					record := []string{
						quoteAll(lot.JanCode),
						quoteAll(lot.Gs1Code),
						quoteAll(item.ProductName),
						quoteAll(fmt.Sprintf("%.2f", lot.JanQuantity)),
						quoteAll(lot.ExpiryDate),
						quoteAll(lot.LotNumber),
					}
					buf.WriteString(strings.Join(record, ",") +
						"\r\n")
				}
				// ▼▼▼【ここを修正】YJ在庫 -> JAN在庫 で判定 ▼▼▼
			} else if item.StockQuantityJan > 0 {
				// ▲▲▲【修正ここまで】▲▲▲
				record := []string{
					quoteAll(""),
					quoteAll(""),
					quoteAll(item.ProductName),
					// ▼▼▼【ここを修正】0.00 -> item.StockQuantityJan ▼▼▼
					quoteAll(fmt.Sprintf("%.2f", item.StockQuantityJan)),
					// ▲▲▲【修正ここまで】▲▲▲
					quoteAll(""),
					quoteAll(""),
				}
				buf.WriteString(strings.Join(record, ",") + "\r\n")
			}
		}

		filename := fmt.Sprintf("不動在庫リスト_%s-%s.csv", startDate, endDate)

		w.Header().Set("Content-Type", "text/csv;charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(filename))

		w.Write(buf.Bytes())
	}
}
