// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\reorder_events.js
import { hiraganaToKatakana, fetchProductMasterByBarcode } from './utils.js';
import { showModal } from './search_modal.js';
import { renderOrderCandidates, addOrUpdateOrderItem } from './reorder_ui.js';
import { openContinuousScanModal } from './reorder_continuous_scan.js';

// DOM要素 (initReorderEventsで初期化)
let outputContainer, kanaNameInput, dosageFormInput, coefficientInput, shelfNumberInput;
let createCsvBtn, barcodeInput, barcodeForm, addFromMasterBtn;
let runBtn, continuousOrderBtn;

/**
 * 単品バーコードスキャン（手動追加）のハンドラ
 */
async function handleOrderBarcodeScan(e) { 
    e.preventDefault();
    const inputValue = barcodeInput.value.trim();
    if (!inputValue) return;
    window.showLoading('製品情報を検索中...');
    try {
        const productMaster = await fetchProductMasterByBarcode(inputValue); 
        addOrUpdateOrderItem(productMaster); 
        barcodeInput.value = '';
        barcodeInput.focus();
    } catch (err) {
        window.showNotification(err.message, 'error');
    } finally {
        window.hideLoading();
    }
}

/**
 * 「品目検索から追加」ボタンのハンドラ
 */
function handleAddFromMaster() { 
  showModal(
        document.getElementById('reorder-view'), 
        (selectedProduct) => {
            // TKRでは採用・未採用の区別なく、選択されたものをそのまま追加
            addOrUpdateOrderItem(selectedProduct);
        },
        {
            searchMode: 'inout', // JCSHMSからも検索可能にする
            allowAdopted: true   // ▼▼▼【追加】採用済みでも選択可能にする ▼▼▼
        }
    );
}

/**
 * 「発注候補を作成」ボタンのハンドラ
 */
async function handleGenerateCandidates() { 
    window.showLoading('発注候補リストを作成中...');
    const params = new URLSearchParams({
        kanaName: hiraganaToKatakana(kanaNameInput.value),
        dosageForm: dosageFormInput.value,
        shelfNumber: shelfNumberInput.value,
        coefficient: coefficientInput.value,
    });

    try {
        const res = await fetch(`/api/reorder/candidates?${params.toString()}`);
        if (!res.ok) {
            const errText = await res.text();
            throw new Error(errText || 'List generation failed');
        }
        const data = await res.json();
        renderOrderCandidates(data, outputContainer); 
    } catch (err) {
        outputContainer.innerHTML = `<p class="status-error">エラー: ${err.message}</p>`;
    } finally {
        window.hideLoading();
    }
}

/**
 * 「CSV作成・発注残登録」ボタンのハンドラ
 */
async function handleCreateCsv(fetchAndRenderReorderCallback) {
    const rows = outputContainer.querySelectorAll('#orderable-table tbody tr');

    if (rows.length === 0) {
        window.showNotification('発注する品目がありません。', 'error');
        return;
    }

    const backorderPayload = [];
    let csvContent = "";
    let hasItemsToOrder = false;

    rows.forEach(row => {
        if (row.classList.contains('provisional-order-item')) {
            return; // 発注不可の行はスキップ
        }
        
        const quantityInput = row.querySelector('.order-quantity-input');
        const quantity = parseInt(quantityInput.value, 10);
        
        if (quantity > 0) {
            hasItemsToOrder = true;
            
            const janCode = row.dataset.janCode;
            const productName = row.cells[0].textContent; 
            const wholesalerCode = row.querySelector('.wholesaler-select').value;

            // TKR CSVフォーマット
            const csvRow = [
                janCode, 
                `"${productName.replace(/"/g, '""')}"`, 
                quantity, 
                wholesalerCode
            ].join(',');
            csvContent += csvRow + "\r\n";

            const orderMultiplier = parseFloat(row.dataset.orderMultiplier) || 0;
            
            backorderPayload.push({
                janCode: janCode,
                yjCode: row.dataset.yjCode,
                packageForm: row.dataset.packageForm,
                janPackInnerQty: parseFloat(row.dataset.janPackInnerQty),
                yjUnitName: row.dataset.yjUnitName,
                yjQuantity: quantity * orderMultiplier, // YJ単位に換算
                productName: row.dataset.productName,
                yjPackUnitQty: parseFloat(row.dataset.yjPackUnitQty) || 0,
                janPackUnitQty: parseFloat(row.dataset.janPackUnitQty) || 0,
                janUnitCode: parseInt(row.dataset.janUnitCode, 10) || 0,
                wholesalerCode: wholesalerCode,
            });
        }
    });

    if (!hasItemsToOrder) {
        window.showNotification('発注数が1以上の品目がありません。', 'error');
        return;
    }

    window.showLoading('発注残を登録中...');
    try {
        // 1. 発注残をDBに登録
        const res = await fetch('/api/orders/place', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(backorderPayload),
        });
        const resData = await res.json();
        if (!res.ok) throw new Error(resData.message || '発注残の登録に失敗しました。');
        
        window.showNotification(resData.message, 'success');

        // 2. Shift-JIS CSVを生成・ダウンロード
        const sjisArray = Encoding.convert(csvContent, {
            to: 'SJIS',
            from: 'UNICODE',
            type: 'array'
        });
        const sjisUint8Array = new Uint8Array(sjisArray);

        const blob = new Blob([sjisUint8Array], { type: 'text/csv; charset=shift_jis' });
        const link = document.createElement("a");
        const url = URL.createObjectURL(blob);
        const now = new Date();
        const timestamp = `${now.getFullYear()}${(now.getMonth()+1).toString().padStart(2, '0')}${now.getDate().toString().padStart(2, '0')}_${now.getHours().toString().padStart(2, '0')}${now.getMinutes().toString().padStart(2, '0')}${now.getSeconds().toString().padStart(2, '0')}`;
        const fileName = `発注書_${timestamp}.csv`;
        link.setAttribute("href", url);
        link.setAttribute("download", fileName);
        link.style.visibility = 'hidden';
        document.body.appendChild(link);
        link.click();
        document.body.removeChild(link);

        // 成功したらリストをクリア
        fetchAndRenderReorderCallback(); 

    } catch(err) {
        window.showNotification(err.message, 'error');
    } finally {
        window.hideLoading();
    }
}

/**
 * テーブル内（除外、発注不可など）のクリックイベントハンドラ
 */
async function handleTableClicks(e, handleGenerateCandidatesCallback) { 
    const target = e.target;
    const row = target.closest('tr');
    if (!row) return;

    // 「発注不可」ボタン (発注可テーブル内)
    if (target.classList.contains('set-unorderable-btn')) {
        const productCode = target.dataset.productCode;
        const productName = row.cells[0].textContent;
        if (!confirm(`「${productName}」を発注不可に設定しますか？\nこの品目は今後、不足品リストに表示されなくなります。`)) {
             return;
        }
        window.showLoading('マスターを更新中...');
        try {
            const res = await fetch('/api/master/set_order_stopped', { 
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ productCode: productCode, status: 1 }),
            });
            const resData = await res.json();
            if (!res.ok) throw new Error(resData.message || '更新に失敗しました。');
            
            window.showNotification(`「${productName}」を発注不可に設定しました。`, 'success');
            
            // 1. 発注可テーブルから行を削除
            row.remove(); 
            
            // 2. 発注不可テーブルに行を再構築して追加 (簡易的に、ページリロード)
            window.showNotification('発注不可リストに移動しました。リストを更新します。', 'info');
            handleGenerateCandidatesCallback(); // 発注候補を再生成して全体を再描画

        } catch(err) {
            window.showNotification(err.message, 'error');
        } finally {
            window.hideLoading();
        }
    } 
    // 「発注に変更」ボタン (発注不可テーブル内)
    else if (target.classList.contains('change-to-orderable-btn')) {
        const productCode = row.dataset.janCode;
        if (!productCode) return;

        window.showLoading('マスターを更新中...');
        try {
            // 1. APIでマスターを発注可 (0) に更新
            const res = await fetch('/api/master/set_order_stopped', { 
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ productCode: productCode, status: 0 }),
            });
            const resData = await res.json();
            if (!res.ok) throw new Error(resData.message || '更新に失敗しました。');

            window.showNotification(`「${row.dataset.productName}」を発注可に変更しました。`, 'success');
            // 2. 発注不可テーブルから行を削除
            row.remove();

            // 3. 発注可テーブルに行を追加 (簡易的に、ページリロード)
            window.showNotification('発注対象リストに移動しました。リストを更新します。', 'info');
            handleGenerateCandidatesCallback(); // 発注候補を再生成して全体を再描画

        } catch(err) {
            window.showNotification(err.message, 'error');
        } finally {
            window.hideLoading();
        }
    } 
    // 「除外」ボタン (発注可テーブル内)
    else if (target.classList.contains('remove-order-item-btn')) {
        const tbody = row.closest('tbody');
        const table = tbody.closest('table');
        row.remove();
        
        if (tbody.children.length === 0 && table.id === 'orderable-table') {
             const header = outputContainer.querySelector('h3');
            if(header) header.textContent = `発注対象品目 (0件)`;
            tbody.innerHTML = '<tr><td colspan="8">発注対象の品目はありません。</td></tr>';
        }
    }
}

/**
 * 発注ビューの全イベントリスナーを初期化
 */
export function initReorderEvents(fetchAndRenderReorderCallback) { 
    runBtn = document.getElementById('generate-order-candidates-btn');
    outputContainer = document.getElementById('order-candidates-output');
    kanaNameInput = document.getElementById('order-kanaName');
    dosageFormInput = document.getElementById('order-dosageForm');
    coefficientInput = document.getElementById('order-reorder-coefficient');
    createCsvBtn = document.getElementById('createOrderCsvBtn');
    barcodeInput = document.getElementById('order-barcode-input');
    barcodeForm = document.getElementById('order-barcode-form');
    shelfNumberInput = document.getElementById('order-shelf-number');
    addFromMasterBtn = document.getElementById('add-order-item-from-master-btn');
    continuousOrderBtn = document.getElementById('continuous-order-btn');

    if (addFromMasterBtn) {
        addFromMasterBtn.addEventListener('click', handleAddFromMaster);
    }
    if (continuousOrderBtn) {
        continuousOrderBtn.addEventListener('click', openContinuousScanModal);
    }
    if (barcodeForm) {
        barcodeForm.addEventListener('submit', handleOrderBarcodeScan);
    }
    if (runBtn) {
        runBtn.addEventListener('click', handleGenerateCandidates);
    }
    if (createCsvBtn) {
        createCsvBtn.addEventListener('click', () => handleCreateCsv(fetchAndRenderReorderCallback));
    }
    if (outputContainer) {
        outputContainer.addEventListener('click', (e) => handleTableClicks(e, handleGenerateCandidates));
    }
}