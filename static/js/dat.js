// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\dat.js
import { showModal } from './search_modal.js';
import { handleFileUpload } from './utils.js';
import { renderTransactionTableHTML, renderEmptyTableHTML } from './common_table.js';

let datUploadBtn, datFileInput, uploadResultContainer, dataTable;
let datSearchBtn, barcodeInput;
let medicodeDownloadBtn; // 追加

// 検索処理
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

        if (dataTable && result.transactions && result.transactions.length > 0) {
             dataTable.innerHTML = renderTransactionTableHTML(result.transactions);
        } else if (dataTable) {
            dataTable.innerHTML = renderEmptyTableHTML();
        }
        
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

// 品目検索モーダルを開く
async function openProductSearchModal() {
    showModal(
        null, 
        (selectedProduct) => { 
            if (barcodeInput) {
                barcodeInput.value = selectedProduct.productCode;
            }
            handleDatSearch();
        }, 
        { 
            searchMode: '', 
        }
    );
}

// 初期表示リセット
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

// ▼▼▼ 追加: 自動受信ハンドラ ▼▼▼
async function handleMedicodeDownload() {
    if(!confirm("MEDICODE-Webに接続し、未受信のDATファイルをダウンロードします。\nブラウザが自動操作されます。\nよろしいですか？")) {
        return;
    }

    if (uploadResultContainer) uploadResultContainer.innerHTML = '<p>MEDICODEへ接続中...</p>';
    window.showLoading('MEDICODEからデータを自動受信中...\n(Chromeが起動します)');

    try {
        const response = await fetch('/api/automation/medicode/download', { method: 'POST' });
        const result = await response.json();

        if (!response.ok) {
            throw new Error(result.message || `サーバーエラー: ${response.status}`);
        }

        if (result.status === 'no_data') {
            window.showNotification(result.message, 'info');
            if (uploadResultContainer) uploadResultContainer.innerHTML = `<p>${result.message}</p>`;
        } else {
            window.showNotification(`${result.message}\n保存先: ${result.filePath}`, 'success');
            if (uploadResultContainer) uploadResultContainer.innerHTML = `<p class="status-success">ダウンロード完了: ${result.filePath}<br>「DATファイル選択」から取り込んでください。</p>`;
        }

    } catch (error) {
        console.error('Medicode automation failed:', error);
        window.showNotification(`自動受信エラー: ${error.message}`, 'error');
        if (uploadResultContainer) uploadResultContainer.innerHTML = `<p class="status-error">エラー: ${error.message}</p>`;
    } finally {
        window.hideLoading();
    }
}
// ▲▲▲ 追加ここまで ▲▲▲

// 初期化関数
export function initDatUpload() {
    datUploadBtn = document.getElementById('datUploadBtn');
    datFileInput = document.getElementById('datFileInput');
    // ▼▼▼ ボタン取得 ▼▼▼
    medicodeDownloadBtn = document.getElementById('medicodeDownloadBtn');

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
            );
        });
    } else {
        console.error('DAT Upload button or file input not found.');
    }

    // ▼▼▼ イベント登録 ▼▼▼
    if (medicodeDownloadBtn) {
        medicodeDownloadBtn.addEventListener('click', handleMedicodeDownload);
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