// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

// transactionType is the kind of a flattened transaction. The values are not
// part of the public API: the extraction exposes the kinds as the
// ExtractedType* strings, and each application maps those onto its own
// transaction type when it imports them.
type transactionType byte

const (
	transactionTypeModifyBalance transactionType = iota
	transactionTypeIncome
	transactionTypeExpense
	transactionTypeTransfer
)
