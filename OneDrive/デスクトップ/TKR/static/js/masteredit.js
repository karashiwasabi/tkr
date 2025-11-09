import { initSearchForm } from './masteredit_search.js';
import { initEditForm } from './masteredit_form.js';

let viewElement, masterListTable;
let usageClassRadios;

export function initMasterEditView() {
    viewElement = document.getElementById('master-edit-view');
    masterListTable = document.getElementById('masterListTable');
    usageClassRadios = document.querySelectorAll('input[name="master_usage_class"]');

    if (!viewElement || !masterListTable) {
        console.error("Master edit view elements not found.");
        return;
    }

    initEditForm();
    initSearchForm();
    
    if (masterListTable) {
        masterListTable.innerHTML = '';
    }

    console.log("Master Edit View Initialized (Hub).");
}