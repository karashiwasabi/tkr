// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\utils.js
import { renderTransactionTableHTML, renderEmptyTableHTML } from './common_table.js';

export function hiraganaToKatakana(str) {
	if (!str) return '';
	return str.replace(/[\u3041-\u3096]/g, function (match) {
		const charCode = match.charCodeAt(0) + 0x60;
		return String.fromCharCode(charCode);
	});
}

export function getLocalDateString(date = null) {
	const today = date instanceof Date ? date : new Date();
	const yyyy = today.getFullYear();
	const mm = String(today.getMonth() + 1).padStart(2, '0');
	const dd = String(today.getDate()).padStart(2, '0');
	return `${yyyy}-${mm}-${dd}`;
}

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

export function toHalfWidthKatakana(str) {
    if (!str) return '';
    const kanaMap = {
        'ガ': 'ｶﾞ', 'ギ': 'ｷﾞ', 'グ': 'ｸﾞ', 'ゲ': 'ｹﾞ', 'ゴ': 'ｺﾞ',
        'ザ': 'ｻﾞ', 'ジ': 'ｼﾞ', 'ズ': 'ｽﾞ', 'ゼ': 'ｾﾞ', 'ゾ': 'ｿﾞ',
        'ダ': 'ﾀﾞ', 'ヂ': 'ﾁﾞ', 'ヅ': 'ﾂﾞ', 'デ': 'ﾃﾞ', 'ド': 'ﾄﾞ',
        'バ': 'ﾊﾞ', 'ビ': 'ﾋﾞ', 'ブ': 'ﾌﾞ', 'ベ': 'ﾍﾞ', 'ボ': 'ﾎﾞ',
        'パ': 'ﾊﾟ', 'ピ': 'ﾋﾟ', 'プ': 'ﾌﾟ', 'ペ': 'ﾍﾟ', 'ポ': 'ﾎﾟ',
        'ヴ': 'ｳﾞ', 'ア': 'ｱ', 'イ': 'ｲ', 'ウ': 'ｳ', 'エ': 'ｴ', 'オ': 'ｵ',
        'カ': 'ｶ', 'キ': 'ｷ', 'ク': 'ｸ', 'ケ': 'ｹ', 'コ': 'ｺ',
        'サ': 'ｻ', 'シ': 'ｼ', 'ス': 'ｽ', 'セ': 'ｾ', 'ソ': 'ｿ',
        'タ': 'ﾀ', 'チ': 'ﾁ', 'ツ': 'ﾂ', 'テ': 'ﾃ', 'ト': 'ﾄ',
        'ナ': 'ﾅ', 'ニ': 'ﾆ', 'ヌ': 'ﾇ', 'ネ': 'ﾈ', 'ノ': 'ﾉ',
        'ハ': 'ﾊ', 'ヒ': 'ﾋ', 'フ': 'ﾌ', 'ヘ': 'ﾍ', 'ホ': 'ﾎ',
        'マ': 'ﾏ', 'ミ': 'ﾐ', 'ム': 'ﾑ', 'メ': 'ﾒ', 'モ': 'ﾓ',
        'ヤ': 'ﾔ', 'ユ': 'ﾕ', 'ヨ': 'ﾖ',
        'ラ': 'ﾗ', 'リ': 'ﾘ', 'ル': 'ﾙ', 'レ': 'ﾚ', 'ロ': 'ﾛ',
        'ワ': 'ﾜ', 'ヲ': 'ｦ', 'ン': 'ﾝ',
        'ァ': 'ｧ', 'ィ': 'ｨ', 'ゥ': 'ｩ', 'ェ': 'ｪ', 'ォ': 'ｫ',
        'ッ': 'ｯ', 'ャ': 'ｬ', 'ュ': 'ｭ', 'ョ': 'ｮ',
        'ー': 'ｰ', '。': '｡', '、': '､', '「': '｢', '」': '｣', '・': '･'
    };
    return str.replace(/[ァ-ンー。、「」・]/g, function(s) {
        return kanaMap[s] || s;
    });
}

/**
 * 日付文字列(YYYYMM or YYYYMMDD)を受け取り、期限状態を判定する
 * 戻り値: 'status-expired' (赤), 'status-near' (青), '' (なし)
 */
export function getExpiryStatus(dateStr) {
    if (!dateStr) return '';
    
    let year, month;
    // YYYYMM または YYYYMMDD に対応
    if (dateStr.length >= 6) {
        year = parseInt(dateStr.substring(0, 4), 10);
        month = parseInt(dateStr.substring(4, 6), 10);
    } else {
        return '';
    }

    const now = new Date();
    const currentYear = now.getFullYear();
    const currentMonth = now.getMonth() + 1;

    // 月差分を計算 (マイナスなら過去、0なら今月、1なら来月)
    const monthDiff = (year - currentYear) * 12 + (month - currentMonth);

    if (monthDiff < 0) {
        return 'status-expired'; // 期限切れ (赤)
    } else if (monthDiff <= 1) {
        return 'status-near';    // 残り1ヶ月以内 (青)
    }

    return ''; // 安全 (緑などは付けない)
}

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
        
        const responseText = await response.text();
        let result;
        try {
          result = JSON.parse(responseText);
        } catch (jsonError) {
            if (!response.ok) {
                throw new Error(responseText || `サーバーエラー (HTTP ${response.status})`);
            }
            result = { message: responseText };
        }

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
        
        if (dataTable && result.records && result.records.length > 0) { 
            dataTable.innerHTML = renderTransactionTableHTML(result.records);
        } else if (dataTable) {
            dataTable.innerHTML = renderEmptyTableHTML();
        }
        
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