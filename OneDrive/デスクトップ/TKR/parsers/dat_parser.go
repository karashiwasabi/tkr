package parsers

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
	"tkr/model" // model パッケージをインポート

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// ParseDat は、固定長のDATファイルからレコードを抽出し、DatRecord のスライスを返します。
func ParseDat(r io.Reader) ([]model.DatRecord, error) {
	// ShiftJIS から UTF-8 へのデコーダーを用意
	decoder := japanese.ShiftJIS.NewDecoder()
	utf8Reader := transform.NewReader(r, decoder)
	scanner := bufio.NewScanner(utf8Reader)

	var records []model.DatRecord
	var currentWholesale string

	for scanner.Scan() {
		line := scanner.Text() // UTF-8 に変換された行
		if len(line) == 0 {
			continue
		}

		// rune スライスに変換してバイト位置ではなく文字位置でアクセス
		runes := []rune(line)

		// TrimSpaceAndGet は、指定範囲の文字列を取得し、前後の空白を除去するヘルパー
		trimSpaceAndGet := func(start, end int) string {
			if start >= len(runes) {
				return ""
			}
			if end > len(runes) {
				end = len(runes)
			}
			return strings.TrimSpace(string(runes[start:end]))
		}

		// parseFloat は文字列を float64 に変換 (エラー時は 0.0)
		parseFloat := func(s string) float64 {
			f, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
			return f
		}
		// parseInt は文字列を int に変換 (エラー時は 0)
		parseInt := func(s string) int {
			i, _ := strconv.Atoi(strings.TrimSpace(s))
			return i
		}

		switch trimSpaceAndGet(0, 1) {
		case "S":
			// Sレコードから卸コードを取得 (文字位置 3 から 13 未満)
			currentWholesale = trimSpaceAndGet(3, 13)
		case "D":
			// Dレコードをパース
			flag := parseInt(trimSpaceAndGet(3, 4))
			date := trimSpaceAndGet(4, 12)
			receiptNumber := trimSpaceAndGet(12, 22)
			lineNumber := trimSpaceAndGet(22, 24)
			janCode := trimSpaceAndGet(25, 38)
			productName := trimSpaceAndGet(38, 78) // UTF-8になっているはず
			datQuantity := parseFloat(trimSpaceAndGet(78, 83))
			unitPrice := parseFloat(trimSpaceAndGet(83, 92))
			subtotal := parseFloat(trimSpaceAndGet(92, 101))
			// 期限とロット (文字位置)
			expiryDate := trimSpaceAndGet(109, 115) // YYMMDD or YYYYMM
			lotNumber := trimSpaceAndGet(115, 121)

			rec := model.DatRecord{
				ClientCode:    currentWholesale,
				Flag:          flag,
				Date:          date,
				ReceiptNumber: receiptNumber,
				LineNumber:    lineNumber,
				JanCode:       janCode,
				ProductName:   productName,
				DatQuantity:   datQuantity,
				UnitPrice:     unitPrice,
				Subtotal:      subtotal,
				ExpiryDate:    expiryDate,
				LotNumber:     lotNumber,
			}
			records = append(records, rec)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading DAT file: %w", err)
	}
	return records, nil
}
