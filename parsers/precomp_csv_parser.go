package parsers

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
)

type ParsedPrecompCSVRecord struct {
	PatientNumber string
	ProductCode   string
	JanQuantity   float64
}

func ParsePrecompCSV(r io.Reader) ([]ParsedPrecompCSVRecord, error) {
	reader := csv.NewReader(SkipBOM(r))
	reader.LazyQuotes = true

	header, err := reader.Read()
	if err == io.EOF {
		return nil, fmt.Errorf("CSVファイルが空です")
	}
	if err != nil {
		return nil, fmt.Errorf("CSVヘッダーの読み取りに失敗: %w", err)
	}

	requiredHeaders := []string{"patient_number", "product_code", "quantity_jan"}
	colIndex, err := getColIndex(header, requiredHeaders)
	if err != nil {
		return nil, err
	}

	var records []ParsedPrecompCSVRecord
	line := 1

	for {
		line++
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("WARN: 予製CSV %d行目の読み取りエラー (スキップ): %v", line, err)
			continue
		}

		get := func(key string) string {
			if idx, ok := colIndex[key]; ok && idx < len(rec) {
				return strings.TrimSpace(rec[idx])
			}
			return ""
		}

		patientNumber := get("patient_number")
		productCode := get("product_code")
		janQty, _ := strconv.ParseFloat(get("quantity_jan"), 64)

		if patientNumber == "" || productCode == "" {
			log.Printf("WARN: 予製CSV %d行目 (患者番号または製品コードが空) (スキップ)", line)
			continue
		}

		records = append(records, ParsedPrecompCSVRecord{
			PatientNumber: patientNumber,
			ProductCode:   productCode,
			JanQuantity:   janQty,
		})
	}

	return records, nil
}
