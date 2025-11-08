// C:\Users\wasab\OneDrive\デスクトップ\TKR\parsers\parser_utils.go
package parsers

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// ▼▼▼【修正】関数名を 'skipBOM' から 'SkipBOM' に変更 ▼▼▼
// SkipBOM はUTF-8 BOMをスキップします。
func SkipBOM(r io.Reader) io.Reader {
	// ▲▲▲【修正ここまで】▲▲▲
	br := bufio.NewReader(r)
	bom := []byte{0xEF, 0xBB, 0xBF}
	peeked, err := br.Peek(3)
	if err != nil {
		return br
	}
	isBOM := true
	for i, b := range bom {
		if peeked[i] != b {
			isBOM = false
			break
		}
	}
	if isBOM {
		br.Read(make([]byte, 3))
	}
	return br
}

// getColIndex はヘッダー名から列インデックスを取得するヘルパーです。
func getColIndex(header []string, required []string) (map[string]int, error) {
	colIndex := make(map[string]int)
	for i, colName := range header {
		colIndex[strings.TrimSpace(colName)] = i
	}
	for _, req := range required {
		if _, ok := colIndex[req]; !ok {
			return nil, fmt.Errorf("必須ヘッダーが見つかりません: %s", req)
		}
	}
	return colIndex, nil
}
