// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/datatable"
)

func TestBuildMicrosoftMoneyTransactionRowItemsUsesMemoAsDescription(t *testing.T) {
	transaction := &ExtractedTransaction{
		Time:    time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
		Type:    ExtractedTypeExpense,
		Account: "Checking",
		Amount:  1234,
		Memo:    "Weekly shop",
		Payee:   "Groceria Market",
	}

	items := buildMicrosoftMoneyTransactionRowItems(transaction)

	assert.Equal(t, "Weekly shop", items[datatable.TRANSACTION_DATA_TABLE_DESCRIPTION], "the memo wins over the payee")
}

func TestBuildMicrosoftMoneyTransactionRowItemsFallsBackToPayee(t *testing.T) {
	transaction := &ExtractedTransaction{
		Time:    time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
		Type:    ExtractedTypeExpense,
		Account: "Checking",
		Amount:  1234,
		Payee:   "Groceria Market",
	}

	items := buildMicrosoftMoneyTransactionRowItems(transaction)

	assert.Equal(t, "Groceria Market", items[datatable.TRANSACTION_DATA_TABLE_DESCRIPTION], "a transaction without a memo keeps the payee as its description")
}

func TestBuildMicrosoftMoneyTransactionRowItemsWithoutDescription(t *testing.T) {
	transaction := &ExtractedTransaction{
		Time:    time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC),
		Type:    ExtractedTypeExpense,
		Account: "Checking",
		Amount:  1234,
	}

	items := buildMicrosoftMoneyTransactionRowItems(transaction)

	assert.Empty(t, items[datatable.TRANSACTION_DATA_TABLE_DESCRIPTION])
}
