// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\reorder_ui.js
import { wholesalerMap } from './master_data.js';

let outputContainer; // このモジュール内で出力先を保持

/**
 * TKRでは .toFixed(2) は不要 (DB側で REAL 型として適切に扱われる)
 */
function formatBalance(balance) {
    if (typeof balance === 'number') {
        return balance;
    }
    return balance;
}

/**
 * 検索モーダルやスキャンから品目をリストに追加または更新します。
 * (旧 reorder.js より移管)
 */
export function addOrUpdateOrderItem(productMaster) {
    if (!outputContainer) outputContainer = document.getElementById('order-candidates-output');
    
    const productCode = productMaster.productCode;
    const yjCode = productMaster.yjCode;

    // 既にリストにあるか確認
    const existingRow = outputContainer.querySelector(`tr[data-jan-code="${productCode}"]`);
    if (existingRow) {
        const quantityInput = existingRow.querySelector('.order-quantity-input');
        if (quantityInput) {
             quantityInput.value = parseInt(quantityInput.value, 10) + 1;
            window.showNotification(`「${productMaster.productName}」の数量を1増やしました。`, 'success');
        }
        return;
    }

     // 卸業者ドロップダウンを生成
    let wholesalerOptions = '<option value="">--- 選択 ---</option>';
    wholesalerMap.forEach((name, code) => {
         const isSelected = (code === productMaster.supplierWholesale);
        wholesalerOptions += `<option value="${code}" ${isSelected ? 'selected' : ''}>${name}</option>`;
    });

    // TKRのテーブル定義 (table.css) に合わせた操作ボタン
    const actionCellHTML = `
        <td class="center col-action order-actions-cell">
            <div class="order-action-buttons">
                 <button type="button" class="remove-order-item-btn btn">除外</button>
                 <button type="button" class="set-unorderable-btn btn" data-product-code="${productMaster.productCode}">発注不可</button>
            </div>
         </td>
    `;
    
    // TKRのテーブル定義 (table.css) に合わせた列構成
    const newRowHTML = `
         <tr data-jan-code="${productMaster.productCode}" 
            data-yj-code="${productMaster.yjCode}"
             data-product-name="${productMaster.productName}"
            data-package-form="${productMaster.packageForm}"
             data-jan-pack-inner-qty="${productMaster.janPackInnerQty}"
            data-yj-unit-name="${productMaster.yjUnitName}"
             data-yj-pack-unit-qty="${productMaster.yjPackUnitQty}"
            data-order-multiplier="${productMaster.yjPackUnitQty}"> 
            <td class="left col-product" colspan="2">${productMaster.productName}</td>
            <td class="left col-maker">${productMaster.makerName || ''}</td>
             <td class="left col-package">${productMaster.formattedPackageSpec}</td>
             <td class="col-wholesaler"><select class="wholesaler-select" style="width: 100%; font-size: 10px;">${wholesalerOptions}</select></td>
            <td class="col-count">${productMaster.yjPackUnitQty} ${productMaster.yjUnitName}</td>
             <td class="col-line"><input type="number" value="1" class="order-quantity-input" style="width: 100%; text-align: right;"></td>
             ${actionCellHTML}
        </tr>
    `;

    let yjGroupWrapper = outputContainer.querySelector(`.order-yj-group-wrapper[data-yj-code="${yjCode}"]`);
 if (yjGroupWrapper) {
        const tbody = yjGroupWrapper.querySelector('tbody');
        tbody.insertAdjacentHTML('beforeend', newRowHTML);
    } else {
         const yjHeaderHTML = `
            <div class="agg-yj-header" style="background-color: #f0f0f0; color: #333; border: 1px solid #ccc;">
                <span>YJ: ${yjCode}</span>
                 <span class="product-name">${productMaster.productName}</span>
             </div>`;
        
        // TKRのテーブル定義 (table.css) に合わせたヘッダー
        const tableHeader = `
             <thead>
                 <tr>
                    <th class="col-product" colspan="2">製品名（包装）</th>
                    <th class="col-maker">メーカー</th>
                     <th class="col-package">包装仕様</th>
                     <th class="col-wholesaler">卸業者</th>
                     <th class="col-count">発注単位</th>
                     <th class="col-line">発注数</th>
                     <th class="col-action">操作</th>
                 </tr>
            </thead>
         `;
            
        const tableHTML = `
            <table class="data-table" style="margin-bottom: 10px;">
                 ${tableHeader}
                 <tbody>
                     ${newRowHTML}
                </tbody>
             </table>`;
            
        const newGroupHTML = `
             <div class="order-yj-group-wrapper" data-yj-code="${yjCode}">
                ${yjHeaderHTML}
                 ${tableHTML}
             </div>`;
        outputContainer.insertAdjacentHTML('beforeend', newGroupHTML);
    }
    window.showNotification(`「${productMaster.productName}」を発注リストに追加しました。`, 'success');
}


/**
 * 発注候補リスト（自動生成）を描画します。
 * (旧 reorder.js より移管)
 */
export function renderOrderCandidates(data, container) {
  
   outputContainer = container; // 出力先をキャッシュ
    if (!data.candidates || data.candidates.length === 0) {
       
  container.innerHTML = "<p>発注が必要な品目はありませんでした。</p>";
        return;
    }

    let html = '';
    // ▼▼▼【ここから修正】 data.candidates が null でないことを保証 ▼▼▼
    (data.candidates || []).forEach(yjGroup => {
    // ▲▲▲【修正ここまで】▲▲▲
        
 const yjShortfall = yjGroup.totalReorderPoint - (yjGroup.endingBalance || 0);

        html += `
             <div class="order-yj-group-wrapper" data-yj-code="${yjGroup.yjCode}">
                 <div class="agg-yj-header" style="background-color: #dc3545;">
                     <span>YJ: ${yjGroup.yjCode}</span>
                    <span class="product-name">${yjGroup.productName}</span>
                    <span class="balance-info">
                         在庫: ${formatBalance(yjGroup.endingBalance)} | 
                         発注点: ${formatBalance(yjGroup.totalReorderPoint)} | 
                         不足数: ${formatBalance(yjShortfall)}
                     </span>
                </div>
         `;

        // ▼▼▼【ここから修正】 yjGroup.packageLedgers が null でないことを保証 ▼▼▼
        const existingBackordersForYj = (yjGroup.packageLedgers || []).flatMap(p => p.existingBackorders || []);
        // ▲▲▲【修正ここまで】▲▲▲
  
       if (existingBackordersForYj.length > 0) {
            html += `<div class="existing-backorders-info">
                     <strong>＜既存の発注残＞</strong>
                     <ul>`;
            existingBackordersForYj.forEach(bo => {
                const wName = wholesalerMap.get(bo.wholesalerCode) || bo.wholesalerCode || '不明';
                html += `<li>${bo.orderDate}: ${bo.productName} - 数量: ${bo.remainingQuantity.toFixed(2)} (${wName})</li>`;
            });
            html += `</ul></div>`;
 }

        html += `
                 <table class="data-table" style="margin-bottom: 10px;">
                     <thead>
                         <tr>
                             <th class="col-product" colspan="2">製品名（包装）</th>
                             <th class="col-maker">メーカー</th>
                             <th class="col-package">包装仕様</th>
                             <th class="col-wholesaler">卸業者</th>
                             <th class="col-count">発注単位</th>
                             <th class="col-line">発注数</th>
                             <th class="col-action">操作</th>
                         </tr>
                    </thead>
                     <tbody>
         `;
        
        // ▼▼▼【ここから修正】 yjGroup.packageLedgers が null でないことを保証 ▼▼▼
        (yjGroup.packageLedgers || []).forEach(pkg => {
        // ▲▲▲【修正ここまで】▲▲▲
            // ▼▼▼【ここから修正】 pkg.masters が null でないことを保証 ▼▼▼
            if (pkg.masters && Array.isArray(pkg.masters)) {
                pkg.masters.forEach(master => {
            // ▲▲▲【修正ここまで】▲▲▲
   
                  const pkgShortfall = pkg.reorderPoint - (pkg.endingBalance || 0);
                    
  
                   if (pkgShortfall > 0) {
 
                        const isOrderStopped = master.isOrderStopped === 1;
                  
       const isOrderable = !isOrderStopped;

              
           const rowClass = !isOrderable ? 'provisional-order-item' : '';
      
                   const disabledAttr = !isOrderable ? 
 'disabled' : '';

                     
    // 発注推奨数を計算 (TKRでは YjPackUnitQty が 0 の場合があるためガード)
              
           const recommendedOrder = master.yjPackUnitQty > 0 ? Math.ceil(pkgShortfall / master.yjPackUnitQty) : 0;
  
                       let rowWholesalerOptions = '<option value="">--- 選択 ---</option>';
                        
                        // ▼▼▼【ここから修正】 data.wholesalers が null でないことを保証 ▼▼▼
                        if (data.wholesalers && Array.isArray(data.wholesalers)) {
                            data.wholesalers.forEach(w => {
                        // ▲▲▲【修正ここまで】▲▲▲
              
               const isSelected = (w.wholesalerCode === master.supplierWholesale);
                            rowWholesalerOptions += `<option 
 value="${w.wholesalerCode}" ${isSelected ? 'selected' : ''}>${w.wholesalerName}</option>`;
                            });
                        }
                        
                        let actionCellHTML = `
                             <td class="center col-action order-actions-cell">
                                 <div class="order-action-buttons">
                        `;
                        if (isOrderable) {
              
               actionCellHTML += '<button type="button" class="remove-order-item-btn btn">除外</button>';
                        } else {
 
                        
     actionCellHTML += '<button type="button" class="change-to-orderable-btn btn">発注に変更</button>';
                        }
             
            actionCellHTML += `
          
                        
    <button type="button" class="set-unorderable-btn btn" data-product-code="${master.productCode}">発注不可</button>
                                 </div>
                             </td>
                       `;
 html += `
                     
                       <tr class="${rowClass}" 
                                data-jan-code="${master.productCode}" 
                                data-yj-code="${yjGroup.yjCode}"
                                 data-product-name="${master.productName}"
                                data-package-form="${master.packageForm}"
                                 data-jan-pack-inner-qty="${master.janPackInnerQty}"
                                 data-yj-unit-name="${master.yjUnitName}"
                                 data-yj-pack-unit-qty="${master.yjPackUnitQty}"
                                 data-order-multiplier="${master.yjPackUnitQty}"> 
                         <td class="left col-product" colspan="2">${master.productName}</td>
                             <td class="left 
 col-maker">${master.makerName || ''}</td>
                             <td class="left col-package">${master.formattedPackageSpec}</td>
                             <td 
 class="col-wholesaler"><select class="wholesaler-select" style="width: 100%; font-size: 10px;" ${disabledAttr}>${rowWholesalerOptions}</select></td>
                                 <td class="col-count">${master.yjPackUnitQty} ${master.yjUnitName}</td>
                                <td class="col-line"><input type="number" value="${recommendedOrder}" class="order-quantity-input" style="width: 100%; text-align: right;" ${disabledAttr}></td>
                             ${actionCellHTML}
                        </tr>
                     `;
                    }
                });
            }
        });
        html += `</tbody></table></div>`;
 });
    container.innerHTML = html;
}