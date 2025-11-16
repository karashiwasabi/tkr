// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\guided_inventory.go
package database

import (
	"database/sql"
	"fmt"
	"strconv"
	"tkr/mappers"
	"tkr/model"
	"tkr/units"

	"github.com/jmoiron/sqlx"
)

func DeleteTransactionsByFlagAndDateAndCodes(tx *sqlx.Tx, flag int, date string, productCodes []string) error {
	if len(productCodes) == 0 {
		return nil
	}

	query, args, err := sqlx.In(`DELETE FROM transaction_records WHERE flag = ?
AND transaction_date = ? AND jan_code IN (?)`, flag, date, productCodes)
	if err != nil {
		return fmt.Errorf("failed to create IN query for deleting transactions: %w", err)
	}
	query = tx.Rebind(query)

	_, err = tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete transactions by flag, date, and codes: %w", err)
	}
	return nil
}

func SaveGuidedInventoryData(tx *sqlx.Tx, date string, yjCode string, allPackagings []*model.ProductMaster, inventoryData map[string]float64, deadstockData []model.DeadStockRecord) error {

	var allProductCodes []string
	mastersMap := make(map[string]*model.ProductMaster)
	for _, pkg := range allPackagings {
		allProductCodes = append(allProductCodes, pkg.ProductCode)
		mastersMap[pkg.ProductCode] = pkg
	}

	if len(allProductCodes) > 0 {
		if err := DeleteTransactionsByFlagAndDateAndCodes(tx, 0, date, allProductCodes); err != nil {
			return fmt.Errorf("failed to delete old inventory records for the same day: %w", err)
		}
	}

	var lastSeq int
	var dateYYMMDD string
	if len(date) >= 8 {
		dateYYMMDD = date[2:8]
	} else if len(date) == 6 {
		dateYYMMDD = date
	} else {
		return fmt.Errorf("invalid date format for receipt number: %s", date)
	}

	prefix := "AJ" + dateYYMMDD

	q := `SELECT receipt_number FROM transaction_records 
		  WHERE receipt_number LIKE ?
ORDER BY receipt_number DESC LIMIT 1`
	var lastReceiptNumber string
	err := tx.Get(&lastReceiptNumber, q, prefix+"%")

	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to get last receipt number sequence for prefix %s: %w", prefix, err)
	}

	lastSeq = 0
	if lastReceiptNumber != "" {
		if len(lastReceiptNumber) == 13 {
			seqStr := lastReceiptNumber[8:]
			lastSeq, _ = strconv.Atoi(seqStr)
		}
	}

	newSeq := lastSeq + 1

	packageStockTotalsYj := make(map[string]float64)

	for productCode, janQty := range inventoryData {
		master, ok := mastersMap[productCode]
		if !ok {
			continue
		}
		yjQty := janQty * master.JanPackInnerQty
		packageKey := fmt.Sprintf("%s|%s|%g|%s", master.YjCode, master.PackageForm, master.JanPackInnerQty, units.ResolveName(master.YjUnitName))
		packageStockTotalsYj[packageKey] += yjQty

	}

	for i, ds := range deadstockData {
		master, ok := mastersMap[ds.ProductCode]
		if !ok {
			continue
		}

		receiptNumber := fmt.Sprintf("%s%05d", prefix, newSeq+i)

		janQty := ds.StockQuantityJan
		yjQty := janQty * master.JanPackInnerQty

		tr := model.TransactionRecord{
			TransactionDate: date,
			Flag:            0,
			ReceiptNumber:   receiptNumber,
			LineNumber:      fmt.Sprintf("%d", i+1),
			JanCode:         master.ProductCode,
			YjCode:          master.YjCode,
			JanQuantity:     janQty,
			YjQuantity:      yjQty,

			ExpiryDate: ds.ExpiryDate,
			LotNumber:  ds.LotNumber,
		}

		tr.UnitPrice = master.NhiPrice
		tr.Subtotal = tr.YjQuantity * tr.UnitPrice

		mappers.MapMasterToTransaction(&tr, master)

		if err := InsertTransactionRecord(tx, tr); err != nil {
			return fmt.Errorf("failed to insert inventory record for %s: %w", ds.ProductCode, err)
		}
	}

	for key, totalYjQty := range packageStockTotalsYj {
		if err := UpsertPackageStockInTx(tx, key, yjCode, totalYjQty, date); err != nil {
			return fmt.Errorf("failed to upsert package_stock for key %s: %w", key, err)
		}
	}

	return nil
}
