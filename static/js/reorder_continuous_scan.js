// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\reorder_continuous_scan.js
import { fetchProductMasterByBarcode } from './utils.js';
import { addOrUpdateOrderItem } from './reorder_ui.js';

// モジュールレベル変数
let scanQueue = [];
let isProcessingQueue = false;

// DOM要素 (initで設定)
let continuousOrderModal, closeContinuousModalBtn;
let continuousBarcodeForm, continuousBarcodeInput, scannedItemsList, scannedItemsCount, processingIndicator;

/**
 * 連続スキャン用: キューの表示を更新
 * (旧 reorder.js より移管)
 */
function updateScannedItemsDisplay() {
    const counts = scanQueue.reduce((acc, code) => {
        acc[code] = (acc[code] || 0) + 1;
        return acc;
    }, {});

    if (scannedItemsList) {
        scannedItemsList.innerHTML = Object.entries(counts).map(([code, count]) => {
            return `<div class="scanned-item">
                        <span class="scanned-item-name">${code}</span>
                        <span class="scanned-item-count">x ${count}</span>
                    </div>`;
        }).join('');
    }
    if (scannedItemsCount) {
        scannedItemsCount.textContent = scanQueue.length;
    }
}

/**
 * 連続スキャン用: スキャンキューを処理
 * (旧 reorder.js より移管)
 */
async function processScanQueue() {
    if (isProcessingQueue) return;

    isProcessingQueue = true;
    if (processingIndicator) processingIndicator.classList.remove('hidden');
    
    while (scanQueue.length > 0) {
        const barcode = scanQueue.shift();
        try {
            const productMaster = await fetchProductMasterByBarcode(barcode); // from utils.js
            addOrUpdateOrderItem(productMaster); // from reorder_ui.js
        } catch (err) {
            console.error(`バーコード[${barcode}]の処理に失敗:`, err);
            window.showNotification(`バーコード[${barcode}]の処理に失敗しました: ${err.message}`, 'error');
        } finally {
            updateScannedItemsDisplay();
        }
    }

    isProcessingQueue = false;
    if (processingIndicator) processingIndicator.classList.add('hidden');
}

/**
 * 連続スキャンモーダルを開く
 * (reorder.js から呼び出される)
 */
export function openContinuousScanModal() {
    scanQueue = [];
    updateScannedItemsDisplay();
    if (continuousOrderModal) {
        continuousOrderModal.classList.remove('hidden');
        document.body.classList.add('modal-open');
        setTimeout(() => continuousBarcodeInput?.focus(), 100);
    }
}

/**
 * 連続スキャンモーダルのイベントを初期化
 * (旧 reorder.js より移管)
 */
export function initContinuousScan() {
    // DOM要素の取得
    continuousOrderModal = document.getElementById('continuous-order-modal');
    closeContinuousModalBtn = document.getElementById('close-continuous-modal-btn');
    continuousBarcodeForm = document.getElementById('continuous-barcode-form');
    continuousBarcodeInput = document.getElementById('continuous-barcode-input');
    scannedItemsList = document.getElementById('scanned-items-list');
    scannedItemsCount = document.getElementById('scanned-items-count');
    processingIndicator = document.getElementById('processing-indicator');

    if (!continuousOrderModal || !closeContinuousModalBtn || !continuousBarcodeForm) {
        console.warn("Continuous scan modal elements not fully found.");
        return;
    }

    // モーダルを閉じるボタン
    closeContinuousModalBtn.addEventListener('click', () => {
        continuousOrderModal.classList.add('hidden');
        document.body.classList.remove('modal-open');
    });

    // モーダル内のバーコードスキャンフォーム
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
}