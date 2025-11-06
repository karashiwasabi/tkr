// C:\Users\wasab\OneDrive\デスクトップ\TKR\mastermanager\manager.go
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
var yjCodeRegex = regexp.MustCompile(`^[0-9A-Z]{11,12}$`)

// JANコードの判定は13桁の数字
var janCodeRegex = regexp.MustCompile(`^[0-9]{13}$`)

// ▼▼▼【ここに追加】GS1コード（14桁）の正規表現 ▼▼▼
var gs1CodeRegex = regexp.MustCompile(`^[0-9]{14}$`)

// ▲▲▲【追加ここまで】▲▲▲

// ▼▼▼【ここに追加】MA2J形式の正規表現 ▼▼▼
var ma2jCodeRegex = regexp.MustCompile(`^MA2J[0-9]{9}$`)

// ▲▲▲【追加ここまで】▲▲▲

// FindOrCreateMaster は、指定されたキー(YJコードまたはJANコード)に対応する製品マスターを探し、
// なければ JCSHMS または仮マスターとして作成する共通関数です。
// ▼▼▼【ここから修正】FindOrCreateMaster 全体を上書き ▼▼▼
func FindOrCreateMaster(tx *sqlx.Tx, productCodeOrKey string, productName string) (*model.ProductMaster, error) {

	var existingMaster model.ProductMaster
	var err error

	isJANKey := janCodeRegex.MatchString(productCodeOrKey)
	isYJKey := yjCodeRegex.MatchString(productCodeOrKey)
	// ▼▼▼【ここに追加】GS1形式のチェック ▼▼▼
	isGS1Key := gs1CodeRegex.MatchString(productCodeOrKey)
	// ▲▲▲【追加ここまで】▲▲▲
	isMA2JKey := ma2jCodeRegex.MatchString(productCodeOrKey)

	// 1. DBで product_master を検索
	if isYJKey {
		// YJコードなら yj_code カラムで検索
		query := "SELECT * FROM product_master WHERE yj_code = ?"
		err = tx.Get(&existingMaster, query, productCodeOrKey)
		log.Printf("Searching master by YJ Code: %s", productCodeOrKey)
		// ▼▼▼【ここから修正】GS1Key も検索対象に追加 ▼▼▼
	} else if isJANKey || isMA2JKey {
		// JANまたはMA2Jなら product_code カラムで検索
		query := "SELECT * FROM product_master WHERE product_code = ?"
		err = tx.Get(&existingMaster, query, productCodeOrKey)
		log.Printf("Searching master by JAN/MA2J Code (Product Code): %s", productCodeOrKey)
	} else if isGS1Key {
		// GS1なら gs1_code カラムで検索
		query := "SELECT * FROM product_master WHERE gs1_code = ?"
		err = tx.Get(&existingMaster, query, productCodeOrKey)
		log.Printf("Searching master by GS1 Code (gs1_code): %s", productCodeOrKey)
		// ▲▲▲【修正ここまで】▲▲▲
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

	// 2. JCSHMS から作成を試みる
	// ▼▼▼【ここから修正】isJANKey または isGS1Key の場合に JCSHMS を検索 ▼▼▼
	if (isJANKey || isGS1Key) && !strings.HasPrefix(productCodeOrKey, "999") && productCodeOrKey != "0000000000000" && productCodeOrKey != "" {

		var jcshmsInfo *model.JcshmsInfo
		var jcshmsErr error

		if isJANKey {
			// JANキーの場合
			log.Printf("Key %s is JAN format. Searching JCSHMS by JAN...", productCodeOrKey)
			jcshmsInfo, jcshmsErr = database.GetJcshmsInfoByJan(tx, productCodeOrKey)
		} else {
			// GS1キーの場合
			log.Printf("Key %s is GS1 format. Searching JCSHMS by GS1...", productCodeOrKey)
			jcshmsInfo, jcshmsErr = database.GetJcshmsInfoByGs1Code(tx, productCodeOrKey)
		}
		// ▲▲▲【修正ここまで】▲▲▲

		if jcshmsErr != nil && jcshmsErr != sql.ErrNoRows {
			return nil, fmt.Errorf("failed to query jcshms/jancode for key %s: %w", productCodeOrKey, jcshmsErr)
		}

		if jcshmsInfo != nil {
			log.Printf("Found info in JCSHMS for key: %s. Creating master...", productCodeOrKey)
			input := JcshmsToProductMasterInput(jcshmsInfo)

			// ▼▼▼【ここから追加】キーがGS1の場合、GS1コードをinputにセットする ▼▼▼
			if isGS1Key && input.Gs1Code == "" {
				input.Gs1Code = productCodeOrKey
			}
			// ▲▲▲【追加ここまで】▲▲▲

			// JCSHMS由来でもYJコードがない場合、MA2Yで採番
			if input.YjCode == "" {
				newYj, seqErr := database.NextSequenceInTx(tx, "MA2Y", "MA2Y", 8)
				if seqErr != nil {
					return nil, fmt.Errorf("failed to get next MA2Y sequence for JCSHMS master (Key: %s): %w", productCodeOrKey, seqErr)
				}
				input.YjCode = newYj
				log.Printf("Generated new YJ code %s for Key %s from JCSHMS (original was empty)", newYj, productCodeOrKey)
			}

			newMaster, upsertErr := UpsertProductMasterSqlx(tx, input)
			if upsertErr != nil {
				return nil, fmt.Errorf("failed to upsert master from JCSHMS for Key %s: %w", productCodeOrKey, upsertErr)
			}
			log.Printf("Successfully created master from JCSHMS for Key: %s (YJ: %s)", productCodeOrKey, newMaster.YjCode)
			return newMaster, nil
		}
	}

	// 3. JCSHMS にもない場合、またはJAN/GS1でない場合は仮マスターを作成
	log.Printf("Info not found in JCSHMS for key: %s (or it was not a valid JAN/GS1). Creating provisional master...", productCodeOrKey)

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

	// ▼▼▼【ここから修正】仮 ProductCode の生成ロジック変更 (GS1形式も除外) ▼▼▼
	provisionalProductCode := productCodeOrKey
	// JAN形式でもGS1形式でもない、または 0x13、または "" の場合は MA2J を採番する
	if (!isJANKey && !isGS1Key) || productCodeOrKey == "0000000000000" || productCodeOrKey == "" {
		// MA2Jシーケンスから新しい仮コードを採番 (13桁: MA2J + 9桁)
		newPJCode, seqErr := database.NextSequenceInTx(tx, "MA2J", "MA2J", 9)
		if seqErr != nil {
			return nil, fmt.Errorf("failed to get next MA2J sequence for provisional master (Key: %s): %w", productCodeOrKey, seqErr)
		}
		provisionalProductCode = newPJCode
		log.Printf("Original key was not JAN/GS1 or was 0x13 or empty, using synthetic Product Code: %s", provisionalProductCode)
	}
	// ▲▲▲【修正ここまで】▲▲▲

	provisionalInput := model.ProductMasterInput{
		ProductCode:         provisionalProductCode,
		YjCode:              provisionalYjCode,
		ProductName:         productName,
		Origin:              "PROVISIONAL",
		UsageClassification: "他",
	}

	// ▼▼▼【ここから追加】キーがGS1の場合、GS1コードも仮マスターにセットする ▼▼▼
	if isGS1Key {
		provisionalInput.Gs1Code = productCodeOrKey
	}
	// ▲▲▲【追加ここまで】▲▲▲

	// ▼▼▼【ここから修正】DATの0埋JAN(0x13)または空JAN("")の場合、KanaNameShortにも製品名をセット ▼▼▼
	if productCodeOrKey == "0000000000000" || productCodeOrKey == "" {
		provisionalInput.KanaNameShort = productName
		log.Printf("Setting KanaNameShort for DAT auto-provisional master (key: '%s'): %s", productCodeOrKey, productName)
	}
	// ▲▲▲【修正ここまで】▲▲▲

	newMaster, upsertErr := UpsertProductMasterSqlx(tx, provisionalInput)
	if upsertErr != nil {
		return nil, fmt.Errorf("failed to upsert provisional master (OrigKey: %s): %w", productCodeOrKey, upsertErr)
	}
	log.Printf("Successfully created provisional master (OrigKey: %s, ProductCode: %s, YJ: %s)", productCodeOrKey, newMaster.ProductCode, newMaster.YjCode)
	return newMaster, nil
}

// ▲▲▲【修正ここまで】FindOrCreateMaster

// JcshmsToProductMasterInput は JCSHMS の情報を model.ProductMasterInput に変換します。
func JcshmsToProductMasterInput(jcshms *model.JcshmsInfo) model.ProductMasterInput {
	var unitNhiPrice float64
	// WASABI: mappers/jcshms_to_master.go のロジック
	if jcshms.NhiPriceFactor > 0 {
		unitNhiPrice = jcshms.NhiPrice * jcshms.NhiPriceFactor
	} else if jcshms.YjPackUnitQty > 0 {
		// TKR: jcshms.PackageNhiPrice (JC050) / jcshms.YjPackUnitQty (JC044)
		unitNhiPrice = jcshms.PackageNhiPrice / jcshms.YjPackUnitQty
	} else {
		// TKR: jcshms.NhiPrice (JC049)
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
		Specification:       strings.TrimSpace(jcshms.Specification),
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

// UpsertProductMasterSqlx は product_master テーブルにデータを挿入または更新します。
func UpsertProductMasterSqlx(tx *sqlx.Tx, input model.ProductMasterInput) (*model.ProductMaster, error) {
	// ▼▼▼【ここを修正】 ':flag_deleteri'ous' -> ':flag_deleterious' ▼▼▼
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
			flag_psychotropic=excluded.flag_psychotropic, 
flag_stimulant=excluded.flag_stimulant, flag_stimulant_raw=excluded.flag_stimulant_raw,
			is_order_stopped=excluded.is_order_stopped, supplier_wholesale=excluded.supplier_wholesale,
			group_code=excluded.group_code, shelf_number=excluded.shelf_number, category=excluded.category, user_notes=excluded.user_notes
	`
	// ▲▲▲【修正ここまで】▲▲▲

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

// ▼▼▼【ここから追加】model.ProductMaster を model.ProductMasterInput に変換するヘルパー ▼▼▼
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

// ▲▲▲【追加ここまで】▲▲▲
