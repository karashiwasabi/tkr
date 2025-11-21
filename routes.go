// C:\Users\wasab\OneDrive\デスクトップ\TKR\routes.go
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
	"tkr/automation"
	"tkr/backorder"
	"tkr/client"
	"tkr/dat"
	"tkr/database"
	"tkr/deadstock"
	"tkr/inout"
	"tkr/inventoryadjustment"
	"tkr/loader"
	"tkr/masteredit"
	"tkr/precomp"
	"tkr/pricing"
	"tkr/product"
	"tkr/reorder"
	"tkr/reprocess"
	"tkr/stock"
	"tkr/units"
	"tkr/usage"
	"tkr/valuation"

	"github.com/jmoiron/sqlx"
)

func SetupRoutes(mux *http.ServeMux, dbConn *sqlx.DB) {

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
	mux.HandleFunc("/api/dat/search", dat.SearchDatHandler(dbConn))
	mux.HandleFunc("/api/usage/upload", usage.UploadUsageHandler(dbConn))

	mux.HandleFunc("/api/receipts/by_date", inout.GetReceiptNumbersByDateHandler(dbConn))
	mux.HandleFunc("/api/transaction/", inout.GetTransactionsByReceiptNumberHandler(dbConn))
	mux.HandleFunc("/api/inout/save", inout.SaveInOutHandler(dbConn))
	mux.HandleFunc("/api/transaction/delete/", inout.DeleteTransactionHandler(dbConn))
	mux.HandleFunc("/api/masters", masteredit.ListMastersHandler(dbConn))
	mux.HandleFunc("/api/masters/update", masteredit.UpdateMasterHandler(dbConn))

	mux.HandleFunc("/api/master/set_order_stopped", masteredit.SetOrderStoppedHandler(dbConn))
	mux.HandleFunc("/api/masters/bulk_update_shelf", masteredit.BulkUpdateShelfHandler(dbConn))

	mux.HandleFunc("/api/inventory/adjust/data", inventoryadjustment.GetInventoryDataHandler(dbConn))
	mux.HandleFunc("/api/inventory/adjust/save", inventoryadjustment.SaveInventoryDataHandler(dbConn))
	mux.HandleFunc("/api/inventory/clear_old", inventoryadjustment.ClearOldInventoryHandler(dbConn))

	mux.HandleFunc("/api/precomp/save", precomp.SavePrecompHandler(dbConn))
	mux.HandleFunc("/api/precomp/load", precomp.LoadPrecompHandler(dbConn))
	mux.HandleFunc("/api/precomp/clear", precomp.ClearPrecompHandler(dbConn))
	mux.HandleFunc("/api/precomp/suspend", precomp.SuspendPrecompHandler(dbConn))
	mux.HandleFunc("/api/precomp/resume", precomp.ResumePrecompHandler(dbConn))
	mux.HandleFunc("/api/precomp/status", precomp.GetStatusPrecompHandler(dbConn))
	mux.HandleFunc("/api/precomp/export/all", precomp.ExportAllPrecompHandler(dbConn))
	mux.HandleFunc("/api/precomp/import/all", precomp.ImportAllPrecompHandler(dbConn))

	mux.HandleFunc("/api/products/search_filtered", product.SearchProductsHandler(dbConn))
	mux.HandleFunc("/api/master/adopt", product.AdoptMasterHandler(dbConn))

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
	mux.HandleFunc("/api/clients/import", client.ImportClientsHandler(dbConn))

	mux.HandleFunc("/api/clients", func(w http.ResponseWriter, r *http.Request) {
		clients, err := database.GetAllClients(dbConn)
		if err != nil {
			http.Error(w, "Failed to get clients", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(clients)
	})

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

	mux.HandleFunc("/api/jcshms/reload", loader.ReloadJCSHMSHandler(dbConn))

	mux.HandleFunc("/api/reprocess/all", reprocess.ProcessTransactionsHandler(dbConn))

	mux.HandleFunc("/api/deadstock/list", deadstock.ListDeadStockHandler(dbConn))
	mux.HandleFunc("/api/deadstock/upload", deadstock.UploadDeadStockCSVHandler(dbConn))
	mux.HandleFunc("/api/deadstock/export", deadstock.ExportDeadStockHandler(dbConn))

	mux.HandleFunc("/api/stock/import/tkr", stock.ImportTKRStockCSVHandler(dbConn))

	mux.HandleFunc("/api/masters/export/all", stock.ExportAllMastersHandler(dbConn))
	mux.HandleFunc("/api/masters/import/all", stock.ImportAllMastersHandler(dbConn))

	mux.HandleFunc("/api/reorder/candidates", reorder.GenerateOrderCandidatesHandler(dbConn))
	mux.HandleFunc("/api/orders/place", reorder.PlaceOrderHandler(dbConn))
	mux.HandleFunc("/api/reorder/export_dat", reorder.ExportFixedLengthDatHandler(dbConn)) // ★追加
	mux.HandleFunc("/api/backorders", backorder.GetBackordersHandler(dbConn))
	mux.HandleFunc("/api/backorders/delete", backorder.DeleteBackorderHandler(dbConn))
	mux.HandleFunc("/api/backorders/bulk_delete_by_id", backorder.BulkDeleteBackordersByIDHandler(dbConn))
	mux.HandleFunc("/api/backorders/bulk_delete_by_date", backorder.BulkDeleteBackordersByDateHandler(dbConn))

	mux.HandleFunc("/api/returns/candidates", reorder.GenerateReturnCandidatesHandler(dbConn))

	mux.HandleFunc("/api/valuation", valuation.GetValuationHandler(dbConn))
	mux.HandleFunc("/api/valuation/export_csv", valuation.ExportValuationCSVHandler(dbConn))

	mux.HandleFunc("/api/pricing/all_masters", pricing.GetAllMastersForPricingHandler(dbConn))
	mux.HandleFunc("/api/pricing/export", pricing.GetExportDataHandler(dbConn))
	mux.HandleFunc("/api/pricing/upload", pricing.UploadQuotesHandler(dbConn))
	mux.HandleFunc("/api/pricing/update", pricing.BulkUpdateHandler(dbConn))
	mux.HandleFunc("/api/pricing/direct_import", pricing.DirectImportHandler(dbConn))
	mux.HandleFunc("/api/pricing/backup_export", pricing.BackupExportHandler(dbConn))

	mux.HandleFunc("/api/automation/medicode/download", automation.DownloadMedicodeDatHandler(dbConn))
}
