// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\config.js
import { handleFileUpload } from './utils.js';
import { refreshWholesalerMap, wholesalerMap } from './master_data.js';

let configSavePathBtn, datFolderPathInput, usageFolderPathInput;
let configSaveDaysBtn, calculationDaysInput;
let configAddWholesalerBtn, wholesalerCodeInput, wholesalerNameInput;
let wholesalerListTableBody;
let exportTkrStockBtn, importTkrStockBtn, importTkrStockInput;
// ▼▼▼【ここから追加】▼▼▼
let exportAllMastersBtn, importAllMastersBtn, importAllMastersInput;
// ▲▲▲【追加ここまで】▲▲▲

// (TKR独自CSV, 外部CSVインポートの結果表示コンテナは暫定的に DAT のものを借用)
// TODO: config_view.html 側に専用のコンテナを配置したほうが望ましい
let migrationUploadResultContainer;
/**
 * 卸一覧テーブルを描画します。
 */
function renderWholesalerList() {
    if (!wholesalerListTableBody) return;
    wholesalerListTableBody.innerHTML = '';
if (wholesalerMap.size === 0) {
        wholesalerListTableBody.innerHTML = '<tr><td colspan="3">登録されている卸はありません。</td></tr>';
        return;
}

    wholesalerMap.forEach((name, code) => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td class="col-config-code">${code}</td>
            <td class="left col-config-name">${name}</td>
            <td class="center col-config-action">
                <button class="btn delete-wholesaler-btn" data-code="${code}">削除</button>
            
</td>
        `;
        wholesalerListTableBody.appendChild(tr);
    });
}

/**
 * 設定ファイル (tkr_config.json) をAPIから読み込み、フォームに反映します。
 */
async function loadConfig() {
    try {
        const response = await fetch('/api/config');
if (!response.ok) {
            throw new Error('設定の読み込みに失敗しました。');
}
        const config = await response.json();
if (datFolderPathInput) {
            datFolderPathInput.value = config.datFolderPath || '';
}
        if (usageFolderPathInput) {
            usageFolderPathInput.value = config.usageFolderPath ||
'';
        }
        if (calculationDaysInput) {
            calculationDaysInput.value = config.calculationPeriodDays ||
90;
        }
    } catch (error) {
        window.showNotification(error.message, 'error');
}
}

/**
 * 卸マスタと設定の両方を読み込みます。
 */
export async function loadConfigAndWholesalers() {
    window.showLoading('設定情報を読み込み中...');
try {
        // 卸マスタを(再)読み込み
        await refreshWholesalerMap();
// 読み込んだマスタでテーブルを描画
        renderWholesalerList();
// 設定ファイルを読み込み
        await loadConfig();
} catch (error) {
        window.showNotification(error.message, 'error');
} finally {
        window.hideLoading();
}
}

/**
 * パス設定を保存します。
 */
async function handleSavePaths() {
    const newConfig = {
        datFolderPath: datFolderPathInput.value,
        usageFolderPath: usageFolderPathInput.value,
        calculationPeriodDays: parseInt(calculationDaysInput.value, 10) ||
90
    };
    await saveConfig(newConfig, 'パス設定を保存しました。');
}

/**
 * 集計期間設定を保存します。
 */
async function handleSaveDays() {
    const newConfig = {
        datFolderPath: datFolderPathInput.value,
        usageFolderPath: usageFolderPathInput.value,
        calculationPeriodDays: parseInt(calculationDaysInput.value, 10) ||
90
    };
    await saveConfig(newConfig, '集計期間を保存しました。');
}

/**
 * 共通の config 保存ロジック
 */
async function saveConfig(configData, successMessage) {
    window.showLoading('設定を保存中...');
try {
        const response = await fetch('/api/config', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(configData),
        });
const result = await response.json();
        if (!response.ok) {
            throw new Error(result.message || '設定の保存に失敗しました。');
}
        window.showNotification(successMessage, 'success');
        await loadConfig();
// 保存後に再読み込み
    } catch (error) {
        window.showNotification(error.message, 'error');
} finally {
        window.hideLoading();
}
}


/**
 * 新しい卸を追加します。
 */
async function handleAddWholesaler() {
    const code = wholesalerCodeInput.value.trim();
    const name = wholesalerNameInput.value.trim();
if (!code || !name) {
        window.showNotification('卸コードと卸名は必須です。', 'warning');
        return;
    }
    window.showLoading('卸を追加中...');
try {
        const response = await fetch('/api/wholesalers/create', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ code, name }),
        });
const result = await response.json();
        if (!response.ok) {
            throw new Error(result.message || '卸の追加に失敗しました。');
}
        window.showNotification(result.message, 'success');
        wholesalerCodeInput.value = '';
        wholesalerNameInput.value = '';
        await refreshWholesalerMap();
// 内部マップを更新
        renderWholesalerList();
// テーブルを再描画
    } catch (error) {
        window.showNotification(error.message, 'error');
} finally {
        window.hideLoading();
}
}

/**
 * 卸を削除します。
 */
async function handleDeleteWholesaler(code) {
    if (!code) return;
    if (!confirm(`卸コード「${code}」を削除しますか？`)) return;

    window.showLoading('卸を削除中...');
try {
        const response = await fetch(`/api/wholesalers/delete/${code}`, {
            method: 'DELETE',
        });
const result = await response.json();
        if (!response.ok) {
            throw new Error(result.message || '卸の削除に失敗しました。');
}
        window.showNotification(result.message, 'success');
        await refreshWholesalerMap();
// 内部マップを更新
        renderWholesalerList();
// テーブルを再描画
    } catch (error) {
        window.showNotification(error.message, 'error');
} finally {
        window.hideLoading();
}
}

/**
 * TKR独自CSVをエクスポートします。
 */
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
            // "filename*=UTF-8''..."
            const filenameMatch = contentDisposition.match(/filename\*=UTF-8''(.+)/);
if (filenameMatch && filenameMatch[1]) {
                filename = decodeURIComponent(filenameMatch[1]);
} else {
                // "filename=..."
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


/**
 * Configビューのイベントリスナーを初期化します。
 */
export function initConfigView() {
    configSavePathBtn = document.getElementById('configSavePathBtn');
    datFolderPathInput = document.getElementById('config-dat-folder-path');
    usageFolderPathInput = document.getElementById('config-usage-folder-path');
configSaveDaysBtn = document.getElementById('configSaveDaysBtn');
    calculationDaysInput = document.getElementById('config-calculation-days');

    configAddWholesalerBtn = document.getElementById('configAddWholesalerBtn');
    wholesalerCodeInput = document.getElementById('config-wholesaler-code');
    wholesalerNameInput = document.getElementById('config-wholesaler-name');
    wholesalerListTableBody = document.getElementById('wholesalerListTable')?.querySelector('tbody');
exportTkrStockBtn = document.getElementById('exportTkrStockBtn');
    importTkrStockBtn = document.getElementById('importTkrStockBtn');
    importTkrStockInput = document.getElementById('importTkrStockInput');
    
    // ▼▼▼【ここから追加】▼▼▼
    exportAllMastersBtn = document.getElementById('exportAllMastersBtn');
    importAllMastersBtn = document.getElementById('importAllMastersBtn');
    importAllMastersInput = document.getElementById('importAllMastersInput');
    // ▲▲▲【追加ここまで】▲▲▲

    // 結果表示コンテナ（DATビューのものを暫定利用）
    migrationUploadResultContainer = document.getElementById('datUploadResultContainer');

    // --- イベントリスナー設定 ---
    if (configSavePathBtn) {
        configSavePathBtn.addEventListener('click', handleSavePaths);
}
    if (configSaveDaysBtn) {
        configSaveDaysBtn.addEventListener('click', handleSaveDays);
}
    if (configAddWholesalerBtn) {
        configAddWholesalerBtn.addEventListener('click', handleAddWholesaler);
}
    if (wholesalerListTableBody) {
        wholesalerListTableBody.addEventListener('click', (e) => {
            if (e.target.classList.contains('delete-wholesaler-btn')) {
                handleDeleteWholesaler(e.target.dataset.code);
            }
        });
}

    // A. TKR独自CSVエクスポート
    if (exportTkrStockBtn) {
        exportTkrStockBtn.addEventListener('click', handleExportTkrStock);
}

    // B. TKR独自CSVインポート (洗い替え)
    if (importTkrStockBtn && importTkrStockInput) {
        importTkrStockBtn.addEventListener('click', () => importTkrStockInput.click());
importTkrStockInput.addEventListener('change', async (event) => {
            
            // 1. 日付入力欄を取得
            const dateInput = document.getElementById('importTkrStockDate');
            const files = event.target.files;

            // 2. バリデーション
            if (!files || files.length === 0) {
        
        return;
            }
            if (!dateInput || !dateInput.value) {
                window.showNotification('棚卸日(CSV適用日)を選択してください。', 'warning');
                event.target.value = ''; // Reset file input
                return;
         
   }
            if (!confirm('【警告】TKR独自CSVを読み込み、現在の在庫をすべて洗い替えます。\nこの操作は取り消せません。\nよろしいですか？')) {
                event.target.value = ''; // 中止
                return;
            }

            // 3. FormData を作成
            const formData = new FormData();
    
        formData.append('file', files[0]);
            formData.append('date', dateInput.value.replace(/-/g, '')); // YYYYMMDD形式で日付を追加

            const apiEndpoint = '/api/stock/import/tkr';
const loadingMessage = 'TKR独自CSV（洗い替え）を処理中...';

            // 4. handleFileUpload の代わりに手動で fetch を実行
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
                    migrationUploadResultContainer.innerHTML = `<h3>${result.message ||
'処理が完了しました。'}</h3>`;
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
// 日付もリセット
            }
        });
}
    
    // ▼▼▼【ここから追加】マスタ移行のイベントリスナー ▼▼▼

    // A. マスタCSVエクスポート
    if (exportAllMastersBtn) {
        exportAllMastersBtn.addEventListener('click', () => {
            window.location.href = '/api/masters/export/all';
        });
    }

    // B. マスタCSVインポート
    if (importAllMastersBtn && importAllMastersInput) {
        importAllMastersBtn.addEventListener('click', () => importAllMastersInput.click());
        importAllMastersInput.addEventListener('change', (event) => {
            
            if (!confirm('【警告】製品マスタCSVをインポートします。\n・product_codeが一致する既存マスタは上書きされます。\n・JCSHMSに存在する品目は、品名や薬価がJCSHMS優先でマージされます。\nよろしいですか？')) {
                event.target.value = ''; // 中止
                return;
            }

            // 汎用アップロード関数 (handleFileUpload) を呼び出す
            handleFileUpload(
                '/api/masters/import/all',
                event.target.files,
                importAllMastersInput,
                migrationUploadResultContainer, // 暫定的にDATの結果欄を使用
                null, // マスタ移行はテーブルを描画しない
                '製品マスタをインポート中...'
            );
        });
    }
    // ▲▲▲【追加ここまで】▲▲▲

    console.log("Config View Initialized.");
}