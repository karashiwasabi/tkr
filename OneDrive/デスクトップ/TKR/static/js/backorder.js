// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\backorder.js
import { hiraganaToKatakana } from './utils.js';
import { wholesalerMap } from './master_data.js';

let view, outputContainer, searchKanaInput, searchWholesalerInput, searchBtn;
let allBackorders = []; // APIから取得した全発注残をキャッシュ

// ▼▼▼【削除】formatSimplePackageSpec は不要になったため削除 ▼▼▼
/*
function formatSimplePackageSpec(bo) {
    // ...
}
*/
// ▲▲▲【削除ここまで】▲▲▲

// ▼▼▼【削除】formatOrderDateTime は不要になったため削除 ▼▼▼
/*
function formatOrderDateTime(dateTimeStr) {
    // ...
}
*/
// ▲▲▲【削除ここまで】▲▲▲

/**
 * 絞り込みと描画を実行します。
 */
function filterAndRender() {
    if (!outputContainer) return;

    const kanaFilter = hiraganaToKatakana(searchKanaInput.value.trim().toLowerCase());
    const wholesalerFilter = searchWholesalerInput.value.trim().toLowerCase();

    const filteredData = allBackorders.filter(bo => {
        // ▼▼▼【修正】検索対象を JAN, YJ, 製品名 に変更 ▼▼▼
        const nameMatch = !kanaFilter || 
                        (bo.productName && bo.productName.toLowerCase().includes(kanaFilter)) ||
                        (bo.yjCode && bo.yjCode.toLowerCase().includes(kanaFilter)) ||
                        (bo.janCode && bo.janCode.toLowerCase().includes(kanaFilter));
        // ▲▲▲【修正ここまで】▲▲▲
        
        const wholesalerMatch = !wholesalerFilter ||
                        (bo.wholesalerCode && bo.wholesalerCode.toLowerCase().includes(wholesalerFilter));

        return nameMatch && wholesalerMatch;
    });

    renderBackorders(filteredData);
}

/**
 * 発注残リストのテーブルHTMLを描画します。
 */
function renderBackorders(data) {
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

        // ▼▼▼【ここから修正】グループヘッダーをご要望の形式に変更 ▼▼▼
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
        // ▲▲▲【修正ここまで】▲▲▲

        // グループ内の品目
        items.forEach(bo => {
            // ▼▼▼【ここから修正】列をご要望の形式に変更 ▼▼▼
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
            // ▲▲▲【修正ここまで】▲▲▲
        });

        html += `</tbody>`;
    });

    html += `</table>`;
    outputContainer.innerHTML = html;
}

/**
 * APIから発注残リストを取得し、キャッシュと描画を行います。
 * (WASABI: backorder.js を TKR 用に修正)
 */
async function loadAndRenderBackorders() {
    outputContainer.innerHTML = '<p>読み込み中...</p>';
    try {
        const res = await fetch('/api/backorders');
        if (!res.ok) throw new Error('発注残リストの読み込みに失敗しました。');
        allBackorders = await res.json();
        
        searchKanaInput.value = '';
        searchWholesalerInput.value = '';

        renderBackorders(allBackorders);
    } catch (err) {
        outputContainer.innerHTML = `<p class="status-error">${err.message}</p>`;
    }
}

/**
 * 発注残ビューのイベントハンドラ
 * (WASABI: backorder.js を TKR 用に修正)
 */
async function handleBackorderEvents(e) {
    const target = e.target;

    // 個別削除ボタン
    if (target.classList.contains('delete-backorder-btn')) {
        const row = target.closest('tr');
        // ▼▼▼【修正】グループヘッダーから YYYYMMDDHHMMSS を取得 ▼▼▼
        const groupHeader = row.closest('tbody.backorder-group')?.querySelector('tr.group-header');
        const orderDateStr = groupHeader ? groupHeader.querySelector('.delete-backorder-group-btn').dataset.orderDate : '不明';
        // ▲▲▲【修正ここまで】▲▲▲

        // ▼▼▼【修正】確認メッセージの品名列インデックスを 2 に変更 ▼▼▼
        if (!confirm(`「${row.cells[2].textContent}」の発注残（発注: ${orderDateStr}）を削除しますか？`)) {
            return;
        }
        // ▲▲▲【修正ここまで】▲▲▲
        const payload = {
            id: parseInt(row.dataset.id, 10),
        };
        window.showLoading();
        try {
            const res = await fetch('/api/backorders/delete', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
            });
            const resData = await res.json();
            if (!res.ok) throw new Error(resData.message || '削除に失敗しました。');
            window.showNotification(resData.message, 'success');
            loadAndRenderBackorders(); // リストを再読み込み
        } catch (err) {
            window.showNotification(err.message, 'error');
        } finally {
            window.hideLoading();
        }
    }

    // 全選択チェックボックス
    if (target.id === 'bo-select-all-checkbox') {
        const isChecked = target.checked;
        document.querySelectorAll('.bo-select-checkbox').forEach(cb => cb.checked = isChecked);
    }

    // 選択項目の一括削除ボタン
    if (target.id === 'bo-bulk-delete-btn') {
        const checkedRows = document.querySelectorAll('.bo-select-checkbox:checked');
        if (checkedRows.length === 0) {
            window.showNotification('削除する項目が選択されていません。', 'error');
            return;
        }
        if (!confirm(`${checkedRows.length}件の発注残を削除します。よろしいですか？`)) {
            return;
        }

        const payload = Array.from(checkedRows).map(cb => {
            const row = cb.closest('tr');
            return {
                id: parseInt(row.dataset.id, 10),
            };
        });
        window.showLoading();
        try {
            const res = await fetch('/api/backorders/bulk_delete_by_id', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
            });
            const resData = await res.json();
            if (!res.ok) throw new Error(resData.message || '一括削除に失敗しました。');
            window.showNotification(resData.message, 'success');
            loadAndRenderBackorders(); // リストを再読み込み
        } catch (err) {
            window.showNotification(err.message, 'error');
        } finally {
            window.hideLoading();
        }
    }

    // 発注書（グループ）単位での一括削除ボタン
    if (target.classList.contains('delete-backorder-group-btn')) {
        const orderDate = target.dataset.orderDate;
        if (!orderDate) return;

        // ▼▼▼【修正】フォーマット済みのきれいな日付ではなく、YYYYMMDDHHMMSS をそのまま表示 ▼▼▼
        if (!confirm(`発注 [${orderDate}] の発注残をすべて削除します。よろしいですか？`)) {
            return;
        }
        // ▲▲▲【修正ここまで】▲▲▲

        const payload = {
            orderDate: orderDate,
        };
        
        window.showLoading();
        try {
            const res = await fetch('/api/backorders/bulk_delete_by_date', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
            });
            const resData = await res.json();
            if (!res.ok) throw new Error(resData.message || 'グループ削除に失敗しました。');
            
            window.showNotification(resData.message, 'success');
            loadAndRenderBackorders(); // リストを再読み込み
        } catch (err) {
            window.showNotification(err.message, 'error');
        } finally {
            window.hideLoading();
        }
    }


    // 棚卸調整ボタン
    if (target.classList.contains('adjust-inventory-btn')) {
        const yjCode = target.dataset.yjCode;
        if (!yjCode) return;

        // 棚卸調整ビューに移動するイベントを発火
        const event = new CustomEvent('loadInventoryAdjustment', {
            detail: { yjCode: yjCode },
            bubbles: true
        });
        document.dispatchEvent(event);

        // 棚卸調整タブをアクティブにする
        document.getElementById('inventoryAdjustmentViewBtn')?.click();
    }
}

export function initBackorderView() {
    view = document.getElementById('backorder-view');
    if (!view) return;
    outputContainer = document.getElementById('backorder-output-container');
    searchKanaInput = document.getElementById('bo-search-kana');
    searchWholesalerInput = document.getElementById('bo-search-wholesaler');
    searchBtn = document.getElementById('bo-search-btn');
    
    // 'show' イベントはTKRにはないので、app.js側で直接 loadAndRenderBackorders を呼ぶ
    view.addEventListener('show', loadAndRenderBackorders);

    outputContainer.addEventListener('click', handleBackorderEvents);
    
    searchBtn.addEventListener('click', filterAndRender);
    searchKanaInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') filterAndRender();
    });
    searchWholesalerInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') filterAndRender();
    });
}