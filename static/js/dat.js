// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\dat.js
import { showModal } 
from './search_modal.js';
// ▼▼▼【修正】handleFileUpload, renderTransactionTableHTML, renderEmptyTableHTML をインポート ▼▼▼
import { hiraganaToKatakana, handleFileUpload } from './utils.js';
import { renderTransactionTableHTML, renderEmptyTableHTML } from './common_table.js';
// ▲▲▲【修正ここまで】▲▲▲

let datUploadBtn, datFileInput, uploadResultContainer, dataTable;
let datSearchBtn, 
barcodeInput;

// ▼▼▼【削除】renderEmptyTable 関数を削除 (common_table.js に移管) ▼▼▼
/*
function renderEmptyTable(dataTable) {
    return `
    <table class="data-table">
        <thead>
            <tr><th rowspan="2" class="col-action">－</th><th class="col-date">日付</th><th class="col-yj">YJ</th><th colspan="2" class="col-product">製品名</th><th class="col-count">個数</th><th class="col-yjqty">YJ数量</th><th class="col-yjpackqty">YJ包装数</th><th class="col-yjunit">YJ単位</th><th class="col-unitprice">単価</th><th class="col-expiry">期限</th><th class="col-wholesaler">卸</th><th class="col-line">行</th></tr>
            <tr><th class="col-flag">種別</th><th class="col-jan">JAN</th><th class="col-package">包装</th><th class="col-maker">メーカー</th><th class="col-form">剤型</th><th class="col-janqty">JAN数量</th><th class="col-janpackqty">JAN包装数</th><th class="col-janunit">JAN単位</th><th class="col-amount">金額</th><th class="col-lot">ロット</th><th class="col-receipt">伝票番号</th><th class="col-ma">MA</th></tr>
        </thead>
        <tbody>
            <tr><td colspan="13">登録されたデータはありません。</td></tr>
        </tbody>
    </table>
    `;
}
*/
// ▲▲▲【削除ここまで】▲▲▲

// (handleDatUpload は削除済み)

// ▼▼▼【修正】handleDatSearch を JSON レスポンス対応に変更 ▼▼▼
async function handleDatSearch() {
    const barcode 
= barcodeInput ? barcodeInput.value.trim() : '';
    if (!barcode) {
        window.showNotification('バーコードを入力してください。', 'warning');
        return;
    }

    if (uploadResultContainer) uploadResultContainer.innerHTML = '<p>検索中...</p>';
    if 
(dataTable) dataTable.innerHTML = '<thead></thead><tbody><tr><td colspan="13">検索中...</td></tr></tbody>';
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
            dataTable.innerHTML = 
renderTransactionTableHTML(result.transactions);
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

// ▼▼▼【修正】openProductSearchModal を修正 (モーダル内検索に移行) ▼▼▼
async function openProductSearchModal() {
    
    // 引数を削除し、初期結果なしでモーダルを開く
    showModal(
        null, // activeRowElement は不要
        (selectedProduct) => { 
            if (barcodeInput) {
                barcodeInput.value = selectedProduct.productCode;
            }
            handleDatSearch();
        }, 
        { 
            // searchMode: 'default' (ProductMasterのみ)
            searchMode: '', 
            // Datビューは採用済み品目のみを検索するが、採用済みでない品目（JCSHMS）が選択された場合は採用プロセスへ進めるため、allowAdoptedは設定しない（デフォルトの採用フローに乗せる）
        }
    );
}
// ▲▲▲【修正ここまで】▲▲▲

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