// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\backorder_events.js
// (新規作成)
import { setBackorders, renderBackorders } from './backorder_ui.js';

let outputContainer, searchKanaInput, searchWholesalerInput;

/**
 * DOM要素をキャッシュする
 */
export function cacheDOMElements(elements) {
    outputContainer = elements.outputContainer;
    searchKanaInput = elements.searchKanaInput;
    searchWholesalerInput = elements.searchWholesalerInput;
}

/**
 * APIから発注残リストを取得し、キャッシュと描画を行います。
 */
export async function loadAndRenderBackorders() {
    outputContainer.innerHTML = '<p>読み込み中...</p>';
    try {
        const res = await fetch('/api/backorders');
        if (!res.ok) throw new Error('発注残リストの読み込みに失敗しました。');
        const data = await res.json();
        
        setBackorders(data); // UIモジュールの状態を更新

        searchKanaInput.value = '';
        searchWholesalerInput.value = '';

        renderBackorders(data); // UIモジュールの描画を呼び出し
    } catch (err) {
         outputContainer.innerHTML = `<p class="status-error">${err.message}</p>`;
    }
}

/**
 * 発注残ビューのイベントハンドラ
 */
export async function handleBackorderEvents(e) {
    const target = e.target;

    // 個別削除ボタン
    if (target.classList.contains('delete-backorder-btn')) {
        const row = target.closest('tr');
        const groupHeader = row.closest('tbody.backorder-group')?.querySelector('tr.group-header');
        const orderDateStr = groupHeader ? groupHeader.querySelector('.delete-backorder-group-btn').dataset.orderDate : '不明';

        if (!confirm(`「${row.cells[2].textContent}」の発注残（発注: ${orderDateStr}）を削除しますか？`)) {
            return;
        }
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

        if (!confirm(`発注 [${orderDate}] の発注残をすべて削除します。よろしいですか？`)) {
            return;
        }

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