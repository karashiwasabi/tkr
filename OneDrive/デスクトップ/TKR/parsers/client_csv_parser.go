// C:\Users\wasab\OneDrive\デスクトップ\TKR\parsers\client_csv_parser.go
package parsers

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"strings"
)

// ParsedClientCSVRecord は得意先CSVの1行を表します。
type ParsedClientCSVRecord struct {
	ClientCode string
	ClientName string
}

// ParseClientCSV は得意先マスタCSVを解析します。
func ParseClientCSV(r io.Reader) ([]ParsedClientCSVRecord, error) {
	// SkipBOM [cite: 4504-4505] を使用
	reader := csv.NewReader(SkipBOM(r))
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("CSVファイルが空です")
	}
	if err != nil {
		return nil, fmt.Errorf("CSVヘッダーの読み取りに失敗: %w", err)
	}

	// getColIndex [cite: 4505-4506] を使用
	requiredHeaders := []string{"client_code", "client_name"}
	colIndex, err := getColIndex(header, requiredHeaders)
	if err != nil {
		return nil, err
	}

	var records []ParsedClientCSVRecord
	line := 1

	for {
		line++
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("WARN: 得意先CSV %d行目の読み取りエラー (スキップ): %v", line, err)
			continue
		}

		get := func(key string) string {
			if idx, ok := colIndex[key]; ok && idx < len(rec) {
				return strings.TrimSpace(rec[idx])
			}
			return ""
		}

		clientCode := get("client_code")
		clientName := get("client_name")

		if clientCode == "" || clientName == "" {
			log.Printf("WARN: 得意先CSV %d行目 (コードまたは名称が空) (スキップ)", line)
			continue
		}

		records = append(records, ParsedClientCSVRecord{
			ClientCode: clientCode,
			ClientName: clientName,
		})
	}

	return records, nil
}
