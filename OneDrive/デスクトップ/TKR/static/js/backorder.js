// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\backorder.js
import { hiraganaToKatakana } from './utils.js';
import { wholesalerMap } from './master_data.js';

let view, outputContainer, searchKanaInput, searchWholesalerInput, searchBtn;
let allBackorders = []; // APIから取得した全発注残をキャッシュ

/**
 * 包装仕様（簡易版）を生成します。
 * [cite_start](WASABI: backorder.js [cite: 38] のロジックをTKR用に移植)
 */
function formatSimplePackageSpec(bo) {
    // TKRのboモデルには `janPackInnerQty` と `yjUnitName` がある
    if (bo.janPackInnerQty > 0) {
        return `${bo.packageForm || ''} ${bo.janPackInnerQty}${bo.yjUnitName || ''}`;
    }
    // フォールバック
    return `${bo.packageForm || ''} ${bo.yjPackUnitQty || 0}${bo.yjUnitName || ''}`;
}

/**
 * 絞り込みと描画を実行します。
 */
function filterAndRender() {
    if (!outputContainer) return;

    const kanaFilter = hiraganaToKatakana(searchKanaInput.value.trim().toLowerCase());
    const wholesalerFilter = searchWholesalerInput.value.trim().toLowerCase();

    const filteredData = allBackorders.filter(bo => {
        const nameMatch = !kanaFilter || 
                          (bo.productName && bo.productName.toLowerCase().includes(kanaFilter)) ||
                          (bo.yjCode && bo.yjCode.toLowerCase().includes(kanaFilter));
        
        const wholesalerMatch = !wholesalerFilter ||
                                (bo.wholesalerCode && bo.wholesalerCode.toLowerCase().includes(wholesalerFilter));

        return nameMatch && wholesalerMatch;
    });

    renderBackorders(filteredData);
}

/**
 * 発注残リストのテーブルHTMLを描画します。
 * [cite_start](WASABI: backorder.js [cite: 39] を TKR 用に修正)
 */
function renderBackorders(data) {
    if (!data || data.length === 0) {
        outputContainer.innerHTML = "<p>対象の発注残はありません。</p>";
        return;
    }

    let html = `
        <div class="backorder-controls">
            <button class="btn" id="bo-bulk-delete-btn">選択した項目を一括削除</button>
        </div>
        <table id="backorder-table" class="data-table">
            <thead>
                <tr>
                    <th class="col-bo-check"><input type="checkbox" id="bo-select-all-checkbox"></th>
                    <th class="col-bo-date">発注日</th>
                    <th class="col-bo-yj">YJコード</th>
                    <th class="col-bo-name">製品名</th>
                    <th class="col-bo-spec">包装仕様</th>
                    <th class="col-bo-order">発注数量(YJ)</th>
                    <th class="col-bo-remain">残数量(YJ)</th>
                    <th class="col-bo-action">操作</th>
                </tr>
            </thead>
            <tbody>
    `;
    data.forEach(bo => {
        const pkgSpec = formatSimplePackageSpec(bo);
        const wholesalerName = wholesalerMap.get(bo.wholesalerCode) || bo.wholesalerCode || '---';

        html += `
            <tr data-id="${bo.id}" data-yj-code="${bo.yjCode}">
                <td class="center col-bo-check"><input type="checkbox" class="bo-select-checkbox"></td>
                <td class="col-bo-date">${bo.orderDate}</td>
                <td class="col-bo-yj">${bo.yjCode}</td>
                <td class="left col-bo-name">${bo.productName}</td>
                <td class="left col-bo-spec">${pkgSpec} [${wholesalerName}]</td>
                <td class="right col-bo-order">${bo.orderQuantity.toFixed(2)}</td>
                <td class="right col-bo-remain">${bo.remainingQuantity.toFixed(2)}</td>
                <td class="center col-bo-action">
                    <button class="btn delete-backorder-btn">削除</button>
                    <button class="btn adjust-inventory-btn" data-yj-code="${bo.yjCode}">棚卸調整</button>
                </td>
            </tr>
        `;
    });
    html += `</tbody></table>`;
    outputContainer.innerHTML = html;
}

/**
 * APIから発注残リストを取得し、キャッシュと描画を行います。
 * [cite_start](WASABI: backorder.js [cite: 67] を TKR 用に修正)
 */
async function loadAndRenderBackorders() {
    outputContainer.innerHTML = '<p>読み込み中...</p>';
    try {
        // TKRには /api/backorders はまだないので、/api/reorder/list (仮) を使用
        // → WASABIの /api/backorders を移植する
        const res = await fetch('/api/backorders');
        if (!res.ok) throw new Error('発注残リストの読み込みに失敗しました。');
        allBackorders = await res.json();
        
        // 絞り込み検索欄の値をリセット
        searchKanaInput.value = '';
        searchWholesalerInput.value = '';

        // 絞り込まずに全件描画
        renderBackorders(allBackorders);
    } catch (err) {
        outputContainer.innerHTML = `<p class="status-error">${err.message}</p>`;
    }
}

/**
 * 発注残ビューのイベントハンドラ
 * [cite_start](WASABI: backorder.js [cite: 70] を TKR 用に修正)
 */
async function handleBackorderEvents(e) {
    const target = e.target;

    // 個別削除ボタン
    if (target.classList.contains('delete-backorder-btn')) {
        const row = target.closest('tr');
        if (!confirm(`「${row.cells[3].textContent}」の発注残（発注日: ${row.cells[1].textContent}）を削除しますか？`)) {
            return;
        }
        const payload = {
            id: parseInt(row.dataset.id, 10),
        };
        window.showLoading();
        try {
            const res = await fetch('/api/backorders/delete', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
            });
            const resData = await res.json();
            if (!res.ok) throw new Error(resData.message || '削除に失敗しました。');
            window.showNotification(resData.message, 'success');
            loadAndRenderBackorders(); // リストを再読み込み
        } catch (err) {
            window.showNotification(err.message, 'error');
        } finally {
            window.hideLoading();
        }
    }

    // 全選択チェックボックス
    if (target.id === 'bo-select-all-checkbox') {
        const isChecked = target.checked;
        document.querySelectorAll('.bo-select-checkbox').forEach(cb => cb.checked = isChecked);
    }

    // 選択項目の一括削除ボタン
    if (target.id === 'bo-bulk-delete-btn') {
        const checkedRows = document.querySelectorAll('.bo-select-checkbox:checked');
        if (checkedRows.length === 0) {
            window.showNotification('削除する項目が選択されていません。', 'error');
            return;
        }
        if (!confirm(`${checkedRows.length}件の発注残を削除します。よろしいですか？`)) {
            return;
        }

        const payload = Array.from(checkedRows).map(cb => {
            const row = cb.closest('tr');
            return {
                id: parseInt(row.dataset.id, 10),
            };
        });
        window.showLoading();
        try {
            const res = await fetch('/api/backorders/bulk_delete', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(payload),
            });
            const resData = await res.json();
            if (!res.ok) throw new Error(resData.message || '一括削除に失敗しました。');
            window.showNotification(resData.message, 'success');
            loadAndRenderBackorders(); // リストを再読み込み
        } catch (err) {
            window.showNotification(err.message, 'error');
        } finally {
            window.hideLoading();
        }
    }

    // 棚卸調整ボタン
    if (target.classList.contains('adjust-inventory-btn')) {
        const yjCode = target.dataset.yjCode;
        if (!yjCode) return;

        // 棚卸調整ビューに移動するイベントを発火
        const event = new CustomEvent('loadInventoryAdjustment', {
            detail: { yjCode: yjCode },
            bubbles: true
        });
        document.dispatchEvent(event);

        // 棚卸調整タブをアクティブにする
        document.getElementById('inventoryAdjustmentViewBtn')?.click();
    }
}

export function initBackorderView() {
    view = document.getElementById('backorder-view');
    if (!view) return;
    outputContainer = document.getElementById('backorder-output-container');
    searchKanaInput = document.getElementById('bo-search-kana');
    searchWholesalerInput = document.getElementById('bo-search-wholesaler');
    searchBtn = document.getElementById('bo-search-btn');
    
    // 'show' イベントはTKRにはないので、app.js側で直接 loadAndRenderBackorders を呼ぶ
    view.addEventListener('show', loadAndRenderBackorders);

    outputContainer.addEventListener('click', handleBackorderEvents);
    
    searchBtn.addEventListener('click', filterAndRender);
    searchKanaInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') filterAndRender();
    });
    searchWholesalerInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') filterAndRender();
    });
}