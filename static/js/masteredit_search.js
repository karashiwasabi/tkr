// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\masteredit_search.js
import { showModal } from './search_modal.js';
import { hiraganaToKatakana, fetchProductMasterByBarcode } from './utils.js';
import { populateEditForm, showEditModal } from './masteredit_form.js';

let 
searchKanaNameInput, searchGenericNameInput, searchShelfNumberInput, searchBtn;
let gs1Form, gs1Input;
// ▼▼▼【ここに追加】新しいボタンの参照 ▼▼▼
let adoptJCSHMSBtn, createNewBtn;
// ▲▲▲【追加ここまで】▲▲▲

async function openProductSearchModalForMaster() {
    const apiUrl = '/api/products/search_filtered';
    const kanaInput = document.getElementById('master_search-kana');
    const genericInput = document.getElementById('master_search-generic');
    const 
shelfInput = document.getElementById('master_search-shelf');
    const selectedUsageRadio = document.querySelector('input[name="master_usage_class"]:checked');
    const kanaName = kanaInput ? hiraganaToKatakana(kanaInput.value.trim()) : '';
    const genericName = genericInput ? genericInput.value.trim() : '';
    const shelfNumber = shelfInput 
? shelfInput.value.trim() : '';
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
            throw new 
Error(`品目リストの取得に失敗しました: ${res.status}`);
        }
        products = await res.json();
    } catch (err) {
        window.hideLoading();
        window.showNotification(err.message, 
'error');
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

// ▼▼▼【ここに追加】JCSHMS採用モーダル用の関数 ▼▼▼
async function openAdoptJCSHMSModal() {
    const apiUrl = '/api/products/search_filtered';
    // JCSHMS採用時は、検索モーダル側の絞り込みをデフォルト（内）にする
    const usageClass = '内';
    
    const params = new URLSearchParams();
    params.append('kanaName', '');
    params.append('genericName', '');
    params.append('shelfNumber', '');
    params.append('dosageForm', usageClass);
    // searchMode: 'inout' を指定して JCSHMS も検索対象にする
    params.append('searchMode', 'inout');

    window.showLoading('JCSHMSを含む品目リストを検索中...');
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
            // モーダル側で採用処理 (adopt API) が実行された後、
            // populateEditForm を呼び出して編集モーダルを開く
            populateEditForm(selectedProduct); 
            showEditModal();
        }, 
        { 
            initialResults: products,
            searchMode: 'inout' // モーダル内部の検索にも適用
        }
    );
}

// ▼▼▼【ここに追加】その他新規作成用の関数 ▼▼▼
function handleCreateNew() {
    // 空のオブジェクト（ただし origin は設定）を渡してフォームを初期化
    populateEditForm({ 
        origin: 'PROVISIONAL',
        yjCode: '',
        productCode: '',
        isOrderStopped: 0,
        flagPoison: 0,
        flagDeleterious: 0,
        flagNarcotic: 0,
        flagPsychotropic: 0,
        flagStimulant: 0,
        flagStimulantRaw: 0
    });
    showEditModal();
}
// ▲▲▲【追加ここまで】▲▲▲


async function 
handleMasterGs1Scan(event) {
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
    
    // ▼▼▼【ここに追加】新しいボタンのDOM取得 ▼▼▼
    adoptJCSHMSBtn = document.getElementById('masterAdoptJCSHMSBtn');
    createNewBtn = document.getElementById('masterCreateNewBtn');
    // ▲▲▲【追加ここまで】▲▲▲


    if (searchBtn) {
 
       searchBtn.addEventListener('click', openProductSearchModalForMaster);
    }
    
    // ▼▼▼【ここに追加】新しいボタンのイベントリスナー ▼▼▼
    if (adoptJCSHMSBtn) {
        adoptJCSHMSBtn.addEventListener('click', openAdoptJCSHMSModal);
    }
    if (createNewBtn) {
        createNewBtn.addEventListener('click', handleCreateNew);
    }
    // ▲▲▲【追加ここまで】▲▲▲

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