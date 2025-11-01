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
	"tkr/database"
	"tkr/mastermanager"
	"tkr/model"
	"tkr/parsers"
	"tkr/render" // ▼▼▼【ここに追加】▼▼▼
	"tkr/units"

	"github.com/jmoiron/sqlx"
)

// ▼▼▼【修正】render.RenderTransactionTableHTML を呼び出す ▼▼▼
func respondJSONError(w http.ResponseWriter, message string, statusCode int) {
	log.Println("Error response:", message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":   message,
		"results":   []interface{}{},
		"tableHTML": render.RenderTransactionTableHTML(nil, nil), // ★ 呼び出し先を変更
	})
}

// ▲▲▲【修正ここまで】▲▲▲

func UploadDatHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		log.Println("Received DAT upload request...")

		// ★ 先に卸マスターを取得・マップ化
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

			file, openErr := fileHeader.Open()
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
				if key == "" || key == "0000000000000" {
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
				if err := database.InsertTransactionRecord(tx, transaction); err != nil {
					log.Printf("Failed to insert transaction record for key %s (Product: %s) in file %s: %v", key, rec.ProductName, fileHeader.Filename, err)
					fileResult["error"] = fmt.Sprintf("Transaction insert failed for key %s: %v", key, err)
					allResults = append(allResults, fileResult)
					tx.Rollback()
					goto nextFileLoop
				}

				currentFileTransactions = append(currentFileTransactions, transaction)
				insertedCount++
			}

			if commitErr := tx.Commit(); commitErr != nil {
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
		// ▼▼▼【修正】render.RenderTransactionTableHTML を呼び出す ▼▼▼
		htmlString := render.RenderTransactionTableHTML(successfullyInsertedTransactions, wholesalerMap)
		// ▲▲▲【修正ここまで】▲▲▲

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

		// ★ 先に卸マスターを取得・マップ化
		wholesalerMap, err := database.GetWholesalerMap(db)
		if err != nil {
			respondJSONError(w, "卸マスターの読み込みに失敗しました。", http.StatusInternalServerError)
			return
		}

		barcode := r.URL.Query().Get("barcode")
		log.Printf("Received DAT search request... Barcode: [%s]", barcode)

		if barcode == "" {
			respondJSONError(w, "バーコードを入力してください。", http.StatusBadRequest)
			return
		}

		gs1Result, parseErr := parsers.ParseGS1_128(barcode)
		if parseErr != nil {
			log.Printf("Error parsing GS1-128 barcode [%s]: %v", barcode, parseErr)
			respondJSONError(w, fmt.Sprintf("バーコード解析エラー: %v", parseErr), http.StatusBadRequest)
			return
		}

		gtin14 := gs1Result.JanCode
		if gtin14 == "" {
			respondJSONError(w, "バーコードから(01)GTINが抽出できませんでした。", http.StatusBadRequest)
			return
		}

		master, err := database.GetProductMasterByGs1Code(db, gtin14)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				log.Printf("Product master not found for GS1_CODE: %s", gtin14)
				respondJSONError(w, "GS1コードに対応するマスターが見つかりません。", http.StatusNotFound)
			} else {
				log.Printf("Error searching product master by GS1_CODE %s: %v", gtin14, err)
				respondJSONError(w, "マスター検索中にデータベースエラーが発生しました。", http.StatusInternalServerError)
			}
			return
		}

		productCode := master.ProductCode

		expiryYYMMDD := gs1Result.ExpiryDate
		expiryYYMM := ""
		if len(expiryYYMMDD) == 6 {
			expiryYYMM = expiryYYMMDD[:4]
		}

		log.Printf("Parsed GS1: GTIN(14)='%s' -> ProductCode(JAN)='%s', Expiry(6)='%s', Expiry(4)='%s', Lot='%s'",
			gtin14, productCode, expiryYYMMDD, expiryYYMM, gs1Result.LotNumber)

		transactions, err := database.SearchTransactions(db, productCode, expiryYYMMDD, expiryYYMM, gs1Result.LotNumber)

		if err != nil {
			log.Printf("Error searching transactions: %v", err)
			respondJSONError(w, "トランザクション検索中にエラーが発生しました。", http.StatusInternalServerError)
			return
		}

		log.Printf("Found %d transactions matching criteria.", len(transactions))

		// ▼▼▼【修正】render.RenderTransactionTableHTML を呼び出す ▼▼▼
		htmlString := render.RenderTransactionTableHTML(transactions, wholesalerMap)
		// ▲▲▲【修正ここまで】▲▲▲

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   fmt.Sprintf("%d 件のデータが見つかりました。", len(transactions)),
			"tableHTML": htmlString,
		})
	}
}

func MapDatToTransaction(dat model.DatRecord, master *model.ProductMaster) model.TransactionRecord {
	// 1. 製品名と規格を連結
	productNameWithSpec := master.ProductName
	if master.Specification != "" {
		productNameWithSpec = master.ProductName + " " + master.Specification
	}

	// 2. 数量の計算 (DATの個数から計算)
	var yjQuantity, janQuantity float64
	if master.YjPackUnitQty > 0 {
		yjQuantity = dat.DatQuantity * master.YjPackUnitQty
	}
	if master.JanPackUnitQty > 0 {
		janQuantity = dat.DatQuantity * master.JanPackUnitQty
	}

	// 3. 詳細な包装仕様を生成 (units.ResolveName を使用)
	yjUnitName := units.ResolveName(master.YjUnitName)
	packageSpec := fmt.Sprintf("%s %g%s", master.PackageForm, master.YjPackUnitQty, yjUnitName)

	janUnitCodeStr := fmt.Sprintf("%d", master.JanUnitCode)
	var janUnitName string

	if master.JanUnitCode == 0 {
		janUnitName = yjUnitName
	} else {
		janUnitName = units.ResolveName(janUnitCodeStr)
	}

	if master.JanPackInnerQty > 0 && master.JanPackUnitQty > 0 {
		packageSpec += fmt.Sprintf(" (%g%s×%g%s)",
			master.JanPackInnerQty,
			yjUnitName,
			master.JanPackUnitQty,
			janUnitName,
		)
	}

	// 4. MAフラグの設定
	var processFlagMA string
	if master.Origin == "JCSHMS" {
		processFlagMA = "COMPLETE"
	} else {
		processFlagMA = "PROVISIONAL"
	}

	return model.TransactionRecord{
		TransactionDate:     dat.Date,
		ClientCode:          dat.ClientCode,
		ReceiptNumber:       dat.ReceiptNumber,
		LineNumber:          dat.LineNumber,
		Flag:                dat.Flag,
		JanCode:             master.ProductCode,
		YjCode:              master.YjCode,
		ProductName:         productNameWithSpec,
		KanaName:            master.KanaName,
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
		YjPackUnitQty:       master.YjPackUnitQty,
		YjUnitName:          yjUnitName,
		UnitPrice:           dat.UnitPrice,
		PurchasePrice:       master.PurchasePrice,
		SupplierWholesale:   master.SupplierWholesale,
		Subtotal:            dat.Subtotal,
		TaxAmount:           0,
		TaxRate:             0,
		ExpiryDate:          dat.ExpiryDate,
		LotNumber:           dat.LotNumber,
		FlagPoison:          master.FlagPoison,
		FlagDeleterious:     master.FlagDeleterious,
		FlagNarcotic:        master.FlagNarcotic,
		FlagPsychotropic:    master.FlagPsychotropic,
		FlagStimulant:       master.FlagStimulant,
		FlagStimulantRaw:    master.FlagStimulantRaw,
		ProcessFlagMA:       processFlagMA,
	}
}

func OpenFileHeader(fh *multipart.FileHeader) (multipart.File, error) {
	return fh.Open()
}

// ▼▼▼【削除】RenderTransactionTableHTML 関数を削除 (render パッケージに移動したため) ▼▼▼
// ( ... 関数の定義 ... )
// ▲▲▲【削除ここまで】▲▲▲
