// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package _default

// defaultTransactionDataCSVFileConverter defines the structure of Honey Ledger default csv file converter
type defaultTransactionDataCSVFileConverter struct {
	defaultTransactionDataPlainTextConverter
}

// Initialize a Honey Ledger default transaction data csv file converter singleton instance
var (
	DefaultTransactionDataCSVFileConverter = &defaultTransactionDataCSVFileConverter{
		defaultTransactionDataPlainTextConverter{
			columnSeparator: ",",
		},
	}
)
