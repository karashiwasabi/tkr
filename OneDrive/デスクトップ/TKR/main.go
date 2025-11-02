// C:\Users\wasab\OneDrive\デスクトップ\TKR\main.go
package main

import (
	"encoding/json" // ▼▼▼【ここに追加】▼▼▼
	"html/template" // ▼▼▼【ここに追加】▼▼▼
	"io/fs"         // ▼▼▼【ここに追加】▼▼▼
	"log"
	"net/http"
	"os" // ▼▼▼【ここに追加】▼▼▼
	"os/exec"
	"path/filepath" // ▼▼▼【ここに追加】▼▼▼
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

// ▼▼▼【ここから追加】HTMLテンプレートを保持する変数 ▼▼▼
var (
	appTemplate *template.Template
	viewsFS     fs.FS
)

// ▲▲▲【追加ここまで】▲▲▲

func main() {
	log.Println("Connecting to database...")
	dbConn, err := sqlx.Open("sqlite3", "./tkr.db?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("db open error: %v", err)
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

	// ▼▼▼【ここから追加】HTMLテンプレートの読み込み処理 ▼▼▼
	// 'static' フォルダをFSとしてキャプチャ
	staticFS := os.DirFS("static")
	// 'static/views' サブディレクトリをFSとしてキャプチャ
	viewsFS, err = fs.Sub(staticFS, "views")
	if err != nil {
		// 'static/views' がなくてもエラーにしない（TKRの構成に合わせて柔軟に）
		log.Printf("WARN: 'static/views' directory not found. Will only load index.html. %v", err)
		// viewsFS が nil でも続行
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
	log.Println("HTML templates loaded and parsed.")
	// ▲▲▲【追加ここまで】▲▲▲

	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// ▼▼▼【修正】ルートハンドラをテンプレートを描画するように変更 ▼▼▼
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// viewsFS から全ビューのファイル名を取得
		viewFiles := []string{}
		if viewsFS != nil {
			viewFiles, err = fs.Glob(viewsFS, "*.html")
			if err != nil {
				log.Printf("Error globbing view files: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
		}

		// 全ビューを結合するためのデータマップ
		viewMap := make(map[string]template.HTML)
		for _, file := range viewFiles {
			// ファイル名 (例: dat_view.html) からキー (dat_view) を作成
			key := strings.TrimSuffix(file, filepath.Ext(file))
			// バッファにビューを描画
			var viewContent strings.Builder
			if err := appTemplate.ExecuteTemplate(&viewContent, file, nil); err != nil {
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
	// ▲▲▲【修正ここまで】▲▲▲

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
			log.Printf("JCSHMS info not found for JAN: %s", janCode)
			http.NotFound(w, r)
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
	mux.HandleFunc("/api/product/by_gs1", product.GetProductByGS1Handler(dbConn))
	mux.HandleFunc("/api/master/by_code/", product.GetMasterByCodeHandler(dbConn))

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

// ( ... openBrowser 関数 ... )
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
