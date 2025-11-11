// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\inventory_adjustment_search.js
import { hiraganaToKatakana, parseBarcode, fetchProductMasterByBarcode } from './utils.js';
import { showModal } from './search_modal.js';

let searchBtn;
let view, outputContainer;
let currentYjCode = null;

async function handleBarcodeScan(e, loadAndRenderDetailsCallback) {
    e.preventDefault();
    
    // ▼▼▼【ここから修正】グローバル変数(barcodeInput)ではなく、イベント(e)から入力要素を取得 ▼▼▼
    const currentBarcodeForm = e.target;
    const currentBarcodeInput = currentBarcodeForm.querySelector('.search-barcode-input'); // フォーム内の入力欄
    if (!currentBarcodeInput) return;

    const inputValue = currentBarcodeInput.value.trim();
    // ▲▲▲【修正ここまで】▲▲▲
    
    if (!inputValue) return;

    window.showLoading('製品情報を検索中...');
    try {
        const productMaster = await fetchProductMasterByBarcode(inputValue);
        
        if (productMaster.yjCode !== currentYjCode) {
            // YJコードが異なる場合、画面を再描画
            currentYjCode = productMaster.yjCode;
            await loadAndRenderDetailsCallback(productMaster.yjCode); 
            window.showNotification(`品目を切り替えました: ${productMaster.productName}`, 'success');
        
        } else {
            // YJコードが同じ場合、既存の画面に入力
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
        window.showNotification(`エラー: ${err.message}`, 'error');
    } finally {
        window.hideLoading();
        
        // ▼▼▼【ここから修正】再描画後も動作するように、IDで入力欄を再取得して操作する ▼▼▼
        const inputToClear = document.getElementById('ia-barcode-input'); // IDで再取得
        if (inputToClear) {
            inputToClear.value = '';
            inputToClear.focus();
        }
        // ▲▲▲【修正ここまで】▲▲▲
    }
}

async function onSelectProductClick(loadAndRenderDetailsCallback) {
    const apiUrl = '/api/products/search_filtered';
    const kanaInput = document.getElementById('ia_search-kana');
    const genericInput = document.getElementById('ia_search-generic');
    const shelfInput = document.getElementById('ia_search-shelf');
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
            currentYjCode = selectedProduct.yjCode;
            loadAndRenderDetailsCallback(selectedProduct.yjCode);
        }, 
        { 
            initialResults: products, 
        }
    );
}

export function setCurrentYjCode(yjCode) {
    currentYjCode = yjCode;
}

export function initSearchForm(loadAndRenderDetailsCallback) {
    view = document.getElementById('inventory-adjustment-view');
    outputContainer = document.getElementById('inventory-adjustment-output');
    searchBtn = document.getElementById('ia-search-btn');
    // barcodeInput = document.getElementById('ia-barcode-input'); // グローバル変数へのキャッシュを削除
    const barcodeForm = document.getElementById('ia-barcode-form');
    if (barcodeForm) {
        barcodeForm.addEventListener('submit', (e) => handleBarcodeScan(e, loadAndRenderDetailsCallback));
    }
    if (searchBtn) {
        searchBtn.addEventListener('click', () => onSelectProductClick(loadAndRenderDetailsCallback));
    }
}