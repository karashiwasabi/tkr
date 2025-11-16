// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\inout_header.js
import { getLocalDateString } from './utils.js';
import { clientMap, refreshClientMap } from './master_data.js';
import { showInputModal } from './common_modal.js';

const NEW_ENTRY_VALUE = '--new--';
let clientSelect, receiptSelect, saveBtn, deleteBtn, headerDateInput, headerTypeSelect;
let newClientName = null;
let currentLoadedReceipt = null;

async function setupClientDropdown(selectEl) {
	if (!selectEl) return;
	const preservedOptions = Array.from(selectEl.querySelectorAll('option[value="--new--"], option[value=""]'));
	selectEl.innerHTML = '';
	preservedOptions.forEach(opt => selectEl.appendChild(opt));

    // ▼▼▼【修正】TKR の clientMap (起動時にロード済み) を使用 ▼▼▼
    try {
		if (clientMap) {
			clientMap.forEach((name, code) => {
				const opt = document.createElement('option');
				opt.value = code;
				opt.textContent = `${code}:${name}`;
				selectEl.appendChild(opt);
			});
		}
	} catch (err) {
		console.error("得意先リストの構築に失敗:", err);
	}
    // ▲▲▲【修正ここまで】▲▲▲
}

async function initializeClientDropdown() {
	clientSelect.innerHTML = `<option value="">選択してください</option>`;
	
	const newOption = document.createElement('option');
	newOption.value = NEW_ENTRY_VALUE;
	newOption.textContent = '--- 新規作成 ---';
	clientSelect.appendChild(newOption);

    // ▼▼▼【修正】TKR の clientMap (起動時にロード済み) を使用 ▼▼▼
    await setupClientDropdown(clientSelect);
    // ▲▲▲【修正ここまで】▲▲▲
}

export function resetHeader() {
	if (!clientSelect || !headerDateInput) return;
	headerDateInput.value = getLocalDateString();
	initializeClientDropdown();
	receiptSelect.innerHTML = `
		<option value="">日付を選択してください</option>
		<option value="${NEW_ENTRY_VALUE}">--- 新規作成 ---</option>
	`;
	headerTypeSelect.value = "11"; // 入庫
	newClientName = null;
	currentLoadedReceipt = null;
	deleteBtn.disabled = true;
	clientSelect.value = ''; // デフォルトを「選択してください」に設定
	headerDateInput.dispatchEvent(new Event('change'));
}

export async function initHeader(getDetailsData, clearDetailsTable, populateDetailsTable) {
	clientSelect = document.getElementById('in-out-client');
	receiptSelect = document.getElementById('in-out-receipt');
	saveBtn = document.getElementById('saveBtn');
	deleteBtn = document.getElementById('deleteBtn');
	headerDateInput = document.getElementById('in-out-date');
	headerTypeSelect = document.getElementById('in-out-type');

	if (!clientSelect || !receiptSelect || !saveBtn || !deleteBtn) return;

	// --- Refactored Receipt Fetching ---
    async function fetchReceipts() {
        const date = headerDateInput.value.replace(/-/g, '');
        const clientCode = clientSelect.value;

        if (!date) return;

        // Don't fetch if the client is a pending new client
        if (clientCode.startsWith('new:')) return;

        const params = new URLSearchParams({ date });
        if (clientCode) {
            params.append('client', clientCode);
        }

        try {
            const res = await fetch(`/api/receipts/by_date?${params.toString()}`);
            if (!res.ok) throw new Error('伝票の取得に失敗');
            const receiptNumbers = await res.json();
            
            const currentReceipt = receiptSelect.value;
            receiptSelect.innerHTML = `
				<option value="">選択してください</option>
				<option value="${NEW_ENTRY_VALUE}">--- 新規作成 ---</option>
			`;
			if (receiptNumbers && receiptNumbers.length > 0) {
				receiptNumbers.forEach(num => {
					const opt = document.createElement('option');
					opt.value = num;
					opt.textContent = num;
                    if (num === currentReceipt) {
                        opt.selected = true;
                    }
					receiptSelect.appendChild(opt);
				});
			}
        } catch (err) { 
            console.error(err);
            receiptSelect.innerHTML = `
				<option value="">日付を選択してください</option>
				<option value="${NEW_ENTRY_VALUE}">--- 新規作成 ---</n>
			`;
        }
    }

    headerDateInput.addEventListener('change', fetchReceipts);

	clientSelect.addEventListener('change', async (e) => {
        const selectedValue = e.target.value;

        if (selectedValue === NEW_ENTRY_VALUE) {
            const name = await showInputModal('新しい得意先名を入力してください');
            if (name && name.trim()) {
                newClientName = name.trim();
                const opt = document.createElement('option');
                opt.value = `new:${newClientName}`;
                opt.textContent = `[新規] ${newClientName}`;
                opt.selected = true;
                clientSelect.appendChild(opt);
                // 新規クライアントの場合、伝票リストはクリアする
                receiptSelect.innerHTML = `
				    <option value="">選択してください</option>
				    <option value="${NEW_ENTRY_VALUE}">--- 新規作成 ---</option>
			    `;
            } else {
                clientSelect.value = '';
                fetchReceipts(); // キャンセルされたら、得意先未選択の状態で再取得
            }
        } else {
            if (!selectedValue.startsWith('new:')) {
                newClientName = null;
            }
            fetchReceipts(); // 既存の得意先が選択されたら、伝票を再取得
        }
	});

	receiptSelect.addEventListener('change', async () => {
		const selectedValue = receiptSelect.value;
		deleteBtn.disabled = (selectedValue === NEW_ENTRY_VALUE || selectedValue === "");

		if (selectedValue === NEW_ENTRY_VALUE || selectedValue === "") {
			clearDetailsTable();
			currentLoadedReceipt = null;
		} else {
			window.showLoading();
			try {
                // ▼▼▼【修正】TKRのAPIエンドポイント /api/transaction を使用 ▼▼▼
				const res = await fetch(`/api/transaction/${selectedValue}`);
				if (!res.ok) throw new Error('明細の読込に失敗');
				const records = await res.json();
				if (records && records.length > 0) {
					currentLoadedReceipt = selectedValue;
					clientSelect.value = records[0].clientCode;
					headerTypeSelect.value = records[0].flag; // 11 or 12
					newClientName = null;
				}
				populateDetailsTable(records);
			} catch (err) {
				console.error(err);
				window.showNotification(err.message, 'error');
			} finally {
				window.hideLoading();
			}
		}
	});

	saveBtn.addEventListener('click', async () => {
		let clientCode = clientSelect.value;
		let clientNameToSave = '';
		let isNewClient = false;
		if (newClientName && clientCode.startsWith('new:')) {
			clientNameToSave = newClientName;
			isNewClient = true;
			clientCode = '';
		} else {
			if (!clientCode || clientCode === NEW_ENTRY_VALUE) {
				window.showNotification('得意先を選択または新規作成してください。', 'error');
				return;
			}
		}
		const records = getDetailsData();
		if (records.length === 0) {
			window.showNotification('保存する明細データがありません。', 'error');
			return;
		}
		const payload = {
			isNewClient, 
            clientCode, 
            clientName: clientNameToSave,
			transactionDate: headerDateInput.value.replace(/-/g, ''),
			transactionTypeFlag: parseInt(headerTypeSelect.value, 10), // 11 or 12
			records: records,
			originalReceiptNumber: currentLoadedReceipt
		};
        console.log('Saving payload:', payload); // デバッグ用ログ
		window.showLoading();
		try {
            // ▼▼▼【修正】TKRのAPIエンドポイント /api/inout/save を使用 ▼▼▼
			const res = await fetch('/api/inout/save', {
				method: 'POST',
				headers: { 'Content-Type': 'application/json' },
				body: JSON.stringify(payload),
			});
			const resData = await res.json();
			if (!res.ok) {
				throw new Error(resData.message || `保存に失敗しました (HTTP ${res.status})`);
			}

			if (resData.newClient) {
                // ▼▼▼【修正】TKR の master_data.js の関数を使用 ▼▼▼
				await refreshClientMap();
                await setupClientDropdown(clientSelect); // ドロップダウンを再構築
			}
			// ▲▲▲【修正ここまで】▲▲▲

			window.showNotification(`データを保存しました。\n伝票番号: ${resData.receiptNumber}`, 'success');
			resetHeader();
			clearDetailsTable();
		} catch (err) {
			console.error(err);
			window.showNotification(err.message, 'error');
		} finally {
			window.hideLoading();
		}
	});

	deleteBtn.addEventListener('click', async () => {
		const receiptNumber = receiptSelect.value;
		if (!receiptNumber || receiptNumber === NEW_ENTRY_VALUE) {
			window.showNotification("削除対象の伝票が選択されていません。", 'error');
			return;
		}
		if (!confirm(`伝票番号 [${receiptNumber}] を完全に削除します。よろしいですか？`)) {
			return;
		}
		window.showLoading();
		try {
            // ▼▼▼【修正】TKRのAPIエンドポイント /api/transaction/delete を使用 ▼▼▼
			const res = await fetch(`/api/transaction/delete/${receiptNumber}`, { method: 'DELETE' });
			const resData = await res.json().catch(() => null);
			if (!res.ok) {
				throw new Error(resData?.message || '削除に失敗しました。');
			}
			window.showNotification(resData.message || `伝票 [${receiptNumber}] を削除しました。`, 'success');
			resetHeader();
			clearDetailsTable();
		} catch(err) {
			console.error(err);
			window.showNotification(err.message, 'error');
		} finally {
			window.hideLoading();
		}
	});

    // 初期化時に日付を設定し、得意先ドロップダウンを構築
    resetHeader();
}