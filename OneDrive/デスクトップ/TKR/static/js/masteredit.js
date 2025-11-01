// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\masteredit.js
import { hiraganaToKatakana } from './utils.js';

let viewElement, masterListTable, masterListBody, masterListHead;
let searchProdNameInput, searchKanaNameInput, searchGenericNameInput, searchShelfNumberInput, searchBtn;
let usageClassRadios;
let editFormContainer;

let saveMasterBtn, cancelEditMasterBtn;
let editFormFields = {};
let currentMasters = []; 

function renderEmptyMasterTable(statusMessage = '検索条件を指定して「検索」ボタンを押してください。') {
    if (!masterListTable) return;
    masterListTable.innerHTML = `
        <thead>
            <tr>
                <th class="col-action"></th>
                <th class="col-yj">YJコード</th>
                <th class="col-gs1">GS1コード</th>
                <th class="col-jan">JANコード</th>
                <th class="col-product">製品名</th>
                <th class="col-kana">カナ名</th>
                <th class="col-maker">メーカー</th>
                <th class="col-generic">一般名</th>
                <th class="col-shelf">棚番</th>
            </tr>
        </thead>
        <tbody>
            <tr><td colspan="9">${statusMessage}</td></tr>
        </tbody>
    `;
    masterListHead = masterListTable.querySelector('thead');
    masterListBody = masterListTable.querySelector('tbody');
}

function renderMasterTable(tableHTML) {
    if (!masterListTable) return;
    
    if (tableHTML) {
        masterListTable.innerHTML = tableHTML;
    } else {
        renderEmptyMasterTable('テーブルデータの受信に失敗しました。');
    }

    masterListHead = masterListTable.querySelector('thead');
    masterListBody = masterListTable.querySelector('tbody');
}


async function fetchAndRenderMasters() {
    console.log("[DEBUG] fetchAndRenderMasters called!");
    const selectedUsageRadio = document.querySelector('input[name="usage_class"]:checked');
    const usageClass = selectedUsageRadio ? selectedUsageRadio.value : '';

    const productName = searchProdNameInput ? searchProdNameInput.value.trim() : '';
    const kanaName = hiraganaToKatakana(searchKanaNameInput ? searchKanaNameInput.value.trim() : '');
    const genericName = searchGenericNameInput ? searchGenericNameInput.value.trim() : '';
    const shelfNumber = searchShelfNumberInput ? searchShelfNumberInput.value.trim() : '';

    if (!usageClass) {
        window.showNotification('内外注区分を選択してください。', 'warning');
        renderEmptyMasterTable('内外注区分を選択してください。');
        currentMasters = [];
        return;
    }

    hideEditModal();

    window.showLoading('マスターデータを検索中...');
    if (masterListTable) {
         masterListTable.innerHTML = `
            <thead>
                <tr>
                    <th class="col-action"></th>
                    <th class="col-yj">YJコード</th>
                    <th class="col-gs1">GS1コード</th>
                    <th class="col-jan">JANコード</th>
                    <th class="col-product">製品名</th>
                    <th class="col-kana">カナ名</th>
                    <th class="col-maker">メーカー</th>
                    <th class="col-generic">一般名</th>
                    <th class="col-shelf">棚番</th>
                </tr>
            </thead>
            <tbody>
                <tr><td colspan="9">検索中...</td></tr>
            </tbody>
        `;
    }

    const params = new URLSearchParams();
    params.append('usage_class', usageClass);
    if (productName.length > 0) params.append('product_name', productName);
    if (kanaName.length > 0) params.append('kana_name', kanaName);
    if (genericName.length > 0) params.append('generic_name', genericName);
    if (shelfNumber.length > 0) params.append('shelf_number', shelfNumber);
    const apiUrl = `/api/masters?${params.toString()}`;

    try {
        const response = await fetch(apiUrl);
        const result = await response.json();

        if (!response.ok) {
            throw new Error(result.message || `Failed to fetch masters: ${response.statusText}`);
        }

        if (result.masters && result.tableHTML != null) {
            currentMasters = Array.isArray(result.masters) ? result.masters : []; 
            renderMasterTable(result.tableHTML); 
            window.showNotification(`${currentMasters.length} 件見つかりました。`, 'info');
        } else {
            console.error("Received unexpected data format from API:", result);
            renderEmptyMasterTable('サーバーから予期しない形式のデータが返されました。');
            window.showNotification('サーバーから予期しない形式のデータが返されました。', 'error');
            currentMasters = [];
        }

    } catch (error) {
        console.error("Error fetching or rendering masters:", error);
        window.showNotification(`マスターの検索に失敗しました: ${error.message}`, 'error');
        renderEmptyMasterTable(`データの検索に失敗しました: ${error.message}`);
        currentMasters = [];
    } finally {
        window.hideLoading();
    }
}

function handleEditClick(event) {
     console.log("Table click detected.");
     const target = event.target;
     if (target.classList.contains('edit-master-btn')) {
        const productCode = target.dataset.code;
        if (!productCode) return;
        console.log("Edit button clicked for:", productCode);

        const masterToEdit = currentMasters.find(m => m.productCode === productCode);
        
        if (!masterToEdit) {
            console.error("Master data not found in cache for code:", productCode);
            window.showNotification('該当するマスターデータが見つかりません。', 'error');
            return;
        }

        populateEditForm(masterToEdit);
        showEditModal();
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
        'gs1Code', 'productName', 'kanaName', 'kanaNameShort', 'genericName',
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
            if (typeof masterValue === 'number') {
                element.value = masterValue;
            } else {
                element.value = masterValue || '';
            }

            if (key !== 'productCode' && element.id !== 'view-product-code') {
                if (isJcshmsOrigin && jcshmsReadonlyKeys.includes(key)) {
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

export function initMasterEditView() {
    viewElement = document.getElementById('master-edit-view');
    masterListTable = document.getElementById('masterListTable');

    searchProdNameInput = document.getElementById('master-search-prod-name');
    searchKanaNameInput = document.getElementById('master-search-kana-name');
    searchGenericNameInput = document.getElementById('master-search-generic-name');
    searchShelfNumberInput = document.getElementById('master-search-shelf-number');
    if (searchProdNameInput) searchProdNameInput.disabled = false;
    if (searchGenericNameInput) searchGenericNameInput.disabled = false;

    searchBtn = document.getElementById('masterSearchBtn');
    editFormContainer = document.getElementById('masterEditModalOverlay');
    usageClassRadios = document.querySelectorAll('input[name="usage_class"]');
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
        // ▼▼▼【修正】fieldIds のバグ修正 ▼▼▼
        let elementId;
        if (id === 'productCode') {
            elementId = 'edit-product-code'; 
        } else {
            elementId = `edit-${id.replace(/([A-Z])/g, '-$1').toLowerCase()}`;
        }
        // ▲▲▲【修正ここまで】▲▲▲
        
        const element = document.getElementById(elementId);
        if (element) {
            editFormFields[id] = element;
        }
    });

    if (saveMasterBtn) {
        saveMasterBtn.addEventListener('click', handleSaveMaster);
    }
    if (cancelEditMasterBtn) {
        cancelEditMasterBtn.addEventListener('click', () => {
            console.log("[DEBUG] Cancel button clicked!");
            hideEditModal();
            console.log("[DEBUG] Calling fetchAndRenderMasters after cancel...");
            fetchAndRenderMasters(); 
        });
        console.log("[DEBUG] Cancel button listener attached.");
    } else {
        console.error("Cancel button not found!");
    }

    searchBtn.addEventListener('click', fetchAndRenderMasters);
    const handleKeyPress = (event) => {
        if (event.key === 'Enter') {
            fetchAndRenderMasters();
        }
    };
    if (searchProdNameInput) searchProdNameInput.addEventListener('keypress', handleKeyPress);
    if (searchKanaNameInput) searchKanaNameInput.addEventListener('keypress', handleKeyPress);
    if (searchGenericNameInput) searchGenericNameInput.addEventListener('keypress', handleKeyPress);
    if (searchShelfNumberInput) searchShelfNumberInput.addEventListener('keypress', handleKeyPress);

    // ▼▼▼【修正】イベントリスナーを masterListTable に設定 ▼▼▼
    if (masterListTable) masterListTable.addEventListener('click', handleEditClick);
    
    renderEmptyMasterTable(); 
    // ▲▲▲【修正ここまで】▲▲▲

    console.log("Master Edit View Initialized.");
}