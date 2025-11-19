// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\inventory_adjustment_search.js
import { hiraganaToKatakana, parseBarcode, fetchProductMasterByBarcode 
} from './utils.js';
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
    
    // ▼▼▼【修正】グローバル変数(barcodeInput)ではなく、イベント(e)から入力要素を取得 ▼▼▼
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
        if (productMaster.yjCode !== 
getCurrentYjCode()) {
           
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

// ▼▼▼【修正】onSelectProductClick を修正 (モーダル内検索に移行) ▼▼▼
async function onSelectProductClick(loadAndRenderDetailsCallback) {
    // 検索フォームから値を取得するロジックは削除し、モーダルに任せる。
    window.showLoading('品目検索モーダルを開いています...');

    showModal(
        view, // activeRowElement としてビュー全体を渡す (後で不要なら null に変更可)
        (selectedProduct) => { 
            // 選択後のコールバック
            setCurrentYjCode(selectedProduct.yjCode);
            loadAndRenderDetailsCallback(selectedProduct.yjCode);
        }, 
        { 
            searchMode: 'default', // ProductMasterのみを検索するモード
            // iaビューは採用済み品目のみを検索するが、採用済みでない品目（JCSHMS）が選択された場合は採用プロセスへ進めるため、allowAdoptedは設定しない（デフォルトの採用フローに乗せる）
        }
    );
    
    window.hideLoading();
}
// ▲▲▲【修正ここまで】▲▲▲

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