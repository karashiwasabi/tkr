// C:\Users\wasab\OneDrive\デスクトップ\TKR\mastermanager/manager.go
package mastermanager

import (
	"database/sql"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"tkr/database"
	"tkr/model"

	"github.com/jmoiron/sqlx"
)

// YJコードの判定 (WASABIのロジックでは主にYJコードをキーとして使用)
// ▼▼▼【ここから修正】YJコードは11桁または12桁の英数字を含む形式に修正 ▼▼▼
var yjCodeRegex = regexp.MustCompile(`^[0-9A-Z]{11,12}$`)

// ▲▲▲【修正ここまで】▲▲▲
// JANコードの判定は13桁の数字
var janCodeRegex = regexp.MustCompile(`^[0-9]{13}$`)

// FindOrCreateMaster は、指定されたキー(YJコードまたはJANコード)に対応する製品マスターを探し、
// なければ JCSHMS または仮マスターとして作成する共通関数です。
func FindOrCreateMaster(tx *sqlx.Tx, productCodeOrKey string, productName string) (*model.ProductMaster, error) {

	var existingMaster model.ProductMaster
	var err error

	isJANKey := janCodeRegex.MatchString(productCodeOrKey)
	isYJKey := yjCodeRegex.MatchString(productCodeOrKey)

	// 1. DBで product_master を検索 (WASABIのロジック: YJコードを優先的に検索)
	if isYJKey {
		// YJコードなら yj_code カラムで検索
		query := "SELECT * FROM product_master WHERE yj_code = ?"
		err = tx.Get(&existingMaster, query, productCodeOrKey)
		log.Printf("Searching master by YJ Code: %s", productCodeOrKey)
	} else if isJANKey {
		// YJでヒットせず、JANなら product_code カラムで検索
		query := "SELECT * FROM product_master WHERE product_code = ?"
		err = tx.Get(&existingMaster, query, productCodeOrKey)
		log.Printf("Searching master by JAN Code (Product Code): %s", productCodeOrKey)
	} else {
		// どちらでもない場合（合成キーなど）は product_code で検索
		query := "SELECT * FROM product_master WHERE product_code = ?"
		err = tx.Get(&existingMaster, query, productCodeOrKey)
		log.Printf("Searching master by Generic Key (Product Code): %s", productCodeOrKey)
	}

	// 見つかった場合
	if err == nil {
		log.Printf("Found existing master in DB (ProductCode: %s, YJ: %s)", existingMaster.ProductCode, existingMaster.YjCode)
		return &existingMaster, nil
	}
	// DBエラー (見つからない以外)
	if err != sql.ErrNoRows {
		return nil, fmt.Errorf("failed to query product_master for key %s: %w", productCodeOrKey, err)
	}

	// --- DB には見つからなかった場合の処理 (作成ロジック) ---
	log.Printf("Product master not found in DB for key: %s. Attempting to create...", productCodeOrKey)

	// 2. JCSHMS から作成を試みる (JANコードの場合のみ)
	if isJANKey && !strings.HasPrefix(productCodeOrKey, "999") {
		jcshmsInfo, jcshmsErr := database.GetJcshmsInfoByJan(tx, productCodeOrKey)
		if jcshmsErr != nil && jcshmsErr != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to query jcshms/jancode for JAN %s: %w", productCodeOrKey, jcshmsErr)
		}

		if jcshmsInfo != nil {
			log.Printf("Found info in JCSHMS for JAN: %s. Creating master...", productCodeOrKey)
			input := JcshmsToProductMasterInput(jcshmsInfo)

			// JCSHMS由来でもYJコードがない場合、MA2Yで採番
			if input.YjCode == "" {
				newYj, seqErr := database.NextSequenceInTx(tx, "MA2Y", "MA2Y", 8)
				if seqErr != nil {
					return nil, fmt.Errorf("failed to get next MA2Y sequence for JCSHMS master (JAN: %s): %w", productCodeOrKey, seqErr)
				}
				input.YjCode = newYj
				log.Printf("Generated new YJ code %s for JAN %s from JCSHMS (original was empty)", newYj, productCodeOrKey)
			}

			newMaster, upsertErr := UpsertProductMasterSqlx(tx, input)
			if upsertErr != nil {
				return nil, fmt.Errorf("failed to upsert master from JCSHMS for JAN %s: %w", productCodeOrKey, upsertErr)
			}
			log.Printf("Successfully created master from JCSHMS for JAN: %s (YJ: %s)", productCodeOrKey, newMaster.YjCode)
			return newMaster, nil
		}
	}

	// 3. JCSHMS にもない場合、またはJANでない場合は仮マスターを作成
	log.Printf("Info not found in JCSHMS for key: %s (or it was not a valid JAN). Creating provisional master...", productCodeOrKey)

	// 仮マスターの YJコードを決定
	provisionalYjCode := productCodeOrKey
	if !isYJKey {
		// YJコード形式でない場合は採番する
		newYj, seqErr := database.NextSequenceInTx(tx, "MA2Y", "MA2Y", 8)
		if seqErr != nil {
			return nil, fmt.Errorf("failed to get next MA2Y sequence for provisional master (Key: %s): %w", productCodeOrKey, seqErr)
		}
		provisionalYjCode = newYj
	}

	// 仮マスターの ProductCode (JAN) を決定
	provisionalProductCode := productCodeOrKey
	if !isJANKey {
		// JANコードでない場合は合成キーを使用
		provisionalProductCode = fmt.Sprintf("9999999999999%s", provisionalYjCode)
		log.Printf("Original key was not JAN, using synthetic Product Code: %s", provisionalProductCode)
	}

	provisionalInput := model.ProductMasterInput{
		ProductCode:         provisionalProductCode,
		YjCode:              provisionalYjCode,
		ProductName:         productName,
		Origin:              "PROVISIONAL",
		UsageClassification: "他",
	}

	newMaster, upsertErr := UpsertProductMasterSqlx(tx, provisionalInput)
	if upsertErr != nil {
		return nil, fmt.Errorf("failed to upsert provisional master (OrigKey: %s): %w", productCodeOrKey, upsertErr)
	}
	log.Printf("Successfully created provisional master (OrigKey: %s, ProductCode: %s, YJ: %s)", productCodeOrKey, newMaster.ProductCode, newMaster.YjCode)
	return newMaster, nil
}

// JcshmsToProductMasterInput (変更なし)
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

	return model.ProductMasterInput{
		ProductCode: jcshms.ProductCode,
		YjCode:      jcshms.YjCode,
		Gs1Code:     jcshms.Gs1Code,
		ProductName: strings.TrimSpace(jcshms.ProductName),
		KanaName:    strings.TrimSpace(jcshms.KanaName),

		KanaNameShort: strings.TrimSpace(jcshms.KanaNameShort),
		GenericName:   strings.TrimSpace(jcshms.GenericName),

		MakerName:           strings.TrimSpace(jcshms.MakerName),
		Specification:       "",
		UsageClassification: strings.TrimSpace(jcshms.UsageClassification),
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

// UpsertProductMasterSqlx (変更なし)
func UpsertProductMasterSqlx(tx *sqlx.Tx, input model.ProductMasterInput) (*model.ProductMaster, error) {
	query := `
		INSERT INTO product_master (
			product_code, yj_code, gs1_code, product_name, kana_name, kana_name_short, generic_name,
			maker_name, specification, usage_classification, package_form, yj_unit_name, yj_pack_unit_qty,
			jan_pack_inner_qty, jan_unit_code, jan_pack_unit_qty, origin,
			nhi_price, purchase_price,
			flag_poison, flag_deleterious, flag_narcotic, flag_psychotropic, flag_stimulant, flag_stimulant_raw,
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
		log.Printf("ERROR: Upsert successful but failed to re-fetch master for %s: %v", input.ProductCode, err)
		return nil, fmt.Errorf("failed to re-fetch master after upsert for %s: %w", input.ProductCode, err)
	}

	return &insertedMaster, nil
}
