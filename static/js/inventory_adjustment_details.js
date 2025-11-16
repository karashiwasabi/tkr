// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\inventory_adjustment_details.js
import { getLocalDateString, parseBarcode, fetchProductMasterByBarcode } from './utils.js';
import { generateFullHtml, createFinalInputRow } from './inventory_adjustment_ui.js';
// ▼▼▼【ここから追加】logic.js から関数をインポート ▼▼▼
import {
    setCache,
    getCache,
    setCurrentYjCode,
    getCurrentYjCode,
    findMaster,
    reverseCalculateStock,
    updateFinalInventoryTotal
} from './inventory_adjustment_logic.js';
// ▲▲▲【追加ここまで】▲▲▲

let outputContainer;

/**
 * 「2. 棚卸入力」セクションのバーコードスキャンを処理します。
 */
async function handleAdjustmentBarcodeScan(e) {
    e.preventDefault();
    const barcodeInput = document.getElementById('adjustment-barcode-input');
    const inputValue = barcodeInput.value.trim();
    if (!inputValue) return;

    let parsedData;
    try {
        parsedData = parseBarcode(inputValue);
    } catch (err) {
        window.showNotification(`バーコード解析エラー: 
${err.message}`, 'error');
        barcodeInput.value = '';
        return;
    }

    if (!parsedData.gtin14) {
        window.showNotification('GS1-128形式のバーコードではありません。', 'error');
        barcodeInput.value = '';
        return;
    }

    window.showLoading('製品情報を検索中...');
    try {
        // TKRの utils.js を使用
        const productMaster = await fetchProductMasterByBarcode(inputValue);

        // ▼▼▼【ここを修正】logic.js の getCurrentYjCode を使用 ▼▼▼
        if (productMaster.yjCode !== getCurrentYjCode()) {
            
throw new Error(`スキャンされた品目(${productMaster.productName})は、現在表示中のYJコード(${getCurrentYjCode()})と異なります。`);
        }
        // ▲▲▲【修正ここまで】▲▲▲

        const productTbody = outputContainer.querySelector(`.final-input-tbody[data-product-code="${productMaster.productCode}"]`);
        if (!productTbody) {
            
throw new Error(`画面内に製品「${productMaster.productName}」の入力欄が見つかりません。`);
        }

        let targetRow = null;
        const rows = productTbody.querySelectorAll('tr.inventory-row');
        // 空の行を探す
        for (let i = 0; i < rows.length; i += 2) {
            const 
expiryInput = rows[i].querySelector('.expiry-input');
            const lotInput = rows[i+1].querySelector('.lot-input');
            if (expiryInput.value.trim() === '' && lotInput.value.trim() === '') {
                targetRow = rows[i];
                break;
            }
        }

        // 空の行がなければ新しい行を追加
        if (!targetRow) {
            const addBtn = productTbody.querySelector('.add-deadstock-row-btn');
            if (addBtn) {
                addBtn.click();
                const newRows = productTbody.querySelectorAll('tr.inventory-row');
                targetRow = newRows[newRows.length - 2]; // 追加された行
            }
        }

        
if (targetRow) {
            const expiryInput = targetRow.querySelector('.expiry-input');
            const lotInput = targetRow.nextElementSibling.querySelector('.lot-input');
            
            if (parsedData.expiryDate) { // YYYYMM 形式
                expiryInput.value = parsedData.expiryDate;
            }
            if (parsedData.lotNumber) {
                lotInput.value = parsedData.lotNumber;
            }
            
window.showNotification('ロット・期限を自動入力しました。', 'success');
            
            const qtyInput = targetRow.nextElementSibling.querySelector('.lot-quantity-input');
            if(qtyInput) {
                qtyInput.focus();
                qtyInput.select();
            }
        } else {
            throw new Error('ロット・期限の入力欄の追加に失敗しました。');
        }

    } catch (err) {
        window.showNotification(err.message, 'error');
    } finally {
        window.hideLoading();
        barcodeInput.value = '';
        barcodeInput.focus();
    }
}

// (前回修正した saveInventoryData 関数)
async function saveInventoryData() {
    const dateInput = document.getElementById('inventory-date');
    if (!dateInput || !dateInput.value) {
        window.showNotification('棚卸日を指定してください。', 'error');
        return;
    }
    if (!confirm(`${dateInput.value}の棚卸データとして保存します。よろしいですか？`)) return;

    // ▼▼▼【ここから修正】理論在庫(inventoryData)と物理明細(deadStockData)を正しく収集する ▼▼▼
    
    const inventoryData = {}; // Key: ProductCode, Value: JAN理論在庫合計 (逆算値)
    const deadStockData = []; // Key: ProductCode, Value: JAN物理在庫明細 (ロット/期限)
    
    // ▼▼▼【修正】logic.js の getCache を使用 ▼▼▼
    const cache = getCache();
    if (!cache || !cache.transactionLedger || cache.transactionLedger.length === 0) {
        window.showNotification('保存対象の品目データが見つかりません。', 'error');
        return;
    }
    const allMasters = (cache.transactionLedger[0].packageLedgers || []).flatMap(pkg => pkg.masters || []);
    // ▲▲▲【修正ここまで】▲▲▲

    allMasters.forEach(master => {
        const productCode = master.productCode;
        const tbody = document.querySelector(`.final-input-tbody[data-product-code="${productCode}"]`);

        // 1. 物理在庫明細 (deadStockData) の収集 (変更なし)
        // ユーザーが入力したロット/期限/数量（＝実在庫）を収集
        if (tbody) {
            const inventoryRows = tbody.querySelectorAll('.inventory-row');
      
            for (let i = 0; i < inventoryRows.length; i += 2) {
                const topRow = inventoryRows[i];
                const bottomRow = inventoryRows[i+1];
                const quantityInput = bottomRow.querySelector('.final-inventory-input, .lot-quantity-input');
                const expiryInput = topRow.querySelector('.expiry-input');
                const lotInput = bottomRow.querySelector('.lot-input');
                
                if (!quantityInput || !expiryInput || !lotInput) continue;
                
                const quantity = parseFloat(quantityInput.value) || 0;
                const expiry = expiryInput.value.trim();
                const lot = lotInput.value.trim();

                deadStockData.push({
                    productCode: productCode,
                    yjCode: master.yjCode,
                    packageForm: master.packageForm,
                    janPackInnerQty: master.janPackInnerQty,
                    yjUnitName: master.yjUnitName,
                    stockQuantityJan: quantity,
                    expiryDate: expiry,
                    lotNumber: lot
                });
            }
        }

        // 2. 理論在庫合計 (inventoryData) の収集
        // 「④ 前日在庫(逆算値)」SPAN から値を取得
        const calculatedStockSpan = document.querySelector(`.calculated-previous-day-stock[data-product-code="${productCode}"]`);
        if (calculatedStockSpan) {
            const theoreticalStock = parseFloat(calculatedStockSpan.textContent) || 0;
            inventoryData[productCode] = theoreticalStock;
        } else {
            // スパンが見つからない場合 (画面に表示されていない？)
            // 念のため、物理在庫合計をフォールバックとして使う
            let totalInputQuantity = 0;
            if (tbody) {
                tbody.querySelectorAll('.final-inventory-input, .lot-quantity-input').forEach(input => {
                    totalInputQuantity += parseFloat(input.value) || 0;
                });
            }
            inventoryData[productCode] = totalInputQuantity;
        }
    });
    
    // ▲▲▲【修正ここまで】▲▲▲

    // ▼▼▼【ここを修正】logic.js の getCurrentYjCode を使用 ▼▼▼
    const payload = {
        date: dateInput.value.replace(/-/g, ''),
        yjCode: getCurrentYjCode(),
        inventoryData: inventoryData, // 理論在庫合計 (逆算値)
        deadStockData: deadStockData, // 物理在庫明細 (ロット入力値)
    };
    // ▲▲▲【修正ここまで】▲▲▲
    
    console.log("Saving inventory data. Payload:", payload);

    window.showLoading();
    try {
        const res = await fetch('/api/inventory/adjust/save', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
        });
        const resData = await res.json();
        if (!res.ok) throw new Error(resData.message || '保存に失敗しました。');
        window.showNotification(resData.message, 'success');
        // ▼▼▼【ここを修正】logic.js の getCurrentYjCode を使用 ▼▼▼
        loadAndRenderDetails(getCurrentYjCode());
        // ▲▲▲【修正ここまで】▲▲▲
    } catch (err) {
        console.error("Failed to save inventory data:", err);
        window.showNotification(err.message, 'error');
    } finally {
        window.hideLoading();
    }
}


// ▼▼▼【ここから修正】yjCodeがnullでもHTMLを描画するように修正 ▼▼▼
export async function loadAndRenderDetails(yjCode) {
    // ▼▼▼【ここを修正】logic.js の setCurrentYjCode を使用 ▼▼▼
    setCurrentYjCode(yjCode);
    // ▲▲▲【修正ここまで】▲▲▲
    
    if (!outputContainer) {
        outputContainer = document.getElementById('inventory-adjustment-output');
    }

    // yjCode がない場合（ヘッダーからの直接遷移時）
    if (!yjCode) {
        // ▼▼▼【ここを修正】logic.js の setCache を使用 ▼▼▼
        setCache({
            transactionLedger: [], // 空の台帳
            yesterdaysStock: null,
            deadStockDetails: [],
            precompDetails: []
        });
        // ▲▲▲【修正ここまで】▲▲▲
        
        // ▼▼▼【ここを修正】logic.js の getCache を使用 ▼▼▼
        const html = generateFullHtml(getCache(), getCache());
        // ▲▲▲【修正ここまで】▲▲▲
        outputContainer.innerHTML = html;
        
        // 日付入力欄にデフォルト値（前日）を設定
        const dateInput = document.getElementById('inventory-date');
        if(dateInput) {
            const yesterday = new Date();
yesterday.setDate(yesterday.getDate() - 1);
dateInput.value = getLocalDateString(yesterday);
        }
        return; // fetch は実行しない
    }
    // ▲▲▲【修正ここまで】▲▲▲

    
// yjCode がある場合は、従来通り fetch を実行
    if (!outputContainer) {
       outputContainer = document.getElementById('inventory-adjustment-output');
    }
    
    
 window.showLoading();
    outputContainer.innerHTML = '<p>データを読み込んでいます...</p>';
    try {
 
       const apiUrl = `/api/inventory/adjust/data?yjCode=${yjCode}`;
        const res = await fetch(apiUrl);
        if (!res.ok) {
 
          const errText = await res.text();
            throw new Error(errText || 'データ取得に失敗しました。');
        }
       
 
        // ▼▼▼【ここを修正】logic.js の setCache, getCache を使用 ▼▼▼
        setCache(await res.json());
        const html = generateFullHtml(getCache(), getCache());
        // ▲▲▲【修正ここまで】▲▲▲
        
        outputContainer.innerHTML = html;
        // ▼▼▼【ここから修正】棚卸日を「前日」に設定 ▼▼▼
        const dateInput = document.getElementById('inventory-date');
        if(dateInput) {
            const yesterday = new Date();
            yesterday.setDate(yesterday.getDate() - 1);
dateInput.value = getLocalDateString(yesterday);
        }
        // ▲▲▲【修正ここまで】▲▲▲
     
        
document.querySelectorAll('.final-input-tbody').forEach(tbody => {
            const productCode = tbody.dataset.productCode;
            if (productCode) {
                // ▼▼▼【ここを修正】logic.js の updateFinalInventoryTotal を使用 ▼▼▼
                updateFinalInventoryTotal(productCode);
                // ▲▲▲【修正ここまで】▲▲▲
            }
        });
        // ▼▼▼【ここを修正】logic.js の reverseCalculateStock を使用 ▼▼▼
        reverseCalculateStock();
        // ▲▲▲【修正ここまで】▲▲▲
    } catch (err) {
        outputContainer.innerHTML = `<p style="color:red;">エラー: ${err.message}</p>`;
    } finally {
        window.hideLoading();
    }
}

export function initDetails() {
    outputContainer = document.getElementById('inventory-adjustment-output');
    if (!outputContainer) {
        console.error("Inventory Adjustment output container not found.");
        return;
    }

    outputContainer.addEventListener('input', (e) => {
        const targetClassList = e.target.classList;
        
        if (targetClassList.contains('precomp-active-check')) {
            // ▼▼▼【ここを修正】logic.js の reverseCalculateStock を使用 ▼▼▼
            reverseCalculateStock();
            // ▲▲▲【修正ここまで】▲▲▲
        }

        if(targetClassList.contains('lot-quantity-input') || 
targetClassList.contains('final-inventory-input')){
            const productCode = e.target.dataset.productCode;
            
// ▼▼▼【ここを修正】logic.js の updateFinalInventoryTotal を使用 ▼▼▼
            updateFinalInventoryTotal(productCode); 
            // ▲▲▲【修正ここまで】▲▲▲
        }
    });

    outputContainer.addEventListener('click', (e) => {
        const target = e.target;
        if (target.classList.contains('add-deadstock-row-btn')) {
            const productCode = target.dataset.productCode;
 
            // ▼▼▼【ここを修正】logic.js の findMaster を使用 ▼▼▼
            const master = findMaster(productCode);
            // ▲▲▲【修正ここまで】▲▲▲
            const 
tbody = document.querySelector(`.final-input-tbody[data-product-code="${productCode}"]`);
            if(master && tbody){
                const newRowHTML = createFinalInputRow(master, null, false);
                tbody.insertAdjacentHTML('beforeend', newRowHTML);
            }
        }
       
 if (target.classList.contains('delete-deadstock-row-btn')) {
            const topRow = target.closest('tr');
            const bottomRow = topRow.nextElementSibling;
            const productCode = bottomRow.querySelector('[data-product-code]')?.dataset.productCode;
            topRow.remove();
            bottomRow.remove();
            // ▼▼▼【ここを修正】logic.js の updateFinalInventoryTotal を使用 ▼▼▼
            if(productCode) updateFinalInventoryTotal(productCode);
            // ▲▲▲【修正ここまで】▲▲▲
        }
        if (target.classList.contains('register-inventory-btn')) {
            saveInventoryData();
        }
    });

    outputContainer.addEventListener('submit', (e) => {
        e.preventDefault();
   
        // ▼▼▼【ここから修正】のロジックを修正 ▼▼▼
        if (e.target.id === 'adjustment-barcode-form') {
            handleAdjustmentBarcodeScan(e);
        }
        // ▲▲▲【修正ここまで】▲▲▲
    });
    console.log("Inventory Adjustment Details Initialized.");
}