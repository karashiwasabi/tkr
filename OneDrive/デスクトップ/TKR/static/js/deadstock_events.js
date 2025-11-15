// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\deadstock_events.js
// (新規作成)
import { renderDeadStockTable } from './deadstock_ui.js';

// DOM要素のキャッシュ
let startDateInput, endDateInput, searchBtn, resultContainer, excludeZeroStockCheckbox;
let csvDateInput, csvFileInput, csvUploadBtn;
let exportCsvBtn;

/**
 * メインの initDeadStockView からDOM要素を受け取り、キャッシュする
 */
export function cacheDOMElements(elements) {
    startDateInput = elements.startDateInput;
    endDateInput = elements.endDateInput;
    searchBtn = elements.searchBtn;
    resultContainer = elements.resultContainer;
    exportCsvBtn = elements.exportCsvBtn;
    excludeZeroStockCheckbox = elements.excludeZeroStockCheckbox;
    csvDateInput = elements.csvDateInput;
    csvFileInput = elements.csvFileInput;
    csvUploadBtn = elements.csvUploadBtn;
}

/**
 * 不動在庫リストを取得・描画する
 */
export async function fetchAndRenderDeadStock() {
    const startDate = startDateInput.value.replace(/-/g, '');
    const endDate = endDateInput.value.replace(/-/g, '');
    if (!startDate || !endDate) {
         window.showNotification('開始日と終了日を指定してください。', 'warning');
        return;
    }

    window.showLoading('不動在庫リストを集計中...');
    resultContainer.innerHTML = '<p>検索中...</p>';

    try {
        const params = new URLSearchParams({ 
            startDate, 
            endDate,
            excludeZeroStock: excludeZeroStockCheckbox.checked
        });
        const response = await fetch(`/api/deadstock/list?${params.toString()}`);

        if (!response.ok) {
             const errorText = await response.text();
            throw new Error(errorText || `サーバーエラー (HTTP ${response.status})`);
        }
        
         const data = await response.json();
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
 * CSVアップロード処理
 */
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
    formData.append('date', date.replace(/-/g, '')); 

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

        fetchAndRenderDeadStock();
    } catch (error) {
         console.error('Failed to upload dead stock CSV:', error);
        window.showNotification(`CSV登録エラー: ${error.message}`, 'error');
    } finally {
         if (csvFileInput) csvFileInput.value = '';
        window.hideLoading();
    }
}

/**
 * CSVエクスポート処理
 */
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

        const contentDisposition = response.headers.get('content-disposition');
        let filename = `不動在庫リスト_${startDate}-${endDate}.csv`;
        if (contentDisposition) {
            const filenameMatch = contentDisposition.match(/filename="(.+?)"/);
            if (filenameMatch && filenameMatch[1]) {
                 filename = filenameMatch[1];
            }
        }
         const blob = await response.blob();
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

/**
 * テーブル内のクリック（棚卸調整）を処理
 */
function handleTableClick(e) {
    if (e.target.classList.contains('adjust-inventory-btn')) {
        const yjCode = e.target.dataset.yjCode;
        if (!yjCode) {
            window.showNotification('YJコードが見つかりません。', 'error');
            return;
        }
        
        const inventoryBtn = document.getElementById('inventoryAdjustmentViewBtn');
        if (inventoryBtn) {
            inventoryBtn.click();
        } else {
            window.showNotification('棚卸調整ビューへの切り替えボタンが見つかりません。', 'error');
            return;
        }

        setTimeout(() => {
            document.dispatchEvent(new CustomEvent('loadInventoryAdjustment', {
                detail: { yjCode: yjCode }
            }));
        }, 100); 
    }
}

/**
 * すべてのイベントリスナーを登録する
 */
export function initDeadStockEventListeners() {
    if (searchBtn) {
        searchBtn.addEventListener('click', fetchAndRenderDeadStock);
    }
    if (excludeZeroStockCheckbox) {
        excludeZeroStockCheckbox.addEventListener('change', fetchAndRenderDeadStock);
    }
    if (csvUploadBtn) {
        csvUploadBtn.addEventListener('click', handleCsvUpload);
    }
    if (exportCsvBtn) {
        exportCsvBtn.addEventListener('click', handleCsvExport);
    }
    if (resultContainer) {
        resultContainer.addEventListener('click', handleTableClick);
    }
}