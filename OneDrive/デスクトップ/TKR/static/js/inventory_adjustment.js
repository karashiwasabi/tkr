// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\inventory_adjustment.js
import { setUnitMap } from './inventory_adjustment_ui.js';
// ▼▼▼【ここを修正】setCurrentYjCode のインポート元を変更 ▼▼▼
import 
{ initSearchForm } from './inventory_adjustment_search.js';
import { setCurrentYjCode } from './inventory_adjustment_logic.js';
// ▲▲▲【修正ここまで】▲▲▲
import { initDetails, loadAndRenderDetails } from './inventory_adjustment_details.js';

export 
async function initInventoryAdjustment() {
    let localUnitMap = {};
    try 
{
        const res = await fetch('/api/units/map');
        if (!res.ok) 
throw new Error('単位マスタの取得に失敗');
        localUnitMap = await res.json();
        setUnitMap(localUnitMap);
    } catch (err) 
{
        console.error(err);
        window.showNotification(err.message, 'error');
    }

    initDetails();
initSearchForm(loadAndRenderDetails);

    // ▼▼▼【ここから修正】'loadInventoryAdjustment' (カスタムイベント) と 'show' (ビュー切替) の両方をリッスンする ▼▼▼
    const view = document.getElementById('inventory-adjustment-view'); // ★ view をここで取得
    
    
// 他のビュー (Deadstock, Backorder) から yjCode 付きで遷移してきた場合
    document.addEventListener('loadInventoryAdjustment', (e) => {
      
  const { yjCode } = e.detail;
 
       if (yjCode) {
            if (view) view.dataset.justLoaded = 'true'; // ★ show イベントでのリセットを抑止するフラグ
            setCurrentYjCode(yjCode);
    loadAndRenderDetails(yjCode);
        }
 
  
 });

    // ヘッダータブから直接 (yjCode なしで) 遷移してきた場合
    if (view) {
        view.addEventListener('show', () => {
  
          // 他のビューからの遷移イベントが発火した直後かどうかをチェック
            const justLoaded = view.dataset.justLoaded 
=== 'true';
            
            if (justLoaded) 
{
                // 他ビューからの遷移直後なら、フラグをリセットして何もしない
          
      // (loadInventoryAdjustment が既に loadAndRenderDetails(yjCode) を呼んでいるため)
                view.dataset.justLoaded 
= 'false';
            } else {
            
    // ヘッダータブからの直接クリックの場合
                setCurrentYjCode(null); // yjCode をリセット
   
             loadAndRenderDetails(null); // 初期メッセージを描画
            
}
        });
    }
    // ▲▲▲【修正ここまで】▲▲▲

    console.log("Inventory Adjustment View Initialized (Hub).");
}