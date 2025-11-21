// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\reorder.js
import { initReorderEvents } from './reorder_events.js';
import { initContinuousScan } from './reorder_continuous_scan.js';

// DOM要素 (リセット用)
let outputContainer, coefficientInput;

/**
 * 発注点リストビューの初期化（メインハブ）
 */
export function initReorderView() {
    // DOM要素の取得 (リセット処理 `fetchAndRenderReorder` で使用)
    outputContainer = document.getElementById('order-candidates-output');
    // ▼▼▼ 修正: 不要な要素取得を削除 ▼▼▼
    coefficientInput = document.getElementById('order-reorder-coefficient');
    // ▲▲▲ 修正ここまで ▲▲▲

    // 3. 連続スキャンモーダルの初期化
    initContinuousScan();

    // 4. 全イベントリスナーの初期化
    initReorderEvents(fetchAndRenderReorder);

    console.log("Reorder View Initialized (Hub).");
}

/**
 * 発注点リストのデータを取得して描画する (ビュー切り替え時のリセット関数)
 */
export async function fetchAndRenderReorder() {
    if (outputContainer) {
        outputContainer.innerHTML = '<p>「発注候補を作成」ボタンを押してください。</p>';
    }
    // ▼▼▼ 修正: 不要な入力欄のリセットを削除 ▼▼▼
    if (coefficientInput) coefficientInput.value = '1.5';
    // ▲▲▲ 修正ここまで ▲▲▲
}