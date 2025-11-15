// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\backorder_ui.js
// (新規作成)
import { hiraganaToKatakana } from './utils.js';
import { wholesalerMap } from './master_data.js';

let outputContainer, searchKanaInput, searchWholesalerInput;
let allBackorders = []; // APIから取得した全発注残をキャッシュ

/**
 * DOM要素をキャッシュする
 */
export function cacheDOMElements(elements) {
    outputContainer = elements.outputContainer;
    searchKanaInput = elements.searchKanaInput;
    searchWholesalerInput = elements.searchWholesalerInput;
}

/**
 * キャッシュされた発注残データをセットする
 */
export function setBackorders(data) {
    allBackorders = data;
}

/**
 * 絞り込みと描画を実行します。
 */
export function filterAndRender() {
    if (!outputContainer) return;

    const kanaFilter = hiraganaToKatakana(searchKanaInput.value.trim().toLowerCase());
    const wholesalerFilter = searchWholesalerInput.value.trim().toLowerCase();

    const filteredData = allBackorders.filter(bo => {
        const nameMatch = !kanaFilter || 
                         (bo.productName && bo.productName.toLowerCase().includes(kanaFilter)) ||
                         (bo.yjCode && bo.yjCode.toLowerCase().includes(kanaFilter)) ||
                         (bo.janCode && bo.janCode.toLowerCase().includes(kanaFilter));
        
        const wholesalerMatch = !wholesalerFilter ||
                         (bo.wholesalerCode && bo.wholesalerCode.toLowerCase().includes(wholesalerFilter));

        return nameMatch && wholesalerMatch;
    });

    renderBackorders(filteredData);
}

/**
 * 発注残リストのテーブルHTMLを描画します。
 */
export function renderBackorders(data) {
    if (!data || data.length === 0) {
        outputContainer.innerHTML = "<p>対象の発注残はありません。</p>";
        return;
    }

    // 1. データを order_date (YYYYMMDDHHMMSS) ごとにグループ化
    const groups = new Map();
    data.forEach(bo => {
         const key = bo.orderDate; // YYYYMMDDHHMMSS
        if (!groups.has(key)) {
             groups.set(key, []);
        }
        groups.get(key).push(bo);
    });

    let html = `
        <div class="backorder-controls">
            <button class="btn" id="bo-bulk-delete-btn">選択した項目を一括削除</button>
        </div>
        <table id="backorder-table" class="data-table">
            <thead>
                 <tr>
                     <th class="col-bo-check"><input type="checkbox" id="bo-select-all-checkbox"></th>
                     <th class="col-bo-jan">JANコード</th>
                    <th class="col-bo-name">製品名</th>
                     <th class="col-bo-remain">残数量(YJ)</th>
                     <th class="col-bo-wholesaler">卸</th>
                     <th class="col-bo-action">操作</th>
                </tr>
             </thead>
    `;
    
    // 2. グループごとにテーブルを描画 (tbody でグループ化)
    groups.forEach((items, orderDateKey) => {
         const totalItems = items.length;

        html += `
            <tbody class="backorder-group">
                 <tr class="group-header">
                    <td colspan="3">
                         <strong>${orderDateKey}</strong> (${totalItems}品目)
                     </td>
                     <td colspan="3" class="right">
                         <button class="btn delete-backorder-group-btn" data-order-date="${orderDateKey}">
                             この発注(${totalItems}品目)を一括削除
                         </button>
                     </td>
                </tr>
        `;

        // グループ内の品目
        items.forEach(bo => {
            const wholesalerName = wholesalerMap.get(bo.wholesalerCode) || bo.wholesalerCode || '---';
            html += `
                 <tr data-id="${bo.id}" data-yj-code="${bo.yjCode}" data-jan-code="${bo.janCode}">
                     <td class="center col-bo-check"><input type="checkbox" class="bo-select-checkbox"></td>
                     <td class="col-bo-jan">${bo.janCode || '(JANなし)'}</td>
                    <td class="left col-bo-name">${bo.productName}</td>
                    <td class="right col-bo-remain">${bo.remainingQuantity.toFixed(2)}</td>
                     <td class="left col-bo-wholesaler">${wholesalerName}</td>
                     <td class="center col-bo-action">
                         <button class="btn delete-backorder-btn">削除</button>
                         <button class="btn adjust-inventory-btn" data-yj-code="${bo.yjCode}">棚卸調整</button>
                     </td>
                </tr>
             `;
        });

        html += `</tbody>`;
    });

    html += `</table>`;
    outputContainer.innerHTML = html;
}