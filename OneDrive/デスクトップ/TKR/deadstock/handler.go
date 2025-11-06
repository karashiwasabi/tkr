// C:\Users\wasab\OneDrive\デスクトップ\TKR\deadstock\handler.go
package deadstock

import (
	"bytes"
	"database/sql" // (csv.NewWriter を使うため残す)
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url" // ▼▼▼【追加】ファイル名エンコード用 ▼▼▼
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
		startDate := r.URL.Query().Get("startDate") // YYYYMMDD
		endDate := r.URL.Query().Get("endDate")     // YYYYMMDD

		if startDate == "" || endDate == "" {
			http.Error(w, "startDate and endDate (YYYYMMDD) are required.", http.StatusBadRequest)
			return
		}

		items, err := database.GetDeadStockList(db, startDate, endDate)
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

		// 1. ファイルを取得
		file, _, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "CSVファイルの読み取りに失敗: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer file.Close()

		// 2. CSVをパース
		records, err := parsers.ParseDeadStockCSV(file)
		if err != nil {
			http.Error(w, "CSVファイルの解析に失敗: "+err.Error(), http.StatusBadRequest)
			return
		}

		if len(records) == 0 {
			http.Error(w, "読み込むデータがありません（0件、または JAN数量 が 0 です）。", http.StatusBadRequest)
			return
		}

		// 3. 当日の日付を取得
		date := r.FormValue("date") // JSから 'YYYYMMDD' 形式で受け取る
		if date == "" || len(date) != 8 {
			http.Error(w, "日付(YYYYMMDD)が不正です。", http.StatusBadRequest)
			return
		}

		// 4. トランザクション開始
		tx, err := db.Beginx()
		if err != nil {
			http.Error(w, "データベーストランザクションの開始に失敗: "+err.Error(), http.StatusInternalServerError)
			return
		}
		defer tx.Rollback() // エラー時はロールバック

		// 5. データベース登録処理 (このファイル内に移植した関数を呼び出す)
		insertedCount, err := registerDeadStockCSVAsInventory(tx, records, date)
		if err != nil {
			log.Printf("ERROR: registerDeadStockCSVAsInventory: %v", err)
			http.Error(w, "棚卸データの登録に失敗しました: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 6. コミット
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

// registerDeadStockCSVAsInventory は、CSVデータを棚卸(flag=0)として登録します。
func registerDeadStockCSVAsInventory(tx *sqlx.Tx, records []parsers.ParsedDeadStockCSVRecord, date string) (int, error) {

	packageStockTotalsYj := make(map[string]float64)
	yjCodeMap := make(map[string]bool)
	var transactionsToInsert []model.TransactionRecord

	// 伝票番号
	var lastSeq int
	var dateYYMMDD string
	if len(date) >= 8 {
		dateYYMMDD = date[2:8]
	} else {
		dateYYMMDD = date
	}
	prefix := "ADJ" + dateYYMMDD

	var lastReceiptNumber string
	err := tx.Get(&lastReceiptNumber, `SELECT receipt_number FROM transaction_records WHERE receipt_number LIKE ? ORDER BY receipt_number DESC LIMIT 1`, prefix+"%")
	if err != nil && err != sql.ErrNoRows {
		return 0, fmt.Errorf("伝票番号の採番に失敗: %w", err)
	}
	if lastReceiptNumber != "" && len(lastReceiptNumber) == 14 {
		seqStr := lastReceiptNumber[9:]
		lastSeq, _ = strconv.Atoi(seqStr)
	}

	// 1. CSVデータをループし、トランザクションレコードを作成
	for i, rec := range records {

		var master *model.ProductMaster
		var err error
		var keyForLog string

		// 1a. マスタ検索 (GS1コード優先、次にJANコード)
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

		// 1b. 見つからなければ仮マスター作成
		if master == nil {
			// FindOrCreateMaster に渡すキーは GS1(あれば) > JAN(なければ)
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

			// 新規作成された仮マスターのGS1コードが空の場合、CSVのGS1コードをセット
			if master.Origin == "PROVISIONAL" && master.Gs1Code == "" && rec.Gs1Code != "" {
				master.Gs1Code = rec.Gs1Code
				input := mastermanager.MasterToInput(master)
				if _, err := mastermanager.UpsertProductMasterSqlx(tx, input); err != nil {
					return 0, fmt.Errorf("仮マスターへのGS1コード更新に失敗 (JAN: %s): %w", master.ProductCode, err)
				}
			}
		}

		log.Printf("CSV Import: Matched (Key: %s) -> Master (JAN: %s, YJ: %s)", keyForLog, master.ProductCode, master.YjCode)

		// 1c. 数量を計算
		janQty := rec.JanQuantity
		yjQty := janQty * master.JanPackInnerQty // ※包装情報がなければ (0なら) YJ数量は 0 になる

		// 1d. 伝票番号
		newSeq := lastSeq + 1 + i // 1行ごとに連番
		receiptNumber := fmt.Sprintf("%s%05d", prefix, newSeq)

		tr := model.TransactionRecord{
			TransactionDate: date,
			Flag:            0, // 棚卸
			ReceiptNumber:   receiptNumber,
			LineNumber:      "1", // CSV 1行 = 1伝票・1明細として扱う
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

		// 1e. package_stock 更新用に集計
		packageKey := fmt.Sprintf("%s|%s|%g|%s", master.YjCode, master.PackageForm, master.JanPackInnerQty, master.YjUnitName)
		packageStockTotalsYj[packageKey] += yjQty
		yjCodeMap[master.YjCode] = true
	}

	if len(transactionsToInsert) == 0 {
		return 0, nil
	}

	// 2. 既存の棚卸データ(flag=0)を削除
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

	// 3. 新しい棚卸データ(flag=0)を挿入
	for _, tr := range transactionsToInsert {
		if err := database.InsertTransactionRecord(tx, tr); err != nil {
			return 0, fmt.Errorf("新しい棚卸データ(flag=0)の挿入に失敗 (JAN: %s): %w", tr.JanCode, err)
		}
	}

	// 4. package_stock を更新
	for key, totalYjQty := range packageStockTotalsYj {
		yjCode := strings.Split(key, "|")[0]
		if err := database.UpsertPackageStockInTx(tx, key, yjCode, totalYjQty, date); err != nil {
			return 0, fmt.Errorf("package_stock の更新に失敗 (Key: %s): %w", key, err)
		}
	}

	return len(transactionsToInsert), nil
}

// ▼▼▼【ここから修正】CSVエクスポートハンドラを6列形式＆手動構築に変更 ▼▼▼

// quoteAll は、csv.Writer を使わずに手動で全フィールドを引用符で囲むためのヘルパーです。
// (フィールド内の " は "" にエスケープします)
func quoteAll(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

func ExportDeadStockHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		startDate := r.URL.Query().Get("startDate") // YYYYMMDD
		endDate := r.URL.Query().Get("endDate")     // YYYYMMDD

		if startDate == "" || endDate == "" {
			http.Error(w, "startDate and endDate (YYYYMMDD) are required.", http.StatusBadRequest)
			return
		}

		// 1. 不動在庫データをDBから取得
		items, err := database.GetDeadStockList(db, startDate, endDate)
		if err != nil {
			http.Error(w, "Failed to get dead stock list: "+err.Error(), http.StatusInternalServerError)
			return
		}

		// 2. CSVバッファの準備
		var buf bytes.Buffer
		// UTF-8 BOM を書き込む (Excelでの文字化け対策)
		buf.Write([]byte{0xEF, 0xBB, 0xBF})

		// csv.Writer を使わずに手動でヘッダーを書き込む
		header := []string{
			"JANコード",
			"GS1コード",
			"品名",
			"JAN数量",
			"期限",
			"ロット",
		}
		// ヘッダーは引用符で囲まない (Excelの慣習)
		buf.WriteString(strings.Join(header, ",") + "\r\n")

		// 3. データをCSV行に変換 (手動で引用符を付与)
		for _, item := range items {
			if len(item.LotDetails) > 0 {
				// ロット明細がある場合 (棚卸履歴あり)
				for _, lot := range item.LotDetails {
					record := []string{
						quoteAll(lot.JanCode),                          // JANコード
						quoteAll(lot.Gs1Code),                          // GS1コード
						quoteAll(item.ProductName),                     // 品名 (親から取得)
						quoteAll(fmt.Sprintf("%.2f", lot.JanQuantity)), // JAN数量
						quoteAll(lot.ExpiryDate),                       // 期限
						quoteAll(lot.LotNumber),                        // ロット
					}
					buf.WriteString(strings.Join(record, ",") + "\r\n")
				}
			} else if item.StockQuantityYj > 0 {
				// ロット明細はないが、理論在庫がある場合 (棚卸履歴なし)
				record := []string{
					quoteAll(""), // JANコード (不明)
					quoteAll(""), // GS1コード (不明)
					quoteAll(item.ProductName),
					quoteAll("0.00"), // JAN数量 (不明)
					quoteAll(""),     // 期限
					quoteAll(""),     // ロット
				}
				buf.WriteString(strings.Join(record, ",") + "\r\n")
			}
			// 在庫が0の品目 (item.StockQuantityYj == 0) はCSVに出力しない
		}

		// 4. ファイルとしてダウンロードさせる
		filename := fmt.Sprintf("不動在庫リスト_%s-%s.csv", startDate, endDate)

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		// RFC 6266 (modern browsers) + URLエンコードで文字化け対策
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+url.PathEscape(filename))

		w.Write(buf.Bytes())
	}
}

// ▲▲▲【修正ここまで】▲▲▲
