// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\config_client_migration.js
import { handleFileUpload } from './utils.js';
import { refreshClientMap, refreshWholesalerMap } from './master_data.js';

let importClientsBtn, importClientsInput;
let migrationUploadResultContainer;

async function handleClientImport(event) {
    if (!confirm('【警告】得意先CSVをインポートします。\n・client_codeが一致する既存データは上書きされます。\nよろしいですか？')) {
        event.target.value = '';
        return;
    }

    await handleFileUpload(
        '/api/clients/import',
        event.target.files,
        importClientsInput,
        migrationUploadResultContainer,
        null, 
        '得意先マスタをインポート中...'
    );

    try {
        await refreshClientMap();
        await refreshWholesalerMap();
        window.showNotification('得意先・卸マスタのキャッシュを更新しました。', 'info');
    } catch (error) {
        console.error('Failed to refresh maps after import:', error);
        window.showNotification('CSVインポート後のキャッシュ更新に失敗しました。', 'error');
    }
}

export function initClientMigration() {
    importClientsBtn = document.getElementById('importClientsBtn');
    importClientsInput = document.getElementById('importClientsInput');
    migrationUploadResultContainer = document.getElementById('datUploadResultContainer');

    if (importClientsBtn && importClientsInput) {
        importClientsBtn.addEventListener('click', () => importClientsInput.click());
        importClientsInput.addEventListener('change', handleClientImport);
    }
}