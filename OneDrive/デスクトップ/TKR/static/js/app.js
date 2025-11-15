// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\app.js

// ▼▼▼【ここから修正】ビューのインポートを削除し、UI/Manager/共通機能をインポート ▼▼▼
import { initSearchModal } from './search_modal.js';
import { loadMasterData } from './master_data.js';
import { initUI, showLoading, hideLoading, showNotification } from './common_ui.js';
import { initViewManager, setActiveView } from './view_manager.js';
// ▲▲▲【修正ここまで】▲▲▲


// ▼▼▼【削除】グローバル/モジュール変数を common_ui.js, view_manager.js に移管 ▼▼▼
// let loadingOverlay, loadingMessage, notificationBox;
// let views, datViewBtn, ... , pricingViewBtn;
// const initializedViews = { ... };
// ▲▲▲【削除ここまで】▲▲▲

// ▼▼▼【ここから修正】グローバル関数を window オブジェクトに登録 ▼▼▼
window.showLoading = showLoading;
window.hideLoading = hideLoading;
window.showNotification = showNotification;
// ▲▲▲【修正ここまで】▲▲▲

// ▼▼▼【削除】showLoading, hideLoading, showNotification, setActiveView を common_ui.js, view_manager.js に移管 ▼▼▼
/*
window.showLoading = (message = '処理中...') => { ... };
window.hideLoading = () => { ... };
window.showNotification = (message, type = 'success') => { ... };
function setActiveView(targetId) { ... }
*/
// ▲▲▲【削除ここまで】▲▲▲

async function 
handleReprocessAll() {
    if (!confirm('全ての取引データを、最新のマスター情報に基づいて再計算します。\nこの処理はデータ量に応じて時間がかかります。\n実行しますか？')) {
        return;
}

    window.showLoading('全取引データを再計算中... (時間がかかる場合があります)');
    try 
{
 
       const response = await fetch('/api/reprocess/all', {
       
     
method: 
'POST', 
        });
        const result = await response.json();
        if (!response.ok) {
    
  
   
   throw new Error(result.message || `サーバーエラー (HTTP ${response.status})`);
        }

        window.showNotification(result.message || '全取引データの再計算が完了しました。', 
'success');
} catch (error) {
 
       console.error('Reprocessing failed:', error);
        window.showNotification(`再計算エラー: ${error.message}`, 'error');
    } finally {
    
    
window.hideLoading();
    }
}

document.addEventListener('DOMContentLoaded', async () => {
    console.log('TKR App Initialized.');

    // ▼▼▼【ここから修正】UIとViewManagerを初期化 ▼▼▼
    initUI();
    initViewManager();
    // ▲▲▲【修正ここまで】▲▲▲

    // ▼▼▼【ここから修正】ボタンのDOM取得をローカル変数に変更 ▼▼▼
    const datViewBtn = document.getElementById('datViewBtn');
    const usageViewBtn = document.getElementById('usageViewBtn');
    const inventoryAdjustmentViewBtn = document.getElementById('inventoryAdjustmentViewBtn');
    const masterEditViewBtn = document.getElementById('masterEditViewBtn');
    const configViewBtn = document.getElementById('configViewBtn'); 
    const inoutViewBtn = document.getElementById('inOutViewBtn');
    const reprocessBtn = document.getElementById('reprocessBtn');
    const deadStockViewBtn = document.getElementById('deadStockViewBtn');
    const precompViewBtn = document.getElementById('precompViewBtn');
    const reorderViewBtn = document.getElementById('reorderViewBtn');
    const backorderViewBtn = document.getElementById('backorderViewBtn');
    const valuationViewBtn = document.getElementById('valuationViewBtn');
    const pricingViewBtn = document.getElementById('pricingViewBtn');
    // ▲▲▲【修正ここまで】▲▲▲
 
    await loadMasterData();
    initSearchModal();

    if (datViewBtn) {
        datViewBtn.addEventListener('click', () => 
setActiveView('dat-upload-view'));
    }
 
   if (usageViewBtn) {
   
     usageViewBtn.addEventListener('click', () => setActiveView('usage-upload-view'));
    }
    if (inventoryAdjustmentViewBtn) {
 
   
    inventoryAdjustmentViewBtn.addEventListener('click', 
() => setActiveView('inventory-adjustment-view'));
    }
    if (masterEditViewBtn) 
    {
     
   masterEditViewBtn.addEventListener('click', 
() => 
setActiveView('master-edit-view'));
    }
    if (configViewBtn) {
        configViewBtn.addEventListener('click', () => setActiveView('config-view'));
    }
 
   if 
(inoutViewBtn) 
{
        inoutViewBtn.addEventListener('click', () => setActiveView('inout-view'));
    }
    if (reprocessBtn) {
   
  
   reprocessBtn.addEventListener('click', 
handleReprocessAll);
    }
    if (deadStockViewBtn) {
        deadStockViewBtn.addEventListener('click', () => setActiveView('deadstock-view'));
    }
 

   if (precompViewBtn) {
  
      precompViewBtn.addEventListener('click', () => setActiveView('precomp-view'));
    }
    if (reorderViewBtn) {
 
  
     reorderViewBtn.addEventListener('click', () => 
setActiveView('reorder-view'));
    }
    if (backorderViewBtn) { 
      
  backorderViewBtn.addEventListener('click', () => setActiveView('backorder-view'));
    }
    
  if (valuationViewBtn) {
       
 valuationViewBtn.addEventListener('click', () => setActiveView('valuation-view'));
    }
 
   if (pricingViewBtn) {
     
   pricingViewBtn.addEventListener('click', () => setActiveView('pricing-view'));
    }
    
    setActiveView('dat-upload-view');
});