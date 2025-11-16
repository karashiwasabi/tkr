// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\search_modal.js
import { hiraganaToKatakana } from './utils.js';
let activeCallback = null;
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

  if (modal && modal.dataset.copyOnly === 'true') {
    if (typeof activeCallback === 'function') {
      activeCallback(product, activeRowElement);
    }
    hideModal();
    return;
  }

  if (product.isAdopted) {
    if (typeof activeCallback === 'function') {
      activeCallback(product, activeRowElement);
    }
    hideModal();
  } else {
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
  const isAdoptFlow = modal.dataset.searchMode === 'inout' && modal.dataset.copyOnly !== 'true';

  products.forEach(p => {
    const productData = JSON.stringify(p);
    const spec = p.formattedPackageSpec || `${p.packageForm || ''} ${p.specification || ''}`;
    
    let buttonHtml = '';
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

async function performSearch() {
    const kanaName = modalKanaInput ? hiraganaToKatakana(modalKanaInput.value.trim()) : '';
    const genericName = modalGenericInput ? modalGenericInput.value.trim() : '';
    const shelfNumber = modalShelfInput ? modalShelfInput.value.trim() : '';
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

    window.showLoading('品目リストを検索中...');
    try {
        const fullUrl = `/api/products/search_filtered?${params.toString()}`;
        const res = await fetch(fullUrl);
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

        if (modal && modal.dataset.copyOnly === 'true') {
            if (typeof activeCallback === 'function') {
                 activeCallback(master, activeRowElement);
            }
            hideModal();
            return;
        }

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


export function initSearchModal() {
  modal = document.getElementById('tkr-search-modal-overlay');
  closeModalBtn = document.getElementById('closeSearchModalBtn');
  const searchResultsTable = document.getElementById('search-results-table');
  searchResultsBody = searchResultsTable ? searchResultsTable.querySelector('tbody') : null;
  
  searchBtn = document.getElementById('product-search-btn');
  modalGs1Form = document.getElementById('modal-search-gs1-form');
  modalGs1Input = document.getElementById('modal-search-gs1');
  modalUsageClassRadios = document.getElementById('modal-search-usage-class');
  
  modalKanaInput = document.getElementById('modal-search-kana');
  modalGenericInput = document.getElementById('modal-search-generic');
  modalShelfInput = document.getElementById('modal-search-shelf');

  if (!modal || !closeModalBtn || !searchResultsBody || !searchBtn) {
    console.error("品目検索モーダルの必須要素が見つかりません。");
    return;
  }
  
  closeModalBtn.addEventListener('click', hideModal);
  searchBtn.addEventListener('click', performSearch);
  if (modalGs1Form) {
    modalGs1Form.addEventListener('submit', handleGs1Search);
  }
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
  
  modal.dataset.searchMode = options.searchMode || '';
  modal.dataset.copyOnly = options.copyOnly || 'false';
  
  modal.classList.remove('hidden');
    
  if (options.initialResults) {
     renderSearchResults(options.initialResults);
  } else {
      searchResultsBody.innerHTML = '<tr><td colspan="6" class="center">検索条件を入力して検索してください。</td></tr>';
  }
}