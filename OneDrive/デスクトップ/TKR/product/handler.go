// C:\Users\wasab\OneDrive\デスクトップ\TKR\product\handler.go
package product

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strings"
	"tkr/barcode" // ▼▼▼【ここに追加】barcode パッケージをインポート ▼▼▼
	"tkr/database"
	"tkr/mappers"       // Viewマッパー
	"tkr/mastermanager" //
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

// ▼▼▼【ここに追加】JcshmsToProductMasterInput の結果を ProductMaster (View用) に変換するヘルパー ▼▼▼
// (SearchProductsHandler が DB に挿入せずに View を生成するために使用)
func inputToMaster(input model.ProductMasterInput) *model.ProductMaster {
	return &model.ProductMaster{
		ProductCode:         input.ProductCode,
		YjCode:              input.YjCode,
		Gs1Code:             input.Gs1Code,
		ProductName:         input.ProductName,
		KanaName:            input.KanaName,
		KanaNameShort:       input.KanaNameShort,
		GenericName:         input.GenericName,
		MakerName:           input.MakerName,
		Specification:       input.Specification,
		UsageClassification: input.UsageClassification,
		PackageForm:         input.PackageForm,
		YjUnitName:          input.YjUnitName,
		YjPackUnitQty:       input.YjPackUnitQty,
		JanPackInnerQty:     input.JanPackInnerQty,
		JanUnitCode:         input.JanUnitCode,
		JanPackUnitQty:      input.JanPackUnitQty,
		Origin:              input.Origin,
		NhiPrice:            input.NhiPrice,
		PurchasePrice:       input.PurchasePrice,
		FlagPoison:          input.FlagPoison,
		FlagDeleterious:     input.FlagDeleterious,
		FlagNarcotic:        input.FlagNarcotic,
		FlagPsychotropic:    input.FlagPsychotropic,
		FlagStimulant:       input.FlagStimulant,
		FlagStimulantRaw:    input.FlagStimulantRaw,
		IsOrderStopped:      input.IsOrderStopped,
		SupplierWholesale:   input.SupplierWholesale,
		GroupCode:           input.GroupCode,
		ShelfNumber:         input.ShelfNumber,
		Category:            input.Category,
		UserNotes:           input.UserNotes,
	}
}

// ▲▲▲【追加ここまで】▲▲▲

func SearchProductsHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		kanaName := q.Get("kanaName")
		dosageForm := q.Get("dosageForm")
		genericName := q.Get("genericName")
		shelfNumber := q.Get("shelfNumber")
		searchMode := q.Get("searchMode")

		// searchModeが'inout'の場合のみ、JCSHMSを含めた統合検索を行う
		if searchMode == "inout" {
			// 1. product_master から検索 (採用済み)
			adoptedMasters, err := database.GetFilteredProductMasters(conn, dosageForm, kanaName, genericName, shelfNumber)
			if err != nil {
				http.Error(w, "Failed to search product_master: "+err.Error(), http.StatusInternalServerError)
				return
			}

			mergedResults := []model.ProductMasterView{}
			seenCodes := make(map[string]bool)

			for _, master := range adoptedMasters {
				view := mappers.ToProductMasterView(&master)
				view.IsAdopted = true
				mergedResults = append(mergedResults, view)
				seenCodes[master.ProductCode] = true
			}

			// 2. jcshms_master から検索 (未採用候補)
			jcshmsResults, err := database.GetFilteredJcshmsInfo(conn, dosageForm, kanaName, genericName)
			if err != nil {
				http.Error(w, "Failed to search jcshms_master: "+err.Error(), http.StatusInternalServerError)

				return
			}

			for _, jcshmsInfo := range jcshmsResults {
				if seenCodes[jcshmsInfo.ProductCode] {
					continue // 既に採用済みのものはスキップ
				}
				// ▼▼▼【ここから修正】mappers.FromJcshmsToProductMaster -> mastermanager.JcshmsToProductMasterInput ▼▼▼
				input := mastermanager.JcshmsToProductMasterInput(jcshmsInfo)
				tempMaster := inputToMaster(input) // View生成のために *model.ProductMaster に変換
				// ▲▲▲【修正ここまで】▲▲▲
				view := mappers.ToProductMasterView(tempMaster)
				view.IsAdopted = false
				mergedResults = append(mergedResults, view)
				seenCodes[jcshmsInfo.ProductCode] = true
			}

			// 3. 最終結果をソート
			sort.Slice(mergedResults, func(i, j int) bool {
				return mergedResults[i].KanaName < mergedResults[j].KanaName
			})

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mergedResults)

		} else {
			// それ以外の画面では、product_masterのみを検索
			localMasters, err := database.GetFilteredProductMasters(conn, dosageForm, kanaName, genericName, shelfNumber)
			if err != nil {
				http.Error(w, "Failed to search product_master: "+err.Error(), http.StatusInternalServerError)
				return
			}

			results := make([]model.ProductMasterView, len(localMasters))
			for i, master := range localMasters {
				view := mappers.ToProductMasterView(&master)
				view.IsAdopted = true // ローカルに存在するので全て採用済み
				results[i] = view
			}

			sort.Slice(results, func(i, j int) bool {
				return results[i].KanaName < results[j].KanaName
			})

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(results)
		}
	}
}

// ▼▼▼【ここから修正】バーコード検索を 13桁JAN / 14桁GS1 両対応に修正 ▼▼▼
func GetProductByBarcodeHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawBarcode := strings.TrimPrefix(r.URL.Path, "/api/product/by_barcode/")
		if rawBarcode == "" {
			http.Error(w, "barcode is required", http.StatusBadRequest)
			return
		}

		log.Printf("API: Received raw barcode: %s", rawBarcode)

		// 1. まず product_master を検索 (13桁JAN/14桁GS1自動振り分け)
		master, err := database.GetProductMasterByBarcode(conn, rawBarcode)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			// データベース検索で「見つからない」以外のエラーが起きた場合
			log.Printf("Error searching product_master by Barcode %s: %v", rawBarcode, err)
			http.Error(w, "データベースエラーが発生しました", http.StatusInternalServerError)
			return
		}

		// 3. product_master に見つかった場合
		if master != nil {
			log.Printf("Found product in product_master: %s", master.ProductName)
			masterView := mappers.ToProductMasterView(master)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(masterView)
			return
		}

		// 4. product_master に見つからなかった場合、JCSHMSを検索
		log.Printf("Product not in product_master, searching JCSHMS...")

		// ▼▼▼【ここから修正】JCSHMS検索も 13桁JAN / 14桁GS1 を振り分ける ▼▼▼
		var jcshmsInfo *model.JcshmsInfo
		var jcshmsErr error
		var gtin14 string // GS1採用時に使用

		if len(rawBarcode) <= 13 {
			// 13桁以下 (JAN)
			log.Printf("Barcode %s is JAN format. Searching JCSHMS by JAN...", rawBarcode)
			jcshmsInfo, jcshmsErr = database.GetJcshmsInfoByJan(conn, rawBarcode)
		} else {
			// 14桁以上 (GS1)
			parsed, parseErr := barcode.Parse(rawBarcode)
			if parseErr != nil {
				http.Error(w, fmt.Sprintf("バーコード解析エラー: %v", parseErr), http.StatusBadRequest)
				return
			}
			gtin14 = parsed.Gtin14
			if gtin14 == "" {
				http.Error(w, "バーコードから製品コード(GTIN)が抽出できません", http.StatusBadRequest)
				return
			}
			log.Printf("Barcode %s is GS1 format (GTIN: %s). Searching JCSHMS by GS1...", rawBarcode, gtin14)
			jcshmsInfo, jcshmsErr = database.GetJcshmsInfoByGs1Code(conn, gtin14)
		}
		// ▲▲▲【修正ここまで】▲▲▲

		if jcshmsErr != nil {
			log.Printf("Error searching JCSHMS by Barcode %s: %v", rawBarcode, jcshmsErr)
			http.Error(w, "JCSHMSマスターの検索中にエラーが発生しました", http.StatusInternalServerError)
			return
		}

		// 5. JCSHMS にも見つからなかった場合
		if jcshmsInfo == nil {
			log.Printf("Product not found in JCSHMS for Barcode %s", rawBarcode)
			http.Error(w, "どのマスターにも製品が見つかりませんでした", http.StatusNotFound)
			return
		}

		// 6. JCSHMS に見つかった場合、product_masterに新規作成
		log.Printf("Found product in JCSHMS: %s. Creating new master...", jcshmsInfo.ProductName)

		input := mastermanager.JcshmsToProductMasterInput(jcshmsInfo)

		// ▼▼▼【ここから追加】14桁GS1の場合、Gs1Codeをセットする ▼▼▼
		if gtin14 != "" && input.Gs1Code == "" {
			input.Gs1Code = gtin14
			log.Printf("Setting Gs1Code from parsed barcode: %s", gtin14)
		}
		// ▲▲▲【追加ここまで】▲▲▲

		tx, err := conn.Beginx()
		if err != nil {
			log.Printf("Failed to begin transaction to create master: %v", err)
			http.Error(w, "トランザクションの開始に失敗しました", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback() // In case of panic or error

		newMaster, err := mastermanager.UpsertProductMasterSqlx(tx, input)
		if err != nil {
			log.Printf("Failed to insert new master from JCSHMS: %v", err)
			http.Error(w, "マスターの新規作成に失敗しました", http.StatusInternalServerError)
			return
		}

		if err := tx.Commit(); err != nil {
			log.Printf("Failed to commit transaction to create master: %v", err)
			http.Error(w, "トランザクションのコミットに失敗しました", http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully created new master for %s", newMaster.ProductName)
		masterView := mappers.ToProductMasterView(newMaster)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(masterView)
	}
}

// ▲▲▲【修正ここまで】▲▲▲

func AdoptMasterHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Gs1Code string `json:"gs1Code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if payload.Gs1Code == "" {
			http.Error(w, "gs1Code is required", http.StatusBadRequest)
			return
		}

		// JCSHMSをGS1コードで検索
		jcshmsInfo, err := database.GetJcshmsInfoByGs1Code(conn, payload.Gs1Code)
		if err != nil {
			log.Printf("Error searching JCSHMS for adoption by GS1 %s: %v", payload.Gs1Code, err)
			http.Error(w, "JCSHMSマスターの検索中にエラーが発生しました", http.StatusInternalServerError)
			return
		}

		if jcshmsInfo == nil {
			http.Error(w, "Adoption failed: Product not found in JCSHMS", http.StatusNotFound)
			return
		}

		// product_masterに新規作成
		// ▼▼▼【ここから修正】mappers.FromJcshmsToProductMaster -> mastermanager.JcshmsToProductMasterInput ▼▼▼
		input := mastermanager.JcshmsToProductMasterInput(jcshmsInfo)
		// ▲▲▲【修正ここまで】▲▲▲

		tx, err := conn.Beginx()
		if err != nil {
			http.Error(w, "トランザクションの開始に失敗しました", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		// ▼▼▼【ここから修正】database.InsertProductMaster -> mastermanager.UpsertProductMasterSqlx ▼▼▼
		newMaster, err := mastermanager.UpsertProductMasterSqlx(tx, input) // Upsert を呼び出し、 *model.ProductMaster を受け取る
		if err != nil {

			http.Error(w, "マスターの新規作成に失敗しました", http.StatusInternalServerError)
			return
		}
		// ▲▲▲【修正ここまで】▲▲▲

		if err := tx.Commit(); err != nil {
			http.Error(w, "トランザクションのコミットに失敗しました", http.StatusInternalServerError)
			return
		}

		log.Printf("Successfully adopted master for %s", newMaster.ProductName)
		masterView := mappers.ToProductMasterView(newMaster)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(masterView)
	}
}
