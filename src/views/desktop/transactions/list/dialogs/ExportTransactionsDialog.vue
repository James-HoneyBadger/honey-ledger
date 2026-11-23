<!-- SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe -->
<!-- SPDX-License-Identifier: GPL-3.0-or-later -->

<template>
    <v-dialog width="600" v-model="showState">
        <v-card class="pa-sm-1 pa-md-2">
            <template #title>
                <h4 class="text-h4">{{ tt('Export Transactions') }}</h4>
            </template>

            <v-card-text>
                <span class="text-body-1">{{ tt('Export the transactions that match the current filter.') }}</span>
                <v-select class="mt-4"
                          item-title="displayName"
                          item-value="type"
                          :label="tt('File Format')"
                          :placeholder="tt('File Format')"
                          :items="fileTypes"
                          v-model="fileType" />
            </v-card-text>

            <v-card-text>
                <div class="d-flex justify-center justify-sm-space-between flex-wrap mt-sm-1 mt-md-2 gap-4">
                    <v-btn color="secondary" variant="tonal" :prepend-icon="mdiClose" @click="close()">{{ tt('Cancel') }}</v-btn>
                    <v-btn color="primary" :append-icon="mdiExport" @click="submit()">{{ tt('Export') }}</v-btn>
                </div>
            </v-card-text>
        </v-card>
    </v-dialog>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue';

import { useI18n } from '@/locales/helpers.ts';

import { KnownFileType } from '@/core/file.ts';

import {
    mdiClose,
    mdiExport
} from '@mdi/js';

const emit = defineEmits<{
    (e: 'export', fileType: string): void;
}>();

const { tt } = useI18n();

const showState = ref<boolean>(false);
const fileType = ref<string>(KnownFileType.CSV.extension);

const fileTypes = computed<{ type: string, displayName: string }[]>(() => [
    {
        type: KnownFileType.CSV.extension,
        displayName: tt('CSV (Comma-separated values) File')
    },
    {
        type: KnownFileType.TSV.extension,
        displayName: tt('TSV (Tab-separated values) File')
    }
]);

function open(defaultFileType?: string): void {
    fileType.value = defaultFileType === KnownFileType.TSV.extension ? KnownFileType.TSV.extension : KnownFileType.CSV.extension;
    showState.value = true;
}

function close(): void {
    showState.value = false;
}

function submit(): void {
    const selectedFileType = fileType.value;

    showState.value = false;
    emit('export', selectedFileType);
}

defineExpose({
    open
});
</script>
