// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\deadstock.js
// ▼▼▼【ここから修正】UI/Events モジュールをインポート ▼▼▼
import { cacheDOMElements as cacheUI, setDefaultDates } from './deadstock_ui.js';
import { cacheDOMElements as cacheEvents, initDeadStockEventListeners, fetchAndRenderDeadStock } from './deadstock_events.js';
// ▲▲▲【修正ここまで】▲▲▲

// ▼▼▼【削除】DOM変数、イベントハンドラをすべて移管 ▼▼▼
// let startDateInput, endDateInput, searchBtn, resultContainer;
// ...
// function setDefaultDates() { ... }
// async function fetchAndRenderDeadStock() { ... }
// function renderDeadStockTable(items) { ... }
// async function handleCsvUpload() { ... }
// async function handleCsvExport() { ... }
// ▲▲▲【削除ここまで】▲▲▲

export function initDeadStockView() {
    // ▼▼▼【ここから修正】DOM要素をすべて取得し、各モジュールに渡す ▼▼▼
    const view = document.getElementById('deadstock-view');
    const elements = {
        startDateInput: document.getElementById('ds-start-date'),
        endDateInput: document.getElementById('ds-end-date'),
        searchBtn: document.getElementById('ds-search-btn'),
        resultContainer: document.getElementById('deadstock-result-container'),
        exportCsvBtn: document.getElementById('ds-export-csv-btn'),
        excludeZeroStockCheckbox: document.getElementById('ds-exclude-zero-stock'),
        csvDateInput: document.getElementById('ds-csv-date'),
        csvFileInput: document.getElementById('ds-csv-file-input'),
        csvUploadBtn: document.getElementById('ds-csv-upload-btn')
    };

    // UIモジュールにDOMをキャッシュ
    cacheUI(elements);
    // EventsモジュールにDOMをキャッシュ
    cacheEvents(elements);
    
    // イベントリスナーを登録
    initDeadStockEventListeners();

    // ビュー表示時の初期化処理
    view.addEventListener('show', () => {
        setDefaultDates();
        fetchAndRenderDeadStock();
    });
    
    // ▲▲▲【修正ここまで】▲▲▲

    console.log("DeadStock View Initialized (Hub).");
}