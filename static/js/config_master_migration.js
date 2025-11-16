// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\config_master_migration.js
import { handleFileUpload } from './utils.js';

let exportAllMastersBtn, importAllMastersBtn, importAllMastersInput;
let importJcshmsBtn;
let migrationUploadResultContainer;

async function handleJcshmsReload() {
    if (!confirm('【警告】SOU/JCSHMS.CSV と SOU/JANCODE.CSV を再読み込みします。\nこの処理は時間がかかります。\n実行しますか？')) {
        return;
    }

    if (migrationUploadResultContainer) migrationUploadResultContainer.innerHTML = '<p>JCSHMSマスターを更新中...</p>';
    window.showLoading('JCSHMSマスターを更新中... (時間がかかります)');

    try {
        const response = await fetch('/api/jcshms/reload', {
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
        window.showNotification(result.message || 'JCSHMSの更新が完了しました。', 'success');

    } catch (error) {
        console.error('JCSHMS reload failed:', error);
        window.showNotification(`エラー: ${error.message}`, 'error');
        if (migrationUploadResultContainer) migrationUploadResultContainer.innerHTML = `<p style="color: red;">エラーが発生しました: ${error.message}</p>`;
    } finally {
        window.hideLoading();
    }
}

async function handleMasterImport(event) {
    if (!confirm('【警告】製品マスタCSVをインポートします。\n・product_codeが一致する既存マスタは上書きされます。\n・JCSHMSに存在する品目は、品名や薬価がJCSHMS優先でマージされます。\nよろしいですか？')) {
        event.target.value = '';
        return;
    }

    await handleFileUpload(
        '/api/masters/import/all',
        event.target.files,
        importAllMastersInput,
        migrationUploadResultContainer,
        null,
        '製品マスタをインポート中...'
    );
}

export function initMasterMigration() {
    exportAllMastersBtn = document.getElementById('exportAllMastersBtn');
    importAllMastersBtn = document.getElementById('importAllMastersBtn');
    importAllMastersInput = document.getElementById('importAllMastersInput');
    importJcshmsBtn = document.getElementById('importJcshmsBtn');
    migrationUploadResultContainer = document.getElementById('datUploadResultContainer');

    if (exportAllMastersBtn) {
        exportAllMastersBtn.addEventListener('click', () => {
            window.location.href = '/api/masters/export/all';
        });
    }

    if (importAllMastersBtn && importAllMastersInput) {
        importAllMastersBtn.addEventListener('click', () => importAllMastersInput.click());
        importAllMastersInput.addEventListener('change', handleMasterImport);
    }
    
    if (importJcshmsBtn) {
        importJcshmsBtn.addEventListener('click', handleJcshmsReload);
    }
}