// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\inout_details_table.js
import { showModal } from './search_modal.js';
// ▼▼▼【修正】TKR の common_table.js を参照 ▼▼▼
import { renderTransactionTableHTML } from './common_table.js';
import { clientMap } from './master_data.js';
// ▲▲▲【修正ここまで】▲▲▲

let tableBody, addRowBtn, tableContainer;
// ▼▼▼【修正】TKRの transactionTypeMap をインポート ▼▼▼
const transactionTypeMap = {
	0: "棚卸", 1: "納品", 2: "返品", 3: "処方",
    11: "入庫", 12: "出庫" // TKRにないものを追加
};
// ▲▲▲【修正ここまで】▲▲▲

function createInoutRowsHTML(record = {}) {
    const rowId = record.lineNumber || `new-${Date.now()}`;
    const janQuantity = record.janQuantity ?? 1;
    const datQuantity = record.datQuantity ?? 1;
    const nhiPrice = record.unitPrice || 0; // TKRでは unitPrice が YJ単位薬価
    const janPackInnerQty = record.janPackInnerQty || 0;
    const yjQuantity = janQuantity * janPackInnerQty;
    const subtotal = yjQuantity * nhiPrice;
    const transactionType = record.flag ? (transactionTypeMap[record.flag] || '') : '';
    console.log('Record flag:', record.flag, 'Type:', typeof record.flag, 'Mapped type:', transactionType); // デバッグ用ログ

    // ▼▼▼【修正】TKRのテーブル定義 (  ) に合わせる (空白列を削除し、col-* クラスを付与) ▼▼▼
    const upperRow = `
        <tr data-row-id="${rowId}">
            <td rowspan="2" class="col-action center"><button class="delete-row-btn btn">削除</button></td>
            <td class="col-date">${record.transactionDate || ''}</td>
            <td class="yj-jan-code col-yj display-yj-code">${record.yjCode || ''}</td>
            <td colspan="2" class="col-product left product-name-cell" style="cursor: pointer; text-decoration: underline; color: blue;">${record.productName || 'ここをクリックして製品を検索'}</td>
            <td class="right col-count display-dat-quantity">${datQuantity.toFixed(2)}</td>
            <td class="right col-yjqty display-yj-quantity">${yjQuantity.toFixed(2)}</td>
            <td class="right col-yjpackqty display-yj-pack-unit-qty">${record.yjPackUnitQty || ''}</td>
            <td class="col-yjunit display-yj-unit-name">${record.yjUnitName || ''}</td>
            <td class="right col-unitprice display-unit-price">${nhiPrice.toFixed(4)}</td>
            <td class="col-expiry"><input type="text" name="expiryDate" value="${record.expiryDate || ''}" placeholder="YYYYMM"></td>
            <td class="left col-wholesaler">${clientMap.get(record.clientCode) || record.clientCode || ''}</td>
            <td class="right col-line">${record.lineNumber || ''}</td>
        </tr>`;

    const lowerRow = `
        <tr data-row-id-lower="${rowId}">
            <td class="col-flag">${transactionType}</td>
            <td class="yj-jan-code col-jan display-jan-code">${record.productCode || record.janCode || ''}</td>
            <td class="left col-package display-package-spec">${record.packageSpec || ''}</td>
            <td class="left col-maker display-maker-name">${record.makerName || ''}</td>
            <td class="left col-form display-usage-classification">${record.usageClassification || ''}</td>
            <td class="right col-janqty"><input type="number" name="janQuantity" value="${janQuantity}" step="any"></td>
            <td class="right col-janpackqty display-jan-pack-unit-qty">${record.janPackUnitQty || ''}</td>
            <td class="col-janunit display-jan-unit-name">${record.janUnitName || ''}</td>
            <td class="right col-amount display-subtotal">${subtotal.toFixed(2)}</td>
            <td class="left col-lot"><input type="text" name="lotNumber" value="${record.lotNumber || ''}"></td>
            <td class="left col-receipt">${record.receiptNumber || ''}</td>
            <td class="left col-ma">${record.processFlagMA || ''}</td>
        </tr>`;
    // ▲▲▲【修正ここまで】▲▲▲

    return upperRow + lowerRow;
}

export function populateDetailsTable(records) {
    if (!tableBody) return;
    if (!records || records.length === 0) {
        clearDetailsTable();
        return;
    }
    tableBody.innerHTML = records.map(createInoutRowsHTML).join('');
    
    tableBody.querySelectorAll('tr[data-row-id]').forEach((row, index) => {
        if (records[index]) {
            const masterData = { ...records[index] };
            // TKRのProductMasterViewと互換性を持たせるため、productCodeもjanCodeとして保存
            masterData.productCode = masterData.janCode;
            delete masterData.id;
            row.dataset.product = JSON.stringify(masterData);
            recalculateRow(row);
        }
    });
}

export function clearDetailsTable() {
    if (tableBody) {
        tableBody.innerHTML = `<tr id="inout-placeholder-row"><td colspan="13">ヘッダーで情報を選択後、「明細を追加」ボタンを押してください。</td></tr>`;
    }
}

export function getDetailsData() {
    console.log("Collecting details data for saving...");
    const records = [];
    const allUpperRows = tableBody.querySelectorAll('tr[data-row-id]');
    allUpperRows.forEach((upperRow) => {
        const productDataString = upperRow.dataset.product;

        if (!productDataString || productDataString === '{}') {
            return;
        }

        const lowerRow = upperRow.nextElementSibling;
        if (!lowerRow) return;

        const productData = JSON.parse(productDataString);
        const janQuantity = parseFloat(lowerRow.querySelector('input[name="janQuantity"]').value) || 0;
        let datQuantity = 0; // TKRでは datQuantity は使われない
        
        const record = {
            productCode: productData.productCode, // TKRは productCode をキーとする
            productName: productData.productName,
            janQuantity: janQuantity,
            datQuantity: datQuantity,
            expiryDate: upperRow.querySelector('input[name="expiryDate"]').value,
            lotNumber: lowerRow.querySelector('input[name="lotNumber"]').value,
        };
        records.push(record);
    });
    console.log(`Total records to be saved: ${records.length}`, records);
    return records;
}

function recalculateRow(upperRow) {
    const productDataString = upperRow.dataset.product;
    if (!productDataString) return;
    const product = JSON.parse(productDataString);
    
    const lowerRow = upperRow.nextElementSibling;
    if (!lowerRow) return;
    
    const janQuantity = parseFloat(lowerRow.querySelector('[name="janQuantity"]').value) || 0;
    const nhiPrice = parseFloat(product.nhiPrice) || 0; // TKRではnhiPrice
    const janPackInnerQty = parseFloat(product.janPackInnerQty) || 0;
    const janPackUnitQty = parseFloat(product.janPackUnitQty) || 0;

    let datQuantity = 0; // TKRでは使わない
    
    const yjQuantity = janQuantity * janPackInnerQty;
    const subtotal = yjQuantity * nhiPrice;

    upperRow.querySelector('.display-dat-quantity').textContent = datQuantity.toFixed(2);
    upperRow.querySelector('.display-yj-quantity').textContent = yjQuantity.toFixed(2);
    lowerRow.querySelector('.display-subtotal').textContent = subtotal.toFixed(2);
}

export function initDetailsTable() {
    tableContainer = document.getElementById('inout-details-container');
    addRowBtn = document.getElementById('addRowBtn');
    if (!tableContainer || !addRowBtn) return;
    
    // ▼▼▼【修正】TKR の common_table.js を使用 ▼▼▼
    tableContainer.innerHTML = renderTransactionTableHTML([], "<tbody></tbody>");
    // ▲▲▲【修正ここまで】▲▲▲
    
    tableBody = tableContainer.querySelector('tbody');
    clearDetailsTable();

    addRowBtn.addEventListener('click', () => {
        const placeholderRow = document.getElementById('inout-placeholder-row');
        if (placeholderRow) {
            placeholderRow.remove();
        }
        tableBody.insertAdjacentHTML('beforeend', createInoutRowsHTML());
    });

    tableBody.addEventListener('click', (e) => {
        if (e.target.classList.contains('delete-row-btn')) {
            const upperRow = e.target.closest('tr');
            const lowerRow = upperRow.nextElementSibling;
            if(lowerRow) lowerRow.remove();
            upperRow.remove();
            if (tableBody.children.length === 0) {
                clearDetailsTable();
            }
        }
        if (e.target.classList.contains('product-name-cell')) {
            const activeRow = e.target.closest('tr');
            // ▼▼▼【修正】TKR の品目検索APIを呼び出す ▼▼▼
            showModal(activeRow, (selectedProduct, targetRow) => {
                
                // TKRのProductMasterView (selectedProduct) をデータとして保存
                targetRow.dataset.product = JSON.stringify(selectedProduct);
         
                const lowerRow = targetRow.nextElementSibling;

                targetRow.querySelector('.display-yj-code').textContent = selectedProduct.yjCode;
                targetRow.querySelector('.product-name-cell').textContent = selectedProduct.productName;
                targetRow.querySelector('.display-yj-pack-unit-qty').textContent = selectedProduct.yjPackUnitQty || '';
                targetRow.querySelector('.display-yj-unit-name').textContent = selectedProduct.yjUnitName || '';
                targetRow.querySelector('.display-unit-price').textContent = (selectedProduct.nhiPrice || 0).toFixed(4);
                
                lowerRow.querySelector('.display-jan-code').textContent = selectedProduct.productCode;
                lowerRow.querySelector('.display-package-spec').textContent = selectedProduct.formattedPackageSpec || '';
                lowerRow.querySelector('.display-maker-name').textContent = selectedProduct.makerName;
                lowerRow.querySelector('.display-usage-classification').textContent = selectedProduct.usageClassification || '';
                lowerRow.querySelector('.display-jan-pack-unit-qty').textContent = selectedProduct.janPackUnitQty || '';
                lowerRow.querySelector('.display-jan-unit-name').textContent = selectedProduct.janUnitName || '';
                
                const quantityInput = lowerRow.querySelector('input[name="janQuantity"]');
                quantityInput.focus();
                quantityInput.select();
                recalculateRow(targetRow);
            }, {
                searchMode: 'inout' // JCSHMS統合検索モードを指定
            });
            // ▲▲▲【修正ここまで】▲▲▲
        }
    });
    tableBody.addEventListener('input', (e) => {
        const upperRow = e.target.closest('tr[data-row-id]') || e.target.closest('tr[data-row-id-lower]')?.previousElementSibling;
        if(upperRow) {
            recalculateRow(upperRow);
        }
    });
}