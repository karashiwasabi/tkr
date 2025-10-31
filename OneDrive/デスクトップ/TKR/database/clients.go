// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\clients.go (全体)
package database

import (
	"database/sql"
	"fmt"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

func CreateClientInTx(tx *sqlx.Tx, code, name string) error {
	const q = `INSERT INTO client_master (client_code, client_name) VALUES (?, ?)`
	_, err := tx.Exec(q, code, name)
	if err != nil {
		return fmt.Errorf("CreateClientInTx failed: %w", err)
	}
	return nil
}

func CheckClientExistsByName(tx *sqlx.Tx, name string) (bool, error) {
	var exists int
	const q = `SELECT 1 FROM client_master WHERE client_name = ? LIMIT 1`
	err := tx.QueryRow(q, name).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("CheckClientExistsByName failed: %w", err)
	}
	return true, nil
}

func GetAllClients(db *sqlx.DB) ([]model.Client, error) {
	var clients []model.Client
	err := db.Select(&clients, "SELECT client_code, client_name FROM client_master ORDER BY client_code")
	if err != nil {
		return nil, fmt.Errorf("failed to get all clients: %w", err)
	}
	return clients, nil
}
