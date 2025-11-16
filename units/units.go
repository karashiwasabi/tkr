// C:\Users\wasab\OneDrive\デスクトップ\TKR\units\units.go

package units

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"tkr/model" // ★ TKRのモデルを参照

	"golang.org/x/text/encoding/japanese"
	"golang.org/x/text/transform"
)

var internalMap map[string]string
var reverseMap map[string]string

// FormatPackageSpecは、JCSHMSのデータから仕様通りの包装文字列を生成します。
func FormatPackageSpec(jcshms *model.JcshmsInfo) string {
	if jcshms == nil {
		return ""
	}

	yjUnitName := ResolveName(jcshms.YjUnitName)
	pkg := fmt.Sprintf("%s %g%s", jcshms.PackageForm, jcshms.YjPackUnitQty, yjUnitName)

	if jcshms.JanPackInnerQty.Valid && jcshms.JanPackUnitQty.Valid && jcshms.JanPackUnitQty.Float64 != 0 {
		resolveJanUnitName := func(code string) string {
			if code != "0" && code != "" {
				return ResolveName(code)
			}
			return "" // 0か空の場合は単位を省略
		}

		janUnitName := resolveJanUnitName(jcshms.JanUnitCode.String)

		pkg += fmt.Sprintf(" (%g%s×%g%s)",
			jcshms.JanPackInnerQty.Float64,
			yjUnitName,
			jcshms.JanPackUnitQty.Float64,
			janUnitName,
		)
	}
	return pkg
}

// (TKRには FormatSimplePackageSpec はありません)

// LoadTANIFile は SOU/TANI.CSV を読み込み、単位コードと単位名のマップをメモリにロードします。
func LoadTANIFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("LoadTANIFile: open %s: %w", path, err)
	}
	defer file.Close()

	decoder := japanese.ShiftJIS.NewDecoder()
	reader := csv.NewReader(transform.NewReader(file, decoder))
	reader.LazyQuotes = true
	reader.FieldsPerRecord = -1

	m := make(map[string]string)
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("LoadTANIFile: read %s: %w", path, err)
		}
		if len(record) < 2 {
			continue
		}
		code := record[0]
		name := record[1]
		m[code] = name
	}
	internalMap = m

	reverseMap = make(map[string]string)
	for code, name := range internalMap {
		reverseMap[name] = code
	}

	return m, nil
}

// ResolveName は単位コードを単位名に変換します。
func ResolveName(code string) string {
	if internalMap == nil {
		return code
	}
	if name, ok := internalMap[code]; ok {
		return name
	}
	return code
}

// ResolveCode は単位名を単位コードに変換します。
func ResolveCode(name string) string {
	if reverseMap == nil {
		return ""
	}
	if code, ok := reverseMap[name]; ok {
		return code
	}
	return ""
}
