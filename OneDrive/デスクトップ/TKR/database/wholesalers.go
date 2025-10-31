// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\wholesalers.go (全体)
package database

import (
	"fmt"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

func GetAllWholesalers(db *sqlx.DB) ([]model.Wholesaler, error) {
	var wholesalers []model.Wholesaler
	err := db.Select(&wholesalers, "SELECT wholesaler_code, wholesaler_name FROM wholesalers ORDER BY wholesaler_code")
	if err != nil {
		return nil, fmt.Errorf("failed to get all wholesalers: %w", err)
	}
	return wholesalers, nil
}

func CreateWholesaler(db *sqlx.DB, code, name string) error {
	const q = `INSERT INTO wholesalers (wholesaler_code, wholesaler_name) VALUES (?, ?)`
	_, err := db.Exec(q, code, name)
	if err != nil {
		return fmt.Errorf("CreateWholesaler failed: %w", err)
	}
	return nil
}

func DeleteWholesaler(db *sqlx.DB, code string) error {
	const q = `DELETE FROM wholesalers WHERE wholesaler_code = ?`
	_, err := db.Exec(q, code)
	if err != nil {
		return fmt.Errorf("failed to delete wholesaler with code %s: %w", code, err)
	}
	return nil
}
