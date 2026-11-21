// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package core

import "context"

// Context is the base context of Honey Ledger
type Context interface {
	context.Context
	GetContextId() string
	GetClientLocale() string
}
