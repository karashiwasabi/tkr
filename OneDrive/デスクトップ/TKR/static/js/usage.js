// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\usage.js
// ▼▼▼【修正】handleFileUpload, renderTransactionTableHTML, renderEmptyTableHTML をインポート ▼▼▼
import { handleFileUpload } from './utils.js';
import { renderTransactionTableHTML, renderEmptyTableHTML } from './common_table.js';
// ▲▲▲【修正ここまで】▲▲▲

let usageUploadBtn, usageFileInput, uploadResultContainer, dataTable;

// ▼▼▼【削除】renderEmptyTable 関数を削除 (common_table.js に移管) ▼▼▼
/*
function renderEmptyTable(dataTable) {
    ...
}
*/
// ▲▲▲【削除ここまで】▲▲▲

// (handleUsageUpload は削除済み)

// ▼▼▼【修正】fetchAndRenderUsage が共通関数を呼ぶように変更 ▼▼▼
export function fetchAndRenderUsage() {
    if (dataTable) {
        dataTable.innerHTML = renderEmptyTableHTML();
    }
    if (uploadResultContainer) {
        uploadResultContainer.innerHTML = '<p>「USAGEファイル選択」ボタンを押してファイルを選んでください。</p>';
    }
}
// ▲▲▲【修正ここまで】▲▲▲

export function initUsageUpload() {
    usageUploadBtn = document.getElementById('usageUploadBtn');
    usageFileInput = document.getElementById('usageFileInput');
    uploadResultContainer = document.getElementById('usageUploadResultContainer');
    dataTable = document.getElementById('usageMainDataTable');

    if (usageUploadBtn && usageFileInput) {
        usageUploadBtn.addEventListener('click', () => {
            usageFileInput.click();
        });
        usageFileInput.addEventListener('change', (event) => {
            handleFileUpload(
                '/api/usage/upload',
                event.target.files,
                usageFileInput,
                uploadResultContainer,
                dataTable,
                'USAGEファイルを処理中...'
                // renderEmptyTable は handleFileUpload 内で共通関数が呼ばれる
            );
        });
    } else {
        console.error('USAGE Upload button or file input not found.');
    }
}