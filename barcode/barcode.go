// C:\Users\wasab\OneDrive\デスクトップ\TKR\barcode\barcode.go
package barcode

import (
	"fmt"
	"strings"
)

// Result はバーコードの解析結果を格納します
type Result struct {
	Gtin14     string // (01) GTIN (14桁)
	ExpiryDate string // (17) 有効期限 (YYMMDD)
	LotNumber  string // (10) ロット番号
}

// aiLengths は可変長AIの最大長を定義します (ロット(10)のみ)
var aiLengths = map[string]int{
	"10": 20, // ロット番号 (最大20桁)
}

// Parse はスキャンされた文字列を自動判別して解析します。
func Parse(code string) (*Result, error) {
	length := len(code)

	if length == 0 {
		return nil, fmt.Errorf("バーコードが空です")
	}

	// 仕様 1: 15桁以上は AI付き文字列
	if length >= 15 {
		// 01から始まるかチェック
		if strings.HasPrefix(code, "01") {
			return parseAIString(code)
		}
		// 01から始まらない15桁以上は不正とみなす
		return nil, fmt.Errorf("15桁以上ですが、AI(01)で始まっていません")
	}

	// 仕様 2: 14桁は GTIN-14
	if length == 14 {
		return &Result{
			Gtin14: code,
		}, nil
	}

	// 仕様 3: 13桁は JAN-13
	if length == 13 {
		return &Result{
			Gtin14: "0" + code, // 先頭に0を加えて14桁にする
		}, nil
	}

	// 仕様 4: 13桁未満は JAN-8 など
	if length < 13 {
		return &Result{
			Gtin14: fmt.Sprintf("%014s", code), // 先頭から0で埋めて14桁にする
		}, nil
	}

	// (ここに来ることはないはずだが、念のため)
	return nil, fmt.Errorf("不明なバーコード形式です")
}

// parseAIString は 15桁以上のAI付き文字列を解析する内部関数です。
// (旧 parsers/gs1_parser.go のロジック)
func parseAIString(code string) (*Result, error) {
	result := &Result{}
	i := 0
	length := len(code)

	for i < length {
		// (01) GTIN (14桁固定)
		if strings.HasPrefix(code[i:], "01") {
			if i+16 > length { // AI(2) + Data(14)
				return nil, fmt.Errorf("AI(01)のデータが不足しています")
			}
			result.Gtin14 = code[i+2 : i+16]
			i += 16
			continue
		}

		// (17) 有効期限 (6桁固定)
		if strings.HasPrefix(code[i:], "17") {
			if i+8 > length { // AI(2) + Data(6)
				return nil, fmt.Errorf("AI(17)のデータが不足しています")
			}

			// ▼▼▼【ここから修正】YYMMDD を YYYYMM 形式に正規化 ▼▼▼
			expiryYYMMDD := code[i+2 : i+8] // YYMMDD (例: "280100")
			if len(expiryYYMMDD) == 6 {
				yy := expiryYYMMDD[0:2] // "28"
				mm := expiryYYMMDD[2:4] // "01"
				// DB保存形式 (YYYYMM) に変換
				result.ExpiryDate = "20" + yy + mm // "202801"
			} else {
				result.ExpiryDate = expiryYYMMDD // 予期せぬ形式の場合はそのまま
			}
			// ▲▲▲【修正ここまで】▲▲▲

			i += 8
			continue
		}

		// (10) ロット番号 (可変長)
		if strings.HasPrefix(code[i:], "10") {
			if i+2 > length { // AI(2)
				return nil, fmt.Errorf("AI(10)のデータが不足しています")
			}

			dataStart := i + 2
			dataEnd := dataStart
			maxLength := aiLengths["10"]

			for dataEnd < length {
				if (dataEnd - dataStart) >= maxLength { // 最大長
					break
				}

				remaining := code[dataEnd:]

				// 次のAIが「完全な形」で存在する場合のみ区切る
				if len(remaining) >= 2 {
					nextAI := remaining[:2]

					// 次が AI(01) か？ (01 + 14桁データ)
					if nextAI == "01" {
						if len(remaining) >= 16 {
							break
						}
					}
					// 次が AI(17) か？ (17 + 6桁データ)
					if nextAI == "17" {
						if len(remaining) >= 8 {
							break
						}
					}
				}
				dataEnd++
			}

			result.LotNumber = code[dataStart:dataEnd]
			i = dataEnd
			continue
		}

		// 不明なAIまたはデータ部分
		i++
	}

	if result.Gtin14 == "" {
		return nil, fmt.Errorf("バーコードからAI(01)GTINが見つかりませんでした")
	}

	return result, nil
}
