// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\product_master_query.go
package database

import (
	"database/sql"
	"fmt"
	"strings"
	"tkr/model"
)

// DBTX インターフェース (変更なし)
type DBTX interface {
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
	NamedExec(query string, arg interface{}) (sql.Result, error)
}

// GetFilteredProductMasters は指定されたフィルター条件で product_master を検索します。
func GetFilteredProductMasters(dbtx DBTX, usageClass, productName, kanaName, genericName, shelfNumber string) ([]model.ProductMaster, error) {

	var masters []model.ProductMaster

	query := `SELECT * FROM product_master`
	mustConditions := []string{}
	args := []interface{}{}

	// 内外注区分 (必須条件として扱う)
	if usageClass != "" {
		mustConditions = append(mustConditions, "usage_classification = ?")
		args = append(args, usageClass)
	} else {
		return []model.ProductMaster{}, nil
	}

	// 製品名は 0文字より大きい場合、条件に追加 (部分一致)
	if len(productName) > 0 {
		mustConditions = append(mustConditions, "product_name LIKE ?")
		args = append(args, "%"+productName+"%")
	}

	// カナ名は 0文字より大きい場合、条件に追加 (前方一致)
	if len(kanaName) > 0 {
		mustConditions = append(mustConditions, "kana_name LIKE ?")
		args = append(args, kanaName+"%") // 前方一致
	}

	// 一般名は 0文字より大きい場合、条件に追加 (部分一致)
	if len(genericName) > 0 {
		mustConditions = append(mustConditions, "generic_name LIKE ?")
		args = append(args, "%"+genericName+"%")
	}

	// 棚番は 0文字より大きい場合、条件に追加 (完全一致)
	if len(shelfNumber) > 0 {
		mustConditions = append(mustConditions, "shelf_number = ?") // 完全一致
		args = append(args, shelfNumber)
	}

	// 最終 WHERE 句の結合
	if len(mustConditions) > 0 {
		query += " WHERE " + strings.Join(mustConditions, " AND ")
	} else {
		return []model.ProductMaster{}, fmt.Errorf("usage class filter is required")
	}

	query += " ORDER BY kana_name" // カナ名順

	err := dbtx.Select(&masters, query, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return []model.ProductMaster{}, nil // 見つからなくてもエラーではない
		}
		return nil, fmt.Errorf("failed to select filtered product masters: %w", err)
	}

	if masters == nil {
		masters = []model.ProductMaster{}
	}

	return masters, nil
}

// GetProductMasterByCode は product_code をキーにレコードを取得します。(変更なし)
func GetProductMasterByCode(dbtx DBTX, code string) (*model.ProductMaster, error) {
	var master model.ProductMaster
	query := `SELECT * FROM product_master WHERE product_code = ?`
	err := dbtx.Get(&master, query, code)
	if err != nil {
		// sql.ErrNoRows も含め、エラーとして返す
		return nil, fmt.Errorf("failed to get product master by code %s: %w", code, err)
	}
	return &master, nil
}

// GetProductMasterByGs1Code はGS1コードで product_master を検索する関数 (変更なし)
func GetProductMasterByGs1Code(dbtx DBTX, gs1Code string) (*model.ProductMaster, error) {
	var master model.ProductMaster
	query := `SELECT * FROM product_master WHERE gs1_code = ?`
	err := dbtx.Get(&master, query, gs1Code)
	if err != nil {
		// sql.ErrNoRows も含め、エラーとして返す
		return nil, fmt.Errorf("failed to get product master by gs1_code %s: %w", gs1Code, err)
	}
	return &master, nil
}

// ▼▼▼【ここから追加】YJコードで product_master を検索する関数 ▼▼▼
// (WASABI: db/product_master.go  より)
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

// ▲▲▲【追加ここまで】▲▲▲
