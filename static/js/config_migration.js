// C:\Users\wasab\OneDrive\デスクトップ\TKR\static\js\config_migration.js
import { initStockMigration } from './config_stock_migration.js';
import { initMasterMigration } from './config_master_migration.js';
import { initClientMigration } from './config_client_migration.js';
import { initPrecompMigration } from './config_precomp_migration.js';
import { initMaintenance } from './config_maintenance.js';

export function initDataMigration() {
    initStockMigration();
    initMasterMigration();
    initClientMigration();
    initPrecompMigration();
    initMaintenance();
}