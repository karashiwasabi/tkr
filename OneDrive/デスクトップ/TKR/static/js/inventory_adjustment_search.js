// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\inventory_adjustment_search.js
import { hiraganaToKatakana, parseBarcode, fetchProductMasterByBarcode } from './utils.js';
import { showModal } from './search_modal.js';
// ▼▼▼【ここに追加】logic.js から YjCode 管理関数をインポート ▼▼▼
import { setCurrentYjCode, getCurrentYjCode } from './inventory_adjustment_logic.js';
// ▲▲▲【追加ここまで】▲▲▲

let searchBtn;
let view, outputContainer;
// ▼▼▼【削除】currentYjCode を logic.js に移管 ▼▼▼
// let currentYjCode = null;
// ▲▲▲【削除ここまで】▲▲▲

async function handleBarcodeScan(e, loadAndRenderDetailsCallback) {
    e.preventDefault();
    
    // ▼▼▼【ここから修正】グローバル変数(barcodeInput)ではなく、イベント(e)から入力要素を取得 ▼▼▼
    const currentBarcodeForm = e.target;
    const currentBarcodeInput = currentBarcodeForm.querySelector('.search-barcode-input');
// フォーム内の入力欄
 
   if (!currentBarcodeInput) return;

    const inputValue = currentBarcodeInput.value.trim();
    // ▲▲▲【修正ここまで】▲▲▲
    
    if (!inputValue) return;

    window.showLoading('製品情報を検索中...');
    try 
{
   
     const productMaster = await fetchProductMasterByBarcode(inputValue);
        
        // ▼▼▼【ここから修正】YJコードが同じでも必ずリロードするように変更 + logic.js を使用 ▼▼▼
        if (productMaster.yjCode !== getCurrentYjCode()) {
           
 window.showNotification(`品目を切り替えました: ${productMaster.productName}`, 'success');
        }
        setCurrentYjCode(productMaster.yjCode);
    await loadAndRenderDetailsCallback(productMaster.yjCode); 
        // ▲▲▲【修正ここまで】▲▲▲

    } catch (err) {
window.showNotification(`エラー: ${err.message}`, 
'error');
    } finally {
        window.hideLoading();
        
        // ▼▼▼【ここから修正】再描画後も動作するように、IDで入力欄を再取得して操作する ▼▼▼
        const inputToClear 
= document.getElementById('ia-barcode-input'); // 
IDで再取得
        if (inputToClear) {
            inputToClear.value 
= '';
            inputToClear.focus();
        }
 
       // ▲▲▲【修正ここまで】▲▲▲
    }
}

async function onSelectProductClick(loadAndRenderDetailsCallback) {
    const apiUrl = 
'/api/products/search_filtered';
    const kanaInput = document.getElementById('ia_search-kana');
const genericInput = document.getElementById('ia_search-generic');
    const shelfInput = document.getElementById('ia_search-shelf');
    const selectedUsageRadio = document.querySelector('input[name="ia_usage_class"]:checked');
    
    const kanaName = kanaInput ? hiraganaToKatakana(kanaInput.value.trim()) : '';
    const 
genericName = genericInput ? genericInput.value.trim() : 
'';
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
    params.append('dosageForm', 
usageClass);

    window.showLoading('品目リストを検索中...');
    let products = [];
    try {
 
       const fullUrl = `${apiUrl}?${params.toString()}`;
        const res = await fetch(fullUrl);
        if (!res.ok) 
{
         
   throw new Error(`品目リストの取得に失敗しました: ${res.status}`);
        }
        products = 
await res.json();
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
            // ▼▼▼【ここを修正】logic.js の setCurrentYjCode を使用 ▼▼▼
            setCurrentYjCode(selectedProduct.yjCode);
            // ▲▲▲【修正ここまで】▲▲▲
            loadAndRenderDetailsCallback(selectedProduct.yjCode);
        }, 
    
    { 
        
    initialResults: products, 
        
}
    );
}

// ▼▼▼【削除】setCurrentYjCode を logic.js に移管 ▼▼▼
/*
export function setCurrentYjCode(yjCode) {
    currentYjCode = yjCode;
}
*/
// ▲▲▲【削除ここまで】▲▲▲

export function initSearchForm(loadAndRenderDetailsCallback) {
    view = document.getElementById('inventory-adjustment-view');
    outputContainer = 
document.getElementById('inventory-adjustment-output');
    searchBtn = document.getElementById('ia-search-btn');
    // barcodeInput = document.getElementById('ia-barcode-input'); // グローバル変数へのキャッシュを削除
  
  const barcodeForm = document.getElementById('ia-barcode-form');
    if (barcodeForm) {
        
barcodeForm.addEventListener('submit', (e) => handleBarcodeScan(e, loadAndRenderDetailsCallback));
    }
    if 
(searchBtn) {
        searchBtn.addEventListener('click', () => onSelectProductClick(loadAndRenderDetailsCallback));
    }
}