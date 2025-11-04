// C:\Users\wasab\OneDrive\デスクトップ\TKR\mappers\mappers.go
package mappers

import (
	"fmt"
	"tkr/model"
	"tkr/units"
)

/**
 * MapMasterToTransaction は、ProductMaster の情報を TransactionRecord にマッピングします。
 *
 * 呼び出し元（dat, usage, inventory）は、事前に tr のユニークな値
 * (Date, Flag, YjQuantity, JanQuantity, UnitPrice, Subtotal など) を
 * 設定しておく必要があります。
 *
 * この関数は、マスター由来の共通情報（製品名、包装仕様、MAフラグなど）を設定します。
 */
func MapMasterToTransaction(tr *model.TransactionRecord, master *model.ProductMaster) {
	// 1. 製品名と規格を連結
	productNameWithSpec := master.ProductName
	if master.Specification != "" {
		productNameWithSpec = master.ProductName + " " + master.Specification
	}
	tr.ProductName = productNameWithSpec

	// 2. 包装仕様
	yjUnitName := units.ResolveName(master.YjUnitName)
	packageSpec := fmt.Sprintf("%s %g%s", master.PackageForm, master.YjPackUnitQty, yjUnitName)
	janUnitCodeStr := fmt.Sprintf("%d", master.JanUnitCode)
	var janUnitName string
	if master.JanUnitCode == 0 {
		janUnitName = yjUnitName
	} else {
		janUnitName = units.ResolveName(janUnitCodeStr)
	}
	if master.JanPackInnerQty > 0 && master.JanPackUnitQty > 0 {
		packageSpec += fmt.Sprintf(" (%g%s×%g%s)",
			master.JanPackInnerQty, yjUnitName, master.JanPackUnitQty, janUnitName)
	}
	tr.PackageSpec = packageSpec

	// 3. MAフラグ (共通ロジック)
	if master.Origin == "JCSHMS" {
		tr.ProcessFlagMA = "COM"
	} else {
		tr.ProcessFlagMA = "PRO"
	}

	// 4. その他のマスター由来フィールド
	tr.JanCode = master.ProductCode
	tr.YjCode = master.YjCode
	tr.KanaName = master.KanaName
	tr.UsageClassification = master.UsageClassification
	tr.PackageForm = master.PackageForm
	tr.MakerName = master.MakerName
	tr.JanPackInnerQty = master.JanPackInnerQty
	tr.JanPackUnitQty = master.JanPackUnitQty
	tr.JanUnitName = janUnitName
	tr.JanUnitCode = janUnitCodeStr
	tr.YjPackUnitQty = master.YjPackUnitQty
	tr.YjUnitName = yjUnitName

	// 5. 参考価格と卸 (UnitPrice と Subtotal は呼び出し元で設定)
	tr.PurchasePrice = master.PurchasePrice
	tr.SupplierWholesale = master.SupplierWholesale

	// 6. 医薬品フラグ
	tr.FlagPoison = master.FlagPoison
	tr.FlagDeleterious = master.FlagDeleterious
	tr.FlagNarcotic = master.FlagNarcotic
	tr.FlagPsychotropic = master.FlagPsychotropic
	tr.FlagStimulant = master.FlagStimulant
	tr.FlagStimulantRaw = master.FlagStimulantRaw
}

func FromJcshmsToProductMaster(jcshms *model.JcshmsInfo) *model.ProductMaster {
	janPackInnerQty := 0.0
	if jcshms.JanPackInnerQty.Valid {
		janPackInnerQty = jcshms.JanPackInnerQty.Float64
	}
	janPackUnitQty := 0.0
	if jcshms.JanPackUnitQty.Valid {
		janPackUnitQty = jcshms.JanPackUnitQty.Float64
	}
	janUnitCode := 0
	if jcshms.JanUnitCode.Valid {
		// TKRでは `units` を使って文字列から数値に変換する必要があるかもしれないが、
		// JCSHMSの時点では文字列なので、ここでは単純な変換は難しい。
		// 一旦0をセットしておく。
	}

	return &model.ProductMaster{
		ProductCode:         jcshms.ProductCode,
		YjCode:              jcshms.YjCode,
		Gs1Code:             jcshms.Gs1Code,
		ProductName:         jcshms.ProductName,
		KanaName:            jcshms.KanaName,
		KanaNameShort:       jcshms.KanaNameShort,
		GenericName:         jcshms.GenericName,
		MakerName:           jcshms.MakerName,
		Specification:       jcshms.Specification,
		UsageClassification: jcshms.UsageClassification,
		PackageForm:         jcshms.PackageForm,
		YjUnitName:          jcshms.YjUnitName,
		YjPackUnitQty:       jcshms.YjPackUnitQty,
		JanPackInnerQty:     janPackInnerQty,
		JanUnitCode:         janUnitCode,
		JanPackUnitQty:      janPackUnitQty,
		Origin:              "JCSHMS", // オリジンをJCSHMSに設定
		NhiPrice:            jcshms.NhiPrice,
		PurchasePrice:       jcshms.NhiPrice, // 初期値として薬価を設定
		FlagPoison:          jcshms.FlagPoison,
		FlagDeleterious:     jcshms.FlagDeleterious,
		FlagNarcotic:        jcshms.FlagNarcotic,
		FlagPsychotropic:    jcshms.FlagPsychotropic,
		FlagStimulant:       jcshms.FlagStimulant,
		FlagStimulantRaw:    jcshms.FlagStimulantRaw,
	}
}
