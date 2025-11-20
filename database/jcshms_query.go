// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\jcshms_query.go
package database

import (
	"database/sql"
	"fmt"
	"strings"

	"tkr/model"
)

// GetJcshmsInfoByJan は JAN コードをキーに jcshms と jancode テーブルを結合して検索します。
func GetJcshmsInfoByJan(dbtx DBTX, janCode string) (*model.JcshmsInfo, error) {
	var info model.JcshmsInfo
	query := `
		SELECT
			j.JC000, j.JC009, j.JC018, j.JC019, j.JC020, j.JC022, j.JC024, j.JC030, j.JC013, j.JC037, j.JC039,
			j.JC044, j.JC049, j.JC050, j.JC122, j.JC124,
			j.JC061, j.JC062, j.JC063, j.JC064, j.JC065, j.JC066,
			ja.JA006, ja.JA007, ja.JA008
		FROM jcshms AS j
		LEFT JOIN jancode AS ja ON j.JC000 = ja.JA001
		WHERE j.JC000 = ?`

	err := dbtx.Get(&info, query, janCode)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("GetJcshmsInfoByJan failed for jan %s: %w", janCode, err)
	}

	return &info, nil
}

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

func GetFilteredJcshmsInfo(dbtx DBTX, usageClass, kanaName, genericName, productName string, drugTypes []string) ([]*model.JcshmsInfo, error) {
	var jcshmsList []*model.JcshmsInfo
	query := `
        SELECT
            j.JC000, j.JC009, j.JC018, j.JC019, j.JC020, j.JC022, j.JC024, j.JC030, j.JC013, j.JC037, j.JC039,
            j.JC044, j.JC049, j.JC050, j.JC122, j.JC124,
            j.JC061, j.JC062, j.JC063, j.JC064, j.JC065, j.JC066,
            ja.JA006, ja.JA007, ja.JA008
        FROM jcshms AS j
		LEFT JOIN jancode AS ja ON j.JC000 = ja.JA001
		WHERE 1=1 `

	var args []interface{}

	if usageClass != "" && usageClass != "all" {
		query += " AND j.JC013 = ?"
		args = append(args, usageClass)
	}
	if kanaName != "" {
		query += " AND j.JC022 LIKE ?"
		args = append(args, kanaName+"%")
	}
	if genericName != "" {
		query += " AND j.JC024 LIKE ?"
		args = append(args, "%"+genericName+"%")
	}
	if productName != "" {
		query += " AND j.JC018 LIKE ?"
		args = append(args, "%"+productName+"%")
	}

	if len(drugTypes) > 0 {
		var drugConditions []string
		for _, dt := range drugTypes {
			switch dt {
			case "poison":
				drugConditions = append(drugConditions, "j.JC061 = 1")
			case "deleterious":
				drugConditions = append(drugConditions, "j.JC062 = 1")
			case "narcotic":
				drugConditions = append(drugConditions, "j.JC063 = 1")
			case "psycho1":
				drugConditions = append(drugConditions, "j.JC064 = 1")
			case "psycho2":
				drugConditions = append(drugConditions, "j.JC064 = 2")
			case "psycho3":
				drugConditions = append(drugConditions, "j.JC064 = 3")
			case "stimulant":
				drugConditions = append(drugConditions, "j.JC065 = 1")
			case "stimulant_raw":
				drugConditions = append(drugConditions, "j.JC066 = 1")
			}
		}
		if len(drugConditions) > 0 {
			query += " AND (" + strings.Join(drugConditions, " OR ") + ")"
		}
	}

	query += " ORDER BY j.JC022 LIMIT 500"

	err := dbtx.Select(&jcshmsList, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to select filtered jcshms info: %w", err)
	}

	return jcshmsList, nil
}
