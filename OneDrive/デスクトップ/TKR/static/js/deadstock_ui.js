// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\deadstock_ui.js
// (新規作成)
import { getLocalDateString } from './utils.js';

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
                    <th class="col-ds-key">PackageKey</th>
                 <th class="col-ds-name">製品名</th>
                    <th class="col-ds-qty">現在庫(JAN)</th>
                    <th class="col-ds-details">棚卸明細 (JAN / 包装仕様 / 在庫数 / 単位 / 期限 / ロット)</th>
                 </tr>
            </thead>
            <tbody>
    `;
    const body = items.map(item => {
        let lotHtml = ''; 
        if (item.lotDetails && item.lotDetails.length > 0) {
            lotHtml = '<ul class="lot-details-list">';
            lotHtml += item.lotDetails.map(lot => {
                const janQty = (lot.JanQuantity || 0).toFixed(2);
                const janCode = lot.JanCode || '(JANなし)';
                const pkgSpec = lot.PackageSpec || '(仕様なし)';
                const lotNum = lot.LotNumber || '(ロットなし)';
                const expiry = lot.ExpiryDate || '(期限なし)';
                const unitName = lot.JanUnitName || ''; 
                
                return `<li>${janCode} / ${pkgSpec} / ${janQty} ${unitName} / ${expiry} / ${lotNum}</li>`;
            }).join('');
            lotHtml += '</ul>';
        } else if (item.stockQuantityJan > 0) {
            lotHtml = '<span class="status-error">在庫あり (明細なし)</span>';
        }

        const stockQty = (item.stockQuantityJan || 0).toFixed(2);
        
        const buttonHtml = item.yjCode ?
 `<button class="btn adjust-inventory-btn" data-yj-code="${item.yjCode}">棚卸調整</button>` : '';

        return `
           <tr>
                <td class="center col-ds-action">${buttonHtml}</td>
                 <td class="left">${item.packageKey}</td>
                <td class="left">${item.productName || '(品名不明)'}</td>
                <td class="right col-ds-qty">${stockQty}</td>
                 <td class="left">${lotHtml}</td>
            </tr>
         `;
    }).join('');

    const footer = `
            </tbody>
         </table>
    `;
    resultContainer.innerHTML = header + body + footer;
}