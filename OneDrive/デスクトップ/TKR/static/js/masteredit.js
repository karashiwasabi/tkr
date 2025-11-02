// C:\Users\wasab\OneDrive\デスケトッフ\TKR\static\js\masteredit.js
// ▼▼▼【修正】showModal と hiraganaToKatakana をインポート ▼▼▼
import { showModal } from './search_modal.js';
import { hiraganaToKatakana, fetchProductMasterByBarcode } from './utils.js';
// ▲▲▲【修正ここまで】▲▲▲

let viewElement, masterListTable; // 削除: masterListBody, masterListHead
let searchKanaNameInput, searchGenericNameInput, searchShelfNumberInput, searchBtn;
let usageClassRadios;
let editFormContainer;

let saveMasterBtn, cancelEditMasterBtn;
let editFormFields = {};
// 削除: currentMasters = [];

let gs1Form, gs1Input;

// ▼▼▼【ここから削除】一覧表示関連の関数をすべて削除 ▼▼▼
/*
function renderEmptyMasterTable(...) { ... }
function renderMasterTable(...) { ... }
export async function fetchAndRenderMasters() { ... }
*/
// ▲▲▲【削除ここまで】▲▲▲

// ▼▼▼【ここから追加】品目検索モーダルを開く関数（dat.js などから流用） ▼▼▼
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

    // モーダルを表示
    showModal(
        null, // マスタ編集では特定の行コンテキストは不要
        (selectedProduct) => { 
            // モーダルで品目が選択されたときのコールバック
            // ProductMasterView (selectedProduct) をフォームに投入
            populateEditForm(selectedProduct); 
            // 編集モーダルを開く
            showEditModal();
        }, 
        { 
            initialResults: products, 
        }
    );
}
// ▲▲▲【追加ここまで】▲▲▲


// GS1スキャン処理（変更なし、ただし currentMasters への保存処理を削除）
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
        
        // ▼▼▼【削除】currentMasters へのキャッシュ処理 ▼▼▼
        // const index = currentMasters.findIndex(m => m.productCode === master.productCode);
        // ...
        // ▲▲▲【削除ここまで】▲▲▲

        populateEditForm(master);
        showEditModal();
        
    } catch (err) {
        window.showNotification(`エラー: ${err.message}`, 'error');
    } finally {
        window.hideLoading();
        gs1Input.value = ''; 
    }
}

// ▼▼▼【削除】handleEditClick (テーブルの一覧表示がなくなったため不要) ▼▼▼
/*
function handleEditClick(event) { ... }
*/
// ▲▲▲【削除ここまで】▲▲▲


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
        // ▼▼▼【修正】 master.ProductMaster[key] ではなく master[key] を参照 ▼▼▼
        // (showModalが返すのはProductMasterViewであり、fetchProductMasterByBarcodeが返すのもProductMasterView (mappers.go / product/handler.go))
        // → populateEditForm が期待するのは ProductMaster (または互換性のあるView) なので master[key] で正しい
        const masterValue = master[key];
        // ▲▲▲【修正ここまで】▲▲▲

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

// handleSaveMaster (変更なし)
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

    // ▼▼▼【ここから修正】品目検索ボタンのリスナーを変更 ▼▼▼
    searchBtn.addEventListener('click', openProductSearchModalForMaster);
    // ▲▲▲【修正ここまで】▲▲▲
    
    // GS1スキャンフォームのリスナー
    if (gs1Form) {
        gs1Form.addEventListener('submit', handleMasterGs1Scan);
    } else {
        console.error("Master edit GS1 scan form not found.");
    }

    // ▼▼▼【ここから修正】フィルター入力欄のEnterキーリスナーを変更 ▼▼▼
    const handleKeyPress = (event) => {
        if (event.key === 'Enter') {
            openProductSearchModalForMaster();
        }
    };
    // ▲▲▲【修正ここまで】▲▲▲
    if (searchKanaNameInput) searchKanaNameInput.addEventListener('keypress', handleKeyPress);
    if (searchGenericNameInput) searchGenericNameInput.addEventListener('keypress', handleKeyPress);
    if (searchShelfNumberInput) searchShelfNumberInput.addEventListener('keypress', handleKeyPress);
    
    // ▼▼▼【削除】テーブルのクリックリスナーは不要 ▼▼▼
    // if (masterListTable) masterListTable.addEventListener('click', handleEditClick);
    // ▲▲▲【削除ここまで】▲▲▲
    
    // ▼▼▼【修正】テーブルを空にする（一覧表示はしないため）▼▼▼
    if (masterListTable) {
        masterListTable.innerHTML = '';
    }
    // ▲▲▲【修正ここまで】▲▲▲

    console.log("Master Edit View Initialized.");
}