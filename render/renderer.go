// C:\Users\wasab\OneDrive\デスクトップ\TKR\render\renderer.go
package render

import (
	"fmt"
	"strconv"
	"strings"
	"tkr/model"
	"tkr/units" // TKRのunitsパッケージをインポート
)

// RenderTransactionTableHTML は、トランザクションレコードのスライスからHTMLテーブル文字列を生成します。
// この関数は dat と usage の両方から呼び出される共有コンポーネントです。
func RenderTransactionTableHTML(transactions []model.TransactionRecord, wholesalerMap map[string]string) string {
	var sb strings.Builder

	sb.WriteString(`
    <thead>
     
   <tr>
            <th rowspan="2" class="col-action">－</th>
            <th class="col-date">日付</th>
            <th class="col-yj">YJ</th>
            <th colspan="2" class="col-product">製品名</th>
            <th class="col-count">個数</th>
            <th class="col-yjqty">YJ数量</th>
            <th class="col-yjpackqty">YJ包装数</th>
  
   
        <th class="col-yjunit">YJ単位</th>
            <th class="col-unitprice">単価</th>
            <th class="col-expiry">期限</th>
            <th class="col-wholesaler">卸</th>
            <th class="col-line">行</th>
        </tr>
        <tr>
            <th class="col-flag">種別</th>
        
   
  <th class="col-jan">JAN</th>
            <th class="col-package">包装</th>
            <th class="col-maker">メーカー</th>
            <th class="col-form">剤型</th>
            <th class="col-janqty">JAN数量</th>
            <th class="col-janpackqty">JAN包装数</th>
            <th class="col-janunit">JAN単位</th>
            <th class="col-amount">金額</th>
    
   
      <th class="col-lot">ロット</th>
            <th class="col-receipt">伝票番号</th>
            <th class="col-ma">MA</th>
        </tr>
    </thead>`)

	sb.WriteString(`<tbody>`)
	if len(transactions) == 0 {
		sb.WriteString(`<tr><td colspan="13">登録されたデータはありません。</td></tr>`)
	} else {
		for _, tx := range transactions {
			formattedDate := tx.TransactionDate
			formattedExpiry := tx.ExpiryDate
			formattedYjQty := strconv.FormatFloat(tx.YjQuantity, 'f', 2, 64)
			formattedUnitPrice := strconv.FormatFloat(tx.UnitPrice, 'f', 2, 64)
			formattedSubtotal := strconv.FormatFloat(tx.Subtotal, 'f', 2, 64)

			var flagText string
			switch tx.Flag {
			case 1:
				flagText = "納品"
			case 2:
				flagText = "返品"
			case 3:
				flagText = "処方"
			default:
				flagText = strconv.Itoa(tx.Flag)
			}

			// 卸名をマップから取得
			var clientOrWholesalerName string
			if tx.Flag == 1 || tx.Flag == 2 { // 納品・返品
				if wholesalerMap != nil {
					if name, ok := wholesalerMap[tx.ClientCode]; ok {
						clientOrWholesalerName = name
					} else {
						clientOrWholesalerName = tx.ClientCode // マップにない場合はコード
					}
				} else {
					clientOrWholesalerName = tx.ClientCode // マップ自体がない
				}
			} else {
				clientOrWholesalerName = tx.ClientCode // 処方などは ClientCode をそのまま（TKRでは現状空のはず）
			}

			sb.WriteString(`<tr>`)
			sb.WriteString(`<td class="center col-action">－</td>`)
			sb.WriteString(fmt.Sprintf(`<td class="center col-date">%s</td>`, formattedDate))
			sb.WriteString(fmt.Sprintf(`<td class="col-yj">%s</td>`, tx.YjCode))
			sb.WriteString(fmt.Sprintf(`<td colspan="2" class="col-product">%s</td>`, tx.ProductName))
			sb.WriteString(fmt.Sprintf(`<td class="right col-count">%.0f</td>`, tx.DatQuantity))
			sb.WriteString(fmt.Sprintf(`<td class="right col-yjqty">%s</td>`, formattedYjQty))
			sb.WriteString(fmt.Sprintf(`<td class="right col-yjpackqty">%.0f</td>`, tx.YjPackUnitQty))
			// ▼▼▼【修正】units.ResolveName を呼び出す ▼▼▼
			sb.WriteString(fmt.Sprintf(`<td class="center col-yjunit">%s</td>`, units.ResolveName(tx.YjUnitName)))
			// ▲▲▲【修正ここまで】▲▲▲
			sb.WriteString(fmt.Sprintf(`<td class="right col-unitprice">%s</td>`, formattedUnitPrice))
			sb.WriteString(fmt.Sprintf(`<td class="center col-expiry">%s</td>`, formattedExpiry))
			sb.WriteString(fmt.Sprintf(`<td class="center col-wholesaler">%s</td>`, clientOrWholesalerName))
			sb.WriteString(fmt.Sprintf(`<td class="center col-line">%s</td>`, tx.LineNumber))
			sb.WriteString(`</tr>`)

			sb.WriteString(`<tr>`)
			sb.WriteString(`<td></td>`)
			sb.WriteString(fmt.Sprintf(`<td class="center col-flag">%s</td>`, flagText))
			sb.WriteString(fmt.Sprintf(`<td class="col-jan">%s</td>`, tx.JanCode))
			sb.WriteString(fmt.Sprintf(`<td class="col-package">%s</td>`, tx.PackageSpec))
			sb.WriteString(fmt.Sprintf(`<td class="col-maker">%s</td>`, tx.MakerName))
			sb.WriteString(`<td class="col-form"></td>`)
			sb.WriteString(fmt.Sprintf(`<td class="right col-janqty">%.2f</td>`, tx.JanQuantity))
			sb.WriteString(fmt.Sprintf(`<td class="right col-janpackqty">%.0f</td>`, tx.JanPackUnitQty))
			// ▼▼▼【修正】units.ResolveName を呼び出す ▼▼▼
			sb.WriteString(fmt.Sprintf(`<td class="center col-janunit">%s</td>`, units.ResolveName(tx.JanUnitName)))
			// ▲▲▲【修正ここまで】▲▲▲
			sb.WriteString(fmt.Sprintf(`<td class="right col-amount">%s</td>`, formattedSubtotal))
			sb.WriteString(fmt.Sprintf(`<td class="col-lot">%s</td>`, tx.LotNumber))
			sb.WriteString(fmt.Sprintf(`<td class="col-receipt">%s</td>`, tx.ReceiptNumber))
			sb.WriteString(fmt.Sprintf(`<td class="center col-ma">%s</td>`, tx.ProcessFlagMA))
			sb.WriteString(`</tr>`)
		}
	}
	sb.WriteString(`</tbody>`)

	return sb.String()
}
