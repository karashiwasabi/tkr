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
	"tkr/mastermanager"
	"tkr/model"
	"tkr/parsers"
	"tkr/render"
	"tkr/units"

	"github.com/jmoiron/sqlx"
)

// (respondJSONError, UploadDatHandler は変更なし)
// ...
func respondJSONError(w http.ResponseWriter, message string, statusCode int) {
	log.Println("Error response:", message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   message,
		"results":   []interface{}{},
		"tableHTML": render.RenderTransactionTableHTML(nil, nil),
	})
}

func UploadDatHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received DAT upload request...")
		wholesalerMap, err := database.GetWholesalerMap(db)
		if err != nil {
			respondJSONError(w, "卸マスターの読み込みに失敗しました。", http.StatusInternalServerError)
			return
		}
		err = r.ParseMultipartForm(32 << 20)
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
			var currentFileTransactions []model.TransactionRecord
			// ▼▼▼【ここを修正】`` タグを削除 ▼▼▼
			file, openErr := fileHeader.Open()
			// ▲▲▲【修正ここまで】▲▲▲
			if openErr != nil {
				log.Printf("Failed to open uploaded file %s: %v", fileHeader.Filename, openErr)
				fileResult["error"] = fmt.Sprintf("Failed to open file: %v", openErr)
				allResults = append(allResults, fileResult)
				continue
			}
			parsedRecords, parseErr := parsers.ParseDat(file)
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
			var insertedCount int = 0
			for _, rec := range parsedRecords {
				key := rec.JanCode
				if key == "" ||
					// ▼▼▼【ここを修正】`` タグを削除 ▼▼▼
					key == "0000000000000" {
					// ▲▲▲【修正ここまで】▲▲▲
					key = fmt.Sprintf("9999999999999%s", rec.ProductName)
				}
				master, err := mastermanager.FindOrCreateMaster(tx, key, rec.ProductName)
				if err != nil {
					log.Printf("Failed to find or create master for key %s (Product: %s) in file %s: %v", key, rec.ProductName, fileHeader.Filename, err)
					fileResult["error"] = fmt.Sprintf("Master creation failed for key %s: %v", key, err)
					allResults = append(allResults, fileResult)
					tx.Rollback()
					goto nextFileLoop
				}
				transaction := MapDatToTransaction(rec, master)
				// ▼▼▼【ここを修正】`` タグを削除 ▼▼▼
				if err := database.InsertTransactionRecord(tx, transaction); err != nil {
					// ▲▲▲【修正ここまで】▲▲▲
					log.Printf("Failed to insert transaction record for key %s (Product: %s) in file %s: %v", key, rec.ProductName, fileHeader.Filename, err)
					fileResult["error"] = fmt.Sprintf("Transaction insert failed for key %s: %v", key, err)
					allResults = append(allResults, fileResult)
					tx.Rollback()
					goto nextFileLoop
				}
				currentFileTransactions = append(currentFileTransactions, transaction)
				insertedCount++
			}
			// ▼▼▼【ここを修正】`` タグを削除 ▼▼▼
			if commitErr := tx.Commit(); commitErr != nil {
				// ▲▲▲【修正ここまで】▲▲▲
				log.Printf("Failed to commit transaction for %s: %v", fileHeader.Filename, commitErr)
				fileResult["error"] = fmt.Sprintf("Failed to commit transaction: %v", commitErr)
				allResults = append(allResults, fileResult)
				goto nextFileLoop
			}
			successfullyInsertedTransactions = append(successfullyInsertedTransactions, currentFileTransactions...)
			log.Printf("Successfully inserted %d records from %s", insertedCount, fileHeader.Filename)
			fileResult["success"] = true
			fileResult["records_inserted"] = insertedCount
			allResults = append(allResults, fileResult)
		nextFileLoop:
			continue
		}
		w.Header().Set("Content-Type", "application/json")
		htmlString := render.RenderTransactionTableHTML(successfullyInsertedTransactions, wholesalerMap)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   fmt.Sprintf("Processed %d DAT file(s). See results for details.", len(processedFiles)),
			"results":   allResults,
			"tableHTML": htmlString,
		})
		log.Println("Finished DAT upload request.")
	}
}

func SearchDatHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			respondJSONError(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			return
		}

		wholesalerMap, err := database.GetWholesalerMap(db)
		if err != nil {
			respondJSONError(w, "卸マスターの読み込みに失敗しました。", http.StatusInternalServerError)
			return
		}

		// ▼▼▼【ここから修正】DB共通関数 GetProductMasterByBarcode を使用 ▼▼▼
		barcodeStr := r.URL.Query().Get("barcode")
		// ▼▼▼【ここを修正】改行と `` タグを削除 ▼▼▼
		log.Printf("Received DAT search request... Barcode: [%s]", barcodeStr)
		// ▲▲▲【修正ここまで】▲▲▲

		if barcodeStr == "" {
			respondJSONError(w, "バーコードを入力してください。", http.StatusBadRequest)
			return
		}

		// DB共通関数でマスターを検索
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

		// 期限とロットはGS1の場合のみ取得
		var expiryYYMMDD, expiryYYMM, lotNumber string
		if len(barcodeStr) > 14 { // 15桁以上の場合のみGS1解析
			gs1Result, parseErr := barcode.Parse(barcodeStr)
			if parseErr == nil && gs1Result != nil { // エラーは無視
				expiryYYMMDD = gs1Result.ExpiryDate
				if len(expiryYYMMDD) == 6 {
					expiryYYMM = expiryYYMMDD[:4]
				}
				lotNumber = gs1Result.LotNumber
			}
		}

		log.Printf("Search criteria: ProductCode(JAN)='%s', Expiry(6)='%s', Expiry(4)='%s', Lot='%s'",
			productCode, expiryYYMMDD, expiryYYMM, lotNumber)

		// ▼▼▼【ここを修正】`` タグを削除 ▼▼▼
		transactions, err := database.SearchTransactions(db, productCode, expiryYYMMDD, expiryYYMM, lotNumber)
		// ▲▲▲【修正ここまで】▲▲▲

		if err != nil {
			// ▼▼▼【ここを修正】`` タグを削除 ▼▼▼
			log.Printf("Error searching transactions: %v", err)
			// ▲▲▲【修正ここまで】▲▲▲
			respondJSONError(w, "トランザクション検索中にエラーが発生しました。", http.StatusInternalServerError)
			return
		}

		log.Printf("Found %d transactions matching criteria.", len(transactions))

		htmlString := render.RenderTransactionTableHTML(transactions, wholesalerMap)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   fmt.Sprintf("%d 件のデータが見つかりました。", len(transactions)),
			"tableHTML": htmlString,
		})
	}
}

// (MapDatToTransaction, OpenFileHeader は変更なし)
// ...
func MapDatToTransaction(dat model.DatRecord, master *model.ProductMaster) model.TransactionRecord {
	productNameWithSpec := master.ProductName
	if master.Specification != "" {
		productNameWithSpec = master.ProductName + " " + master.Specification
	}
	var yjQuantity, janQuantity float64
	if master.YjPackUnitQty > 0 {
		yjQuantity = dat.DatQuantity * master.YjPackUnitQty
	}
	if master.JanPackUnitQty > 0 {
		janQuantity = dat.DatQuantity * master.JanPackUnitQty
	}
	yjUnitName := units.ResolveName(master.YjUnitName)
	packageSpec := fmt.Sprintf("%s %g%s", master.PackageForm, master.YjPackUnitQty, yjUnitName)
	janUnitCodeStr := fmt.Sprintf("%d", master.JanUnitCode)
	var janUnitName string
	if master.JanUnitCode == 0 {
		janUnitName = yjUnitName
	} else {
		janUnitName = units.ResolveName(janUnitCodeStr)
	}
	// ▼▼▼【ここを修正】if と { を同じ行にする ▼▼▼
	if master.JanPackInnerQty > 0 && master.JanPackUnitQty > 0 {
		// ▲▲▲【修正ここまで】▲▲▲
		packageSpec += fmt.Sprintf(" (%g%s×%g%s)",
			master.JanPackInnerQty,
			yjUnitName,
			master.JanPackUnitQty,
			janUnitName,
		)
	}
	var processFlagMA string
	// ▼▼▼【ここを修正】if と { を同じ行にする ▼▼▼
	if master.Origin == "JCSHMS" {
		// ▲▲▲【修正ここまで】▲▲▲
		processFlagMA = "COM"
	} else {
		processFlagMA = "PRO"
	}
	return model.TransactionRecord{
		TransactionDate: dat.Date,
		ClientCode:      dat.ClientCode,
		ReceiptNumber:   dat.ReceiptNumber,
		LineNumber:      dat.LineNumber,
		Flag:            dat.Flag,
		JanCode:         master.ProductCode,
		YjCode:          master.YjCode,
		// ▼▼▼【ここを修正】`` タグを削除 ▼▼▼
		ProductName: productNameWithSpec,
		KanaName:    master.KanaName,
		// ▲▲▲【修正ここまで】▲▲▲
		UsageClassification: master.UsageClassification,
		PackageForm:         master.PackageForm,
		PackageSpec:         packageSpec,
		MakerName:           master.MakerName,
		DatQuantity:         dat.DatQuantity,
		JanPackInnerQty:     master.JanPackInnerQty,
		JanQuantity:         janQuantity,
		JanPackUnitQty:      master.JanPackUnitQty,
		JanUnitName:         janUnitName,
		JanUnitCode:         janUnitCodeStr,
		YjQuantity:          yjQuantity,
		// ▼▼▼【ここを修正】`` タグを削除 ▼▼▼
		YjPackUnitQty: master.YjPackUnitQty,
		YjUnitName:    yjUnitName,
		// ▲▲▲【修正ここまで】▲▲▲
		UnitPrice:         dat.UnitPrice,
		PurchasePrice:     master.PurchasePrice,
		SupplierWholesale: master.SupplierWholesale,
		Subtotal:          dat.Subtotal,
		TaxAmount:         0,
		TaxRate:           0,
		ExpiryDate:        dat.ExpiryDate,
		LotNumber:         dat.LotNumber,
		// ▼▼▼【ここを修正】`` タグを削除 ▼▼▼
		FlagPoison:      master.FlagPoison,
		FlagDeleterious: master.FlagDeleterious,
		// ▲▲▲【修正ここまで】▲▲▲
		FlagNarcotic:     master.FlagNarcotic,
		FlagPsychotropic: master.FlagPsychotropic,
		FlagStimulant:    master.FlagStimulant,
		FlagStimulantRaw: master.FlagStimulantRaw,
		ProcessFlagMA:    processFlagMA,
	}
}

func OpenFileHeader(fh *multipart.FileHeader) (multipart.File, error) {
	return fh.Open()
}
