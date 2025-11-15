// C:\Users\wasab\OneDrive\デスクトップ\TKR\pricing\pricing_export_handler.go
package pricing

import (
	"encoding/csv"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"tkr/database"
	"tkr/mappers"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

// GetExportDataHandler は見積依頼CSVを作成します
func GetExportDataHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		wholesalerName := r.URL.Query().Get("wholesalerName")
		unregisteredOnlyStr := r.URL.Query().Get("unregisteredOnly")
		unregisteredOnly := unregisteredOnlyStr == "true"

		if wholesalerName == "" {
			writeJsonError(w, "Wholesaler name is required", http.StatusBadRequest)
			return
		}

		allMasters, err := database.GetAllProductMasters(db)
		if err != nil {
			writeJsonError(w, "Failed to get products for export", http.StatusInternalServerError)
			return
		}

		var mastersToProcess []*model.ProductMaster
		for _, p := range allMasters {
			if strings.HasPrefix(p.ProductCode, "MA2J") {
				continue
			}
			mastersToProcess = append(mastersToProcess, p)
		}

		var dataToExport []*model.ProductMaster
		if unregisteredOnly {
			for _, p := range mastersToProcess {
				if p.SupplierWholesale == "" {

					dataToExport = append(dataToExport, p)
				}
			}
			if len(dataToExport) == 0 {
				writeJsonError(w, "未登録の品目が見つかりませんでした。", http.StatusNotFound)
				return
			}
		} else {
			dataToExport = mastersToProcess
		}

		dateStr := r.URL.Query().Get("date")
		fileType := "ALL"
		if unregisteredOnly {
			fileType = "UNREGISTERED"
		}
		fileName := fmt.Sprintf("価格見積依頼_%s_%s_%s.csv", wholesalerName, fileType, dateStr)
		fileName = url.PathEscape(fileName)

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+fileName)
		w.Write([]byte{0xEF, 0xBB, 0xBF})

		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()

		headers := []string{"product_code", "product_name", "maker_name", "package_spec", "purchase_price"}
		if err := csvWriter.Write(headers); err != nil {
			log.Printf("Failed to write CSV header: %v", err)
		}

		for _, m := range dataToExport {
			view := mappers.ToProductMasterView(m)
			formattedSpec := view.FormattedPackageSpec

			record := []string{
				fmt.Sprintf("%q", m.ProductCode),
				m.ProductName,
				m.MakerName,
				formattedSpec,
				"",
			}
			if err := csvWriter.Write(record); err != nil {
				log.Printf("Failed to write product row to CSV (JAN: %s): %v", m.ProductCode, err)
			}
		}
	}
}

// BackupExportHandler は現在の納入価・卸をCSVにバックアップします
func BackupExportHandler(db *sqlx.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		allMasters, err := database.GetAllProductMasters(db)
		if err != nil {
			writeJsonError(w, "Failed to get products for backup export", http.StatusInternalServerError)
			return
		}

		now := time.Now()
		fileName := fmt.Sprintf("納入価・卸バックアップ_%s.csv", now.Format("20060102_150405"))
		fileName = url.PathEscape(fileName)

		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename*=UTF-8''"+fileName)
		w.Write([]byte{0xEF, 0xBB, 0xBF})

		csvWriter := csv.NewWriter(w)
		defer csvWriter.Flush()

		headers := []string{"product_code", "product_name", "maker_name", "package_spec", "purchase_price", "supplier_wholesale"}
		if err := csvWriter.Write(headers); err != nil {
			log.Printf("Failed to write CSV header: %v", err)
		}

		for _, m := range allMasters {
			view := mappers.ToProductMasterView(m)
			formattedSpec := view.FormattedPackageSpec

			record := []string{
				fmt.Sprintf("%q", m.ProductCode),
				m.ProductName,
				m.MakerName,
				formattedSpec,
				strconv.FormatFloat(m.PurchasePrice, 'f', 2, 64),
				m.SupplierWholesale,
			}
			if err := csvWriter.Write(record); err != nil {
				log.Printf("Failed to write product row to CSV (JAN: %s): %v", m.ProductCode, err)
			}
		}
	}
}
