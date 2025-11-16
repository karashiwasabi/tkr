// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\clients.go (全体)
package database

import (
	"database/sql"
	"fmt"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

// ▼▼▼【ここから追加】GetClientMap ▼▼▼
// GetClientMap は全ての得意先（卸として使用）をマップで取得します。
func GetClientMap(db *sqlx.DB) (map[string]string, error) {
	clients, err := GetAllClients(db)
	if err != nil {
		return nil, fmt.Errorf("failed to get client list for map: %w", err)
	}

	clientMap := make(map[string]string)
	for _, c := range clients {
		clientMap[c.ClientCode] = c.ClientName
	}
	return clientMap, nil
}

// ▲▲▲【追加ここまで】▲▲▲

// UpsertClientInTx は得意先マスタにデータを挿入または置換します。
func UpsertClientInTx(tx *sqlx.Tx, code, name string) error {
	// client_master の client_name は UNIQUE
	// client_code が競合した場合は、client_name を更新します。
	const q = `
		INSERT INTO client_master (client_code, client_name) 
		VALUES (?, ?)
		ON CONFLICT(client_code) DO UPDATE SET
			client_name = excluded.client_name
	`
	_, err := tx.Exec(q, code, name)
	if err != nil {
		// client_name の UNIQUE 制約違反も考慮
		return fmt.Errorf("UpsertClientInTx (Code: %s, Name: %s) failed: %w", code, name, err)
	}
	return nil
}

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
