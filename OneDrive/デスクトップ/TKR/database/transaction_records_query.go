// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\transaction_records_query.go
package database

import (
	"fmt"
	"strings" // ★ strings をインポート
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

const insertTransactionQuery = `
INSERT INTO transaction_records (
    transaction_date, client_code, receipt_number, line_number, flag,
    jan_code, yj_code, product_name, kana_name, usage_classification, package_form, package_spec, maker_name,
    dat_quantity, jan_pack_inner_qty, jan_quantity, jan_pack_unit_qty, jan_unit_name, jan_unit_code,
    yj_quantity, yj_pack_unit_qty, yj_unit_name, unit_price, purchase_price, supplier_wholesale,
    subtotal, tax_amount, tax_rate, expiry_date, lot_number, flag_poison,
    flag_deleterious, flag_narcotic, flag_psychotropic, flag_stimulant,
    flag_stimulant_raw, process_flag_ma
) VALUES (
    :transaction_date, :client_code, :receipt_number, :line_number, :flag,
    :jan_code, :yj_code, :product_name, :kana_name, :usage_classification, :package_form, :package_spec, :maker_name,
    :dat_quantity, :jan_pack_inner_qty, :jan_quantity, :jan_pack_unit_qty, :jan_unit_name, :jan_unit_code,
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

func DeleteUsageTransactionsInDateRange(tx *sqlx.Tx, minDate, maxDate string) error {
	const q = `DELETE FROM transaction_records WHERE flag = 3 AND transaction_date BETWEEN ? AND ?`
	_, err := tx.Exec(q, minDate, maxDate)
	if err != nil {
		return fmt.Errorf("failed to delete usage transactions: %w", err)
	}
	return nil
}

func GetTransactionsByProductCodes(db *sqlx.DB, productCodes []string) (map[string][]model.TransactionRecord, error) {
	transactionsMap := make(map[string][]model.TransactionRecord)
	if len(productCodes) == 0 {
		return transactionsMap, nil
	}

	const batchSize = 500
	for i := 0; i < len(productCodes); i += batchSize {
		end := i + batchSize
		if end > len(productCodes) {
			end = len(productCodes)
		}
		batch := productCodes[i:end]

		if len(batch) > 0 {
			query, args, err := sqlx.In(`
				SELECT * FROM transaction_records 
				WHERE jan_code IN (?) 
				ORDER BY transaction_date, id`, batch)
			if err != nil {
				return nil, fmt.Errorf("failed to create IN query: %w", err)
			}
			query = db.Rebind(query)

			var transactions []model.TransactionRecord
			err = db.Select(&transactions, query, args...)
			if err != nil {
				return nil, fmt.Errorf("failed to select transactions: %w", err)
			}

			for _, t := range transactions {
				transactionsMap[t.JanCode] = append(transactionsMap[t.JanCode], t)
			}
		}
	}
	return transactionsMap, nil
}

// ▼▼▼【ここから追加】動的AND検索（期限はOR）の関数 ▼▼▼
func SearchTransactions(db *sqlx.DB, janCode string, expiryYYMMDD string, expiryYYMM string, lotNumber string) ([]model.TransactionRecord, error) {
	var transactions []model.TransactionRecord

	conditions := []string{}
	args := []interface{}{}

	if janCode != "" {
		conditions = append(conditions, "jan_code = ?")
		args = append(args, janCode)
	}

	if expiryYYMMDD != "" && expiryYYMM != "" {
		conditions = append(conditions, "(expiry_date = ? OR expiry_date = ?)")
		args = append(args, expiryYYMM)   // 4桁
		args = append(args, expiryYYMMDD) // 6桁
	} else if expiryYYMMDD != "" {
		conditions = append(conditions, "expiry_date = ?")
		args = append(args, expiryYYMMDD)
	} else if expiryYYMM != "" {
		conditions = append(conditions, "expiry_date = ?")
		args = append(args, expiryYYMM)
	}

	if lotNumber != "" {
		conditions = append(conditions, "lot_number = ?")
		args = append(args, lotNumber)
	}

	if len(conditions) == 0 {
		return transactions, nil
	}

	query := `SELECT * FROM transaction_records WHERE `
	query += strings.Join(conditions, " AND ")
	query += ` ORDER BY transaction_date, id`

	err := db.Select(&transactions, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to select transactions by dynamic criteria: %w", err)
	}

	return transactions, nil
}

// ▲▲▲【追加ここまで】▲▲▲
