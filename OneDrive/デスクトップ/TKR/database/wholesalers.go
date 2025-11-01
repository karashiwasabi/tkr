// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\wholesalers.go
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

// ▼▼▼【ここから追加】卸コードと卸名のマップを取得する関数 ▼▼▼
func GetWholesalerMap(db *sqlx.DB) (map[string]string, error) {
	wholesalers, err := GetAllWholesalers(db)
	if err != nil {
		return nil, fmt.Errorf("failed to get wholesaler list for map: %w", err)
	}

	wholesalerMap := make(map[string]string)
	for _, w := range wholesalers {
		wholesalerMap[w.WholesalerCode] = w.WholesalerName
	}
	return wholesalerMap, nil
}

// ▲▲▲【追加ここまで】▲▲▲

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
