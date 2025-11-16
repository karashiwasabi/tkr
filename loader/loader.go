package loader

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"
	"tkr/database"

	"github.com/jmoiron/sqlx" // sqlx をインポート
	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// 各テーブルのカラムのデータ型情報 (CSVからの変換用)
// キーは CSV の列インデックス (0始まり)
var tableSchemas = map[string]map[int]string{
	"jcshms": { // JCSHMS.CSV
		44:  "real",    // JC044 (列インデックス 44)
		49:  "real",    // JC049
		50:  "real",    // JC050
		61:  "integer", // JC061
		62:  "integer", // JC062
		63:  "integer", // JC063
		64:  "integer", // JC064
		65:  "integer", // JC065
		66:  "integer", // JC066
		124: "real",    // JC124
	},
	"jancode": { // JANCODE.CSV
		6: "real", // JA006 (列インデックス 6)
		8: "real", // JA008
	},
}

// InitDatabase はデータベーススキーマを適用し、マスターCSVをロードします。
func InitDatabase(db *sqlx.DB) error {
	log.Println("Applying database schema...")
	if err := applySchema(db); err != nil {
		return fmt.Errorf("failed to apply schema.sql: %w", err)
	}
	log.Println("Schema applied successfully.")

	// CSVファイルのパス (SOUフォルダは TKR フォルダ直下に配置する想定)
	jcshmsPath := "SOU/JCSHMS.CSV"
	jancodePath := "SOU/JANCODE.CSV"
	taniPath := "SOU/TANI.CSV" // TANI.CSV も追加

	// ファイル存在チェック (任意ですが、エラーメッセージが分かりやすくなります)
	if _, err := os.Stat(jcshmsPath); os.IsNotExist(err) {
		log.Printf("WARN: %s not found, skipping.", jcshmsPath)
	} else {
		log.Printf("Loading %s...", jcshmsPath)
		// JCSHMS は 125 列, ヘッダーなし
		if err := LoadCSV(db, jcshmsPath, "jcshms", 125, false); err != nil {
			return fmt.Errorf("failed to load %s: %w", jcshmsPath, err)
		}
		log.Printf("Loaded %s successfully.", jcshmsPath)
	}

	if _, err := os.Stat(jancodePath); os.IsNotExist(err) {
		log.Printf("WARN: %s not found, skipping.", jancodePath)
	} else {
		log.Printf("Loading %s...", jancodePath)
		// JANCODE は 30 列, ヘッダーあり(スキップ)
		if err := LoadCSV(db, jancodePath, "jancode", 30, true); err != nil {
			return fmt.Errorf("failed to load %s: %w", jancodePath, err)
		}
		log.Printf("Loaded %s successfully.", jancodePath)
	}

	// TANI.CSV のロード処理も追加（必要に応じて）
	if _, err := os.Stat(taniPath); os.IsNotExist(err) {
		log.Printf("WARN: %s not found, skipping TANI units loading.", taniPath)
	} else {
		// TANI.CSV は units パッケージ側でロードするかもしれません。
		// ここでロードする場合は、適切なテーブル定義と LoadCSV 呼び出しを追加します。
		log.Printf("Note: TANI.CSV exists but loading logic is not implemented here yet.")
	}

	// ▼▼▼【ここから追加】シーケンスの初期化 ▼▼▼
	// (LoadCSVの後、DB接続が確立しているこのタイミングでシーケンスを初期化)
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction for sequence initialization: %w", err)
	}
	defer tx.Rollback() // エラー時

	if err := database.InitializeSequenceFromMaxYjCode(tx); err != nil {
		log.Printf("WARN: Failed to initialize MA2Y sequence: %v", err)
		// エラーでも続行
	}
	if err := database.InitializeSequenceFromMaxProductCode(tx); err != nil {
		log.Printf("WARN: Failed to initialize MA2J sequence: %v", err)
		// エラーでも続行
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit sequence initialization: %w", err)
	}
	log.Println("Code sequences initialized.")
	// ▲▲▲【追加ここまで】▲▲▲

	return nil
}

// applySchema は schema.sql ファイルを読み込んで実行します。
func applySchema(db *sqlx.DB) error {
	schemaBytes, err := os.ReadFile("schema.sql")
	if err != nil {
		return fmt.Errorf("could not read schema.sql: %w", err)
	}
	// sqlx.DB でも Exec をそのまま使えます
	_, err = db.Exec(string(schemaBytes))
	if err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}
	return nil
}

// LoadCSV は指定されたCSVファイルを読み込み、指定テーブルにデータを挿入（または置換）します。
// ▼▼▼【ここを修正】 名前付き返り値 `(err error)` を使用 ▼▼▼
func LoadCSV(db *sqlx.DB, filepath, tablename string, expectedColumns int, skipHeader bool) (err error) {
	f, err := os.Open(filepath)
	if err != nil {
		return fmt.Errorf("could not open file %s: %w", filepath, err)
	}
	defer f.Close()

	// Shift-JISデコーダーを設定
	r := csv.NewReader(transform.NewReader(f, japanese.ShiftJIS.NewDecoder()))
	r.LazyQuotes = true    // ダブルクォートが不完全でも許容
	r.FieldsPerRecord = -1 // 可変長カラムを許容 (後でチェック)

	// ヘッダー行をスキップする場合
	if skipHeader {
		if _, err := r.Read(); err != nil && err != io.EOF {
			return fmt.Errorf("failed to skip header in %s: %w", filepath, err)
		}
	}

	// sqlx.Tx を使ってトランザクションを開始
	tx, err := db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// ▼▼▼【ここを修正】 defer ロジックを linter に優しい形に変更 ▼▼▼
	defer func() {
		if p := recover(); p != nil {
			// パニックが発生した場合
			tx.Rollback()
			panic(p) // パニックを再スロー
		} else if err != nil {
			// 関数がエラーを返そうとしている場合 (err が nil でない)
			log.Printf("Rolling back transaction for %s due to error: %v", tablename, err)
			tx.Rollback() // ロールバック
		} else {
			// 関数がエラーなしで終了しようとしている場合 (err が nil)
			err = tx.Commit() // コミットし、結果を名前付き返り値 err に代入
			if err != nil {
				log.Printf("Error committing transaction for %s: %v", tablename, err)
			}
		}
	}()
	// ▲▲▲【修正ここまで】▲▲▲

	// INSERT OR REPLACE 文を準備
	placeholders := strings.Repeat("?,", expectedColumns-1) + "?"
	query := fmt.Sprintf("INSERT OR REPLACE INTO %s VALUES (%s)", tablename, placeholders)
	// sqlx.Tx でも Prepare をそのまま使えます
	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement for %s: %w", tablename, err) // err を返す
	}
	defer stmt.Close()

	// テーブルスキーマ情報を取得 (型変換用)
	schema := tableSchemas[tablename]
	rowCount := 0

	for {
		row, readErr := r.Read()
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			log.Printf("WARN: Error reading row in %s (skipping): %v", filepath, readErr)
			continue // エラー行はスキップして処理を続行
		}

		// カラム数が期待値と異なる場合はスキップ
		if len(row) < expectedColumns {
			// log.Printf("WARN: Skipping row in %s due to insufficient columns (expected %d, got %d): %v", filepath, expectedColumns, len(row), row)
			continue
		}
		// カラム数が多すぎる場合は、期待される数に切り詰める
		if len(row) > expectedColumns {
			row = row[:expectedColumns]
		}

		// SQLステートメントに渡す引数スライスを作成
		args := make([]interface{}, expectedColumns)
		for i := 0; i < expectedColumns; i++ {
			val := strings.TrimSpace(row[i]) // 前後の空白を除去

			// スキーマ情報に基づいて型変換を試みる
			if colType, ok := schema[i]; ok {
				switch colType {
				case "real":
					num, parseErr := strconv.ParseFloat(val, 64)
					if parseErr != nil {
						args[i] = 0.0 // パース失敗時は 0.0
					} else {
						args[i] = num
					}
				case "integer":
					num, parseErr := strconv.ParseInt(val, 10, 64)
					if parseErr != nil {
						args[i] = 0 // パース失敗時は 0
					} else {
						args[i] = num
					}
				default: // "text" or unknown type
					args[i] = val
				}
			} else {
				// スキーマ情報がない場合は文字列として扱う
				args[i] = val
			}
		}

		// 準備されたステートメントを実行
		if _, execErr := stmt.Exec(args...); execErr != nil {
			log.Printf("WARN: Failed to insert row into %s (skipping): %v | Data: %v", tablename, execErr, args)
			err = fmt.Errorf("failed to execute statement for %s: %w", tablename, execErr) // err に代入
			return err                                                                     // エラーを返す (defer が Rollback を実行する)
		}
		rowCount++
	}

	log.Printf("Inserted or replaced %d rows into %s", rowCount, tablename)

	// 成功時は err = nil のまま defer が実行され、Commit される
	return nil
}
