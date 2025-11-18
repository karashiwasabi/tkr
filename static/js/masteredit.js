// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\masteredit.js
import { initSearchForm } from './masteredit_search.js';
import { initEditForm } from './masteredit_form.js';
// ▼▼▼ 追加 ▼▼▼
import { initShelfRegistration, openShelfRegistrationModal } from './shelf_registration.js';
// ▲▲▲ 追加ここまで ▲▲▲

let viewElement, masterListTable;
let usageClassRadios;
// ▼▼▼ 追加 ▼▼▼
let openShelfRegBtn;
// ▲▲▲ 追加ここまで ▲▲▲

export function initMasterEditView() {
    viewElement = document.getElementById('master-edit-view');
    masterListTable = document.getElementById('masterListTable');
    usageClassRadios = document.querySelectorAll('input[name="master_usage_class"]');
    // ▼▼▼ 追加 ▼▼▼
    openShelfRegBtn = document.getElementById('openShelfRegBtn');
    // ▲▲▲ 追加ここまで ▲▲▲

    if (!viewElement || !masterListTable) {
        console.error("Master edit view elements not found.");
        return;
    }

    initEditForm();
    initSearchForm();
    
    // ▼▼▼ 追加 ▼▼▼
    initShelfRegistration();
    if (openShelfRegBtn) {
        openShelfRegBtn.addEventListener('click', openShelfRegistrationModal);
    }
    // ▲▲▲ 追加ここまで ▲▲▲
    
    if (masterListTable) {
        masterListTable.innerHTML = '';
    }

    console.log("Master Edit View Initialized (Hub).");
}