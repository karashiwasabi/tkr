// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\search_modal.js
import { hiraganaToKatakana } from './utils.js';

let activeCallback = 
null;
let activeRowElement = null;
let modal, closeModalBtn, searchResultsBody;
let searchBtn;
let modalGs1Input, modalUsageClassRadios, modalKanaInput, modalGenericInput, modalShelfInput;
let modalGs1Form;

function hideModal() {
  if (modal) {
    
modal.classList.add('hidden');
    document.body.classList.remove('modal-open');
  }
}

function handleResultClick(event) {
  if (!event.target.classList.contains('select-product-btn')) return;

  const product = JSON.parse(event.target.dataset.product);

  // コピーモード（参照のみ）の場合
  if (modal && modal.dataset.copyOnly === 'true') {
    if (typeof activeCallback 
=== 'function') {
      activeCallback(product, activeRowElement);
    }
    hideModal();
    return;
  }

  // 採用済みの場合
  if (product.isAdopted) {
    // allowAdoptedフラグが立っている（発注画面など）か、通常の選択フローならそのままコールバック
 
   if (typeof activeCallback === 'function') {
      activeCallback(product, activeRowElement);
    }
    hideModal();
  } else {
    // 未採用の場合、マスタ採用フローへ
 
   if (!confirm(`「${product.productName}」をマスターに新規採用します。よろしいですか？`)) {
      return;
    }
    window.showLoading('マスターに採用中...');
    fetch('/api/master/adopt', {
        method: 'POST',
  
      headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ gs1Code: product.gs1Code, productCode: product.productCode })
    })
 
   .then(res => {
     if (!res.ok) {
            return res.text().then(text => { throw new Error(text || '採用処理に失敗しました') });
        }
        return res.json();
     })
   
 .then(adoptedMaster => {
        window.hideLoading();
        window.showNotification(`「${adoptedMaster.productName}」をマスターに採用しました。`, 'success');
        
 
       if (typeof activeCallback === 'function') {
            activeCallback(adoptedMaster, activeRowElement);
    
    }
        hideModal();
    })
    .catch(err => {
       
 window.hideLoading();
        window.showNotification(`エラー: ${err.message}`, 'error');
    });
  }
}

function renderSearchResults(products) {
  if (!products || products.length === 0) {
   
 searchResultsBody.innerHTML = '<tr><td colspan="6" class="center">該当する製品が見つかりません。</td></tr>';
    return;
  }
  let html = '';
  
  // ▼▼▼【修正】採用済みボタンを無効化する条件に allowAdopted のチェックを追加 ▼▼▼
  // 「inoutモード」かつ「コピーモードでない」かつ「採用済み許可(allowAdopted)がない」場合のみ無効化
  const isAdoptFlow = modal.dataset.searchMode === 'inout' && 

                      modal.dataset.copyOnly !== 'true' &&
    
                  modal.dataset.allowAdopted !== 'true';
  // ▲▲▲【修正ここまで】▲▲▲

  products.forEach(p => {
   
 const productData = JSON.stringify(p);
    const spec = p.formattedPackageSpec || `${p.packageForm || ''} ${p.specification || ''}`;
    
    let buttonHtml = 
'';
    if (p.isAdopted && isAdoptFlow) {
        buttonHtml = `<button type="button" class="btn" disabled>採用済</button>`;
    } else {
  
      buttonHtml = `<button type="button" class="select-product-btn btn" data-product='${productData.replace(/'/g, "&apos;")}'>選択</button>`;
    }

    html += `
      
<tr class="${p.isAdopted ? 'adopted-item' : ''}">
        <td class="left">${p.productName || ''}</td>
        <td class="left">${p.makerName || ''}</td>
  
       <td class="left">${spec}</td>
        <td>${p.yjCode || ''}</td>
        <td>${p.productCode || ''}</td>
 
        <td>${buttonHtml}</td>
      </tr>
    `;
  });
  searchResultsBody.innerHTML = html;
}

// ▼▼▼【修正】performSearch をモーダル内部のフォームから値を取得するように修正 ▼▼▼
async function performSearch() {
  
    const kanaName = modalKanaInput ? hiraganaToKatakana(modalKanaInput.value.trim()) : '';
    const genericName = modalGenericInput ? modalGenericInput.value.trim() : '';
    const shelfNumber = modalShelfInput ? modalShelfInput.value.trim() : '';
    // modalUsageClassRadios (コンテナ) から選択されたラジオボタンの値を取得
    const selectedUsageRadio = modalUsageClassRadios ? document.querySelector('input[name="modal_usage_class"]:checked') : null;
    const usageClass = selectedUsageRadio ? selectedUsageRadio.value : '';
    
    const params = new URLSearchParams();
    params.append('kanaName', kanaName);
    params.append('genericName', genericName);
    params.append('shelfNumber', shelfNumber);
    params.append('dosageForm', usageClass);

    if (modal.dataset.searchMode) {
 
       params.append('searchMode', modal.dataset.searchMode);
    }
    
    // ▼▼▼【追加】検索条件チェック (内外注区分が未選択の場合は警告) ▼▼▼
    if (!usageClass) {
        window.showNotification('内外注区分を選択してください。', 'warning');
        return;
    }
    // ▲▲▲【追加ここまで】▼▼▼


    window.showLoading('品目リストを検索中...');
    try {
        const fullUrl = `/api/products/search_filtered?${params.toString()}`;
        const res 
= await fetch(fullUrl);
        if (!res.ok) {
             throw new Error(`品目リストの取得に失敗しました: ${res.status}`);
        }
       
 const products = await res.json();
        renderSearchResults(products);
    } catch (err) {
        searchResultsBody.innerHTML = '<tr><td colspan="6" class="center" style="color:red;">検索エラー: ' + err.message + '</td></tr>';
window.showNotification(err.message, 'error');
    } finally {
         window.hideLoading();
    }
}
// ▲▲▲【修正ここまで】▲▲▲

// ▼▼▼【修正】handleGs1Search を修正 (モーダルを閉じる/コールバック呼び出し) ▼▼▼
async function handleGs1Search(event) {
    event.preventDefault();
    if (!modalGs1Input) return;
    const barcode = modalGs1Input.value.trim();
    if (!barcode) return;
    
    window.showLoading('マスターを検索中...');
    try {
        const res = await fetch(`/api/product/by_barcode/${barcode}`);
        if (!res.ok) {
            const errText = await res.text();
            throw new Error(errText || '製品の検索に失敗しました。');
        }
        const master = await res.json();
        
        // 常にコールバックを呼び出し、モーダルを閉じる
        if (typeof activeCallback === 'function') {
            activeCallback(master, activeRowElement);
        }
        hideModal();

    } catch (err) {
        window.showNotification(`エラー: ${err.message}`, 'error');
    } finally {
         window.hideLoading();
        modalGs1Input.value = '';
    }
}
// ▲▲▲【修正ここまで】▼▼▼


export function initSearchModal() {
  modal = document.getElementById('tkr-search-modal-overlay');
  closeModalBtn = document.getElementById('closeSearchModalBtn');
  const searchResultsTable = document.getElementById('search-results-table');
  searchResultsBody = searchResultsTable ? searchResultsTable.querySelector('tbody') : null;
  
  searchBtn = document.getElementById('product-search-btn');
modalGs1Form = document.getElementById('modal-search-gs1-form');
  modalGs1Input = document.getElementById('modal-search-gs1');
  
  // ▼▼▼【修正】モーダル内部のフォーム要素を取得 ▼▼▼
  modalUsageClassRadios = document.getElementById('modal-search-usage-class'); // ラジオボタンのコンテナ
  modalKanaInput = document.getElementById('modal-search-kana');
  modalGenericInput = document.getElementById('modal-search-generic');
  modalShelfInput = document.getElementById('modal-search-shelf');
  // ▲▲▲【修正ここまで】▼▼▼


  if (!modal || !closeModalBtn || !searchResultsBody || !searchBtn) {
   
 console.error("品目検索モーダルの必須要素が見つかりません。");
    return;
  }
  
  closeModalBtn.addEventListener('click', hideModal);
  searchBtn.addEventListener('click', performSearch);
  
  // ▼▼▼【修正】GS1フォームのイベントリスナーを登録 ▼▼▼
  if (modalGs1Form) {
    modalGs1Form.addEventListener('submit', handleGs1Search);
  }
  // ▲▲▲【修正ここまで】▼▼▼
  
  searchResultsBody.addEventListener('click', handleResultClick);
  
  // ▼▼▼【追加】Enterキーで検索を実行するリスナーをフィルター入力欄に追加 ▼▼▼
  if (modalKanaInput) {
    modalKanaInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') performSearch();
    });
  }
  if (modalGenericInput) {
    modalGenericInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') performSearch();
    });
  }
  if (modalShelfInput) {
    modalShelfInput.addEventListener('keypress', (e) => {
        if (e.key === 'Enter') performSearch();
    });
  }
  // ▲▲▲【追加ここまで】▼▼▼
}

export function showModal(rowElement, callback, options = 
{}) {
  if (!modal) {
      console.error("Search modal is not initialized.");
      return;
  }
  document.body.classList.add('modal-open');
  activeRowElement = rowElement;
  activeCallback = callback; 
  
  modal.dataset.searchMode = options.searchMode 
|| '';
  modal.dataset.copyOnly = options.copyOnly || 'false';
  // ▼▼▼【追加】採用済み選択許可フラグを設定 ▼▼▼
  modal.dataset.allowAdopted = options.allowAdopted ? 'true' : 'false';
  // ▲▲▲【追加ここまで】▼▼▼
  
  modal.classList.remove('hidden');
    
  if (options.initialResults) {
   
  renderSearchResults(options.initialResults);
  } else {
      searchResultsBody.innerHTML = '<tr><td colspan="6" class="center">検索条件を入力して検索してください。</td></tr>';
  }
}