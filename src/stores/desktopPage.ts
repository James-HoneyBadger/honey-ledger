// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

import { ref } from 'vue';
import { defineStore } from 'pinia';

export const useDesktopPageStore = defineStore('desktopPages', () => {
    const showAddTransactionDialogInTransactionList = ref<boolean>(false);

    function setShowAddTransactionDialogInTransactionList(): void {
        showAddTransactionDialogInTransactionList.value = true;
    }

    function resetShowAddTransactionDialogInTransactionList(): void {
        showAddTransactionDialogInTransactionList.value = false;
    }

    return {
        // states
        showAddTransactionDialogInTransactionList,
        // functions
        setShowAddTransactionDialogInTransactionList,
        resetShowAddTransactionDialogInTransactionList
    }
});
