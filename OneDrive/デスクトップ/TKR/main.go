// C:\Users\wasab\OneDrive\デスクトップ\TKR\main.go
package main

import (
	"encoding/json"
	"html/template"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"tkr/config"
	"tkr/dat"
	"tkr/database"
	"tkr/inventoryadjustment"
	"tkr/loader"
	"tkr/masteredit"
	"tkr/product"
	"tkr/units"
	"tkr/usage"
)

var (
	appTemplate   *template.Template
	viewsFS       fs.FS
	searchFormsFS fs.FS
)

func main() {
	log.Println("Connecting to database...")
	dbConn, err := sqlx.Open("sqlite3", "./tkr.db?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("db open error: %v",
			err)
	}
	defer dbConn.Close()
	log.Println("Database connection successful.")

	if _, err := config.LoadConfig(); err != nil {
		log.Printf("WARN: Failed to load config file: %v. Using defaults.", err)
	}

	if err := loader.InitDatabase(dbConn); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	log.Println("Database initialization complete.")

	if _, err := units.LoadTANIFile("SOU/TANI.CSV"); err != nil {
		log.Printf("WARN: Failed to load TANI.CSV: %v. Unit names may not display correctly.", err)
	} else {
		log.Println("Unit (TANI.CSV) master loaded successfully.")
	}

	// 'static' フォルダをFSとしてキャプチャ
	staticFS := os.DirFS("static")
	// 'static/views' サブディレクトリをFSとしてキャプチャ
	viewsFS, err = fs.Sub(staticFS, "views")
	if err != nil {
		log.Printf("WARN: 'static/views' directory not found. Will only load index.html. %v", err)
	}

	// 共通部品 search_form_group.html のための FS を設定
	searchFormsFS, err = fs.Sub(staticFS, "views")
	if err != nil {
		log.Fatalf("Failed to sub views directory for search forms: %v", err)
	}

	// メインの index.html をパース
	appTemplate, err = template.ParseFS(staticFS, "index.html")
	if err != nil {
		log.Fatalf("Failed to parse index.html: %v", err)
	}

	if viewsFS != nil {
		// viewsFS (static/views) 内のすべての .html ファイルを追加でパース
		appTemplate, err = appTemplate.ParseFS(viewsFS, "*.html")
		if err != nil {
			log.Fatalf("Failed to parse views/*.html: %v", err)
		}
	}

	// 共通部品 search_form_group.html をパースに追加
	if searchFormsFS != nil {
		appTemplate, err = appTemplate.ParseFS(searchFormsFS, "search_form_group.html")
		if err != nil {
			log.Fatalf("Failed to parse views/search_form_group.html: %v", err)
		}
	}

	// ▼▼▼【ここから追加】共通部品 common_search_modal.html をパースに追加 ▼▼▼
	if viewsFS != nil {
		appTemplate, err = appTemplate.ParseFS(viewsFS, "common_search_modal.html")
		if err != nil {
			log.Fatalf("Failed to parse views/common_search_modal.html: %v", err)
		}
	}
	// ▲▲▲【追加ここまで】▲▲▲

	log.Println("HTML templates loaded and parsed.")

	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir("./static"))))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// viewsFS から全ビューのファイル名を取得
		viewFiles := []string{}
		if viewsFS != nil {
			// search_form_group.html と common_search_modal.html は部品なので除外する
			files, err := fs.Glob(viewsFS, "*.html")
			if err != nil {
				log.Printf("Error globbing view files: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			for _, file := range files {
				if file != "search_form_group.html" && file != "common_search_modal.html" {
					viewFiles = append(viewFiles, file)
				}
			}
		}

		// 全ビューを結合するためのデータマップ
		viewMap := make(map[string]template.HTML)
		for _, file := range viewFiles {
			// ファイル名 (例: dat_view.html) からキー (dat_view) を作成
			key := strings.TrimSuffix(file, filepath.Ext(file))

			data := struct {
				Prefix             string
				BarcodeFormID      string
				BarcodeFormInputID string
				SearchButtonID     string
				SearchButtonText   string
			}{}

			// プリフィックスとIDを設定
			switch key {
			case "dat_view":
				data.Prefix = "dat_"
				data.BarcodeFormID = "dat-barcode-form"
				data.BarcodeFormInputID = "dat-search-barcode"
				data.SearchButtonID = "datOpenSearchModalBtn"
				data.SearchButtonText = "品目検索..."
			case "inventory_adjustment_view":
				data.Prefix = "ia_"
				data.BarcodeFormID = "ia-barcode-form"
				data.BarcodeFormInputID = "ia-barcode-input"
				data.SearchButtonID = "ia-search-btn"
				data.SearchButtonText = "この条件で検索"
			case "master_edit_view":
				data.Prefix = "master_"
				data.BarcodeFormID = "master-barcode-form"
				data.BarcodeFormInputID = "master-search-gs1-barcode"
				data.SearchButtonID = "masterSearchBtn"
				data.SearchButtonText = "品目検索..."
			default:
				data.Prefix = ""
			}

			// バッファにビューを描画
			var viewContent strings.Builder
			if err := appTemplate.ExecuteTemplate(&viewContent, file, data); err != nil {
				log.Printf("Error executing view template %s: %v", file, err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			viewMap[key] = template.HTML(viewContent.String())
		}

		// メインの index.html テンプレートに全ビューを埋め込んで描画
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		err = appTemplate.ExecuteTemplate(w, "index.html", struct {
			Views map[string]template.HTML
		}{
			Views: viewMap,
		})
		if err != nil {
			log.Printf("Error executing main template: %v", err)
		}
	})

	// ( ... JCSHMS APIハンドラ ... )
	mux.HandleFunc("/api/jcshms/", func(w http.ResponseWriter, r *http.Request) {
		janCode := strings.TrimPrefix(r.URL.Path, "/api/jcshms/")
		if janCode == "" {
			http.Error(w, "JAN code is required", http.StatusBadRequest)
			return
		}
		log.Printf("API request received for JAN: %s", janCode)
		info, err := database.GetJcshmsInfoByJan(dbConn, janCode)
		if err != nil {
			log.Printf("Error querying database for JAN %s: %v", janCode, err)
			http.Error(w, "Failed to retrieve JCSHMS info", http.StatusInternalServerError)
			return
		}
		if info == nil {
			log.Printf("JCSHMS info not found for JAN: %s",
				janCode)
			http.NotFound(w,
				r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(info); err != nil {
			log.Printf("Error encoding JSON response for JAN %s: %v", janCode, err)
		}
		log.Printf("Successfully returned JCSHMS info for JAN: %s", janCode)
	})

	// ( ... 他のAPIハンドラ ... )
	mux.HandleFunc("/api/dat/upload", dat.UploadDatHandler(dbConn))
	mux.HandleFunc("/api/dat/search", dat.SearchDatHandler(dbConn))
	mux.HandleFunc("/api/usage/upload", usage.UploadUsageHandler(dbConn))

	mux.HandleFunc("/api/masters", masteredit.ListMastersHandler(dbConn))
	mux.HandleFunc("/api/masters/update", masteredit.UpdateMasterHandler(dbConn))

	mux.HandleFunc("/api/inventory/adjust/data", inventoryadjustment.GetInventoryDataHandler(dbConn))
	mux.HandleFunc("/api/inventory/adjust/save", inventoryadjustment.SaveInventoryDataHandler(dbConn))

	mux.HandleFunc("/api/products/search_filtered", product.SearchProductsHandler(dbConn))

	mux.HandleFunc("/api/product/by_barcode/", product.GetProductByBarcodeHandler(dbConn))
	mux.HandleFunc("/api/master/by_code/", func(w http.ResponseWriter, r *http.Request) {
		janCode := strings.TrimPrefix(r.URL.Path, "/api/master/by_code/")
		if janCode == "" {
			http.Error(w, "JAN code is required", http.StatusBadRequest)
			return
		}
		master, err := database.GetProductMasterByCode(dbConn, janCode)
		if err != nil {
			log.Printf("Error retrieving master by code %s: %v", janCode, err)
			http.Error(w, "Master not found or database error", http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(master)
	})

	mux.HandleFunc("/api/wholesalers/list", ListWholesalersHandler(dbConn))
	mux.HandleFunc("/api/wholesalers/create", CreateWholesalerHandler(dbConn))
	mux.HandleFunc("/api/wholesalers/delete/", DeleteWholesalerHandler(dbConn))

	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			GetConfigHandler()(w, r)
		case http.MethodPost:
			SaveConfigHandler()(w, r)
		default:
			http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		}
	})

	mux.HandleFunc("/api/units/map", units.GetTaniMapHandler())

	port := ":8080"
	log.Printf("Starting server on http://localhost%s", port)

	openBrowser("http://localhost:8080")

	if err := http.ListenAndServe(port, mux); err != nil {
		log.Fatalf("server start error: %v", err)
	}
}

func openBrowser(url string) {
	var err error
	switch runtime.GOOS {
	case "windows":
		err = exec.Command("rundll32", "url.dll,FileProtocolHandler", url).Start()
	case "darwin":
		err = exec.Command("open", url).Start()
	default:
		err = exec.Command("xdg-open", url).Start()
	}
	if err != nil {
		log.Printf("failed to open browser: %v", err)
	}
}
