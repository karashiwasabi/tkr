// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\deadstock.js
import { getLocalDateString } from './utils.js';

let startDateInput, endDateInput, searchBtn, resultContainer;

/**
 * 期間のデフォルト値を設定（例: 90日前から本日）
 */
function setDefaultDates() {
    const endDate = new Date();
    const startDate = new Date();
    startDate.setDate(endDate.getDate() - 90);

    if (startDateInput) {
        startDateInput.value = getLocalDateString(startDate);
    }
    if (endDateInput) {
        endDateInput.value = getLocalDateString(endDate);
    }
}

/**
 * APIを叩いて不動在庫リストを取得・描画する
 */
async function fetchAndRenderDeadStock() {
    const startDate = startDateInput.value.replace(/-/g, '');
    const endDate = endDateInput.value.replace(/-/g, '');

    if (!startDate || !endDate) {
        window.showNotification('開始日と終了日を指定してください。', 'warning');
        return;
    }

    window.showLoading('不動在庫リストを集計中...');
    resultContainer.innerHTML = '<p>検索中...</p>';

    try {
        const params = new URLSearchParams({ startDate, endDate });
        const response = await fetch(`/api/deadstock/list?${params.toString()}`);

        if (!response.ok) {
            const errorText = await response.text(); // Get the plain text error
            throw new Error(errorText || `サーバーエラー (HTTP ${response.status})`);
        }
        
        const data = await response.json(); // Now it's safe to parse JSON

        if (data.errors && data.errors.length > 0) {
            window.showNotification(data.errors.join('\n'), 'error');
        }

        renderDeadStockTable(data.items);

    } catch (error) {
        console.error('Failed to fetch dead stock list:', error);
        resultContainer.innerHTML = `<p class="status-error">エラー: ${error.message}</p>`;
        window.showNotification(`エラー: ${error.message}`, 'error');
    } finally {
        window.hideLoading();
    }
}

/**
 * 取得した不動在庫アイテムをHTMLテーブルに変換して描画する
 */
function renderDeadStockTable(items) {
    if (!items || items.length === 0) {
        resultContainer.innerHTML = '<p>対象期間の不動在庫は見つかりませんでした。</p>';
        return;
    }

    const header = `
        <table id="deadstock-table" class="data-table">
            <thead>
                <tr>
                    <th class="col-ds-key">PackageKey</th>
                    <th class="col-ds-name">製品名</th>
                    <th class="col-ds-qty">現在庫(YJ)</th>
                    <th class="col-ds-details">棚卸明細 (JAN / 包装仕様 / 在庫数 / 単位 / 期限 / ロット)</th>
                    </tr>
            </thead>
            <tbody>
    `;

    const body = items.map(item => {
        let lotHtml = '棚卸履歴なし'; // デフォルト（在庫0の場合）
        if (item.lotDetails && item.lotDetails.length > 0) {
            lotHtml = '<ul class="lot-details-list">';
            // ▼▼▼【ここから修正】明細の表示内容を変更 ▼▼▼
            lotHtml += item.lotDetails.map(lot => {
                const janQty = (lot.JanQuantity || 0).toFixed(2);
                const janCode = lot.JanCode || '(JANなし)';
                const pkgSpec = lot.PackageSpec || '(仕様なし)';
                const lotNum = lot.LotNumber || '(ロットなし)';
                const expiry = lot.ExpiryDate || '(期限なし)';
                const unitName = lot.JanUnitName || ''; // 単位名
                
                return `<li>${janCode} / ${pkgSpec} / ${janQty} ${unitName} / ${expiry} / ${lotNum}</li>`;
            }).join('');
            // ▲▲▲【修正ここまで】▲▲▲
            lotHtml += '</ul>';
        } else if (item.stockQuantityYj > 0) {
            lotHtml = '<span class="status-error">在庫あり (明細なし)</span>';
        }

        const stockQty = (item.stockQuantityYj || 0).toFixed(2);
        return `
            <tr>
                <td class="left">${item.packageKey}</td>
                <td class="left">${item.productName || '(品名不明)'}</td>
                <td class="right">${stockQty}</td>
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

/**
 * 不動在庫ビューの初期化
 */
export function initDeadStockView() {
    startDateInput = document.getElementById('ds-start-date');
    endDateInput = document.getElementById('ds-end-date');
    searchBtn = document.getElementById('ds-search-btn');
    resultContainer = document.getElementById('deadstock-result-container');

    if (!searchBtn) return;

    searchBtn.addEventListener('click', fetchAndRenderDeadStock);
    
    // デフォルト日付を設定
    setDefaultDates();
    console.log("DeadStock View Initialized.");
}