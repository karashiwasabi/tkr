// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\reorder.js
import { hiraganaToKatakana } from './utils.js';

let searchBtn, resultContainer, kanaInput, usageClassRadios;

/**
 * 発注点リストテーブルのHTMLを生成する
 * @param {Array<object>} items - APIから取得した発注点アイテムの配列
 * @returns {string} HTML文字列
 */
function renderReorderTable(items) {
    if (!items || items.length === 0) {
        return '<p>発注が必要な品目はありません。</p>';
    }

    const header = `
        <table id="reorder-table" class="data-table">
            <thead>
                <tr>
                    <th class="col-yj">YJコード</th>
                    <th class="col-product">製品名</th>
                    <th class="col-package">包装 (PackageKey)</th>
                    <th class="col-yjqty">現在庫(YJ)</th>
                    <th class="col-yjqty">発注点(YJ)</th>
                    <th class="col-yjqty">最大出庫(YJ)</th>
                    <th class="col-yjqty">予製引当(YJ)</th>
                </tr>
            </thead>
            <tbody>
    `;

    const body = items.map(item => {
        const currentStock = (item.effectiveEndingBalance || 0).toFixed(2);
        const reorderPoint = (item.reorderPoint || 0).toFixed(2);
        const maxUsage = (item.maxUsage || 0).toFixed(2);
        const precompTotal = (item.precompoundedTotal || 0).toFixed(2);

        // PackageKeyからYJコードなどを除外し、包装仕様のみを抽出
        const keyParts = (item.packageKey || '').split('|');
        const displayPackage = keyParts.length === 4 ? `${keyParts[1]} ${keyParts[2]}${keyParts[3]}` : item.packageKey;

        return `
            <tr>
                <td class="yj-jan-code col-yj">${item.yjCode || ''}</td>
                <td class="left col-product">${item.productName || ''}</td>
                <td class="left col-package">${displayPackage}</td>
                <td class="right col-yjqty status-error">${currentStock}</td>
                <td class="right col-yjqty">${reorderPoint}</td>
                <td class="right col-yjqty">${maxUsage}</td>
                <td class="right col-yjqty">${precompTotal}</td>
            </tr>
        `;
    }).join('');

    const footer = `</tbody></table>`;
    return header + body + footer;
}

/**
 * 発注点リストビューの初期化
 */
export function initReorderView() {
    searchBtn = document.getElementById('reorder-search-btn');
    resultContainer = document.getElementById('reorder-result-container');
    kanaInput = document.getElementById('reorder-kana-search');
    usageClassRadios = document.getElementById('reorder-usage-class');

    if (searchBtn) {
        searchBtn.addEventListener('click', fetchAndRenderReorder);
    }
    if (kanaInput) {
        kanaInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') {
                fetchAndRenderReorder();
            }
        });
    }

    console.log("Reorder View Initialized.");
}

/**
 * 発注点リストのデータを取得して描画する
 */
export async function fetchAndRenderReorder() {
    if (!resultContainer) return;

    const kanaName = kanaInput ? hiraganaToKatakana(kanaInput.value.trim()) : '';
    const selectedUsageRadio = usageClassRadios ? usageClassRadios.querySelector('input[name="reorder_usage_class"]:checked') : null;
    const dosageForm = selectedUsageRadio ? selectedUsageRadio.value : 'all';

    const params = new URLSearchParams();
    params.append('kanaName', kanaName);
    params.append('dosageForm', dosageForm);

    window.showLoading('発注点リストを集計中...');
    resultContainer.innerHTML = '<p>検索中...</p>';

    try {
        const response = await fetch(`/api/reorder/list?${params.toString()}`);
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText || `サーバーエラー (HTTP ${response.status})`);
        }
        
        const items = await response.json();
        resultContainer.innerHTML = renderReorderTable(items);
        window.showNotification(`${items.length}件の品目が発注点を下回っています。`, 'success');

    } catch (error) {
        console.error('Failed to fetch reorder list:', error);
        resultContainer.innerHTML = `<p class="status-error">エラー: ${error.message}</p>`;
        window.showNotification(`エラー: ${error.message}`, 'error');
    } finally {
        window.hideLoading();
    }
}