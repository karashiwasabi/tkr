// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\inventory_adjustment_logic.js
import { getLocalDateString } from './utils.js';

let lastLoadedDataCache = null;
let currentYjCode = null;

export function setCache(data) {
    lastLoadedDataCache = data;
}

export function getCache() {
    return lastLoadedDataCache;
}

export function setCurrentYjCode(yjCode) {
    currentYjCode = yjCode;
}

export function getCurrentYjCode() {
    return currentYjCode;
}

/**
 * キャッシュからマスターデータを検索します。
 */
export function findMaster(productCode) {
    if (!lastLoadedDataCache || !lastLoadedDataCache.transactionLedger || lastLoadedDataCache.transactionLedger.length === 0) {
        return null;
    }
    for (const pkgLedger of lastLoadedDataCache.transactionLedger[0].packageLedgers) {
        const masterView = (pkgLedger.masters || []).find(m => m.productCode === productCode);
        if (masterView) {
            return masterView;
        }
    }
    return null;
}

/**
 * 予製の有効合計を更新します。
 */
export function updatePrecompTotalDisplay() {
    let total = 0;
    document.querySelectorAll('.precomp-active-check:checked').forEach(cb => {
        total += parseFloat(cb.dataset.quantity) || 0;
    });
    const totalEl = document.getElementById('precomp-active-total');
    if (totalEl) {
        totalEl.textContent = `有効合計: ${total.toFixed(2)}`;
    }
}

/**
 * ①実在庫入力 + ②予製チェック + ③本日の入出庫 から、④前日在庫(逆算値)を計算します。
 */
export function reverseCalculateStock() {
    const todayStr = getLocalDateString().replace(/-/g, '');
    const precompTotalsByProduct = {};
    const calculationErrorByProduct = {};

    // 1. 画面上の予製チェックボックスから、本日有効な予製数を集計
    document.querySelectorAll('.precomp-active-check:checked').forEach(cb => {
        const productCode = cb.dataset.productCode;
        const master = findMaster(productCode);
        if (!master) return;
        const yjQuantity = parseFloat(cb.dataset.quantity) || 0;

        if (master.janPackInnerQty > 0) {
            const janQuantity = yjQuantity / master.janPackInnerQty;
            precompTotalsByProduct[productCode] = (precompTotalsByProduct[productCode] || 0) + janQuantity;
        } else if (yjQuantity > 0) {
            calculationErrorByProduct[productCode] = '包装数量(内)未設定';
        }
    });

    updatePrecompTotalDisplay();

    // 2. キャッシュされている取引履歴から、本日の変動数（NetChange）を集計
    const todayNetChangeByProduct = {};
    if (lastLoadedDataCache && lastLoadedDataCache.transactionLedger) {
        lastLoadedDataCache.transactionLedger.forEach(yjGroup => {
            if (yjGroup.packageLedgers) {
                yjGroup.packageLedgers.forEach(pkg => {
                    if (pkg.transactions) {
                        pkg.transactions.forEach(tx => {
                            // 日付が本日で、かつ棚卸(0)ではないレコードが対象
                            if (tx.transactionDate === todayStr && tx.flag !== 0) {

                                let janQty = tx.janQuantity || 0;
                                // JAN数量がなくYJ数量のみある場合の換算
                                if (janQty === 0 && tx.yjQuantity) {
                                    if (tx.janPackInnerQty > 0) {
                                        janQty = tx.yjQuantity / tx.janPackInnerQty;
                                    } else if (tx.yjQuantity !== 0) {
                                        calculationErrorByProduct[tx.janCode] = '包装数量(内)未設定';
                                    }
                                }

                                // 符号判定
                                let sign = 0;
                                // 在庫が増える取引: 納品(1), 入庫(11)
                                if (tx.flag === 1 || tx.flag === 11) {
                                    sign = 1;
                                }
                                // 在庫が減る取引: 返品(2), 処方(3), 出庫(12)
                                else if (tx.flag === 2 || tx.flag === 3 || tx.flag === 12) {
                                    sign = -1;
                                }

                                const signedJanQty = janQty * sign;
                                todayNetChangeByProduct[tx.janCode] = (todayNetChangeByProduct[tx.janCode] || 0) + signedJanQty;
                            }
                        });
                    }
                });
            }
        });
    }

    // 3. 画面表示の更新: 実在庫 - 変動 = 前日在庫（理論値）
    document.querySelectorAll('.physical-stock-display').forEach(displaySpan => {
        const productCode = displaySpan.dataset.productCode;
        const calculatedSpan = document.querySelector(`.calculated-previous-day-stock[data-product-code="${productCode}"]`);

        if (calculationErrorByProduct[productCode]) {
            if (calculatedSpan) calculatedSpan.innerHTML = `<span class="status-error">${calculationErrorByProduct[productCode]}</span>`;
            updateFinalInventoryTotal(productCode);
            return;
        }

        // ① 実在庫（ユーザー入力値）
        const physicalStockToday = parseFloat(displaySpan.textContent) || 0;
        
        // ② 予製分
        const precompStock = precompTotalsByProduct[productCode] || 0;
        
        // 合計在庫（本日時点）
        const totalStockToday = physicalStockToday + precompStock;

        // ③ 本日の変動
        const netChangeToday = todayNetChangeByProduct[productCode] || 0;

        // ④ 前日在庫（逆算値）
        const calculatedPreviousDayStock = totalStockToday - netChangeToday;

        if (calculatedSpan) calculatedSpan.textContent = calculatedPreviousDayStock.toFixed(2);
    });
}

/**
 * ロット・期限入力の合計を「① 実在庫数量」に反映し、理論在庫との不一致をチェックします。
 */
export function updateFinalInventoryTotal(productCode) {
    const tbody = document.querySelector(`.final-input-tbody[data-product-code="${productCode}"]`);
    if (!tbody) return;
    let totalQuantity = 0;
    tbody.querySelectorAll('.final-inventory-input, .lot-quantity-input').forEach(input => {
        totalQuantity += parseFloat(input.value) || 0;
    });
    
    const physicalStockDisplay = document.querySelector(`.physical-stock-display[data-product-code="${productCode}"]`);
    if (physicalStockDisplay) {
        // 合計を表示
        physicalStockDisplay.textContent = totalQuantity.toFixed(2);

        // 理論在庫との比較 (不一致なら緑色、一致ならグレー)
        const theoretical = parseFloat(physicalStockDisplay.dataset.theoretical) || 0;
        if (Math.abs(totalQuantity - theoretical) > 0.01) {
            physicalStockDisplay.classList.add('stock-mismatch'); // 緑になる
        } else {
            physicalStockDisplay.classList.remove('stock-mismatch'); // グレーに戻る
        }
    }

    // 実在庫が変わったので、逆算値を再計算
    reverseCalculateStock();
}