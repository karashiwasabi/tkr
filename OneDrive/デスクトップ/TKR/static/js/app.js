// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\app.js
import { initDatUpload, fetchAndRenderDat } from './dat.js';
import { initMasterEditView } from './masteredit.js';
import { initConfigView, loadConfigAndWholesalers } from './config.js'; 
import { initUsageUpload, fetchAndRenderUsage } from './usage.js';
import { initInventoryAdjustment } from './inventory_adjustment_logic.js';
import { initSearchModal } from './search_modal.js';
import { loadMasterData } from './master_data.js';
// ▼▼▼【ここに追加】▼▼▼
import { initInOut, resetInOutView } from './inout.js';
// ▲▲▲【追加ここまで】▲▲▲

let loadingOverlay, loadingMessage, notificationBox;
// ▼▼▼【ここに追加】▼▼▼
let views, datViewBtn, usageViewBtn, inventoryAdjustmentViewBtn, masterEditViewBtn, configViewBtn, inoutViewBtn;
// ▲▲▲【追加ここまで】▲▲▲
const initializedViews = {
    dat: false,
    usage: false,
    inventoryAdjustment: false,
    masterEdit: false,
    config: false,
    // ▼▼▼【ここに追加】▼▼▼
    inout: false,
    // ▲▲▲【追加ここまで】▲▲▲
};

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

function setActiveView(targetId) {
    if (!views) views = document.querySelectorAll('.view');
    views.forEach(view => {
        if (view.id === targetId) {
            view.classList.add('active');
        } else {
            view.classList.remove('active');
        }
    });
    switch (targetId) {
        case 'dat-upload-view':
            if (!initializedViews.dat) {
                console.log("Initializing DAT view...");
                initDatUpload();
                initializedViews.dat = true;
            }
            fetchAndRenderDat();
            break;
        case 'usage-upload-view':
            if (!initializedViews.usage) {
                console.log("Initializing USAGE view...");
                initUsageUpload();
                initializedViews.usage = true;
            }
            fetchAndRenderUsage();
            break;
        case 'inventory-adjustment-view':
            if (!initializedViews.inventoryAdjustment) {
                console.log("Initializing Inventory Adjustment view...");
                initInventoryAdjustment();
                initializedViews.inventoryAdjustment = true;
            }
            document.dispatchEvent(new CustomEvent('loadInventoryAdjustment', { detail: {} }));
            break;
        case 'master-edit-view':
            if (!initializedViews.masterEdit) {
                console.log("Initializing Master Edit view...");
                initMasterEditView();
                initializedViews.masterEdit = true;
            }
            break;
        case 'config-view':
            if (!initializedViews.config) {
                console.log("Initializing Config view...");
                initConfigView();
                initializedViews.config = true;
            }
            loadConfigAndWholesalers();
            break;
        // ▼▼▼【ここに追加】▼▼▼
        case 'inout-view':
            if (!initializedViews.inout) {
                console.log("Initializing In/Out view...");
                initInOut();
                initializedViews.inout = true;
            }
            resetInOutView();
            break;
        // ▲▲▲【追加ここまで】▲▲▲
    }
}

document.addEventListener('DOMContentLoaded', async () => {
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
    // ▼▼▼【ここに追加】▼▼▼
    inoutViewBtn = document.getElementById('inOutViewBtn');
    // ▲▲▲【追加ここまで】▲▲▲

    await loadMasterData();

    initSearchModal();

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
    // ▼▼▼【ここに追加】▼▼▼
    if (inoutViewBtn) {
        inoutViewBtn.addEventListener('click', () => setActiveView('inout-view'));
    }
    // ▲▲▲【追加ここまで】▲▲▲

    setActiveView('dat-upload-view');
});
