// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\common_table.js
// ▼▼▼【修正】clientMap もインポートする ▼▼▼
import { wholesalerMap, clientMap } from './master_data.js';
// ▲▲▲【修正ここまで】▲▲▲

// (WASABI: inventory_adjustment_ui.js より)
const transactionTypeMap = {
	0: "棚卸", 1: "納品", 2: "返品", 3: "処方",
   
 11: "入庫", 12: "出庫"
};
/**
 * 空のテーブルHTMLを生成します。
 * (dat.js/usage.js より移設)
 */
export function renderEmptyTableHTML(columnCount = 13) {
    return `
   
 <thead>
         <tr>
            <th rowspan="2" 
class="col-action">－</th>
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
    </thead>
    <tbody>
         <tr><td 
colspan="13">登録されたデータはありません。</td></tr>
    </tbody>
    `;
}


/**
 * 取引履歴テーブルのHTMLを生成します。
 * (inventory_adjustment_ui.js の renderStandardTable を移設・改名)
 * @param {Array<object>} records - トランザクションレコードの配列
 * @param {string|null} customBody - (オプション) tbodyの内容をカスタムHTMLで置き換える場合
 * @returns {string} - テーブル全体のHTML文字列
 */
export function renderTransactionTableHTML(records, customBody = null) {
    const header = 
`<thead>
        <tr><th rowspan="2" class="col-action">－</th><th class="col-date">日付</th><th class="col-yj">YJ</th><th colspan="2" class="col-product">製品名</th><th class="col-count">個数</th><th class="col-yjqty">YJ数量</th><th class="col-yjpackqty">YJ包装数</th><th class="col-yjunit">YJ単位</th><th class="col-unitprice">単価</th><th class="col-expiry">期限</th><th class="col-wholesaler">卸</th><th class="col-line">行</th></tr>
  
      <tr><th class="col-flag">種別</th><th class="col-jan">JAN</th><th class="col-package">包装</th><th class="col-maker">メーカー</th><th class="col-form">剤型</th><th class="col-janqty">JAN数量</th><th class="col-janpackqty">JAN包装数</th><th class="col-janunit">JAN単位</th><th class="col-amount">金額</th><th class="col-lot">ロット</th><th class="col-receipt">伝票番号</th><th class="col-ma">MA</th></tr></thead>`;
    let bodyHtml;
    if (customBody) {
 
       bodyHtml = customBody;
    } else {
         // ▼▼▼【ここを修正】colspan="14" -> "13" ▼▼▼
        bodyHtml = `<tbody>${(!records || records.length === 0) ?
 '<tr><td colspan="13">対象データがありません。</td></tr>' : records.map(rec => {
  
          
             let 
clientDisplayHtml = '';
             // ▼▼▼【修正】rec.clientCode が存在する場合のみ trim() を実行 ▼▼▼
    
         const clientCode = rec.clientCode ? rec.clientCode.trim() : '';
        
     // ▲▲▲【修正ここまで】▲▲▲

             // ▼▼▼【ここから修正】flag 11, 12 の場合に clientMap を参照するロジックを追加 ▼▼▼
             if (rec.flag === 1 || 
rec.flag === 2) { // 納品・返品
                 clientDisplayHtml = wholesalerMap.get(clientCode) || 
clientCode || '';
             } else if (rec.flag === 11 || rec.flag === 12) { // 入庫・出庫
                 clientDisplayHtml = clientMap.get(clientCode) || clientCode || '';
    
         } else {
             
    clientDisplayHtml = clientCode || ''; // 処方(3), 棚卸(0) など
             }
             // ▲▲▲【修正ここまで】▲▲▲

 
           // ▼▼▼【修正】ハードコードされた "left", "right" クラスを削除 ▼▼▼
        
     const top = `<tr><td rowspan="2" class="col-action"></td>
              
   <td class="col-date">${rec.transactionDate || ''}</td><td class="yj-jan-code col-yj">${rec.yjCode || ''}</td><td class="col-product" colspan="2">${rec.productName || ''}</td>
          
       <td class="col-count">${rec.datQuantity?.toFixed(2) || ''}</td><td class="col-yjqty">${rec.yjQuantity?.toFixed(2) || ''}</td><td class="col-yjpackqty">${rec.yjPackUnitQty || ''}</td><td class="col-yjunit">${rec.yjUnitName || ''}</td>
     
            <td class="col-unitprice">${rec.unitPrice?.toFixed(4) || ''}</td><td class="col-expiry">${rec.expiryDate || ''}</td><td class="col-wholesaler">${clientDisplayHtml}</td><td class="col-line">${rec.lineNumber || ''}</td></tr>`;
            
            const 
bottom = `<tr><td class="col-flag">${transactionTypeMap[rec.flag] || rec.flag}</td><td class="yj-jan-code col-jan">${rec.janCode || ''}</td><td class="col-package">${rec.packageSpec || ''}</td><td class="col-maker">${rec.makerName || ''}</td>
         
        <td class="col-form">${rec.usageClassification || ''}</td><td class="col-janqty">${rec.janQuantity?.toFixed(2) || ''}</td><td class="col-janpackqty">${rec.janPackUnitQty || ''}</td><td class="col-janunit">${rec.janUnitName || ''}</td>
    
             <td class="col-amount">${rec.subtotal?.toFixed(2) || ''}</td><td class="col-lot">${rec.lotNumber || ''}</td><td class="col-receipt">${rec.receiptNumber || ''}</td><td class="col-ma">${rec.processFlagMA 
|| ''}</td></tr>`;
// ▲▲▲【修正ここまで】▲▲▲
            return top + bottom;
        }).join('')}</tbody>`;
    }
    
return `<table class="data-table">${header}${bodyHtml}</table>`;
}