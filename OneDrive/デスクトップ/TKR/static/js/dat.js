// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\dat.js
import { showModal } from './search_modal.js';
// ▼▼▼【修正】handleFileUpload, renderTransactionTableHTML, renderEmptyTableHTML をインポート ▼▼▼
import { hiraganaToKatakana, handleFileUpload } from './utils.js';
import { renderTransactionTableHTML, renderEmptyTableHTML } from './common_table.js';
// ▲▲▲【修正ここまで】▲▲▲

let datUploadBtn, datFileInput, uploadResultContainer, dataTable;
let datSearchBtn, barcodeInput;

// ▼▼▼【削除】renderEmptyTable 関数を削除 (common_table.js に移管) ▼▼▼
/*
function renderEmptyTable(dataTable) {
    ...
}
*/
// ▲▲▲【削除ここまで】▲▲▲

// (handleDatUpload は削除済み)

// ▼▼▼【修正】handleDatSearch を JSON レスポンス対応に変更 ▼▼▼
async function handleDatSearch() {
    const barcode = barcodeInput ? barcodeInput.value.trim() : '';
    if (!barcode) {
        window.showNotification('バーコードを入力してください。', 'warning');
        return;
    }

    if (uploadResultContainer) uploadResultContainer.innerHTML = '<p>検索中...</p>';
    if (dataTable) dataTable.innerHTML = '<thead></thead><tbody><tr><td colspan="13">検索中...</td></tr></tbody>';
    window.showLoading('データを検索中...');
    try {
        const params = new URLSearchParams();
        params.append('barcode', barcode);
        const response = await fetch(`/api/dat/search?${params.toString()}`, {
            method: 'GET',
        });
        const result = await response.json();

        if (!response.ok) {
            throw new Error(result.message || `サーバーエラー (HTTP ${response.status})`);
        }

        if (uploadResultContainer) {
            uploadResultContainer.innerHTML = `<p>${result.message || '検索が完了しました。'}</p>`;
        }

        // ▼▼▼【修正】result.tableHTML -> result.transactions と共通関数呼び出しに変更 ▼▼▼
        if (dataTable && result.transactions && result.transactions.length > 0) {
            dataTable.innerHTML = renderTransactionTableHTML(result.transactions);
        } else if (dataTable) {
            dataTable.innerHTML = renderEmptyTableHTML();
        }
        // ▲▲▲【修正ここまで】▲▲▲
        
        window.showNotification(result.message || '検索が完了しました。', 'success');
    } catch (error) {
        console.error('Search failed:', error);
        if (uploadResultContainer) uploadResultContainer.innerHTML = `<p style="color: red;">エラーが発生しました: ${error.message}</p>`;
        window.showNotification(`エラー: ${error.message}`, 'error');
        if (dataTable) {
            dataTable.innerHTML = `<thead></thead><tbody><tr><td colspan="13" class="status-error">エラーが発生しました: ${error.message}</td></tr></tbody>`;
        }
    } finally {
        window.hideLoading();
        if (barcodeInput) {
            barcodeInput.value = '';
            barcodeInput.focus();
        }
    }
}
// ▲▲▲【修正ここまで】▲▲▲

async function openProductSearchModal() {
    const apiUrl = '/api/products/search_filtered';
    const kanaInput = document.getElementById('dat_search-kana');
    const genericInput = document.getElementById('dat_search-generic');
    const shelfInput = document.getElementById('dat_search-shelf');
    const selectedUsageRadio = document.querySelector('input[name="dat_usage_class"]:checked');
    
    const kanaName = kanaInput ? hiraganaToKatakana(kanaInput.value.trim()) : '';
    const genericName = genericInput ? genericInput.value.trim() : '';
    const shelfNumber = shelfInput ? shelfInput.value.trim() : '';
    const usageClass = selectedUsageRadio ? selectedUsageRadio.value : '';

    if (!usageClass) {
        window.showNotification('内外注区分を選択してください。', 'warning');
        return;
    }

    const params = new URLSearchParams();
    params.append('kanaName', kanaName);
    params.append('genericName', genericName);
    params.append('shelfNumber', shelfNumber);
    params.append('dosageForm', usageClass);

    window.showLoading('品目リストを検索中...');
    let products = [];
    try {
        const fullUrl = `${apiUrl}?${params.toString()}`;
        const res = await fetch(fullUrl);
        if (!res.ok) {
            throw new Error(`品目リストの取得に失敗しました: ${res.status}`);
        }
        products = await res.json();
    } catch (err) {
        window.hideLoading();
        window.showNotification(err.message, 'error');
        return;
    } finally {
        window.hideLoading();
    }

    showModal(
        null,
        (selectedProduct) => { 
            if (barcodeInput) {
                barcodeInput.value = selectedProduct.productCode;
            }
            handleDatSearch();
        }, 
        { 
            initialResults: products, 
        }
    );
}

// ▼▼▼【修正】fetchAndRenderDat が共通関数を呼ぶように変更 ▼▼▼
export function fetchAndRenderDat() {
    if (dataTable) {
        dataTable.innerHTML = renderEmptyTableHTML();
    }
    if (uploadResultContainer) {
        uploadResultContainer.innerHTML = '<p>「DATファイル選択」ボタンを押してファイルを選んでください。</p>';
    }
    if (barcodeInput) {
        barcodeInput.value = '';
    }
}
// ▲▲▲【修正ここまで】▲▲▲

export function initDatUpload() {
    datUploadBtn = document.getElementById('datUploadBtn');
    datFileInput = document.getElementById('datFileInput');
    uploadResultContainer = document.getElementById('datUploadResultContainer');
    dataTable = document.getElementById('datMainDataTable');
    datSearchBtn = document.getElementById('datOpenSearchModalBtn');
    barcodeInput = document.getElementById('dat-search-barcode');
    const barcodeForm = document.getElementById('dat-barcode-form');

    if (datUploadBtn && datFileInput) {
        datUploadBtn.addEventListener('click', () => {
            datFileInput.click();
        });
        datFileInput.addEventListener('change', (event) => {
            handleFileUpload(
                '/api/dat/upload',
                event.target.files,
                datFileInput,
                uploadResultContainer,
                dataTable,
                'DATファイルを処理中...'
                // renderEmptyTable は handleFileUpload 内で共通関数が呼ばれる
            );
        });
    } else {
        console.error('DAT Upload button or file input not found.');
    }

    if (datSearchBtn) {
        datSearchBtn.addEventListener('click', () => {
            openProductSearchModal(); 
        });
    } else {
        console.error('DAT Search button (open modal) not found.');
    }
    if (barcodeForm && barcodeInput) {
        const handleScanSubmit = (event) => {
            event.preventDefault();
            handleDatSearch();
        };
        barcodeForm.addEventListener('submit', handleScanSubmit);
    } else {
        console.error('DAT barcode form or input field not found.');
    }
}