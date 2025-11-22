// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\backorder_events.js
import { showModal } from './search_modal.js';
import { filterAndRender, renderBackorders, setBackorders } from './backorder_ui.js';

let outputContainer, searchKanaInput;

export function cacheDOMElements(elements) {
    outputContainer = elements.outputContainer;
    searchKanaInput = elements.searchKanaInput;
    // 卸コード入力欄のキャッシュを削除
}

export async function loadAndRenderBackorders() {
    window.showLoading('発注残データを読み込み中...');
    try {
        const res = await fetch('/api/backorders');
        if (!res.ok) throw new Error('読み込みに失敗しました');
        const data = await res.json();
        
        setBackorders(data);
        renderBackorders(data);
        
        // フォームリセット: カナ検索のみクリア
        if(searchKanaInput) searchKanaInput.value = '';
        // 卸コードのリセット処理を削除 (これがエラーの原因でした)
        
    } catch (err) {
        if(outputContainer) outputContainer.innerHTML = `<p class="status-error">エラー: ${err.message}</p>`;
    } finally {
        window.hideLoading();
    }
}

export async function handleBackorderEvents(e) {
    const target = e.target;
    
    // 個別削除
    if (target.classList.contains('delete-backorder-btn')) {
        const row = target.closest('tr');
        const id = row.dataset.id;
        const name = row.querySelector('.col-bo-name').textContent;
        
        if (!confirm(`「${name}」の発注残を削除しますか？`)) return;
        
        window.showLoading('削除中...');
        try {
            const res = await fetch('/api/backorders/delete', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ id: parseInt(id, 10) })
            });
            if (!res.ok) throw new Error('削除に失敗しました');
            
            window.showNotification('削除しました', 'success');
            loadAndRenderBackorders();
        } catch(err) {
            window.showNotification(err.message, 'error');
        } finally {
            window.hideLoading();
        }
    }
    
    // グループ一括削除
    if (target.classList.contains('delete-backorder-group-btn')) {
        const date = target.dataset.orderDate;
        if (!confirm(`発注日: ${date} のデータをすべて削除しますか？`)) return;
        
        window.showLoading('一括削除中...');
        try {
            const res = await fetch('/api/backorders/bulk_delete_by_date', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ orderDate: date })
            });
            if (!res.ok) throw new Error('削除に失敗しました');
            
            window.showNotification('一括削除しました', 'success');
            loadAndRenderBackorders();
        } catch(err) {
            window.showNotification(err.message, 'error');
        } finally {
            window.hideLoading();
        }
    }
    
    // 棚卸調整へのリンク (機能が必要であれば)
    if (target.classList.contains('adjust-inventory-btn')) {
        // 将来的な実装のため、ボタンの定義のみ残しています
        // 現時点では動作しません
        window.showNotification('棚卸調整画面へ移動機能は未実装です', 'info');
    }

    // チェックボックスによる一括削除
    if (target.id === 'bo-bulk-delete-btn') {
        const checkboxes = document.querySelectorAll('.bo-select-checkbox:checked');
        if (checkboxes.length === 0) {
            window.showNotification('削除する項目を選択してください', 'warning');
            return;
        }
        
        if (!confirm(`選択した ${checkboxes.length} 件の発注残を削除しますか？`)) return;
        
        const ids = Array.from(checkboxes).map(cb => parseInt(cb.closest('tr').dataset.id, 10));
        
        window.showLoading('削除中...');
        try {
            const res = await fetch('/api/backorders/bulk_delete_by_id', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ ids: ids })
            });
            if (!res.ok) throw new Error('削除に失敗しました');
            
            window.showNotification('削除しました', 'success');
            loadAndRenderBackorders();
        } catch(err) {
            window.showNotification(err.message, 'error');
        } finally {
            window.hideLoading();
        }
    }
    
    // 全選択チェックボックス
    if (target.id === 'bo-select-all-checkbox') {
        const checkboxes = document.querySelectorAll('.bo-select-checkbox');
        checkboxes.forEach(cb => cb.checked = target.checked);
    }
}