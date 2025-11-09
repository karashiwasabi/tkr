import { handleFileUpload } from './utils.js';

let exportTkrStockBtn, importTkrStockBtn, importTkrStockInput;
let exportAllMastersBtn, importAllMastersBtn, importAllMastersInput;
// ▼▼▼【ここに追加】予製（一括）のボタン参照 ▼▼▼
let precompExportAllBtn, precompImportAllBtn, precompImportAllInput;
// ▲▲▲【追加ここまで】▲▲▲

let migrationUploadResultContainer;

async function handleExportTkrStock() {
    window.showLoading('TKR在庫CSVをエクスポート中...');
    try {
        const response = await fetch('/api/stock/export/current');
        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText || `サーバーエラー (HTTP ${response.status})`);
        }

        const contentDisposition = response.headers.get('content-disposition');
        let filename = 'TKR在庫データ.csv';
        if (contentDisposition) {
            const filenameMatch = contentDisposition.match(/filename\*=UTF-8''(.+)/);
            if (filenameMatch && filenameMatch[1]) {
                filename = decodeURIComponent(filenameMatch[1]);
            } else {
                const filenameMatchFallback = contentDisposition.match(/filename="(.+?)"/);
                if (filenameMatchFallback && filenameMatchFallback[1]) {
                    filename = filenameMatchFallback[1];
                }
            }
        }
        
        const blob = await response.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        a.remove();
        window.URL.revokeObjectURL(url);
        
        window.showNotification('TKR在庫CSVをエクスポートしました。', 'success');
    } catch (error) {
        console.error('Failed to export TKR stock CSV:', error);
        window.showNotification(`CSVエクスポートエラー: ${error.message}`, 'error');
    } finally {
        window.hideLoading();
    }
}

export function initDataMigration() {
    exportTkrStockBtn = document.getElementById('exportTkrStockBtn');
    importTkrStockBtn = document.getElementById('importTkrStockBtn');
    importTkrStockInput = document.getElementById('importTkrStockInput');
    
    exportAllMastersBtn = document.getElementById('exportAllMastersBtn');
    importAllMastersBtn = document.getElementById('importAllMastersBtn');
    importAllMastersInput = document.getElementById('importAllMastersInput');

    // ▼▼▼【ここに追加】予製（一括）のボタンIDを取得 ▼▼▼
    precompExportAllBtn = document.getElementById('precomp-export-all-btn');
    precompImportAllBtn = document.getElementById('precomp-import-all-btn');
    precompImportAllInput = document.getElementById('precomp-import-all-input');
    // ▲▲▲【追加ここまで】▲▲▲

    migrationUploadResultContainer = document.getElementById('datUploadResultContainer');

    if (exportTkrStockBtn) {
        exportTkrStockBtn.addEventListener('click', handleExportTkrStock);
    }

    if (importTkrStockBtn && importTkrStockInput) {
        importTkrStockBtn.addEventListener('click', () => importTkrStockInput.click());
        importTkrStockInput.addEventListener('change', async (event) => {
            
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
        });
    }
    
    if (exportAllMastersBtn) {
        exportAllMastersBtn.addEventListener('click', () => {
            window.location.href = '/api/masters/export/all';
        });
    }

    if (importAllMastersBtn && importAllMastersInput) {
        importAllMastersBtn.addEventListener('click', () => importAllMastersInput.click());
        importAllMastersInput.addEventListener('change', (event) => {
            
            if (!confirm('【警告】製品マスタCSVをインポートします。\n・product_codeが一致する既存マスタは上書きされます。\n・JCSHMSに存在する品目は、品名や薬価がJCSHMS優先でマージされます。\nよろしいですか？')) {
                event.target.value = ''; 
                return;
            }

            handleFileUpload(
                '/api/masters/import/all',
                event.target.files,
                importAllMastersInput,
                migrationUploadResultContainer, 
                null, 
                '製品マスタをインポート中...'
            );
        });
    }

    // ▼▼▼【ここから追加】予製（一括）のイベントリスナー ▼▼▼
    if (precompExportAllBtn) {
        precompExportAllBtn.addEventListener('click', () => {
            window.location.href = '/api/precomp/export/all';
        });
    }

    if (precompImportAllBtn && precompImportAllInput) {
        precompImportAllBtn.addEventListener('click', () => precompImportAllInput.click());
        precompImportAllInput.addEventListener('change', (event) => {
            
            if (!confirm('【警告】予製CSVをインポートします。\n既存の予製データは、CSVに含まれる患者番号についてのみ洗い替え（上書き）されます。\nよろしいですか？')) {
                event.target.value = ''; 
                return;
            }

            handleFileUpload(
                '/api/precomp/import/all',
                event.target.files,
                precompImportAllInput,
                migrationUploadResultContainer, 
                null, 
                '予製（一括）をインポート中...'
            );
        });
    }
    // ▲▲▲【追加ここまで】▲▲▲
}