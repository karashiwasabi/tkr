// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\deadstock.js
import { getLocalDateString } from './utils.js';

// ▼▼▼【ここに追加】excludeZeroStockCheckbox を追加 ▼▼▼
let startDateInput, endDateInput, searchBtn, resultContainer;
let csvDateInput, csvFileInput, csvUploadBtn;
let exportCsvBtn, excludeZeroStockCheckbox;
// ▲▲▲【追加ここまで】▲▲▲

function setDefaultDates() {
    const 
endDate = new Date();
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
        // ▼▼▼【ここに追加】excludeZeroStock パラメータを追加 ▼▼▼
        const params = new URLSearchParams({ 
            startDate, 
            endDate,
            excludeZeroStock: excludeZeroStockCheckbox.checked
        });
        // ▲▲▲【追加ここまで】▲▲▲
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
        resultContainer.innerHTML 
= `<p class="status-error">エラー: ${error.message}</p>`;
        window.showNotification(`エラー: ${error.message}`, 'error');
    } finally {
        window.hideLoading();
    }
}

function renderDeadStockTable(items) {
    if (!items || 
items.length === 0) {
        resultContainer.innerHTML = '<p>対象期間の不動在庫は見つかりませんでした。</p>';
        return;
    }

    const header = `
     
   <table id="deadstock-table" class="data-table">
            <thead>
          
      <tr>
                    <th 
class="col-ds-action">操作</th>
                    <th class="col-ds-key">PackageKey</th>
      
      
                     
<th class="col-ds-name">製品名</th>
            
              
      <th class="col-ds-qty">現在庫(JAN)</th>
                    <th class="col-ds-details">棚卸明細 (JAN / 包装仕様 / 在庫数 / 単位 / 期限 / ロット)</th>
               
     </tr>
      
                
 </thead>
            <tbody>
    `;
    const body = items.map(item => {
    
    // ▼▼▼【ここを修正】「棚卸履歴なし」を削除し、デフォルトを空('')にする ▼▼▼
        let lotHtml = ''; 
        // ▲▲▲【修正ここまで】▲▲▲
        if (item.lotDetails && item.lotDetails.length > 0) {
     
       lotHtml = '<ul class="lot-details-list">';
            lotHtml += item.lotDetails.map(lot => {
 
               const janQty = (lot.JanQuantity || 0).toFixed(2);
  
     
          
                 
const janCode = lot.JanCode || '(JANなし)';
                const pkgSpec = lot.PackageSpec || '(仕様なし)';
 
               const lotNum = lot.LotNumber || '(ロットなし)';
       
         const expiry = lot.ExpiryDate || '(期限なし)';
             
   const unitName 
 = lot.JanUnitName || ''; 
        
         
       
                return `<li>${janCode} / ${pkgSpec} 
/ ${janQty} ${unitName} / ${expiry} / ${lotNum}</li>`;
            }).join('');
         
   lotHtml += '</ul>';
        // ▼▼▼【ここを修正】stockQuantityYj -> stockQuantityJan ▼▼▼
        } else if (item.stockQuantityJan > 0) 
 {
       
     lotHtml = '<span class="status-error">在庫あり (明細なし)</span>';
        }
        // ▲▲▲【修正ここまで】▲▲▲

 
        // ▼▼▼【ここを修正】stockQuantityYj -> stockQuantityJan ▼▼▼
        const 
stockQty = (item.stockQuantityJan || 0).toFixed(2);
        // ▲▲▲【修正ここまで】▲▲▲
        
        const buttonHtml = item.yjCode ?
 `<button class="btn adjust-inventory-btn" data-yj-code="${item.yjCode}">棚卸調整</button>` : '';

        return `
          
  <tr>
                <td class="center col-ds-action">${buttonHtml}</td>
       
         <td class="left">${item.packageKey}</td>
                <td 
class="left">${item.productName || '(品名不明)'}</td>
                <td class="right col-ds-qty">${stockQty}</td>
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
    formData.append('date', date.replace(/-/g, '')); 

    window.showLoading('棚卸CSVを登録中...');
    try {
        const response = await fetch('/api/deadstock/upload', {
  
          method: 'POST',
            body: formData,
   
     });
        if (!response.ok) {
            const errorText = await response.text();
            throw new 
Error(errorText || `サーバーエラー (HTTP ${response.status})`);
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

async function handleCsvExport() {
    const startDate = startDateInput.value.replace(/-/g, '');
    const endDate = endDateInput.value.replace(/-/g, '');
    if (!startDate || !endDate) 
{
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
        let 
filename = `不動在庫リスト_${startDate}-${endDate}.csv`;
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

export function initDeadStockView() {
    startDateInput = document.getElementById('ds-start-date');
    endDateInput = document.getElementById('ds-end-date');
    searchBtn = document.getElementById('ds-search-btn');
    resultContainer = document.getElementById('deadstock-result-container');
    exportCsvBtn 
= document.getElementById('ds-export-csv-btn');
    // ▼▼▼【ここに追加】▼▼▼
    excludeZeroStockCheckbox = document.getElementById('ds-exclude-zero-stock');
    // ▲▲▲【追加ここまで】▲▲▲

    csvDateInput = document.getElementById('ds-csv-date');
    csvFileInput = document.getElementById('ds-csv-file-input');
    csvUploadBtn = document.getElementById('ds-csv-upload-btn');

    if (searchBtn) {
        searchBtn.addEventListener('click', fetchAndRenderDeadStock);
    }
    // ▼▼▼【ここに追加】チェックボックスのイベントリスナー ▼▼▼
    if (excludeZeroStockCheckbox) {
        excludeZeroStockCheckbox.addEventListener('change', fetchAndRenderDeadStock);
    }
    // ▲▲▲【追加ここまで】▲▲▲
    

    if (csvUploadBtn) {
        csvUploadBtn.addEventListener('click', handleCsvUpload);
    }

    if (exportCsvBtn) {
     
   exportCsvBtn.addEventListener('click', handleCsvExport);
    }

    if (resultContainer) {
        resultContainer.addEventListener('click', (e) => {
     
       if (e.target.classList.contains('adjust-inventory-btn')) {
                const yjCode 
= e.target.dataset.yjCode;
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
        });
    }
   
 
    setDefaultDates();
    console.log("DeadStock View Initialized.");
}