// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\utils.js
// (hiraganaToKatakana, getLocalDateString は変更なし)
// ...
export function hiraganaToKatakana(str) {
	if (!str) return '';
return str.replace(/[\u3041-\u3096]/g, function (match) {
		const charCode = match.charCodeAt(0) + 0x60;
		return String.fromCharCode(charCode);
	});
}
export function getLocalDateString() {
	const today = new Date();
const yyyy = today.getFullYear();
	const mm = String(today.getMonth() + 1).padStart(2, '0');
	const dd = String(today.getDate()).padStart(2, '0');
	return `${yyyy}-${mm}-${dd}`;
}

// ▼▼▼【ここから修正】Goの barcode.go と同じロジックに書き換える ▼▼▼

/**
 * AI付き文字列を解析する内部関数
 * (Go: barcode/barcode.go  の parseAIString と同等)
 */
function parseAIString(code) {
	let rest = code;
const data = { gtin14: '', expiryDate: '', lotNumber: '' };
// (01) GTIN
	if (rest.startsWith('01')) {
		if (rest.length < 16) throw new Error('AI(01)のデータが不足しています');
		data.gtin14 = rest.substring(2, 16);
		rest = rest.substring(16);
} else {
		throw new Error('AI(01)が見つかりません');
	}

	// (17) 有効期限
	if (rest.startsWith('17')) {
		if (rest.length < 8) return data;
// データ不足の場合はここで終了
		const yy_mm_dd = rest.substring(2, 8);
		if (yy_mm_dd.length === 6) {
			const yy = yy_mm_dd.substring(0, 2);
			const mm = yy_mm_dd.substring(2, 4);
data.expiryDate = `20${yy}${mm}`; // YYMMDD -> YYYYMM
		}
		rest = rest.substring(8);
	}

	// (10) ロット番号
	if (rest.startsWith('10')) {
		if (rest.length <= 2) return data;
// データなし

		rest = rest.substring(2);
		// AI(10)部分を削除

		let endIndex = rest.length;
		const maxLength = 20; // AI(10)の最大長

		// 次のAI(17または01)を探す
		const next17 = rest.indexOf('17');
if (next17 > -1 && rest.length >= next17 + 8) {
			// AI(17)+6桁
			endIndex = next17;
		}

		const next01 = rest.indexOf('01');
if (next01 > -1 && rest.length >= next01 + 16) {
			// AI(01)+14桁
			if (next01 < endIndex) {
				// (17)よりも手前に(01)がある場合
				endIndex = next01;
}
		}

		// 最大長(20桁)の制限
		if (endIndex > maxLength) {
			endIndex = maxLength;
		}

		data.lotNumber = rest.substring(0, endIndex);
	}

	return data;
}

/**
 * GS1-128バーコードまたはJAN/GTINを解析する
 * (Go: barcode/barcode.go  の Parse と同仕様)
 */
export function parseBarcode(code) {
	const length = code.length;
if (length === 0) {
		throw new Error('バーコードが空です');
	}

	// 仕様 1: 15桁以上は AI付き文字列
	if (length >= 15) {
		if (code.startsWith('01')) {
			return parseAIString(code);
}
		throw new Error('15桁以上ですが、AI(01)で始まっていません');
	}

	// 仕様 2: 14桁は GTIN-14
	if (length === 14) {
		return { gtin14: code, expiryDate: '', lotNumber: '' };
}

	// 仕様 3: 13桁は JAN-13
	if (length === 13) {
		return { gtin14: '0' + code, expiryDate: '', lotNumber: '' };
}

	// 仕様 4: 13桁未満は JAN-8 など (先頭ゼロ埋め)
	if (length < 13) {
		return { gtin14: code.padStart(14, '0'), expiryDate: '', lotNumber: '' };
}

	// (ここに来ることはないはず)
	throw new Error('不明なバーコード形式です');
}
// ▲▲▲【修正ここまで】▲▲▲

// ▼▼▼【ここから追加】バーコードでマスターを検索する共通API関数 ▼▼▼
/**
 * バーコード（JANまたはGS1）を使ってマスターを検索する共通API関数
 * @param {string} barcode - JAN(13桁以下)またはGS1(14桁以上)のバーコード文字列
 * @returns {Promise<object>} 成功時: ProductMasterView オブジェクト
 * @throws {Error} 失敗時: エラーメッセージ付きの Error オブジェクト
 */
export async function fetchProductMasterByBarcode(barcode) {
	if (!barcode) {
		throw new Error('バーコードが空です');
}

	// Go側の単一APIを呼び出す
	const res = await fetch(`/api/product/by_barcode/${barcode}`);

	if (!res.ok) {
		if (res.status === 404) {
			// "Product not found" (Go側)
			throw new Error('このバーコードはマスターに登録されていません。');
}
		// その他のGo側エラー (バーコード解析エラー、DBエラーなど)
		const errText = await res.text().catch(() => `マスター検索失敗: ${res.status}`);
		throw new Error(errText);
	}
	return await res.json();
}
// ▲▲▲【追加ここまで】▲▲▲

// (TKR版の toHalfWidthKatakana はそのまま)
const kanaMap = {
	'ガ': 'ｶﾞ',
	'ギ': 'ｷﾞ',
	'グ': 'ｸﾞ',
	'ゲ': 'ｹﾞ',
	'ゴ': 'ｺﾞ',
	'ザ': 'ｻﾞ',
	'ジ': 'ｼﾞ',
	'ズ': 'ｽﾞ',
	'ゼ': 'ｾﾞ',
	'ゾ': 'ｿﾞ',
	'ダ': 'ﾀﾞ',
	'ヂ': 'ﾁﾞ',
	'ヅ': 'ﾂﾞ',
	'デ': 'ﾃﾞ',
	'ド': 'ﾄﾞ',
	'バ': 'ﾊﾞ',
	'ビ': 'ﾋﾞ',
	'ブ': 'ﾌﾞ',
	'ベ': 'ﾍﾞ',
	'ボ': 'ﾎﾞ',
	'パ': 'ﾊﾟ',
	'ピ': 'ﾋﾟ',
	'プ': 'ﾌﾟ',
	'ペ': 'ﾍﾟ',
	'ポ': 'ﾎﾟ',
	'ア': 'ｱ',
	'イ': 'ｲ',
	'ウ': 'ｳ',
	'エ': 'ｴ',
	'オ': 'ｵ',
	'カ': 'ｶ',
	'キ': 'ｷ',
	'ク': 'ｸ',
	'ケ': 'ｹ',
	'コ': 'ｺ',
	'サ': 'ｻ',
	'シ': 'ｼ',
	'ス': 'ｽ',
	'セ': 'ｾ',
	'ソ': 'ｿ',
	'タ': 'ﾀ',
	'チ': 'ﾁ',
	'ツ': 'ﾂ',
	'テ': 'ﾃ',
	'ト': 'ﾄ',
	'ナ': 'ﾅ',
	'ニ': 'ﾆ',
	'ヌ': 'ﾇ',
	'ネ': 'ﾈ',
	'ノ': 'ﾉ',
	'ハ': 'ﾊ',
	'ヒ': 'ﾋ',
	'フ': 'ﾌ',
	'ヘ': 'ﾍ',
	'ホ': 'ﾎ',
	'マ': 'ﾏ',
	'ミ': 'ﾐ',
	'ム': 'ﾑ',
	'メ': 'ﾒ',
	'モ': 'ﾓ',
	'ヤ': 'ﾔ',
	'ユ': 'ﾕ',
	'ヨ': 'ﾖ',
	'ラ': 'ﾗ',
	'リ': 'ﾘ',
	'ル': 'ﾙ',
	'レ': 'ﾚ',
	'ロ': 'ﾛ',
	'ワ': 'ﾜ',
	'ヲ': 'ｦ',
	'ン': 'ﾝ',
	'ァ': 'ｧ',
	'ィ': 'ｨ',
	'ゥ': 'ｩ',
	'ェ': 'ｪ',
	'ォ': 'ｫ',
	'ッ': 'ｯ',
	'ャ': 'ｬ',
	'ュ': 'ｭ',
	'ョ': 'ｮ',
	'。': '｡',
	'「': '｢',
	'」': '｣',
	'、': '､',
	'・': '･',
	'ー': 'ｰ',
	'ヴ': 'ｳﾞ',
	'A': 'ｱ',
	'B': 'ｲ',
	'C': 'ｳ',
	'D': 'ｴ',
	'E': 'ｵ',
	'0': 
'0',
	'1': '1',
	'2': '2',
	'3': '3',
	'4': '4',
	'5': '5',
	'6': '6',
	'7': '7',
	'8': '8',
	'9': '9',
};
export function toHalfWidthKatakana(str) {
	if (!str) return '';
let result = str.toUpperCase();

	result = hiraganaToKatakana(result);
	let halfWidth = '';
for (const char of result) {
		if (kanaMap[char]) {
			halfWidth += kanaMap[char];
		} else {
			const code = char.charCodeAt(0);
if (code >= 0xff01 && code <= 0xff5e) {
				halfWidth += String.fromCharCode(code - 0xfee0);
} else if (char === '　') {
				halfWidth += ' ';
			} else {
				halfWidth += char;
			}
		}
	}
	return halfWidth.replace(/ /g, '');
}