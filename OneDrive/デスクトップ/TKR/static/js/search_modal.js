// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\search_modal.js
import { hiraganaToKatakana } from './utils.js';

let activeCallback = null;
let activeRowElement = null; // どの行から呼び出されたかを保持
let skipQueryLengthCheck = false; 

// モーダルのDOM要素
let modal, closeModalBtn, searchInput, searchBtn, searchResultsBody;

/**
 * モーダルを非表示にする
 */
function hideModal() {
  if (modal) {
        modal.classList.add('hidden');
        document.body.classList.remove('modal-open'); // TKRにはこのクラスはないかもしれないが念のため
    }
}

/**
 * 検索結果テーブルの「選択」ボタンが押されたときの処理
 */
function handleResultClick(event) {
  if (event.target && event.target.classList.contains('select-product-btn')) {
    const product = JSON.parse(event.target.dataset.product);
    if (typeof activeCallback === 'function') {
      activeCallback(product, activeRowElement);
    }
    hideModal();
  }
}

/**
 * 検索を実行する
 */
async function performSearch() {
  const query = hiraganaToKatakana(searchInput.value.trim());
  if (!skipQueryLengthCheck && query.length < 2) {
    window.showNotification('検索キーワードを2文字以上入力してください。', 'warning');
    searchResultsBody.innerHTML = '<tr><td colspan="6" class="center">2文字以上入力して検索してください。</td></tr>';
    return;
  }
  
  // 検索APIのURLをモーダルのdata属性から取得
  const searchApi = modal.dataset.searchApi || '/api/products/search_filtered'; // デフォルト
  searchResultsBody.innerHTML = '<tr><td colspan="6" class="center">検索中...</td></tr>';

  try {
    const separator = searchApi.includes('?') ? '&' : '?';
    const fullUrl = `${searchApi}${separator}q=${encodeURIComponent(query)}`;
    
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

/**
 * 検索結果をテーブルに描画する
 */
function renderSearchResults(products) {
  if (!products || products.length === 0) {
    searchResultsBody.innerHTML = '<tr><td colspan="6" class="center">該当する製品が見つかりません。</td></tr>';
    return;
  }

  let html = '';
  products.forEach(p => {
    // TKRでは isAdopted フラグはないため、クラス付与はしない
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

/**
 * モーダルを初期化する (app.jsから呼ばれる)
 */
export function initSearchModal() {
  modal = document.getElementById('tkr-search-modal-overlay');
  closeModalBtn = document.getElementById('closeSearchModalBtn');
  searchInput = document.getElementById('product-search-input');
  searchBtn = document.getElementById('product-search-btn');
  const searchResultsTable = document.getElementById('search-results-table');
  searchResultsBody = searchResultsTable ? searchResultsTable.querySelector('tbody') : null;

  if (!modal || !closeModalBtn || !searchInput || !searchBtn || !searchResultsBody) {
    console.error("品目検索モーダルの必須要素が見つかりません。");
    return;
  }

  closeModalBtn.addEventListener('click', hideModal);
  searchBtn.addEventListener('click', performSearch);
  
  searchInput.addEventListener('keypress', (e) => {
    if (e.key === 'Enter') {
      e.preventDefault();
      performSearch();
    }
  });

  searchResultsBody.addEventListener('click', handleResultClick);
  console.log("Search Modal Initialized.");
}

/**
 * モーダルを表示する (外部のJSから呼び出される)
 * @param {HTMLElement} rowElement - 呼び出し元の行要素 (オプション)
 * @param {Function} callback - 選択後に実行されるコールバック関数
 * @param {object} options - オプション (searchApi, initialResults, skipQueryLengthCheck)
 */
export function showModal(rowElement, callback, options = {}) {
  if (!modal) {
      console.error("Search modal is not initialized.");
      return;
  }
  
  document.body.classList.add('modal-open');
  activeRowElement = rowElement; // 呼び出し元を記憶
  activeCallback = callback; 
  
  skipQueryLengthCheck = options.skipQueryLengthCheck || false;
  searchInput.placeholder = skipQueryLengthCheck ? '絞り込み (Enterで検索)' : '製品名またはカナ名（2文字以上）';
  
  const searchApi = options.searchApi || '/api/products/search_filtered'; // デフォルトAPI
  modal.dataset.searchApi = searchApi;
  
  modal.classList.remove('hidden');
  searchInput.value = '';
  
  setTimeout(() => {
      searchInput.focus();
  }, 100); // モーダル表示アニメーション後にフォーカス

  if (options.initialResults) {
      renderSearchResults(options.initialResults);
  } else {
      searchResultsBody.innerHTML = '<tr><td colspan="6" class="center">検索キーワードを入力してください。</td></tr>';
  }
}