import { getLocalDateString } from './utils.js';
import { generateFullHtml, createFinalInputRow } from './inventory_adjustment_ui.js';

let outputContainer;
let lastLoadedDataCache = null;
let currentYjCode = null;

function updatePrecompTotalDisplay() {
    let total = 0;
    document.querySelectorAll('.precomp-active-check:checked').forEach(cb => {
        total += parseFloat(cb.dataset.quantity) || 0;
    });
    const totalEl = document.getElementById('precomp-active-total');
    if (totalEl) {
        totalEl.textContent = `有効合計: ${total.toFixed(2)}`;
    }
}

function findMaster(productCode) {
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

function reverseCalculateStock() {
    const todayStr = getLocalDateString().replace(/-/g, '');
    const precompTotalsByProduct = {};
    const calculationErrorByProduct = {};
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

    const todayNetChangeByProduct = {};
    if (lastLoadedDataCache && lastLoadedDataCache.transactionLedger) {
        lastLoadedDataCache.transactionLedger.forEach(yjGroup => {
            if (yjGroup.packageLedgers) {
                yjGroup.packageLedgers.forEach(pkg => {
                    if (pkg.transactions) {
                        pkg.transactions.forEach(tx => {
      
                            if (tx.transactionDate === todayStr && tx.flag !== 0) {
                                let janQty = tx.janQuantity || 0;
                         
                                 if (janQty === 0 && tx.yjQuantity) {
                                    if (tx.janPackInnerQty > 0) {
                                     janQty = tx.yjQuantity / tx.janPackInnerQty;
     
                                    } else if (tx.yjQuantity !== 0) {
                                     calculationErrorByProduct[tx.janCode] = '包装数量(内)未設定';
    }
                                }
          
                                 const signedJanQty = janQty * (tx.flag === 1 ?
 1 : (tx.flag === 3 ? -1 : 0));
                                todayNetChangeByProduct[tx.janCode] = (todayNetChangeByProduct[tx.janCode] || 0) + signedJanQty;
                            }
                        });
                    }
                });
            }
        });
    }
    document.querySelectorAll('.physical-stock-input').forEach(input => {
        const productCode = input.dataset.productCode;
        const displaySpan = document.querySelector(`.calculated-previous-day-stock[data-product-code="${productCode}"]`);
        const finalInput = document.querySelector(`.final-inventory-input[data-product-code="${productCode}"]`); 
        if (calculationErrorByProduct[productCode]) {
            if (displaySpan) displaySpan.innerHTML = `<span class="status-error">${calculationErrorByProduct[productCode]}</span>`;
         
            if (finalInput) 
            finalInput.value = '';
        
            updateFinalInventoryTotal(productCode);
            return;
        }
        const physicalStockToday = parseFloat(input.value) || 0;

        const precompStock = precompTotalsByProduct[productCode] || 0;
        const totalStockToday = physicalStockToday + precompStock;
     
        const netChangeToday = todayNetChangeByProduct[productCode] || 0;
        const calculatedPreviousDayStock = totalStockToday - netChangeToday;
        if (displaySpan) displaySpan.textContent = calculatedPreviousDayStock.toFixed(2);
    });
}

function updateFinalInventoryTotal(productCode) {
    const tbody = document.querySelector(`.final-input-tbody[data-product-code="${productCode}"]`);
    if (!tbody) return;
    let totalQuantity = 0;
    tbody.querySelectorAll('.final-inventory-input, .lot-quantity-input').forEach(input => { 
        totalQuantity += parseFloat(input.value) || 0;
    });
    const physicalStockInput = document.querySelector(`.physical-stock-input[data-product-code="${productCode}"]`);
    if (physicalStockInput) {
        physicalStockInput.value = totalQuantity.toFixed(2);
    }

    reverseCalculateStock();
}

async function saveInventoryData() {
    const dateInput = document.getElementById('inventory-date');
    if (!dateInput || !dateInput.value) {
        window.showNotification('棚卸日を指定してください。', 'error');
        return;
    }
    if (!confirm(`${dateInput.value}の棚卸データとして保存します。よろしいですか？`)) return;
    const inventoryData = {};
    const deadStockData = [];
    if (!lastLoadedDataCache || !lastLoadedDataCache.transactionLedger || lastLoadedDataCache.transactionLedger.length === 0) {
        window.showNotification('保存対象の品目データが見つかりません。', 'error');
        return;
    }
    const allMasters = (lastLoadedDataCache.transactionLedger[0].packageLedgers || []).flatMap(pkg => pkg.masters || []);
    allMasters.forEach(master => {
        const productCode = master.productCode;
        const tbody = document.querySelector(`.final-input-tbody[data-product-code="${productCode}"]`);
        if (!tbody) {
            inventoryData[productCode] = 0;
            return;
        };
        let totalInputQuantity = 0;
        const inventoryRows = tbody.querySelectorAll('.inventory-row');
        for (let 
 
i = 0; i < inventoryRows.length; i += 2) {
            const topRow = inventoryRows[i];
            const bottomRow = inventoryRows[i+1];
            const quantityInput = bottomRow.querySelector('.final-inventory-input, .lot-quantity-input');
            const expiryInput = topRow.querySelector('.expiry-input');
            const lotInput = bottomRow.querySelector('.lot-input');
            if (!quantityInput 
 || 
!expiryInput || !lotInput) continue;
            const quantity = parseFloat(quantityInput.value) || 0;
            const expiry = expiryInput.value.trim();
            const lot = lotInput.value.trim();
            totalInputQuantity += quantity;

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
        inventoryData[productCode] = totalInputQuantity;
    });
    const payload = {
        date: dateInput.value.replace(/-/g, ''),
        yjCode: currentYjCode,
        inventoryData: inventoryData,
        deadStockData: deadStockData,
    };
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
        loadAndRenderDetails(currentYjCode);
    } catch (err) {
        console.error("Failed to save inventory data:", err);
        window.showNotification(err.message, 'error');
    } finally {
        window.hideLoading();
    }
}

export async function loadAndRenderDetails(yjCode) {
    currentYjCode = yjCode;
    if (!yjCode) {
        window.showNotification('YJコードを指定してください。', 'error');
        return;
    }
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
        
        lastLoadedDataCache = await res.json();
        const html = generateFullHtml(lastLoadedDataCache, lastLoadedDataCache);
        outputContainer.innerHTML = html;
        
        const dateInput = document.getElementById('inventory-date');
        if(dateInput) {
            dateInput.value = getLocalDateString();
        }
        
        document.querySelectorAll('.final-input-tbody').forEach(tbody => {
            const productCode = tbody.dataset.productCode;
            if (productCode) {
                updateFinalInventoryTotal(productCode);
            }
        });
        reverseCalculateStock();
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
            reverseCalculateStock();
        }

        if(targetClassList.contains('lot-quantity-input') || targetClassList.contains('final-inventory-input')){
            const productCode = e.target.dataset.productCode;
            updateFinalInventoryTotal(productCode); 
        }
    });

    outputContainer.addEventListener('click', (e) => {
        const target = e.target;
        if (target.classList.contains('add-deadstock-row-btn')) {
            const productCode = target.dataset.productCode;
            const master = findMaster(productCode);
            const tbody = document.querySelector(`.final-input-tbody[data-product-code="${productCode}"]`);
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
            if(productCode) updateFinalInventoryTotal(productCode);
        }
        if (target.classList.contains('register-inventory-btn')) {
            saveInventoryData();
        }
    });

    outputContainer.addEventListener('submit', (e) => {
        e.preventDefault();
    });

    console.log("Inventory Adjustment Details Initialized.");
}