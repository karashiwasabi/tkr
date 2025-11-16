// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\masteredit_ui.js
import { wholesalerMap } from './master_data.js';

let editFormFields = {};

const jcshmsReadonlyKeys = [
    'yjCode', 'gs1Code', 'productName', 'kanaName', 'kanaNameShort', 'genericName',
    'makerName', 'specification', 'usageClassification', 'packageForm',
    'yjUnitName', 'yjPackUnitQty', 'janPackInnerQty', 'janUnitCode',
    'janPackUnitQty', 'nhiPrice', 'flagPoison', 'flagDeleterious',
    'flagNarcotic', 'flagPsychotropic', 'flagStimulant', 'flagStimulantRaw'
];

export function setFormFields(fields) {
    editFormFields = fields;
}

export function getFormFields() {
    return editFormFields;
}

export function populateFormWithJCSHMS(item) {
    console.log("--- JCSHMSデータコピー処理開始 ---");
    console.log("受け取ったJCSHMSデータ (ProductMasterView):", JSON.parse(JSON.stringify(item)));
    console.log("フォーム要素のマップ:", editFormFields);
    const populateAndLog = (key, value) => {
        const targetElement = editFormFields[key];
        if (targetElement) {
            const finalValue = (value === undefined || value === null) ? '' : value;
            console.log(`[処理中] 項目 '${key}' (要素ID: ${targetElement.id})`);
            console.log(`  - 値: `, finalValue);
            try {
                targetElement.value = finalValue;
                if (String(targetElement.value) === String(finalValue)) {
                    console.log(`  -> [成功] 値の設定を確認しました。`);
                } else {
                    console.warn(`  -> [検証失敗] 値を設定しましたが、読み取った値が異なります。設定後の値: '${targetElement.value}'`);
                }
            } catch (e) {
                console.error(`  -> [エラー] 値の設定中に例外が発生しました: ${e.message}`);
            }
        } else {
            console.warn(`[スキップ] 項目 '${key}' はフォーム内に対応する要素が見つかりません。`);
        }
    };

    populateAndLog('productName', (item.productName || '').trim());
    populateAndLog('yjCode', item.yjCode || '');
    populateAndLog('kanaName', (item.kanaName || '').trim());
    populateAndLog('kanaNameShort', (item.kanaNameShort || '').trim());
    populateAndLog('genericName', (item.genericName || '').trim());
    populateAndLog('makerName', (item.makerName || '').trim());
    populateAndLog('specification', (item.specification || '').trim());
    populateAndLog('usageClassification', (item.usageClassification || '').trim());
    populateAndLog('packageForm', (item.packageForm || '').trim());
    populateAndLog('yjUnitName', (item.yjUnitName || '').trim());
    populateAndLog('yjPackUnitQty', item.yjPackUnitQty || 0);
    populateAndLog('nhiPrice', item.nhiPrice || 0);
    populateAndLog('flagPoison', item.flagPoison || 0);
    populateAndLog('flagDeleterious', item.flagDeleterious || 0);
    populateAndLog('flagNarcotic', item.flagNarcotic || 0);
    populateAndLog('flagPsychotropic', item.flagPsychotropic || 0);
    populateAndLog('flagStimulant', item.flagStimulant || 0);
    populateAndLog('flagStimulantRaw', item.flagStimulantRaw || 0);
    populateAndLog('janPackInnerQty', item.janPackInnerQty || 0);
    populateAndLog('janUnitCode', parseInt(item.janUnitCode, 10) || 0);
    populateAndLog('janPackUnitQty', item.janPackUnitQty || 0);
    console.log("--- JCSHMSデータコピー処理終了 ---");
    window.showNotification('データコピー処理完了。', 'info');
}

export function populateEditForm(master) {
    console.log("Populating form for:", master);
    const viewProductCode = document.getElementById('view-product-code');
    if (viewProductCode) {
        viewProductCode.value = master.productCode || '';
    }

    const isJcshmsOrigin = master.origin === 'JCSHMS';
    const isMa2yCode = master.yjCode && master.yjCode.startsWith('MA2Y');
    const isNewCreation = !master.productCode;
    
    const referenceJCSHMSBtn = document.getElementById('referenceJCSHMSBtn');
    if (referenceJCSHMSBtn) {
        if (isJcshmsOrigin) {
            referenceJCSHMSBtn.style.display = 'none';
        } else {
            referenceJCSHMSBtn.style.display = 'inline-block';
        }
    }

    for (const key in editFormFields) {
        const element = editFormFields[key];
        const masterValue = master[key];

        if (element) {
            if (key === 'supplierWholesale' && element.tagName === 'SELECT') {
                element.value = masterValue || '';
            } 
            else if (typeof masterValue === 'number') {
                element.value = masterValue;
            } else {
                element.value = masterValue || '';
            }

            if (key !== 'productCode' && element.id !== 'view-product-code') {
                if (key === 'origin') {
                    element.readOnly = true;
                    element.classList.add('readonly-field');
                    if (isNewCreation) {
                        element.value = 'PROVISIONAL';
                    }
                } 
                else if (key === 'yjCode') {
                    if (isNewCreation) {
                        element.readOnly = false;
                        element.classList.remove('readonly-field');
                    } else if (isJcshmsOrigin && !isMa2yCode) {
                        element.readOnly = true;
                        element.classList.add('readonly-field');
                    } else {
                        element.readOnly = false;
                        element.classList.remove('readonly-field');
                    }
                } 
                else if (isJcshmsOrigin && jcshmsReadonlyKeys.includes(key)) {
                    element.readOnly = true;
                    element.classList.add('readonly-field'); 
                } else {
                    element.readOnly = false;
                    element.classList.remove('readonly-field');
                }
            }
        }
    }

    if (editFormFields['productCode']) {
        if (isNewCreation) {
            editFormFields['productCode'].value = ''; 
        }
    }
    if (viewProductCode) {
        if (isNewCreation) {
            viewProductCode.value = ''; 
            viewProductCode.readOnly = false; 
            viewProductCode.classList.remove('readonly-field');
            editFormFields['productCode'].value = ''; 
        } else {
            viewProductCode.readOnly = true; 
            viewProductCode.classList.add('readonly-field');
        }
    }
}

export function showEditModal() {
    const editFormContainer = document.getElementById('masterEditModalOverlay');
    if (editFormContainer) {
        editFormContainer.classList.remove('hidden');
    }
}

export function hideEditModal() {
    const editFormContainer = document.getElementById('masterEditModalOverlay');
    if (editFormContainer) {
        editFormContainer.classList.add('hidden');
    }
}

export function setupWholesalerDropdown() {
    const selectElement = editFormFields['supplierWholesale'];
    if (!selectElement || selectElement.tagName !== 'SELECT') {
        console.error("Wholesaler select element not found in edit form.");
        return;
    }
    selectElement.innerHTML = '<option value="">--- 選択なし ---</option>';
    wholesalerMap.forEach((name, code) => {
        const option = document.createElement('option');
        option.value = code;
        option.textContent = `${code}: ${name}`;
        selectElement.appendChild(option);
    });
}