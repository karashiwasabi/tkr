// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\config.js

let usageFolderPathInput;
let datFolderPathInput;
// ▼▼▼【ここに追加】▼▼▼
let calculationDaysInput;
let savePathBtn;
let saveDaysBtn; // 集計期間保存ボタン
// ▲▲▲【追加ここまで】▲▲▲

let wholesalerListTableBody;
let newWholesalerCodeInput, newWholesalerNameInput, addWholesalerBtn;

// --- 1. 設定 (パス・期間) ---

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
        // ▼▼▼【ここに追加】▼▼▼
        if (calculationDaysInput) {
            calculationDaysInput.value = config.calculationPeriodDays || 90;
        }
        // ▲▲▲【追加ここまで】▲▲▲

    } catch (error) {
        console.error("Error loading config:", error);
        window.showNotification(error.message, 'error');
    }
}

async function saveConfig() {
    const usagePath = usageFolderPathInput ? usageFolderPathInput.value : '';
    const datPath = datFolderPathInput ? datFolderPathInput.value : '';
    // ▼▼▼【ここに追加】▼▼▼
    const calcDays = calculationDaysInput ? parseInt(calculationDaysInput.value, 10) : 90;
    // ▲▲▲【追加ここまで】▲▲▲
    
    window.showLoading('設定を保存中...');
    try {
        const response = await fetch('/api/config', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            // ▼▼▼【修正】CalculationPeriodDays も保存する ▼▼▼
            body: JSON.stringify({ 
                usageFolderPath: usagePath,
                datFolderPath: datPath,
                calculationPeriodDays: calcDays 
            }),
            // ▲▲▲【修正ここまで】▲▲▲
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

// --- 2. 卸コード管理 (変更なし) ---

async function loadWholesalers() {
    if (!wholesalerListTableBody) return;
    wholesalerListTableBody.innerHTML = '<tr><td colspan="3">読み込み中...</td></tr>';
    try {
        const response = await fetch('/api/wholesalers/list');
        if (!response.ok) {
            throw new Error(`卸一覧の読み込みに失敗しました: ${response.statusText}`);
        }
        const wholesalers = await response.json();
        renderWholesalerTable(wholesalers);
    } catch (error) {
        console.error("Error loading wholesalers:", error);
        window.showNotification(error.message, 'error');
        wholesalerListTableBody.innerHTML = `<tr><td colspan="3" class="status-error">${error.message}</td></tr>`;
    }
}

function renderWholesalerTable(wholesalers) {
    if (!wholesalerListTableBody) return;
    if (!wholesalers || wholesalers.length === 0) {
        wholesalerListTableBody.innerHTML = '<tr><td colspan="3">登録されている卸コードはありません。</td></tr>';
        return;
    }
    
    let tableHtml = '';
    wholesalers.forEach(w => {
        tableHtml += `
            <tr data-code="${w.wholesalerCode}">
                <td class="left">${w.wholesalerCode}</td>
                <td class="left">${w.wholesalerName}</td>
                <td class="center">
                    <button class="delete-wholesaler-btn btn" data-code="${w.wholesalerCode}">削除</button>
                </td>
            </tr>
        `;
    });
    wholesalerListTableBody.innerHTML = tableHtml;
}

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
        
        loadWholesalers(); // テーブルを再読み込み
        
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
        loadWholesalers(); // テーブルを再読み込み
        
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

    // ▼▼▼【ここに追加】集計期間 ▼▼▼
    calculationDaysInput = document.getElementById('config-calculation-days');
    saveDaysBtn = document.getElementById('configSaveDaysBtn');
    // ▲▲▲【追加ここまで】▲▲▲
    
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
    // ▼▼▼【ここに追加】▼▼▼
    if (saveDaysBtn) {
        saveDaysBtn.addEventListener('click', saveConfig); // 同じ保存関数を呼ぶ
    }
    // ▲▲▲【追加ここまで】▲▲▲
    
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
    
    // 画面表示時にデータをロードする
    document.addEventListener('setActiveView', (event) => {
        if (event.detail.viewId === 'config-view') {
            loadConfig();
            loadWholesalers();
        }
    });
    console.log("Config View Initialized.");
}