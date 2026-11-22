// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package services

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/James-HoneyBadger/honey-ledger/pkg/models"
)

func analysisTestTransaction(id int64, txnType models.TransactionDbType, amount int64, accountId int64, categoryId int64, transactionTime int64) *models.Transaction {
	return &models.Transaction{
		TransactionId:   id,
		Type:            txnType,
		Amount:          amount,
		AccountId:       accountId,
		CategoryId:      categoryId,
		TransactionTime: transactionTime,
	}
}

func TestAssembleFinancialAnalysis(t *testing.T) {
	startTime := time.Unix(1700000000, 0)
	endTime := startTime.AddDate(0, 1, 0)
	day := int64(24 * 60 * 60 * 1000)

	source := financialAnalysisDataSource{
		startTime: startTime,
		endTime:   endTime,
		accounts: []*models.Account{
			{AccountId: 1, Name: "Chequing", Category: models.ACCOUNT_CATEGORY_CHECKING_ACCOUNT, Currency: "USD", Balance: 10000},
			{AccountId: 2, Name: "Savings", Category: models.ACCOUNT_CATEGORY_SAVINGS_ACCOUNT, Currency: "USD", Balance: 5000},
		},
		categories: []*models.TransactionCategory{
			{CategoryId: 10, Name: "Groceries"},
			{CategoryId: 20, Name: "Rent"},
		},
		transactions: []*models.Transaction{
			analysisTestTransaction(1, models.TRANSACTION_DB_TYPE_INCOME, 500000, 1, 0, startTime.Unix()*1000),
			// Monthly rent: three occurrences about a month apart.
			analysisTestTransaction(2, models.TRANSACTION_DB_TYPE_EXPENSE, 120000, 2, 20, startTime.Unix()*1000),
			analysisTestTransaction(3, models.TRANSACTION_DB_TYPE_EXPENSE, 120000, 2, 20, startTime.Unix()*1000+30*day),
			analysisTestTransaction(4, models.TRANSACTION_DB_TYPE_EXPENSE, 120000, 2, 20, startTime.Unix()*1000+60*day),
			// Ordinary grocery spending.
			analysisTestTransaction(5, models.TRANSACTION_DB_TYPE_EXPENSE, 4000, 1, 10, startTime.Unix()*1000+day),
			analysisTestTransaction(6, models.TRANSACTION_DB_TYPE_EXPENSE, 4500, 1, 10, startTime.Unix()*1000+2*day),
			analysisTestTransaction(7, models.TRANSACTION_DB_TYPE_EXPENSE, 4200, 1, 10, startTime.Unix()*1000+3*day),
			analysisTestTransaction(8, models.TRANSACTION_DB_TYPE_EXPENSE, 4300, 1, 10, startTime.Unix()*1000+4*day),
			analysisTestTransaction(9, models.TRANSACTION_DB_TYPE_EXPENSE, 4100, 1, 10, startTime.Unix()*1000+5*day),
			// A one-off expense far above the account median.
			analysisTestTransaction(10, models.TRANSACTION_DB_TYPE_EXPENSE, 90000, 1, 20, startTime.Unix()*1000+6*day),
		},
	}

	response := assembleFinancialAnalysis(source)

	assert.Equal(t, int64(10), response.TransactionCount)
	assert.Len(t, response.Totals, 1)
	assert.Equal(t, "USD", response.Totals[0].Currency)
	assert.Equal(t, int64(500000), response.Totals[0].Income)
	assert.Equal(t, int64(471100), response.Totals[0].Expense)
	assert.Equal(t, int64(28900), response.Totals[0].Net)

	assert.Len(t, response.RecurringPayments, 1)
	assert.Equal(t, "Rent", response.RecurringPayments[0].Category)
	assert.Equal(t, int64(120000), response.RecurringPayments[0].Amount)
	assert.Equal(t, int64(3), response.RecurringPayments[0].Count)

	assert.Len(t, response.UnusualExpenses, 1)
	assert.Equal(t, int64(90000), response.UnusualExpenses[0].Amount)

	assert.Equal(t, int64(120000), response.LargestExpenses[0].Amount)
	assert.Len(t, response.Accounts, 2)
	assert.Equal(t, "Chequing", response.Accounts[0].Name)
}

func TestAssembleFinancialAnalysis_RecurringRequiresMonthlyCadence(t *testing.T) {
	startTime := time.Unix(1700000000, 0)
	day := int64(24 * 60 * 60 * 1000)

	source := financialAnalysisDataSource{
		startTime: startTime,
		endTime:   startTime.AddDate(0, 1, 0),
		accounts: []*models.Account{
			{AccountId: 1, Name: "Chequing", Category: models.ACCOUNT_CATEGORY_CHECKING_ACCOUNT, Currency: "USD"},
		},
		categories: []*models.TransactionCategory{
			{CategoryId: 20, Name: "Rent"},
		},
		transactions: []*models.Transaction{
			analysisTestTransaction(1, models.TRANSACTION_DB_TYPE_EXPENSE, 120000, 1, 20, startTime.Unix()*1000),
			analysisTestTransaction(2, models.TRANSACTION_DB_TYPE_EXPENSE, 120000, 1, 20, startTime.Unix()*1000+2*day),
			analysisTestTransaction(3, models.TRANSACTION_DB_TYPE_EXPENSE, 120000, 1, 20, startTime.Unix()*1000+4*day),
		},
	}

	response := assembleFinancialAnalysis(source)

	assert.Empty(t, response.RecurringPayments)
}
