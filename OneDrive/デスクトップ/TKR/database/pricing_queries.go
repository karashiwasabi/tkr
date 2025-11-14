package database

import (
	"fmt"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

// UpsertProductQuotesInTx は複数の見積データをトランザクション内でUPSERTします。
func UpsertProductQuotesInTx(tx *sqlx.Tx, quotes []model.ProductQuote) error {
	const q = `
		INSERT INTO product_quotes (
			product_code, wholesaler_code, quote_price, quote_date
		) VALUES (
			:product_code, :wholesaler_code, :quote_price, :quote_date
		)
		ON CONFLICT(product_code, wholesaler_code) DO UPDATE SET
			quote_price = excluded.quote_price,
			quote_date = excluded.quote_date
	`
	// NamedExecでスライスをバルクインサート
	_, err := tx.NamedExec(q, quotes)
	if err != nil {
		return fmt.Errorf("failed to bulk upsert product quotes: %w", err)
	}
	return nil
}

// GetAllProductQuotes は全ての見積データを取得し、[product_code][wholesaler_code] -> price のマップを返します。
func GetAllProductQuotes(dbtx DBTX) (map[string]map[string]float64, error) {
	var quotes []model.ProductQuote
	const q = `SELECT product_code, wholesaler_code, quote_price FROM product_quotes`

	err := dbtx.Select(&quotes, q)
	if err != nil {
		return nil, fmt.Errorf("failed to get all product quotes: %w", err)
	}

	resultMap := make(map[string]map[string]float64)
	for _, q := range quotes {
		if _, ok := resultMap[q.ProductCode]; !ok {
			resultMap[q.ProductCode] = make(map[string]float64)
		}
		resultMap[q.ProductCode][q.WholesalerCode] = q.QuotePrice
	}
	return resultMap, nil
}
