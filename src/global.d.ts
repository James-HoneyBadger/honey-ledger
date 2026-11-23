// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

declare const __HONEY_LEDGER_IS_PRODUCTION__: boolean;
declare const __HONEY_LEDGER_VERSION__: string;
declare const __HONEY_LEDGER_BUILD_UNIX_TIME__: string;
declare const __HONEY_LEDGER_BUILD_COMMIT_HASH__: string;
declare const __HONEY_LEDGER_LICENSE__: string;
declare const __HONEY_LEDGER_THIRD_PARTY_LICENSES__: LicenseInfo[];

declare interface LicenseInfo {
    name: string;
    copyright?: string;
    url?: string;
    license?: string;
    licenseUrl?: string;
}

interface Window {
    HONEY_LEDGER_SERVER_SETTINGS?: {
        [key: string]: string | number | boolean | undefined | null;
    };
    honeyLedger?: {
        platform: string;
        isDesktopApp: boolean;
        apiBaseUrl?: string;
        showNotification?: (title: string, body: string) => void;
        onQuickAddTransaction?: (callback: () => void) => void;
        onImportTransactions?: (callback: () => void) => void;
        onExportTransactions?: (callback: (fileType: string) => void) => void;
        saveFile?: (fileName: string, data: Uint8Array) => Promise<boolean>;
    };
}

interface Navigator {
    browserLanguage?: string;
}

interface Credential {
    rawId: ArrayBuffer;
    response: {
        clientDataJSON: ArrayBuffer;
        attestationObject: ArrayBuffer;
        userHandle: ArrayBuffer;
    };
}
