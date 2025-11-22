// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\backorder_ui.js
import { hiraganaToKatakana } from './utils.js';

let outputContainer, searchKanaInput;
let allBackorders = [];

export function cacheDOMElements(elements) {
    outputContainer = elements.outputContainer;
    searchKanaInput = elements.searchKanaInput;
    // 卸コード入力欄のキャッシュを削除
}

export function setBackorders(data) {
    allBackorders = data;
}

export function filterAndRender() {
    if (!outputContainer) return;

    const kanaFilter = searchKanaInput && searchKanaInput.value ? hiraganaToKatakana(searchKanaInput.value.trim().toLowerCase()) : '';
    
    const filteredData = allBackorders.filter(bo => {
        // 卸コードによる絞り込みを削除し、製品名・JAN・YJのみで検索
        const nameMatch = !kanaFilter || 
                         (bo.productName && bo.productName.toLowerCase().includes(kanaFilter)) ||
                         (bo.yjCode && bo.yjCode.toLowerCase().includes(kanaFilter)) ||
                         (bo.janCode && bo.janCode.toLowerCase().includes(kanaFilter));
        return nameMatch;
    });
    renderBackorders(filteredData);
}

export function renderBackorders(data) {
    if (!data || data.length === 0) {
        outputContainer.innerHTML = "<p>対象の発注残はありません。</p>";
        return;
    }

    const groups = new Map();
    data.forEach(bo => {
         const key = bo.orderDate;
        if (!groups.has(key)) {
             groups.set(key, []);
        }
        groups.get(key).push(bo);
    });

    // テーブルヘッダーから「卸」列を削除
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
                     <th class="col-bo-action">操作</th>
                </tr>
             </thead>
    `;

    groups.forEach((items, orderDateKey) => {
         const totalItems = items.length;

        html += `
            <tbody class="backorder-group">
                 <tr class="group-header">
                    <td colspan="4">
                          <strong>${orderDateKey}</strong> (${totalItems}品目)
                     </td>
                     <td class="right">
                         <button class="btn delete-backorder-group-btn" data-order-date="${orderDateKey}">
                             この発注(${totalItems}品目)を一括削除
                         </button>
                     </td>
                </tr>
        `;

        items.forEach(bo => {
            // 卸列(td)を削除
            html += `
                 <tr data-id="${bo.id}" data-yj-code="${bo.yjCode}" data-jan-code="${bo.janCode}">
                     <td class="center col-bo-check"><input type="checkbox" class="bo-select-checkbox"></td>
                     <td class="col-bo-jan">${bo.janCode || '(JANなし)'}</td>
                     <td class="left col-bo-name">${bo.productName}</td>
                     <td class="right col-bo-remain">${bo.remainingQuantity.toFixed(2)}</td>
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