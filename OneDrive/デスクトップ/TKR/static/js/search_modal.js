// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\search_modal.js
import { hiraganaToKatakana } from './utils.js';

let activeCallback = null;
let activeRowElement = null;

let modal, closeModalBtn, searchBtn, searchResultsBody;
let modalGs1Input, modalUsageClassRadios, modalKanaInput, modalGenericInput, modalShelfInput;
let modalGs1Form;

function hideModal() {
  if (modal) {
        modal.classList.add('hidden');
        document.body.classList.remove('modal-open');
    }
}

function handleResultClick(event) {
  if (event.target && event.target.classList.contains('select-product-btn')) {
    const product = JSON.parse(event.target.dataset.product);
    if (typeof activeCallback === 'function') {
      activeCallback(product, activeRowElement);
    }
    hideModal();
  }
}

async function performSearch() {
  // ▼▼▼【修正】一般名(modalGenericInput)も取得する ▼▼▼
  const kanaName = hiraganaToKatakana(modalKanaInput.value.trim());
  const genericName = modalGenericInput.value.trim(); // ★追加
  const shelfNumber = modalShelfInput.value.trim();
  const selectedUsageRadio = document.querySelector('input[name="modal_usage_class"]:checked');
  const usageClass = selectedUsageRadio ? selectedUsageRadio.value : '';
  // ▲▲▲【修正ここまで】▲▲▲

  const searchApi = modal.dataset.searchApi || '/api/products/search_filtered';
  
  const params = new URLSearchParams();
  params.append('kanaName', kanaName);
  params.append('genericName', genericName); // ★追加
  params.append('shelfNumber', shelfNumber);
  params.append('dosageForm', usageClass);

  searchResultsBody.innerHTML = '<tr><td colspan="6" class="center">検索中...</td></tr>';

  try {
    const fullUrl = `${searchApi}?${params.toString()}`;
    
    const res = await fetch(fullUrl);
    if (!res.ok) {
        throw new Error(`サーバーエラー: ${res.status}`);
    }
    const products = await res.json();
    renderSearchResults(products);
  } catch (err) {
    searchResultsBody.innerHTML = `<tr><td colspan="6" class="center" style="color:red;">${err.message}</td></tr>`;
  }
}

// ... (handleGs1Search, renderSearchResults 関数は変更なし) ...
async function handleGs1Search(event) {
    event.preventDefault();
    const barcode = modalGs1Input.value.trim();
    if (!barcode) return;
    let gs1Code = barcode;
    if (barcode.startsWith('01') && barcode.length >= 16) {
        gs1Code = barcode.substring(2, 16);
    }
    searchResultsBody.innerHTML = '<tr><td colspan="6" class="center">GS1コードで検索中...</td></tr>';
    window.showLoading('GS1コードで検索中...');
    try {
        const res = await fetch(`/api/product/by_gs1?gs1_code=${gs1Code}`);
        if (!res.ok) {
             if (res.status === 404) {
                throw new Error('このGS1コードはマスターに登録されていません。');
             }
            throw new Error(`サーバーエラー: ${res.status}`);
        }
        const product = await res.json();
        renderSearchResults([product]);
    } catch (err) {
        searchResultsBody.innerHTML = `<tr><td colspan="6" class="center" style="color:red;">${err.message}</td></tr>`;
    } finally {
        window.hideLoading();
        modalGs1Input.value = '';
    }
}
function renderSearchResults(products) {
  if (!products || products.length === 0) {
    searchResultsBody.innerHTML = '<tr><td colspan="6" class="center">該当する製品が見つかりません。</td></tr>';
    return;
  }
  let html = '';
  products.forEach(p => {
    const productData = JSON.stringify(p);
    html += `
      <tr>
        <td class="left">${p.productName || ''}</td>
        <td class="left">${p.makerName || ''}</td>
        <td class="left">${p.formattedPackageSpec}</td>
        <td>${p.yjCode || ''}</td>
        <td>${p.productCode || ''}</td>
        <td><button type="button" class="select-product-btn btn" data-product='${productData.replace(/'/g, "&apos;")}'>選択</button></td>
      </tr>
    `;
  });
  searchResultsBody.innerHTML = html;
}


export function initSearchModal() {
  modal = document.getElementById('tkr-search-modal-overlay');
  closeModalBtn = document.getElementById('closeSearchModalBtn');
  searchBtn = document.getElementById('product-search-btn');
  const searchResultsTable = document.getElementById('search-results-table');
  searchResultsBody = searchResultsTable ? searchResultsTable.querySelector('tbody') : null;

  modalGs1Form = document.getElementById('modal-search-gs1-form');
  modalGs1Input = document.getElementById('modal-search-gs1');
  modalUsageClassRadios = document.getElementById('modal-search-usage-class');
  modalKanaInput = document.getElementById('modal-search-kana');
  modalGenericInput = document.getElementById('modal-search-generic'); // 取得
  modalShelfInput = document.getElementById('modal-search-shelf');

  if (!modal || !closeModalBtn || !searchBtn || !searchResultsBody || !modalGs1Form || !modalGenericInput) { // チェックに追加
    console.error("品目検索モーダルの必須要素が見つかりません。");
    return;
  }

  closeModalBtn.addEventListener('click', hideModal);
  searchBtn.addEventListener('click', performSearch);
  modalGs1Form.addEventListener('submit', handleGs1Search);

  const handleKeyPress = (e) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      performSearch();
    }
  };
  modalKanaInput.addEventListener('keypress', handleKeyPress);
  modalGenericInput.addEventListener('keypress', handleKeyPress); // リスナー追加
  modalShelfInput.addEventListener('keypress', handleKeyPress);

  searchResultsBody.addEventListener('click', handleResultClick);
  console.log("Search Modal Initialized.");
}

// ... (showModal 関数は変更なし) ...
export function showModal(rowElement, callback, options = {}) {
  if (!modal) {
      console.error("Search modal is not initialized.");
      return;
  }
  document.body.classList.add('modal-open');
  activeRowElement = rowElement;
  activeCallback = callback; 
  const searchApi = options.searchApi || '/api/products/search_filtered';
  modal.dataset.searchApi = searchApi;
  modal.classList.remove('hidden');
  modalGs1Input.value = '';
  modalKanaInput.value = '';
  modalGenericInput.value = '';
  modalShelfInput.value = '';
  document.querySelector('input[name="modal_usage_class"][value=""]').checked = true;
  setTimeout(() => {
      modalGs1Input.focus();
  }, 100);
  if (options.initialResults) {
      renderSearchResults(options.initialResults);
  } else {
      searchResultsBody.innerHTML = '<tr><td colspan="6" class="center">検索ボタンを押してください。</td></tr>';
  }
}