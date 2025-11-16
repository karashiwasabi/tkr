// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\config_stock_migration.js
import { handleFileUpload } from './utils.js';

let importTkrStockBtn, importTkrStockInput;
let migrationUploadResultContainer;

async function handleTkrStockUpload(event) {
    const dateInput = document.getElementById('importTkrStockDate');
    const files = event.target.files;

    if (!files || files.length === 0) {
        return;
    }

    if (!dateInput || !dateInput.value) {
        window.showNotification('棚卸日(CSV適用日)を選択してください。', 'warning');
        event.target.value = '';
        return;
    }

    if (!confirm('【警告】TKR独自CSVを読み込み、現在の在庫をすべて洗い替えます。\nこの操作は取り消せません。\nよろしいですか？')) {
        event.target.value = '';
        return;
    }

    const formData = new FormData();
    formData.append('file', files[0]);
    formData.append('date', dateInput.value.replace(/-/g, ''));

    const apiEndpoint = '/api/stock/import/tkr';
    const loadingMessage = 'TKR独自CSV（洗い替え）を処理中...';

    if (migrationUploadResultContainer) migrationUploadResultContainer.innerHTML = '<p>ファイルをアップロード中...</p>';
    window.showLoading(loadingMessage);

    try {
        const response = await fetch(apiEndpoint, {
            method: 'POST',
            body: formData,
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
        window.showNotification(result.message || 'ファイルの処理が完了しました。', 'success');
    } catch (error) {
        console.error('Upload failed:', error);
        window.showNotification(`エラー: ${error.message}`, 'error');
        if (migrationUploadResultContainer) migrationUploadResultContainer.innerHTML = `<p style="color: red;">エラーが発生しました: ${error.message}</p>`;
    } finally {
        window.hideLoading();
        if (importTkrStockInput) importTkrStockInput.value = '';
        if (dateInput) dateInput.value = '';
    }
}

export function initStockMigration() {
    importTkrStockBtn = document.getElementById('importTkrStockBtn');
    importTkrStockInput = document.getElementById('importTkrStockInput');
    migrationUploadResultContainer = document.getElementById('datUploadResultContainer');

    if (importTkrStockBtn && importTkrStockInput) {
        importTkrStockBtn.addEventListener('click', () => importTkrStockInput.click());
        importTkrStockInput.addEventListener('change', handleTkrStockUpload);
    }
}