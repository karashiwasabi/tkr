// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\common_ui.js
// (新規作成)

let loadingOverlay, loadingMessage, notificationBox;

/**
 * app.jsの起動時にDOM要素をキャッシュします。
 */
export function initUI() {
    loadingOverlay = document.getElementById('loading-overlay');
    loadingMessage = document.getElementById('loading-message');
    notificationBox = document.getElementById('notification-box');
}

/**
 * ローディングオーバーレイを表示します。
 * @param {string} message - 表示するメッセージ
 */
export const showLoading = (message = '処理中...') => {
    if (!loadingOverlay) loadingOverlay = document.getElementById('loading-overlay');
    if (!loadingMessage) loadingMessage = document.getElementById('loading-message');
    if (loadingMessage) loadingMessage.textContent = message;
    if (loadingOverlay) loadingOverlay.classList.remove('hidden');
};

/**
 * ローディングオーバーレイを非表示にします。
 */
export const hideLoading = () => {
    if (!loadingOverlay) loadingOverlay = document.getElementById('loading-overlay');
    if (loadingOverlay) loadingOverlay.classList.add('hidden');
};

/**
 * 通知メッセージを表示します。
 * @param {string} message - 表示するメッセージ
 * @param {'success'|'error'|'warning'|'info'} type - メッセージの種類
 */
export const showNotification = (message, type = 'success') => {
    if (!notificationBox) notificationBox = document.getElementById('notification-box');
    if (notificationBox) {
        notificationBox.textContent = message;
        notificationBox.className = 'notification-box';
        notificationBox.classList.add(type);
        notificationBox.classList.add('show');
        setTimeout(() => {
            notificationBox.classList.remove('show');
        }, 3000);
    }
};