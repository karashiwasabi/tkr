// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\config_precomp_migration.js
import { handleFileUpload } from './utils.js';

let precompExportAllBtn, precompImportAllBtn, precompImportAllInput;
let migrationUploadResultContainer;

async function handlePrecompImport(event) {
    if (!confirm('【警告】予製CSVをインポートします。\n既存の予製データは、CSVに含まれる患者番号についてのみ洗い替え（上書き）されます。\nよろしいですか？')) {
        event.target.value = '';
        return;
    }

    await handleFileUpload(
        '/api/precomp/import/all',
        event.target.files,
        precompImportAllInput,
        migrationUploadResultContainer,
        null,
        '予製（一括）をインポート中...'
    );
}

export function initPrecompMigration() {
    precompExportAllBtn = document.getElementById('precomp-export-all-btn');
    precompImportAllBtn = document.getElementById('precomp-import-all-btn');
    precompImportAllInput = document.getElementById('precomp-import-all-input');
    migrationUploadResultContainer = document.getElementById('datUploadResultContainer');

    if (precompExportAllBtn) {
        precompExportAllBtn.addEventListener('click', () => {
            window.location.href = '/api/precomp/export/all';
        });
    }
    
    if (precompImportAllBtn && precompImportAllInput) {
        precompImportAllBtn.addEventListener('click', () => precompImportAllInput.click());
        precompImportAllInput.addEventListener('change', handlePrecompImport);
    }
}