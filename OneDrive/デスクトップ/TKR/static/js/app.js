// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\app.js
import { initDatUpload, fetchAndRenderDat } from './dat.js';
import { initMasterEditView, fetchAndRenderMasters } from './masteredit.js';
import { initConfigView, loadConfigAndWholesalers } from './config.js'; 
import { initUsageUpload, fetchAndRenderUsage } from './usage.js';
import { initInventoryAdjustment } from './inventory_adjustment_logic.js';
import { initSearchModal } from './search_modal.js';

let loadingOverlay, loadingMessage, notificationBox;
let views, datViewBtn, usageViewBtn, inventoryAdjustmentViewBtn, masterEditViewBtn, configViewBtn;

// ▼▼▼【ここから追加】初期化済みフラグ ▼▼▼
const initializedViews = {
    dat: false,
    usage: false,
    inventoryAdjustment: false,
    masterEdit: false,
    config: false,
};
// ▲▲▲【追加ここまで】▲▲▲

window.showLoading = (message = '処理中...') => {
    if (!loadingOverlay) loadingOverlay = document.getElementById('loading-overlay');
    if (!loadingMessage) loadingMessage = document.getElementById('loading-message');
    if (loadingMessage) loadingMessage.textContent = message;
    if (loadingOverlay) loadingOverlay.classList.remove('hidden');
};

window.hideLoading = () => {
    if (!loadingOverlay) loadingOverlay = document.getElementById('loading-overlay');
    if (loadingOverlay) loadingOverlay.classList.add('hidden');
};

window.showNotification = (message, type = 'success') => {
    if (!notificationBox) notificationBox = document.getElementById('notification-box');
    if (notificationBox) {
        notificationBox.textContent = message;
        notificationBox.className = 'notification-box';
        notificationBox.classList.add(type);
        notificationBox.classList.add('show');
        setTimeout(() => {
            notificationBox.classList.remove('show');
        }, 3000);
    }
};

// ▼▼▼【ここから修正】setActiveView をJS初期化の中心にする ▼▼▼
function setActiveView(targetId) {
    if (!views) views = document.querySelectorAll('.view');
    views.forEach(view => {
        if (view.id === targetId) {
            view.classList.add('active');
        } else {
            view.classList.remove('active');
        }
    });

    // --- JSの遅延初期化ロジック ---
    // 各ビューは、初めて表示されるときに一度だけ初期化(init)され、
    // 毎回データをロードする関数が呼ばれる
    switch (targetId) {
        case 'dat-upload-view':
            if (!initializedViews.dat) {
                console.log("Initializing DAT view...");
                initDatUpload(); // イベントリスナーを登録
                initializedViews.dat = true;
            }
            fetchAndRenderDat(); // データを表示
            break;
        case 'usage-upload-view':
            if (!initializedViews.usage) {
                console.log("Initializing USAGE view...");
                initUsageUpload(); // イベントリスナーを登録
                initializedViews.usage = true;
            }
            fetchAndRenderUsage(); // データを表示
            break;
        case 'inventory-adjustment-view':
            if (!initializedViews.inventoryAdjustment) {
                console.log("Initializing Inventory Adjustment view...");
                initInventoryAdjustment(); // イベントリスナーを登録
                initializedViews.inventoryAdjustment = true;
            }
            // イベントを発火させて、棚卸調整ビューに「表示されたこと」を通知
            document.dispatchEvent(new CustomEvent('loadInventoryAdjustment', { detail: {} }));
            break;
        case 'master-edit-view':
            if (!initializedViews.masterEdit) {
                console.log("Initializing Master Edit view...");
                initMasterEditView(); // イベントリスナーを登録
                initializedViews.masterEdit = true;
            }
            
            // マスタ編集ビュー表示時に検索を実行
            const masterListBody = document.querySelector('#masterListTable tbody');
            if (masterListBody && masterListBody.children.length <= 1) {
                 fetchAndRenderMasters(); // 初回検索を実行
            }
            break;
        case 'config-view':
            if (!initializedViews.config) {
                console.log("Initializing Config view...");
                initConfigView(); // イベントリスナーを登録
                initializedViews.config = true;
            }
            // 設定ビュー表示時に設定と卸一覧をロード
            loadConfigAndWholesalers();
            break;
    }
}
// ▲▲▲【修正ここまで】▲▲▲

document.addEventListener('DOMContentLoaded', () => {
    console.log('TKR App Initialized.');

    loadingOverlay = document.getElementById('loading-overlay');
    loadingMessage = document.getElementById('loading-message');
    notificationBox = document.getElementById('notification-box');
    views = document.querySelectorAll('.view');
    datViewBtn = document.getElementById('datViewBtn');
    usageViewBtn = document.getElementById('usageViewBtn');
    inventoryAdjustmentViewBtn = document.getElementById('inventoryAdjustmentViewBtn');
    masterEditViewBtn = document.getElementById('masterEditViewBtn');
    configViewBtn = document.getElementById('configViewBtn'); 

    // ▼▼▼【ここから修正】起動時の init 呼び出しを削除 ▼▼▼
    // initDatUpload(); // -> setActiveView に移動
    // initUsageUpload(); // -> setActiveView に移動
    // initMasterEditView(); // -> setActiveView に移動
    // initConfigView();  // -> setActiveView に移動
    // initInventoryAdjustment(); // -> setActiveView に移動

    // 検索モーダルだけは共通部品なので、起動時に初期化する
    initSearchModal();
    // ▲▲▲【修正ここまで】▲▲▲

    // ヘッダーボタンのイベントリスナー (変更なし)
    if (datViewBtn) {
        datViewBtn.addEventListener('click', () => setActiveView('dat-upload-view'));
    }
    if (usageViewBtn) {
        usageViewBtn.addEventListener('click', () => setActiveView('usage-upload-view'));
    }
    if (inventoryAdjustmentViewBtn) {
        inventoryAdjustmentViewBtn.addEventListener('click', () => setActiveView('inventory-adjustment-view'));
    }
    if (masterEditViewBtn) 
    {
        masterEditViewBtn.addEventListener('click', () => setActiveView('master-edit-view'));
    }
    if (configViewBtn) {
        configViewBtn.addEventListener('click', () => setActiveView('config-view'));
    }

    // ▼▼▼【修正】デフォルトビューをアクティブにする (これにより dat.js が初期化される) ▼▼▼
    setActiveView('dat-upload-view');
    // ▲▲▲【修正ここまで】▲▲▲
});

// (showMasterEditView イベントリスナーは削除)