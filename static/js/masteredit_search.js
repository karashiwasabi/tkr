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

// ▼▼▼【修正】openProductSearchModalForMaster を修正 (モーダル内検索に移行) ▼▼▼
async function openProductSearchModalForMaster() {
    // 検索フォームから値を取得するロジックは削除し、モーダルに任せる。
    window.showLoading('品目検索モーダルを開いています...');

    showModal(
        null,
      (selectedProduct) => { 
            populateEditForm(selectedProduct); 
            showEditModal();
        }, 
        { 
          // master_edit_viewは採用済み品目のみを検索するが、採用済みでない品目（JCSHMS）が選択された場合は採用プロセスへ進めるため、allowAdoptedは設定しない（デフォルトの採用フローに乗せる）
          searchMode: 'default', 
        }
    );
    
    window.hideLoading();
}
// ▲▲▲【修正ここまで】▲▲▲

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
    } catch (err) 
{
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
            searchMode: 
'inout' // モーダル内部の検索にも適用
        }
    );
}

function handleCreateNew() {
    // 空のオブジェクト（ただし origin は設定）を渡してフォームを初期化
   
 populateEditForm({ 
        origin: 'PROVISIONAL',
        yjCode: '',
        productCode: 
'',
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


async function 
handleMasterGs1Scan(event) {
    event.preventDefault();
    if (!gs1Input) return;
    
    const barcode = gs1Input.value.trim();
    if (!barcode) 
{
        window.showNotification('バーコードが入力されていません。', 'warning');
        return;
    }

 
   window.showLoading('マスターを検索中...');
    try {
        const master = await 
fetchProductMasterByBarcode(barcode);
        
        populateEditForm(master);
        showEditModal();
    } catch (err) {
    
    window.showNotification(`エラー: ${err.message}`, 'error');
    } finally {
        window.hideLoading();
        gs1Input.value 
= '';
    }
}

export function initSearchForm() {
   
    // ▼▼▼【削除】モーダル内の検索フォーム要素の取得を削除 ▼▼▼
    // searchKanaNameInput = document.getElementById('master_search-kana');
    // searchGenericNameInput = document.getElementById('master_search-generic');
    // searchShelfNumberInput = document.getElementById('master_search-shelf');
    // ▲▲▲【削除ここまで】▲▲▲

    gs1Form = document.getElementById('master-barcode-form');
    gs1Input = document.getElementById('master-search-gs1-barcode');

    if (searchGenericNameInput) searchGenericNameInput.disabled = false;
searchBtn = document.getElementById('masterSearchBtn');
    
    adoptJCSHMSBtn = document.getElementById('masterAdoptJCSHMSBtn');
    createNewBtn = document.getElementById('masterCreateNewBtn');


    if (searchBtn) {
 
      
 searchBtn.addEventListener('click', openProductSearchModalForMaster);
    }
    
    if (adoptJCSHMSBtn) {
        adoptJCSHMSBtn.addEventListener('click', openAdoptJCSHMSModal);
}
    if (createNewBtn) {
        createNewBtn.addEventListener('click', handleCreateNew);
    }

    if (gs1Form) {
  
      
gs1Form.addEventListener('submit', handleMasterGs1Scan);
    } else {
        console.error("Master edit GS1 scan form not found.");
    }
}