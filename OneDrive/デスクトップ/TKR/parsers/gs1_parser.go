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
			if i+16 > length { // AI(2) + Data(14)
				return nil, fmt.Errorf("AI(01)のデータが不足しています")
			}
			result.JanCode = code[i+2 : i+16]
			i += 16
			continue
		}

		// (17) 有効期限 (6桁固定)
		if strings.HasPrefix(code[i:], "17") {
			if i+8 > length { // AI(2) + Data(6)
				return nil, fmt.Errorf("AI(17)のデータが不足しています")
			}
			result.ExpiryDate = code[i+2 : i+8] // YYMMDD
			i += 8
			continue
		}

		// (10) ロット番号 (可変長)
		if strings.HasPrefix(code[i:], "10") {
			if i+2 > length { // AI(2) + 少なくとも1桁のデータ
				return nil, fmt.Errorf("AI(10)のデータが不足しています")
			}

			dataStart := i + 2
			dataEnd := dataStart
			maxLength := aiLengths["10"]

			for dataEnd < length {
				if (dataEnd - dataStart) >= maxLength { // 最大長に達したら終了
					break
				}

				remaining := code[dataEnd:]

				// ▼▼▼【修正】次のAIが「完全な形」で存在する場合のみ区切る ▼▼▼
				if len(remaining) >= 2 {
					nextAI := remaining[:2]

					// 次が AI(01) か？ (01 + 14桁データ が必要)
					if nextAI == "01" {
						if len(remaining) >= 16 { // AI(2) + Data(14) = 16
							break // データが十分あるので、AI(10)はここまで
						}
					}

					// 次が AI(17) か？ (17 + 6桁データ が必要)
					if nextAI == "17" {
						if len(remaining) >= 8 { // AI(2) + Data(6) = 8
							break // データが十分あるので、AI(10)はここまで
						}
					}
					// (注: もし AI(10) が続く場合もFNC1が必要だが、FNC1省略運用ではAI(10)の連続は解析不能)
				}
				// ▲▲▲【修正ここまで】▲▲▲

				dataEnd++
			}

			result.LotNumber = code[dataStart:dataEnd]
			i = dataEnd
			continue
		}

		// 不明なAIまたはデータ部分
		i++
	}

	// (01)GTINが見つからなかった場合
	if result.JanCode == "" {
		// (17)や(10)だけでも情報があれば良しとする
		if result.ExpiryDate != "" || result.LotNumber != "" {
			return result, nil
		}
		return nil, fmt.Errorf("バーコードから有効な情報(01, 17, 10)が見つかりませんでした")
	}

	return result, nil
}
