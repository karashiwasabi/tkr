// C:\Users\wasab\OneDrive\デスクトップ\TKR\dat\handler.go
package dat

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"mime/multipart"
	"net/http"
	"tkr/barcode"
	"tkr/database"
	"tkr/mappers"
	"tkr/mastermanager"
	"tkr/model"
	"tkr/parsers"

	"github.com/jmoiron/sqlx"
)

func respondJSONError(w http.ResponseWriter, message string, statusCode int) {
	log.Println("Error response:", message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": message,
		"results": []interface{}{},
	})
}

func UploadDatHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received DAT upload request...")

		err := r.ParseMultipartForm(32 << 20)
		if err != nil {
			respondJSONError(w, "File upload error: "+err.Error(), http.StatusBadRequest)
			return
		}
		defer r.MultipartForm.RemoveAll()
		var processedFiles []string
		var successfullyInsertedTransactions []model.TransactionRecord
		var allResults []map[string]interface{}
		for _, fileHeader := range r.MultipartForm.File["file"] {
			log.Printf("Processing file: %s", fileHeader.Filename)
			processedFiles = append(processedFiles, fileHeader.Filename)
			fileResult := map[string]interface{}{"filename": fileHeader.Filename}
			file, openErr := fileHeader.Open()
			if openErr != nil {
				log.Printf("Failed to open uploaded file %s: %v", fileHeader.Filename, openErr)
				fileResult["error"] = fmt.Sprintf("Failed to open file: %v", openErr)
				allResults =
					append(allResults, fileResult)
				continue
			}
			// ▼▼▼【修正】TKRのパーサー(バイト列処理)を呼び出す ▼▼▼
			parsedRecords, parseErr := parsers.ParseDat(file)
			// ▲▲▲【修正ここまで】▲▲▲
			file.Close()
			if parseErr != nil {
				log.Printf("Failed to parse DAT file %s: %v", fileHeader.Filename, parseErr)
				fileResult["error"] = fmt.Sprintf("Failed to parse DAT file: %v", parseErr)
				allResults = append(allResults, fileResult)
				continue
			}
			log.Printf("Parsed %d records from %s", len(parsedRecords), fileHeader.Filename)
			fileResult["records_parsed"] = len(parsedRecords)

			tx, txErr := db.Beginx()
			if txErr != nil {
				log.Printf("Failed to start transaction for %s: %v", fileHeader.Filename, txErr)
				fileResult["error"] = fmt.Sprintf("Failed to start transaction: %v", txErr)
				allResults = append(allResults, fileResult)
				continue
			}

			insertedTransactions, processErr := ProcessDatRecords(tx, parsedRecords)
			if processErr != nil {
				log.Printf("Failed to process DAT records for %s: %v", fileHeader.Filename, processErr)
				fileResult["error"] = fmt.Sprintf("Failed to process records: %v", processErr)
				allResults = append(allResults, fileResult)
				tx.Rollback()
				continue
			}

			if commitErr := tx.Commit(); commitErr != nil {
				log.Printf("Failed to commit transaction for %s: %v", fileHeader.Filename, commitErr)
				fileResult["error"] = fmt.Sprintf("Failed to commit transaction: %v", commitErr)
				allResults = append(allResults, fileResult)
				continue
			}

			successfullyInsertedTransactions = append(successfullyInsertedTransactions, insertedTransactions...)
			log.Printf("Successfully inserted %d records from %s", len(insertedTransactions), fileHeader.Filename)
			fileResult["success"] = true
			fileResult["records_inserted"] = len(insertedTransactions)
			allResults = append(allResults, fileResult)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message": fmt.Sprintf("Processed %d DAT file(s). See results for details.", len(processedFiles)),
			"results": allResults,
			"records": successfullyInsertedTransactions,
		})
		log.Println("Finished DAT upload request.")
	}
}

// ▼▼▼【ここから追加】重複除去ロジック (WASABI: dat/handler.go  より) ▼▼▼
func removeDatDuplicates(records []model.DatRecord) []model.DatRecord {
	seen := make(map[string]struct{})
	var result []model.DatRecord
	for _, r := range records {
		// WASABIのキー定義 に基づき、TKRのmodel.DatRecord  のフィールドでキーを作成
		key := fmt.Sprintf("%s|%s|%s|%s", r.Date, r.ClientCode, r.ReceiptNumber, r.LineNumber)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, r)
	}
	return result
}

// ▲▲▲【追加ここまで】▲▲▲

func ProcessDatRecords(tx *sqlx.Tx, parsedRecords []model.DatRecord) ([]model.TransactionRecord, error) {
	var insertedTransactions []model.TransactionRecord
	var insertedCount int = 0

	// ▼▼▼【ここに追加】重複除去を実行 ▼▼▼
	recordsToProcess := removeDatDuplicates(parsedRecords)
	// ▲▲▲【追加ここまで】▲▲▲

	// ▼▼▼【ここから修正】ループ対象を parsedRecords -> recordsToProcess に変更 ▼▼▼
	for _, rec := range recordsToProcess {
		// ▲▲▲【修正ここまで】▲▲▲
		key := rec.JanCode

		// ▼▼▼【ここから修正】「キーが000...」の分岐を削除し、FindOrCreateMaster呼び出し後にOriginをチェックするロジックに変更 ▼▼▼
		/*
					if key == "" ||
			key == "0000000000000" {
						// ユーザーの要求: DATファイルの商品名をプロダクトマスタのkana_name_shortに照合
						// なければ、DAT商品名をプロダクトマスタの商品名とkana_name_shortに記載
						foundMaster, err := database.GetProductMasterByKanaNameShort(tx, rec.ProductName)
						if err != nil && !errors.Is(err, sql.ErrNoRows) {
							return nil, fmt.Errorf("failed to search product master by kana_name_short for %s: %w", rec.ProductName, err)
						}

						if foundMaster != nil {
							// 照合できた場合、そのマスターのProductCodeを使用
							key = foundMaster.ProductCode
							log.Printf("Found existing master by kana_name_short for '%s', using ProductCode: %s", rec.ProductName, key)

							// ▼▼▼【ここから修正】▼▼▼
							// もし見つかったマスターがJCSHMS由来でなく (PROVISIONALなど)、
							// かつ kana_name_short がDATの製品名と異なる場合、DATの製品名で更新する。
							if foundMaster.Origin != "JCSHMS" && foundMaster.KanaNameShort != rec.ProductName {
								log.Printf("Updating non-JCSHMS master (ProductCode: %s) KanaNameShort from '%s' to '%s'",
									foundMaster.ProductCode, foundMaster.KanaNameShort, rec.ProductName)

								input := mastermanager.MasterToInput(foundMaster)
								input.KanaNameShort = rec.ProductName // DATの製品名で上書き

								updatedMaster, upsertErr := mastermanager.UpsertProductMasterSqlx(tx, input)
								if upsertErr != nil {
									return nil, fmt.Errorf("failed to update kana_name_short for existing master %s: %w", foundMaster.ProductCode, upsertErr)
								}
								*foundMaster = *updatedMaster // 元のポインタが指す中身を更新
							}
							// ▲▲▲【修正ここまで】▲▲▲

						} else {
							// ▼▼▼【ここを修正】合成キー(999...)を生成せず、元のキー(0x13)をそのまま使う ▼▼▼
							key = rec.JanCode // ( "0000000000000" または "" が key になる)
							log.Printf("No master found by kana_name_short for '%s', using original key: %s", rec.ProductName, key)
						}
					}
		*/

		master, err := mastermanager.FindOrCreateMaster(tx, key, rec.ProductName)
		if err != nil {
			return nil, fmt.Errorf("master creation failed for key %s: %w", key, err)
		}

		// FindOrCreateMaster の結果、マスターがJCSHMS由来でなかった場合、
		// DATファイルに記載の製品名 (rec.ProductName) を kana_name_short に設定（上書き）する。
		if master.Origin != "JCSHMS" && master.KanaNameShort != rec.ProductName {
			// (キーが "000..." の場合は FindOrCreateMaster 内部 で設定済みの可能性もあるが、
			// 「有効なJANだがJCSHMSになく仮マスター作成」や「手動登録マスター」の場合、
			// このブロックで kana_name_short がDAT製品名で更新される)

			log.Printf("Updating non-JCSHMS master (ProductCode: %s) KanaNameShort from '%s' to '%s'",
				master.ProductCode, master.KanaNameShort, rec.ProductName)

			input := mastermanager.MasterToInput(master)
			input.KanaNameShort = rec.ProductName // DATの製品名で上書き

			updatedMaster, upsertErr := mastermanager.UpsertProductMasterSqlx(tx, input)
			if upsertErr != nil {
				return nil, fmt.Errorf("failed to update kana_name_short for existing master %s: %w", master.ProductCode, upsertErr)
			}
			*master = *updatedMaster // ポインタが指す中身を更新
		}
		// ▲▲▲【修正ここまで】▲▲▲

		transaction := model.TransactionRecord{
			TransactionDate: rec.Date,
			ClientCode:      rec.ClientCode,
			ReceiptNumber:   rec.ReceiptNumber,
			LineNumber:      rec.LineNumber,
			Flag:            rec.Flag,
			DatQuantity:     rec.DatQuantity,
			UnitPrice:       rec.UnitPrice,
			Subtotal:        rec.Subtotal,
			ExpiryDate:      rec.ExpiryDate,
			LotNumber:       rec.LotNumber,
		}

		// ▼▼▼【ここから修正】数量計算のバグを修正 ▼▼▼
		// 1. JanQuantity は DAT数量(rec.DatQuantity) と等価
		transaction.JanQuantity = rec.DatQuantity

		// 2. YjQuantity は (DAT数量 * JAN包装内入数) で計算
		// (もし JanPackInnerQty が 0 なら YjQuantity も 0 になるが、それはマスタ設定の問題)
		transaction.YjQuantity = rec.DatQuantity * master.JanPackInnerQty

		if transaction.UnitPrice == 0 {
			transaction.UnitPrice = master.NhiPrice
			// ▼▼▼【ここを修正】金額計算も YjQuantity を基準にする ▼▼▼
			transaction.Subtotal = transaction.YjQuantity * transaction.UnitPrice
		}

		mappers.MapMasterToTransaction(&transaction,
			master)

		if err := database.InsertTransactionRecord(tx, transaction); err != nil {
			return nil, fmt.Errorf("transaction insert failed for key %s: %w", key, err)
		}
		insertedTransactions = append(insertedTransactions, transaction)
		insertedCount++
	}
	return insertedTransactions, nil
}

func SearchDatHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondJSONError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		barcodeStr := r.URL.Query().Get("barcode")
		log.Printf("Received DAT search request... Barcode: [%s]", barcodeStr)

		if barcodeStr == "" {
			respondJSONError(w, "バーコードを入力してください。", http.StatusBadRequest)
			return
		}

		master, err := database.GetProductMasterByBarcode(db, barcodeStr)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Printf("Product master not found for Barcode: %s", barcodeStr)
				respondJSONError(w, "バーコードに対応するマスターが見つかりません。", http.StatusNotFound)
			} else {
				log.Printf("Error searching product master by Barcode %s: %v", barcodeStr, err)
				respondJSONError(w, fmt.Sprintf("マスター検索エラー: %v", err), http.StatusInternalServerError)
			}
			return
		}

		productCode := master.ProductCode
		var expiryYYMMDD, expiryYYMM, lotNumber string
		if len(barcodeStr) > 14 {
			gs1Result,
				parseErr := barcode.Parse(barcodeStr)
			if parseErr == nil && gs1Result != nil {
				expiryYYMMDD = gs1Result.ExpiryDate
				if len(expiryYYMMDD) == 6 {
					expiryYYMM = expiryYYMMDD[:4]
				}
				lotNumber = gs1Result.LotNumber
			}
		}

		log.Printf("Search criteria: ProductCode(JAN)='%s', Expiry(6)='%s', Expiry(4)='%s', Lot='%s'",
			productCode, expiryYYMMDD, expiryYYMM, lotNumber)

		transactions, err := database.SearchTransactions(db, productCode, expiryYYMMDD, expiryYYMM, lotNumber)

		if err != nil {
			log.Printf("Error searching transactions: %v", err)
			respondJSONError(w, "トランザクション検索中にエラーが発生しました。", http.StatusInternalServerError)
			return
		}

		log.Printf("Found %d transactions matching criteria.", len(transactions))

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":      fmt.Sprintf("%d 件のデータが見つかりました。", len(transactions)),
			"transactions": transactions,
		})
	}
}

func OpenFileHeader(fh *multipart.FileHeader) (multipart.File, error) {
	return fh.Open()
}
