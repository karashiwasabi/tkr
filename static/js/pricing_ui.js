// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\pricing_ui.js
// (新規作成)
import { wholesalerMap } from './master_data.js';

// DOM要素
let outputContainer, makerFilterInput, unregisteredFilterCheckbox, wholesalerSelect;

// モジュール内グローバル変数 (状態管理)
let fullPricingData = [];
let orderedWholesalers = [];
let lastSelectedWholesaler = '';

// TKRはIntl.NumberFormatをサポートしていない可能性があるため、簡易版を使用
const formatCurrency = (value) => {
    const num = Math.floor(value || 0);
    return `￥${num.toString().replace(/(\d)(?=(\d{3})+(?!\d))/g, '$1,')}`;
};

/**
 * 状態（データ）を外部から設定
 */
export function setFullPricingData(data) {
    fullPricingData = data;
}
export function setOrderedWholesalers(data) {
    orderedWholesalers = data;
}
export function getFullPricingData() {
    return fullPricingData;
}

/**
 * 比較テーブルをHTMLに描画します。
 */
export function renderComparisonTable(data) {
    if (!data || data.length === 0) {
        outputContainer.innerHTML = '<p>表示対象のデータがありませんでした。</p>';
        return;
    }

    const wholesalerHeaders = orderedWholesalers.length > 0 ?
        orderedWholesalers.map(w => `<th>${w}</th>`).join('') : '';

    // 卸名 -> 卸コード の逆引きマップ
    const wholesalerReverseMap = new Map();
    for (const [code, name] of wholesalerMap.entries()) {
        wholesalerReverseMap.set(name, code);
    }

    let tableHTML = `
        <table class="data-table">
            <thead>
                <tr>
                    <th rowspan="2" class="col-product" style="width: 20%;">製品名</th>
                    <th rowspan="2" class="col-package" style="width: 15%;">包装</th>
                    <th rowspan="2" class="col-maker" style="width: 15%;">メーカー</th>
                    <th rowspan="2" class="col-unitprice" style="width: 8%;">現納入価</th>
                    <th colspan="${orderedWholesalers.length || 1}">卸提示価格</th>
                    <th rowspan="2" class="col-wholesaler" style="width: 10%;">採用卸</th>
                    <th rowspan="2" class="col-unitprice" style="width: 10%;">決定納入価</th>
                </tr>
                <tr>
                    ${wholesalerHeaders}
                </tr>
            </thead>
            <tbody>
    `;
    data.forEach(p => {
        const productCode = p.productCode;
        let wholesalerOptions = '<option value="">--- 選択 ---</option>';
        for (const [wCode, wName] of wholesalerMap.entries()) {
            const isSelected = (wCode === p.supplierWholesale);
            wholesalerOptions += `<option value="${wCode}" ${isSelected ? 'selected' : ''}>${wName}</option>`;
        }
        const quoteCells = orderedWholesalers.length > 0 
            ? orderedWholesalers.map(w => {
            const price = (p.quotes || {})[w];
            if (price === undefined) return '<td>-</td>';
            const lowestPrice = Math.min(...Object.values(p.quotes || {}).filter(v => typeof v === 'number'));
            const style = (price === lowestPrice) ? 'style="background-color: #d1e7dd; font-weight: bold;"' : '';
            return `<td class="right" ${style}>${price.toFixed(2)}</td>`;
        }).join('') : '<td>-</td>';
        
        const initialPrice = p.purchasePrice > 0 ? p.purchasePrice.toFixed(2) : '';
        
        tableHTML += `
            <tr data-product-code="${productCode}">
                <td class="left">${p.productName}</td>
                <td class="left">${p.formattedPackageSpec || ''}</td>
                <td class="left">${p.makerName}</td>
                <td class="right">${p.purchasePrice.toFixed(2)}</td>
                ${quoteCells}
                <td><select class="supplier-select">${wholesalerOptions}</select></td>
                <td><input type="number" class="manual-price-input" step="0.01" value="${initialPrice}"></td>
                </tr>
        `;
    });
    tableHTML += `</tbody></table>`;
    outputContainer.innerHTML = tableHTML;
}

/**
 * フィルターを適用してテーブルを再描画します。
 */
export function applyFiltersAndRender() {
    let dataToRender = fullPricingData;
    if (unregisteredFilterCheckbox.checked) {
        dataToRender = dataToRender.filter(p => !p.supplierWholesale);
    }
    
    const filterText = makerFilterInput.value.trim().toLowerCase();
    if (filterText) {
        dataToRender = dataToRender.filter(p => 
            p.makerName && p.makerName.toLowerCase().includes(filterText)
        );
    }
    renderComparisonTable(dataToRender); 
}

/**
 * 卸業者ドロップダウン（ヘッダー）を読み込みます。
 */
export function loadWholesalerDropdown() {
    wholesalerSelect.innerHTML = '<option value="">選択してください</option>';
    for (const [code, name] of wholesalerMap.entries()) {
        const opt = document.createElement('option');
        opt.value = code; 
        opt.textContent = name; 
        wholesalerSelect.appendChild(opt); 
    }
    if (lastSelectedWholesaler) {
        wholesalerSelect.value = lastSelectedWholesaler;
    }
}

/**
 * 「最安値に設定」ボタンのハンドラ
 */
export function handleSetLowestPrice() {
    const rows = outputContainer.querySelectorAll('tbody tr');
    if (rows.length === 0) return;
    
    const wholesalerReverseMap = new Map();
    for (const [code, name] of wholesalerMap.entries()) {
        wholesalerReverseMap.set(name, code);
    }

    rows.forEach(row => {
        const productCode = row.dataset.productCode;
        const productData = fullPricingData.find(p => p.productCode === productCode);
        if (!productData || !productData.quotes) return;

        let lowestPrice = Infinity;
        let bestWholesalerName = '';
        
        orderedWholesalers.forEach(wholesalerName => {
            const price = productData.quotes[wholesalerName];
 
            if (price !== undefined && price < lowestPrice) {
                lowestPrice = price;
                bestWholesalerName = wholesalerName;
            }
        });

        if (bestWholesalerName) {
            const bestWholesalerCode = wholesalerReverseMap.get(bestWholesalerName);
            const supplierSelect = row.querySelector('.supplier-select');
            const priceInput = row.querySelector('.manual-price-input');
            
            if(supplierSelect) supplierSelect.value = bestWholesalerCode;
            if(priceInput) priceInput.value = lowestPrice.toFixed(2);
        }
    });
    window.showNotification('すべての品目を最安値に設定しました。', 'success');
}

/**
 * テーブル内の「採用卸」変更イベントハンドラ
 */
export function handleSupplierChange(event) {
    if (!event.target.classList.contains('supplier-select')) return;
    const row = event.target.closest('tr');
    const productCode = row.dataset.productCode;
    const selectedWholesalerName = event.target.options[event.target.selectedIndex].text;
    const priceInput = row.querySelector('.manual-price-input');
    const productData = fullPricingData.find(p => p.productCode === productCode);
    
    if (productData && productData.quotes) {
        const newPrice = productData.quotes[selectedWholesalerName];
        if (newPrice !== undefined) {
            priceInput.value = newPrice.toFixed(2);
        } else {
            priceInput.value = productData.purchasePrice > 0 ? productData.purchasePrice.toFixed(2) : '';
        }
    }
}

/**
 * 卸選択（ヘッダー）の変更イベントハンドラ
 */
export function handleWholesalerSelectChange() {
    lastSelectedWholesaler = wholesalerSelect.value;
}

/**
 * UI関連のDOM要素をキャッシュします。
 */
export function initPricingUI() {
    outputContainer = document.getElementById('pricing-output-container');
    makerFilterInput = document.getElementById('pricing-maker-filter');
    unregisteredFilterCheckbox = document.getElementById('pricing-unregistered-filter');
    wholesalerSelect = document.getElementById('pricing-wholesaler-select');
}