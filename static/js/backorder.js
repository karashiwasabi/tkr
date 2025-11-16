// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\backorder.js
// ▼▼▼【ここから修正】UI/Events モジュールをインポート ▼▼▼
import { cacheDOMElements as cacheUI, filterAndRender } from './backorder_ui.js';
import { cacheDOMElements as cacheEvents, loadAndRenderBackorders, handleBackorderEvents } from './backorder_events.js';
// ▲▲▲【修正ここまで】▲▲▲


export function initBackorderView() {
    // ▼▼▼【ここから修正】DOM要素をすべて取得し、各モジュールに渡す ▼▼▼
    const view = document.getElementById('backorder-view');
    if (!view) return;
    
    const elements = {
        outputContainer: document.getElementById('backorder-output-container'),
        searchKanaInput: document.getElementById('bo-search-kana'),
        searchWholesalerInput: document.getElementById('bo-search-wholesaler')
    };
    
    // イベント専用のDOM
    const searchBtn = document.getElementById('bo-search-btn');

    // UIモジュールにDOMをキャッシュ
    cacheUI(elements);
    // EventsモジュールにDOMをキャッシュ
    cacheEvents(elements);
    
    // イベントリスナーを登録
    view.addEventListener('show', loadAndRenderBackorders);
    elements.outputContainer.addEventListener('click', handleBackorderEvents);
    
    searchBtn.addEventListener('click', filterAndRender);
    elements.searchKanaInput.addEventListener('keypress', (e) => {
         if (e.key === 'Enter') filterAndRender();
    });
    elements.searchWholesalerInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') filterAndRender();
    });
    // ▲▲▲【修正ここまで】▲▲▲
}