// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\usage.js

let usageFolderPathInput;
let saveUsageConfigBtn;
let manualImportUsageBtn;
let usageImportResultContainer;
let usageDataTableBody;

async function loadUsageConfig() {
    console.log("Loading usage config...");
    try {
        const response = await fetch('/api/usage/config');
        if (!response.ok) {
            throw new Error(`Failed to load config: ${response.statusText}`);
        }
        const config = await response.json();
        if (usageFolderPathInput && config.usageFolderPath) {
            usageFolderPathInput.value = config.usageFolderPath;
        }
        console.log("Usage config loaded:", config);
    } catch (error) {
        console.error("Error loading usage config:", error);
        window.showNotification(`設定の読み込みに失敗しました: ${error.message}`, 'error');
    }
}

async function saveUsageConfig() {
    const folderPath = usageFolderPathInput ? usageFolderPathInput.value : '';
    console.log("Saving usage config:", folderPath);
    window.showLoading('設定を保存中...');
    try {
        const response = await fetch('/api/usage/config', {
            method: 'POST',
            headers: {
                'Content-Type': 'application/json',
            },
            body: JSON.stringify({ usageFolderPath: folderPath }),
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
        console.log("Usage config saved.");
    } catch (error) {
        console.error("Error saving usage config:", error);
        window.showNotification(`設定の保存に失敗しました: ${error.message}`, 'error');
    } finally {
        window.hideLoading();
    }
}

async function triggerManualImport() {
    console.log("Triggering manual usage import...");
    if (usageImportResultContainer) usageImportResultContainer.textContent = '取り込み処理を実行中...';
    window.showLoading('処方ファイルを取り込み中...');
    try {
        const response = await fetch('/api/usage/import', {
            method: 'POST', // Trigger import via POST
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

        // Display summary message
        if (usageImportResultContainer) {
            usageImportResultContainer.textContent = result.message || '取り込み処理が完了しました。';
        }
        window.showNotification(result.message || '処方ファイルの取り込みが完了しました。', result.success ? 'success' : 'warning');

        // Render results table (assuming result.history is an array of import attempts)
        renderUsageImportHistory(result.history); // Implement this function if needed

        console.log("Manual import finished:", result);

    } catch (error) {
        console.error("Error during manual import:", error);
         if (usageImportResultContainer) {
             usageImportResultContainer.textContent = `エラー: ${error.message}`;
         }
        window.showNotification(`取り込みエラー: ${error.message}`, 'error');
    } finally {
        window.hideLoading();
    }
}

// Function to render import history (Placeholder - adapt based on actual API response)
function renderUsageImportHistory(history) {
    if (!usageDataTableBody) return;

    if (!history || history.length === 0) {
        usageDataTableBody.innerHTML = '<tr><td colspan="5">取り込み履歴はありません。</td></tr>';
        return;
    }

    let tableHtml = '';
    // Sort history newest first if needed: history.sort((a, b) => new Date(b.timestamp) - new Date(a.timestamp));
    history.forEach(entry => {
        const statusClass = entry.success ? 'status-success' : 'status-error';
        const statusText = entry.success ? '成功' : 'エラー';
        const timestamp = entry.timestamp ? new Date(entry.timestamp).toLocaleString() : '-';
        tableHtml += `
            <tr>
                <td class="center">${timestamp}</td>
                <td class="left">${entry.fileName || '-'}</td>
                <td class="center ${statusClass}">${statusText}</td>
                <td class="right">${entry.processedCount || 0}</td>
                <td class="left">${entry.error || ''}</td>
            </tr>
        `;
    });
    usageDataTableBody.innerHTML = tableHtml;
}


export function initUsageView() {
    usageFolderPathInput = document.getElementById('usageFolderPath');
    saveUsageConfigBtn = document.getElementById('saveUsageConfigBtn');
    manualImportUsageBtn = document.getElementById('manualImportUsageBtn');
    usageImportResultContainer = document.getElementById('usageImportResultContainer');
    const usageDataTable = document.getElementById('usageDataTable');
    usageDataTableBody = usageDataTable ? usageDataTable.querySelector('tbody') : null;

    if (saveUsageConfigBtn) {
        saveUsageConfigBtn.addEventListener('click', saveUsageConfig);
    }
    if (manualImportUsageBtn) {
        manualImportUsageBtn.addEventListener('click', triggerManualImport);
    }

    // Load initial config when the view is initialized (or becomes active)
    loadUsageConfig();
    // Initialize or load existing import history if available
    renderUsageImportHistory([]); // Start with an empty table

    console.log("Usage View Initialized.");
}