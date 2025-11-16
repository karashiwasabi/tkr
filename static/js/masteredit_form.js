// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\masteredit_form.js
import { 
    setFormFields, 
    setupWholesalerDropdown, 
    hideEditModal as hideEditModalUI,
    populateEditForm as populateEditFormUI,
    showEditModal as showEditModalUI
} from './masteredit_ui.js';
import { handleSaveMaster, handleReferenceJCSHMS } from './masteredit_events.js';

let saveMasterBtn, cancelEditMasterBtn, referenceJCSHMSBtn;

export function initEditForm() {
    const editFormContainer = document.getElementById('masterEditModalOverlay');
    saveMasterBtn = document.getElementById('saveMasterBtn');
    cancelEditMasterBtn = document.getElementById('cancelEditMasterBtn');
    referenceJCSHMSBtn = document.getElementById('referenceJCSHMSBtn');
    const form = document.getElementById('masterEditForm');
    
    if (!form || !editFormContainer) {
        console.error("Master edit form or modal overlay not found!");
        return;
    }

    const editFormFields = {};
    const toCamelCase = (s) => {
        return s.replace(/^edit-/, '').replace(/-(\w)/g, (_, p1) => p1.toUpperCase());
    };

    form.querySelectorAll('input, select').forEach(element => {
        if (element.id) {
            if (element.id === 'edit-product-code') {
                editFormFields['productCode'] = element;
            } else if (element.id.startsWith('edit-')) {
                const key = toCamelCase(element.id);
                editFormFields[key] = element;
            }
        }
    });
    
    setFormFields(editFormFields);
    setupWholesalerDropdown();
    
    if (saveMasterBtn) {
        saveMasterBtn.addEventListener('click', handleSaveMaster);
    }
    if (cancelEditMasterBtn) {
        cancelEditMasterBtn.addEventListener('click', hideEditModalUI);
    }
    if (referenceJCSHMSBtn) {
        referenceJCSHMSBtn.addEventListener('click', handleReferenceJCSHMS);
    }
    
    console.log("Initialized editFormFields map:", editFormFields);
}

export function populateEditForm(master) {
    populateEditFormUI(master);
}

export function showEditModal() {
    showEditModalUI();
}

export function hideEditModal() {
    hideEditModalUI();
}