// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\search_modal.js
import { hiraganaToKatakana, parseBarcode, fetchProductMasterByBarcode } from './utils.js';
let activeCallback = null;
let activeRowElement = null;

let modal, closeModalBtn, searchResultsBody;
// 検索UI関連の変数は削除
// let modalGs1Input, modalUsageClassRadios, modalKanaInput, modalGenericInput, modalShelfInput;
// let modalGs1Form;

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

// performSearch 関数を削除
// handleGs1Search 関数を削除

function renderSearchResults(products) {
  if (!products || products.length === 0) {
    searchResultsBody.innerHTML = '<tr><td colspan="6" class="center">該当する製品が見つかりません。</td></tr>';
    return;
  }
  let html = '';
  products.forEach(p => {
    const productData = JSON.stringify(p);
    html += `
      <tr class="${p.isAdopted ? 'adopted-item' : ''}">
        <td class="left">${p.productName || ''}</td>
        <td class="left">${p.makerName || ''}</td>
        <td class="left">${p.formattedPackageSpec}</td>
        <td>${p.yjCode || ''}</td>
        <td>${p.productCode || ''}</td>
        <td><button typebutton" class="select-product-btn btn" data-product='${productData.replace(/'/g, "&apos;")}'>選択</button></td>
      </tr>
  
 
    `;
  });
  searchResultsBody.innerHTML = html;
}
export function initSearchModal() {
  modal = document.getElementById('tkr-search-modal-overlay');
  closeModalBtn = document.getElementById('closeSearchModalBtn');
  // searchBtn = document.getElementById('product-search-btn'); // 削除
  const searchResultsTable = document.getElementById('search-results-table');
  searchResultsBody = searchResultsTable ? searchResultsTable.querySelector('tbody') : null;
  // modalGs1Form = document.getElementById('modal-search-gs1-form'); // 削除
  // modalGs1Input = document.getElementById('modal-search-gs1'); // 削除
  // ... (他の検索UI要素の取得も削除)

  if (!modal || !closeModalBtn || !searchResultsBody) {
    console.error("品目検索モーダルの必須要素が見つかりません。");
    return;
  }
  
  closeModalBtn.addEventListener('click', hideModal);
  // 検索ボタン、GS1フォーム、キープレスイベントリスナーを削除
  searchResultsBody.addEventListener('click', handleResultClick);
}

export function showModal(rowElement, callback, options = {}) {
  if (!modal) {
      console.error("Search modal is not initialized.");
      return;
  }
  document.body.classList.add('modal-open');
  activeRowElement = rowElement;
  activeCallback = callback; 
  
  modal.classList.remove('hidden');
    
  if (options.initialResults) {
      renderSearchResults(options.initialResults);
  } else {
      searchResultsBody.innerHTML = '<tr><td colspan="6" class="center">メイン画面で検索を実行してください。</td></tr>';
  }
}