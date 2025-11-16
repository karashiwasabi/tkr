// C:\Users\wasab\OneDrive\デスクトップ\TKR\aggregation\helpers.go
package aggregation

import (
	"fmt"
	"tkr/database"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

func GetFilteredMastersAndYjCodes(conn *sqlx.DB, filters model.AggregationFilters) (map[string][]*model.ProductMaster, []string, error) {
	query := `SELECT * FROM product_master p WHERE 1=1 ` //
	var args []interface{}
	if filters.YjCode != "" { //
		query += " AND p.yj_code = ? " //
		args = append(args, filters.YjCode)
	}
	if filters.KanaName != "" {
		query += " AND p.kana_name LIKE ? "       //
		args = append(args, filters.KanaName+"%") //
	}
	if filters.GenericName != "" {
		query += " AND p.generic_name LIKE ?" //
		args = append(args, "%"+filters.GenericName+"%")
	}
	if filters.DosageForm != "" && filters.DosageForm != "all" {
		query += " AND p.usage_classification = ?" //
		args = append(args, filters.DosageForm)
	}
	if filters.ShelfNumber != "" {
		query += " AND p.shelf_number LIKE ? " //
		args = append(args, "%"+filters.ShelfNumber+"%")
	}

	query += " ORDER BY p.kana_name " //

	rows, err := conn.Queryx(query, args...)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	mastersByYjCode := make(map[string][]*model.ProductMaster)
	yjCodeMap := make(map[string]bool)
	var yjCodes []string //

	for rows.Next() {
		var m model.ProductMaster
		if err := rows.StructScan(&m); err != nil {
			return nil, nil, err
		}
		if m.YjCode != "" {
			mastersByYjCode[m.YjCode] = append(mastersByYjCode[m.YjCode], &m) //
			if !yjCodeMap[m.YjCode] {
				yjCodeMap[m.YjCode] = true
				yjCodes = append(yjCodes, m.YjCode) //
			}
		}
	}

	return mastersByYjCode, yjCodes, nil
}

func getAllProductCodesForYjCodes(conn *sqlx.DB, yjCodes []string) ([]string, error) { //
	if len(yjCodes) == 0 {
		return []string{}, nil
	}
	query, args, err := sqlx.In(`
		SELECT DISTINCT product_code FROM product_master WHERE yj_code IN (?)
		UNION
		SELECT DISTINCT jan_code FROM transaction_records WHERE yj_code IN (?)`, yjCodes, yjCodes) //
	if err != nil {
		return nil, fmt.Errorf("failed to create IN query for all product codes: %w", err)
	}
	query = conn.Rebind(query)
	var codes []string
	if err := conn.Select(&codes, query, args...); err != nil { //
		return nil, err
	}
	var validCodes []string
	for _, code := range codes {
		if code != "" {
			validCodes = append(validCodes, code)
		}
	}
	return validCodes, nil
}

func getTransactionsByProductCodes(conn *sqlx.DB, productCodes []string) (map[string][]model.TransactionRecord, error) { //
	transactionsMap := make(map[string][]model.TransactionRecord)
	if len(productCodes) == 0 {
		return transactionsMap, nil
	}
	const batchSize = 500
	for i := 0; i < len(productCodes); i += batchSize { //
		end := i + batchSize
		if end > len(productCodes) {
			end = len(productCodes)
		}
		batch := productCodes[i:end]
		if len(batch) > 0 {
			query, args, err := sqlx.In("SELECT "+database.TransactionColumns+" FROM transaction_records WHERE jan_code IN (?) ORDER BY transaction_date, id", batch) //
			if err != nil {
				return nil, err
			}
			query = conn.Rebind(query)
			rows, err := conn.Query(query, args...)
			if err != nil {
				return nil, err
			}
			for rows.Next() {
				t, err := database.ScanTransactionRecord(rows)
				if err != nil {
					rows.Close()
					return nil, err
				}
				transactionsMap[t.JanCode] = append(transactionsMap[t.JanCode], *t)
			}
			rows.Close()
		}
	}
	return transactionsMap, nil
}
