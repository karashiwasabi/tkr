// C:\Users\wasab\OneDrive\デスクトップ\TKR\main.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"runtime"
	"strings"

	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"

	"tkr/config" // ★ Import config
	"tkr/dat"
	"tkr/database"
	"tkr/loader"
	"tkr/masteredit"
	// "tkr/usage" // ★ USAGE削除
)

func main() {
	log.Println("Connecting to database...")
	dbConn, err := sqlx.Open("sqlite3", "./tkr.db?_journal_mode=WAL&_busy_timeout=5000")
	if err != nil {
		log.Fatalf("db open error: %v", err)
	}
	defer dbConn.Close()
	log.Println("Database connection successful.")

	// ★ Load config on startup
	if _, err := config.LoadConfig(); err != nil {
		log.Printf("WARN: Failed to load config file: %v. Using defaults.", err)
	}

	if err := loader.InitDatabase(dbConn); err != nil {
		log.Fatalf("Database initialization failed: %v", err)
	}
	log.Println("Database initialization complete.")

	mux := http.NewServeMux()

	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		http.ServeFile(w, r, "./static/index.html")
	})

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

	mux.HandleFunc("/api/dat/upload", dat.UploadDatHandler(dbConn))

	mux.HandleFunc("/api/masters", masteredit.ListMastersHandler(dbConn))
	mux.HandleFunc("/api/masters/update", masteredit.UpdateMasterHandler(dbConn))

	// ★ USAGE削除 (関連ハンドラを削除)
	/*
		mux.HandleFunc("/api/usage/config", func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet {
				usage.GetUsageConfigHandler(dbConn)(w, r)
			} else if r.Method == http.MethodPost {
				usage.SaveUsageConfigHandler(dbConn)(w, r)
			} else {
				http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
			}
		})
		mux.HandleFunc("/api/usage/import", usage.ImportUsageHandler(dbConn)) // Assumes POST method
	*/

	port := ":8080"
	log.Printf("Starting server on http://localhost%s", port)

	// ▼▼▼【ここから修正】openBrowser 関数を呼び出す ▼▼▼
	openBrowser("http://localhost:8080")
	// ▲▲▲【修正ここまで】▲▲▲

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
