// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\return_list.js
import { hiraganaToKatakana } from './utils.js';

let outputContainer, kanaNameInput, dosageFormInput, coefficientInput, shelfNumberInput;
let runBtn;

export function initReturnListView() {
    outputContainer = document.getElementById('return-candidates-output');
    kanaNameInput = document.getElementById('return-kanaName');
    dosageFormInput = document.getElementById('return-dosageForm');
    coefficientInput = document.getElementById('return-coefficient');
    shelfNumberInput = document.getElementById('return-shelf-number');
    runBtn = document.getElementById('generate-return-candidates-btn');

    if (runBtn) {
        runBtn.addEventListener('click', handleGenerateReturnCandidates);
    }

    console.log("Return List View Initialized.");
}

async function handleGenerateReturnCandidates() {
    window.showLoading('返品候補リストを作成中...');
    
    const params = new URLSearchParams({
        kanaName: hiraganaToKatakana(kanaNameInput.value),
        dosageForm: dosageFormInput.value,
        shelfNumber: shelfNumberInput.value,
        coefficient: coefficientInput.value,
    });

    try {
        const res = await fetch(`/api/returns/candidates?${params.toString()}`);
        if (!res.ok) {
            const errText = await res.text();
            throw new Error(errText || 'リストの生成に失敗しました');
        }
        const candidates = await res.json();
        renderReturnTable(candidates);
    } catch (err) {
        outputContainer.innerHTML = `<p class="status-error">エラー: ${err.message}</p>`;
    } finally {
        window.hideLoading();
    }
}

function renderReturnTable(candidates) {
    if (!candidates || candidates.length === 0) {
        outputContainer.innerHTML = "<p>返品対象となる品目はありませんでした。</p>";
        return;
    }

    // style属性を削除し、classを使用
    let html = `
        <div class="return-list-condition">
            条件: 理論在庫 > ( 返品係数 × 発注点 + 予製 ) + 最小JAN包装数量<br>
            ※箱単位（包装単位）で返品可能な数量のみ表示しています。
        </div>
        <table class="data-table">
            <thead>
                <tr>
                    <th>製品名 (包装)</th>
                    <th>メーカー</th>
                    <th>最小包装単位</th>
                    <th>理論在庫</th>
                    <th>閾値</th>
                    <th>返品推奨数</th>
                    <th>単位</th>
                </tr>
            </thead>
            <tbody>
    `;

    candidates.forEach(item => {
        const master = item.representative;
        const spec = master.formattedPackageSpec || item.packageKey;
        
        // 箱数の表示（style削除、class使用）
        const boxInfo = item.returnableBoxes ? ` <span class="spec-info">(${item.returnableBoxes}箱)</span>` : '';

        html += `
            <tr>
                <td class="left">${item.productName}<br><span class="spec-info">${spec}</span></td>
                <td class="left">${master.makerName || ''}</td>
                <td class="right">${item.minJanPackQty}</td>
                <td class="right">${item.theoreticalStock.toFixed(2)}</td>
                <td class="right">${item.threshold.toFixed(2)}</td>
                <td class="right excess-qty">${item.excessQuantity.toFixed(2)}${boxInfo}</td>
                <td class="center">${item.unitName}</td>
            </tr>
        `;
    });

    html += `</tbody></table>`;
    outputContainer.innerHTML = html;
}