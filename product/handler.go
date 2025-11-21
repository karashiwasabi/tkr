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
	"tkr/barcode"
	"tkr/database"
	"tkr/mappers"
	"tkr/mastermanager"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

// (inputToMaster 関数は変更なし)
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

func SearchProductsHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		kanaName := q.Get("kanaName")
		dosageForm := q.Get("dosageForm")
		genericName := q.Get("genericName")
		shelfNumber := q.Get("shelfNumber")
		searchMode := q.Get("searchMode")

		productName := q.Get("productName")
		drugTypesRaw := q.Get("drugTypes")
		var drugTypes []string
		if drugTypesRaw != "" {
			drugTypes = strings.Split(drugTypesRaw, ",")
		}

		if searchMode == "inout" {
			// --- 「JCSHMSから採用」フロー (マスタ編集, 入出庫明細, 発注追加) ---

			// 1. ローカルマスタ(product_master)を先に検索してマップ化する
			//    これにより、JCSHMS側で見つかった品目がローカルにもある場合、ローカルの情報を優先できる（卸情報など）
			localMasters, err := database.GetFilteredProductMasters(conn, dosageForm, kanaName, genericName, shelfNumber, productName, drugTypes)
			if err != nil {
				http.Error(w, "Failed to search product_master: "+err.Error(), http.StatusInternalServerError)
				return
			}
			localMasterMap := make(map[string]model.ProductMaster)
			for _, m := range localMasters {
				localMasterMap[m.ProductCode] = m
			}

			// 2. 採用済みの全JANコードマップを取得 (検索条件にヒットしなくても、採用済みかどうかの判定用)
			adoptedCodeMap, err := database.GetAllAdoptedProductCodesMap(conn)
			if err != nil {
				http.Error(w, "Failed to get adopted product map: "+err.Error(), http.StatusInternalServerError)
				return
			}

			mergedResults := []model.ProductMasterView{}
			seenCodes := make(map[string]bool)

			// 3. JCSHMS から検索
			jcshmsResults, err := database.GetFilteredJcshmsInfo(conn, dosageForm, kanaName, genericName, productName, drugTypes)
			if err != nil {
				http.Error(w, "Failed to search jcshms_master: "+err.Error(), http.StatusInternalServerError)
				return
			}

			for _, jcshmsInfo := range jcshmsResults {
				jan := jcshmsInfo.ProductCode

				// ★修正ポイント: ローカルマスタに同じJANがあれば、そちらを優先して使う
				if localM, exists := localMasterMap[jan]; exists {
					view := mappers.ToProductMasterView(&localM)
					view.IsAdopted = true
					mergedResults = append(mergedResults, view)
				} else {
					// ローカルになければJCSHMSのデータを使う
					input := mastermanager.JcshmsToProductMasterInput(jcshmsInfo)
					tempMaster := inputToMaster(input)
					view := mappers.ToProductMasterView(tempMaster)

					// 採用済マップで照会 (検索条件には引っかからなかったが、実は採用済みの場合など)
					if _, ok := adoptedCodeMap[view.ProductCode]; ok {
						view.IsAdopted = true
						// ※ここで本来ならそのローカルマスタを取得すべきだが、検索条件外ならJCSHMS表示で妥協するか、
						// 個別に取得するコストを天秤にかける。
						// 基本的に「検索して追加」の場面なので、検索条件にヒットしたlocalMasterMapで十分カバーできるはず。
					} else {
						view.IsAdopted = false
					}
					mergedResults = append(mergedResults, view)
				}
				seenCodes[jan] = true
			}

			// 4. ローカルマスタのみに存在する品目（PROVISIONALなど）を追加
			for _, master := range localMasters {
				if seenCodes[master.ProductCode] {
					continue // 既に追加済みならスキップ
				}
				view := mappers.ToProductMasterView(&master)
				view.IsAdopted = true
				mergedResults = append(mergedResults, view)
				seenCodes[master.ProductCode] = true
			}

			// 5. 最終結果をソート
			sort.Slice(mergedResults, func(i, j int) bool {
				return mergedResults[i].KanaName < mergedResults[j].KanaName
			})

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mergedResults)

		} else {
			// --- 通常検索 (棚卸調整, DAT取込など) ---
			// ここは変更なし (ローカルマスタのみ検索)

			localMasters, err := database.GetFilteredProductMasters(conn, dosageForm, kanaName, genericName, shelfNumber, productName, drugTypes)
			if err != nil {
				http.Error(w, "Failed to search product_master: "+err.Error(), http.StatusInternalServerError)
				return
			}

			results := make([]model.ProductMasterView, len(localMasters))
			for i, master := range localMasters {
				view := mappers.ToProductMasterView(&master)
				view.IsAdopted = true
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

// ... (GetProductByBarcodeHandler 以降は変更なし)
func GetProductByBarcodeHandler(conn *sqlx.DB) http.HandlerFunc {
	// (省略: 変更なし)
	return func(w http.ResponseWriter, r *http.Request) {
		rawBarcode := strings.TrimPrefix(r.URL.Path, "/api/product/by_barcode/")
		if rawBarcode == "" {
			http.Error(w, "barcode is required", http.StatusBadRequest)
			return
		}

		log.Printf("API: Received raw barcode: %s", rawBarcode)

		master, err := database.GetProductMasterByBarcode(conn, rawBarcode)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			log.Printf("Error searching product_master by Barcode %s: %v", rawBarcode, err)
			http.Error(w, "データベースエラーが発生しました", http.StatusInternalServerError)
			return
		}

		if master != nil {
			log.Printf("Found product in product_master: %s", master.ProductName)
			masterView := mappers.ToProductMasterView(master)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(masterView)
			return
		}

		log.Printf("Product not in product_master, searching JCSHMS...")

		var jcshmsInfo *model.JcshmsInfo
		var jcshmsErr error
		var gtin14 string

		if len(rawBarcode) <= 13 {
			log.Printf("Barcode %s is JAN format. Searching JCSHMS by JAN...", rawBarcode)
			jcshmsInfo, jcshmsErr = database.GetJcshmsInfoByJan(conn, rawBarcode)
		} else {
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

		if jcshmsErr != nil {
			log.Printf("Error searching JCSHMS by Barcode %s: %v", rawBarcode, jcshmsErr)
			http.Error(w, "JCSHMSマスターの検索中にエラーが発生しました", http.StatusInternalServerError)
			return
		}

		if jcshmsInfo == nil {
			log.Printf("Product not found in JCSHMS for Barcode %s", rawBarcode)
			http.Error(w, "どのマスターにも製品が見つかりませんでした", http.StatusNotFound)
			return
		}

		log.Printf("Found product in JCSHMS: %s. Creating new master...", jcshmsInfo.ProductName)

		input := mastermanager.JcshmsToProductMasterInput(jcshmsInfo)

		if gtin14 != "" && input.Gs1Code == "" {
			input.Gs1Code = gtin14
			log.Printf("Setting Gs1Code from parsed barcode: %s", gtin14)
		}

		tx, err := conn.Beginx()
		if err != nil {
			log.Printf("Failed to begin transaction to create master: %v", err)
			http.Error(w, "トランザクションの開始に失敗しました", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

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

func AdoptMasterHandler(conn *sqlx.DB) http.HandlerFunc {
	// (省略: 変更なし)
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Gs1Code     string `json:"gs1Code"`
			ProductCode string `json:"productCode"`
		}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}

		if payload.Gs1Code == "" && payload.ProductCode == "" {
			http.Error(w, "gs1Code or productCode is required", http.StatusBadRequest)
			return
		}

		var jcshmsInfo *model.JcshmsInfo
		var err error

		if payload.Gs1Code != "" {
			jcshmsInfo, err = database.GetJcshmsInfoByGs1Code(conn, payload.Gs1Code)
		} else {
			jcshmsInfo, err = database.GetJcshmsInfoByJan(conn, payload.ProductCode)
		}

		if err != nil {
			log.Printf("Error searching JCSHMS for adoption by Key: %v", err)
			http.Error(w, "JCSHMSマスターの検索中にエラーが発生しました", http.StatusInternalServerError)
			return
		}

		if jcshmsInfo == nil {
			http.Error(w, "Adoption failed: Product not found in JCSHMS", http.StatusNotFound)
			return
		}

		input := mastermanager.JcshmsToProductMasterInput(jcshmsInfo)

		tx, err := conn.Beginx()
		if err != nil {
			http.Error(w, "トランザクションの開始に失敗しました", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		newMaster, err := mastermanager.UpsertProductMasterSqlx(tx, input)
		if err != nil {
			http.Error(w, "マスターの新規作成に失敗しました", http.StatusInternalServerError)
			return
		}

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
