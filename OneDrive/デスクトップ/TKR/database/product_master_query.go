package database

import (
	"database/sql"
	"fmt"
	"strings"
	"tkr/barcode"
	"tkr/model"
)

type DBTX interface {
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
	NamedExec(query string, arg interface{}) (sql.Result, error)
}

func GetFilteredProductMasters(dbtx DBTX, usageClass, kanaName, genericName, shelfNumber string) ([]model.ProductMaster, error) {

	var masters []model.ProductMaster

	query := `SELECT * FROM product_master`
	mustConditions := []string{}
	args := []interface{}{}

	if usageClass != "" {
		mustConditions = append(mustConditions, "usage_classification = ?")
		args = append(args, usageClass)
	} else {
		return []model.ProductMaster{}, nil
	}

	if len(kanaName) > 0 {
		mustConditions = append(mustConditions, "kana_name LIKE ?")
		args = append(args, kanaName+"%")
	}

	if len(genericName) > 0 {
		mustConditions = append(mustConditions, "generic_name LIKE ?")
		args = append(args, "%"+genericName+"%")
	}

	if len(shelfNumber) > 0 {
		mustConditions = append(mustConditions, "shelf_number = ?")
		args = append(args, shelfNumber)
	}

	if len(mustConditions) > 0 {
		query += " WHERE " + strings.Join(mustConditions, " AND ")
	} else {
		return []model.ProductMaster{}, fmt.Errorf("usage class filter is required")
	}

	query += " ORDER BY kana_name"

	err := dbtx.Select(&masters, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return []model.ProductMaster{}, nil
		}
		return nil, fmt.Errorf("failed to select filtered product masters: %w", err)
	}

	if masters == nil {
		masters = []model.ProductMaster{}
	}

	return masters, nil
}

func GetProductMasterByCode(dbtx DBTX, code string) (*model.ProductMaster, error) {
	var master model.ProductMaster
	query := `SELECT * FROM product_master WHERE product_code = ?`
	err := dbtx.Get(&master, query, code)
	if err != nil {
		return nil, fmt.Errorf("failed to get product master by code %s: %w", code, err)
	}
	return &master, nil
}

func GetProductMasterByGs1Code(dbtx DBTX, gs1Code string) (*model.ProductMaster, error) {
	var master model.ProductMaster
	query := `SELECT * FROM product_master WHERE gs1_code = ?`
	err := dbtx.Get(&master, query, gs1Code)
	if err != nil {
		return nil, fmt.Errorf("failed to get product master by gs1_code %s: %w", gs1Code, err)
	}
	return &master, nil
}

func GetProductMasterByBarcode(dbtx DBTX, barcodeStr string) (*model.ProductMaster, error) {
	if barcodeStr == "" {
		return nil, fmt.Errorf("バーコードが空です")
	}

	if len(barcodeStr) <= 13 {
		return GetProductMasterByCode(dbtx, barcodeStr)
	}

	gs1Result, parseErr := barcode.Parse(barcodeStr)
	if parseErr != nil {
		return nil, fmt.Errorf("バーコード解析エラー: %w", parseErr)
	}

	gtin14 := gs1Result.Gtin14
	if gtin14 == "" {
		return nil, fmt.Errorf("バーコードからGTIN(14桁)が抽出できませんでした")
	}

	return GetProductMasterByGs1Code(dbtx, gtin14)
}

func GetProductMastersByYjCode(dbtx DBTX, yjCode string) ([]*model.ProductMaster, error) {
	var masters []*model.ProductMaster
	query := `SELECT * FROM product_master WHERE yj_code = ?
ORDER BY product_code`
	err := dbtx.Select(&masters, query, yjCode)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*model.ProductMaster{}, nil
		}
		return nil, fmt.Errorf("failed to select product masters by yj_code %s: %w", yjCode, err)
	}
	if masters == nil {
		masters = []*model.ProductMaster{}
	}
	return masters, nil
}

const insertProductMasterQuery = `
INSERT INTO product_master (
    product_code, yj_code, gs1_code, product_name, kana_name, kana_name_short, 
    generic_name, maker_name, specification, usage_classification, package_form, 
    yj_unit_name, yj_pack_unit_qty, jan_pack_inner_qty, jan_unit_code, jan_pack_unit_qty, 
    origin, nhi_price, purchase_price, flag_poison, flag_deleterious, flag_narcotic, 
    flag_psychotropic, flag_stimulant, flag_stimulant_raw, is_order_stopped, 
    supplier_wholesale, group_code, shelf_number, category, user_notes
) VALUES (
    :product_code, :yj_code, :gs1_code, :product_name, :kana_name, :kana_name_short, 
    :generic_name, :maker_name, :specification, :usage_classification, :package_form, 
    :yj_unit_name, :yj_pack_unit_qty, :jan_pack_inner_qty, :jan_unit_code, :jan_pack_unit_qty, 
    :origin, :nhi_price, :purchase_price, :flag_poison, :flag_deleterious, :flag_narcotic, 
    :flag_psychotropic, :flag_stimulant, :flag_stimulant_raw, :is_order_stopped, 
    :supplier_wholesale, :group_code, :shelf_number, :category, :user_notes
)`

func InsertProductMaster(dbtx DBTX, master *model.ProductMaster) error {
	_, err := dbtx.NamedExec(insertProductMasterQuery, master)
	if err != nil {
		return fmt.Errorf("failed to insert product master: %w", err)
	}
	return nil
}
