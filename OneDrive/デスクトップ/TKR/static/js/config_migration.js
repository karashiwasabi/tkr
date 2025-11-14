// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\config_migration.js
import { handleFileUpload } from './utils.js';
// ▼▼▼【ここに追加】 master_data.js からリフレッシュ関数をインポート ▼▼▼
import { refreshClientMap, refreshWholesalerMap } from './master_data.js';
// ▲▲▲【追加ここまで】▲▲▲

let importTkrStockBtn, importTkrStockInput;
let exportAllMastersBtn, importAllMastersBtn, importAllMastersInput;
// ▼▼▼【ここに追加】予製（一括）のボタン参照 ▼▼▼
let precompExportAllBtn, precompImportAllBtn, precompImportAllInput;
// ▲▲▲【追加ここまで】▲▲▲
// ▼▼▼【ここに追加】得意先インポートのボタン参照 ▼▼▼
let importClientsBtn, importClientsInput;
// ▲▲▲【追加ここまで】▲▲▲
// ▼▼▼【ここに追加】JCSHMS更新ボタンの参照 ▼▼▼
let importJcshmsBtn;
// ▲▲▲【追加ここまで】▲▲▲

let migrationUploadResultContainer;

export function initDataMigration() {
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

    // ▼▼▼【ここに追加】得意先インポートのボタンIDを取得 ▼▼▼
    importClientsBtn = document.getElementById('importClientsBtn');
    importClientsInput = document.getElementById('importClientsInput');
    // ▲▲▲【追加ここまで】▲▲▲

    // ▼▼▼【ここに追加】JCSHMS更新ボタンのIDを取得 ▼▼▼
    importJcshmsBtn = document.getElementById('importJcshmsBtn');
    // ▲▲▲【追加ここまで】▲▲▲

    migrationUploadResultContainer = document.getElementById('datUploadResultContainer');

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
        // ▼▼▼【修正】 async を追加 ▼▼▼
        importAllMastersInput.addEventListener('change', async (event) => {
             
    
            if (!confirm('【警告】製品マスタCSVをインポートします。\n・product_codeが一致する既存マスタは上書きされます。\n・JCSHMSに存在する品目は、品名や薬価がJCSHMS優先でマージされます。\nよろしいですか？')) {
                 
                 
                 event.target.value = ''; 
                 return;
 
            }

             // ▼▼▼【修正】 await を追加 ▼▼▼
            await handleFileUpload(
                 '/api/masters/import/all',
                 
                 event.target.files,
                 
                 importAllMastersInput,
                 migrationUploadResultContainer, 
                 
                 null, 
                 
                 '製品マスタをインポート中...'
            );
            // ▲▲▲【修正ここまで】▲▲▲
         });
    }

    // ▼▼▼【ここから修正】得意先インポートのイベントリスナー ▼▼▼
    
    if (importClientsBtn && importClientsInput) {
        importClientsBtn.addEventListener('click', () => importClientsInput.click());
        // ▼▼▼【修正】 async を追加 ▼▼▼
        importClientsInput.addEventListener('change', async (event) => {
    
             
            if (!confirm('【警告】得意先CSVをインポートします。\n・client_codeが一致する既存データは上書きされます。\nよろしいですか？')) {
 
                event.target.value = ''; 
                 return;
            
            }

            // ▼▼▼【修正】 await を追加 ▼▼▼
            await handleFileUpload(
                 '/api/clients/import',
                 event.target.files,
                 importClientsInput,
                 migrationUploadResultContainer, 
                 null, // 得意先インポートはテーブル描画なし
                 '得意先マスタをインポート中...'
            );
            // ▲▲▲【修正ここまで】▲▲▲

            // ▼▼▼【ここに追加】インポート完了後、JSのキャッシュを更新する ▼▼▼
            try {
                await refreshClientMap();
                await refreshWholesalerMap();
                window.showNotification('得意先・卸マスタのキャッシュを更新しました。', 'info');
            } catch (error) {
                console.error('Failed to refresh maps after import:', error);
                window.showNotification('CSVインポート後のキャッシュ更新に失敗しました。', 'error');
            }
            // ▲▲▲【追加ここまで】▲▲▲
        });
    }
    // ▲▲▲【修正ここまで】▲▲▲

    // ▼▼▼【ここから追加】JCSHMS更新のイベントリスナー ▼▼▼
    if (importJcshmsBtn) {
         
        importJcshmsBtn.addEventListener('click', async () => {
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
         });
    }
    // ▲▲▲【追加ここまで】▲▲▲

    // ▼▼▼【ここから追加】予製（一括）のイベントリスナー ▼▼▼
    if (precompExportAllBtn) {
        
        precompExportAllBtn.addEventListener('click', () => {
             window.location.href = '/api/precomp/export/all';
         });
    }

    
    if (precompImportAllBtn && precompImportAllInput) {
        precompImportAllBtn.addEventListener('click', () => precompImportAllInput.click());
        // ▼▼▼【修正】 async を追加 ▼▼▼
        precompImportAllInput.addEventListener('change', async (event) => {
    
             
       
        
             if (!confirm('【警告】予製CSVをインポートします。\n既存の予製データは、CSVに含まれる患者番号についてのみ洗い替え（上書き）されます。\nよろしいですか？')) {
                
 
                 event.target.value = ''; 
                 return;
            
             }

            // ▼▼▼【修正】 await を追加 ▼▼▼
            await handleFileUpload(
                 '/api/precomp/import/all',
                 
                 event.target.files,
                 precompImportAllInput,
 
                 migrationUploadResultContainer, 
                 
                 null, 
                 
                 '予製（一括）をインポート中...'
            );
            // ▲▲▲【修正ここまで】▲▲▲
             
        });
    }
    // ▲▲▲【追加ここまで】▲▲▲
}