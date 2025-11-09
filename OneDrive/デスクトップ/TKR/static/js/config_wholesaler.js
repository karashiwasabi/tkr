import { refreshWholesalerMap, wholesalerMap } from './master_data.js';

let configAddWholesalerBtn, wholesalerCodeInput, wholesalerNameInput;
let wholesalerListTableBody;

function renderWholesalerList() {
    if (!wholesalerListTableBody) return;
    wholesalerListTableBody.innerHTML = '';
    if (wholesalerMap.size === 0) {
        wholesalerListTableBody.innerHTML = '<tr><td colspan="3">登録されている卸はありません。</td></tr>';
        return;
    }

    wholesalerMap.forEach((name, code) => {
        const tr = document.createElement('tr');
        tr.innerHTML = `
            <td class="col-config-code">${code}</td>
            <td class="left col-config-name">${name}</td>
            <td class="center col-config-action">
                <button class="btn delete-wholesaler-btn" data-code="${code}">削除</button>
            </td>
        `;
        wholesalerListTableBody.appendChild(tr);
    });
}

async function handleAddWholesaler() {
    const code = wholesalerCodeInput.value.trim();
    const name = wholesalerNameInput.value.trim();
    if (!code || !name) {
        window.showNotification('卸コードと卸名は必須です。', 'warning');
        return;
    }
    window.showLoading('卸を追加中...');
    try {
        const response = await fetch('/api/wholesalers/create', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({ code, name }),
        });
        const result = await response.json();
        if (!response.ok) {
            throw new Error(result.message || '卸の追加に失敗しました。');
        }
        window.showNotification(result.message, 'success');
        wholesalerCodeInput.value = '';
        wholesalerNameInput.value = '';
        await refreshWholesalerMap();
        renderWholesalerList();
    } catch (error) {
        window.showNotification(error.message, 'error');
    } finally {
        window.hideLoading();
    }
}

async function handleDeleteWholesaler(code) {
    if (!code) return;
    if (!confirm(`卸コード「${code}」を削除しますか？`)) return;

    window.showLoading('卸を削除中...');
    try {
        const response = await fetch(`/api/wholesalers/delete/${code}`, {
            method: 'DELETE',
        });
        const result = await response.json();
        if (!response.ok) {
            throw new Error(result.message || '卸の削除に失敗しました。');
        }
        window.showNotification(result.message, 'success');
        await refreshWholesalerMap();
        renderWholesalerList();
    } catch (error) {
        window.showNotification(error.message, 'error');
    } finally {
        window.hideLoading();
    }
}

export async function initWholesalerManagement() {
    configAddWholesalerBtn = document.getElementById('configAddWholesalerBtn');
    wholesalerCodeInput = document.getElementById('config-wholesaler-code');
    wholesalerNameInput = document.getElementById('config-wholesaler-name');
    wholesalerListTableBody = document.getElementById('wholesalerListTable')?.querySelector('tbody');

    if (configAddWholesalerBtn) {
        configAddWholesalerBtn.addEventListener('click', handleAddWholesaler);
    }
    
    if (wholesalerListTableBody) {
        wholesalerListTableBody.addEventListener('click', (e) => {
            if (e.target.classList.contains('delete-wholesaler-btn')) {
                handleDeleteWholesaler(e.target.dataset.code);
            }
        });
    }
    
    renderWholesalerList();
}