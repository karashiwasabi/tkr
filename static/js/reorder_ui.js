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
export function addOrUpdateOrderItem(productMaster) 
{
    if (!outputContainer) outputContainer = document.getElementById('order-candidates-output');
    
    const productCode = productMaster.productCode;
    const yjCode = productMaster.yjCode;

    // 既にリストにあるか確認 (発注可テーブル・不可テーブル両方を探す)
    const existingRow = 
outputContainer.querySelector(`tr[data-jan-code="${productCode}"]`);
    
    if (existingRow) {
        // 既に「発注不可」リストにある場合
        if (existingRow.classList.contains('provisional-order-item')) {
            if (confirm(`「${productMaster.productName}」は発注不可に設定されています。\n発注対象に変更してリストに追加しますか？`)) {
                // 発注不可リストから削除
                const tbody = existingRow.closest('tbody');
                const table = tbody.closest('table');
                const wrapper = table.closest('details');
                existingRow.remove();
                if (tbody.children.length === 0 && wrapper) {
                    // 発注不可リストが空になった場合（暫定対応）
                    wrapper.innerHTML = "<p>発注不可に設定されている不足品はありません。</p>";
                }
                // このまま続行し、発注可リストに追加する
            } else {
                return; // キャンセルしたら何もしない
            }
        } 
        // 既に「発注可」リストにある場合
        else {
            const quantityInput = existingRow.querySelector('.order-quantity-input');
            if (quantityInput) {
      
       quantityInput.value = parseInt(quantityInput.value, 10) + 1;
                window.showNotification(`「${productMaster.productName}」の数量を1増やしました。`, 'success');
            }
            return;
}
    }

     // 卸業者ドロップダウンを生成
    let wholesalerOptions = '<option value="">--- 選択 ---</option>';
    wholesalerMap.forEach((name, code) => {
    
     const isSelected = (code === productMaster.supplierWholesale);
        wholesalerOptions += `<option value="${code}" ${isSelected ? 
'selected' : ''}>${name}</option>`;
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
             <td class="col-line"><input type="number" 
value="1" class="order-quantity-input" style="width: 100%; text-align: right;"></td>
             ${actionCellHTML}
      
  </tr>
    `;

    // 「発注対象品目」テーブルを探す
    let orderableTbody = outputContainer.querySelector('#orderable-table tbody');

    // まだ「発注対象品目」テーブルが存在しない場合 (手動追加が最初の場合)
    if (!orderableTbody) {
        // outputContainer をクリア (「候補はありません」等を消す)
        outputContainer.innerHTML = ''; 
        
        // 上部セクション（発注対象）をまるごと作成
        let headerHtml = `<h3>発注対象品目 (1件)</h3>`;
        headerHtml += `
            <table id="orderable-table" class="data-table" style="margin-bottom: 20px;">
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
                ${newRowHTML}
            </tbody>
            </table>
        `;
        
        // 下部セクション（発注不可）も作成（空）
        headerHtml += `
            <details open style="margin-top: 20px;">
                <summary style="font-weight: bold; font-size: 1.1em; cursor: pointer;">発注不可品目（参考: 0件）</summary>
                <div style="padding-top: 10px;">
                    <p>発注不可に設定されている不足品はありません。</p>
                </div>
            </details>
        `;
        
        outputContainer.innerHTML = headerHtml;
    } 
    // 「発注対象品目」テーブルが既に存在する場合
    else {
        // プレースホルダー（「発注対象の品目はありません。」）があれば削除
        const placeholderRow = orderableTbody.querySelector('td[colspan="8"]');
        if (placeholderRow) {
            orderableTbody.innerHTML = '';
        }
        
        orderableTbody.insertAdjacentHTML('beforeend', newRowHTML);
        // 件数表示を更新
        const header = outputContainer.querySelector('h3'); // 最初のh3
        if(header) header.textContent = `発注対象品目 (${orderableTbody.children.length}件)`;
    }
    
    window.showNotification(`「${productMaster.productName}」を発注リストに追加しました。`, 'success');
}


/**
 * 発注候補リスト（自動生成）を描画します。
 * (旧 
reorder.js より移管)
 */
export function renderOrderCandidates(data, container) {
  
   outputContainer = container; // 出力先をキャッシュ
    if (!data.candidates || data.candidates.length === 
0) {
       
  container.innerHTML = "<p>発注が必要な品目はありませんでした。</p>";
        return;
    }

    let allItems = [];
    
    // 1. データをフラットな品目リストに解体
    (data.candidates || []).forEach(yjGroup => {
        (yjGroup.packageLedgers || []).forEach(pkg => {
            if (pkg.masters && Array.isArray(pkg.masters)) {
                pkg.masters.forEach(master => {
                    const pkgShortfall = pkg.reorderPoint - (pkg.endingBalance || 0);
                    
                    // ▼▼▼【修正】 発注残(effectiveEndingBalance)を考慮した不足数の場合のみリストアップ ▼▼▼
                    const effectiveShortfall = pkg.reorderPoint - pkg.effectiveEndingBalance;
                    
                    if (effectiveShortfall > 0) { // 発注残を考慮しても不足している品目のみを対象
                        allItems.push({
                            master: master,
                            pkg: pkg,
                            yjGroup: yjGroup,
                            shortfall: effectiveShortfall // 発注残考慮後の不足数を渡す
                        });
                    }
                    // ▲▲▲【修正ここまで】▲▲▲
                });
            }
        });
    });

    if (allItems.length === 0) {
        container.innerHTML = "<p>発注が必要な品目はありませんでした。（発注残でカバーされています）</p>";
        return;
    }

    // 2. 発注可・不可でリストを分割
    const orderableItems = allItems.filter(item => item.master.isOrderStopped !== 1);
    const stoppedItems = allItems.filter(item => item.master.isOrderStopped === 1);

    // 3. 卸業者ドロップダウンの元データを準備
    let baseWholesalerOptions = '<option value="">--- 選択 ---</option>';
    if (data.wholesalers && Array.isArray(data.wholesalers)) {
        data.wholesalers.forEach(w => {
            baseWholesalerOptions += `<option value="${w.wholesalerCode}">${w.wholesalerName}</option>`;
        });
    }
    
    let html = '';

    // 4. 上部セクション（発注対象）の構築
    html += `<h3>発注対象品目 (${orderableItems.length}件)</h3>`;
    if (orderableItems.length > 0) {
        html += `
            <table id="orderable-table" class="data-table" style="margin-bottom: 20px;">
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
        
        orderableItems.forEach(item => {
            const { master, pkg, yjGroup, shortfall } = item;

            // 発注推奨数
            const recommendedOrder = master.yjPackUnitQty > 0 ? Math.ceil(shortfall / master.yjPackUnitQty) : 0;
            
            // 卸業者ドロップダウン（選択状態を反映）
            let rowWholesalerOptions = '<option value="">--- 選択 ---</option>';
            (data.wholesalers || []).forEach(w => {
                const isSelected = (w.wholesalerCode === master.supplierWholesale);
                rowWholesalerOptions += `<option value="${w.wholesalerCode}" ${isSelected ? 'selected' : ''}>${w.wholesalerName}</option>`;
            });

            // 操作ボタン
            const actionCellHTML = `
                <td class="center col-action order-actions-cell">
                    <div class="order-action-buttons">
                        <button type="button" class="remove-order-item-btn btn">除外</button>
                        <button type="button" class="set-unorderable-btn btn" data-product-code="${master.productCode}">発注不可</button>
                    </div>
                </td>
            `;

            html += `
                <tr data-jan-code="${master.productCode}" 
                    data-yj-code="${yjGroup.yjCode}"
                    data-product-name="${master.productName}"
                    data-package-form="${master.packageForm}"
                    data-jan-pack-inner-qty="${master.janPackInnerQty}"
                    data-yj-unit-name="${master.yjUnitName}"
                    data-yj-pack-unit-qty="${master.yjPackUnitQty}"
                    data-order-multiplier="${master.yjPackUnitQty}"> 
                    <td class="left col-product" colspan="2">${master.productName}</td>
                    <td class="left col-maker">${master.makerName || ''}</td>
                    <td class="left col-package">${master.formattedPackageSpec}</td>
                    <td class="col-wholesaler"><select class="wholesaler-select" style="width: 100%; font-size: 10px;">${rowWholesalerOptions}</select></td>
                    <td class="col-count">${master.yjPackUnitQty} ${master.yjUnitName}</td>
                    <td class="col-line"><input type="number" value="${recommendedOrder}" class="order-quantity-input" style="width: 100%; text-align: right;"></td>
                    ${actionCellHTML}
                </tr>
            `;
        });
        html += `</tbody></table>`;
        
    } else {
        html += `<p>発注対象の品目はありません。</p>`;
    }

    // 5. 下部セクション（発注不可）の構築
    html += `
        <details open style="margin-top: 20px;">
            <summary style="font-weight: bold; font-size: 1.1em; cursor: pointer;">発注不可品目（参考: ${stoppedItems.length}件）</summary>
            <div style="padding-top: 10px;">
    `;
    
    if (stoppedItems.length > 0) {
        html += `
            <table id="stopped-table" class="data-table" style="font-size: 10px; background-color: #f9f9f9;">
            <thead>
                <tr>
                    <th class="col-product" colspan="2">製品名（包装）</th>
                    <th class="col-maker">メーカー</th>
                    <th class="col-package">包装仕様</th>
                    <th class="col-yj">YJ</th>
                    <th class="col-count">在庫</th>
                    <th class="col-line">発注点</th>
                    <th class="col-action">操作</th>
                </tr>
            </thead>
            <tbody>
        `;

        stoppedItems.forEach(item => {
            const { master, pkg, yjGroup } = item;

            // 発注不可品用の操作ボタン（「発注に変更」のみ）
            const actionCellHTML = `
                <td class="center col-action order-actions-cell">
                    <div class="order-action-buttons">
                        <button type="button" class="change-to-orderable-btn btn">発注に変更</button>
                    </div>
                </td>
            `;

            html += `
                <tr class="provisional-order-item" 
                    data-jan-code="${master.productCode}" 
                    data-yj-code="${yjGroup.yjCode}"
                    data-product-name="${master.productName}"
                    data-package-form="${master.packageForm}"
                    data-jan-pack-inner-qty="${master.janPackInnerQty}"
                    data-yj-unit-name="${master.yjUnitName}"
                    data-yj-pack-unit-qty="${master.yjPackUnitQty}"
                    data-order-multiplier="${master.yjPackUnitQty}"> 
                    <td class="left col-product" colspan="2">${master.productName}</td>
                    <td class="left col-maker">${master.makerName || ''}</td>
                    <td class="left col-package">${master.formattedPackageSpec}</td>
                    <td class="col-yj">${yjGroup.yjCode}</td>
                    <td class="right col-count">${formatBalance(pkg.effectiveEndingBalance)}</td>
                    <td class="right col-line">${formatBalance(pkg.reorderPoint)}</td>
                    ${actionCellHTML}
                </tr>
            `;
        });
        html += `</tbody></table>`;
    } else {
        html += `<p>発注不可に設定されている不足品はありません。</p>`;
    }
    
    html += `</div></details>`;

    // 6. 最終HTMLをコンテナに設定
    container.innerHTML = html;
}