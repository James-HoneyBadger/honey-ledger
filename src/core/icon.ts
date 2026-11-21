// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

export type LineAwesomeIconClassName = string;

export interface IconInfo extends Record<string, unknown> {
    readonly icon: LineAwesomeIconClassName;
}

export interface IconInfoWithId extends IconInfo {
    readonly id: string;
    readonly icon: LineAwesomeIconClassName;
}
