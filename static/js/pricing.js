// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\pricing.js
// ▼▼▼【ここから修正】インポートを UI と Events に変更 ▼▼▼
import { 
    initPricingUI, 
    loadWholesalerDropdown, 
    applyFiltersAndRender,
    handleSetLowestPrice,
    handleSupplierChange,
    handleWholesalerSelectChange
} from './pricing_ui.js';
import { 
    initPricingEvents, 
    loadInitialMasters 
} from './pricing_events.js';
// ▲▲▲【修正ここまで】▲▲▲

export function initPricingView() {
    const view = document.getElementById('pricing-view');
if (!view) return;

    // ▼▼▼【ここから修正】UIとイベントの初期化を呼び出す ▼▼▼
    
    // 1. UIモジュールを初期化 (DOM要素キャッシュ)
    initPricingUI();
    
    // 2. イベントモジュールを初期化 (API関連のボタンイベント登録)
    initPricingEvents();
    
    // 3. 'show' イベントでUIを初期化
    view.addEventListener('show', () => 
{
        loadWholesalerDropdown();
        loadInitialMasters();
    });
    
    // 4. UIモジュール内のDOMイベントを登録
    const outputContainer = document.getElementById('pricing-output-container');
    const makerFilterInput = document.getElementById('pricing-maker-filter');
    const unregisteredFilterCheckbox = document.getElementById('pricing-unregistered-filter');
    const setLowestPriceBtn = document.getElementById('set-lowest-price-btn');
    const wholesalerSelect = document.getElementById('pricing-wholesaler-select');

    if (makerFilterInput) {
        makerFilterInput.addEventListener('input', applyFiltersAndRender);
    }
    if (unregisteredFilterCheckbox) {
        unregisteredFilterCheckbox.addEventListener('change', 
applyFiltersAndRender);
    }
    if (outputContainer) {
        outputContainer.addEventListener('change', handleSupplierChange);
    }
    if (setLowestPriceBtn) {
        setLowestPriceBtn.addEventListener('click', handleSetLowestPrice);
    }
    if (wholesalerSelect) {
        wholesalerSelect.addEventListener('change', handleWholesalerSelectChange);
    }
    // ▲▲▲【修正ここまで】▲▲▲
}