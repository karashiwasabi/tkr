// C:\Users\wasab\OneDrive\デスクトップ\TKR\usage\handler.go
package usage

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"strconv"
	"strings"
	"tkr/config" // TKRのconfig

	// ▼▼▼【修正】dat -> render ▼▼▼
	"tkr/render" // (RenderTransactionTableHTMLのため)
	// ▲▲▲【修正ここまで】▲▲▲
	"tkr/database"
	"tkr/mastermanager"
	"tkr/model"
	"tkr/parsers"
	"tkr/units" // TKRのunits

	"github.com/jmoiron/sqlx"
)

// respondJSONError はTKRのDATハンドラ に倣ったエラー応答関数です。
func respondJSONError(w http.ResponseWriter, message string, statusCode int) {
	log.Println("Error response:", message)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message": message,
		"results": []interface{}{},
		// ▼▼▼【修正】renderパッケージの関数を呼ぶ ▼▼▼
		"tableHTML": render.RenderTransactionTableHTML(nil, nil),
		// ▲▲▲【修正ここまで】▲▲▲
	})
}

// UploadUsageHandler は自動または手動でのUSAGEファイルアップロードを処理します。
func UploadUsageHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var file io.Reader
		var err error

		// TKRでは手動アップロード（multipart/form-data）のみをまず実装します。
		// 自動取込（POST）ロジックはWASABI から移植します。

		if strings.Contains(r.Header.Get("Content-Type"), "multipart/form-data") {
			// 手動アップロードの場合
			log.Println("Processing manual USAGE file upload...")
			var f multipart.File
			f, _, err = r.FormFile("file")
			if err != nil {
				respondJSONError(w, "ファイルの取得に失敗しました: "+err.Error(), http.StatusBadRequest)
				return
			}
			defer f.Close()
			file = f
		} else {
			// 自動取込（Content-TypeなしのPOST）の場合
			log.Println("Processing automatic USAGE file import...")
			cfg, cfgErr := config.LoadConfig()
			if cfgErr != nil {
				respondJSONError(w, "設定ファイルの読み込みに失敗: "+cfgErr.Error(), http.StatusInternalServerError)
				return
			}
			if cfg.UsageFolderPath == "" {
				respondJSONError(w, "処方取込フォルダパス(usageFolderPath)が設定されていません。", http.StatusBadRequest)
				return
			}

			rawPath := cfg.UsageFolderPath
			// WASABI と同様にパスをクリーンアップ
			unquotedPath := strings.Trim(strings.TrimSpace(rawPath), "\"")
			filePath := strings.ReplaceAll(unquotedPath, "\\", "/")

			log.Printf("Opening specified USAGE file: %s", filePath)
			f, fErr := os.Open(filePath)
			if fErr != nil {
				displayError := fmt.Sprintf("設定されたパスのファイルを開けませんでした。\nパス: %s\nエラー: %v", filePath, fErr)
				respondJSONError(w, displayError, http.StatusInternalServerError)
				return
			}
			defer f.Close()
			file = f
		}

		// ▼▼▼【ここから修正】卸マップ取得とHTML生成を追加 ▼▼▼

		// 卸マスターを取得・マップ化 (HTML描画用)
		// (処方データは卸と関係ないが、RenderTransactionTableHTMLが引数を取るため nil マップでなく空マップを渡す)
		wholesalerMap, err := database.GetWholesalerMap(db)
		if err != nil {
			respondJSONError(w, "卸マスターの読み込みに失敗しました。", http.StatusInternalServerError)
			return
		}

		// 共通の処理関数を呼び出す
		processedRecords, procErr := processUsageFile(db, file)
		if procErr != nil {
			respondJSONError(w, procErr.Error(), http.StatusInternalServerError)
			return
		}

		// TKRのDATハンドラと同様に、HTMLテーブルを生成
		// ▼▼▼【修正】render.RenderTransactionTableHTML を呼び出す ▼▼▼
		htmlString := render.RenderTransactionTableHTML(processedRecords, wholesalerMap)
		// ▲▲▲【修正ここまで】▲▲▲

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"message":   fmt.Sprintf("%d件の処方データを処理しました。", len(processedRecords)),
			"results":   []interface{}{}, // DATハンドラに合わせる
			"tableHTML": htmlString,      // HTMLをレスポンスに含める
		})
		// ▲▲▲【修正ここまで】▲▲▲
	}
}

// processUsageFile はファイルストリームから処方データを解析しDBに登録する共通関数です。
func processUsageFile(db *sqlx.DB, file io.Reader) ([]model.TransactionRecord, error) {
	// 1. CSVパース (TKR/parsers/usage_parser.go)
	parsed, err := parsers.ParseUsage(file)
	if err != nil {
		return nil, fmt.Errorf("USAGEファイルの解析に失敗しました: %w", err)
	}

	// 2. 重複除去 (WASABI のロジック)
	filtered := removeUsageDuplicates(parsed)
	if len(filtered) == 0 {
		return []model.TransactionRecord{}, nil
	}

	// 3. トランザクション開始 (TKR流)
	tx, err := db.Beginx()
	if err != nil {
		return nil, fmt.Errorf("トランザクションの開始に失敗: %w", err)
	}
	defer tx.Rollback() // エラー時は自動ロールバック

	// 4. 既存データ削除 (WASABI のロジック)
	minDate, maxDate := "99999999", "00000000"
	for _, rec := range filtered {
		if rec.Date < minDate {
			minDate = rec.Date
		}
		if rec.Date > maxDate {
			maxDate = rec.Date
		}
	}

	if err := database.DeleteUsageTransactionsInDateRange(tx, minDate, maxDate); err != nil {
		return nil, fmt.Errorf("既存の処方データ削除に失敗: %w", err)
	}

	// 5. マスター特定とデータ挿入
	var finalRecords []model.TransactionRecord
	for i, rec := range filtered {
		// 5a. マスター特定 (TKR/mastermanager)
		key := rec.JanCode
		if key == "" || key == "0000000000000" {
			// JANがない場合はYJコードをキーにする (TKRのmastermanagerはYJもJANも扱える)
			key = rec.YjCode
			if key == "" {
				// 両方ない場合は製品名で合成キー
				key = fmt.Sprintf("9999999999999%s", rec.ProductName)
			}
		}

		master, err := mastermanager.FindOrCreateMaster(tx, key, rec.ProductName)
		if err != nil {
			return nil, fmt.Errorf("マスターの特定/作成に失敗 (Key: %s, Name: %s): %w", key, rec.ProductName, err)
		}

		// 5b. TransactionRecordへのマッピング (TKR/dat/handler.go を参考に)
		transaction := MapUsageToTransaction(rec, master, i+1)

		// 5c. DB挿入 (TKR/database)
		if err := database.InsertTransactionRecord(tx, transaction); err != nil {
			return nil, fmt.Errorf("処方レコードの挿入に失敗 (JAN: %s): %w", transaction.JanCode, err)
		}

		finalRecords = append(finalRecords, transaction)
	}

	// 6. コミット
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("トランザクションのコミットに失敗: %w", err)
	}

	return finalRecords, nil
}

// MapUsageToTransaction は、パースした処方データとマスターからDB用のTransactionRecordを作成します。
// TKRの `dat.MapDatToTransaction` を参考にします。
func MapUsageToTransaction(rec model.UnifiedInputRecord, master *model.ProductMaster, lineNumber int) model.TransactionRecord {
	// 1. 製品名と規格を連結
	productNameWithSpec := master.ProductName
	if master.Specification != "" {
		productNameWithSpec = master.ProductName + " " + master.Specification
	}

	// 2. 数量の計算 (USAGEはYJ数量が直接入ってくる)
	yjQuantity := rec.YjQuantity
	var janQuantity float64
	if master.JanPackInnerQty > 0 {
		janQuantity = yjQuantity / master.JanPackInnerQty
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

	// 5. 単価と金額 (処方の場合は薬価 [NhiPrice] を使用)
	unitPrice := master.NhiPrice
	subtotal := yjQuantity * unitPrice

	return model.TransactionRecord{
		TransactionDate:     rec.Date,
		ClientCode:          "", // 処方では得意先コードは不要
		ReceiptNumber:       fmt.Sprintf("USAGE-%s", rec.Date),
		LineNumber:          strconv.Itoa(lineNumber),
		Flag:                3, // 処方
		JanCode:             master.ProductCode,
		YjCode:              master.YjCode,
		ProductName:         productNameWithSpec,
		KanaName:            master.KanaName,
		UsageClassification: master.UsageClassification,
		PackageForm:         master.PackageForm,
		PackageSpec:         packageSpec,
		MakerName:           master.MakerName,
		DatQuantity:         0, // DAT由来ではない
		JanPackInnerQty:     master.JanPackInnerQty,
		JanQuantity:         janQuantity,
		JanPackUnitQty:      master.JanPackUnitQty,
		JanUnitName:         janUnitName,
		JanUnitCode:         janUnitCodeStr,
		YjQuantity:          yjQuantity,
		YjPackUnitQty:       master.YjPackUnitQty,
		YjUnitName:          yjUnitName,
		UnitPrice:           unitPrice, // 薬価
		PurchasePrice:       master.PurchasePrice,
		SupplierWholesale:   master.SupplierWholesale,
		Subtotal:            subtotal, // 薬価 * YJ数量
		TaxAmount:           0,
		TaxRate:             0,
		ExpiryDate:          "", // 処方データにはない
		LotNumber:           "", // 処方データにはない
		FlagPoison:          master.FlagPoison,
		FlagDeleterious:     master.FlagDeleterious,
		FlagNarcotic:        master.FlagNarcotic,
		FlagPsychotropic:    master.FlagPsychotropic,
		FlagStimulant:       master.FlagStimulant,
		FlagStimulantRaw:    master.FlagStimulantRaw,
		ProcessFlagMA:       processFlagMA,
	}
}

// removeUsageDuplicates は処方レコードから重複を除外します (WASABI のロジック)
func removeUsageDuplicates(records []model.UnifiedInputRecord) []model.UnifiedInputRecord {
	seen := make(map[string]struct{})
	var result []model.UnifiedInputRecord
	for _, r := range records {
		// 処方データは 日付+JAN+YJ+製品名 が同じなら重複とみなす
		key := fmt.Sprintf("%s|%s|%s|%s", r.Date, r.JanCode, r.YjCode, r.ProductName)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, r)
	}
	return result
}
