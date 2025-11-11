// C:\Users\wasab\OneDrive\デスクトップ\TKR\parsers\dat_parser.go
package parsers

import (
	"bufio"
	"bytes" // bytes パッケージを追加
	"fmt"
	"io"
	"strconv"
	"strings"   // strings パッケージを追加
	"tkr/model" // model パッケージをインポート

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

// ▼▼▼【ここから修正】UTF-8に事前変換せず、バイトスライスとしてパースする ▼▼▼

// sjisToUTF8 は Shift-JIS のバイトスライスを UTF-8 文字列に変換します。
func sjisToUTF8(b []byte) string {
	// 0x00 (NULL文字) はトリム対象の空白とはみなされないため、
	// 先に ReplaceAll で除去してから TrimSpace をかける
	trimmedBytes := bytes.Trim(b, "\x00 ") // NULL文字と空白を除去
	utf8Bytes, _, _ := transform.Bytes(japanese.ShiftJIS.NewDecoder(), trimmedBytes)
	return string(utf8Bytes)
}

// trimSpaceAndGetString は、バイトスライスの指定範囲を文字列として取得し、空白とNULL文字を除去します。
func trimSpaceAndGetString(line []byte, start, end int) string {
	if start >= len(line) {
		return ""
	}
	if end > len(line) {
		end = len(line)
	}
	// 0x00 (NULL文字) と 0x20 (スペース) を両端から除去
	trimmedSlice := bytes.Trim(line[start:end], "\x00 ")
	return string(trimmedSlice)
}

// parseFloat は、バイトスライスの指定範囲を float64 に変換します (エラー時は 0.0)
func parseFloat(line []byte, start, end int) float64 {
	s := trimSpaceAndGetString(line, start, end)
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// parseInt は、バイトスライスの指定範囲を int に変換します (エラー時は 0)
func parseInt(line []byte, start, end int) int {
	s := trimSpaceAndGetString(line, start, end)
	i, _ := strconv.Atoi(s)
	return i
}

// ▼▼▼【ここから追加】有効期限を YYYYMM に正規化する関数 ▼▼▼
func normalizeExpiryDate(rawDate string) string {
	rawDate = strings.TrimSpace(rawDate)
	l := len(rawDate)

	switch {
	case l == 8: // YYYYMMDD (例: 20281231)
		return rawDate[0:6] // "202812"
	case l == 6: // YYMMDD (例: 280131)
		yy := rawDate[0:2]
		mm := rawDate[2:4]
		return "20" + yy + mm // "202801"
	case l == 4: // YYMM (例: 2812)
		yy := rawDate[0:2]
		mm := rawDate[2:4]
		return "20" + yy + mm // "202812"
	default:
		return rawDate // 不明な形式はそのまま返す
	}
}

// ▲▲▲【追加ここまで】▲▲▲

// ParseDat は、固定長のDATファイルからレコードを抽出し、DatRecord のスライスを返します。
func ParseDat(r io.Reader) ([]model.DatRecord, error) {
	// UTF-8へのデコーダーを削除
	scanner := bufio.NewScanner(r)

	var records []model.DatRecord
	var currentWholesale string

	for scanner.Scan() {
		lineBytes := scanner.Bytes() // UTF-8ではなく、元のバイト列を取得

		if len(lineBytes) == 0 {
			continue
		}

		// レコード区分 (1バイト目)
		recordType := ""
		if len(lineBytes) > 0 {
			recordType = string(lineBytes[0:1])
		}

		switch recordType {
		case "S":
			// Sレコードから卸コードを取得 (バイト位置 3 から 13 未満)
			currentWholesale = trimSpaceAndGetString(lineBytes, 3, 13)
		case "D":
			// Dレコード (最低 121 バイト必要と仮定)
			if len(lineBytes) < 121 {
				continue
			}

			// バイト位置に基づいてパース
			flag := parseInt(lineBytes, 3, 4)
			date := trimSpaceAndGetString(lineBytes, 4, 12)
			receiptNumber := trimSpaceAndGetString(lineBytes, 12, 22)
			lineNumber := trimSpaceAndGetString(lineBytes, 22, 24)
			janCode := trimSpaceAndGetString(lineBytes, 25, 38)

			// 製品名 (バイト位置 38-78) のみ Shift-JIS から UTF-8 に変換
			productName := sjisToUTF8(lineBytes[38:78])

			datQuantity := parseFloat(lineBytes, 78, 83)
			unitPrice := parseFloat(lineBytes, 83, 92)
			subtotal := parseFloat(lineBytes, 92,
				101)

			// 期限とロット (バイト位置)
			rawExpiryDate := trimSpaceAndGetString(lineBytes, 109, 115)
			lotNumber := trimSpaceAndGetString(lineBytes, 115, 121)

			// ▼▼▼【ここを修正】正規化関数を呼び出す ▼▼▼
			expiryDate := normalizeExpiryDate(rawExpiryDate)
			// ▲▲▲【修正ここまで】▲▲▲

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
				ExpiryDate:    expiryDate, // 正規化後の YYYYMM 形式
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

// ▲▲▲【修正ここまで】▲▲▲
