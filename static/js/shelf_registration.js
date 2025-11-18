// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\shelf_registration.js
import { parseBarcode, fetchProductMasterByBarcode } from './utils.js';

let modal, closeBtn, barcodeForm, barcodeInput, shelfInput, listContainer, countSpan, executeBtn, clearBtn;
let scanList = []; // { productCode: string, productName: string, isNew: boolean }

export function initShelfRegistration() {
    modal = document.getElementById('shelfRegistrationModal');
    closeBtn = document.getElementById('closeShelfRegModalBtn');
    barcodeForm = document.getElementById('shelf-reg-barcode-form');
    barcodeInput = document.getElementById('shelf-reg-barcode');
    shelfInput = document.getElementById('shelf-reg-number');
    listContainer = document.getElementById('shelf-scan-list');
    countSpan = document.getElementById('shelf-reg-count');
    executeBtn = document.getElementById('shelf-reg-execute-btn');
    clearBtn = document.getElementById('shelf-reg-clear-btn');

    if (!modal) return;

    closeBtn.addEventListener('click', closeModal);
    barcodeForm.addEventListener('submit', handleScan);
    executeBtn.addEventListener('click', handleBulkRegister);
    clearBtn.addEventListener('click', clearList);
}

export function openShelfRegistrationModal() {
    if (modal) {
        modal.classList.remove('hidden');
        scanList = [];
        renderList();
        shelfInput.value = ''; // 初期化するか維持するかは要件次第だが一旦クリア
        barcodeInput.value = '';
        setTimeout(() => shelfInput.focus(), 100); // 最初は棚番入力へフォーカス
    }
}

function closeModal() {
    if (modal) modal.classList.add('hidden');
}

function clearList() {
    if (confirm('スキャン済みリストをクリアしますか？')) {
        scanList = [];
        renderList();
        barcodeInput.focus();
    }
}

async function handleScan(e) {
    e.preventDefault();
    const rawInput = barcodeInput.value.trim();
    if (!rawInput) return;

    // 棚番が入力されているかチェック（必須ではないが、警告は出さない）
    // 連続スキャンのリズムを崩さないため、棚番は後からでも入力可とする

    try {
        // 1. フォーマット検証 (utils.js の parseBarcode を利用)
        // これにより不正な文字列を弾く
        parseBarcode(rawInput); 
        
        // 2. サーバーへ問い合わせ (既存API利用: FindOrCreateMasterが走る)
        // これにより未知のJANなら自動的に仮登録される
        const master = await fetchProductMasterByBarcode(rawInput);

        // 3. リストに追加 (重複チェックはしない、最後にスキャンしたものが一番上に来る仕様)
        // ただし、全く同じJANが連続している場合は音を鳴らす等のフィードバックがあっても良いが、
        // ここでは単純に追加する
        
        // 新規作成かどうかは、origin が PROVISIONAL かつ、作成日時が直近...などの判定が難しいので
        // 簡易的に「サーバーから返ってきたが、scanListにまだ無い」ものを強調するわけではなく
        // サーバーレスポンスに isNew フラグはないため、運用上は「リストに出れば登録準備OK」とする。
        // UI上、JCSHMS由来かPROVISIONALかで色を変える手はある。
        const isProvisional = master.origin === 'PROVISIONAL';

        const item = {
            productCode: master.productCode,
            productName: master.productName,
            isProvisional: isProvisional,
            shelfNumber: master.shelfNumber || '' // 現在の棚番
        };

        // 先頭に追加
        scanList.unshift(item);
        renderList();

        // 成功音の代わりに、入力欄をクリアして次へ
        barcodeInput.value = '';

    } catch (err) {
        // エラー音を鳴らす代わりに通知を表示 (連続スキャンなので控えめに)
        console.error(err);
        window.showNotification(`エラー: ${err.message}`, 'error');
        barcodeInput.select(); // エラー時は選択状態にして書き換えやすくする
    }
}

function renderList() {
    if (!listContainer) return;
    
    listContainer.innerHTML = scanList.map(item => {
        const badge = item.isProvisional ? `<span class="new-badge">仮登録</span>` : '';
        const currentShelf = item.shelfNumber ? `(現: ${item.shelfNumber})` : '(棚番なし)';
        
        return `
            <div class="scanned-item ${item.isProvisional ? 'is-new' : ''}">
                <div class="item-main">
                    <span>${badge}${item.productName}</span>
                </div>
                <div class="item-sub">
                    <span>${item.productCode}</span>
                    <span>${currentShelf}</span>
                </div>
            </div>
        `;
    }).join('');

    if (countSpan) countSpan.textContent = scanList.length;
}

async function handleBulkRegister() {
    const shelfNumber = shelfInput.value.trim();
    if (!shelfNumber) {
        window.showNotification('登録する棚番を入力してください。', 'warning');
        shelfInput.focus();
        return;
    }

    if (scanList.length === 0) {
        window.showNotification('商品がスキャンされていません。', 'warning');
        barcodeInput.focus();
        return;
    }

    // ユニークなJANコードのリストを作成
    const productCodes = [...new Set(scanList.map(item => item.productCode))];

    if (!confirm(`棚番「${shelfNumber}」を、リストの${productCodes.length}件の品目に一括登録します。\nよろしいですか？`)) {
        return;
    }

    window.showLoading('棚番を一括更新中...');
    try {
        const res = await fetch('/api/masters/bulk_update_shelf', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                productCodes: productCodes,
                shelfNumber: shelfNumber
            })
        });

        const result = await res.json();
        if (!res.ok) {
            throw new Error(result.message || '更新に失敗しました。');
        }

        window.showNotification(result.message, 'success');
        
        // リセット
        scanList = [];
        renderList();
        barcodeInput.focus();

    } catch (err) {
        window.showNotification(err.message, 'error');
    } finally {
        window.hideLoading();
    }
}