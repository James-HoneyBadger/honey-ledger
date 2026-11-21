// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

export function getLicense(): string {
    return __HONEY_LEDGER_LICENSE__;
}

export function getThirdPartyLicenses(): LicenseInfo[] {
    return __HONEY_LEDGER_THIRD_PARTY_LICENSES__ || [];
}
