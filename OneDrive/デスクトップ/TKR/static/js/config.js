// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\config.js
// ▼▼▼【修正】インポート先を変更 ▼▼▼
// import { fetchWholesalers } from './utils.js'; // 削除
import { wholesalerMap, refreshWholesalerMap } from './master_data.js'; // 変更
// ▲▲▲【修正ここまで】▲▲▲

let usageFolderPathInput;
let datFolderPathInput;
let calculationDaysInput;
let savePathBtn;
let saveDaysBtn;

let wholesalerListTableBody;
let newWholesalerCodeInput, newWholesalerNameInput, addWholesalerBtn;

// --- 1. 設定 (パス・期間) ---

export async function loadConfigAndWholesalers() {
    await loadConfig();
    await loadWholesalers(); // 変更
}

async function loadConfig() {
    try {
        const response = await fetch('/api/config');
        if (!response.ok) {
            throw new Error(`設定の読み込みに失敗しました: ${response.statusText}`);
        }
        const config = await response.json();
        if (usageFolderPathInput) {
            usageFolderPathInput.value = config.usageFolderPath || '';
        }
        if (datFolderPathInput) {
            datFolderPathInput.value = config.datFolderPath || '';
        }
        if (calculationDaysInput) {
            calculationDaysInput.value = config.calculationPeriodDays || 90;
        }
    } catch (error) {
        console.error("Error loading config:", error);
        window.showNotification(error.message, 'error');
    }
}

async function saveConfig() {
    const usagePath = usageFolderPathInput ? usageFolderPathInput.value : '';
    const datPath = datFolderPathInput ? datFolderPathInput.value : '';
    const calcDays = calculationDaysInput ? parseInt(calculationDaysInput.value, 10) : 90;
    
    window.showLoading('設定を保存中...');
    try {
        const response = await fetch('/api/config', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ 
                usageFolderPath: usagePath,
                datFolderPath: datPath,
                calculationPeriodDays: calcDays 
            }),
        });
        if (!response.ok) {
            let errorText = `サーバーエラー (HTTP ${response.status})`;
            try {
                const text = await response.text();
                errorText = text || errorText;
            } catch (e) {}
            throw new Error(errorText);
        }
        
        const result = await response.json();
        window.showNotification(result.message || '設定を保存しました。', 'success');
    } catch (error) {
        console.error("Error saving config:", error);
        window.showNotification(`設定の保存に失敗しました: ${error.message}`, 'error');
    } finally {
        window.hideLoading();
    }
}

// --- 2. 卸コード管理 ---

// ▼▼▼【修正】loadWholesalers をキャッシュマップ参照に変更 ▼▼▼
async function loadWholesalers() {
    if (!wholesalerListTableBody) return;
    wholesalerListTableBody.innerHTML = '<tr><td colspan="3">読み込み中...</td></tr>';
    
    try {
        // APIを叩く代わりに、ロード済みのマップを描画する
        renderWholesalerTable(wholesalerMap);
    } catch (error) {
        console.error("Error loading wholesalers:", error);
        wholesalerListTableBody.innerHTML = `<tr><td colspan="3" class="status-error">${error.message}</td></tr>`;
    }
}
// ▲▲▲【修正ここまで】▲▲▲

// ▼▼▼【修正】renderWholesalerTable を Map 対応に変更 ▼▼▼
function renderWholesalerTable(map) {
    if (!wholesalerListTableBody) return;
    if (!map || map.size === 0) {
        wholesalerListTableBody.innerHTML = '<tr><td colspan="3">登録されている卸コードはありません。</td></tr>';
        return;
    }
    
    let tableHtml = '';
    // Mapをイテレート
    map.forEach((name, code) => {
        tableHtml += `
            <tr data-code="${code}">
                <td class="left">${code}</td>
                <td class="left">${name}</td>
                <td class="center">
                    <button class="delete-wholesaler-btn btn" data-code="${code}">削除</button>
                </td>
            </tr>
        `;
    });
    wholesalerListTableBody.innerHTML = tableHtml;
}
// ▲▲▲【修正ここまで】▲▲▲

async function handleAddWholesaler() {
    const code = newWholesalerCodeInput ? newWholesalerCodeInput.value.trim() : '';
    const name = newWholesalerNameInput ? newWholesalerNameInput.value.trim() : '';

    if (!code || !name) {
        window.showNotification('卸コードと卸名の両方を入力してください。', 'warning');
        return;
    }
    
    window.showLoading('卸コードを追加中...');
    try {
        const response = await fetch('/api/wholesalers/create', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ code: code, name: name }),
        });
        if (!response.ok) {
            let errorText = `サーバーエラー (HTTP ${response.status})`;
            try {
                const text = await response.text();
                errorText = text || errorText;
            } catch (e) {}
            throw new Error(errorText);
        }

        const result = await response.json();
        window.showNotification(result.message || '追加しました。', 'success');
        if (newWholesalerCodeInput) newWholesalerCodeInput.value = '';
        if (newWholesalerNameInput) newWholesalerNameInput.value = '';
        
        // ▼▼▼【修正】マップを更新してから再描画 ▼▼▼
        await refreshWholesalerMap(); // グローバルマップを更新
        loadWholesalers(); // テーブルを再読み込み (キャッシュから)
        // ▲▲▲【修正ここまで】▲▲▲
        
    } catch (error) {
        console.error("Error adding wholesaler:", error);
        window.showNotification(`追加に失敗しました: ${error.message}`, 'error');
    } finally {
        window.hideLoading();
    }
}

async function handleDeleteWholesaler(code) {
    if (!code) return;
    if (!confirm(`卸コード「${code}」を削除しますか？`)) {
        return;
    }
    
    window.showLoading('卸コードを削除中...');
    try {
        const response = await fetch(`/api/wholesalers/delete/${code}`, {
            method: 'DELETE',
        });
        if (!response.ok) {
            let errorText = `サーバーエラー (HTTP ${response.status})`;
            try {
                const text = await response.text();
                errorText = text || errorText;
            } catch (e) {}
            throw new Error(errorText);
        }

        const result = await response.json();
        window.showNotification(result.message || '削除しました。', 'success');

        // ▼▼▼【修正】マップを更新してから再描画 ▼▼▼
        await refreshWholesalerMap(); // グローバルマップを更新
        loadWholesalers(); // テーブルを再読み込み (キャッシュから)
        // ▲▲▲【修正ここまで】▲▲▲
        
    } catch (error) {
        console.error("Error deleting wholesaler:", error);
        window.showNotification(`削除に失敗しました: ${error.message}`, 'error');
    } finally {
        window.hideLoading();
    }
}

// --- 3. 初期化 ---

export function initConfigView() {
    // パス設定
    usageFolderPathInput = document.getElementById('config-usage-folder-path');
    datFolderPathInput = document.getElementById('config-dat-folder-path');
    savePathBtn = document.getElementById('configSavePathBtn');

    // 集計期間
    calculationDaysInput = document.getElementById('config-calculation-days');
    saveDaysBtn = document.getElementById('configSaveDaysBtn');

    // 卸管理
    const wholesalerListTable = document.getElementById('wholesalerListTable');
    wholesalerListTableBody = wholesalerListTable ? wholesalerListTable.querySelector('tbody') : null;
    newWholesalerCodeInput = document.getElementById('config-wholesaler-code');
    newWholesalerNameInput = document.getElementById('config-wholesaler-name');
    addWholesalerBtn = document.getElementById('configAddWholesalerBtn');

    // イベントリスナー
    if (savePathBtn) {
        savePathBtn.addEventListener('click', saveConfig);
    }
    if (saveDaysBtn) {
        saveDaysBtn.addEventListener('click', saveConfig); // 同じ保存関数を呼ぶ
    }
    
    if (addWholesalerBtn) {
        addWholesalerBtn.addEventListener('click', handleAddWholesaler);
    }
    
    if (wholesalerListTable) {
        wholesalerListTable.addEventListener('click', (event) => {
            if (event.target.classList.contains('delete-wholesaler-btn')) {
                handleDeleteWholesaler(event.target.dataset.code);
            }
        });
    }
    
    // 画面表示時のロード処理は app.js の setActiveView に移管

    console.log("Config View Initialized.");
}