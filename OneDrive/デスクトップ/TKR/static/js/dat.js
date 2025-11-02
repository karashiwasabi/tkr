// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\dat.js
import { showModal } from './search_modal.js';
import { hiraganaToKatakana } from './utils.js';

// ▼▼▼【ここから追加】グローバル変数化 ▼▼▼
let datUploadBtn, datFileInput, uploadResultContainer, dataTable;
let datSearchBtn, barcodeInput;
// ▲▲▲【追加ここまで】▲▲▲

function renderEmptyTable(dataTable) {
    if (!dataTable) return;
    const columnCount = 13;
    dataTable.innerHTML = `
    <thead>
        <tr>
            <th rowspan="2" class="col-action">－</th>
            <th class="col-date">日付</th>
            <th class="col-yj">YJ</th>
            <th colspan="2" class="col-product">製品名</th>
            <th class="col-count">個数</th>
            <th class="col-yjqty">YJ数量</th>
      
           <th class="col-yjpackqty">YJ包装数</th>
            <th class="col-yjunit">YJ単位</th>
            <th class="col-unitprice">単価</th>
            <th class="col-expiry">期限</th>
            <th class="col-wholesaler">卸</th>
            <th class="col-line">行</th>
        </tr>
        <tr>
            
 <th class="col-flag">種別</th>
            <th class="col-jan">JAN</th>
            <th class="col-package">包装</th>
            <th class="col-maker">メーカー</th>
            <th class="col-form">剤型</th>
            <th class="col-janqty">JAN数量</th>
            <th class="col-janpackqty">JAN包装数</th>
            <th class="col-janunit">JAN単位</th>
        
     <th class="col-amount">金額</th>
            <th class="col-lot">ロット</th>
            <th class="col-receipt">伝票番号</th>
            <th class="col-ma">MA</th>
        </tr>
    </thead>
    <tbody>
        <tr><td colspan="${columnCount}">登録されたデータはありません。</td></tr>
    </tbody>
    `;
}

async function handleDatUpload(files, datFileInput, uploadResultContainer, dataTable) {
    if (!files || files.length === 0) {
        return;
    }

    if (uploadResultContainer) uploadResultContainer.innerHTML = '<p>ファイルをアップロード中...</p>';
    if (dataTable) dataTable.innerHTML = '<thead></thead><tbody><tr><td colspan="13">処理中...</td></tr></tbody>';
    window.showLoading('DATファイルを処理中...');
    const formData = new FormData();
    for (const file of files) {
        formData.append('file', file);
    }

    try {
        const response = await fetch('/api/dat/upload', {
            method: 'POST',
            body: formData,
        });
        const result = await response.json(); 

        if (!response.ok) {
            throw new Error(result.message || `サーバーエラー (HTTP ${response.status})`);
        }

        let summaryHtml = `<h3>${result.message || '処理が完了しました。'}</h3>`;
        if (result.results && Array.isArray(result.results)) {
            summaryHtml += '<ul>';
            result.results.forEach(fileResult => {
                const statusClass = fileResult.success ? 'status-success' : 'status-error';
                const statusText = fileResult.success ? '成功' : 'エラー';
                const errorDetail = fileResult.error ? `: ${fileResult.error}` : '';
                const parsed = fileResult.records_parsed || 0;
       
                 const inserted = fileResult.records_inserted || 0;

                summaryHtml += `<li><strong>${fileResult.filename}:</strong> `;
                summaryHtml += `<span class="${statusClass}">${statusText}</span> (パース: ${parsed}件, 登録: ${inserted}件)${errorDetail}`;
                summaryHtml += '</li>';
            });
            summaryHtml += '</ul>';
        }
        if (uploadResultContainer) uploadResultContainer.innerHTML = summaryHtml;
        if (dataTable && result.tableHTML != null) { 
            dataTable.innerHTML = result.tableHTML;
        } else if (dataTable) {
            renderEmptyTable(dataTable);
        }
        
        window.showNotification(result.message || 'DATファイルの処理が完了しました。', 'success');
    } catch (error) {
        console.error('Upload failed:', error);
        if (uploadResultContainer) uploadResultContainer.innerHTML = `<p style="color: red;">エラーが発生しました: ${error.message}</p>`;
        window.showNotification(`エラー: ${error.message}`, 'error');
        if (dataTable) {
            dataTable.innerHTML = `<thead></thead><tbody><tr><td colspan="13" class="status-error">エラーが発生しました: ${error.message}</td></tr></tbody>`;
        }
    } finally {
        window.hideLoading();
        if (datFileInput) datFileInput.value = '';
    }
}

// ▼▼▼【ここから修正】引数を削除し、グローバル変数を参照する ▼▼▼
async function handleDatSearch() {
    const barcode = barcodeInput ? barcodeInput.value.trim() : '';
    if (!barcode) {
// ▼▼▼【修正ここまで】▲▲▲
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
            uploadResultContainer.innerHTML = `<p>${result.message ||
'検索が完了しました。'}</p>`;
        }

        if (dataTable && result.tableHTML != null) {
            dataTable.innerHTML = result.tableHTML;
        } else if (dataTable) {
            renderEmptyTable(dataTable);
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
// ▼▼▼【ここから修正】引数を削除し、グローバル変数を参照する ▼▼▼
        if (barcodeInput) {
            barcodeInput.value = '';
            barcodeInput.focus();
        }
// ▼▼▼【修正ここまで】▲▲▲
    }
}

// ▼▼▼【ここから追加】品目検索モーダルを開く関数 ▼▼▼
async function openProductSearchModal() {
    const apiUrl = '/api/products/search_filtered';
// ▼▼▼【修正】IDのハイフン(-)をアンダースコア(_)に変更 ▼▼▼
    const kanaInput = document.getElementById('dat_search-kana');
    const genericInput = document.getElementById('dat_search-generic');
    const shelfInput = document.getElementById('dat_search-shelf');
    // ▲▲▲【修正ここまで】▲▲▲
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

    // モーダルを表示
    showModal(
        null, // DATビューでは特定の行コンテキストは不要
        (selectedProduct) => { 
            // モーダルで品目が選択されたときのコールバック
            if (barcodeInput) {
                barcodeInput.value = selectedProduct.productCode; // JANコードをGS1入力欄に設定
            }
            // GS1入力欄を使って（JANコードのみで）検索を実行
            handleDatSearch(); // 引数なし
        }, 
        { 
            initialResults: products, 
        }
    );
}
// ▲▲▲【追加ここまで】▲▲▲


// ▼▼▼【ここから追加】app.jsから呼ばれる関数 ▼▼▼
export function fetchAndRenderDat() {
    // DAT画面は表示されるたびに空にする
    renderEmptyTable(dataTable);
    if (uploadResultContainer) {
        uploadResultContainer.innerHTML = '<p>「DATファイル選択」ボタンを押してファイルを選んでください。</p>';
    }
    if (barcodeInput) {
        barcodeInput.value = '';
    }
}
// ▲▲▲【追加ここまで】▲▲▲

export function initDatUpload() {
    datUploadBtn = document.getElementById('datUploadBtn');
    datFileInput = document.getElementById('datFileInput');
    uploadResultContainer = document.getElementById('datUploadResultContainer');
    dataTable = document.getElementById('datMainDataTable');

    datSearchBtn = document.getElementById('datOpenSearchModalBtn');
    barcodeInput = document.getElementById('dat-search-barcode');
    // ▼▼▼【ここから追加】フォームのDOMを取得 ▼▼▼
    const barcodeForm = document.getElementById('dat-barcode-form');
    // ▲▲▲【追加ここまで】▲▲▲

    if (datUploadBtn && datFileInput) {
        datUploadBtn.addEventListener('click', () => {
            datFileInput.click();
        });
        datFileInput.addEventListener('change', (event) => {
            handleDatUpload(event.target.files, datFileInput, uploadResultContainer, dataTable);
        });
    } else {
        console.error('DAT Upload button or file input not found.');
    }

    if (datSearchBtn) {
        // ▼▼▼【修正】品目検索ボタンのリスナー (barcodeInputのチェックを削除) ▼▼▼
        datSearchBtn.addEventListener('click', () => {
            openProductSearchModal(); 
        });
        // ▲▲▲【修正ここまで】▲▲▲
    } else {
        console.error('DAT Search button (open modal) not found.');
    }

    // ▼▼▼【ここから修正】GS1スキャン入力のイベントリスナーを 'submit' に変更 ▼▼▼
    if (barcodeForm && barcodeInput) {
        const handleScanSubmit = (event) => {
            event.preventDefault(); // ページリロードを防ぐ
            handleDatSearch(); // 検索実行
        };
        barcodeForm.addEventListener('submit', handleScanSubmit);

        // 既存の 'keypress' リスナーは削除
        // const handleKeyPress = (event) => { ... };
        // barcodeInput.addEventListener('keypress', handleKeyPress);

    } else {
        console.error('DAT barcode form or input field not found.');
    }
    // ▲▲▲【修正ここまで】▲▲▲
}