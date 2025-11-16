// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\transaction_records_query.go
package database

import (
	"database/sql"
	"fmt"
	"strings"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

const TransactionColumns = `
    id, transaction_date, client_code, 

receipt_number, line_number, flag,
    jan_code, yj_code, product_name, kana_name, usage_classification, package_form, package_spec, maker_name,
    dat_quantity, jan_pack_inner_qty, jan_quantity, jan_pack_unit_qty, jan_unit_name, jan_unit_code,
    yj_quantity, yj_pack_unit_qty, yj_unit_name, unit_price, purchase_price, supplier_wholesale,
    subtotal, tax_amount, tax_rate, expiry_date, lot_number, flag_poison,
    flag_deleterious, flag_narcotic, flag_psychotropic, flag_stimulant,
    flag_stimulant_raw, process_flag_ma`

func ScanTransactionRecord(row interface{ Scan(...interface{}) error }) (*model.TransactionRecord, error) {
	var r model.TransactionRecord
	err := row.Scan(
		&r.ID, &r.TransactionDate, &r.ClientCode, &r.ReceiptNumber, &r.LineNumber, &r.Flag,
		&r.JanCode, &r.YjCode, &r.ProductName, &r.KanaName, &r.UsageClassification, &r.PackageForm, &r.PackageSpec, &r.MakerName,
		&r.DatQuantity, &r.JanPackInnerQty, &r.JanQuantity, &r.JanPackUnitQty, &r.JanUnitName, &r.JanUnitCode,
		&r.YjQuantity, &r.YjPackUnitQty, &r.YjUnitName, &r.UnitPrice, &r.PurchasePrice, &r.SupplierWholesale,
		&r.Subtotal,
		&r.TaxAmount,
		&r.TaxRate, &r.ExpiryDate, &r.LotNumber, &r.FlagPoison,
		&r.FlagDeleterious, &r.FlagNarcotic, &r.FlagPsychotropic, &r.FlagStimulant,
		&r.FlagStimulantRaw, &r.ProcessFlagMA,
	)
	if err != nil {
		return nil, err
	}
	return &r, nil
}

const insertTransactionQuery = `
INSERT OR REPLACE INTO transaction_records (
    transaction_date, client_code, receipt_number, line_number, flag,
    jan_code, yj_code, product_name, kana_name, usage_classification, package_form, package_spec, maker_name,
    dat_quantity, jan_pack_inner_qty, jan_quantity, jan_pack_unit_qty, jan_unit_name, jan_unit_code,
    yj_quantity, yj_pack_unit_qty, yj_unit_name, unit_price, purchase_price, supplier_wholesale,
    subtotal, tax_amount, tax_rate, expiry_date, lot_number, flag_poison,
    flag_deleterious, flag_narcotic, flag_psychotropic, flag_stimulant,
    flag_stimulant_raw, process_flag_ma
) VALUES (
    :transaction_date, :client_code, :receipt_number, :line_number, :flag,
    :jan_code, :yj_code, :product_name, :kana_name, :usage_classification, 
:package_form, 
:package_spec, :maker_name,
    :dat_quantity, 
:jan_pack_inner_qty, :jan_quantity, :jan_pack_unit_qty, :jan_unit_name, :jan_unit_code,
   

 :yj_quantity, :yj_pack_unit_qty, :yj_unit_name, :unit_price, :purchase_price, :supplier_wholesale,
    :subtotal, :tax_amount, :tax_rate, :expiry_date, :lot_number, :flag_poison,
    :flag_deleterious, :flag_narcotic, :flag_psychotropic, :flag_stimulant,
    :flag_stimulant_raw, :process_flag_ma
)`

func InsertTransactionRecord(tx *sqlx.Tx, record model.TransactionRecord) error {
	_, err := tx.NamedExec(insertTransactionQuery, record)
	if err != nil {
		return fmt.Errorf("failed to insert transaction record: %w", err)
	}
	return nil
}

const updateTransactionQuery = `
UPDATE transaction_records SET
    transaction_date = :transaction_date,
    client_code = :client_code,
    receipt_number = :receipt_number,
    line_number = :line_number, 
    flag = :flag,
 
   jan_code = 
:jan_code,
    yj_code = :yj_code,
   

 product_name = :product_name,
    kana_name = :kana_name,
    usage_classification = :usage_classification,
    package_form = :package_form,
    package_spec = :package_spec,
    maker_name = :maker_name,
    dat_quantity = :dat_quantity,
    jan_pack_inner_qty = :jan_pack_inner_qty,
    jan_quantity = :jan_quantity,
    jan_pack_unit_qty = :jan_pack_unit_qty,
    jan_unit_name = :jan_unit_name,
    jan_unit_code = :jan_unit_code,
    yj_quantity = :yj_quantity,
    
yj_pack_unit_qty = :yj_pack_unit_qty, 
    
 yj_unit_name = :yj_unit_name,
   
     unit_price = :unit_price,
 

   purchase_price = :purchase_price,
    supplier_wholesale = :supplier_wholesale,
    subtotal = :subtotal,
    tax_amount = :tax_amount,
    tax_rate = :tax_rate,
    expiry_date = :expiry_date,
    lot_number = :lot_number,
    flag_poison = :flag_poison,
    flag_deleterious = :flag_deleterious,
    flag_narcotic = :flag_narcotic,
    flag_psychotropic = :flag_psychotropic,
    flag_stimulant = :flag_stimulant,
    flag_stimulant_raw = :flag_stimulant_raw,
    process_flag_ma = :process_flag_ma
WHERE id = :id`

func UpdateFullTransactionInTx(tx *sqlx.Tx, rec *model.TransactionRecord) error {
	_, err := tx.NamedExec(updateTransactionQuery, rec)
	if err !=
		nil {
		return fmt.Errorf("UpdateFullTransactionInTx failed for ID %d: %w", rec.ID, err)
	}
	return nil
}

func PersistTransactionRecordsInTx(tx *sqlx.Tx, records []model.TransactionRecord) error {
	stmt, err := tx.PrepareNamed(insertTransactionQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare statement for transaction_records: %w", err)
	}
	defer stmt.Close()

	for _, rec := range records {
		_, err = stmt.Exec(rec)
		if err != nil {
			return fmt.Errorf("failed to exec statement for transaction_records (JAN: %s): %w", rec.JanCode, err)
		}
	}
	return nil
}

func DeleteUsageTransactionsInDateRange(tx *sqlx.Tx, minDate, maxDate string) error {
	// ▼▼▼【ここを修正】flag = '2' -> '3' (処方) に変更 ▼▼▼
	const q = `DELETE FROM transaction_records WHERE flag = '3' AND transaction_date BETWEEN ?
AND ?`
	// ▲▲▲【修正ここまで】▲▲▲
	_, err := tx.Exec(q, minDate, maxDate)
	if err != nil {
		return fmt.Errorf("failed to delete usage transactions in date range: %w", err)
	}
	return nil
}

func GetTransactionsByProductCodes(db *sqlx.DB, productCodes []string) (map[string][]model.TransactionRecord, error) {
	transactionsByProductCode := make(map[string][]model.TransactionRecord)
	if len(productCodes) == 0 {
		return transactionsByProductCode, nil
	}

	batchSize := 100

	for i := 0; i < len(productCodes); i += batchSize {
		end := i + batchSize
		if end > len(productCodes) {
			end = len(productCodes)
		}
		batch := productCodes[i:end]

		var records []model.TransactionRecord
		query, args, err := sqlx.In(`SELECT `+TransactionColumns+` FROM transaction_records WHERE jan_code IN (?)`, batch)
		if err != nil {
			return nil, fmt.Errorf("failed to create IN query for batch: %w", err)
		}
		query = db.Rebind(query)
		err = db.Select(&records, query, args...)
		if err != nil {
			return nil, fmt.Errorf("failed to select transactions for batch: %w", err)
		}

		for _, r := range records {
			transactionsByProductCode[r.JanCode] = append(transactionsByProductCode[r.JanCode], r)
		}
	}

	return transactionsByProductCode, nil
}

func SearchTransactions(db *sqlx.DB, janCode string, expiryYYYYMM string, lotNumber string) ([]model.TransactionRecord, error) {
	var records []model.TransactionRecord

	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT ")
	queryBuilder.WriteString(TransactionColumns)
	queryBuilder.WriteString(" FROM transaction_records WHERE flag IN (1, 2)")

	args :=
		[]interface{}{}

	if janCode != "" {
		queryBuilder.WriteString(" AND jan_code = ?")
		args = append(args, janCode)
	}

	if expiryYYYYMM != "" {
		queryBuilder.WriteString(" AND expiry_date = ?")
		args = append(args, expiryYYYYMM)
	}

	if lotNumber != "" {
		queryBuilder.WriteString(" AND lot_number = ?")
		args = append(args, lotNumber)
	}

	queryBuilder.WriteString(" ORDER BY transaction_date DESC, id DESC")

	err := db.Select(&records, queryBuilder.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search transactions: %w", err)
	}
	return records, nil
}

func GetReceiptNumbersByDate(db *sqlx.DB, date string, prefix string, clientCode string) ([]string, error) {
	var numbers []string

	query := "SELECT DISTINCT receipt_number FROM transaction_records"
	conditions := []string{"transaction_date = ?", "receipt_number LIKE ?"}
	args := []interface{}{date, prefix +
		"%"}

	if clientCode != "" {
		conditions = append(conditions, "client_code = ?")
		args = append(args, clientCode)
	}

	query +=
		" WHERE " + strings.Join(conditions, " AND ")
	query += " ORDER BY receipt_number"

	err := db.Select(&numbers, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipt numbers by date: %w", err)
	}
	return numbers, nil
}

func GetTransactionsByReceiptNumber(db *sqlx.DB, receiptNumber string) ([]model.TransactionRecord, error) {
	var records []model.TransactionRecord
	q := `SELECT ` + TransactionColumns + ` FROM transaction_records WHERE receipt_number = ?
ORDER BY line_number`

	err := db.Select(&records, q, receiptNumber)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions by receipt number: %w", err)
	}
	return records, nil
}

func DeleteTransactionsByReceiptNumberInTx(tx *sqlx.Tx, receiptNumber string) error {
	q := `DELETE FROM transaction_records WHERE receipt_number = ?`
	_, err := tx.Exec(q, receiptNumber)
	if err != nil {
		return fmt.Errorf("failed to delete transactions for receipt %s: %w", receiptNumber, err)
	}
	return nil
}

func GetLatestInventoryDetailsByYjCode(dbtx DBTX, yjCode string) ([]model.TransactionRecord, error) {
	var latestInventoryDate sql.NullString
	err := dbtx.Get(&latestInventoryDate, `
		SELECT MAX(last_inventory_date) 
		FROM package_stock 
		WHERE yj_code = ?`,
		yjCode)

	if err != nil && err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to get latest inventory date from package_stock for %s: %w", yjCode, err)
	}

	if !latestInventoryDate.Valid ||
		latestInventoryDate.String == "" {
		return []model.TransactionRecord{}, nil
	}

	var records []model.TransactionRecord
	q := `SELECT ` + TransactionColumns + ` 
		  FROM transaction_records 
		  WHERE yj_code = ?
AND transaction_date = ? 
		  AND flag = 0 
		  ORDER BY jan_code, expiry_date, lot_number`

	err = dbtx.Select(&records, q, yjCode, latestInventoryDate.String)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest inventory (flag=0) transactions for %s on %s: %w", yjCode, latestInventoryDate.String, err)
	}

	return records, nil
}
