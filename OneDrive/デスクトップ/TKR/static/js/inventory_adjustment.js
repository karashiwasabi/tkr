// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\inventory_adjustment.js
import { hiraganaToKatakana, getLocalDateString } from './utils.js';
// TKRではモーダルは masteredit.js が管理するため、ここでは直接参照しない

// ### グローバル変数 ###
let view, outputContainer;
let dosageFormFilter, kanaNameInput, selectProductBtn, barcodeInput, shelfNumberInput;
let currentYjCode = null;
let lastLoadedDataCache = null;
let unitMap = {};

// ### UI描画 (UI module) ###

const transactionTypeMap = {
	0: "棚卸", 1: "納品", 2: "返品", 3: "処方",
};

/**
 * 標準的な取引履歴テーブルのHTMLを生成する
 * [cite_start](WASABI: inventory_adjustment_ui.js [cite: 1162-1170] より移植・TKR用に簡略化)
 */
function renderStandardTable(id, records) {
    const header = `<thead>
        <tr><th rowspan="2">－</th><th>日付</th><th>YJ</th><th colspan="2">製品名</th><th>個数</th><th>YJ数量</th><th>YJ包装数</th><th>YJ単位</th><th>単価</th><th></th><th>期限</th><th>卸</th><th>行</th></tr>
        <tr><th>種別</th><th>JAN</th><th>包装</th><th>メーカー</th><th>剤型</th><th>JAN数量</th><th>JAN包装数</th><th>JAN単位</th><th>金額</th><th></th><th>ロット</th><th>伝票番号</th><th>MA</th></tr></thead>`;
    
    let bodyHtml = `<tbody>${(!records || records.length === 0) ?
        '<tr><td colspan="14">対象データがありません。</td></tr>' : records.map(rec => {
        
        // TKRの卸マップはグローバルではないため、ここでは rec.clientCode をそのまま表示
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
 * [cite_start](WASABI: inventory_adjustment_ui.js [cite: 1171-1177] より移植)
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
        const pkgHeader = `
            <div class="agg-pkg-header" style="margin-top: 10px;">
                <span>包装: ${pkg.packageKey}</span>
                <span class="balance-info">
                    本日理論在庫(包装計): ${(pkg.endingBalance || 0).toFixed(2)} ${yjGroup.yjUnitName}
                </span>
            </div>
        `;
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
 * [cite_start](WASABI: inventory_adjustment_ui.js [cite: 1181-1189] より移植)
 */
function createFinalInputRow(master, deadStockRecord = null, isPrimary = false) {
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

    const topRow = `<tr class="inventory-row"><td rowspan="2"><div style="display: flex; flex-direction: column; gap: 4px;">${actionButtons}</div></td>
        <td>(棚卸日)</td><td class="yj-jan-code">${master.yjCode}</td><td class="left" colspan="2">${master.productName}</td>
        <td></td><td></td><td class="right">${master.yjPackUnitQty || ''}</td><td>${master.yjUnitName || ''}</td>
        <td></td><td></td><td><input type="text" class="expiry-input" placeholder="YYYYMM" value="${expiry}"></td><td></td><td></td></tr>`;
    
    const bottomRow = `<tr class="inventory-row"><td>棚卸</td><td class="yj-jan-code">${master.productCode}</td>
        <td>${master.formattedPackageSpec || ''}</td><td>${master.makerName || ''}</td><td>${master.usageClassification || ''}</td>
        <td><input type="number" class="${quantityInputClass}" data-product-code="${master.productCode}" placeholder="${quantityPlaceholder}" value="${quantity}" step="any"></td>
        <td class="right">${master.janPackUnitQty || ''}</td><td>${master.janUnitName || ''}</td>
        <td></td><td></td><td><input type="text" class="lot-input" placeholder="ロット番号" value="${lot}"></td><td></td><td></td></tr>`;
    
    return topRow + bottomRow;
}

/**
 * 「2. 棚卸入力」セクション全体のHTMLを生成する
 * [cite_start](WASABI: inventory_adjustment_ui.js [cite: 1190-1216] より移植)
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

        let html = `
        <div class="package-input-group" style="margin-bottom: 20px;">
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
                        <span class="info-text" style="color: #dc3545;">(この数値が棚卸データとして登録されます)</span>
                        <span class="info-text" style="margin-left: auto;">前日理論在庫(包装計): ${yesterdaysTotalStockDisplay}</span>
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
                        <div style="margin-top: 10px;">
                            <p style="font-size: 10px; font-weight: bold; margin-bottom: 4px;">ロット・期限を個別入力</p>
                            ${finalInputTable}
                        </div>
                    </div>`;
        }).join('');

        html += `</div>`;
        return html;
    }).join('');
    
    return `<div class="input-section" style="margin-top: 20px;">
        <h3 class="view-subtitle">2. 棚卸入力</h3>
        <div class="inventory-input-area">
            <div style="display: flex; gap: 20px; align-items: flex-end;">
                <div class="form-group">
                    <label for="inventory-date">棚卸日:</label>
                    <input type="date" id="inventory-date" style="padding: 3px;">
                </div>
                <form id="adjustment-barcode-form" style="flex-grow: 1; margin: 0;">
                    <div class="form-group">
                        <label for="adjustment-barcode-input">バーコードでロット・期限入力</label>
                        <input type="text" id="adjustment-barcode-input" inputmode="latin" placeholder="GS1-128バーコードをスキャンしてEnter" style="ime-mode: disabled; font-size: 1.1em; padding: 3px;">
                    </div>
                </form>
            </div>
        </div>
        ${packageGroupsHtml}
    </div>`;
}

/**
 * 「3. 参考」セクションのHTMLを生成する
 * [cite_start](WASABI: inventory_adjustment_ui.js [cite: 1217-1226] より移植)
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

    return `
        <div class="summary-section" style="margin-top: 20px;">
            <h3 class="view-subtitle">3. 参考：現在登録済みのロット・期限情報</h3>
            <p style="font-size: 10px; margin-bottom: 5px;">※このリストは参照用です。棚卸情報を保存するには、上の「2. 棚卸入力」の欄に改めて入力してください。</p>
            <table class="data-table">
                <thead>
                    <tr>
                        <th style="width: 40%;">製品名</th>
                        <th style="width: 15%;">在庫数量(JAN)</th>
                        <th style="width: 20%;">使用期限</th>
                        <th style="width: 25%;">ロット番号</th>
                    </tr>
                </thead>
                <tbody>
                    ${rowsHtml}
                </tbody>
            </table>
        </div>
    `;
}

/**
 * 棚卸調整画面の全HTMLを生成する
 * [cite_start](WASABI: inventory_adjustment_ui.js [cite: 1227-1232] より移植)
 */
function generateFullHtml(data, cache) {
    lastLoadedDataCache = cache;
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

    return `<h2 style="text-align: center; margin-bottom: 15px; font-size: 1.2em;">【棚卸調整】 ${productName} (YJ: ${yjGroup.yjCode})</h2>
        ${summaryLedgerHtml}
        ${inputSectionsHtml}
        ${deadStockReferenceHtml}`;
}

// ### ロジック (Logic module) ###

/**
 * GS1-128バーコードを簡易解析する
 * [cite_start](WASABI: inventory_adjustment_logic.js [cite: 1037-1049] より移植)
 */
function parseGS1_128(code) {
    let rest = code;
    const data = {};

    if (rest.startsWith('01')) {
        if (rest.length < 16) return null;
        data.gs1Code = rest.substring(2, 16);
        rest = rest.substring(16);
    } else {
        return null;
    }

    if (rest.startsWith('17')) {
        if (rest.length < 8) return data;
        const yy_mm_dd = rest.substring(2, 8);
        if (yy_mm_dd.length === 6) {
            const yy = yy_mm_dd.substring(0, 2);
            const mm = yy_mm_dd.substring(2, 4);
            data.expiryDate = `20${yy}${mm}`; // YYYYMM に変換
        } else {
            data.expiryDate = yy_mm_dd;
        }
        rest = rest.substring(8);
    }

    if (rest.startsWith('10')) {
        const groupSeparatorIndex = rest.indexOf('\x1D'); // FNC1 (GS)
        if (groupSeparatorIndex !== -1) {
            data.lotNumber = rest.substring(2, groupSeparatorIndex);
        } else {
            data.lotNumber = rest.substring(2);
        }
    }
   
    return data;
}

/**
 * 棚卸入力画面でのバーコードスキャン処理
 * [cite_start](WASABI: inventory_adjustment_logic.js [cite: 1050-1070] より移植)
 */
async function handleAdjustmentBarcodeScan(e) {
    e.preventDefault();
    const barcodeInput = document.getElementById('adjustment-barcode-input');
    const inputValue = barcodeInput.value.trim();
    if (!inputValue) return;

    const parsedData = parseGS1_128(inputValue);
    if (!parsedData || !parsedData.gs1Code) {
        window.showNotification('GS1-128形式のバーコードではありません。', 'error');
        barcodeInput.value = '';
        return;
    }

    window.showLoading('製品情報を検索中...');
    try {
        const res = await fetch(`/api/product/by_gs1?gs1_code=${parsedData.gs1Code}`);
        let productMaster;
        if (!res.ok) {
            // TKRには仮マスター作成機能は移植しない（先にマスタ編集で作成してもらう）
            throw new Error('このGS1コードはマスターに登録されていません。');
        } else {
            productMaster = await res.json();
        }

        const productTbody = outputContainer.querySelector(`.final-input-tbody[data-product-code="${productMaster.productCode}"]`);
        if (!productTbody) {
            throw new Error(`画面内に製品「${productMaster.productName}」の入力欄が見つかりません。`);
        }

        let targetRow = null;
        const rows = productTbody.querySelectorAll('tr.inventory-row');
        for (let i = 0; i < rows.length; i += 2) {
            const expiryInput = rows[i].querySelector('.expiry-input');
            const lotInput = rows[i+1].querySelector('.lot-input');
            if (expiryInput.value.trim() === '' && lotInput.value.trim() === '') {
                targetRow = rows[i];
                break;
            }
        }

        if (!targetRow) {
            const addBtn = productTbody.querySelector('.add-deadstock-row-btn');
            if (addBtn) {
                addBtn.click();
                const newRows = productTbody.querySelectorAll('tr.inventory-row');
                targetRow = newRows[newRows.length - 2];
            }
        }

        if (targetRow) {
            const expiryInput = targetRow.querySelector('.expiry-input');
            const lotInput = targetRow.nextElementSibling.querySelector('.lot-input');
            if (parsedData.expiryDate) {
                expiryInput.value = parsedData.expiryDate;
            }
            if (parsedData.lotNumber) {
                lotInput.value = parsedData.lotNumber;
            }
            window.showNotification('ロット・期限を自動入力しました。', 'success');
        } else {
            throw new Error('ロット・期限の入力欄の追加に失敗しました。');
        }

    } catch (err) {
        window.showNotification(err.message, 'error');
    } finally {
        window.hideLoading();
        barcodeInput.value = '';
        barcodeInput.focus();
    }
}

/**
 * フィルターエリアでのバーコードスキャン処理
 * [cite_start](WASABI: inventory_adjustment_logic.js [cite: 1071-1081] より移植)
 */
async function handleBarcodeScan(e) {
    e.preventDefault();
    
    const barcodeInput = document.getElementById('ia-barcode-input');
    const inputValue = barcodeInput.value.trim();
    if (!inputValue) return;

    let gs1Code = '';
    if (inputValue.startsWith('01') && inputValue.length > 16) {
        const parsedData = parseGS1_128(inputValue);
        if (parsedData) {
            gs1Code = parsedData.gs1Code;
        }
    }
    
    if (!gs1Code) {
        // GS1(01)で始まらない場合、入力全体をGS1コードとみなす（TKRの運用に合わせる）
        gs1Code = inputValue;
    }

    if (!gs1Code) {
        window.showNotification('有効なGS1コードではありません。', 'error');
        return;
    }
   
    window.showLoading('製品情報を検索中...');
    try {
        const res = await fetch(`/api/product/by_gs1?gs1_code=${gs1Code}`);
        if (!res.ok) {
            throw new Error('このGS1コードはマスターに登録されていません。');
        } else {
            const productMaster = await res.json();
            await loadAndRenderDetails(productMaster.yjCode);
            barcodeInput.value = '';
            barcodeInput.focus();
        }
    } catch (err) {
        window.showNotification(err.message, 'error');
    } finally {
        window.hideLoading();
    }
}

/**
 * 「品目を選択...」ボタンの処理
 * [cite_start](WASABI: inventory_adjustment_logic.js [cite: 1088-1094] より移植)
 */
async function onSelectProductClick() {
    const dosageForm = dosageFormFilter.value;
    const kanaName = hiraganaToKatakana(kanaNameInput.value);
    const shelfNumber = shelfNumberInput.value.trim();
    
    const params = new URLSearchParams({
        dosageForm: dosageForm,
        kanaName: kanaName,
        shelfNumber: shelfNumber,
    });
    const apiUrl = `/api/products/search_filtered?${params.toString()}`;
    
    window.showLoading();
    try {
        const res = await fetch(apiUrl);
        if (!res.ok) throw new Error('品目リストの取得に失敗しました。');
        const products = await res.json();
        window.hideLoading();

        // TKRではモーダルは masteredit.js が管理している
        // 代わりに、検索結果の最初の品目を自動選択する
        if (products.length > 0) {
            const selectedProduct = products[0];
            loadAndRenderDetails(selectedProduct.yjCode);
        } else {
            window.showNotification('該当する品目が見つかりません。', 'warning');
        }

    } catch (err) {
        window.hideLoading();
        window.showNotification(err.message, 'error');
    }
}

/**
 * YJコードを指定して全データを読み込み、描画する
 * [cite_start](WASABI: inventory_adjustment_logic.js [cite: 1095-1104] より移植)
 */
async function loadAndRenderDetails(yjCode) {
    currentYjCode = yjCode;
    if (!yjCode) {
        window.showNotification('YJコードを指定してください。', 'error');
        return;
    }
    window.showLoading();
    outputContainer.innerHTML = '<p>データを読み込んでいます...</p>';
    try {
        const apiUrl = `/api/inventory/adjust/data?yjCode=${yjCode}`;
        const res = await fetch(apiUrl);
        if (!res.ok) {
            const errText = await res.text();
            throw new Error(errText || 'データ取得に失敗しました。');
        }
        
        lastLoadedDataCache = await res.json();
        const html = generateFullHtml(lastLoadedDataCache, lastLoadedDataCache);
        outputContainer.innerHTML = html;
        
        const dateInput = document.getElementById('inventory-date');
        if(dateInput) {
            // TKRでは日付を固定せず、当日の日付をデフォルトにする
            dateInput.value = getLocalDateString();
        }
    } catch (err) {
        outputContainer.innerHTML = `<p style="color:red;">エラー: ${err.message}</p>`;
    } finally {
        window.hideLoading();
    }
}

/**
 * 「本日の実在庫」入力時に「前日在庫（逆算値）」を計算する
 * [cite_start](WASABI: inventory_adjustment_logic.js [cite: 1114-1130] より移植)
 */
function reverseCalculateStock() {
    const todayStr = getLocalDateString().replace(/-/g, '');
    const calculationErrorByProduct = {};
    
    // TKRには予製(Precomp)はないため、予製関連ロジックは削除

    const todayNetChangeByProduct = {};
    if (lastLoadedDataCache && lastLoadedDataCache.transactionLedger) {
        lastLoadedDataCache.transactionLedger.forEach(yjGroup => {
            if (yjGroup.packageLedgers) {
                yjGroup.packageLedgers.forEach(pkg => {
                    if (pkg.transactions) {
                        pkg.transactions.forEach(tx => {
                            // TKRでは flag 0 (棚卸) は変動に含めない
                            if (tx.transactionDate === todayStr && tx.flag !== 0) {
                                let janQty = tx.janQuantity || 0;
                                if (janQty === 0 && tx.yjQuantity) {
                                    if (tx.janPackInnerQty > 0) {
                                        janQty = tx.yjQuantity / tx.janPackInnerQty;
                                    } else if (tx.yjQuantity !== 0) { // 0以外の変動があるのに内入数がない
                                        calculationErrorByProduct[tx.janCode] = '包装数量(内)未設定';
                                    }
                                }
                                // TKRの変動フラグ (1:納品, 3:処方)
                                const signedJanQty = janQty * (tx.flag === 1 ? 1 : (tx.flag === 3 ? -1 : 0));
                                todayNetChangeByProduct[tx.janCode] = (todayNetChangeByProduct[tx.janCode] || 0) + signedJanQty;
                            }
                        });
                    }
                });
            }
        });
    }

    document.querySelectorAll('.physical-stock-input').forEach(input => {
        const productCode = input.dataset.productCode;
        const displaySpan = document.querySelector(`.calculated-previous-day-stock[data-product-code="${productCode}"]`);
        const finalInput = document.querySelector(`.final-inventory-input[data-product-code="${productCode}"]`);

        if (calculationErrorByProduct[productCode]) {
            if (displaySpan) displaySpan.innerHTML = `<span style="color: red;">${calculationErrorByProduct[productCode]}</span>`;
            if (finalInput) finalInput.value = '';
            updateFinalInventoryTotal(productCode);
            return;
        }

        const physicalStockToday = parseFloat(input.value) || 0;
        // TKRには予製(Precomp)はない
        const totalStockToday = physicalStockToday;
        const netChangeToday = todayNetChangeByProduct[productCode] || 0;
        const calculatedPreviousDayStock = totalStockToday - netChangeToday;

        if (displaySpan) displaySpan.textContent = calculatedPreviousDayStock.toFixed(2);
        if (finalInput) {
            finalInput.value = calculatedPreviousDayStock.toFixed(2);
            updateFinalInventoryTotal(productCode);
        }
    });
}

/**
 * 転記欄・ロット入力欄の合計を計算（TKRでは合計表示はしないが、関数はロジックで必要）
 * [cite_start](WASABI: inventory_adjustment_logic.js [cite: 1131-1132] より)
 */
function updateFinalInventoryTotal(productCode) {
    const tbody = document.querySelector(`.final-input-tbody[data-product-code="${productCode}"]`);
    if (!tbody) return;
    let totalQuantity = 0;
    tbody.querySelectorAll('.final-inventory-input, .lot-quantity-input').forEach(input => {
        totalQuantity += parseFloat(input.value) || 0;
    });
    // TKRでは合計を表示するUIがないため、計算のみ
}

/**
 * キャッシュからマスターデータを検索する
 * [cite_start](WASABI: inventory_adjustment_logic.js [cite: 1133-1136] より)
 */
function findMaster(productCode) {
    if (!lastLoadedDataCache || !lastLoadedDataCache.transactionLedger || lastLoadedDataCache.transactionLedger.length === 0) {
        return null;
    }
    for (const pkgLedger of lastLoadedDataCache.transactionLedger[0].packageLedgers) {
        const master = (pkgLedger.masters || []).find(m => m.productCode === productCode);
        if (master) {
            return master;
        }
    }
    return null;
}

/**
 * 「登録」ボタン押下時の処理
 * [cite_start](WASABI: inventory_adjustment_logic.js [cite: 1137-1151] より移植)
 */
async function saveInventoryData() {
    const dateInput = document.getElementById('inventory-date');
    if (!dateInput || !dateInput.value) {
        window.showNotification('棚卸日を指定してください。', 'error');
        return;
    }
    if (!confirm(`${dateInput.value}の棚卸データとして保存します。よろしいですか？`)) return;

    const inventoryData = {}; // Key: ProductCode, Value: JanQuantity
    const deadStockData = [];
    
    if (!lastLoadedDataCache || !lastLoadedDataCache.transactionLedger || lastLoadedDataCache.transactionLedger.length === 0) {
        window.showNotification('保存対象の品目データが見つかりません。', 'error');
        return;
    }

    const allMasters = (lastLoadedDataCache.transactionLedger[0].packageLedgers || []).flatMap(pkg => pkg.masters || []);
    
    allMasters.forEach(master => {
        const productCode = master.productCode;
        const tbody = document.querySelector(`.final-input-tbody[data-product-code="${productCode}"]`);
        if (!tbody) {
            inventoryData[productCode] = 0; // 画面にない＝在庫0として登録
            return;
        };

        let totalInputQuantity = 0;
        const inventoryRows = tbody.querySelectorAll('.inventory-row');
        for (let i = 0; i < inventoryRows.length; i += 2) {
            const topRow = inventoryRows[i];
            const bottomRow = inventoryRows[i+1];
            const quantityInput = bottomRow.querySelector('.final-inventory-input, .lot-quantity-input');
            const expiryInput = topRow.querySelector('.expiry-input');
            const lotInput = bottomRow.querySelector('.lot-input');
            if (!quantityInput || !expiryInput || !lotInput) continue;
        
            const quantity = parseFloat(quantityInput.value) || 0;
            const expiry = expiryInput.value.trim();
            const lot = lotInput.value.trim();
            
            totalInputQuantity += quantity;
            
            // 在庫があり、かつロットまたは期限が入力されているものだけを dead_stock_list に保存
            if (quantity > 0 && (expiry || lot)) {
                deadStockData.push({ 
                    productCode: productCode, 
                    yjCode: master.yjCode, 
                    packageForm: master.packageForm,
                    janPackInnerQty: master.janPackInnerQty, 
                    yjUnitName: master.yjUnitName,
                    stockQuantityJan: quantity, 
                    expiryDate: expiry, 
                    lotNumber: lot 
                });
            }
        }
        inventoryData[productCode] = totalInputQuantity; // JAN単位の合計在庫
    });

    const payload = {
        date: dateInput.value.replace(/-/g, ''),
        yjCode: currentYjCode,
        inventoryData: inventoryData,
        deadStockData: deadStockData,
    };

    window.showLoading();
    try {
        const res = await fetch('/api/inventory/adjust/save', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(payload),
        });
        const resData = await res.json();
        if (!res.ok) throw new Error(resData.message || '保存に失敗しました。');
        
        window.showNotification(resData.message, 'success');
        // 保存成功後、データを再読み込み
        loadAndRenderDetails(currentYjCode);
    } catch (err) {
        window.showNotification(err.message, 'error');
    } finally {
        window.hideLoading();
    }
}

/**
 * ビューの初期化とイベントリスナーの設定
 * [cite_start](WASABI: inventory_adjustment_logic.js [cite: 1152-1159] より移植)
 */
export async function initInventoryAdjustment() {
    try {
        const res = await fetch('/api/units/map');
        if (!res.ok) throw new Error('単位マスタの取得に失敗');
        unitMap = await res.json();
    } catch (err) {
        console.error(err);
        window.showNotification(err.message, 'error');
    }

    view = document.getElementById('inventory-adjustment-view');
    if (!view) return;

    dosageFormFilter = document.getElementById('ia-dosageForm');
    kanaNameInput = document.getElementById('ia-kanaName');
    selectProductBtn = document.getElementById('ia-select-product-btn');
    outputContainer = document.getElementById('inventory-adjustment-output');
    barcodeInput = document.getElementById('ia-barcode-input');
    const barcodeForm = document.getElementById('ia-barcode-form');
    shelfNumberInput = document.getElementById('ia-shelf-number');

    if (barcodeForm) {
        barcodeForm.addEventListener('submit', handleBarcodeScan);
    }
    if (selectProductBtn) {
        selectProductBtn.addEventListener('click', onSelectProductClick);
    }
    
    outputContainer.addEventListener('input', (e) => {
        const targetClassList = e.target.classList;
        if (targetClassList.contains('physical-stock-input')) {
            reverseCalculateStock();
        }
        if(targetClassList.contains('lot-quantity-input') || targetClassList.contains('final-inventory-input')){
            const productCode = e.target.dataset.productCode;
            updateFinalInventoryTotal(productCode);
        }
    });

    outputContainer.addEventListener('click', (e) => {
        const target = e.target;
        if (target.classList.contains('add-deadstock-row-btn')) {
            const productCode = target.dataset.productCode;
            const master = findMaster(productCode);
            const tbody = document.querySelector(`.final-input-tbody[data-product-code="${productCode}"]`);
            if(master && tbody){
                const newRowHTML = createFinalInputRow(master, null, false);
                tbody.insertAdjacentHTML('beforeend', newRowHTML);
            }
        }
      
        if (target.classList.contains('delete-deadstock-row-btn')) {
            const topRow = target.closest('tr');
            const bottomRow = topRow.nextElementSibling;
            const productCode = bottomRow.querySelector('[data-product-code]')?.dataset.productCode;
            topRow.remove();
            bottomRow.remove();
            if(productCode) updateFinalInventoryTotal(productCode);
        }
        if (target.classList.contains('register-inventory-btn')) {
            saveInventoryData();
        }
    });

    outputContainer.addEventListener('submit', (e) => {
        if (e.target.id === 'adjustment-barcode-form') {
            handleAdjustmentBarcodeScan(e);
        }
    });

    // app.js からの 'setActiveView' イベントをリッスン
    document.addEventListener('loadInventoryAdjustment', (e) => {
        const { yjCode } = e.detail;
        if (yjCode) {
            // 他のビューからジャンプしてきた場合
            dosageFormFilter.value = '';
            kanaNameInput.value = '';
            shelfNumberInput.value = '';
            loadAndRenderDetails(yjCode);
        }
    });
}