// C:\Users\wasab\OneDrive\デスクトップ\WASABI\static\js\utils.js

/**
 * 文字列内のひらがなをカタカナに変換します。
 * @param {string} str 変換する文字列
 * @returns {string} カタカナに変換された文字列
 */
export function hiraganaToKatakana(str) {
    if (!str) return ''; 
    return str.replace(/[\u3041-\u3096]/g, function(match) {
        const charCode = match.charCodeAt(0) + 0x60;
        return String.fromCharCode(charCode);
    }); 
}

/**
 * 現在のPCのローカル日付を 'YYYY-MM-DD' 形式の文字列で返します。
 * @returns {string} 'YYYY-MM-DD' 形式の文字列
 */
export function getLocalDateString() {
    const today = new Date(); 
    const yyyy = today.getFullYear();
    const mm = String(today.getMonth() + 1).padStart(2, '0');
    const dd = String(today.getDate()).padStart(2, '0');
    return `${yyyy}-${mm}-${dd}`; 
}

// ▼▼▼【ここから修正】ひらがな・全角カナ・全角英数を半角カナに変換する関数を追加 ▼▼▼

const kanaMap = {
    'ガ': 'ｶﾞ', 'ギ': 'ｷﾞ', 'グ': 'ｸﾞ', 'ゲ': 'ｹﾞ', 'ゴ': 'ｺﾞ',
    'ザ': 'ｻﾞ', 'ジ': 'ｼﾞ', 'ズ': 'ｽﾞ', 'ゼ': 'ｾﾞ', 'ゾ': 'ｿﾞ',
    'ダ': 'ﾀﾞ', 'ヂ': 'ﾁﾞ', 'ヅ': 'ﾂﾞ', 'デ': 'ﾃﾞ', 'ド': 'ﾄﾞ',
    'バ': 'ﾊﾞ', 'ビ': 'ﾋﾞ', 'ブ': 'ﾌﾞ', 'ベ': 'ﾍﾞ', 'ボ': 'ﾎﾞ',
    'パ': 'ﾊﾟ', 'ピ': 'ﾋﾟ', 'プ': 'ﾌﾟ', 'ペ': 'ﾍﾟ', 'ポ': 'ﾎﾟ',
    'ア': 'ｱ', 'イ': 'ｲ', 'ウ': 'ｳ', 'エ': 'ｴ', 'オ': 'ｵ',
    'カ': 'ｶ', 'キ': 'ｷ', 'ク': 'ｸ', 'ケ': 'ｹ', 'コ': 'ｺ',
    
    'サ': 'ｻ', 'シ': 'ｼ', 'ス': 'ｽ', 'セ': 'ｾ', 'ソ': 'ｿ', 
    'タ': 'ﾀ', 'チ': 'ﾁ', 'ツ': 'ﾂ', 'テ': 'ﾃ', 'ト': 'ﾄ',
    'ナ': 'ﾅ', 'ニ': 'ﾆ', 'ヌ': 'ﾇ', 'ネ': 'ﾈ', 'ノ': 'ﾉ',
    'ハ': 'ﾊ', 'ヒ': 'ﾋ', 'フ': 'ﾌ', 'ヘ': 'ﾍ', 'ホ': 'ﾎ',
    'マ': 'ﾏ', 'ミ': 'ﾐ', 'ム': 'ﾑ', 'メ': 'ﾒ', 'モ': 'ﾓ',
    'ヤ': 'ﾔ', 'ユ': 'ﾕ', 'ヨ': 'ﾖ',
    'ラ': 'ﾗ', 'リ': 'ﾘ', 'ル': 'ﾙ', 'レ': 'ﾚ', 'ロ': 'ﾛ',
    'ワ': 'ﾜ', 'ヲ': 'ｦ', 'ン': 'ﾝ',
    'ァ': 'ｧ', 'ィ': 'ｨ', 
    'ゥ': 'ｩ', 'ェ': 'ｪ', 'ォ': 'ｫ', 
    'ッ': 'ｯ', 'ャ': 'ｬ', 'ュ': 'ｭ', 'ョ': 'ｮ',
    '。': '｡', '「': '｢', '」': '｣', '、': '､', '・': '･', 'ー': 'ｰ', 'ヴ': 'ｳﾞ',
    'A': 'ｱ', 'B': 'ｲ', 'C': 'ｳ', 'D': 'ｴ', 'E': 'ｵ', // 例：英字もカナに変換（JCSHMSの慣習）
    '0': '0', '1': '1', '2': '2', '3': '3', '4': '4', '5': '5', '6': '6', '7': '7', '8': '8', '9': '9',
}; 
/**
 * 文字列に含まれるひらがな、全角カタカナ、全角英数を半角カタカナに変換します。
 * @param {string} str 変換する文字列
 * @returns {string} 半角カタカナに変換された文字列
 */
export function toHalfWidthKatakana(str) {
    if (!str) return ''; 
    let result = str.toUpperCase(); // 全て大文字に

    // 1. ひらがなをカタカナに変換
    result = hiraganaToKatakana(result); 
// 2. 全角カタカナ・英字・記号を半角カタカナに変換
    let halfWidth = ''; 
    for (const char of result) {
        if (kanaMap[char]) {
            halfWidth += kanaMap[char]; 
        } else {
            // 全角英数字を半角に（toHalfWidth の代わり）
            const code = char.charCodeAt(0); 
            if (code >= 0xFF01 && code <= 0xFF5E) { // 全角英数記号
                halfWidth += String.fromCharCode(code - 0xFEE0); 
            } else if (char === '　') { // 全角スペース
                halfWidth += ' '; 
            } else {
                halfWidth += char; 
            }
        }
    }
    return halfWidth.replace(/ /g, ''); 
// スペースを全て削除
}
// ▲▲▲【修正ここまで】▲▲▲

// export function toHalfWidth(str) { ... } は使用しないため削除