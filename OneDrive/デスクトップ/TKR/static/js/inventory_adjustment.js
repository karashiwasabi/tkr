import { setUnitMap } from './inventory_adjustment_ui.js';
import { initSearchForm, setCurrentYjCode } from './inventory_adjustment_search.js';
import { initDetails, loadAndRenderDetails } from './inventory_adjustment_details.js';

export async function initInventoryAdjustment() {
    let localUnitMap = {};
    try {
        const res = await fetch('/api/units/map');
        if (!res.ok) throw new Error('単位マスタの取得に失敗');
        localUnitMap = await res.json();
        setUnitMap(localUnitMap);
    } catch (err) {
        console.error(err);
        window.showNotification(err.message, 'error');
    }

    initDetails();
    initSearchForm(loadAndRenderDetails);

    document.addEventListener('loadInventoryAdjustment', (e) => {
        const { yjCode } = e.detail;
        if (yjCode) {
            setCurrentYjCode(yjCode);
            loadAndRenderDetails(yjCode);
        }
    });

    console.log("Inventory Adjustment View Initialized (Hub).");
}