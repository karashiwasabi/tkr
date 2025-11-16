// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\jcshms_query.go
package database

import (
	"database/sql"
	"fmt"
	"strings"

	// sqlx は Get で必要なので残す
	"tkr/model"
)

// ▼▼▼ 重複していた DBTX インターフェース定義を削除 ▼▼▼
/*
type DBTX interface {
	Get(dest interface{}, query string, args ...interface{}) error
	Select(dest interface{}, query string, args ...interface{}) error
}
*/
// ▲▲▲ 削除 ▲▲▲

// GetJcshmsInfoByJan は JAN コードをキーに jcshms と jancode テーブルを結合して検索します。
// 引数の dbtx DBTX は product_master_query.go で定義されたインターフェース型を使う
func GetJcshmsInfoByJan(dbtx DBTX, janCode string) (*model.JcshmsInfo, error) {
	var info model.JcshmsInfo
	// ▼▼▼【ここを修正】 j.JC020 をSELECTに追加 ▼▼▼
	// ▼▼▼【修正】[source]タグを文字列の外に移動 ▼▼▼
	query := `
		SELECT
			j.JC000, j.JC009, j.JC018, j.JC019, j.JC020, j.JC022, j.JC024, j.JC030, j.JC013, j.JC037, j.JC039,
			j.JC044, j.JC049, j.JC050, j.JC122, j.JC124,
			j.JC061, j.JC062, j.JC063, j.JC064, j.JC065, j.JC066,
			ja.JA006, ja.JA007, ja.JA008
		FROM jcshms AS j
		LEFT JOIN jancode AS ja ON j.JC000 = ja.JA001
		WHERE j.JC000 = ?`
	// ▲▲▲【修正ここまで】▲▲▲

	err := dbtx.Get(&info, query, janCode) // product_master_query.go で定義された DBTX.Get を使う

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("GetJcshmsInfoByJan failed for jan %s: %w", janCode, err)
	}

	return &info, nil
}

// ▼▼▼【修正】[source]タグを文字列の外に移動 ▼▼▼
func GetJcshmsInfoByGs1Code(dbtx DBTX, gs1Code string) (*model.JcshmsInfo, error) {
	var info model.JcshmsInfo
	query := `
		SELECT
			j.JC000, j.JC009, j.JC018, j.JC019, j.JC020, j.JC022, j.JC024, j.JC030, j.JC013, j.JC037, j.JC039,
			j.JC044, j.JC049, j.JC050, j.JC122, j.JC124,
			j.JC061, j.JC062, j.JC063, j.JC064, j.JC065, j.JC066,
			ja.JA006, ja.JA007, ja.JA008
		FROM jcshms AS j
		LEFT JOIN jancode AS ja ON j.JC000 = ja.JA001
		WHERE j.JC122 = ?`

	err := dbtx.Get(&info, query, gs1Code)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("GetJcshmsInfoByGs1Code failed for gs1_code %s: %w", gs1Code, err)
	}

	return &info, nil
}

// ▲▲▲【修正ここまで】▲▲▲

// ▼▼▼【修正】[source]タグを文字列の外に移動 ▼▼▼
func GetFilteredJcshmsInfo(dbtx DBTX, usageClass, kanaName, genericName string) ([]*model.JcshmsInfo, error) {
	var jcshmsList []*model.JcshmsInfo
	query := `
        SELECT
            j.JC000, j.JC009, j.JC018, j.JC019, j.JC020, j.JC022, j.JC024, j.JC030, j.JC013, j.JC037, j.JC039,
            j.JC044, j.JC049, j.JC050, j.JC122, j.JC124,
            j.JC061, j.JC062, j.JC063, j.JC064, j.JC065, j.JC066,
  
           ja.JA006, ja.JA007, ja.JA008
        FROM jcshms AS j
		LEFT JOIN jancode AS ja ON j.JC000 = ja.JA001`

	var conditions []string
	var args []interface{}

	if usageClass != "" {
		conditions = append(conditions, "j.JC013 = ?")
		args = append(args, usageClass)
	}
	if kanaName != "" {
		conditions = append(conditions, "j.JC022 LIKE ?")
		args = append(args, kanaName+"%")
	}
	if genericName != "" {
		conditions = append(conditions, "j.JC024 LIKE ?")
		args = append(args, "%"+genericName+"%")
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY j.JC022 LIMIT 500" // Add a reasonable limit

	err := dbtx.Select(&jcshmsList, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to select filtered jcshms info: %w", err)
	}

	return jcshmsList, nil
}

// ▲▲▲【修正ここまで】▲▲▲
