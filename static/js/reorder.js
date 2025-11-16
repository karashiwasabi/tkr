// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\reorder.js
// (旧 reorder.js を分割し、ハブ機能として再構築)

// 1. 依存モジュールのインポート
import { initReorderEvents } from './reorder_events.js';
import { initContinuousScan } from './reorder_continuous_scan.js';

// 2. DOM要素 (リセット用)
let outputContainer, kanaNameInput, dosageFormInput, coefficientInput, shelfNumberInput;

/**
 * 発注点リストビューの初期化（メインハブ）
 * (旧 reorder.js より再構築)
 */
export function initReorderView() {
    // DOM要素の取得 (リセット処理 `fetchAndRenderReorder` で使用)
    outputContainer = document.getElementById('order-candidates-output');
    kanaNameInput = document.getElementById('order-kanaName');
    dosageFormInput = document.getElementById('order-dosageForm');
    coefficientInput = document.getElementById('order-reorder-coefficient');
    shelfNumberInput = document.getElementById('order-shelf-number');

    // 3. 連続スキャンモーダルの初期化
    initContinuousScan(); // from reorder_continuous_scan.js

    // 4. 全イベントリスナーの初期化
    // (CSV作成ボタン等が `fetchAndRenderReorder` を呼び出せるよう、コールバックとして渡す)
    initReorderEvents(fetchAndRenderReorder); // from reorder_events.js

    console.log("Reorder View Initialized (Hub).");
}

/**
 * 発注点リストのデータを取得して描画する (ビュー切り替え時のリセット関数)
 * (旧 reorder.js より)
 */
export async function fetchAndRenderReorder() {
    // (DOM要素は initReorderView でキャッシュ済み)
    if (outputContainer) {
        outputContainer.innerHTML = '<p>「発注候補を作成」ボタンを押してください。</p>';
    }
    if (kanaNameInput) kanaNameInput.value = '';
    if (dosageFormInput) dosageFormInput.value = '';
    if (shelfNumberInput) shelfNumberInput.value = '';
    if (coefficientInput) coefficientInput.value = '1.5';
}