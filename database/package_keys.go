// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\package_keys.go
package database

import (
	"fmt"
	"tkr/model"
	"tkr/units"
)

// GeneratePackageKey はプロダクトマスタから一意のパッケージキーを生成します。
// システム全体でこの関数を使用することで、キーの生成ルールを統一します。
func GeneratePackageKey(m *model.ProductMaster) string {
	// YjCode | PackageForm | JanPackInnerQty | YjUnitName(Resolved)
	return fmt.Sprintf("%s|%s|%g|%s",
		m.YjCode,
		m.PackageForm,
		m.JanPackInnerQty,
		units.ResolveName(m.YjUnitName),
	)
}
