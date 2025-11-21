// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\reorder_events.js
import { fetchProductMasterByBarcode } from './utils.js';
import { showModal } from './search_modal.js';
import { renderOrderCandidates, addOrUpdateOrderItem } from './reorder_ui.js';
import { openContinuousScanModal } from './reorder_continuous_scan.js';

let outputContainer, coefficientInput;
let createCsvBtn, barcodeInput, barcodeForm, addFromMasterBtn;
let runBtn, continuousOrderBtn;
// ▼▼▼ 追加: 変数定義 ▼▼▼
let createDatBtn;
// ▲▲▲ 追加ここまで ▲▲▲
let reservationBtn, reservationModal, reservationDateTimeInput, cancelReservationBtn, confirmReservationBtn;

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

function handleAddFromMaster() { 
  showModal(
        document.getElementById('reorder-view'), 
        (selectedProduct) => {
            addOrUpdateOrderItem(selectedProduct);
        },
        {
            searchMode: 'inout',
            allowAdopted: true
        }
    );
}

async function handleGenerateCandidates() { 
    window.showLoading('発注候補リストを作成中...');
    // 不要な検索条件を削除し、空文字を送る
    const params = new URLSearchParams({
        kanaName: '',
        dosageForm: '',
        shelfNumber: '',
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

function getOrderItems(rows) {
    const backorderPayload = [];
    let hasItemsToOrder = false;

    rows.forEach(row => {
        if (row.classList.contains('provisional-order-item')) {
            return;
        }
       
        const quantityInput = row.querySelector('.order-quantity-input');
        const quantity = parseInt(quantityInput.value, 10);
        
        if (quantity > 0) {
            hasItemsToOrder = true;
            
            const janCode = row.dataset.janCode;
            const wholesalerCode = row.querySelector('.wholesaler-select').value;
            const orderMultiplier = parseFloat(row.dataset.orderMultiplier) || 0;
            
            backorderPayload.push({
                janCode: janCode,
                yjCode: row.dataset.yjCode,
                packageForm: row.dataset.packageForm,
                janPackInnerQty: parseFloat(row.dataset.janPackInnerQty),
                yjUnitName: row.dataset.yjUnitName,
                yjQuantity: quantity * orderMultiplier,
                productName: row.dataset.productName,
                yjPackUnitQty: parseFloat(row.dataset.yjPackUnitQty) || 0,
                janPackUnitQty: parseFloat(row.dataset.janPackUnitQty) || 0,
                janUnitCode: parseInt(row.dataset.janUnitCode, 10) || 0,
                wholesalerCode: wholesalerCode,
            });
        }
    });

    return { backorderPayload, hasItemsToOrder };
}

async function handleCreateCsv(fetchAndRenderReorderCallback) {
    const rows = outputContainer.querySelectorAll('#orderable-table tbody tr');

    if (rows.length === 0) {
        window.showNotification('発注する品目がありません。', 'error');
        return;
    }

    const { backorderPayload, hasItemsToOrder } = getOrderItems(rows);

    if (!hasItemsToOrder) {
        window.showNotification('発注数が1以上の品目がありません。', 'error');
        return;
    }

    let csvContent = "";
    rows.forEach(row => {
        if (row.classList.contains('provisional-order-item')) return;
        const quantityInput = row.querySelector('.order-quantity-input');
        const quantity = parseInt(quantityInput.value, 10);
    
        if (quantity > 0) {
            const janCode = row.dataset.janCode;
            const productName = row.cells[0].textContent; 
            const wholesalerCode = row.querySelector('.wholesaler-select').value;
            const csvRow = [janCode, `"${productName.replace(/"/g, '""')}"`, quantity, wholesalerCode].join(',');
            csvContent += csvRow + "\r\n";
        }
    });

    window.showLoading('発注残を登録中...');
    try {
        const res = await fetch('/api/orders/place', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(backorderPayload),
        });
 
        const resData = await res.json();
        if (!res.ok) throw new Error(resData.message || '発注残の登録に失敗しました。');
        
        window.showNotification(resData.message, 'success');

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

        fetchAndRenderReorderCallback();
    } catch(err) {
        window.showNotification(err.message, 'error');
    } finally {
        window.hideLoading();
    }
}

async function handlePlaceReservation(fetchAndRenderReorderCallback) {
    const dateVal = reservationDateTimeInput.value;
    if (!dateVal) {
        window.showNotification('日時を指定してください。', 'warning');
        return;
    }

    const rows = outputContainer.querySelectorAll('#orderable-table tbody tr');
    const { backorderPayload, hasItemsToOrder } = getOrderItems(rows);

    if (!hasItemsToOrder) {
        window.showNotification('発注数が1以上の品目がありません。', 'error');
        return;
    }

    backorderPayload.forEach(item => {
        item.orderDate = dateVal;
    });

    window.showLoading('予約発注を登録中...');
    try {
        const res = await fetch('/api/orders/place', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(backorderPayload),
        });
        const resData = await res.json();
        if (!res.ok) throw new Error(resData.message || '予約の登録に失敗しました。');
        
        window.showNotification(resData.message, 'success');
        reservationModal.classList.add('hidden');
        fetchAndRenderReorderCallback();
    } catch(err) {
        window.showNotification(err.message, 'error');
    } finally {
        window.hideLoading();
    }
}

// ▼▼▼ 追加: DAT作成ハンドラ ▼▼▼
async function handleCreateDat() {
    const rows = outputContainer.querySelectorAll('#orderable-table tbody tr');
    if (rows.length === 0) {
        window.showNotification('発注する品目がありません。', 'error');
        return;
    }

    const payload = [];
    rows.forEach(row => {
        if (row.classList.contains('provisional-order-item')) return;
        
        const quantityInput = row.querySelector('.order-quantity-input');
        const quantity = parseFloat(quantityInput.value) || 0;
        
        if (quantity > 0) {
            const janCode = row.dataset.janCode;
            const wholesalerCode = row.querySelector('.wholesaler-select').value;
            const kanaNameShort = row.dataset.kanaNameShort || ''; // UIで追加した属性を取得

            if (!wholesalerCode) {
                // 卸未選択はスキップ、あるいはエラーにする？とりあえずスキップ
                return; 
            }

            payload.push({
                janCode: janCode,
                wholesalerCode: wholesalerCode,
                orderQuantity: quantity,
                kanaNameShort: kanaNameShort
            });
        }
    });

    if (payload.length === 0) {
        window.showNotification('発注数1以上の有効な品目がありません。', 'error');
        return;
    }

    if(!confirm("固定長DATファイルを作成しますか？\n※MEDICODEユーザーIDが薬局IDとして使用されます。")) {
        return;
    }

    window.showLoading('DATファイルを作成中...');
    try {
        const res = await fetch('/api/reorder/export_dat', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
        });

        if (!res.ok) {
            const errText = await res.text();
            throw new Error(errText || 'DAT作成に失敗しました。');
        }

        // ファイルダウンロード処理
        const blob = await res.blob();
        const url = window.URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        
        // ファイル名を取得 (Content-Dispositionから)
        const contentDisposition = res.headers.get('Content-Disposition');
        let fileName = 'order.dat';
        if (contentDisposition && contentDisposition.indexOf('filename*=') !== -1) {
            const match = contentDisposition.match(/filename\*=UTF-8''(.+)/);
            if (match && match[1]) {
                fileName = decodeURIComponent(match[1]);
            }
        }
        
        a.download = fileName;
        document.body.appendChild(a);
        a.click();
        a.remove();
        window.URL.revokeObjectURL(url);

        window.showNotification('DATファイルをダウンロードしました。', 'success');

    } catch (err) {
        window.showNotification(err.message, 'error');
    } finally {
        window.hideLoading();
    }
}
// ▲▲▲ 追加ここまで ▲▲▲

async function handleTableClicks(e, handleGenerateCandidatesCallback) { 
    const target = e.target;
    const row = target.closest('tr');
    if (!row) return;

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
            row.remove(); 
            
            window.showNotification('発注不可リストに移動しました。リストを更新します。', 'info');
            handleGenerateCandidatesCallback();

        } catch(err) {
            window.showNotification(err.message, 'error');
        } finally {
            window.hideLoading();
        }
    } 
    else if (target.classList.contains('change-to-orderable-btn')) {
        const productCode = row.dataset.janCode;
        if (!productCode) return;

        window.showLoading('マスターを更新中...');
        try {
            const res = await fetch('/api/master/set_order_stopped', { 
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ productCode: productCode, status: 0 }),
            });
            const resData = await res.json();
            if (!res.ok) throw new Error(resData.message || '更新に失敗しました。');

            window.showNotification(`「${row.dataset.productName}」を発注可に変更しました。`, 'success');
            row.remove();

            window.showNotification('発注対象リストに移動しました。リストを更新します。', 'info');
            handleGenerateCandidatesCallback();

        } catch(err) {
            window.showNotification(err.message, 'error');
        } finally {
            window.hideLoading();
        }
    } 
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

export function initReorderEvents(fetchAndRenderReorderCallback) { 
    runBtn = document.getElementById('generate-order-candidates-btn');
    outputContainer = document.getElementById('order-candidates-output');
    coefficientInput = document.getElementById('order-reorder-coefficient');
    createCsvBtn = document.getElementById('createOrderCsvBtn');
    barcodeInput = document.getElementById('order-barcode-input');
    barcodeForm = document.getElementById('order-barcode-form');
    addFromMasterBtn = document.getElementById('add-order-item-from-master-btn');
    continuousOrderBtn = document.getElementById('continuous-order-btn');

    // ▼▼▼ 追加: ボタン要素取得 ▼▼▼
    createDatBtn = document.getElementById('createFixedLengthDatBtn');
    // ▲▲▲ 追加ここまで ▲▲▲

    reservationBtn = document.getElementById('reservation-order-btn');
    reservationModal = document.getElementById('reservation-modal');
    reservationDateTimeInput = document.getElementById('reservation-datetime');
    cancelReservationBtn = document.getElementById('cancel-reservation-btn');
    confirmReservationBtn = document.getElementById('confirm-reservation-btn');

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

    // ▼▼▼ 追加: イベントリスナー登録 ▼▼▼
    if (createDatBtn) {
        createDatBtn.addEventListener('click', handleCreateDat);
    }
    // ▲▲▲ 追加ここまで ▲▲▲

    if (reservationBtn) {
        reservationBtn.addEventListener('click', () => {
            const rows = outputContainer.querySelectorAll('#orderable-table tbody tr');
            if (rows.length === 0) {
                window.showNotification('発注する品目がありません。', 'error');
                return;
            }
            
            const tomorrow = new Date();
            tomorrow.setDate(tomorrow.getDate() + 1);
            tomorrow.setHours(9, 0, 0, 0);
            
            const yyyy = tomorrow.getFullYear();
            const mm = String(tomorrow.getMonth() + 1).padStart(2, '0');
            const dd = String(tomorrow.getDate()).padStart(2, '0');
            const hh = String(tomorrow.getHours()).padStart(2, '0');
            const min = String(tomorrow.getMinutes()).padStart(2, '0');
            
            reservationDateTimeInput.value = `${yyyy}-${mm}-${dd}T${hh}:${min}`;
            reservationModal.classList.remove('hidden');
        });
    }

    if (cancelReservationBtn) {
        cancelReservationBtn.addEventListener('click', () => {
            reservationModal.classList.add('hidden');
        });
    }

    if (confirmReservationBtn) {
        confirmReservationBtn.addEventListener('click', () => handlePlaceReservation(fetchAndRenderReorderCallback));
    }
}