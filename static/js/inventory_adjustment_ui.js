// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\inventory_adjustment_ui.js
import { getLocalDateString } from './utils.js';
import { renderTransactionTableHTML } from './common_table.js';
import { clientMap } from './master_data.js';

let unitMap = {};
const transactionTypeMap = {
	0: "棚卸", 1: "納品", 2: "返品", 3: "処方",
    5: "予製",
    11: "入庫", 12: "出庫"
};

export function setUnitMap(map) {
    unitMap = map;
}

function generateSummaryLedgerHtml(yjGroup, yesterdaysTotal, cache) {
    const endDate = getLocalDateString();
    const startDate = new Date();
    // ▼▼▼【修正】表示期間を -30 から -90 に変更 ▼▼▼
    startDate.setDate(startDate.getDate() - 90);
    // ▲▲▲【修正ここまで】▲▲▲
    const startDateStr = startDate.toISOString().slice(0, 10);
    
    let packageLedgerHtml = (yjGroup.packageLedgers || []).map(pkg => {
        const sortedTxs = (pkg.transactions || []).sort((a, b) => 
            (a.transactionDate + a.id).toString().localeCompare(b.transactionDate + b.id)
        );

        let yesterdaysPkgStock = 0;
        if(cache && cache.yesterdaysStock && cache.yesterdaysStock.packageLedgers){
            const prevPkg = cache.yesterdaysStock.packageLedgers.find(p => p.packageKey === pkg.packageKey);
            if(prevPkg) {
                 yesterdaysPkgStock = prevPkg.endingBalance || 0;
            }
        }
        
        const balanceDisplay = (yesterdaysPkgStock || 0).toFixed(2);

        const pkgHeader = `
            <div class="agg-pkg-header adj-pkg-margin">
                 <span>包装: ${pkg.packageKey}</span>
                 <span class="balance-info">
                     理論在庫(包装計): ${balanceDisplay} ${yjGroup.yjUnitName}
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
        </div>
        ${packageLedgerHtml}
    </div>`;
}

function generateSummaryPrecompHtml(precompDetails) {
    const precompTransactions = (precompDetails || []).map(p => ({
        ...p,
        transactionDate: (p.transactionDate || '').slice(0, 8),
        flag: 5,
        clientCodeDisplay: clientMap.get(p.clientCode) || p.clientCode || '', 
    }));
    
    const renderTKRPrecompTable = (records) => {
        const transactionTypeMap = {
            0: "棚卸", 1: "納品", 2: "返品", 3: "処方",
            5: "予製",
            11: "入庫", 12: "出庫"
        };
        
        const header = `<thead>
             <tr><th rowspan="2" class="col-action">有効</th><th class="col-date">日付</th><th class="col-yj">YJ</th><th colspan="2" class="col-product">製品名</th><th class="col-count">個数</th><th class="col-yjqty">YJ数量</th><th class="col-yjpackqty">YJ包装数</th><th class="col-yjunit">YJ単位</th><th class="col-unitprice">単価</th><th class="col-expiry">期限</th><th class="col-wholesaler">患者</th><th class="col-line">行</th></tr>
            <tr><th class="col-flag">種別</th><th class="col-jan">JAN</th><th class="col-package">包装</th><th class="col-maker">メーカー</th><th class="col-form">剤型</th><th class="col-janqty">JAN数量</th><th class="col-janpackqty">JAN包装数</th><th class="col-janunit">JAN単位</th><th class="col-amount">金額</th><th class="col-lot">ロット</th><th class="col-receipt">伝票番号</th><th class="col-ma">MA</th></tr></thead>`;
        
        let bodyHtml = `<tbody>${(!records || records.length === 0) ?
            '<tr><td colspan="13">対象データがありません。</td></tr>' : records.map(rec => {
            
            const top = `<tr><td rowspan="2" class="col-action center"><input type="checkbox" 
                 class="precomp-active-check" data-quantity="${rec.yjQuantity}" data-product-code="${rec.janCode}" checked></td>
                  <td class="col-date">${rec.transactionDate || ''}</td><td class="yj-jan-code col-yj">${rec.yjCode || ''}</td><td class="left col-product" colspan="2">${rec.productName || ''}</td>
                 <td class="right col-count">${rec.datQuantity?.toFixed(2) || ''}</td><td class="right col-yjqty">${rec.yjQuantity?.toFixed(2) || ''}</td><td class="right col-yjpackqty">${rec.yjPackUnitQty || ''}</td><td class="col-yjunit">${rec.yjUnitName || ''}</td>
                 <td class="right col-unitprice">${rec.unitPrice?.toFixed(4) || ''}</td><td class="col-expiry">${rec.expiryDate || ''}</td><td class="left col-wholesaler">${rec.clientCodeDisplay}</td><td class="right col-line">${rec.lineNumber || ''}</td></tr>`;
            
                const bottom = `<tr><td class="col-flag">${transactionTypeMap[rec.flag] || rec.flag}</td><td class="yj-jan-code col-jan">${rec.janCode || ''}</td><td class="left col-package">${rec.packageSpec || ''}</td><td class="left col-maker">${rec.makerName || ''}</td>
                 <td class="left col-form">${rec.usageClassification || ''}</td><td class="right col-janqty">${rec.janQuantity?.toFixed(2) || ''}</td><td class="right col-janpackqty">${rec.janPackUnitQty || ''}</td><td class="col-janunit">${rec.janUnitName || ''}</td>
                 <td class="right col-amount">${rec.subtotal?.toFixed(2) || ''}</td><td class="left col-lot">${rec.lotNumber || ''}</td><td class="left col-receipt">${rec.receiptNumber || ''}</td><td class="left col-ma">${rec.processFlagMA || ''}</td></tr>`;
                return top + bottom;
            }).join('')}</tbody>`;
        return `<table class="data-table">${header}${bodyHtml}</table>`;
    };

    return `<div class="summary-section" style="margin-top: 15px;">
        <div class="report-section-header"><h4>予製払出明細 (全体)</h4>
         <span class="header-total" id="precomp-active-total">有効合計: 0.00</span></div>
        ${renderTKRPrecompTable(precompTransactions)}</div>`;
}

export function createFinalInputRow(master, deadStockRecord = null, isPrimary = false) {
    const actionButtons = isPrimary ?
        `
          <button type="button" class="btn add-deadstock-row-btn" data-product-code="${master.productCode}">＋</button>
          <button type="button" class="btn register-inventory-btn">登録</button>
        ` : `<button type="button" class="btn delete-deadstock-row-btn">－</button>`;
    const quantityInputClass = isPrimary ?
        'final-inventory-input' : 'lot-quantity-input';
    const quantityPlaceholder = isPrimary ? '目安をここに転記' : 'ロット数量';
    const quantity = deadStockRecord ? deadStockRecord.stockQuantityJan : '';
    const expiry = deadStockRecord ? deadStockRecord.expiryDate : '';
    const lot = deadStockRecord ? deadStockRecord.lotNumber : '';

    const topRow = `<tr class="inventory-row"><td rowspan="2" class="col-action"><div class="action-buttons">${actionButtons}</div></td>
        <td class="col-date">(棚卸日)</td><td class="yj-jan-code col-yj">${master.yjCode}</td><td class="left col-product" colspan="2">${master.productName}</td>
         <td class="col-count"></td><td class="right col-yjqty"></td><td class="right col-yjpackqty">${master.yjPackUnitQty || ''}</td><td class="col-yjunit">${master.yjUnitName || ''}</td>
        <td class="right col-unitprice">${master.nhiPrice?.toFixed(4) || ''}</td><td class="col-expiry"><input type="text" class="expiry-input" placeholder="YYYYMM" value="${expiry}"></td><td class="col-wholesaler"></td><td class="col-line"></td></tr>`;
    
    const bottomRow = `<tr class="inventory-row"><td class="col-flag">棚卸</td><td class="yj-jan-code col-jan">${master.productCode}</td>
        <td class="col-package">${master.formattedPackageSpec || ''}</td><td class="col-maker">${master.makerName || ''}</td><td class="col-form">${master.usageClassification || ''}</td>
        <td class="right col-janqty"><input type="number" class="${quantityInputClass}" data-product-code="${master.productCode}" placeholder="${quantityPlaceholder}" value="${quantity}" step="any"></td>
        <td class="right col-janpackqty">${master.janPackUnitQty || ''}</td><td class="col-janunit">${master.janUnitName || ''}</td>
        <td class="right col-amount"></td><td class="col-lot"><input type="text" class="lot-input" placeholder="ロット番号" value="${lot}"></td><td class="col-receipt"></td><td class="col-ma"></td></tr>`;
    return topRow + bottomRow;
}

function generateInputSectionsHtml(packageLedgers, yjUnitName = '単位', cache, yesterdaysTotal) {
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
            const userInputArea = `
                 <div class="user-input-area">
                     <div class="form-group">
                         <label>① 本日の実在庫数量（予製除く）:</label>
                         <span
                             class="physical-stock-display" 
                             data-product-code="${master.productCode}">0.00</span>
                         <span>(${janUnitName})</span>
                         <span class="info-text">本日理論在庫(包装計): ${totalStockDisplay}</span>
                     </div>
                     <div class="form-group">
                         <label>② 前日在庫(逆算値):</label>
                         <span class="calculated-previous-day-stock" data-product-code="${master.productCode}">0.00</span>
                         <span>(${janUnitName})</span>
                         <span class="info-text stock-info">(この数値が棚卸データとして登録されます)</span>
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
                [], 
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

function generateDeadStockReferenceHtml(deadStockRecords, cache) {
    if (!deadStockRecords || deadStockRecords.length === 0) {
        const rowsHtml = '<tr><td colspan="5">品目を選択すると表示されます。</td></tr>';
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
        `;
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

export function generateFullHtml(data, cache) {

    if (!data || !data.transactionLedger || data.transactionLedger.length === 0) {
        
        const yesterdaysTotal = 0;
        const yjGroup = { productName: '(品目未選択)', yjCode: '---', packageLedgers: [], yjUnitName: '単位' }; 

        const summaryLedgerHtml = '<div class="summary-section"><h3 class="view-subtitle">1. 全体サマリー</h3><p>品目を選択すると表示されます。</p></div>';
        
        const summaryPrecompHtml = `<div class="summary-section" style="margin-top: 15px;">
           <div class="report-section-header"><h4>予製払出明細 (全体)</h4>
           <span class="header-total" id="precomp-active-total">有効合計: 0.00</span></div>
           <table class="data-table"><thead>
           <tr><th rowspan="2" class="col-action">有効</th><th class="col-date">日付</th><th class="col-yj">YJ</th><th colspan="2" class="col-product">製品名</th><th class="col-count">個数</th><th class="col-yjqty">YJ数量</th><th class="col-yjpackqty">YJ包装数</th><th class="col-yjunit">YJ単位</th><th class="col-unitprice">単価</th><th class="col-expiry">期限</th><th class="col-wholesaler">患者</th><th class="col-line">行</th></tr>
           <tr><th class="col-flag">種別</th><th class="col-jan">JAN</th><th class="col-package">包装</th><th class="col-maker">メーカー</th><th class="col-form">剤型</th><th class="col-janqty">JAN数量</th><th class="col-janpackqty">JAN包装数</th><th class="col-janunit">JAN単位</th><th class="col-amount">金額</th><th class="col-lot">ロット</th><th class="col-receipt">伝票番号</th><th class="col-ma">MA</th></tr></thead>
           <tbody><tr><td colspan="13">品目を選択すると表示されます。</td></tr></tbody></table></div>`;

        const inputSectionsHtml = generateInputSectionsHtml(yjGroup.packageLedgers, yjGroup.yjUnitName, cache, yesterdaysTotal);
        const deadStockReferenceHtml = generateDeadStockReferenceHtml([], cache);

        return `<h2 class="adj-header">【棚卸調整】 ${yjGroup.productName} (YJ: ${yjGroup.yjCode})</h2>
            ${summaryLedgerHtml}
            ${summaryPrecompHtml}
            ${inputSectionsHtml}
            ${deadStockReferenceHtml}`;
    }

    const yjGroup = data.transactionLedger[0];
    const productName = yjGroup.productName;
    const yesterdaysTotal = data.yesterdaysStock ? (data.yesterdaysStock.endingBalance || 0) : 0;
    
    const summaryLedgerHtml = generateSummaryLedgerHtml(yjGroup, yesterdaysTotal, cache);
    const summaryPrecompHtml = generateSummaryPrecompHtml(data.precompDetails);
    const inputSectionsHtml = generateInputSectionsHtml(yjGroup.packageLedgers, yjGroup.yjUnitName, cache, yesterdaysTotal);
    const deadStockReferenceHtml = generateDeadStockReferenceHtml(data.deadStockDetails, cache);
    
    return `<h2 class="adj-header">【棚卸調整】 ${productName} (YJ: ${yjGroup.yjCode})</h2>
        ${summaryLedgerHtml}
         ${summaryPrecompHtml}
        ${inputSectionsHtml}
        ${deadStockReferenceHtml}`;
}