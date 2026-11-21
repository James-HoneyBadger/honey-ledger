// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package custom

import "github.com/James-HoneyBadger/honey-ledger/pkg/core"

// CustomTransactionDataParser represents the parser for custom transaction data files
type CustomTransactionDataParser interface {
	ParseDataLines(ctx core.Context, data []byte) ([][]string, error)
}
