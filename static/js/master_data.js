// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\master_data.js

export let wholesalerMap = new Map();
export let clientMap = new Map();

/**
 * 卸マスタのリストを取得します。
 */
// ▼▼▼【ここから修正】元の /api/wholesalers/list を参照する状態に戻す ▼▼▼
async function fetchWholesalers() {
    try {
        const response = await fetch('/api/wholesalers/list');
        if (!response.ok) {
            throw new Error(`卸一覧の読み込みに失敗しました: ${response.statusText}`);
        }
          return await response.json();
    } catch (error) {
        console.error("Error loading wholesalers:", error);
        window.showNotification(error.message, 'error');
        return [];
    }
}
// ▲▲▲【修正ここまで】▲▲▲

/**
 * 卸マスタのマップを取得します。
 */
// ▼▼▼【ここから修正】wholesaler API のレスポンス (wholesalerCode, wholesalerName) をパースする元の状態に戻す ▼▼▼
async function fetchWholesalerMap() {
    const wholesalers = await fetchWholesalers();
    const map = new Map();
    if (wholesalers) {
        wholesalers.forEach(w => {
            // ▼▼▼【修正】キー登録時に trim() を追加 ▼▼▼
            if (w.wholesalerCode) {
                map.set(w.wholesalerCode.trim(), w.wholesalerName);
            }
            // ▲▲▲【修正ここまで】▲▲▲
        });
    }
    return map;
}
// ▲▲▲【修正ここまで】▲▲▲

/**
 * 内部の wholesalerMap をAPIから取得した最新のデータで更新します。
 */
async function fetchAndPopulateWholesalers() {
    wholesalerMap = await fetchWholesalerMap();
}

/**
 * ▼▼▼【ここから修正】得意先マスタを読み込む (WASABI: master_data.js より) ▼▼▼
 */
async function fetchAndPopulateClients() {
	try {
		const res = await fetch('/api/clients');
		if (!res.ok) {
			throw new Error('得意先マスターの取得に失敗しました。');
		}
		const clients = await res.json();
		clientMap.clear();
		if (clients) {
            // ▼▼▼【修正】キー登録時に trim() を追加 ▼▼▼
			clients.forEach(c => {
                if (c.clientCode) {
                    clientMap.set(c.clientCode.trim(), c.clientName);
                }
            });
            // ▲▲▲【修正ここまで】▲▲▲
		}
	} catch (error) {
		console.error("Error loading clients:", error);
        window.showNotification(error.message, 'error');
	}
}

/**
 * 得意先マップのキャッシュを強制的に更新します。
 */
export async function refreshClientMap() {
    try {
         await fetchAndPopulateClients();
        console.log('得意先マップを更新しました。');
    } catch (error) {
         console.error("得意先マップの更新に失敗しました:", error);
        window.showNotification('得意先リストの更新に失敗しました。', 'error');
    }
}
// ▲▲▲【修正ここまで】▲▲▲

/**
 * 外部モジュール（config.jsなど）から卸マスタのキャッシュを強制的に更新するために呼び出されます。
 */
export async function refreshWholesalerMap() {
    try {
         await fetchAndPopulateWholesalers();
        console.log('卸業者マップを更新しました。');
    } catch (error) {
        console.error("卸業者マップの更新に失敗しました:", error);
        window.showNotification('卸業者リストの更新に失敗しました。', 'error');
    }
}

/**
 * アプリ起動時(app.js)に一度だけ呼び出され、全てのマスターデータをキャッシュします。
 */
export async function loadMasterData() {
    try {
        await Promise.all([
             fetchAndPopulateClients(),
             fetchAndPopulateWholesalers()
         ]);
        console.log('マスターデータを読み込みました。');
    } catch (error) {
         console.error("マスターデータの読み込み中にエラーが発生しました:", error);
        window.showNotification('マスターデータの読み込みに失敗しました。', 'error');
    }
}