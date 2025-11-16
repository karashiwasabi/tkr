// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\masteredit_events.js
import { showModal } from './search_modal.js';
import { getFormFields, populateFormWithJCSHMS } from './masteredit_ui.js';

export async function handleSaveMaster() {
    const editFormFields = getFormFields();
    const viewProductCode = document.getElementById('view-product-code');
    if (viewProductCode && editFormFields['productCode']) {
        if (!viewProductCode.readOnly) {
            editFormFields['productCode'].value = viewProductCode.value.trim();
        }
    }

    const inputData = {};
    const floatKeys = [
        'yjPackUnitQty', 'janPackInnerQty', 'janPackUnitQty', 
'nhiPrice', 'purchasePrice'
    ];
    const intKeys = [
        'janUnitCode', 'flagPoison', 'flagDeleterious', 'flagNarcotic',
        'flagPsychotropic', 'flagStimulant', 'flagStimulantRaw', 'isOrderStopped'
    ];
    for (const key in editFormFields) {
        const element = editFormFields[key];
        if (element) {
            const value = element.value;
            if (floatKeys.includes(key)) {
                inputData[key] = parseFloat(value) || 0;
            } else if (intKeys.includes(key)) {
                inputData[key] = parseInt(value, 10) || 0;
            } else {
                inputData[key] = value;
            }
        }
    }

    console.log("Saving master:", inputData);
    window.showLoading('マスターデータを保存中...');

    try {
        const response = await fetch('/api/masters/update', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(inputData),
        });
        if (!response.ok) {
            let errorText = `サーバーエラー (HTTP ${response.status})`;
            try {
                const text = await response.text();
                errorText = text || errorText;
            } catch (e) {
            }
            throw new Error(errorText);
        }

        const result = await response.json();
        
        let message = result.message || '保存しました。';
        let messageType = 'success';

        if (result.alert) {
            message += `\n\n警告: ${result.alert}`;
            messageType = 'warning'; 
        }
        window.showNotification(message, messageType);

    } catch (error) {
        console.error('Save failed:', error);
        window.showNotification(`保存エラー: ${error.message}`, 'error');
    } finally {
        window.hideLoading();
    }
}

export function handleReferenceJCSHMS() {
    showModal(
        null, 
        (selectedProduct) => {
            populateFormWithJCSHMS(selectedProduct);
        }, 
        {
            searchMode: 'inout',
            copyOnly: true
        }
    );
}