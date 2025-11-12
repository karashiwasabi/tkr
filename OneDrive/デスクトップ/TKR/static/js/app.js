// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\app.js
import { initDatUpload, fetchAndRenderDat } from './dat.js';
import { initMasterEditView } from './masteredit.js';
import { initConfigView, loadConfigAndWholesalers } from './config.js'; 
import { initUsageUpload, fetchAndRenderUsage } from './usage.js';
import { initInventoryAdjustment } from './inventory_adjustment.js';
import { initSearchModal } from './search_modal.js';
import { loadMasterData } from './master_data.js';
import { initInOut, resetInOutView } from './inout.js';
import { initDeadStockView } from './deadstock.js';
import { initPrecomp, resetPrecompView } from './precomp.js';
import { initReorderView, fetchAndRenderReorder } from './reorder.js';
import { initBackorderView } from './backorder.js';
// ▼▼▼【ここに追加】▼▼▼
import { initValuationView } from './valuation.js';
// ▲▲▲【追加ここまで】▲▲▲

let loadingOverlay, loadingMessage, notificationBox;
// ▼▼▼【ここに追加】valuationViewBtn を追加 ▼▼▼
let views, datViewBtn, usageViewBtn, inventoryAdjustmentViewBtn, masterEditViewBtn, configViewBtn, inoutViewBtn, reprocessBtn, deadStockViewBtn, precompViewBtn, reorderViewBtn, backorderViewBtn, valuationViewBtn;
// ▲▲▲【追加ここまで】▲▲▲
const initializedViews = {
    dat: false,
    usage: false,
    inventoryAdjustment: false,
    masterEdit: false,
    config: false,
    inout: false,
    deadstock: false,
    precomp: false,
    reorder: false,
    backorder: false,
    // ▼▼▼【ここに追加】▼▼▼
    valuation: false,
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
        view.classList.remove('active');
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
        case 'inout-view':
            if (!initializedViews.inout) {
                console.log("Initializing In/Out view...");
                initInOut();
                initializedViews.inout = true;
            }
            resetInOutView();
            break;
        case 'deadstock-view':
            if (!initializedViews.deadstock) {
                console.log("Initializing DeadStock view...");
                initDeadStockView();
                initializedViews.deadstock = true;
            }
            break;
        case 'precomp-view':
            if (!initializedViews.precomp) {
                console.log("Initializing Precomp view...");
                initPrecomp();
                initializedViews.precomp = true;
            }
            resetPrecompView();
            break;
        case 'reorder-view':
            if (!initializedViews.reorder) {
                console.log("Initializing Reorder view...");
                initReorderView(); 
                initializedViews.reorder = true;
            }
            fetchAndRenderReorder(); 
            break;
        case 'backorder-view': 
            if (!initializedViews.backorder) {
                console.log("Initializing Backorder view...");
                initBackorderView(); 
                initializedViews.backorder = true;
            }
            break;
        // ▼▼▼【ここに追加】▼▼▼
        case 'valuation-view':
            if (!initializedViews.valuation) {
                console.log("Initializing Valuation view...");
                initValuationView();
                initializedViews.valuation = true;
            }
            break;
        // ▲▲▲【追加ここまで】▲▲▲
    }

    const targetView = document.getElementById(targetId);
    if (targetView) {
        targetView.classList.add('active');
        targetView.dispatchEvent(new CustomEvent('show'));
    }
}

async function handleReprocessAll() {
    if (!confirm('全ての取引データを、最新のマスター情報に基づいて再計算します。\nこの処理はデータ量に応じて時間がかかります。\n実行しますか？')) {
        return;
    }

    window.showLoading('全取引データを再計算中... (時間がかかる場合があります)');
    try {
        const response = await fetch('/api/reprocess/all', {
            method: 'POST', 
        });
        const result = await response.json();
        if (!response.ok) {
            throw new Error(result.message || `サーバーエラー (HTTP ${response.status})`);
        }

        window.showNotification(result.message || '全取引データの再計算が完了しました。', 'success');
    } catch (error) {
        console.error('Reprocessing failed:', error);
        window.showNotification(`再計算エラー: ${error.message}`, 'error');
    } finally {
        window.hideLoading();
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
    inoutViewBtn = document.getElementById('inOutViewBtn');
    reprocessBtn = document.getElementById('reprocessBtn');
    deadStockViewBtn = document.getElementById('deadStockViewBtn');
    precompViewBtn = document.getElementById('precompViewBtn');
    reorderViewBtn = document.getElementById('reorderViewBtn');
    backorderViewBtn = document.getElementById('backorderViewBtn');
    // ▼▼▼【ここに追加】▼▼▼
    valuationViewBtn = document.getElementById('valuationViewBtn');
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
    if (inoutViewBtn) {
        inoutViewBtn.addEventListener('click', () => setActiveView('inout-view'));
    }
    if (reprocessBtn) {
        reprocessBtn.addEventListener('click', handleReprocessAll);
    }
    if (deadStockViewBtn) {
        deadStockViewBtn.addEventListener('click', () => setActiveView('deadstock-view'));
    }
    if (precompViewBtn) {
        precompViewBtn.addEventListener('click', () => setActiveView('precomp-view'));
    }
    if (reorderViewBtn) {
        reorderViewBtn.addEventListener('click', () => setActiveView('reorder-view'));
    }
    if (backorderViewBtn) { 
        backorderViewBtn.addEventListener('click', () => setActiveView('backorder-view'));
    }
    // ▼▼▼【ここに追加】▼▼▼
    if (valuationViewBtn) {
        valuationViewBtn.addEventListener('click', () => setActiveView('valuation-view'));
    }
    // ▲▲▲【追加ここまで】▲▲▲

    setActiveView('dat-upload-view');
});