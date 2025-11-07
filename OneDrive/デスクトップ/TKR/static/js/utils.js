// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\utils.js
// ▼▼▼【ここから追加】common_table.js から関数をインポート ▼▼▼
import { renderTransactionTableHTML, renderEmptyTableHTML } from './common_table.js';
// ▲▲▲【追加ここまで】▲▲▲

export function hiraganaToKatakana(str) {
	if (!str) return '';
	return str.replace(/[\u3041-\u3096]/g, function (match) {
		const charCode = match.charCodeAt(0) + 0x60;
		return String.fromCharCode(charCode);
	});
}

// ▼▼▼【ここから修正】引数(date)を尊重するように修正 ▼▼▼
export function getLocalDateString(date = null) {
	const today = date instanceof Date ? date : new Date();
	// 引数がなければ new Date() を使う
	const yyyy = today.getFullYear();
	const mm = String(today.getMonth() + 1).padStart(2, '0');
	const dd = String(today.getDate()).padStart(2, '0');
	return `${yyyy}-${mm}-${dd}`;
}
// ▲▲▲【修正ここまで】▲▲▲

export function parseBarcode(code) {
	const length = code.length;
	if (length === 0) {
		throw new Error('バーコードが空です');
	}
	if (length >= 15) {
		if (code.startsWith('01')) {
			return parseAIString(code);
		}
		throw new Error('15桁以上ですが、AI(01)で始まっていません');
	}
	if (length === 14) {
		return { gtin14: code, expiryDate: '', lotNumber: '' };
	}
	if (length === 13) {
		return { gtin14: '0' + code, expiryDate: '', lotNumber: '' };
	}
	if (length < 13) {
		return { gtin14: code.padStart(14, '0'), expiryDate: '', lotNumber: '' };
	}
	throw new Error('不明なバーコード形式です');
}
function parseAIString(code) {
	let rest = code;
	const data = { gtin14: '', expiryDate: '', lotNumber: '' };
	if (rest.startsWith('01')) {
		if (rest.length < 16) throw new Error('AI(01)のデータが不足しています');
		data.gtin14 = rest.substring(2, 16);
		rest = rest.substring(16);
	} else {
		throw new Error('AI(01)が見つかりません');
	}
	if (rest.startsWith('17')) {
		if (rest.length < 8) return data;
		const yy_mm_dd = rest.substring(2, 8);
		if (yy_mm_dd.length === 6) {
			const yy = yy_mm_dd.substring(0, 2);
			const mm = yy_mm_dd.substring(2, 4);
			data.expiryDate = `20${yy}${mm}`;
		}
		rest = rest.substring(8);
	}
	if (rest.startsWith('10')) {
		if (rest.length <= 2) return data;
		rest = rest.substring(2);
		let endIndex = rest.length;
		const maxLength = 20;
		const next17 = rest.indexOf('17');
		if (next17 > -1 && rest.length >= next17 + 8) {
			endIndex = next17;
		}
		const next01 = rest.indexOf('01');
		if (next01 > -1 && rest.length >= next01 + 16) {
			if (next01 < endIndex) {
				endIndex = next01;
			}
		}
		if (endIndex > maxLength) {
			endIndex = maxLength;
		}
		data.lotNumber = rest.substring(0, endIndex);
	}
	return data;
}

export async function fetchProductMasterByBarcode(barcode) {
	if (!barcode) {
		throw new Error('バーコードが空です');
	}
	const res = await fetch(`/api/product/by_barcode/${barcode}`);
	if (!res.ok) {
		if (res.status === 404) {
			throw new Error('このバーコードはマスターに登録されていません。');
		}
		const errText = await res.text().catch(() => `マスター検索失敗: ${res.status}`);
		throw new Error(errText);
	}
	return await res.json();
}

// (master_data.js に移管したため削除)
// export async function fetchWholesalers() { ... }
// export async function fetchWholesalerMap() { ... }

export function toHalfWidthKatakana(str) {
// ... (中略: toHalfWidthKatakana は変更なし) ...
}

// ▼▼▼【ここから修正】共通ファイルアップロード関数 (Go APIがJSONを返す前提に変更) ▼▼▼
/**
 * ファイルアップロードを処理する共通関数
 * @param {string} apiEndpoint - アップロード先のAPI URL (例: '/api/dat/upload')
 * @param {FileList} files - <input type="file"> から取得したファイルリスト
 * @param {HTMLElement} fileInput - ファイル入力要素 (処理後にリセットするため)
 * @param {HTMLElement} uploadResultContainer - 処理結果のサマリーを表示するコンテナ
 * @param {HTMLElement} dataTable - 処理結果のテーブルを表示するコンテナ
 * @param {string} loadingMessage - ローディング中に表示するメッセージ (例: 'DATファイルを処理中...')
 */
export async function handleFileUpload(apiEndpoint, files, fileInput, uploadResultContainer, dataTable, loadingMessage) {
    if (!files || files.length === 0) {
      return;
    }

    if (uploadResultContainer) uploadResultContainer.innerHTML = '<p>ファイルをアップロード中...</p>';
    if (dataTable) dataTable.innerHTML = '<thead></thead><tbody><tr><td colspan="13">処理中...</td></tr></tbody>';
    window.showLoading(loadingMessage);

    const formData = new FormData();
    for (const file of files) {
        formData.append('file', file);
    }

    try {
        const response = await fetch(apiEndpoint, {
            method: 'POST',
            body: formData,
        });
        
        // ▼▼▼【ここから修正】JSONパース失敗に備える ▼▼▼
        const responseText = await response.text();
        let result;
        try {
            result = JSON.parse(responseText);
        } catch (jsonError) {
            // JSONパースに失敗した場合 (サーバーがプレーンテキストのエラーを返した)
            if (!response.ok) {
                // サーバーエラー (400, 500) かつ JSON ではない
                throw new Error(responseText || `サーバーエラー (HTTP ${response.status})`);
            }
            // サーバー 200 OK だが JSON ではない (通常ありえないが念のため)
            result = { message: responseText };
        }
        // ▲▲▲【修正ここまで】▲▲▲

        if (!response.ok) {
            throw new Error(result.message || `サーバーエラー (HTTP ${response.status})`);
        }

        let summaryHtml = `<h3>${result.message || '処理が完了しました。'}</h3>`;
        if (result.results && Array.isArray(result.results)) {
            summaryHtml += '<ul>';
            result.results.forEach(fileResult => {
                const statusClass = fileResult.success ? 'status-success' : 'status-error';
                const statusText = fileResult.success ? '成功' : 'エラー';
                const errorDetail = fileResult.error ? `: ${fileResult.error}` : '';
                const parsed = fileResult.records_parsed || 0;
                const inserted = fileResult.records_inserted || 0;

                summaryHtml += `<li><strong>${fileResult.filename}:</strong> `;
                summaryHtml += `<span class="${statusClass}">${statusText}</span> (パース: ${parsed}件, 登録: ${inserted}件)${errorDetail}`;
                summaryHtml += '</li>';
            });
            summaryHtml += '</ul>';
        }
        if (uploadResultContainer) uploadResultContainer.innerHTML = summaryHtml;
        // ▼▼▼【修正】result.tableHTML を削除し、result.records を共通関数に渡す ▼▼▼
        if (dataTable && result.records && result.records.length > 0) { 
            dataTable.innerHTML = renderTransactionTableHTML(result.records);
        } else if (dataTable) {
            dataTable.innerHTML = renderEmptyTableHTML();
        }
        // ▲▲▲【修正ここまで】▲▲▲
        
        window.showNotification(result.message || 'ファイルの処理が完了しました。', 'success');
    } catch (error) {
        console.error('Upload failed:', error);
        if (uploadResultContainer) uploadResultContainer.innerHTML = `<p style="color: red;">エラーが発生しました: ${error.message}</p>`;
        window.showNotification(`エラー: ${error.message}`, 'error');
        if (dataTable) {
            dataTable.innerHTML = `<thead></thead><tbody><tr><td colspan="13" class="status-error">エラーが発生しました: ${error.message}</td></tr></tbody>`;
        }
    } finally {
        window.hideLoading();
        if (fileInput) fileInput.value = '';
    }
}
// ▲▲▲【修正ここまで】▲▲▲