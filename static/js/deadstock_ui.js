// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\deadstock_ui.js
import { getLocalDateString, getExpiryStatus } from './utils.js';

// DOM要素のキャッシュ
let startDateInput, endDateInput, csvDateInput, resultContainer, excludeZeroStockCheckbox;

/**
 * メインの initDeadStockView からDOM要素を受け取り、キャッシュする
 */
export function cacheDOMElements(elements) {
    startDateInput = elements.startDateInput;
    endDateInput = elements.endDateInput;
    csvDateInput = elements.csvDateInput;
    resultContainer = elements.resultContainer;
    excludeZeroStockCheckbox = elements.excludeZeroStockCheckbox;
}

/**
 * デフォルトの日付を設定する
 */
export function setDefaultDates() {
    const endDate = new Date();
    const startDate = new Date();
    startDate.setDate(endDate.getDate() - 90);

    if (startDateInput) {
        startDateInput.value = getLocalDateString(startDate);
    }
    if (endDateInput) {
        endDateInput.value = getLocalDateString(endDate);
    }
    if (csvDateInput) {
        csvDateInput.value = getLocalDateString(endDate);
    }
}

/**
 * 不動在庫テーブルを描画する
 */
export function renderDeadStockTable(items) {
    if (!items || items.length === 0) {
        resultContainer.innerHTML = '<p>対象期間の不動在庫は見つかりませんでした。</p>';
        return;
    }

    const header = `
     <table id="deadstock-table" class="data-table">
         <thead>
             <tr>
              <th class="col-ds-action">操作</th>
              <th class="col-ds-product">製品情報 (PackageKey)</th>
              <th class="col-ds-qty">現在庫 (JAN換算)</th>
              <th class="col-ds-details">棚卸明細 (JAN / 包装仕様 / 在庫数 / 単位 / 期限 / ロット)</th>
          </tr>
         </thead>
         <tbody>
    `;
    
    const body = items.map(item => {
         let lotHtml = ''; 
         // ▼▼▼ 追加: 明細合計の計算 ▼▼▼
         let detailsSumJan = 0;
         // ▲▲▲ 追加ここまで ▲▲▲

         if (item.lotDetails && item.lotDetails.length > 0) {
             lotHtml = '<ul class="lot-details-list">';
             lotHtml += item.lotDetails.map(lot => {
                 const janQty = (lot.JanQuantity || 0);
                 // ▼▼▼ 追加: 合計に加算 ▼▼▼
                 detailsSumJan += janQty;
                 // ▲▲▲ 追加ここまで ▲▲▲
                 const janQtyDisplay = janQty.toFixed(2);
                 const janCode = lot.JanCode || '(JANなし)';
                 const pkgSpec = lot.PackageSpec || '(仕様なし)';
                 const lotNum = lot.LotNumber || '(ロットなし)';
                 const expiry = lot.ExpiryDate || '(期限なし)';
                 const unitName = lot.JanUnitName || ''; 
                 
                 const statusClass = getExpiryStatus(lot.ExpiryDate);
                 const expiryDisplay = statusClass ? `<span class="${statusClass}">${expiry}</span>` : expiry;

                 return `<li>${janCode} / ${pkgSpec} / ${janQtyDisplay} ${unitName} / ${expiryDisplay} / ${lotNum}</li>`;
             }).join('');
             lotHtml += '</ul>';
         } else if (item.currentStockYj > 0 && (!item.lotDetails || item.lotDetails.length === 0)) {
             // 現在庫があるのに棚卸明細がない（棚卸後にのみ入荷した等）
             lotHtml = '<span style="color: #666; font-size: 0.9em;">(前回棚卸明細なし)</span>';
         } else {
             lotHtml = '<span style="color: #999;">(在庫なし)</span>';
         }

        // ▼▼▼ 修正: 表示と赤字判定ロジック ▼▼▼
       
        // 1. 表示用在庫数 = 現在の理論在庫 (CurrentStockYj) をJAN換算
        let currentStockJan = 0;
        if (item.janPackInnerQty > 0) {
            currentStockJan = item.currentStockYj / item.janPackInnerQty;
        }
        const displayStockQty = currentStockJan.toFixed(2);

        // 2. 比較対象の決定 (棚卸明細があればその合計、なければ内部的な棚卸数)
        //    これにより、当日入荷等で内部的な棚卸数が0になっていても、明細(2個)と現在庫(2個)が合えば一致とみなす
        let referenceStockJan = 0;
        if (item.lotDetails && item.lotDetails.length > 0) {
            referenceStockJan = detailsSumJan;
        } else {
            if (item.janPackInnerQty > 0) {
                referenceStockJan = item.stockQuantityYj / item.janPackInnerQty;
            }
        }

        const diff = Math.abs(currentStockJan - referenceStockJan);

        let qtyDisplayHtml = displayStockQty;

        // 3. 差異があれば赤字にする
        if (diff > 0.001) {
            // 参考情報として「比較対象の数量」を表示
            qtyDisplayHtml = `
                <span class="status-diff" title="現在(理論): ${displayStockQty} / 比較対象: ${referenceStockJan.toFixed(2)}">
                    ${displayStockQty}
                </span>
                <br><span style="font-size:0.8em; color:#dc3545;">(棚卸時: ${referenceStockJan.toFixed(2)})</span>
            `;
        }
        // ▲▲▲ 修正ここまで ▲▲▲
        
        const buttonHtml = item.yjCode ? `<button class="btn adjust-inventory-btn" data-yj-code="${item.yjCode}">棚卸調整</button>` : '';

        return `
           <tr>
             <td class="center col-ds-action">${buttonHtml}</td>
             <td class="left col-ds-product">
                  <span class="product-name">${item.productName || '(品名不明)'}</span><br>
                  <span class="package-key-sub">${item.packageKey}</span>
             </td>
              <td class="right col-ds-qty">${qtyDisplayHtml}</td>
              <td class="left col-ds-details">${lotHtml}</td>
           </tr>
       `;
    }).join('');

    const footer = `
         </tbody>
         </table>
    `;
    resultContainer.innerHTML = header + body + footer;
}