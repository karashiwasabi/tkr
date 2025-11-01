// C:\Users\wasab\OneDrive\デスクトップ\TKR\parsers\gs1_parser.go
package parsers

import (
	"fmt"
	"strings"
)

// GS1Result はGS1-128バーコードの解析結果を格納します
type GS1Result struct {
	JanCode    string // (01) GTIN (14桁)
	ExpiryDate string // (17) 有効期限 (YYMMDD)
	LotNumber  string // (10) ロット番号
}

// aiLengths は可変長AIの最大長を定義します (今回はロット(10)のみ)
var aiLengths = map[string]int{
	"10": 20, // ロット番号 (最大20桁)
}

// ParseGS1_128 はGS1-128形式の文字列を解析します
func ParseGS1_128(code string) (*GS1Result, error) {
	result := &GS1Result{}
	i := 0
	length := len(code)

	if length == 0 {
		return nil, fmt.Errorf("バーコードが空です")
	}

	for i < length {
		// (01) GTIN (14桁固定)
		if strings.HasPrefix(code[i:], "01") {
			if i+16 > length {
				return nil, fmt.Errorf("AI(01)のデータが不足しています")
			}
			result.JanCode = code[i+2 : i+16]
			i += 16
			continue
		}

		// (17) 有効期限 (6桁固定)
		if strings.HasPrefix(code[i:], "17") {
			if i+8 > length {
				return nil, fmt.Errorf("AI(17)のデータが不足しています")
			}
			result.ExpiryDate = code[i+2 : i+8] // YYMMDD
			i += 8
			continue
		}

		// (10) ロット番号 (可変長)
		if strings.HasPrefix(code[i:], "10") {
			if i+2 >= length {
				return nil, fmt.Errorf("AI(10)のデータが不足しています")
			}

			dataStart := i + 2
			// 次のAI開始位置、または文字列の終わりまでを探す
			dataEnd := dataStart
			maxLength := aiLengths["10"]
			count := 0

			for dataEnd < length && count < maxLength {
				// 次のAI開始識別子 (例: '01', '17', '10') があれば、そこが区切り
				if dataEnd+2 <= length {
					nextAI := code[dataEnd : dataEnd+2]
					if _, fixed := aiLengths[nextAI]; fixed { // 可変長AI
						break
					}
					if nextAI == "01" || nextAI == "17" { // 固定長AI
						break
					}
				}
				dataEnd++
				count++
			}

			result.LotNumber = code[dataStart:dataEnd]
			i = dataEnd
			continue
		}

		// 不明なAIまたはデータ部分
		// ここでは(01)(17)(10)以外は無視して進む
		i++
	}

	// GTIN(01)は必須とする (もしバーコードスキャンが(17)や(10)から始まる場合、このチェックは外す)
	if result.JanCode == "" {
		// (01)がなくても他の情報があれば良しとする場合
		if result.ExpiryDate != "" || result.LotNumber != "" {
			return result, nil
		}
		return nil, fmt.Errorf("バーコードから有効な情報(01, 17, 10)が見つかりませんでした")
	}

	return result, nil
}
