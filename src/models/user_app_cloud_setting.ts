// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

import type { ApplicationCloudSetting } from '@/core/setting.ts';

export interface UserApplicationCloudSettingsUpdateRequest {
    readonly settings: ApplicationCloudSetting[];
    readonly fullUpdate: boolean;
}
