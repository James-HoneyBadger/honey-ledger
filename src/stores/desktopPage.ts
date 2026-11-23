// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

import { ref } from 'vue';
import { defineStore } from 'pinia';

export const useDesktopPageStore = defineStore('desktopPages', () => {
    const showAddTransactionDialogInTransactionList = ref<boolean>(false);
    const showImportTransactionDialogInTransactionList = ref<boolean>(false);
    const showExportTransactionDialogInTransactionList = ref<boolean>(false);
    const exportTransactionFileType = ref<string | undefined>(undefined);

    function setShowAddTransactionDialogInTransactionList(): void {
        showAddTransactionDialogInTransactionList.value = true;
    }

    function resetShowAddTransactionDialogInTransactionList(): void {
        showAddTransactionDialogInTransactionList.value = false;
    }

    function setShowImportTransactionDialogInTransactionList(): void {
        showImportTransactionDialogInTransactionList.value = true;
    }

    function resetShowImportTransactionDialogInTransactionList(): void {
        showImportTransactionDialogInTransactionList.value = false;
    }

    function setShowExportTransactionDialogInTransactionList(fileType?: string): void {
        exportTransactionFileType.value = fileType;
        showExportTransactionDialogInTransactionList.value = true;
    }

    function resetShowExportTransactionDialogInTransactionList(): void {
        showExportTransactionDialogInTransactionList.value = false;
        exportTransactionFileType.value = undefined;
    }

    return {
        // states
        showAddTransactionDialogInTransactionList,
        showImportTransactionDialogInTransactionList,
        showExportTransactionDialogInTransactionList,
        exportTransactionFileType,
        // functions
        setShowAddTransactionDialogInTransactionList,
        resetShowAddTransactionDialogInTransactionList,
        setShowImportTransactionDialogInTransactionList,
        resetShowImportTransactionDialogInTransactionList,
        setShowExportTransactionDialogInTransactionList,
        resetShowExportTransactionDialogInTransactionList
    }
});
