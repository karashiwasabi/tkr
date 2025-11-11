// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\reorder.js
// (WASABI: static/js/orders.js を TKR 用に移植・改変)
import { hiraganaToKatakana, parseBarcode, fetchProductMasterByBarcode } from './utils.js';
import { wholesalerMap } from './master_data.js';
import { showModal } from './search_modal.js';

// --- TKR版 発注機能 ---

let outputContainer, kanaNameInput, dosageFormInput, coefficientInput, shelfNumberInput;
let createCsvBtn, barcodeInput, barcodeForm, addFromMasterBtn;

let continuousOrderModal, continuousOrderBtn, closeContinuousModalBtn;
let continuousBarcodeForm, continuousBarcodeInput, scannedItemsList, scannedItemsCount, processingIndicator;
let scanQueue = [];
let isProcessingQueue = false;

/**
 * TKRでは .toFixed(2) は不要 (DB側で REAL 型として適切に扱われる)
 */
function formatBalance(balance) {
    if (typeof balance === 'number') {
        return balance;
    }
    return balance;
}

/**
 * 検索モーダルからのコールバック。品目をリストに追加または更新します。
 * (WASABI: orders.js を TKR 用に修正)
 */
function addOrUpdateOrderItem(productMaster) {
    if (!outputContainer) outputContainer = document.getElementById('order-candidates-output');
    
    const productCode = productMaster.productCode;
    const yjCode = productMaster.yjCode;

    // 既にリストにあるか確認
    const existingRow = outputContainer.querySelector(`tr[data-jan-code="${productCode}"]`);
    if (existingRow) {
        const quantityInput = existingRow.querySelector('.order-quantity-input');
        if (quantityInput) {
            quantityInput.value = parseInt(quantityInput.value, 10) + 1;
            window.showNotification(`「${productMaster.productName}」の数量を1増やしました。`, 'success');
        }
        return;
    }

    // 卸業者ドロップダウンを生成
    let wholesalerOptions = '<option value="">--- 選択 ---</option>';
    wholesalerMap.forEach((name, code) => {
        const isSelected = (code === productMaster.supplierWholesale);
        wholesalerOptions += `<option value="${code}" ${isSelected ? 'selected' : ''}>${name}</option>`;
    });

    // TKRのテーブル定義 (table.css) に合わせた操作ボタン
    const actionCellHTML = `
        <td class="center col-action order-actions-cell">
            <div class="order-action-buttons">
                <button type="button" class="remove-order-item-btn btn">除外</button>
                <button type="button" class="set-unorderable-btn btn" data-product-code="${productMaster.productCode}">発注不可</button>
            </div>
        </td>
    `;
    
    // TKRのテーブル定義 (table.css) に合わせた列構成
    const newRowHTML = `
        <tr data-jan-code="${productMaster.productCode}" 
            data-yj-code="${productMaster.yjCode}"
            data-product-name="${productMaster.productName}"
            data-package-form="${productMaster.packageForm}"
            data-jan-pack-inner-qty="${productMaster.janPackInnerQty}"
            data-yj-unit-name="${productMaster.yjUnitName}"
            data-yj-pack-unit-qty="${productMaster.yjPackUnitQty}"
            data-order-multiplier="${productMaster.yjPackUnitQty}"> 
            
            <td class="left col-product" colspan="2">${productMaster.productName}</td>
            <td class="left col-maker">${productMaster.makerName || ''}</td>
            <td class="left col-package">${productMaster.formattedPackageSpec}</td>
            <td class="col-wholesaler"><select class="wholesaler-select" style="width: 100%; font-size: 10px;">${wholesalerOptions}</select></td>
            <td class="col-count">${productMaster.yjPackUnitQty} ${productMaster.yjUnitName}</td>
            <td class="col-line"><input type="number" value="1" class="order-quantity-input" style="width: 100%; text-align: right;"></td>
            ${actionCellHTML}
        </tr>
    `;

    let yjGroupWrapper = outputContainer.querySelector(`.order-yj-group-wrapper[data-yj-code="${yjCode}"]`);

    if (yjGroupWrapper) {
        const tbody = yjGroupWrapper.querySelector('tbody');
        tbody.insertAdjacentHTML('beforeend', newRowHTML);
    } else {
        const yjHeaderHTML = `
            <div class="agg-yj-header" style="background-color: #f0f0f0; color: #333; border: 1px solid #ccc;">
                <span>YJ: ${yjCode}</span>
                <span class="product-name">${productMaster.productName}</span>
            </div>`;
        
        // TKRのテーブル定義 (table.css) に合わせたヘッダー
        const tableHeader = `
            <thead>
                <tr>
                    <th class="col-product" colspan="2">製品名（包装）</th>
                    <th class="col-maker">メーカー</th>
                    <th class="col-package">包装仕様</th>
                    <th class="col-wholesaler">卸業者</th>
                    <th class="col-count">発注単位</th>
                    <th class="col-line">発注数</th>
                    <th class="col-action">操作</th>
                </tr>
            </thead>
        `;
            
        const tableHTML = `
            <table class="data-table" style="margin-bottom: 10px;">
                ${tableHeader}
                <tbody>
                    ${newRowHTML}
                </tbody>
            </table>`;
            
        const newGroupHTML = `
            <div class="order-yj-group-wrapper" data-yj-code="${yjCode}">
                ${yjHeaderHTML}
                ${tableHTML}
            </div>`;
        outputContainer.insertAdjacentHTML('beforeend', newGroupHTML);
    }
    window.showNotification(`「${productMaster.productName}」を発注リストに追加しました。`, 'success');
}


/**
 * 連続スキャン用: キューの表示を更新
 * (WASABI: orders.js より)
 */
function updateScannedItemsDisplay() {
    const counts = scanQueue.reduce((acc, code) => {
        acc[code] = (acc[code] || 0) + 1;
        return acc;
    }, {});
    scannedItemsList.innerHTML = Object.entries(counts).map(([code, count]) => {
        return `<div class="scanned-item">
                    <span class="scanned-item-name">${code}</span>
                    <span class="scanned-item-count">x ${count}</span>
                </div>`;
    }).join('');
    scannedItemsCount.textContent = scanQueue.length;
}

/**
 * 連続スキャン用: スキャンキューを処理
 * (WASABI: orders.js を TKR 用に修正)
 */
async function processScanQueue() {
    if (isProcessingQueue) return;

    isProcessingQueue = true;
    processingIndicator.classList.remove('hidden');
    while (scanQueue.length > 0) {
        const barcode = scanQueue.shift();
        try {
            // TKRの fetchProductMasterByBarcode を使用
            const productMaster = await fetchProductMasterByBarcode(barcode);
            addOrUpdateOrderItem(productMaster);
        } catch (err) {
            console.error(`バーコード[${barcode}]の処理に失敗:`, err);
            window.showNotification(`バーコード[${barcode}]の処理に失敗しました: ${err.message}`, 'error');
        } finally {
            updateScannedItemsDisplay();
        }
    }

    isProcessingQueue = false;
    processingIndicator.classList.add('hidden');
}


/**
 * 発注候補リスト（自動生成）を描画します。
 * (WASABI: orders.js を TKR 用に修正)
 */
function renderOrderCandidates(data, container) {
    if (!data.candidates || data.candidates.length === 0) {
        container.innerHTML = "<p>発注が必要な品目はありませんでした。</p>";
        return;
    }

    let html = '';
    data.candidates.forEach(yjGroup => {
        const yjShortfall = yjGroup.totalReorderPoint - (yjGroup.endingBalance || 0);

        html += `
            <div class="order-yj-group-wrapper" data-yj-code="${yjGroup.yjCode}">
                <div class="agg-yj-header" style="background-color: #dc3545;">
                    <span>YJ: ${yjGroup.yjCode}</span>
                    <span class="product-name">${yjGroup.productName}</span>
                    <span class="balance-info">
                        在庫: ${formatBalance(yjGroup.endingBalance)} | 
                        発注点: ${formatBalance(yjGroup.totalReorderPoint)} | 
                        不足数: ${formatBalance(yjShortfall)}
                    </span>
                </div>
        `;

        const existingBackordersForYj = yjGroup.packageLedgers.flatMap(p => p.existingBackorders || []);
        if (existingBackordersForYj.length > 0) {
            html += `<div class="existing-backorders-info">
                        <strong>＜既存の発注残＞</strong>
                        <ul>`;
            existingBackordersForYj.forEach(bo => {
                const wName = wholesalerMap.get(bo.wholesalerCode) || bo.wholesalerCode || '不明';
                html += `<li>${bo.orderDate}: ${bo.productName} - 数量: ${bo.remainingQuantity.toFixed(2)} (${wName})</li>`;
            });
            html += `</ul></div>`;
        }

        html += `
                <table class="data-table" style="margin-bottom: 10px;">
                    <thead>
                        <tr>
                            <th class="col-product" colspan="2">製品名（包装）</th>
                            <th class="col-maker">メーカー</th>
                            <th class="col-package">包装仕様</th>
                            <th class="col-wholesaler">卸業者</th>
                            <th class="col-count">発注単位</th>
                            <th class="col-line">発注数</th>
                            <th class="col-action">操作</th>
                        </tr>
                    </thead>
                    <tbody>
        `;
        
        yjGroup.packageLedgers.forEach(pkg => {
            if (pkg.masters && pkg.masters.length > 0) {
                pkg.masters.forEach(master => {
                    const pkgShortfall = pkg.reorderPoint - (pkg.endingBalance || 0);
                    
                    if (pkgShortfall > 0) {
                        const isOrderStopped = master.isOrderStopped === 1;
                        const isOrderable = !isOrderStopped;

                        const rowClass = !isOrderable ? 'provisional-order-item' : '';
                        const disabledAttr = !isOrderable ? 'disabled' : '';

                        // 発注推奨数を計算 (TKRでは YjPackUnitQty が 0 の場合があるためガード)
                        const recommendedOrder = master.yjPackUnitQty > 0 ? Math.ceil(pkgShortfall / master.yjPackUnitQty) : 0;
                        
                        let rowWholesalerOptions = '<option value="">--- 選択 ---</option>';
                        data.wholesalers.forEach(w => {
                            const isSelected = (w.wholesalerCode === master.supplierWholesale);
                            rowWholesalerOptions += `<option value="${w.wholesalerCode}" ${isSelected ? 'selected' : ''}>${w.wholesalerName}</option>`;
                        });
                        
                        let actionCellHTML = `
                            <td class="center col-action order-actions-cell">
                                <div class="order-action-buttons">
                        `;
                        if (isOrderable) {
                            actionCellHTML += '<button type="button" class="remove-order-item-btn btn">除外</button>';
                        } else {
                            actionCellHTML += '<button type="button" class="change-to-orderable-btn btn">発注に変更</button>';
                        }
                        actionCellHTML += `
                                    <button type="button" class="set-unorderable-btn btn" data-product-code="${master.productCode}">発注不可</button>
                                </div>
                            </td>
                        `;
                        
                        html += `
                            <tr class="${rowClass}" 
                                data-jan-code="${master.productCode}" 
                                data-yj-code="${yjGroup.yjCode}"
                                data-product-name="${master.productName}"
                                data-package-form="${master.packageForm}"
                                data-jan-pack-inner-qty="${master.janPackInnerQty}"
                                data-yj-unit-name="${master.yjUnitName}"
                                data-yj-pack-unit-qty="${master.yjPackUnitQty}"
                                data-order-multiplier="${master.yjPackUnitQty}"> 
                                
                                <td class="left col-product" colspan="2">${master.productName}</td>
                                <td class="left col-maker">${master.makerName || ''}</td>
                                <td class="left col-package">${master.formattedPackageSpec}</td>
                                <td class="col-wholesaler"><select class="wholesaler-select" style="width: 100%; font-size: 10px;" ${disabledAttr}>${rowWholesalerOptions}</select></td>
                                <td class="col-count">${master.yjPackUnitQty} ${master.yjUnitName}</td>
                                <td class="col-line"><input type="number" value="${recommendedOrder}" class="order-quantity-input" style="width: 100%; text-align: right;" ${disabledAttr}></td>
                                ${actionCellHTML}
                            </tr>
                        `;
                    }
                });
            }
        });
        html += `</tbody></table></div>`;
    });
    container.innerHTML = html;
}

/**
 * 単品バーコードスキャン（手動追加）のハンドラ
 * (WASABI: orders.js を TKR 用に修正)
 */
async function handleOrderBarcodeScan(e) {
    e.preventDefault();
    const inputValue = barcodeInput.value.trim();
    if (!inputValue) return;

    window.showLoading('製品情報を検索中...');
    try {
        // TKRの fetchProductMasterByBarcode を使用
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
 * 発注点リストビューの初期化（旧 reorder.js の置き換え）
 */
export function initReorderView() {
    const view = document.getElementById('reorder-view');
    if (!view) return;

    const runBtn = document.getElementById('generate-order-candidates-btn');
    outputContainer = document.getElementById('order-candidates-output');
    kanaNameInput = document.getElementById('order-kanaName');
    dosageFormInput = document.getElementById('order-dosageForm');
    coefficientInput = document.getElementById('order-reorder-coefficient');
    createCsvBtn = document.getElementById('createOrderCsvBtn');
    barcodeInput = document.getElementById('order-barcode-input');
    barcodeForm = document.getElementById('order-barcode-form');
    shelfNumberInput = document.getElementById('order-shelf-number');
    addFromMasterBtn = document.getElementById('add-order-item-from-master-btn');

    continuousOrderModal = document.getElementById('continuous-order-modal');
    continuousOrderBtn = document.getElementById('continuous-order-btn');
    closeContinuousModalBtn = document.getElementById('close-continuous-modal-btn');
    continuousBarcodeForm = document.getElementById('continuous-barcode-form');
    continuousBarcodeInput = document.getElementById('continuous-barcode-input');
    scannedItemsList = document.getElementById('scanned-items-list');
    scannedItemsCount = document.getElementById('scanned-items-count');
    processingIndicator = document.getElementById('processing-indicator');

    // 「品目検索から追加」ボタン
    addFromMasterBtn.addEventListener('click', () => {
        showModal(
            view, 
            (selectedProduct) => {
                // selectedProduct は TKR の ProductMasterView
                // TKRでは採用・未採用の区別なく、選択されたものをそのまま追加
                addOrUpdateOrderItem(selectedProduct);
            },
            {
                searchMode: 'inout' // JCSHMSからも検索可能にする
            }
        );
    });

    // 「連続スキャンで追加」ボタン
    continuousOrderBtn.addEventListener('click', () => {
        scanQueue = [];
        updateScannedItemsDisplay();
        continuousOrderModal.classList.remove('hidden');
        document.body.classList.add('modal-open');
        setTimeout(() => continuousBarcodeInput.focus(), 100);
    });
    closeContinuousModalBtn.addEventListener('click', () => {
        continuousOrderModal.classList.add('hidden');
        document.body.classList.remove('modal-open');
    });
    continuousBarcodeForm.addEventListener('submit', (e) => {
        e.preventDefault();
        const barcode = continuousBarcodeInput.value.trim();
        if (barcode) {
            scanQueue.push(barcode);
            updateScannedItemsDisplay();
            processScanQueue(); // 非同期でキュー処理を開始
        }
        continuousBarcodeInput.value = '';
    });

    // 「バーコードで単品追加」フォーム
    if (barcodeForm) {
        barcodeForm.addEventListener('submit', handleOrderBarcodeScan);
    }
    
    // 「発注候補を作成」ボタン
    runBtn.addEventListener('click', async () => {
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
    });

    // 「CSV作成・発注残登録」ボタン
    createCsvBtn.addEventListener('click', async () => {
        const rows = outputContainer.querySelectorAll('tbody tr');
        if (rows.length === 0) {
            window.showNotification('発注する品目がありません。', 'error');
            return;
        }

        const backorderPayload = [];
        let csvContent = "JANコード,品名,数量,卸コード\r\n"; // TKR用のCSVヘッダー
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
                const productName = row.cells[0].textContent; // TKRのテーブルレイアウト (colspan=2)
                const wholesalerCode = row.querySelector('.wholesaler-select').value;
    
                // TKR CSVフォーマット
                const csvRow = [
                    janCode, 
                    `"${productName.replace(/"/g, '""')}"`, // 品名を""で囲む
                    quantity, 
                    wholesalerCode
                ].join(',');
                csvContent += csvRow + "\r\n";

                const orderMultiplier = parseFloat(row.dataset.orderMultiplier) || 0;
                
                backorderPayload.push({
                    yjCode: row.dataset.yjCode,
                    packageForm: row.dataset.packageForm,
                    janPackInnerQty: parseFloat(row.dataset.janPackInnerQty),
                    yjUnitName: row.dataset.yjUnitName,
                    yjQuantity: quantity * orderMultiplier, // YJ単位に換算
                    productName: row.dataset.productName,
                    yjPackUnitQty: parseFloat(row.dataset.yjPackUnitQty) || 0,
                    janPackUnitQty: parseFloat(row.dataset.janPackUnitQty) || 0,
                    janUnitCode: parseInt(row.dataset.janUnitCode, 10) || 0,
                    wholesalerCode: wholesalerCode, // TKRでは string
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
            // TKRには Encoding ライブラリがないため、Blobを直接 Shift-JIS で作成
            const sjisArray = Encoding.convert(csvContent, {
                to: 'SJIS',
                from: 'UNICODE',
                type: 'array'
            });
            const sjisUint8Array = new Uint8Array(sjisArray);

            const blob = new Blob([sjisUint8Array], { type: 'text/csv; charset=shift_jis' }); // MIMEタイプ指定
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
            fetchAndRenderReorder(); 

        } catch(err) {
            window.showNotification(err.message, 'error');
        } finally {
            window.hideLoading();
        }
    });

    // リスト内のボタン（除外、発注不可など）のイベントリスナー
    outputContainer.addEventListener('click', async (e) => {
        const target = e.target;
        const row = target.closest('tr');
        if (!row) return;

        // 「発注不可」ボタン
        if (target.classList.contains('set-unorderable-btn')) {
            const productCode = target.dataset.productCode;
            const productName = row.cells[0].textContent;
            if (!confirm(`「${productName}」を発注不可に設定しますか？\nこの品目は今後、不足品リストに表示されなくなります。`)) {
                return;
            }
            window.showLoading('マスターを更新中...');
            try {
                // TKRのAPI (mastermanager.goで定義)
                const res = await fetch('/api/master/set_order_stopped', { 
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({ productCode: productCode, status: 1 }),
                });
                const resData = await res.json();
                if (!res.ok) throw new Error(resData.message || '更新に失敗しました。');
                
                row.classList.add('provisional-order-item');
                row.querySelector('.wholesaler-select').disabled = true;
                row.querySelector('.order-quantity-input').disabled = true;
                target.disabled = true;
                
                window.showNotification(`「${productName}」を発注不可に設定しました。`, 'success');
            } catch(err) {
                window.showNotification(err.message, 'error');
            } finally {
                window.hideLoading();
            }
        } 
        // 「発注に変更」ボタン
        else if (target.classList.contains('change-to-orderable-btn')) {
            row.classList.remove('provisional-order-item');
            row.querySelector('.wholesaler-select').disabled = false;
            row.querySelector('.order-quantity-input').disabled = false;

            target.textContent = '除外';
            target.classList.remove('change-to-orderable-btn');
            target.classList.add('remove-order-item-btn');
        } 
        // 「除外」ボタン
        else if (target.classList.contains('remove-order-item-btn')) {
            const tbody = row.closest('tbody');
            const table = tbody.closest('table');
            const wrapper = table.closest('.order-yj-group-wrapper');
            row.remove();
            
            if (tbody.children.length === 0 && wrapper) {
                wrapper.remove();
            }
        }
    });

    console.log("Reorder View Initialized (Replaced).");
}

/**
 * 発注点リストのデータを取得して描画する (旧 reorder.js の関数を置き換え)
 */
export async function fetchAndRenderReorder() {
    if (outputContainer) {
        outputContainer.innerHTML = '<p>「発注候補を作成」ボタンを押してください。</p>';
    }
    if (kanaNameInput) kanaNameInput.value = '';
    if (dosageFormInput) dosageFormInput.value = '';
    if (shelfNumberInput) shelfNumberInput.value = '';
    if (coefficientInput) coefficientInput.value = '1.3';
}