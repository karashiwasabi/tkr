// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\backorder.js
import { cacheDOMElements as cacheUI, filterAndRender } from './backorder_ui.js';
import { cacheDOMElements as cacheEvents, loadAndRenderBackorders, handleBackorderEvents } from './backorder_events.js';

export function initBackorderView() {
    const view = document.getElementById('backorder-view');
    if (!view) return;
    
    const elements = {
        outputContainer: document.getElementById('backorder-output-container'),
        searchKanaInput: document.getElementById('bo-search-kana')
        // 卸コード入力欄の取得を削除
    };
    
    const searchBtn = document.getElementById('bo-search-btn');

    cacheUI(elements);
    cacheEvents(elements);
    
    view.addEventListener('show', loadAndRenderBackorders);
    elements.outputContainer.addEventListener('click', handleBackorderEvents);
    
    if (searchBtn) {
        searchBtn.addEventListener('click', filterAndRender);
    }
    
    if (elements.searchKanaInput) {
        elements.searchKanaInput.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') filterAndRender();
        });
    }

    console.log("Backorder View Initialized (Hub).");
}