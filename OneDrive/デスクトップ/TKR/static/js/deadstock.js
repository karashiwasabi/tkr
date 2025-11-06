// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\deadstock.js
import { getLocalDateString } from './utils.js';

let startDateInput, endDateInput, searchBtn, resultContainer;
let csvDateInput, csvFileInput, csvUploadBtn;
// ▼▼▼【ここから修正】変数名を変更 ▼▼▼
let exportCsvBtn;
// ▲▲▲【修正ここまで】▲▲▲


/**
 * 期間のデフォルト値を設定（例: 90日前から本日）
 */
function setDefaultDates() {
    const endDate = new Date();
    const startDate = new Date();
    startDate.setDate(endDate.getDate() - 90);

    if (startDateInput) {
        startDateInput.value = getLocalDateString(startDate);
    }
    if (endDateInput) {
        endDateInput.value = getLocalDateString(endDate);
    }
    if (csvDateInput) {
        csvDateInput.value = getLocalDateString(endDate);
    }
}

/**
 * APIを叩いて不動在庫リストを取得・描画する
 */
async function fetchAndRenderDeadStock() {
    const startDate = startDateInput.value.replace(/-/g, '');
    const endDate = endDateInput.value.replace(/-/g, '');

    if (!startDate || !endDate) {
        window.showNotification('開始日と終了日を指定してください。', 'warning');
        return;
    }

    window.showLoading('不動在庫リストを集計中...');
    resultContainer.innerHTML = '<p>検索中...</p>';

    try {
        const params = new URLSearchParams({ startDate, endDate });
        const response = await fetch(`/api/deadstock/list?${params.toString()}`);

        if (!response.ok) {
            const errorText = await response.text(); // Get the plain text error
            throw new Error(errorText || `サーバーエラー (HTTP ${response.status})`);
        }
        
        const data = await response.json(); // Now it's safe to parse JSON

        if (data.errors && data.errors.length > 0) {
            window.showNotification(data.errors.join('\n'), 'error');
        }

        renderDeadStockTable(data.items);

    } catch (error) {
        console.error('Failed to fetch dead stock list:', error);
        resultContainer.innerHTML = `<p class="status-error">エラー: ${error.message}</p>`;
        window.showNotification(`エラー: ${error.message}`, 'error');
    } finally {
        window.hideLoading();
    }
}

/**
 * 取得した不動在庫アイテムをHTMLテーブルに変換して描画する
 */
function renderDeadStockTable(items) {
    if (!items || items.length === 0) {
        resultContainer.innerHTML = '<p>対象期間の不動在庫は見つかりませんでした。</p>';
        return;
    }

    const header = `
        <table id="deadstock-table" class="data-table">
            <thead>
                <tr>
                    <th class="col-ds-key">PackageKey</th>
                    <th class="col-ds-name">製品名</th>
                    <th class="col-ds-qty">現在庫(YJ)</th>
                    <th class="col-ds-details">棚卸明細 (JAN / 包装仕様 / 在庫数 / 単位 / 期限 / ロット)</th>
                    </tr>
            </thead>
            <tbody>
    `;

    const body = items.map(item => {
        let lotHtml = '棚卸履歴なし'; // デフォルト（在庫0の場合）
        if (item.lotDetails && item.lotDetails.length > 0) {
            lotHtml = '<ul class="lot-details-list">';
            // ▼▼▼【ここから修正】明細の表示内容を変更 ▼▼▼
            lotHtml += item.lotDetails.map(lot => {
                const janQty = (lot.JanQuantity || 0).toFixed(2);
                const janCode = lot.JanCode || '(JANなし)';
                const pkgSpec = lot.PackageSpec || '(仕様なし)';
                const lotNum = lot.LotNumber || '(ロットなし)';
                const expiry = lot.ExpiryDate || '(期限なし)';
                const unitName = lot.JanUnitName || ''; // 単位名
                
                return `<li>${janCode} / ${pkgSpec} / ${janQty} ${unitName} / ${expiry} / ${lotNum}</li>`;
            }).join('');
            // ▲▲▲【修正ここまで】▲▲▲
            lotHtml += '</ul>';
        } else if (item.stockQuantityYj > 0) {
            lotHtml = '<span class="status-error">在庫あり (明細なし)</span>';
        }

        const stockQty = (item.stockQuantityYj || 0).toFixed(2);
        return `
            <tr>
                <td class="left">${item.packageKey}</td>
                <td class="left">${item.productName || '(品名不明)'}</td>
                <td class="right">${stockQty}</td>
                <td class="left">${lotHtml}</td>
            </tr>
        `;
    }).join('');

    const footer = `
            </tbody>
        </table>
    `;

    resultContainer.innerHTML = header + body + footer;
}

async function handleCsvUpload() {
    const file = csvFileInput.files[0];
    const date = csvDateInput.value;

    if (!file) {
        window.showNotification('CSVファイルを選択してください。', 'warning');
        return;
    }
    if (!date) {
        window.showNotification('棚卸日を選択してください。', 'warning');
        return;
    }
    
    if (!confirm(`${date} の棚卸データとしてCSVを登録します。\n※この日付の既存棚卸データは、CSVに含まれる品目（YJコード）についてのみ上書きされます。\nよろしいですか？`)) {
        return;
    }

    const formData = new FormData();
    formData.append('file', file);
    formData.append('date', date.replace(/-/g, '')); // YYYYMMDD形式で送信

    window.showLoading('棚卸CSVを登録中...');
    
    try {
        const response = await fetch('/api/deadstock/upload', {
            method: 'POST',
            body: formData,
        });

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText || `サーバーエラー (HTTP ${response.status})`);
        }

        const result = await response.json();
        window.showNotification(result.message || '棚卸CSVを登録しました。', 'success');

        // 登録が成功したら、リストを再検索する
        fetchAndRenderDeadStock();

    } catch (error) {
        console.error('Failed to upload dead stock CSV:', error);
        window.showNotification(`CSV登録エラー: ${error.message}`, 'error');
    } finally {
        // ファイル入力をリセット
        if (csvFileInput) csvFileInput.value = '';
        window.hideLoading();
    }
}

// ▼▼▼【ここから修正】CSVエクスポート処理（IDと変数名を修正） ▼▼▼
async function handleCsvExport() {
    const startDate = startDateInput.value.replace(/-/g, '');
    const endDate = endDateInput.value.replace(/-/g, '');

    if (!startDate || !endDate) {
        window.showNotification('開始日と終了日を指定してください。', 'warning');
        return;
    }
    
    window.showLoading('CSVデータを生成中...');

    try {
        const params = new URLSearchParams({ startDate, endDate });
        const response = await fetch(`/api/deadstock/export?${params.toString()}`);

        if (!response.ok) {
            const errorText = await response.text();
            throw new Error(errorText || `サーバーエラー (HTTP ${response.status})`);
        }

        // ファイル名を設定
        const contentDisposition = response.headers.get('content-disposition');
        let filename = `不動在庫リスト_${startDate}-${endDate}.csv`;
        if (contentDisposition) {
            const filenameMatch = contentDisposition.match(/filename="(.+?)"/);
            if (filenameMatch && filenameMatch[1]) {
                filename = filenameMatch[1];
            }
        }

        // CSVデータをBlobとして取得
        const blob = await response.blob();
        
        // ダウンロードリンクを作成してクリック
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        a.remove();
        window.URL.revokeObjectURL(url);
        
        window.showNotification('CSVをエクスポートしました。', 'success');

    } catch (error) {
        console.error('Failed to export CSV:', error);
        window.showNotification(`CSVエクスポートエラー: ${error.message}`, 'error');
    } finally {
        window.hideLoading();
    }
}
// ▲▲▲【修正ここまで】▲▲▲


/**
 * 不動在庫ビューの初期化
 */
export function initDeadStockView() {
    // 期間検索
    startDateInput = document.getElementById('ds-start-date');
    endDateInput = document.getElementById('ds-end-date');
    searchBtn = document.getElementById('ds-search-btn');
    resultContainer = document.getElementById('deadstock-result-container');
    // ▼▼▼【ここから修正】IDと変数名を修正 ▼▼▼
    exportCsvBtn = document.getElementById('ds-export-csv-btn'); 
    // ▲▲▲【修正ここまで】▲▲▲

    // CSVアップロード
    csvDateInput = document.getElementById('ds-csv-date');
    csvFileInput = document.getElementById('ds-csv-file-input');
    csvUploadBtn = document.getElementById('ds-csv-upload-btn');

    if (searchBtn) {
        searchBtn.addEventListener('click', fetchAndRenderDeadStock);
    }
    
    if (csvUploadBtn) {
        csvUploadBtn.addEventListener('click', handleCsvUpload);
    }

    // ▼▼▼【ここから修正】変数名を修正 ▼▼▼
    if (exportCsvBtn) {
        exportCsvBtn.addEventListener('click', handleCsvExport);
    }
    // ▲▲▲【修正ここまで】▲▲▲
    
    // デフォルト日付を設定
    setDefaultDates();
    console.log("DeadStock View Initialized.");
}