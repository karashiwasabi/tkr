// C:\Users\wasab\OneDrive\デスクトップ\TKR\mappers\view.go
package mappers

import (
	"database/sql"
	"fmt"
	"tkr/model"
	"tkr/units"
)

// ToProductMasterView は、*model.ProductMaster を画面表示用の model.ProductMasterView に変換します。
// (WASABI: mappers/view.go  より移植)
func ToProductMasterView(master *model.ProductMaster) model.ProductMasterView {
	if master == nil {
		return model.ProductMasterView{}
	}

	// TKRの JcshmsInfo 構造体（units.FormatPackageSpec が要求する型）に合わせる
	tempJcshmsInfo := model.JcshmsInfo{
		PackageForm:     master.PackageForm,
		YjUnitName:      master.YjUnitName,
		YjPackUnitQty:   master.YjPackUnitQty,
		JanPackInnerQty: sql.NullFloat64{Float64: master.JanPackInnerQty, Valid: true},
		JanPackUnitQty:  sql.NullFloat64{Float64: master.JanPackUnitQty, Valid: true},
		JanUnitCode:     sql.NullString{String: fmt.Sprintf("%d", master.JanUnitCode), Valid: true},
	}

	// JAN単位名を解決する
	var janUnitName string
	if master.JanUnitCode == 0 {
		janUnitName = master.YjUnitName
	} else {
		janUnitName = units.ResolveName(fmt.Sprintf("%d", master.JanUnitCode))
	}

	return model.ProductMasterView{
		ProductMaster:        *master,
		FormattedPackageSpec: units.FormatPackageSpec(&tempJcshmsInfo),
		JanUnitName:          janUnitName,
	}
}

// StockLedgerYJGroupView は StockLedgerYJGroup の画面表示用です。
// (WASABI: guidedinventory/handler.go  より)
type StockLedgerYJGroupView struct {
	model.StockLedgerYJGroup
	PackageLedgers []StockLedgerPackageGroupView `json:"packageLedgers"`
}

// StockLedgerPackageGroupView は StockLedgerPackageGroup の画面表示用です。
// (WASABI: guidedinventory/handler.go  より)
type StockLedgerPackageGroupView struct {
	model.StockLedgerPackageGroup
	Masters []model.ProductMasterView `json:"masters"`
}

// ResponseDataView は棚卸調整画面の全データを保持するコンテナです。
// (WASABI: guidedinventory/handler.go  より)
type ResponseDataView struct {
	TransactionLedger []StockLedgerYJGroupView `json:"transactionLedger"`
	YesterdaysStock   *StockLedgerYJGroupView  `json:"yesterdaysStock"`
	DeadStockDetails  []model.DeadStockRecord  `json:"deadStockDetails"`

	// ▼▼▼【ここに追加】(WASABI: guidedinventory/handler.go [cite: 542] より) ▼▼▼
	PrecompDetails []model.TransactionRecord `json:"precompDetails"`
	// ▲▲▲【追加ここまで】▲▲▲

}

// ConvertToView はDBモデルを集計・表示用モデルに変換します。
// (WASABI: guidedinventory/handler.go  より移植)
func ConvertToView(yjGroups []model.StockLedgerYJGroup) []StockLedgerYJGroupView {
	if yjGroups == nil {
		return nil
	}

	viewGroups := make([]StockLedgerYJGroupView, 0, len(yjGroups))

	for _, group := range yjGroups {
		newYjGroup := StockLedgerYJGroupView{
			StockLedgerYJGroup: group,
			PackageLedgers:     make([]StockLedgerPackageGroupView, 0, len(group.PackageLedgers)),
		}

		for _, pkg := range group.PackageLedgers {
			newPkgGroup := StockLedgerPackageGroupView{
				StockLedgerPackageGroup: pkg,
				Masters:                 make([]model.ProductMasterView, 0, len(pkg.Masters)),
			}

			for _, master := range pkg.Masters {
				newMasterView := ToProductMasterView(master)
				newPkgGroup.Masters = append(newPkgGroup.Masters, newMasterView)
			}
			newYjGroup.PackageLedgers = append(newYjGroup.PackageLedgers, newPkgGroup)
		}
		viewGroups = append(viewGroups, newYjGroup)
	}
	return viewGroups
}
