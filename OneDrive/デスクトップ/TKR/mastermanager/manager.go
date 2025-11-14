// C:\Users\wasab\OneDrive\デスクトップ\TKR\mastermanager\manager.go
package mastermanager

import (
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"tkr/database"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

var yjCodeRegex = regexp.MustCompile(`^[0-9A-Z]{11,12}$`)
var janCodeRegex = regexp.MustCompile(`^[0-9]{13}$`)
var gs1CodeRegex = regexp.MustCompile(`^[0-9]{14}$`)
var ma2jCodeRegex = regexp.MustCompile(`^MA2J[0-9]{9}$`)

func FindOrCreateMaster(tx *sqlx.Tx, productCodeOrKey string, productName string) (*model.ProductMaster, error) {

	if productCodeOrKey == "0000000000000" ||
		productCodeOrKey == "" {
		if productName != "" {
			var existingMaster model.ProductMaster
			err := tx.Get(&existingMaster, "SELECT * FROM product_master WHERE kana_name_short = ?", productName)

			if err == nil {
				return &existingMaster, nil
			}
			if err != sql.ErrNoRows {
				return nil, fmt.Errorf("failed to query product_master by kana_name_short for %s: %w", productName, err)
			}
		}
	}

	var existingMaster model.ProductMaster
	var err error

	isJANKey := janCodeRegex.MatchString(productCodeOrKey)
	isYJKey := yjCodeRegex.MatchString(productCodeOrKey)
	isGS1Key := gs1CodeRegex.MatchString(productCodeOrKey)
	isMA2JKey := ma2jCodeRegex.MatchString(productCodeOrKey)

	if isYJKey {
		query := "SELECT * FROM product_master WHERE yj_code = ?"
		err = tx.Get(&existingMaster, query, productCodeOrKey)
	} else if isJANKey || isMA2JKey {
		query := "SELECT * FROM product_master WHERE product_code = ?"
		err = tx.Get(&existingMaster, query, productCodeOrKey)
	} else if isGS1Key {
		query := "SELECT * FROM product_master WHERE gs1_code = ?"
		err = tx.Get(&existingMaster, query, productCodeOrKey)
	} else {
		query := "SELECT * FROM product_master WHERE product_code = ?"
		err = tx.Get(&existingMaster, query, productCodeOrKey)
	}

	if err == nil {
		return &existingMaster, nil
	}
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query product_master for key %s: %w", productCodeOrKey, err)
	}

	if (isJANKey || isGS1Key) && !strings.HasPrefix(productCodeOrKey, "999") && productCodeOrKey != "0000000000000" && productCodeOrKey != "" {

		var jcshmsInfo *model.JcshmsInfo
		var jcshmsErr error

		if isJANKey {
			jcshmsInfo, jcshmsErr = database.GetJcshmsInfoByJan(tx, productCodeOrKey)
		} else {
			jcshmsInfo, jcshmsErr = database.GetJcshmsInfoByGs1Code(tx, productCodeOrKey)
		}

		if jcshmsErr != nil && jcshmsErr != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to query jcshms/jancode for key %s: %w", productCodeOrKey, jcshmsErr)
		}

		if jcshmsInfo != nil {
			input := JcshmsToProductMasterInput(jcshmsInfo)

			if isGS1Key && input.Gs1Code == "" {
				input.Gs1Code = productCodeOrKey
			}

			if input.YjCode == "" {
				newYj, seqErr := database.NextSequenceInTx(tx, "MA2Y", "MA2Y", 8)
				if seqErr != nil {
					return nil, fmt.Errorf("failed to get next MA2Y sequence for JCSHMS master (Key: %s): %w", productCodeOrKey, seqErr)
				}
				input.YjCode = newYj
			}

			newMaster, upsertErr := UpsertProductMasterSqlx(tx, input)
			if upsertErr != nil {
				return nil, fmt.Errorf("failed to upsert master from JCSHMS for Key %s: %w", productCodeOrKey, upsertErr)
			}
			return newMaster, nil
		}
	}

	provisionalYjCode := productCodeOrKey
	if !isYJKey {
		newYj, seqErr := database.NextSequenceInTx(tx, "MA2Y", "MA2Y", 8)
		if seqErr != nil {
			return nil, fmt.Errorf("failed to get next MA2Y sequence for provisional master (Key: %s): %w", productCodeOrKey, seqErr)
		}
		provisionalYjCode = newYj
	}

	provisionalProductCode := productCodeOrKey
	if (!isJANKey && !isGS1Key) ||
		productCodeOrKey == "0000000000000" || productCodeOrKey == "" {
		newPJCode, seqErr := database.NextSequenceInTx(tx, "MA2J", "MA2J", 9)
		if seqErr != nil {
			return nil, fmt.Errorf("failed to get next MA2J sequence for provisional master (Key: %s): %w", productCodeOrKey, seqErr)
		}
		provisionalProductCode = newPJCode
	}

	dbProductName := productName
	if !strings.HasPrefix(productName, "◆") {
		dbProductName = "◆" + productName
	}

	provisionalInput := model.ProductMasterInput{
		ProductCode:         provisionalProductCode,
		YjCode:              provisionalYjCode,
		ProductName:         dbProductName,
		Origin:              "PROVISIONAL",
		UsageClassification: "他",
	}

	if isGS1Key {
		provisionalInput.Gs1Code = productCodeOrKey
	}

	if productCodeOrKey == "0000000000000" || productCodeOrKey == "" {
		provisionalInput.KanaNameShort = productName
	}

	newMaster, upsertErr := UpsertProductMasterSqlx(tx, provisionalInput)
	if upsertErr != nil {
		return nil, fmt.Errorf("failed to upsert provisional master (OrigKey: %s): %w", productCodeOrKey, upsertErr)
	}
	return newMaster, nil
}

func JcshmsToProductMasterInput(jcshms *model.JcshmsInfo) model.ProductMasterInput {
	var unitNhiPrice float64
	if jcshms.NhiPriceFactor > 0 {
		unitNhiPrice = jcshms.NhiPrice * jcshms.NhiPriceFactor
	} else if jcshms.YjPackUnitQty > 0 {
		unitNhiPrice = jcshms.PackageNhiPrice / jcshms.YjPackUnitQty
	} else {
		unitNhiPrice = jcshms.NhiPrice
	}
	janUnitCodeInt, _ := strconv.Atoi(jcshms.JanUnitCode.String)

	// ▼▼▼【ここから修正】剤型区分のロジックをご要望に合わせて変更 ▼▼▼
	var usageClass string
	if strings.TrimSpace(jcshms.YjCode) == "" {
		// YJコード(JC009)が空白の場合は、JC013の値に関わらず強制的に「他」を設定
		usageClass = "他"
	} else {
		// YJコードが存在する場合
		usageClass = strings.TrimSpace(jcshms.UsageClassification) // JC013の値を取得
		if usageClass == "" {
			// YJコードは存在するが、JC013が空白だった場合も「他」を設定
			usageClass = "他"
		}
	}
	// ▲▲▲【修正ここまで】▲▲▲

	return model.ProductMasterInput{
		ProductCode: jcshms.ProductCode,
		YjCode:      jcshms.YjCode,
		Gs1Code:     jcshms.Gs1Code,
		ProductName: strings.TrimSpace(jcshms.ProductName),
		KanaName:    strings.TrimSpace(jcshms.KanaName),

		KanaNameShort: strings.TrimSpace(jcshms.KanaNameShort),
		GenericName:   strings.TrimSpace(jcshms.GenericName),

		MakerName:           strings.TrimSpace(jcshms.MakerName),
		Specification:       strings.TrimSpace(jcshms.Specification),
		UsageClassification: usageClass, // ▼▼▼【修正】変更後の変数を設定
		PackageForm:         strings.TrimSpace(jcshms.PackageForm),
		YjUnitName:          strings.TrimSpace(jcshms.YjUnitName),
		YjPackUnitQty:       jcshms.YjPackUnitQty,
		JanPackInnerQty:     jcshms.JanPackInnerQty.Float64,
		JanUnitCode:         janUnitCodeInt,
		JanPackUnitQty:      jcshms.JanPackUnitQty.Float64,
		Origin:              "JCSHMS",
		NhiPrice:            unitNhiPrice,
		PurchasePrice:       0,
		FlagPoison:          jcshms.FlagPoison,
		FlagDeleterious:     jcshms.FlagDeleterious,
		FlagNarcotic:        jcshms.FlagNarcotic,
		FlagPsychotropic:    jcshms.FlagPsychotropic,
		FlagStimulant:       jcshms.FlagStimulant,
		FlagStimulantRaw:    jcshms.FlagStimulantRaw,
		IsOrderStopped:      0,
		SupplierWholesale:   "",
		GroupCode:           "",
		ShelfNumber:         "",
		Category:            "",
		UserNotes:           "",
	}
}

func UpsertProductMasterSqlx(tx *sqlx.Tx, input model.ProductMasterInput) (*model.ProductMaster, error) {
	query := `
		INSERT INTO product_master (
			product_code, yj_code, gs1_code, product_name, kana_name, kana_name_short, generic_name,
			maker_name, 
specification, usage_classification, package_form, yj_unit_name, yj_pack_unit_qty,
			jan_pack_inner_qty, jan_unit_code, jan_pack_unit_qty, origin,
			nhi_price, purchase_price,
			flag_poison, flag_deleterious, flag_narcotic, flag_psychotropic, flag_stimulant, 
flag_stimulant_raw,
			is_order_stopped, supplier_wholesale,
			group_code, shelf_number, category, user_notes
		) VALUES (
			:product_code, :yj_code, :gs1_code, :product_name, :kana_name, :kana_name_short, :generic_name,
			:maker_name, :specification, :usage_classification, :package_form, :yj_unit_name, :yj_pack_unit_qty,
			:jan_pack_inner_qty, :jan_unit_code, :jan_pack_unit_qty, :origin,
			:nhi_price, :purchase_price,
			:flag_poison, :flag_deleterious, :flag_narcotic, :flag_psychotropic, :flag_stimulant, :flag_stimulant_raw,
			:is_order_stopped, :supplier_wholesale,
			:group_code, :shelf_number, :category, :user_notes
		)
		ON CONFLICT(product_code) DO UPDATE SET
			yj_code=excluded.yj_code, gs1_code=excluded.gs1_code, product_name=excluded.product_name, kana_name=excluded.kana_name,
			kana_name_short=excluded.kana_name_short, generic_name=excluded.generic_name,
			maker_name=excluded.maker_name, specification=excluded.specification, usage_classification=excluded.usage_classification,
			package_form=excluded.package_form, yj_unit_name=excluded.yj_unit_name, yj_pack_unit_qty=excluded.yj_pack_unit_qty,
			jan_pack_inner_qty=excluded.jan_pack_inner_qty, jan_unit_code=excluded.jan_unit_code, jan_pack_unit_qty=excluded.jan_pack_unit_qty,
			origin=excluded.origin, nhi_price=excluded.nhi_price, purchase_price=excluded.purchase_price,
			flag_poison=excluded.flag_poison, flag_deleterious=excluded.flag_deleterious, flag_narcotic=excluded.flag_narcotic,
			flag_psychotropic=excluded.flag_psychotropic, flag_stimulant=excluded.flag_stimulant, flag_stimulant_raw=excluded.flag_stimulant_raw,
			is_order_stopped=excluded.is_order_stopped, supplier_wholesale=excluded.supplier_wholesale,
			group_code=excluded.group_code, shelf_number=excluded.shelf_number, category=excluded.category, user_notes=excluded.user_notes
	`

	_, err := tx.NamedExec(query, input)
	if err != nil {
		return nil, fmt.Errorf("NamedExec for upsert failed: %w", err)
	}

	var insertedMaster model.ProductMaster
	err = tx.Get(&insertedMaster, "SELECT * FROM product_master WHERE product_code = ?", input.ProductCode)

	if err != nil {
		return nil, fmt.Errorf("failed to re-fetch master after upsert for %s: %w", input.ProductCode, err)
	}

	return &insertedMaster,
		nil
}

func MasterToInput(m *model.ProductMaster) model.ProductMasterInput {
	return model.ProductMasterInput{
		ProductCode:         m.ProductCode,
		YjCode:              m.YjCode,
		Gs1Code:             m.Gs1Code,
		ProductName:         m.ProductName,
		KanaName:            m.KanaName,
		KanaNameShort:       m.KanaNameShort,
		GenericName:         m.GenericName,
		MakerName:           m.MakerName,
		Specification:       m.Specification,
		UsageClassification: m.UsageClassification,
		PackageForm:         m.PackageForm,
		YjUnitName:          m.YjUnitName,
		YjPackUnitQty:       m.YjPackUnitQty,
		JanPackInnerQty:     m.JanPackInnerQty,
		JanUnitCode:         m.JanUnitCode,
		JanPackUnitQty:      m.JanPackUnitQty,
		Origin:              m.Origin,
		NhiPrice:            m.NhiPrice,
		PurchasePrice:       m.PurchasePrice,
		FlagPoison:          m.FlagPoison,
		FlagDeleterious:     m.FlagDeleterious,
		FlagNarcotic:        m.FlagNarcotic,
		FlagPsychotropic:    m.FlagPsychotropic,
		FlagStimulant:       m.FlagStimulant,
		FlagStimulantRaw:    m.FlagStimulantRaw,
		IsOrderStopped:      m.IsOrderStopped,
		SupplierWholesale:   m.SupplierWholesale,
		GroupCode:           m.GroupCode,
		ShelfNumber:         m.ShelfNumber,
		Category:            m.Category,
		UserNotes:           m.UserNotes,
	}
}
