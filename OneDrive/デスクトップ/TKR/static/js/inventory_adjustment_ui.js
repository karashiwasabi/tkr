// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\inventory_adjustment_ui.js
import { getLocalDateString } from './utils.js';

let unitMap = {};
// トランザクション種別マップ (TKR版)
const transactionTypeMap = {
	0: "棚卸", 1: "納品", 2: "返品", 3: "処方",
};
/**
 * 単位マップを外部から設定する
 * (WASABI: inventory_adjustment_ui.js より)
 * @param {object} map 
 */
export function setUnitMap(map) {
    unitMap = map;
}

/**
 * 標準的な取引履歴テーブルのHTMLを生成する
 * (WASABI: inventory_adjustment_ui.js より移植・TKR用に簡略化)
 */
function renderStandardTable(id, records) {
    const header = `<thead>
        <tr><th rowspan="2" class="col-action">－</th><th class="col-date">日付</th><th class="col-yj">YJ</th><th colspan="2" class="col-product">製品名</th><th class="col-count">個数</th><th class="col-yjqty">YJ数量</th><th class="col-yjpackqty">YJ包装数</th><th class="col-yjunit">YJ単位</th><th class="col-unitprice">単価</th><th></th><th class="col-expiry">期限</th><th class="col-wholesaler">卸</th><th class="col-line">行</th></tr>
        <tr><th class="col-flag">種別</th><th class="col-jan">JAN</th><th class="col-package">包装</th><th class="col-maker">メーカー</th><th class="col-form">剤型</th><th class="col-janqty">JAN数量</th><th class="col-janpackqty">JAN包装数</th><th class="col-janunit">JAN単位</th><th class="col-amount">金額</th><th></th><th class="col-lot">ロット</th><th class="col-receipt">伝票番号</th><th class="col-ma">MA</th></tr></thead>`;
    let bodyHtml = `<tbody>${(!records || records.length === 0) ?
        '<tr><td colspan="14">対象データがありません。</td></tr>' : records.map(rec => {
        
        // TKRでは卸マップはグローバルではないため、ここでは rec.clientCode をそのまま表示
        const clientDisplayHtml = rec.clientCode || '';

        const top = `<tr><td rowspan="2"></td>
            <td>${rec.transactionDate || ''}</td><td class="yj-jan-code">${rec.yjCode || ''}</td><td class="left" colspan="2">${rec.productName || ''}</td>
            <td class="right">${rec.datQuantity?.toFixed(2) || ''}</td><td class="right">${rec.yjQuantity?.toFixed(2) || ''}</td><td class="right">${rec.yjPackUnitQty || ''}</td><td>${rec.yjUnitName || ''}</td>
            <td class="right">${rec.unitPrice?.toFixed(4) || ''}</td><td></td><td>${rec.expiryDate || ''}</td><td class="left">${clientDisplayHtml}</td><td class="right">${rec.lineNumber || ''}</td></tr>`;
        const bottom = `<tr><td>${transactionTypeMap[rec.flag] || rec.flag}</td><td class="yj-jan-code">${rec.janCode || ''}</td><td>${rec.packageSpec || ''}</td><td>${rec.makerName || ''}</td>
            <td>${rec.usageClassification || ''}</td><td class="right">${rec.janQuantity?.toFixed(2) || ''}</td><td class="right">${rec.janPackUnitQty || ''}</td><td>${rec.janUnitName || ''}</td>
            <td class="right">${rec.subtotal?.toFixed(2) || ''}</td><td></td><td>${rec.lotNumber || ''}</td><td class="left">${rec.receiptNumber || ''}</td><td class="left">${rec.processFlagMA || ''}</td></tr>`;
        return top + bottom;
    }).join('')}</tbody>`;
    return `<table class="data-table" id="${id}">${header}${bodyHtml}</table>`;
}

/**
 * 「1. 全体サマリー」セクションのHTMLを生成する
 * (WASABI: inventory_adjustment_ui.js より移植)
 */
function generateSummaryLedgerHtml(yjGroup, yesterdaysTotal) {
    const endDate = getLocalDateString();
    const startDate = new Date();
    startDate.setDate(startDate.getDate() - 30); // TKRでも一旦30日固定で表示
    const startDateStr = startDate.toISOString().slice(0, 10);
    let packageLedgerHtml = (yjGroup.packageLedgers || []).map(pkg => {
        const sortedTxs = (pkg.transactions || []).sort((a, b) => 
            (a.transactionDate + a.id).toString().localeCompare(b.transactionDate + b.id)
        );
        // ▼▼▼【修正】style 属性を class 属性に変更 ▼▼▼
        const pkgHeader = `
            <div class="agg-pkg-header adj-pkg-margin">
                <span>包装: ${pkg.packageKey}</span>
                <span class="balance-info">
                    本日理論在庫(包装計): ${(pkg.endingBalance || 0).toFixed(2)} ${yjGroup.yjUnitName}
                </span>
            </div>
        `;
        // ▲▲▲【修正ここまで】▲▲▲
        const txTable = renderStandardTable(`ledger-table-${pkg.packageKey.replace(/[^a-zA-Z0-9]/g, '')}`, sortedTxs);
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
 * (WASABI: inventory_adjustment_ui.js より移植)
 */
export function createFinalInputRow(master, deadStockRecord = null, isPrimary = false) {
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

    // ▼▼▼【修正】style 属性を class 属性に変更 ▼▼▼
    const topRow = `<tr class="inventory-row"><td rowspan="2"><div class="action-buttons">${actionButtons}</div></td>
        <td>(棚卸日)</td><td class="yj-jan-code">${master.yjCode}</td><td class="left" colspan="2">${master.productName}</td>
        <td></td><td></td><td class="right">${master.yjPackUnitQty || ''}</td><td>${master.yjUnitName || ''}</td>
        <td></td><td></td><td><input type="text" class="expiry-input" placeholder="YYYYMM" value="${expiry}"></td><td></td><td></td></tr>`;
    // ▲▲▲【修正ここまで】▲▲▲
    const bottomRow = `<tr class="inventory-row"><td>棚卸</td><td class="yj-jan-code">${master.productCode}</td>
        <td>${master.formattedPackageSpec || ''}</td><td>${master.makerName || ''}</td><td>${master.usageClassification || ''}</td>
        <td><input type="number" class="${quantityInputClass}" data-product-code="${master.productCode}" placeholder="${quantityPlaceholder}" value="${quantity}" step="any"></td>
        <td class="right">${master.janPackUnitQty || ''}</td><td>${master.janUnitName || ''}</td>
        <td></td><td></td><td><input type="text" class="lot-input" placeholder="ロット番号" value="${lot}"></td><td></td><td></td></tr>`;
    
    return topRow + bottomRow;
}

/**
 * 「2. 棚卸入力」セクション全体のHTMLを生成する
 * (WASABI: inventory_adjustment_ui.js より移植)
 */
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
            const janUnitName = firstMaster.janUnitName; // ToProductMasterViewで既に解決済み
            
            if (janPackInnerQty > 0) {
                const totalJanStock = (pkgLedger.endingBalance || 0) / janPackInnerQty;
                totalStockDisplay = `${totalJanStock.toFixed(2)} ${janUnitName}`;
                
                const yesterdaysTotalJanStock = yesterdaysPkgStock / janPackInnerQty;
                yesterdaysTotalStockDisplay = `${yesterdaysTotalJanStock.toFixed(2)} ${janUnitName}`;
            }
        }

        // ▼▼▼【修正】style 属性を class 属性に変更 ▼▼▼
        let html = `
        <div class="package-input-group">
            <div class="agg-pkg-header">
                <span>包装: ${pkgLedger.packageKey}</span>
            </div>`;
        html += (pkgLedger.masters || []).map(master => {
            if (!master) return '';
            
            const janUnitName = master.janUnitName; // ToProductMasterViewで既に解決済み
            
            const userInputArea = `
                <div class="user-input-area">
                    <div class="form-group">
                        <label>① 本日の実在庫数量（予製除く）:</label>
                        <input type="number" class="physical-stock-input" data-product-code="${master.productCode}" step="any">
                        <span>(${janUnitName})</span>
                        <span class="info-text">本日理論在庫(包装計): ${totalStockDisplay}</span>
                    </div>
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
            const finalInputTable = renderStandardTable(`final-table-${master.productCode}`, 
                [], false, 
                `<tbody class="final-input-tbody" data-product-code="${master.productCode}">${finalInputTbodyHtml}</tbody>`);
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
    // ▲▲▲【修正ここまで】▲▲▲
}

/**
 * 「3. 参考」セクションのHTMLを生成する
 * (WASABI: inventory_adjustment_ui.js より移植)
 */
function generateDeadStockReferenceHtml(deadStockRecords, cache) {
    if (!deadStockRecords || deadStockRecords.length === 0) {
        return '';
    }

    const getProductName = (productCode) => {
        if (!cache || !cache.transactionLedger) return productCode;
        for (const yjGroup of cache.transactionLedger) {
            for (const pkg of yjGroup.packageLedgers) {
                const master = (pkg.masters || []).find(m => m.productCode === productCode);
                if (master) return master.productName;
            }
        }
        return productCode;
    };

    const rowsHtml = deadStockRecords.map(rec => `
        <tr>
            <td class="left">${getProductName(rec.productCode)}</td>
            <td class="right">${rec.stockQuantityJan.toFixed(2)}</td>
            <td>${rec.expiryDate || ''}</td>
            <td class="left">${rec.lotNumber || ''}</td>
        </tr>
    `).join('');
    // ▼▼▼【修正】style 属性を class 属性に変更 ▼▼▼
    return `
        <div class="summary-section input-section">
            <h3 class="view-subtitle">3. 参考：現在登録済みのロット・期限情報</h3>
            <p class="reference-section-header">※このリストは参照用です。棚卸情報を保存するには、上の「2. 棚卸入力」の欄に改めて入力してください。</p>
            <table class="data-table reference-table">
                <thead>
                    <tr>
                        <th class="col-ref-product">製品名</th>
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
    // ▲▲▲【修正ここまで】▲▲▲
}

/**
 * 棚卸調整画面の全HTMLを生成する
 * (WASABI: inventory_adjustment_ui.js より移植)
 */
export function generateFullHtml(data, cache) {
    if (!data.transactionLedger || data.transactionLedger.length === 0) {
        return '<p>対象の製品データが見つかりませんでした。</p>';
    }
    const yjGroup = data.transactionLedger[0];
    const productName = yjGroup.productName;
    const yesterdaysTotal = data.yesterdaysStock ? (data.yesterdaysStock.endingBalance || 0) : 0;
    
    const summaryLedgerHtml = generateSummaryLedgerHtml(yjGroup, yesterdaysTotal);
    // TKRには予製(Precomp)はないため、サマリーは削除
    const inputSectionsHtml = generateInputSectionsHtml(yjGroup.packageLedgers, yjGroup.yjUnitName, cache, yesterdaysTotal);
    const deadStockReferenceHtml = generateDeadStockReferenceHtml(data.deadStockDetails, cache);
    // ▼▼▼【修正】style 属性を class 属性に変更 ▼▼▼
    return `<h2 class="adj-header">【棚卸調整】 ${productName} (YJ: ${yjGroup.yjCode})</h2>
        ${summaryLedgerHtml}
        ${inputSectionsHtml}
        ${deadStockReferenceHtml}`;
    // ▲▲▲【修正ここまで】▲▲▲
}