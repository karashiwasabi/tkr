// C:\Users\wasab\OneDrive\デスクトップ\TKR\main.go
package main

import (
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
	"tkr/loader"
	"tkr/units"
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

	staticFS := os.DirFS("static")
	viewsFS, err = fs.Sub(staticFS, "views")
	if err != nil {
		log.Printf("WARN: 'static/views' directory not found. Will only load index.html. %v", err)
	}

	searchFormsFS, err = fs.Sub(staticFS, "views")
	if err != nil {
		log.Fatalf("Failed to sub views directory for search forms: %v", err)
	}

	appTemplate, err = template.ParseFS(staticFS, "index.html")
	if err != nil {
		log.Fatalf("Failed to parse index.html: %v", err)
	}

	if viewsFS != nil {
		appTemplate, err = appTemplate.ParseFS(viewsFS, "*.html")
		if err != nil {
			log.Fatalf("Failed to parse views/*.html: %v", err)
		}
	}

	if searchFormsFS != nil {
		appTemplate, err = appTemplate.ParseFS(searchFormsFS, "search_form_group.html")
		if err != nil {
			log.Fatalf("Failed to parse views/search_form_group.html: %v", err)
		}
	}

	if viewsFS != nil {
		appTemplate, err = appTemplate.ParseFS(viewsFS, "common_search_modal.html")
		if err != nil {
			log.Fatalf("Failed to parse views/common_search_modal.html: %v", err)
		}
		appTemplate, err = appTemplate.ParseFS(viewsFS, "common_input_modal.html")
		if err != nil {
			log.Fatalf("Failed to parse views/common_input_modal.html: %v", err)
		}

		// ▼▼▼【ここから追加】config用サブテンプレートの読み込み ▼▼▼
		appTemplate, err = appTemplate.ParseFS(viewsFS, "_config_paths.html")
		if err != nil {
			log.Fatalf("Failed to parse views/_config_paths.html: %v", err)
		}
		appTemplate, err = appTemplate.ParseFS(viewsFS, "_config_aggregation.html")
		if err != nil {
			log.Fatalf("Failed to parse views/_config_aggregation.html: %v", err)
		}
		appTemplate, err = appTemplate.ParseFS(viewsFS, "_config_wholesalers.html")
		if err != nil {
			log.Fatalf("Failed to parse views/_config_wholesalers.html: %v", err)
		}
		appTemplate, err = appTemplate.ParseFS(viewsFS, "_config_migration.html")
		if err != nil {
			log.Fatalf("Failed to parse views/_config_migration.html: %v", err)
		}
		// ▲▲▲【追加ここまで】▲▲▲
	}

	log.Println("HTML templates loaded and parsed.")

	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/",
		http.FileServer(http.Dir("./static"))))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		viewFiles := []string{}
		if viewsFS != nil {
			files, err := fs.Glob(viewsFS, "*.html")
			if err != nil {
				log.Printf("Error globbing view files: %v", err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			for _, file := range files {
				// ▼▼▼【ここを修正】サブテンプレートを除外リストに追加 ▼▼▼
				if file != "search_form_group.html" && file != "common_search_modal.html" && file != "common_input_modal.html" &&
					!strings.HasPrefix(file, "_config_") {
					viewFiles = append(viewFiles, file)
				}
				// ▲▲▲【修正ここまで】▲▲▲
			}
		}

		viewMap := make(map[string]template.HTML)
		for _, file := range viewFiles {
			key := strings.TrimSuffix(file, filepath.Ext(file))

			data :=
				struct {
					Prefix             string
					BarcodeFormID      string
					BarcodeFormInputID string
					SearchButtonID     string
					SearchButtonText   string
				}{}

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
				data.SearchButtonText = "既存マスタ検索"
			default:
				data.Prefix = ""
			}

			var viewContent strings.Builder
			if err := appTemplate.ExecuteTemplate(&viewContent, file, data); err != nil {
				log.Printf("Error executing view template %s: %v", file, err)
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			viewMap[key] = template.HTML(viewContent.String())
		}

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

	SetupRoutes(mux, dbConn)

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
