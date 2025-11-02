// C:\Users\wasab\OneDrive\デスクトップ\TKR\database\guided_inventory.go
package database

import (
	"fmt"
	"tkr/model"
	"tkr/units"

	"github.com/jmoiron/sqlx"
)

// DeleteTransactionsByFlagAndDateAndCodes は、指定されたフラグ、日付、製品コード群に一致する取引データを削除します。
// (WASABI: db/transaction_records.go  より移植)
func DeleteTransactionsByFlagAndDateAndCodes(tx *sqlx.Tx, flag int, date string, productCodes []string) error {
	if len(productCodes) == 0 {
		return nil
	}

	query, args, err := sqlx.In(`DELETE FROM transaction_records WHERE flag = ? AND transaction_date = ? AND jan_code IN (?)`, flag, date, productCodes)
	if err != nil {
		return fmt.Errorf("failed to create IN query for deleting transactions: %w", err)
	}
	query = tx.Rebind(query)

	_, err = tx.Exec(query, args...)
	if err != nil {
		return fmt.Errorf("failed to delete transactions by flag, date, and codes: %w", err)
	}
	return nil
}

// SaveGuidedInventoryData は、棚卸調整画面からの入力データを棚卸レコード(flag=0)として保存します。
// (WASABI: db/guided_inventory.go  より移植・TKR用に修正)
func SaveGuidedInventoryData(tx *sqlx.Tx, date string, yjCode string, allPackagings []*model.ProductMaster, inventoryData map[string]float64, deadstockData []model.DeadStockRecord) error {

	var allProductCodes []string
	mastersMap := make(map[string]*model.ProductMaster)
	for _, pkg := range allPackagings {
		allProductCodes = append(allProductCodes, pkg.ProductCode)
		mastersMap[pkg.ProductCode] = pkg
	}

	// 1. 同日の既存棚卸データ(flag=0)を削除
	if len(allProductCodes) > 0 {
		if err := DeleteTransactionsByFlagAndDateAndCodes(tx, 0, date, allProductCodes); err != nil {
			return fmt.Errorf("failed to delete old inventory records for the same day: %w", err)
		}
	}

	receiptNumber := fmt.Sprintf("ADJ-%s-%s", date, yjCode)
	var productCodesWithInventory []string

	// 2. 新しい棚卸データを挿入
	for i, productCode := range allProductCodes {
		master, ok := mastersMap[productCode]
		if !ok {
			continue
		}

		janQty := inventoryData[productCode]
		// 在庫が0より大きい（＝入力があった）コードを記録
		if janQty > 0 {
			productCodesWithInventory = append(productCodesWithInventory, productCode)
		}

		tr := model.TransactionRecord{
			TransactionDate: date,
			Flag:            0, // 棚卸
			ReceiptNumber:   receiptNumber,
			LineNumber:      fmt.Sprintf("%d", i+1),
			JanCode:         master.ProductCode,
			YjCode:          master.YjCode,
			ProductName:     master.ProductName, // ここは MapProductMasterToTransaction で上書きされる
			JanQuantity:     janQty,
			YjQuantity:      janQty * master.JanPackInnerQty,
			ProcessFlagMA:   "COMPLETE", // 棚卸データは常にCOMPLETE
		}

		if master.Origin == "JCSHMS" {
			tr.ProcessFlagMA = "COMPLETE"
		} else {
			tr.ProcessFlagMA = "PROVISIONAL"
		}

		// マスター情報をマッピング
		MapMasterToTransactionForInventory(&tr, master)

		if err := InsertTransactionRecord(tx, tr); err != nil {
			return fmt.Errorf("failed to insert inventory record for %s: %w", productCode, err)
		}
	}

	// 3. ロット・期限情報を更新 (入力があったもののみ)
	if len(productCodesWithInventory) > 0 {
		var relevantDeadstockData []model.DeadStockRecord
		for _, ds := range deadstockData {
			for _, pid := range productCodesWithInventory {
				if ds.ProductCode == pid {
					relevantDeadstockData = append(relevantDeadstockData, ds)
					break
				}
			}
		}

		// このYJコードに関連する既存のロット・期限情報を一度すべて削除
		if err := DeleteDeadStockByProductCodesInTx(tx, allProductCodes); err != nil {
			return fmt.Errorf("failed to delete old dead stock records: %w", err)
		}
		// 新しいロット・期限情報（在庫>0のもの）を保存
		if len(relevantDeadstockData) > 0 {
			if err := SaveDeadStockListInTx(tx, relevantDeadstockData); err != nil {
				return fmt.Errorf("failed to upsert new dead stock records: %w", err)
			}
		}
	}

	return nil
}

// MapMasterToTransactionForInventory は棚卸レコード(flag=0)用にマッピングを行います。
// dat/handler.go の MapDatToTransaction と usage/handler.go の MapUsageToTransaction を参考にします。
func MapMasterToTransactionForInventory(tr *model.TransactionRecord, master *model.ProductMaster) {
	// 1. 製品名と規格を連結
	productNameWithSpec := master.ProductName
	if master.Specification != "" {
		productNameWithSpec = master.ProductName + " " + master.Specification
	}
	tr.ProductName = productNameWithSpec

	// 2. 数量 (YjQuantity, JanQuantity) は呼び出し元で設定済み

	// 3. 包装仕様
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

	// 4.
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

	// 5. 単価と金額 (棚卸の場合は薬価を単価とし、金額を計算)
	tr.UnitPrice = master.NhiPrice
	tr.Subtotal = tr.YjQuantity * tr.UnitPrice
	tr.PurchasePrice = master.PurchasePrice         // 参考情報として仕入単価も記録
	tr.SupplierWholesale = master.SupplierWholesale // 参考情報

	// 6. フラグ
	tr.FlagPoison = master.FlagPoison
	tr.FlagDeleterious = master.FlagDeleterious
	tr.FlagNarcotic = master.FlagNarcotic
	tr.FlagPsychotropic = master.FlagPsychotropic
	tr.FlagStimulant = master.FlagStimulant
	tr.FlagStimulantRaw = master.FlagStimulantRaw
}
