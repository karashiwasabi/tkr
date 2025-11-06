// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\inventory_adjustment_ui.js
import { getLocalDateString } from './utils.js';
import { renderTransactionTableHTML } from './common_table.js';
let unitMap = {};

/**
 * 単位マップを外部から設定する
 */
export function setUnitMap(map) {
    unitMap = map;
}

/**
 * 「1. 全体サマリー」セクションのHTMLを生成する
 */
function generateSummaryLedgerHtml(yjGroup, yesterdaysTotal) {
// ... (変更なし) ...
    const endDate = getLocalDateString();
    const startDate = new Date();
    startDate.setDate(startDate.getDate() - 30);
    const startDateStr = startDate.toISOString().slice(0, 10);
    let packageLedgerHtml = (yjGroup.packageLedgers || []).map(pkg => {
        const sortedTxs = (pkg.transactions || []).sort((a, b) => 
            (a.transactionDate + a.id).toString().localeCompare(b.transactionDate + b.id)
        );
        const pkgHeader = `
            <div class="agg-pkg-header adj-pkg-margin">
                 <span>包装: ${pkg.packageKey}</span>
     
 
            <span class="balance-info">
                    本日理論在庫(包装計): ${(pkg.endingBalance || 0).toFixed(2)} ${yjGroup.yjUnitName}
                </span>
            </div>
        `;
        const txTable = renderTransactionTableHTML(sortedTxs);
        return pkgHeader + txTable;
    }).join('');
    return `<div class="summary-section">
        <h3 class="view-subtitle">1. 全体サマリー</h3>
        <div class="report-section-header">
            <h4>在庫元帳 (期間: ${startDateStr} ～ ${endDate})</h4>
            <span class="header-total">【参考】前日理論在庫合計: ${yesterdaysTotal.toFixed(2)} ${yjGroup.yjUnitName}</span>
        </div>
        ${packageLedgerHtml}
    </div>`;
}

/**
 * 「2. 棚卸入力」セクションの入力行のHTMLを生成する
 */
export function createFinalInputRow(master, deadStockRecord = null, isPrimary = false) {
// ... (変更なし) ...
    const actionButtons = isPrimary ?
        `
        <button type="button" class="btn add-deadstock-row-btn" data-product-code="${master.productCode}">＋</button>
        <button type="button" class="btn register-inventory-btn">登録</button>
    ` : `<button type="button" class="btn delete-deadstock-row-btn">－</button>`;
    const quantityInputClass = isPrimary ? 'final-inventory-input' : 'lot-quantity-input';
    const quantityPlaceholder = isPrimary ? '目安をここに転記' : 'ロット数量';
    const quantity = deadStockRecord ? deadStockRecord.stockQuantityJan : '';
    const expiry = deadStockRecord ? deadStockRecord.expiryDate : '';
    const lot = deadStockRecord ? deadStockRecord.lotNumber : '';

    // ▼▼▼【修正】単価(col-unitprice)をYJ単位薬価に、金額(col-amount)を空欄に変更 ▼▼▼
    const topRow = `<tr class="inventory-row"><td rowspan="2" class="col-action"><div class="action-buttons">${actionButtons}</div></td>
        <td class="col-date">(棚卸日)</td><td class="yj-jan-code col-yj">${master.yjCode}</td><td class="left col-product" colspan="2">${master.productName}</td>
        <td class="col-count"></td><td class="right col-yjqty"></td><td class="right col-yjpackqty">${master.yjPackUnitQty || ''}</td><td class="col-yjunit">${master.yjUnitName || ''}</td>
        <td class="right col-unitprice">${master.nhiPrice?.toFixed(4) || ''}</td><td class="col-expiry"><input type="text" class="expiry-input" placeholder="YYYYMM" value="${expiry}"></td><td class="col-wholesaler"></td><td class="col-line"></td></tr>`;
    
    const bottomRow = `<tr class="inventory-row"><td class="col-flag">棚卸</td><td class="yj-jan-code col-jan">${master.productCode}</td>
        <td class="col-package">${master.formattedPackageSpec || ''}</td><td class="col-maker">${master.makerName || ''}</td><td class="col-form">${master.usageClassification || ''}</td>
        <td class="right col-janqty"><input type="number" class="${quantityInputClass}" data-product-code="${master.productCode}" placeholder="${quantityPlaceholder}" value="${quantity}" step="any"></td>
        <td class="right col-janpackqty">${master.janPackUnitQty || ''}</td><td class="col-janunit">${master.janUnitName || ''}</td>
        <td class="right col-amount"></td><td class="col-lot"><input type="text" class="lot-input" placeholder="ロット番号" value="${lot}"></td><td class="col-receipt"></td><td class="col-ma"></td></tr>`;
    // ▲▲▲【修正ここまで】▲▲▲
    
    return topRow + bottomRow;
}

/**
 * 「2. 棚卸入力」セクション全体のHTMLを生成する
 */
function generateInputSectionsHtml(packageLedgers, yjUnitName = '単位', cache, yesterdaysTotal) {
// ... (変更なし) ...
    const packageGroupsHtml = (packageLedgers || []).map(pkgLedger => {
        let yesterdaysPkgStock = 0;
        if(cache.yesterdaysStock && cache.yesterdaysStock.packageLedgers){
            const prevPkg = cache.yesterdaysStock.packageLedgers.find(p => p.packageKey === pkgLedger.packageKey);
            if(prevPkg) {
                yesterdaysPkgStock = prevPkg.endingBalance || 0;
        
 
     }
        }
        
        let totalStockDisplay = `${(pkgLedger.endingBalance || 0).toFixed(2)} ${yjUnitName}`;
        let yesterdaysTotalStockDisplay = `${yesterdaysPkgStock.toFixed(2)} ${yjUnitName}`;

        if (pkgLedger.masters && pkgLedger.masters.length > 0) {
            const firstMaster = pkgLedger.masters[0];
            const janPackInnerQty = firstMaster.janPackInnerQty;
         
   
  const janUnitName = firstMaster.janUnitName;
            
            if (janPackInnerQty > 0) {
                const totalJanStock = (pkgLedger.endingBalance || 0) / janPackInnerQty;
                totalStockDisplay = `${totalJanStock.toFixed(2)} ${janUnitName}`;
                
                const yesterdaysTotalJanStock = yesterdaysPkgStock / janPackInnerQty;
                yesterdaysTotalStockDisplay = `${yesterdaysTotalJanStock.toFixed(2)} ${janUnitName}`;
            }
        }
        let html = `
        <div class="package-input-group">
            <div class="agg-pkg-header">
                <span>包装: ${pkgLedger.packageKey}</span>
            </div>`;
        html += (pkgLedger.masters || []).map(master => {
            if (!master) return '';
            
            const janUnitName = master.janUnitName;
            
            // ▼▼▼【ここから修正】「① 本日の実在庫数量」に readonly 属性を追加 ▼▼▼
            const userInputArea = `
        
                 <div class="user-input-area">
        
             <div class="form-group">
                        <label>① 本日の実在庫数量（予製除く）:</label>
                        <input type="number" class="physical-stock-input" data-product-code="${master.productCode}" step="any" readonly>
               
                 <span>(${janUnitName})</span>
          
               <span class="info-text">本日理論在庫(包装計): ${totalStockDisplay}</span>
                    </div>
            // ▲▲▲【修正ここまで】▲▲▲
                    <div class="form-group">
          
                       <label>② 前日在庫(逆算値):</label>
                  
       <span class="calculated-previous-day-stock" data-product-code="${master.productCode}">0.00</span>
                        <span>(${janUnitName})</span>
                        <span class="info-text stock-info">(この数値が棚卸データとして登録されます)</span>
        
                         <span class="info-text align-right">前日理論在庫(包装計): ${yesterdaysTotalStockDisplay}</span>
               
      </div>
                </div>`;
            const relevantDeadStock = (cache.deadStockDetails || []).filter(ds => ds.productCode === master.productCode);
            let finalInputTbodyHtml;
            if (relevantDeadStock.length > 0) {
                finalInputTbodyHtml = relevantDeadStock.map((rec, index) => createFinalInputRow(master, rec, index === 0)).join('');
            } else {
                finalInputTbodyHtml = createFinalInputRow(master, null, true);
            }
            
            const finalInputTable = renderTransactionTableHTML(
                [], // データは customBody で渡す
                `<tbody class="final-input-tbody" data-product-code="${master.productCode}">${finalInputTbodyHtml}</tbody>`
            );
            return `<div class="product-input-group">
                        ${userInputArea}
                        <div>
                            <p class="lot-input-header">ロット・期限を個別入力</p>
                     
 
        ${finalInputTable}
                        </div>
                    </div>`;
        }).join('');

        html += `</div>`;
        return html;
    }).join('');
    
    return `<div class="input-section">
        <h3 class="view-subtitle">2. 棚卸入力</h3>
        <div class="inventory-input-area">
            <div class="adj-date-group">
                <div class="form-group">
                    <label for="inventory-date">棚卸日:</label>
                    <input type="date" id="inventory-date">
                 </div>
 
 
                <form id="adjustment-barcode-form" class="adj-barcode-form">
                    <div class="form-group">
                        <label for="adjustment-barcode-input">バーコードでロット・期限入力</label>
                        <input type="text" id="adjustment-barcode-input" inputmode="latin" placeholder="GS1-128バーコードをスキャンしてEnter">
        
  
            </div>
                </form>
            </div>
        </div>
        ${packageGroupsHtml}
    </div>`;
}

/**
 * 「3. 参考」セクションのHTMLを生成する
 */
// ▼▼▼【ここから修正】「3. 参考」セクションのレイアウトを変更 ▼▼▼
function generateDeadStockReferenceHtml(deadStockRecords, cache) {
    if (!deadStockRecords || deadStockRecords.length === 0) {
        return '';
    }

    // マスタ情報（特に package_spec）を引けるようにマップを作成
    const masterMap = new Map();
    if (cache && cache.transactionLedger) {
        for (const yjGroup of cache.transactionLedger) {
            for (const pkg of yjGroup.packageLedgers) {
                for (const master of pkg.masters) {
                    masterMap.set(master.productCode, master);
                }
            }
        }
    }

    const rowsHtml = deadStockRecords.map(rec => {
        const master = masterMap.get(rec.productCode);
        const packageSpec = master ? master.formattedPackageSpec : '(仕様不明)';
        
        return `
        <tr>
            <td class="yj-jan-code col-ref-jan">${rec.productCode}</td>
            <td class="left col-ref-spec">${packageSpec}</td>
            <td class="right col-ref-qty">${rec.stockQuantityJan.toFixed(2)}</td>
            <td class="center col-ref-expiry">${rec.expiryDate || ''}</td>
            <td class="left col-ref-lot">${rec.lotNumber || ''}</td>
        </tr>
    `
    }).join('');

    return `
        <div class="summary-section input-section">
            <h3 class="view-subtitle">3. 参考：現在登録済みのロット・期限情報</h3>
            <p class="reference-section-header">※このリストは参照用です。棚卸情報を保存するには、上の「2. 棚卸入力」の欄に改めて入力してください。</p>
            <table class="data-table reference-table">
                <thead>
                    <tr>
                        <th class="col-ref-jan">JAN</th>
                        <th class="col-ref-spec">包装仕様</th>
                        <th class="col-ref-qty">在庫数量(JAN)</th>
                        <th class="col-ref-expiry">使用期限</th>
                        <th class="col-ref-lot">ロット番号</th>
                    </tr>
                </thead>
             
  
    <tbody>
                    ${rowsHtml}
                </tbody>
            </table>
        </div>
    `;
}
// ▲▲▲【修正ここまで】▲▲▲

/**
 * 棚卸調整画面の全HTMLを生成する
 */
export function generateFullHtml(data, cache) {
// ... (変更なし) ...
    if (!data.transactionLedger || data.transactionLedger.length === 0) {
        return '<p>対象の製品データが見つかりませんでした。</p>';
    }
    const yjGroup = data.transactionLedger[0];
    const productName = yjGroup.productName;
    const yesterdaysTotal = data.yesterdaysStock ?
 (data.yesterdaysStock.endingBalance || 0) : 0;
    
    const summaryLedgerHtml = generateSummaryLedgerHtml(yjGroup, yesterdaysTotal);
    const inputSectionsHtml = generateInputSectionsHtml(yjGroup.packageLedgers, yjGroup.yjUnitName, cache, yesterdaysTotal);
    const deadStockReferenceHtml = generateDeadStockReferenceHtml(data.deadStockDetails, cache);
    return `<h2 class="adj-header">【棚卸調整】 ${productName} (YJ: ${yjGroup.yjCode})</h2>
        ${summaryLedgerHtml}
        ${inputSectionsHtml}
        ${deadStockReferenceHtml}`;
}