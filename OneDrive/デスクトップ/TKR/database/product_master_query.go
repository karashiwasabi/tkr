// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\product_master_query.go
package database

import (
	"database/sql"
	"fmt"
	"strings"
	"tkr/barcode"
	"tkr/model"
	"tkr/units"

	"github.com/jmoiron/sqlx"
)

type DBTX interface {
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
	NamedExec(query string, arg interface{}) (sql.Result, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row
	Rebind(query string) string
}

func GetAllProductMasters(dbtx DBTX) ([]*model.ProductMaster, error) {
	var masters []*model.ProductMaster
	query := `SELECT * FROM product_master`
	err := dbtx.Select(&masters, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return []*model.ProductMaster{},
				nil
		}
		return nil, fmt.Errorf("failed to select all product masters: %w", err)
	}
	if masters == nil {
		masters = []*model.ProductMaster{}
	}
	return masters, nil
}

// ▼▼▼【修正】[source]タグを文字列の外に移動 ▼▼▼
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

	var nameConditions []string
	if len(kanaName) > 0 {
		nameConditions = append(nameConditions, "kana_name LIKE ?")
		args = append(args, kanaName+"%")
	}

	if len(genericName) > 0 {
		nameConditions = append(nameConditions, "generic_name LIKE ?")
		args = append(args, "%"+genericName+"%")
	}

	if len(nameConditions) > 0 {
		mustConditions = append(mustConditions, "("+strings.Join(nameConditions, " OR ")+")")
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

// ▲▲▲【修正ここまで】▲▲▲

func GetProductMasterByCode(dbtx DBTX, code string) (*model.ProductMaster, error) {
	var master model.ProductMaster
	query := `SELECT * FROM product_master WHERE product_code = ?`
	err := dbtx.Get(&master, query, code)
	if err != nil {
		return nil, fmt.Errorf("failed to get product master by code %s: %w", code, err)
	}
	return &master, nil
}

// ▼▼▼【修正】 { をシグネチャと同じ行に移動 ▼▼▼
func GetProductMasterByGs1Code(dbtx DBTX, gs1Code string) (*model.ProductMaster, error) {
	var master model.ProductMaster
	query := `SELECT * FROM product_master WHERE gs1_code = ?`
	err := dbtx.Get(&master, query, gs1Code)
	if err != nil {
		return nil, fmt.Errorf("failed to get product master by gs1_code %s: %w", gs1Code, err)
	}
	return &master, nil
}

// ▲▲▲【修正ここまで】▲▲▲

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

// ▼▼▼【修正】[source]タグを文字列の外に移動 ▼▼▼
func GetProductMastersByYjCode(dbtx DBTX, yjCode string) ([]*model.ProductMaster, error) {
	var masters []*model.ProductMaster
	query := `SELECT * FROM product_master WHERE yj_code = ? ORDER BY product_code`
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

// ▲▲▲【修正ここまで】▲▲▲

func GetProductCodesByYjCodes(dbtx DBTX, yjCodes []string) ([]string, error) {
	if len(yjCodes) == 0 {
		return []string{}, nil
	}
	query, args, err := sqlx.In(`SELECT DISTINCT product_code FROM product_master WHERE yj_code IN (?)`, yjCodes)
	if err != nil {
		return nil, fmt.Errorf("failed to create IN query for GetProductCodesByYjCodes: %w", err)
	}
	query = dbtx.Rebind(query)
	var codes []string
	if err := dbtx.Select(&codes, query, args...); err != nil {
		return nil, err
	}
	return codes, nil
}

func GetProductMasterByKanaNameShort(dbtx DBTX, kanaNameShort string) (*model.ProductMaster, error) {
	var master model.ProductMaster
	query := `SELECT * FROM product_master WHERE kana_name_short = ?`
	err := dbtx.Get(&master, query, kanaNameShort)
	if err != nil {
		return nil, fmt.Errorf("failed to get product master by kana_name_short %s: %w", kanaNameShort, err)
	}
	return &master, nil
}

// ▼▼▼【修正】[source]タグを文字列の外に移動 ▼▼▼
const insertProductMasterQuery = `
INSERT INTO product_master (
    product_code, yj_code, gs1_code, product_name, kana_name, kana_name_short, 
    generic_name, maker_name, specification, usage_classification, 
 package_form, 
    yj_unit_name, yj_pack_unit_qty, jan_pack_inner_qty, jan_unit_code, jan_pack_unit_qty, 
    origin, nhi_price, purchase_price, flag_poison, flag_deleterious, flag_narcotic, 
    
 flag_psychotropic, flag_stimulant, flag_stimulant_raw, 
is_order_stopped, 
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

// ▲▲▲【修正ここまで】▲▲▲

// ▼▼▼【ここから追加】PackageKeyを全件取得するヘルパー関数 ▼▼▼

// MasterPackageKeyInfo は、マスターからPackageKeyを構築するための情報を保持します。
type MasterPackageKeyInfo struct {
	PackageKey     string
	YjCode         string
	Representative *model.ProductMaster
}

// GetAllPackageKeysFromMasters は、product_masterテーブルに存在するすべてのPackageKeyと
// その代表マスター情報をマップで返します。
func GetAllPackageKeysFromMasters(dbtx DBTX) (map[string]MasterPackageKeyInfo, error) {
	// 1. 全マスターを取得
	var allMasters []*model.ProductMaster
	query := `SELECT * FROM product_master WHERE yj_code != ''`
	err := dbtx.Select(&allMasters, query)
	if err != nil {
		return nil, fmt.Errorf("failed to select all product masters for package key generation: %w", err)
	}

	// 2. PackageKey ごとに分類し、代表マスター（JCSHMS優先）を選定
	mastersByPackageKey := make(map[string][]*model.ProductMaster)
	keyInfoMap := make(map[string]MasterPackageKeyInfo)

	for _, m := range allMasters {
		// PackageKeyを生成 (aggregation.go と同じロジック)
		key := fmt.Sprintf("%s|%s|%g|%s", m.YjCode, m.PackageForm, m.JanPackInnerQty, units.ResolveName(m.YjUnitName))
		mastersByPackageKey[key] = append(mastersByPackageKey[key], m)

		// 代表マスターを選定
		if info, ok := keyInfoMap[key]; !ok {
			// まだキーが登録されていなければ、このマスターを暫定代表とする
			keyInfoMap[key] = MasterPackageKeyInfo{
				PackageKey:     key,
				YjCode:         m.YjCode,
				Representative: m,
			}
		} else {
			// 既に代表がいる場合、JCSHMS由来のマスターを優先する
			if info.Representative.Origin != "JCSHMS" && m.Origin == "JCSHMS" {
				info.Representative = m
				keyInfoMap[key] = info
			}
		}
	}

	return keyInfoMap, nil
}

// ▲▲▲【追加ここまで】▲▲▲
