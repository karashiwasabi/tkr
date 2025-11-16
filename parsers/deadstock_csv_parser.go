// C:\Users\wasab\OneDrive\デスクトップ\TKR\parsers\deadstock_csv_parser.go
package parsers

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

type ParsedDeadStockCSVRecord struct {
	ProductCode string
	Gs1Code     string
	ProductName string
	JanQuantity float64
	ExpiryDate  string
	LotNumber   string
}

func ParseDeadStockCSV(r io.Reader) ([]ParsedDeadStockCSVRecord, error) {
	// ▼▼▼【修正】skipBOM -> SkipBOM ▼▼▼
	reader := csv.NewReader(SkipBOM(r))
	// ▲▲▲【修正ここまで】▲▲▲
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("CSVファイルが空です")
	}
	if err != nil {
		return nil, fmt.Errorf("CSVヘッダーの読み取りに失敗: %w", err)
	}

	colIndex := make(map[string]int)
	for i, colName := range header {
		colIndex[strings.TrimSpace(colName)] = i
	}

	requiredHeaders := []string{"JANコード", "JAN数量"}
	for _, req := range requiredHeaders {
		if _, ok := colIndex[req]; !ok {
			return nil, fmt.Errorf("必須ヘッダーが見つかりません: %s", req)
		}
	}

	var records []ParsedDeadStockCSVRecord
	line := 1

	for {
		line++
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("WARN: CSV %d行目の読み取りエラー (スキップ): %v", line, err)
			continue
		}

		get := func(key string) string {
			if idx, ok := colIndex[key]; ok && idx < len(rec) {
				val := strings.TrimSpace(rec[idx])
				return val
			}
			return ""
		}

		getFloat := func(key string) float64 {
			s := get(key)
			f, _ := strconv.ParseFloat(s, 64)
			return f
		}

		janQty := getFloat("JAN数量")
		if janQty == 0 {
			continue
		}

		parsedRec := ParsedDeadStockCSVRecord{
			ProductCode: get("JANコード"),
			Gs1Code:     get("GS1コード"),
			ProductName: get("品名"),
			JanQuantity: janQty,
			ExpiryDate:  get("期限"),
			LotNumber:   get("ロット"),
		}

		if parsedRec.ProductCode == "" {
			parsedRec.ProductCode = parsedRec.Gs1Code
		}

		records = append(records, parsedRec)
	}
	return records, nil
}
