// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\usage.js

// ▼▼▼【ここから追加】グローバル変数化 ▼▼▼
let usageUploadBtn, usageFileInput, uploadResultContainer, dataTable;
// ▲▲▲【追加ここまで】▲▲▲

// dat.js の renderEmptyTable をコピー
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

// dat.js の handleDatUpload をコピーし、USAGE用に変更
async function handleUsageUpload(files, usageFileInput, uploadResultContainer, dataTable) {
    if (!files || files.length === 0) {
        return;
    }

    if (uploadResultContainer) uploadResultContainer.innerHTML = '<p>USAGEファイルをアップロード中...</p>';
    if (dataTable) dataTable.innerHTML = '<thead></thead><tbody><tr><td colspan="13">処理中...</td></tr></tbody>';
    window.showLoading('USAGEファイルを処理中...');

    const formData = new FormData();
    for (const file of files) {
        formData.append('file', file);
    }

    try {
        const response = await fetch('/api/usage/upload', {
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
        
        window.showNotification(result.message || 'USAGEファイルの処理が完了しました。', 'success');

    } catch (error) {
        console.error('Upload failed:', error);
        if (uploadResultContainer) uploadResultContainer.innerHTML = `<p style="color: red;">エラーが発生しました: ${error.message}</p>`;
        window.showNotification(`エラー: ${error.message}`, 'error');
        if (dataTable) {
            dataTable.innerHTML = `<thead></thead><tbody><tr><td colspan="13" class="status-error">エラーが発生しました: ${error.message}</td></tr></tbody>`;
        }
    } finally {
        window.hideLoading();
        if (usageFileInput) usageFileInput.value = '';
    }
}

// ▼▼▼【ここから追加】app.jsから呼ばれる関数 ▼▼▼
export function fetchAndRenderUsage() {
    // USAGE画面も表示されるたびに空にする
    renderEmptyTable(dataTable);
    if (uploadResultContainer) {
        uploadResultContainer.innerHTML = '<p>「USAGEファイル選択」ボタンを押してファイルを選んでください。</p>';
    }
}
// ▲▲▲【追加ここまで】▲▲▲

// dat.js の initDatUpload をコピーし、USAGE用に変更
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
            handleUsageUpload(event.target.files, usageFileInput, uploadResultContainer, dataTable);
        });
    } else {
        console.error('USAGE Upload button or file input not found.');
    }

     // ▼▼▼【削除】起動時の描画処理は fetchAndRenderUsage に移動 ▼▼▼
     // renderEmptyTable(dataTable);
     // if (uploadResultContainer) {
     //    uploadResultContainer.innerHTML = '<p>「USAGEファイル選択」ボタンを押してファイルを選んでください。</p>';
     // }
     // ▲▲▲【削除ここまで】▲▲▲
}