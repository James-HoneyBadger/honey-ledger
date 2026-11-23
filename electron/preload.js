// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

const { contextBridge, ipcRenderer } = require('electron');

contextBridge.exposeInMainWorld('honeyLedger', {
    platform: process.platform,
    isDesktopApp: true,
    // The app:// protocol handler proxies API calls transparently,
    // so the Vue app just needs to know the origin for API calls
    apiBaseUrl: 'app://honey-ledger',

    // Desktop notifications — send from renderer to main process
    showNotification: (title, body) => {
        ipcRenderer.send('show-notification', { title, body });
    },

    // Listen for quick-add transaction shortcut from main process
    onQuickAddTransaction: (callback) => {
        ipcRenderer.on('quick-add-transaction', () => callback());
    },

    // Listen for the File menu import/export entries
    onImportTransactions: (callback) => {
        ipcRenderer.on('import-transactions', () => callback());
    },

    onExportTransactions: (callback) => {
        ipcRenderer.on('export-transactions', (_event, fileType) => callback(fileType));
    },

    // Save an exported file through the native save dialog.
    saveFile: (fileName, data) => {
        return ipcRenderer.invoke('save-file', { fileName, data });
    }
});
