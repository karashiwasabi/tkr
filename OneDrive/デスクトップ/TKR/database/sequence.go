// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\sequence.go
package database

import (
	"database/sql"
	"fmt"
	"log"
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
	_, err = tx.Exec(`UPDATE code_sequences SET last_no = ? WHERE name = ?`, newNo, name)
	if err != nil {
		return "", fmt.Errorf("failed to update sequence '%s': %w", name, err)
	}

	format := fmt.Sprintf("%s%%0%dd", prefix, padding)
	newCode := fmt.Sprintf(format, newNo)

	if name == "MA2Y" || name == "MA2J" {
		log.Printf("INFO: [Sequence] Auto-incrementing '%s'. Fetched last_no: %d. Generated new code: %s", name, lastNo, newCode)
	}

	return newCode, nil
}

// ▼▼▼【ここから修正】ログ出力のロジックを 'CL' にも追加 ▼▼▼
func InitializeSequenceFromMaxClientCode(tx *sqlx.Tx) error {
	var maxCode sql.NullString
	err := tx.Get(&maxCode, "SELECT client_code FROM client_master ORDER BY client_code DESC LIMIT 1")

	maxNum := 0 // デフォルト (該当レコードなしの場合)
	if err != nil {
		if err == sql.ErrNoRows {
			// レコードなし (ErrNoRows自体はエラーではない)
		} else {
			return err // その他のDBエラー
		}
	}

	// レコードが見つかった場合
	if maxCode.Valid && strings.HasPrefix(maxCode.String, "CL") {
		numPart := strings.TrimPrefix(maxCode.String, "CL")
		maxNum, _ = strconv.Atoi(numPart) // maxNum に代入
	}

	// ログを出力
	log.Printf("INFO: [Sequence] Setting 'CL' last_no to %d", maxNum)

	// maxNum を使用
	_, err = tx.Exec(`UPDATE code_sequences SET last_no = ? WHERE name = 'CL'`, maxNum)
	return err
}

// ▲▲▲【修正ここまで】▲▲▲

func InitializeSequenceFromMaxYjCode(tx *sqlx.Tx) error {
	var maxYj sql.NullString
	err := tx.Get(&maxYj,
		"SELECT yj_code FROM product_master WHERE yj_code LIKE 'MA2Y%' ORDER BY yj_code DESC LIMIT 1")

	maxNum := 0 // デフォルト (該当レコードなしの場合)
	if err != nil {
		if err == sql.ErrNoRows {
			// レコードなし (ErrNoRows自体はエラーではない)
		} else {
			return err // その他のDBエラー
		}
	}

	// レコードが見つかった場合
	if maxYj.Valid && strings.HasPrefix(maxYj.String, "MA2Y") {
		numPart := strings.TrimPrefix(maxYj.String, "MA2Y")
		maxNum, _ = strconv.Atoi(numPart)
	}

	// ログを出力（何番に設定しようとしているか）
	log.Printf("INFO: [Sequence] Setting 'MA2Y' last_no to %d", maxNum)

	_, err = tx.Exec(`UPDATE code_sequences SET last_no = ? WHERE name = 'MA2Y'`, maxNum)
	return err
}

func InitializeSequenceFromMaxProductCode(tx *sqlx.Tx) error {
	var maxCode sql.NullString
	// product_code カラムから MA2J... の最大値を取得
	err := tx.Get(&maxCode, "SELECT product_code FROM product_master WHERE product_code LIKE 'MA2J%' ORDER BY product_code DESC LIMIT 1")

	maxNum := 0 // デフォルト (該当レコードなしの場合)
	if err != nil {
		if err == sql.ErrNoRows {
			// レコードなし (ErrNoRows自体はエラーではない)
		} else {
			return err
		}
	}

	// レコードが見つかった場合
	if maxCode.Valid && strings.HasPrefix(maxCode.String, "MA2J") {
		numPart := strings.TrimPrefix(maxCode.String, "MA2J")
		maxNum, _ = strconv.Atoi(numPart)
	}

	// ログを出力（何番に設定しようとしているか）
	log.Printf("INFO: [Sequence] Setting 'MA2J' last_no to %d", maxNum)

	_, err = tx.Exec(`UPDATE code_sequences SET last_no = ? WHERE name = 'MA2J'`, maxNum)
	return err
}
