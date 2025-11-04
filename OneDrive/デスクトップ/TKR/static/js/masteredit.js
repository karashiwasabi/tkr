// C:\Users\wasab\OneDrive\デスケトッフ\TKR\static\js\masteredit.js
import { showModal } from './search_modal.js';
// ▼▼▼【修正】インポート先を変更 ▼▼▼
import { hiraganaToKatakana, fetchProductMasterByBarcode } from './utils.js';
import { wholesalerMap } from './master_data.js';
// ▲▲▲【修正ここまで】▲▲▲

let viewElement, masterListTable;
let searchKanaNameInput, searchGenericNameInput, searchShelfNumberInput, searchBtn;
let usageClassRadios;
let editFormContainer;

let saveMasterBtn, cancelEditMasterBtn;
let editFormFields = {};

let gs1Form, gs1Input;

// ▼▼▼【削除】キャッシュ変数は master_data.js にあるため不要 ▼▼▼
// let wholesalerCache = [];
// ▲▲▲【削除ここまで】▲▲▲


async function openProductSearchModalForMaster() {
    const apiUrl = '/api/products/search_filtered';
    const kanaInput = document.getElementById('master_search-kana');
    const genericInput = document.getElementById('master_search-generic');
    const shelfInput = document.getElementById('master_search-shelf');
    const selectedUsageRadio = document.querySelector('input[name="master_usage_class"]:checked');
    
    const kanaName = kanaInput ? hiraganaToKatakana(kanaInput.value.trim()) : '';
    const genericName = genericInput ? genericInput.value.trim() : '';
    const shelfNumber = shelfInput ? shelfInput.value.trim() : '';
    const usageClass = selectedUsageRadio ? selectedUsageRadio.value : '';

    if (!usageClass) {
        window.showNotification('内外注区分を選択してください。', 'warning');
        return;
    }

    const params = new URLSearchParams();
    params.append('kanaName', kanaName);
    params.append('genericName', genericName);
    params.append('shelfNumber', shelfNumber);
    params.append('dosageForm', usageClass);

    window.showLoading('品目リストを検索中...');
    let products = [];
    try {
        const fullUrl = `${apiUrl}?${params.toString()}`;
        const res = await fetch(fullUrl);
        if (!res.ok) {
            throw new Error(`品目リストの取得に失敗しました: ${res.status}`);
        }
        products = await res.json();
    } catch (err) {
        window.hideLoading();
        window.showNotification(err.message, 'error');
        return;
    } finally {
        window.hideLoading();
    }

    showModal(
        null,
        (selectedProduct) => { 
            populateEditForm(selectedProduct); 
            showEditModal();
        }, 
        { 
            initialResults: products, 
        }
    );
}

async function handleMasterGs1Scan(event) {
    event.preventDefault();
    if (!gs1Input) return;
    
    const barcode = gs1Input.value.trim();
    if (!barcode) {
        window.showNotification('バーコードが入力されていません。', 'warning');
        return;
    }

    window.showLoading('マスターを検索中...');
    try {
        const master = await fetchProductMasterByBarcode(barcode);
        
        populateEditForm(master);
        showEditModal();
        
    } catch (err) {
        window.showNotification(`エラー: ${err.message}`, 'error');
    } finally {
        window.hideLoading();
        gs1Input.value = ''; 
    }
}


function populateEditForm(master) {
     console.log("Populating form for:", master);
    const viewProductCode = document.getElementById('view-product-code');
    if (viewProductCode) {
        viewProductCode.value = master.productCode || '';
    }

    const isJcshmsOrigin = master.origin === 'JCSHMS';
    const jcshmsReadonlyKeys = [
        'yjCode', 'gs1Code', 'productName', 'kanaName', 'kanaNameShort', 'genericName',
        'makerName', 'specification', 'usageClassification', 'packageForm',
        'yjUnitName', 'yjPackUnitQty', 'janPackInnerQty', 'janUnitCode',
        'janPackUnitQty', 'nhiPrice', 'flagPoison', 'flagDeleterious',
        'flagNarcotic', 'flagPsychotropic', 'flagStimulant', 'flagStimulantRaw',
        'isOrderStopped'
    ];
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
}

function showEditModal() {
    if (editFormContainer) {
        editFormContainer.classList.remove('hidden');
    }
}

function hideEditModal() {
    if (editFormContainer) {
        editFormContainer.classList.add('hidden');
    }
}

async function handleSaveMaster() {
    const inputData = {};
    const floatKeys = [
        'yjPackUnitQty', 'janPackInnerQty', 'janPackUnitQty', 'nhiPrice', 'purchasePrice'
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

    if (!inputData.productCode) {
        window.showNotification('JANコードが不明なため保存できません。', 'error');
        return;
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
        window.showNotification(result.message || '保存しました。', 'success');
    } catch (error) {
        console.error('Save failed:', error);
        window.showNotification(`保存エラー: ${error.message}`, 'error');
    } finally {
        window.hideLoading();
    }
}

// ▼▼▼【修正】卸マスタ <select> 設定関数 (API呼び出しを削除し、キャッシュマップを使用) ▼▼▼
function setupWholesalerDropdown() {
    const selectElement = editFormFields['supplierWholesale'];
    if (!selectElement || selectElement.tagName !== 'SELECT') {
        console.error("Wholesaler select element not found in edit form.");
        return;
    }

    // (キャッシュ読み込みAPI呼び出しを削除)

    selectElement.innerHTML = '<option value="">--- 選択なし ---</option>';
    // グローバルな wholesalerMap を使用
    wholesalerMap.forEach((name, code) => {
        const option = document.createElement('option');
        option.value = code;
        option.textContent = `${code}: ${name}`;
        selectElement.appendChild(option);
    });
}
// ▲▲▲【修正ここまで】▲▲▲

export function initMasterEditView() {
    viewElement = document.getElementById('master-edit-view');
    masterListTable = document.getElementById('masterListTable');

    searchKanaNameInput = document.getElementById('master_search-kana');
    searchGenericNameInput = document.getElementById('master_search-generic');
    searchShelfNumberInput = document.getElementById('master_search-shelf');
    
    gs1Form = document.getElementById('master-barcode-form');
    gs1Input = document.getElementById('master-search-gs1-barcode');

    if (searchGenericNameInput) searchGenericNameInput.disabled = false;

    searchBtn = document.getElementById('masterSearchBtn');
    editFormContainer = document.getElementById('masterEditModalOverlay');
    usageClassRadios = document.querySelectorAll('input[name="master_usage_class"]');
    if (!viewElement || !masterListTable || !searchBtn || !editFormContainer) {
        console.error("Master edit view elements not found.");
        return;
    }

    saveMasterBtn = document.getElementById('saveMasterBtn');
    cancelEditMasterBtn = document.getElementById('cancelEditMasterBtn');
    const fieldIds = [
        'productCode', 'yjCode', 'gs1Code', 'productName', 'kanaName',
        'kanaNameShort', 'genericName', 'makerName', 'specification',
        'usageClassification', 'packageForm', 'yjUnitName', 'yjPackUnitQty',
        'janPackInnerQty', 'janUnitCode', 'janPackUnitQty', 'origin',
        'nhiPrice', 'purchasePrice', 'flagPoison', 'flagDeleterious',
        'flagNarcotic', 'flagPsychotropic', 'flagStimulant', 'flagStimulantRaw',
        'isOrderStopped', 'supplierWholesale', 'groupCode', 'shelfNumber',
        'category', 'userNotes'
    ];
    fieldIds.forEach(id => {
        let elementId;
        if (id === 'productCode') {
            elementId = 'edit-product-code'; 
        } else {
            elementId = `edit-${id.replace(/([A-Z])/g, '-$1').toLowerCase()}`;
        }
        
        const element = document.getElementById(elementId);
        if (element) 
 
{
            editFormFields[id] = element;
        }
    });

    setupWholesalerDropdown();

    if (saveMasterBtn) {
        saveMasterBtn.addEventListener('click', handleSaveMaster);
    }
    if (cancelEditMasterBtn) {
        cancelEditMasterBtn.addEventListener('click', () => {
            console.log("[DEBUG] Cancel button clicked!");
            hideEditModal(); 
        });
        console.log("[DEBUG] Cancel button listener attached.");
    } else {
        console.error("Cancel button not found!");
    }

    searchBtn.addEventListener('click', openProductSearchModalForMaster);
    
    if (gs1Form) {
        gs1Form.addEventListener('submit', handleMasterGs1Scan);
    } else {
        console.error("Master edit GS1 scan form not found.");
    }

    const handleKeyPress = (event) => {
        if (event.key === 'Enter') {
            openProductSearchModalForMaster();
        }
    };
    if (searchKanaNameInput) searchKanaNameInput.addEventListener('keypress', handleKeyPress);
    if (searchGenericNameInput) searchGenericNameInput.addEventListener('keypress', handleKeyPress);
    if (searchShelfNumberInput) searchShelfNumberInput.addEventListener('keypress', handleKeyPress);
    
    if (masterListTable) {
        masterListTable.innerHTML = '';
    }

    console.log("Master Edit View Initialized.");
}