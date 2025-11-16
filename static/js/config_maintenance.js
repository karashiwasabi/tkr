// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\config_maintenance.js

let migrationUploadResultContainer;

async function handleClearOldInventory() {
    if (!confirm('【警告】最新の棚卸日を除く、それ以前のすべての棚卸履歴(flag=0)を削除します。\n在庫評価などで過去の日付を参照すると、計算結果が変わる可能性があります。\nこの操作は取り消せません。\n\n実行しますか？')) {
        return;
    }

    if (migrationUploadResultContainer) migrationUploadResultContainer.innerHTML = '<p>古い棚卸履歴を削除中...</p>';
    window.showLoading('古い棚卸履歴を削除中...');
    
    try {
        const response = await fetch('/api/inventory/clear_old', {
            method: 'POST',
        });
        
        const responseText = await response.text();
        let result;
        try {
            result = JSON.parse(responseText);
        } catch (jsonError) {
            if (!response.ok) {
                throw new Error(responseText || `サーバーエラー (HTTP ${response.status})`);
            }
            result = { message: responseText };
        }

        if (!response.ok) {
            throw new Error(result.message || `サーバーエラー (HTTP ${response.status})`);
        }
        
        if (migrationUploadResultContainer) {
            migrationUploadResultContainer.innerHTML = `<h3>${result.message || '処理が完了しました。'}</h3>`;
        }
        window.showNotification(result.message || '古い棚卸履歴を削除しました。', 'success');

    } catch (error) {
        console.error('Old inventory clear failed:', error);
        window.showNotification(`エラー: ${error.message}`, 'error');
        if (migrationUploadResultContainer) migrationUploadResultContainer.innerHTML = `<p style="color: red;">エラーが発生しました: ${error.message}</p>`;
    } finally {
        window.hideLoading();
    }
}

export function initMaintenance() {
    migrationUploadResultContainer = document.getElementById('datUploadResultContainer');
    const clearOldInventoryBtn = document.getElementById('clearOldInventoryBtn');
    
    if (clearOldInventoryBtn) {
        clearOldInventoryBtn.addEventListener('click', handleClearOldInventory);
    }
}