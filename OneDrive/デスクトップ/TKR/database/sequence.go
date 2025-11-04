// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\sequence.go (全体)
package database

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/jmoiron/sqlx"
)

func NextSequenceInTx(tx *sqlx.Tx, name, prefix string, padding int) (string, error) {
	var lastNo int
	err := tx.Get(&lastNo, "SELECT last_no FROM code_sequences WHERE name = ?", name)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("sequence '%s' not found", name)
		}
		return "", fmt.Errorf("failed to get sequence '%s': %w", name, err)
	}

	newNo := lastNo + 1
	_, err = tx.Exec("UPDATE code_sequences SET last_no = ? WHERE name = ?", newNo, name)
	if err != nil {
		return "", fmt.Errorf("failed to update sequence '%s': %w", name, err)
	}

	format := fmt.Sprintf("%s%%0%dd", prefix, padding)
	return fmt.Sprintf(format, newNo), nil
}

func InitializeSequenceFromMaxClientCode(tx *sqlx.Tx) error {
	var maxCode sql.NullString
	err := tx.Get(&maxCode, "SELECT client_code FROM client_master ORDER BY client_code DESC LIMIT 1")
	if err != nil {
		if err == sql.ErrNoRows {
			_, err = tx.Exec("UPDATE code_sequences SET last_no = 0 WHERE name = 'CL'")
			return err
		}
		return err
	}
	if maxCode.Valid && strings.HasPrefix(maxCode.String, "CL") {
		numPart := strings.TrimPrefix(maxCode.String, "CL")
		maxNum, _ := strconv.Atoi(numPart)
		_, err = tx.Exec("UPDATE code_sequences SET last_no = ? WHERE name = 'CL'", maxNum)
		return err
	}
	return nil
}

func InitializeSequenceFromMaxYjCode(tx *sqlx.Tx) error {
	var maxYj sql.NullString
	err := tx.Get(&maxYj, "SELECT yj_code FROM product_master WHERE yj_code LIKE 'MA2Y%' ORDER BY yj_code DESC LIMIT 1")
	if err != nil {
		if err == sql.ErrNoRows {
			_, err = tx.Exec("UPDATE code_sequences SET last_no = 0 WHERE name = 'MA2Y'")
			return err
		}
		return err
	}
	if maxYj.Valid && strings.HasPrefix(maxYj.String, "MA2Y") {
		numPart := strings.TrimPrefix(maxYj.String, "MA2Y")
		maxNum, _ := strconv.Atoi(numPart)
		_, err = tx.Exec("UPDATE code_sequences SET last_no = ? WHERE name = 'MA2Y'", maxNum)
		return err
	}
	return nil
}

// ▼▼▼【ここから追加】MA2Jシーケンス初期化関数 (MA2Y  をコピーして修正) ▼▼▼
func InitializeSequenceFromMaxProductCode(tx *sqlx.Tx) error {
	var maxCode sql.NullString
	// product_code カラムから MA2J... の最大値を取得
	err := tx.Get(&maxCode, "SELECT product_code FROM product_master WHERE product_code LIKE 'MA2J%' ORDER BY product_code DESC LIMIT 1")
	if err != nil {
		if err == sql.ErrNoRows {
			// レコードがない場合は0で初期化
			_, err = tx.Exec("UPDATE code_sequences SET last_no = 0 WHERE name = 'MA2J'")
			return err
		}
		return err
	}
	if maxCode.Valid && strings.HasPrefix(maxCode.String, "MA2J") {
		numPart := strings.TrimPrefix(maxCode.String, "MA2J")
		maxNum, _ := strconv.Atoi(numPart)
		// 取得した最大値でシーケンスを更新
		_, err = tx.Exec("UPDATE code_sequences SET last_no = ? WHERE name = 'MA2J'", maxNum)
		return err
	}
	return nil
}

// ▲▲▲【追加ここまで】▲▲▲
