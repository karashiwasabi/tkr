// C:\Users\wasab\OneDrive\デスクトップ\TKR\parsers\deadstock_csv_parser.go
package parsers

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"strconv"
	"strings"
	// TKRのモデルを参照
)

// ParsedDeadStockCSVRecord はCSVから読み取った棚卸明細です。
type ParsedDeadStockCSVRecord struct {
	// ▼▼▼【ここから修正】CSVの6列に対応 ▼▼▼
	ProductCode string  // JANコード
	Gs1Code     string  // GS1コード
	ProductName string  // 品名
	JanQuantity float64 // JAN数量
	ExpiryDate  string  // 期限
	LotNumber   string  // ロット
	// ▲▲▲【修正ここまで】▲▲▲
}

func skipBOM(r io.Reader) io.Reader {
	br := bufio.NewReader(r)
	bom := []byte{0xEF, 0xBB, 0xBF}
	peeked, err := br.Peek(3)
	if err != nil {
		// BOMである可能性がない（3バイトない）ため、そのままリーダーを返す
		return br
	}
	// 3バイトがBOMと一致するか確認
	isBOM := true
	for i, b := range bom {
		if peeked[i] != b {
			isBOM = false
			break
		}
	}

	if isBOM {
		br.Read(make([]byte, 3)) // 3バイト（BOM）を読み飛ばす
	}
	return br // BOMが除去された（またはBOMでなかった）リーダーを返す
}

// ParseDeadStockCSV は不動在庫CSVを解析し、ParsedDeadStockCSVRecordのスライスを返します。
func ParseDeadStockCSV(r io.Reader) ([]ParsedDeadStockCSVRecord, error) {
	reader := csv.NewReader(skipBOM(r))
	reader.LazyQuotes = true

	// ヘッダー行を読み取り、列インデックスをマップする
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

	// ▼▼▼【ここから修正】必須ヘッダーを日本語に変更 ▼▼▼
	requiredHeaders := []string{"JANコード", "JAN数量"}
	for _, req := range requiredHeaders {
		if _, ok := colIndex[req]; !ok {
			return nil, fmt.Errorf("必須ヘッダーが見つかりません: %s", req)
		}
	}
	// ▲▲▲【修正ここまで】▲▲▲

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

		// ▼▼▼【ここから修正】ヘルパー関数 get を修正 ▼▼▼
		get := func(key string) string {
			if idx, ok := colIndex[key]; ok && idx < len(rec) {
				val := strings.TrimSpace(rec[idx])
				return val
			}
			return ""
		}
		// ▲▲▲【修正ここまで】▲▲▲

		getFloat := func(key string) float64 {
			s := get(key)
			f, _ := strconv.ParseFloat(s, 64)
			return f
		}

		// ▼▼▼【ここから修正】JAN数量が 0 または空の場合はスキップ ▼▼▼
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

		// ProductCodeが空の場合はGS1コードで代替（マスタ検索用）
		if parsedRec.ProductCode == "" {
			parsedRec.ProductCode = parsedRec.Gs1Code
		}
		// ▲▲▲【修正ここまで】▲▲▲

		records = append(records, parsedRec)
	}
	return records, nil
}
