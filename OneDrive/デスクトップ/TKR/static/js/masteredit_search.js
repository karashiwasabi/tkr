import { showModal } from './search_modal.js';
import { hiraganaToKatakana, fetchProductMasterByBarcode } from './utils.js';
import { populateEditForm, showEditModal } from './masteredit_form.js';

let searchKanaNameInput, searchGenericNameInput, searchShelfNumberInput, searchBtn;
let gs1Form, gs1Input;

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

export function initSearchForm() {
    searchKanaNameInput = document.getElementById('master_search-kana');
    searchGenericNameInput = document.getElementById('master_search-generic');
    searchShelfNumberInput = document.getElementById('master_search-shelf');
    
    gs1Form = document.getElementById('master-barcode-form');
    gs1Input = document.getElementById('master-search-gs1-barcode');

    if (searchGenericNameInput) searchGenericNameInput.disabled = false;

    searchBtn = document.getElementById('masterSearchBtn');

    if (searchBtn) {
        searchBtn.addEventListener('click', openProductSearchModalForMaster);
    }
    
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
}