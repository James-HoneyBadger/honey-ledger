// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package avatars

import "github.com/James-HoneyBadger/honey-ledger/pkg/models"

// AvatarProvider is user avatar provider interface
type AvatarProvider interface {
	GetAvatarUrl(user *models.User) string
}
