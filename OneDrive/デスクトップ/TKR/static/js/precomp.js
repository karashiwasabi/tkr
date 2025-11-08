// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\precomp.js
import { initHeader, resetHeader } from './precomp_header.js';
import { initDetailsTable, clearDetailsTable } from './precomp_details_table.js';

export function resetPrecompView() {
    resetHeader();
    clearDetailsTable();
}

export function initPrecomp() {
    initHeader();
    initDetailsTable();
}