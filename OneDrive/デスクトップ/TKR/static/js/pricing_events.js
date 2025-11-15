// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\pricing_events.js
// (新規作成)
import { 
    setFullPricingData, 
    setOrderedWholesalers, 
    applyFiltersAndRender, 
    getFullPricingData 
} from './pricing_ui.js';

// DOM要素
let wholesalerSelect, exportBtn, uploadInput, bulkUpdateBtn, exportUnregisteredBtn;
let directImportBtn, directImportInput;

/**
 * マスター更新APIを呼び出します。
 */
async function sendUpdatePayload(payload) {
    if (payload.length === 0) {
        window.showNotification('更新するデータがありません。', 'error');
        return;
    }
    window.showLoading('納入価と採用卸をマスターに保存中...');
    try {
        const res = await fetch('/api/pricing/update', {
             method: 'POST',
            headers: { 'Content-Type': 'application/json' },
             body: JSON.stringify(payload),
        });
        const resData = await res.json();
        if (!res.ok) {
            throw new Error(resData.message || 'マスターの更新に失敗しました。');
        }
        window.showNotification(resData.message, 'success');
        
        // ローカルキャッシュも更新
        const currentData = getFullPricingData();
        payload.forEach(update => {
            const product = currentData.find(p => p.productCode === update.productCode);
            if (product) {
                 product.purchasePrice = update.newPrice;
                product.supplierWholesale = update.newWholesaler;
            }
        });
        setFullPricingData(currentData); // 更新したデータをUIモジュールにセット
        applyFiltersAndRender();
    } catch (err) {
         window.showNotification(err.message, 'error');
    } finally {
        window.hideLoading();
    }
}

/**
 * 「一括更新」ボタンのハンドラ
 */
async function handleBulkUpdate() {
    if (!confirm('表示されている全ての行の内容でマスターデータを一括更新します。よろしいですか？')) {
        return;
    }
    const rows = document.querySelectorAll('#pricing-output-container tbody tr');
    const payload = [];
    rows.forEach(row => {
         const productCode = row.dataset.productCode;
        const supplierSelect = row.querySelector('.supplier-select');
         const manualPriceInput = row.querySelector('.manual-price-input');
        const selectedWholesalerCode = supplierSelect.value;
        const price = parseFloat(manualPriceInput.value);
 
         if (productCode && selectedWholesalerCode && !isNaN(price)) {
            payload.push({ productCode, newPrice: price, newWholesaler: selectedWholesalerCode });
        } else if (productCode && !selectedWholesalerCode) {
           payload.push({ productCode, newPrice: 0, newWholesaler: '' });
        }
    });
    await sendUpdatePayload(payload);
}

/**
 * 見積CSVアップロードハンドラ
 */
async function handleUpload() {
    const files = Array.from(uploadInput.files);
    if (files.length === 0) return;
    window.showLoading('見積CSVをアップロード・解析中...');
    try {
        const formData = new FormData();
        const wholesalerNames = [];
        const processedFiles = [];

        files.forEach(file => {
            const match = file.name.match(/^(\d+)_/);
             const priority = match ? parseInt(match[1], 10) : Infinity;
            
             const nameParts = file.name.replace(/^\d+_/, '').split('_');
           
             if (nameParts.length > 1) {
                 processedFiles.push({ file: file, priority: priority, wholesalerName: nameParts[1] });
            } else {
                 window.showNotification(`ファイル名から卸名を抽出できませんでした: ${file.name}`, 'error');
             }
        });
        
        processedFiles.sort((a, b) => a.priority - b.priority);
        
        processedFiles.forEach(item => {
             formData.append('files', item.file);
            wholesalerNames.push(item.wholesalerName);
        });
        
        if (formData.getAll('files').length === 0) {
            throw new Error('処理できる有効なファイルがありませんでした。');
        }
       
        wholesalerNames.forEach(name => formData.append('wholesalerNames', name));
        
        const res = await fetch('/api/pricing/upload', {
             method: 'POST',
            body: formData,
         });
        
        if (!res.ok) {
            const errText = await res.text();
            throw new Error(errText || 'アップロード処理に失敗しました。'); 
        }
 
        const responseData = await res.json();
        setFullPricingData(responseData.productData);
        setOrderedWholesalers(responseData.wholesalerOrder);
        
        window.showNotification('見積CSVを読み込みました。', 'success');
        applyFiltersAndRender();
    } catch (err) {
        window.showNotification(err.message, 'error');
    } finally {
        window.hideLoading();
        uploadInput.value = '';
    }
}

/**
 * 見積テンプレートCSVエクスポートハンドラ
 */
async function handleExport(unregisteredOnly = false) {
    const selectedWholesalerName = wholesalerSelect.options[wholesalerSelect.selectedIndex].text;
    
    if (!wholesalerSelect.value) {
         window.showNotification('テンプレートを出力する卸業者を選択してください。', 'error'); 
        return;
    }

    const date = new Date();
    const dateStr = `${date.getFullYear()}${(date.getMonth()+1).toString().padStart(2, '0')}${date.getDate().toString().padStart(2, '0')}`;
    const params = new URLSearchParams({
         wholesalerName: selectedWholesalerName,
        unregisteredOnly: unregisteredOnly,
        date: dateStr,
     });
    window.location.href = `/api/pricing/export?${params.toString()}`;
}

/**
 * 画面表示時の初期データロード
 */
export async function loadInitialMasters() {
    const outputContainer = document.getElementById('pricing-output-container');
    outputContainer.innerHTML = '<p>製品マスターと既存の見積データを読み込んでいます...</p>';
    window.showLoading('データ読み込み中...');
    try {
        const res = await fetch('/api/pricing/all_masters');
        if (!res.ok) throw new Error('製品マスターの読み込みに失敗しました。');
        
        const responseData = await res.json();
        setFullPricingData(responseData.productData);
        setOrderedWholesalers(responseData.wholesalerOrder);

        applyFiltersAndRender();
    } catch(err) {
        outputContainer.innerHTML = `<p style="color:red;">${err.message}</p>`;
    } finally {
         window.hideLoading();
    }
}

/**
 * バックアップCSV一括インポートハンドラ
 */
async function handleDirectImport(event) {
    const file = event.target.files[0];
     if (!file) return;

    if (!confirm('選択したファイルの内容で納入価と卸情報を一括更新します。この操作は元に戻せません。よろしいですか？')) {
         event.target.value = ''; 
        return;
    }

     const formData = new FormData();
    formData.append('file', file);

     window.showLoading('納入価・卸を一括インポート中...');
    try {
        const res = await fetch('/api/pricing/direct_import', {
            method: 'POST',
             body: formData,
        });
         const resData = await res.json();
        if (!res.ok) {
             throw new Error(resData.message || 'インポートに失敗しました。');
         }
        window.showNotification(resData.message, 'success');
         loadInitialMasters(); // データを再読み込み
    } catch (err) {
        console.error(err);
         window.showNotification(`エラー: ${err.message}`, 'error');
    } finally {
         window.hideLoading();
        event.target.value = ''; 
    }
}

/**
 * APIを呼び出すイベントリスナーを登録します。
 */
export function initPricingEvents() {
    wholesalerSelect = document.getElementById('pricing-wholesaler-select');
    exportBtn = document.getElementById('pricing-export-btn');
    uploadInput = document.getElementById('pricing-upload-input'); 
    bulkUpdateBtn = document.getElementById('pricing-bulk-update-btn'); 
    exportUnregisteredBtn = document.getElementById('pricing-export-unregistered-btn');
    directImportBtn = document.getElementById('pricing-direct-import-btn');
    directImportInput = document.getElementById('pricing-direct-import-input');

    if (directImportBtn) {
        directImportBtn.addEventListener('click', () => directImportInput.click());
    }
    if (directImportInput) {
        directImportInput.addEventListener('change', handleDirectImport);
    }
    if (uploadInput) {
        uploadInput.addEventListener('change', handleUpload); 
    }
    if (bulkUpdateBtn) {
        bulkUpdateBtn.addEventListener('click', handleBulkUpdate); 
    }
    if (exportBtn) {
        exportBtn.addEventListener('click', () => handleExport(false));
    }
    if (exportUnregisteredBtn) {
        exportUnregisteredBtn.addEventListener('click', () => handleExport(true));
    }
}