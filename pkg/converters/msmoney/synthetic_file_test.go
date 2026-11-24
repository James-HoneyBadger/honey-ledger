// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/msmoney/synthetic"
)

func TestSyntheticMoneyFileExtractsEndToEnd(t *testing.T) {
	data := synthetic.MoneyFile()

	assert.True(t, IsMicrosoftMoneyDataFile(data))
	assert.False(t, IsMicrosoftMoneyBackupFile(data))

	extracted, err := ExtractMicrosoftMoneyData(newTestContext(), data)
	require.NoError(t, err)

	// Two USD accounts, sorted by name.
	require.Len(t, extracted.Accounts, 2)
	assert.Equal(t, ExtractedAccount{Name: "Checking", Currency: "USD"}, extracted.Accounts[0])
	assert.Equal(t, ExtractedAccount{Name: "Savings", Currency: "USD"}, extracted.Accounts[1])

	// Three transactions: two split parts and the transfer.
	require.Len(t, extracted.Transactions, 3)
	day := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)

	groceries := extracted.Transactions[0]
	assert.Equal(t, ExtractedTypeExpense, groceries.Type)
	assert.Equal(t, day, groceries.Time)
	assert.Equal(t, "Checking", groceries.Account)
	assert.Equal(t, "USD", groceries.Currency)
	assert.Equal(t, int64(3000), groceries.Amount)
	assert.Equal(t, "Food", groceries.Category, "the nested category keeps its parent")
	assert.Equal(t, "Groceries", groceries.SubCategory)
	assert.Equal(t, "Groceries", groceries.Memo)

	rent := extracted.Transactions[1]
	assert.Equal(t, ExtractedTypeExpense, rent.Type)
	assert.Equal(t, int64(2000), rent.Amount)
	assert.Empty(t, rent.Category, "a top level category has no parent")
	assert.Equal(t, "Rent", rent.SubCategory)

	transfer := extracted.Transactions[2]
	assert.Equal(t, ExtractedTypeTransfer, transfer.Type)
	assert.Equal(t, "Checking", transfer.Account)
	assert.Equal(t, int64(10000), transfer.Amount)
	assert.Equal(t, "Savings", transfer.TransferAccount)
	assert.Equal(t, "USD", transfer.TransferCurrency)
	assert.Equal(t, "Move to savings", transfer.Memo)

	// Skipped: the split total, the receiving side of the transfer, and the
	// investment row.
	assert.Equal(t, 3, extracted.Skipped)
}

func TestSyntheticBackupFileExtractsTheSameData(t *testing.T) {
	data := synthetic.BackupFile()

	assert.True(t, IsMicrosoftMoneyBackupFile(data))
	assert.False(t, IsMicrosoftMoneyDataFile(data))

	fromMoney, err := ExtractMicrosoftMoneyData(newTestContext(), synthetic.MoneyFile())
	require.NoError(t, err)

	fromBackup, err := ExtractMicrosoftMoneyData(newTestContext(), data)
	require.NoError(t, err)

	assert.Equal(t, fromMoney, fromBackup, "the .mbf wrapper must not change the extracted data")
}
