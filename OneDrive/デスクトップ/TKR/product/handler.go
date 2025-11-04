// C:\Users\wasab\OneDrive\デスクトップ\TKR\product\handler.go
package product

import (
	"database/sql"

	// ▼▼▼【ここに追加】▼▼▼
	"encoding/json"
	"errors" // ▼▼▼【ここに追加】▼▼▼
	"fmt"    // ▼▼▼【ここに追加】▼▼▼
	"log"    // ▼▼▼【ここに追加】▼▼▼
	"net/http"
	"sort"
	"strings"
	"tkr/barcode"

	// ▼▼▼【ここに追加】▼▼▼
	"tkr/database"
	"tkr/mappers" // Viewマッパー
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

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
				tempMaster := mappers.FromJcshmsToProductMaster(jcshmsInfo)
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

// ▼▼▼【ここから修正】バーコード検索を単一APIに共通化（DB共通関数を使用） ▼▼▼
func GetProductByBarcodeHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		rawBarcode := strings.TrimPrefix(r.URL.Path, "/api/product/by_barcode/")
		if rawBarcode == "" {
			http.Error(w, "barcode is required", http.StatusBadRequest)
			return
		}

		log.Printf("API: Received raw barcode: %s", rawBarcode)

		// 1. バーコードを解析
		parsed, err := barcode.Parse(rawBarcode)
		if err != nil {
			http.Error(w, fmt.Sprintf("バーコード解析エラー: %v", err), http.StatusBadRequest)
			return
		}
		gtin14 := parsed.Gtin14
		if gtin14 == "" {
			http.Error(w, "バーコードから製品コード(GTIN)が抽出できません", http.StatusBadRequest)
			return
		}

		// 2. まず product_master を検索
		master, err := database.GetProductMasterByGs1Code(conn, gtin14)
		if err != nil && !errors.Is(err, sql.ErrNoRows) {
			// データベース検索で「見つからない」以外のエラーが起きた場合
			log.Printf("Error searching product_master by GS1 %s: %v", gtin14, err)
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
		jcshmsInfo, err := database.GetJcshmsInfoByGs1Code(conn, gtin14)
		if err != nil {
			log.Printf("Error searching JCSHMS by GS1 %s: %v", gtin14, err)
			http.Error(w, "JCSHMSマスターの検索中にエラーが発生しました", http.StatusInternalServerError)
			return
		}

		// 5. JCSHMS にも見つからなかった場合
		if jcshmsInfo == nil {
			log.Printf("Product not found in JCSHMS for GS1 %s", gtin14)
			http.Error(w, "どのマスターにも製品が見つかりませんでした", http.StatusNotFound)
			return
		}

		// 6. JCSHMS に見つかった場合、product_masterに新規作成
		log.Printf("Found product in JCSHMS: %s. Creating new master...", jcshmsInfo.ProductName)
		newMaster := mappers.FromJcshmsToProductMaster(jcshmsInfo)
		newMaster.Gs1Code = gtin14 // 解析で得たGTIN-14をセット

		tx, err := conn.Beginx()
		if err != nil {
			log.Printf("Failed to begin transaction to create master: %v", err)
			http.Error(w, "トランザクションの開始に失敗しました", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback() // In case of panic or error

		if err := database.InsertProductMaster(tx, newMaster); err != nil {
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
		newMaster := mappers.FromJcshmsToProductMaster(jcshmsInfo)

		tx, err := conn.Beginx()
		if err != nil {
			http.Error(w, "トランザクションの開始に失敗しました", http.StatusInternalServerError)
			return
		}
		defer tx.Rollback()

		if err := database.InsertProductMaster(tx, newMaster); err != nil {
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

// ▲▲▲【修正ここまで】▲▲▲

// ▼▼▼【ここから削除】古いハンドラを削除 ▼▼▼
/*
func GetProductByGS1Handler(conn *sqlx.DB) http.HandlerFunc
{
	return func(w http.ResponseWriter, r *http.Request) {
// ... (古いコード [cite: 804] 削除) ...
	}
}
func GetMasterByCodeHandler(conn *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
// ... (古いコード [cite: 804] 削除) ...
	}
}
*/
// ▲▲▲【削除ここまで】▲▲▲
