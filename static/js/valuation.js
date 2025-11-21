// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\valuation.js

import { hiraganaToKatakana, getLocalDateString } from './utils.js';
import { showModal } from './search_modal.js'; // TKRの検索モーダル

let view, dateInput, runBtn, outputContainer, kanaNameInput, dosageFormInput, exportCsvBtn;
let searchModalBtn, searchStatusDiv; // 追加
let reportDataCache = null; // サーバーからの生データをキャッシュ

// (WASABI: valuation.js より)
const formatCurrency = (value) => {
    // TKRにはIntl.NumberFormatがないため、簡易的なカンマ区切り
    const num = Math.floor(value || 0);
    return `￥${num.toString().replace(/(\d)(?=(\d{3})+(?!\d))/g, '$1,')}`;
};

/**
 * サーバーからの計算結果 (インタラクティブビュー) を描画します
 */
function renderInteractiveView() {
    if (!reportDataCache || reportDataCache.length === 0) {
        outputContainer.innerHTML = '<p>表示するデータがありません。</p>';
        return;
    }

    let html = '';
    let grandTotalNhiValue = 0;
    let grandTotalPurchaseValue = 0;
    // TKRの剤型マップ
    const ucMap = {"内": "内", "外": "外", "歯": "歯", "注": "注", "機": "機", "他": "他"};

    reportDataCache.forEach(group => {
        grandTotalNhiValue += group.totalNhiValue;
        grandTotalPurchaseValue += group.totalPurchaseValue;

        const ucName = ucMap[group.usageClassification.trim()] || group.usageClassification;
        html += `<div class="agg-yj-header">${ucName} (合計薬価: ${formatCurrency(group.totalNhiValue)} | 合計納入価: ${formatCurrency(group.totalPurchaseValue)})</div>`;

        group.detailRows.forEach(row => {
            let warningHtml = '';
            if (row.showAlert) {
                warningHtml = `<span class="warning-link" data-product-code="${row.productCode}" style="color: red; font-weight: bold; cursor: pointer; text-decoration: underline; margin-left: 15px;">[JCSHMS採用]</span>`;
            }
           
            // TKRの valuation.css に合わせたクラス名
            html += `
                <div class="item-row">
                    <div class="item-row-left">
                        <span class="product-name">${row.productName}</span>
                        <span class="package-spec">${row.packageKey}</span>
                        ${warningHtml}
                    </div>
                    <div class="item-row-right">
                        <span class="value-item">在庫: <span class="value">${row.stock.toFixed(2)}</span> ${row.yjUnitName}</span>
                        <span class="value-item">納入価金額: <span class="value">${formatCurrency(row.totalPurchaseValue)}</span></span>
                        <span class="value-item">薬価金額: <span class="value">${formatCurrency(row.totalNhiValue)}</span></span>
                    </div>
                </div>
            `;
        });
    });
    html += `
        <div class="valuation-grand-total">
            <span>総合計 (薬価): ${formatCurrency(grandTotalNhiValue)}</span>
            <span>総合計 (納入価): ${formatCurrency(grandTotalPurchaseValue)}</span>
        </div>
    `;

    // 「最終帳票を作成」ボタンを削除し、代わりにCSS印刷用のボタンを残す
    html += `<div style="text-align: right; margin-top: 20px;"><button id="print-valuation-report-btn" class="btn" style="background-color: #198754; color: white;">この画面を印刷</button></div>`;
    outputContainer.innerHTML = html;
}


/**
 * 「在庫評価を実行」ボタンのハンドラ
 */
async function runCalculation() {
    const date = dateInput.value.replace(/-/g, '');
    if (!date) {
        window.showNotification('評価基準日を指定してください。', 'error');
        return;
    }
    window.showLoading('在庫評価を集計中...');
    try {
        const kanaName = hiraganaToKatakana(kanaNameInput.value);
        const dosageForm = dosageFormInput.value;
        const params = new URLSearchParams({
            date: date,
            kanaName: kanaName,
            dosageForm: dosageForm,
        });
        const res = await fetch(`/api/valuation?${params.toString()}`);
        if (!res.ok) {
            const errText = await res.text();
            throw new Error(errText || '在庫評価の計算に失敗しました。');
        }
        reportDataCache = await res.json(); 
        renderInteractiveView();

    } catch (err) {
        outputContainer.innerHTML = `<p style="color:red;">${err.message}</p>`;
    } finally {
        window.hideLoading();
    }
}

/**
 * 「CSVエクスポート」ボタンのハンドラ
 */
function handleExportCSV() {
    const date = dateInput.value.replace(/-/g, '');
    if (!date) {
        window.showNotification('評価基準日を指定してください。', 'error');
        return;
    }
    const kanaName = hiraganaToKatakana(kanaNameInput.value);
    const dosageForm = dosageFormInput.value;
    const params = new URLSearchParams({
        date: date,
        kanaName: kanaName,
        dosageForm: dosageForm,
    });
    window.location.href = `/api/valuation/export_csv?${params.toString()}`;
}

/**
 * 品目検索モーダルを開く
 */
function openSearchModal() {
    showModal(null, (selectedProduct) => {
        // 選択された品目の情報を隠しフィールドにセット
        if (kanaNameInput) kanaNameInput.value = selectedProduct.productName;
        if (searchStatusDiv) {
            searchStatusDiv.textContent = `絞り込み中: ${selectedProduct.productName}`;
            // キャンセルボタンのようなものを付けても良いが、ここではシンプルに表示のみ
        }
        // 自動的に実行
        runCalculation();
    }, {
        searchMode: 'default'
    });
}

/**
 * 在庫評価ビューの初期化
 */
export function initValuationView() {
    view = document.getElementById('valuation-view');
    if (!view) return;

    dateInput = document.getElementById('valuation-date');
    runBtn = document.getElementById('run-valuation-btn');
    outputContainer = document.getElementById('valuation-output-container');
    kanaNameInput = document.getElementById('val-kanaName');
    dosageFormInput = document.getElementById('val-dosageForm');
    exportCsvBtn = document.getElementById('export-valuation-csv-btn');
    
    // 追加要素
    searchModalBtn = document.getElementById('valOpenSearchModalBtn');
    searchStatusDiv = document.getElementById('val-search-status');

    dateInput.value = getLocalDateString();
    runBtn.addEventListener('click', runCalculation);
    
    if (exportCsvBtn) {
        exportCsvBtn.addEventListener('click', handleExportCSV);
    }

    if (searchModalBtn) {
        searchModalBtn.addEventListener('click', openSearchModal);
    }

    outputContainer.addEventListener('click', async (e) => {
        // [JCSHMS採用] リンクの処理
        if (e.target.classList.contains('warning-link')) {
            // ... (採用ロジックは変更なし) ...
            const productCode = e.target.dataset.productCode;
            showModal(e.target, async (selectedProduct) => {
                if (!confirm(`「${selectedProduct.productName}」をマスターに新規採用しますか？`)) {
                    return;
                }
                window.showLoading();
                try {
                    const res = await fetch('/api/master/adopt', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ productCode: selectedProduct.productCode, gs1Code: selectedProduct.gs1Code })
                    });
                    const resData = await res.json();
                    if (!res.ok) throw new Error(resData.message || '採用処理に失敗しました');
                    
                    window.showNotification(`「${resData.productName}」を登録しました。在庫評価を更新します。`, 'success');
                    await runCalculation();
                } catch (err) {
                    window.showNotification(err.message, 'error');
                } finally {
                    window.hideLoading();
                }
            }, {
                searchMode: 'inout', 
                copyOnly: false 
            });
        }
        
        // 「この画面を印刷」ボタン
        if (e.target.id === 'print-valuation-report-btn') {
            view.classList.add('print-this-view'); 
            window.print();
        }
    });

    window.addEventListener('afterprint', () => {
        view.classList.remove('print-this-view');
    });

    console.log("Valuation View Initialized.");
}