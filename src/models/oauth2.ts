// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

export interface OAuth2CallbackLoginRequest {
    readonly password?: string;
    readonly passcode?: string;
    readonly token?: string;
}
