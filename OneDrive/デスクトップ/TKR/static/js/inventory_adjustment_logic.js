// C:\Users\wasab\OneDrive\デスケトッフ\TKR\static\js\inventory_adjustment_logic.js
import { hiraganaToKatakana, getLocalDateString, parseBarcode, fetchProductMasterByBarcode } from './utils.js';
import { setUnitMap, generateFullHtml, createFinalInputRow } from './inventory_adjustment_ui.js';
import { showModal } from './search_modal.js';

let view, outputContainer;
let searchBtn, barcodeInput;
let currentYjCode = null;
let lastLoadedDataCache = null;

async function handleBarcodeScan(e) {
    e.preventDefault();
    
    const barcodeInput = document.getElementById('ia-barcode-input');
    const inputValue = barcodeInput.value.trim();
    if (!inputValue) return;

    window.showLoading('製品情報を検索中...');
    try {
        
        const productMaster = await fetchProductMasterByBarcode(inputValue);
        if (productMaster.yjCode !== currentYjCode) {
            
            await loadAndRenderDetails(productMaster.yjCode);
            window.showNotification(`品目を切り替えました: ${productMaster.productName}`, 'success');

        } else {
            
            
            let parsedData = null;
            if (inputValue.length > 14) {
                try {
                    parsedData = parseBarcode(inputValue);
                } catch (err) {
                    window.showNotification(`GS1解析エラー: ${err.message}`, 'warning');
                }
            }
            
            
            const productTbody = outputContainer.querySelector(`.final-input-tbody[data-product-code="${productMaster.productCode}"]`);
            if (!productTbody) {
                throw new Error(`画面内に製品「${productMaster.productName}」の入力欄が見つかりません。`);
            }

            
            let targetRow = null;
            const rows = productTbody.querySelectorAll('tr.inventory-row');
            for (let i = 0; i < rows.length; i += 2) {
                const expiryInput = rows[i].querySelector('.expiry-input');
                const lotInput = rows[i+1].querySelector('.lot-input');
                if (expiryInput.value.trim() === '' && lotInput.value.trim() === '') {
                    targetRow = rows[i];
                    break;
                }
            }

            
            if (!targetRow) {
                const addBtn = productTbody.querySelector('.add-deadstock-row-btn');
                if (addBtn) {
                    addBtn.click();
                    const newRows = productTbody.querySelectorAll('tr.inventory-row');
                    targetRow = newRows[newRows.length - 2];
                }
            }

            
            if (targetRow) {
                if (parsedData && parsedData.expiryDate) {
                    const expiryInput = targetRow.querySelector('.expiry-input');
                    expiryInput.value = parsedData.expiryDate;
                }
                if (parsedData && parsedData.lotNumber) {
                    const lotInput = targetRow.nextElementSibling.querySelector('.lot-input');
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
        }

    } catch (err) {
        window.showNotification(err.message, 'error');
    } finally {
        window.hideLoading();
        barcodeInput.value = '';
        barcodeInput.focus();
    }
}

async function onSelectProductClick() {
    const apiUrl = '/api/products/search_filtered';
// ▼▼▼【修正】IDのハイフン(-)をアンダースコア(_)に変更 ▼▼▼
    const kanaInput = document.getElementById('ia_search-kana');
    const genericInput = document.getElementById('ia_search-generic');
    const shelfInput = document.getElementById('ia_search-shelf');
    // ▲▲▲【修正ここまで】▲▲▲
    const selectedUsageRadio = document.querySelector('input[name="ia_usage_class"]:checked');
    
    const kanaName = kanaInput ? hiraganaToKatakana(kanaInput.value.trim()) : '';
    const genericName = genericInput ? genericInput.value.trim() : '';
    const shelfNumber = shelfInput ? shelfInput.value.trim() : '';
    const usageClass = selectedUsageRadio ? selectedUsageRadio.value : '';

    if (!usageClass) {
        window.showNotification('内外注区分を選択してください。', 'warning');
        return;
    }

    const params = new URLSearchParams();
    params.append('kanaName', kanaName);
    params.append('genericName', genericName);
    params.append('shelfNumber', shelfNumber);
    params.append('dosageForm', usageClass);

    window.showLoading('品目リストを検索中...');
    let products = [];
    try {
        const fullUrl = `${apiUrl}?${params.toString()}`;
        const res = await fetch(fullUrl);
        if (!res.ok) {
            throw new Error(`品目リストの取得に失敗しました: ${res.status}`);
        }
        products = await res.json();
    } catch (err) {
        window.hideLoading();
        window.showNotification(err.message, 'error');
        return;
    } finally {
        window.hideLoading();
    }

    showModal(
        view, 
        (selectedProduct) => { 
            loadAndRenderDetails(selectedProduct.yjCode);
        }, 
        { 
            initialResults: products, 
        }
    );
}

async function loadAndRenderDetails(yjCode) {
    currentYjCode = yjCode;
    if (!yjCode) {
        window.showNotification('YJコードを指定してください。', 'error');
        return;
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
    } catch (err) {
        outputContainer.innerHTML = `<p style="color:red;">エラー: ${err.message}</p>`;
    } finally {
        window.hideLoading();
    }
}

function reverseCalculateStock() {
    const todayStr = getLocalDateString().replace(/-/g, '');
    const calculationErrorByProduct = {};
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
// ▼▼▼【修正】style 属性を削除 ▼▼▼
            if (displaySpan) displaySpan.innerHTML = `<span class="status-error">${calculationErrorByProduct[productCode]}</span>`;
            // ▲▲▲【修正ここまで】▲▲▲
            if (finalInput) finalInput.value = '';
        
            updateFinalInventoryTotal(productCode);
            return;
        }
        const physicalStockToday = parseFloat(input.value) || 0;
        const totalStockToday = physicalStockToday;
        const netChangeToday = todayNetChangeByProduct[productCode] || 0;
        const calculatedPreviousDayStock = totalStockToday - netChangeToday;
        if (displaySpan) displaySpan.textContent = calculatedPreviousDayStock.toFixed(2);
        if (finalInput) {
            finalInput.value = calculatedPreviousDayStock.toFixed(2);
            updateFinalInventoryTotal(productCode);
        }
    });
}

function updateFinalInventoryTotal(productCode) {
    const tbody = document.querySelector(`.final-input-tbody[data-product-code="${productCode}"]`);
    if (!tbody) return;
    let totalQuantity = 0;
    tbody.querySelectorAll('.final-inventory-input, .lot-quantity-input').forEach(input => {
        totalQuantity += parseFloat(input.value) || 0;
    });
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
            if (!quantityInput || 
!expiryInput || !lotInput) continue;
            const quantity = parseFloat(quantityInput.value) || 0;
            const expiry = expiryInput.value.trim();
            const lot = lotInput.value.trim();
            totalInputQuantity += quantity;
            if (quantity > 0 && (expiry || lot)) {
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
        inventoryData[productCode] = totalInputQuantity;
    });
    const payload = {
        date: dateInput.value.replace(/-/g, ''),
        yjCode: currentYjCode,
        inventoryData: inventoryData,
        deadStockData: deadStockData,
    };
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
        window.showNotification(err.message, 'error');
    } finally {
        window.hideLoading();
    }
}

export async function initInventoryAdjustment() {
    let localUnitMap = {};
    try {
        const res = await fetch('/api/units/map');
        if (!res.ok) throw new Error('単位マスタの取得に失敗');
        localUnitMap = await res.json();
        setUnitMap(localUnitMap);
    } catch (err) {
        console.error(err);
        window.showNotification(err.message, 'error');
    }

    view = document.getElementById('inventory-adjustment-view');
    if (!view) {
        console.error("Inventory Adjustment View not found.");
        return;
    }
    
    searchBtn = document.getElementById('ia-search-btn');
    outputContainer = document.getElementById('inventory-adjustment-output');
    barcodeInput = document.getElementById('ia-barcode-input');
    const barcodeForm = document.getElementById('ia-barcode-form');
    
    if (barcodeForm) {
        barcodeForm.addEventListener('submit', handleBarcodeScan);
    }
    if (searchBtn) {
        searchBtn.addEventListener('click', onSelectProductClick);
    }

    outputContainer.addEventListener('input', (e) => {
        const targetClassList = e.target.classList;
        if (targetClassList.contains('physical-stock-input')) {
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
                const 
newRowHTML = createFinalInputRow(master, null, false);
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
    document.addEventListener('loadInventoryAdjustment', (e) => {
        const { yjCode } = e.detail;
        if (yjCode) {
            loadAndRenderDetails(yjCode);
        }
    });
    console.log("Inventory Adjustment View Initialized.");
}