// C:\Users\wasab\OneDrive\デスクトップ\TKR\parsers\stock_csv_parser.go
package parsers

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

type ParsedExternalStockCSVRecord struct {
	JanCode     string
	JanQuantity float64
	ExpiryDate  string
	LotNumber   string
}

type ParsedTKRStockCSVRecord struct {
	PackageKey  string
	ProductName string
	JanQuantity float64
}

func ParseExternalStockCSV(r io.Reader) ([]ParsedExternalStockCSVRecord, error) {
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

	requiredHeaders := []string{"JANコード", "JAN数量"}
	colIndex, err := getColIndex(header, requiredHeaders)
	if err != nil {
		return nil, err
	}

	idxExpiry, hasExpiry := colIndex["期限"]
	idxLot, hasLot := colIndex["ロット"]

	var records []ParsedExternalStockCSVRecord
	line := 1
	for {
		line++
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("WARN: CSV %d行目の読み取りエラー (スキップ): %v", line,
				err)
			continue
		}

		get := func(idx int) string {
			if idx < len(rec) {
				return strings.TrimSpace(rec[idx])
			}
			return ""
		}

		janQty, _ := strconv.ParseFloat(get(colIndex["JAN数量"]), 64)
		if janQty <= 0 {
			continue
		}

		parsedRec := ParsedExternalStockCSVRecord{
			JanCode:     get(colIndex["JANコード"]),
			JanQuantity: janQty,
		}

		if hasExpiry {
			parsedRec.ExpiryDate = get(idxExpiry)
		}
		if hasLot {
			parsedRec.LotNumber = get(idxLot)
		}

		records = append(records, parsedRec)
	}
	return records, nil
}

func ParseTKRStockCSV(r io.Reader) ([]ParsedTKRStockCSVRecord, error) {
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

	requiredHeaders := []string{"PackageKey", "JAN数量"}
	colIndex, err := getColIndex(header, requiredHeaders)
	if err != nil {
		return nil, err
	}

	var records []ParsedTKRStockCSVRecord
	line := 1
	for {
		line++
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("WARN: CSV %d行目の読み取りエラー (スキップ): %v",
				line, err)
			continue
		}

		get := func(idx int) string {
			if idx < len(rec) {
				return strings.TrimSpace(rec[idx])
			}
			return ""
		}

		janQty, _ := strconv.ParseFloat(get(colIndex["JAN数量"]), 64)

		parsedRec := ParsedTKRStockCSVRecord{
			PackageKey:  get(colIndex["PackageKey"]),
			JanQuantity: janQty,
		}

		records = append(records, parsedRec)
	}
	return records, nil
}
