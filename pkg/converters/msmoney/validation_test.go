// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validationSample() *ExtractedData {
	day := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)

	return &ExtractedData{
		Accounts: []ExtractedAccount{
			{Name: "Checking", Currency: "USD"},
			{Name: "Savings", Currency: "USD"},
			{Name: "Unused", Currency: "USD"},
		},
		Transactions: []ExtractedTransaction{
			{Time: day, Type: ExtractedTypeExpense, Account: "Checking", Currency: "USD", Amount: 1234, Category: "Food", SubCategory: "Groceries", Memo: "Milk"},
			{Time: day, Type: ExtractedTypeTransfer, Account: "Checking", Currency: "USD", Amount: 10000, TransferAccount: "Savings", TransferCurrency: "USD", TransferAmount: 10000},
		},
	}
}

func TestValidateTransactionsCleanData(t *testing.T) {
	report := ValidateTransactions(validationSample())

	require.Len(t, report.Findings, 1, "only the unused account is reported")
	assert.Equal(t, FindingAccountWithoutTransactions, report.Findings[0].Kind)
	assert.Equal(t, SeverityInfo, report.Findings[0].Severity)
	assert.False(t, report.IsClean())
	assert.Empty(t, report.Confirmed())
	assert.Empty(t, report.Uncertain())
}

func TestValidateTransactionsReportsDuplicates(t *testing.T) {
	data := validationSample()
	data.Transactions = append(data.Transactions, data.Transactions[0])

	report := ValidateTransactions(data)

	require.Len(t, report.Findings, 2)
	assert.Equal(t, FindingDuplicateTransaction, report.Findings[0].Kind)
	assert.Equal(t, SeverityConfirmed, report.Findings[0].Severity)
	require.Len(t, report.Confirmed(), 1)
}

func TestValidateTransactionsReportsTransferFallback(t *testing.T) {
	data := validationSample()
	data.Transactions[1].TransferAmount = 0

	report := ValidateTransactions(data)

	require.Len(t, report.Findings, 2)
	assert.Equal(t, FindingTransferAmountFallback, report.Findings[0].Kind)
	assert.Equal(t, SeverityUncertain, report.Findings[0].Severity, "an unmatched receiving side is an interpretation, not a fact")
	require.Len(t, report.Uncertain(), 1)
}

func TestValidateTransactionsReportsOrphanTransfer(t *testing.T) {
	data := validationSample()
	data.Transactions[1].TransferAccount = "Mystery"

	report := ValidateTransactions(data)

	// Orphan finding plus both now-inactive accounts (Savings no longer
	// receives the transfer).
	require.Len(t, report.Findings, 3)
	assert.Equal(t, FindingOrphanTransferAccount, report.Findings[0].Kind)
	assert.Equal(t, SeverityConfirmed, report.Findings[0].Severity)
}

func TestValidateTransactionsReportsSameCurrencyMismatch(t *testing.T) {
	data := validationSample()
	data.Transactions[1].TransferAmount = 9500 // differs from Amount 10000, same currency

	report := ValidateTransactions(data)

	require.Len(t, report.Findings, 2)
	assert.Equal(t, FindingTransferAmountMismatch, report.Findings[0].Kind)
	assert.Equal(t, SeverityUncertain, report.Findings[0].Severity)
}

func TestValidateTransactionsSplitTotalMismatch(t *testing.T) {
	day := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)

	data := &ExtractedData{
		Accounts: []ExtractedAccount{
			{Name: "Checking", Currency: "USD"},
		},
		Transactions: []ExtractedTransaction{
			{Time: day, Type: ExtractedTypeExpense, Account: "Checking", Currency: "USD", Amount: 3000, Split: true, splitParent: 100, Category: "Food", SubCategory: "Groceries"},
			{Time: day, Type: ExtractedTypeExpense, Account: "Checking", Currency: "USD", Amount: 2000, Split: true, splitParent: 100, Category: "Food", SubCategory: "Groceries"},
		},
		splitTotals: map[int64]int64{100: 7000},
	}

	report := ValidateTransactions(data)

	require.Len(t, report.Findings, 1)
	assert.Equal(t, FindingSplitTotalMismatch, report.Findings[0].Kind)
	assert.Equal(t, SeverityConfirmed, report.Findings[0].Severity)
	assert.Contains(t, report.Findings[0].Message, "50.00")
	assert.Contains(t, report.Findings[0].Message, "70.00")
}

func TestValidateTransactionsSplitTotalsAgree(t *testing.T) {
	day := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)

	data := &ExtractedData{
		Accounts: []ExtractedAccount{
			{Name: "Checking", Currency: "USD"},
		},
		Transactions: []ExtractedTransaction{
			{Time: day, Type: ExtractedTypeExpense, Account: "Checking", Currency: "USD", Amount: 3000, Split: true, splitParent: 100},
			{Time: day, Type: ExtractedTypeExpense, Account: "Checking", Currency: "USD", Amount: 2000, Split: true, splitParent: 100},
		},
		splitTotals: map[int64]int64{100: 5000},
	}

	assert.Empty(t, ValidateTransactions(data).Findings)
}

func TestValidateTransactionsReportsBrokenSplitReferences(t *testing.T) {
	data := validationSample()
	data.splitReferencesWithoutRows = 2

	report := ValidateTransactions(data)

	require.Len(t, report.Findings, 2)
	finding := report.Findings[1]
	assert.Equal(t, FindingSplitBrokenReference, finding.Kind)
	assert.Equal(t, SeverityConfirmed, finding.Severity)
}

func TestValidateTransactionsSummary(t *testing.T) {
	day := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)

	data := &ExtractedData{
		Accounts: []ExtractedAccount{
			{Name: "Checking", Currency: "USD"},
		},
		Transactions: []ExtractedTransaction{
			{Time: day, Type: ExtractedTypeIncome, Account: "Checking", Currency: "USD", Amount: 50000},
			{Time: day, Type: ExtractedTypeExpense, Account: "Checking", Currency: "USD", Amount: 1234},
			{Time: day.AddDate(0, 0, 1), Type: ExtractedTypeTransfer, Account: "Checking", Currency: "USD", Amount: 1000, TransferAccount: "Savings", TransferCurrency: "USD", TransferAmount: 1000},
		},
	}

	report := ValidateTransactions(data)

	summary := report.Summary
	assert.Equal(t, 3, summary.TransactionCount)
	assert.Equal(t, 1, summary.IncomeCount)
	assert.Equal(t, 1, summary.ExpenseCount)
	assert.Equal(t, 1, summary.TransferCount)
	assert.Equal(t, int64(50000), summary.IncomeTotal)
	assert.Equal(t, int64(1234), summary.ExpenseTotal)
	assert.Equal(t, day, summary.Earliest)
	assert.Equal(t, day.AddDate(0, 0, 1), summary.Latest)
}
