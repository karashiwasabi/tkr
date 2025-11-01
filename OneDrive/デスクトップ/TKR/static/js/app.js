// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\app.js
import { initDatUpload } from './dat.js';
import { initMasterEditView } from './masteredit.js';
import { initConfigView } from './config.js'; // ★【追加】

let loadingOverlay, loadingMessage, notificationBox;
let views, datViewBtn, masterEditViewBtn, configViewBtn; // ★【追加】configViewBtn

window.showLoading = (message = '処理中...') => {
    if (!loadingOverlay) loadingOverlay = document.getElementById('loading-overlay');
    if (!loadingMessage) loadingMessage = document.getElementById('loading-message');
    if (loadingMessage) loadingMessage.textContent = message;
    if (loadingOverlay) loadingOverlay.classList.remove('hidden');
};
window.hideLoading = () => {
    if (!loadingOverlay) loadingOverlay = document.getElementById('loading-overlay');
    if (loadingOverlay) loadingOverlay.classList.add('hidden');
};
window.showNotification = (message, type = 'success') => {
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

function setActiveView(targetId) {
    if (!views) views = document.querySelectorAll('.view');
    views.forEach(view => {
        if (view.id === targetId) {
            view.classList.add('active');
        } else {
            view.classList.remove('active');
        }
    });
    
    // ★【追加】カスタムイベントを発火させて、各JSモジュールにビューの変更を通知
    document.dispatchEvent(new CustomEvent('setActiveView', { detail: { viewId: targetId } }));

    if (targetId === 'master-edit-view') {
        const masterListBody = document.querySelector('#masterListTable tbody');
        if(masterListBody && masterListBody.children.length <= 1) {
             document.dispatchEvent(new CustomEvent('showMasterEditView'));
        }
    }
}

document.addEventListener('DOMContentLoaded', () => {
    console.log('TKR App Initialized.');

    loadingOverlay = document.getElementById('loading-overlay');
    loadingMessage = document.getElementById('loading-message');
    notificationBox = document.getElementById('notification-box');
    views = document.querySelectorAll('.view');
    datViewBtn = document.getElementById('datViewBtn');
    masterEditViewBtn = document.getElementById('masterEditViewBtn');
    configViewBtn = document.getElementById('configViewBtn'); // ★【追加】

    initDatUpload();
    initMasterEditView();
    initConfigView(); // ★【追加】

    if (datViewBtn) {
        datViewBtn.addEventListener('click', () => setActiveView('dat-upload-view'));
    }
    if (masterEditViewBtn) 
    {
        masterEditViewBtn.addEventListener('click', () => setActiveView('master-edit-view'));
    }
    // ★【追加】
    if (configViewBtn) {
        configViewBtn.addEventListener('click', () => setActiveView('config-view'));
    }

    setActiveView('dat-upload-view');
});
document.addEventListener('showMasterEditView', () => {
    console.log("Event: showMasterEditView received");
});