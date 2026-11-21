// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

export interface VersionInfo {
    readonly version: string;
    readonly commitHash: string;
    readonly buildTime?: string;
}
