// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\view_manager.js
// (新規作成)

// ▼▼▼ app.js からビュー初期化関数をすべて移管 ▼▼▼
import { initDatUpload, fetchAndRenderDat } from './dat.js';
import { initMasterEditView } from './masteredit.js';
import { initConfigView, loadConfigAndWholesalers } from './config.js';
import { initUsageUpload, fetchAndRenderUsage } from './usage.js';
import { initInventoryAdjustment } from './inventory_adjustment.js';
import { initInOut, resetInOutView } from './inout.js';
import { initDeadStockView } from './deadstock.js';
import { initPrecomp, resetPrecompView } from './precomp.js';
import { initReorderView, fetchAndRenderReorder } from './reorder.js';
import { initBackorderView } from './backorder.js';
import { initValuationView } from './valuation.js';
import { initPricingView } from './pricing.js';
// ▲▲▲ 移管ここまで ▲▲▲

let views;

// ▼▼▼ app.js から initializedViews を移管 ▼▼▼
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
    valuation: false,
    pricing: false,
};
// ▲▲▲ 移管ここまで ▲▲▲

/**
 * app.jsの起動時にDOM要素(views)をキャッシュします。
 */
export function initViewManager() {
    views = document.querySelectorAll('.view');
}

// ▼▼▼ app.js から setActiveView を移管 ▼▼▼
/**
 * 指定されたビューをアクティブにし、必要に応じて初期化します。
 * @param {string} targetId - アクティブにするビューのID
 */
export function setActiveView(targetId) {
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
        case 'valuation-view':
            if (!initializedViews.valuation) {
                console.log("Initializing Valuation view...");
                initValuationView();
                initializedViews.valuation = true;
            }
            break;
        case 'pricing-view':
            if (!initializedViews.pricing) {
                console.log("Initializing Pricing view...");
                initPricingView();
                initializedViews.pricing = true;
            }
            break;
    }

    const targetView = document.getElementById(targetId);
    if (targetView) {
        targetView.classList.add('active');
        targetView.dispatchEvent(new CustomEvent('show'));
    }
}
// ▲▲▲ 移管ここまで ▲▲▲