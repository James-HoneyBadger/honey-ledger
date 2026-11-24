// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import "context"

// Context is the application context the extraction runs with. Honey Ledger
// and HB SunRise each pass their own context type; it satisfies this interface
// structurally, so this module never imports an application.
type Context interface {
	context.Context
	GetContextId() string
	GetClientLocale() string
}
