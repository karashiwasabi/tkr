import { refreshWholesalerMap } from './master_data.js';
import { initWholesalerManagement } from './config_wholesaler.js';
import { initDataMigration } from './config_migration.js';

let configSavePathBtn, datFolderPathInput, usageFolderPathInput;
let configSaveDaysBtn, calculationDaysInput;

async function loadConfig() {
    try {
        const response = await fetch('/api/config');
        if (!response.ok) {
            throw new Error('設定の読み込みに失敗しました。');
        }
        const config = await response.json();
        if (datFolderPathInput) {
            datFolderPathInput.value = config.datFolderPath || '';
        }
        if (usageFolderPathInput) {
            usageFolderPathInput.value = config.usageFolderPath || '';
        }
        if (calculationDaysInput) {
            calculationDaysInput.value = config.calculationPeriodDays || 90;
        }
    } catch (error) {
        window.showNotification(error.message, 'error');
    }
}

export async function loadConfigAndWholesalers() {
    window.showLoading('設定情報を読み込み中...');
    try {
        await refreshWholesalerMap();
        await initWholesalerManagement(); 
        await loadConfig();
    } catch (error) {
        window.showNotification(error.message, 'error');
    } finally {
        window.hideLoading();
    }
}

async function handleSavePaths() {
    const newConfig = {
        datFolderPath: datFolderPathInput.value,
        usageFolderPath: usageFolderPathInput.value,
        calculationPeriodDays: parseInt(calculationDaysInput.value, 10) || 90
    };
    await saveConfig(newConfig, 'パス設定を保存しました。');
}

async function handleSaveDays() {
    const newConfig = {
        datFolderPath: datFolderPathInput.value,
        usageFolderPath: usageFolderPathInput.value,
        calculationPeriodDays: parseInt(calculationDaysInput.value, 10) || 90
    };
    await saveConfig(newConfig, '集計期間を保存しました。');
}

async function saveConfig(configData, successMessage) {
    window.showLoading('設定を保存中...');
    try {
        const response = await fetch('/api/config', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(configData),
        });
        const result = await response.json();
        if (!response.ok) {
            throw new Error(result.message || '設定の保存に失敗しました。');
        }
        window.showNotification(successMessage, 'success');
        await loadConfig();
    } catch (error) {
        window.showNotification(error.message, 'error');
    } finally {
        window.hideLoading();
    }
}

export function initConfigView() {
    configSavePathBtn = document.getElementById('configSavePathBtn');
    datFolderPathInput = document.getElementById('config-dat-folder-path');
    usageFolderPathInput = document.getElementById('config-usage-folder-path');
    configSaveDaysBtn = document.getElementById('configSaveDaysBtn');
    calculationDaysInput = document.getElementById('config-calculation-days');

    if (configSavePathBtn) {
        configSavePathBtn.addEventListener('click', handleSavePaths);
    }
    if (configSaveDaysBtn) {
        configSaveDaysBtn.addEventListener('click', handleSaveDays);
    }

    initDataMigration();

    console.log("Config View Initialized.");
}