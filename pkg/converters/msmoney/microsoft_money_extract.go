// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import (
	"time"

	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
	"github.com/James-HoneyBadger/honey-ledger/pkg/models"
)

// Transaction kinds of an extracted transaction
const (
	ExtractedTypeIncome   = "Income"
	ExtractedTypeExpense  = "Expense"
	ExtractedTypeTransfer = "Transfer"
)

// ExtractedAccount is an account of a Microsoft Money file
type ExtractedAccount struct {
	Name     string
	Currency string
}

// ExtractedTransaction is one transaction of a Microsoft Money file. The amount
// is in hundredths of the currency and is always positive; Type tells whether
// the money came in, went out or moved to TransferAccount.
type ExtractedTransaction struct {
	Time             time.Time
	Type             string
	Account          string
	Currency         string
	Amount           int64
	TransferAccount  string
	TransferCurrency string
	Category         string
	SubCategory      string
	Memo             string
}

// ExtractedData is everything of a Microsoft Money file that can be exported
type ExtractedData struct {
	Accounts     []ExtractedAccount
	Transactions []ExtractedTransaction
	// Skipped counts the rows that cannot be exported: investment activity,
	// split totals and rows without an amount, account or date.
	Skipped int
}

// ExtractMicrosoftMoneyData reads a Microsoft Money data file (.mny) or backup
// (.mbf) and returns its accounts and transactions without needing a ledger.
func ExtractMicrosoftMoneyData(ctx core.Context, data []byte) (*ExtractedData, error) {
	if IsMicrosoftMoneyBackupFile(data) {
		database, err := extractMicrosoftMoneyDatabaseFromBackup(ctx, data)

		if err != nil {
			return nil, errs.Or(err, errs.ErrImportMicrosoftMoneyBackupFileNotReadable)
		}

		data = database
	}

	if !IsMicrosoftMoneyDataFile(data) {
		return nil, errs.ErrImportFileTypeNotSupported
	}

	moneyData, err := readMicrosoftMoneyData(ctx, data)

	if err != nil {
		return nil, err
	}

	result := &ExtractedData{
		Accounts:     make([]ExtractedAccount, 0, len(moneyData.Accounts)),
		Transactions: make([]ExtractedTransaction, 0, len(moneyData.Transactions)),
		Skipped:      moneyData.Skipped,
	}

	for _, account := range moneyData.Accounts {
		result.Accounts = append(result.Accounts, ExtractedAccount{Name: account.Name, Currency: account.Currency})
	}

	for _, transaction := range moneyData.Transactions {
		extracted := ExtractedTransaction{
			Time:        transaction.Time,
			Account:     transaction.Account.Name,
			Currency:    transaction.Account.Currency,
			Amount:      transaction.Amount,
			Category:    transaction.Category,
			SubCategory: transaction.SubCategory,
			Memo:        transaction.Description,
		}

		switch transaction.Type {
		case models.TRANSACTION_TYPE_INCOME:
			extracted.Type = ExtractedTypeIncome
		case models.TRANSACTION_TYPE_TRANSFER:
			extracted.Type = ExtractedTypeTransfer
			extracted.Category, extracted.SubCategory = "", ""
		default:
			extracted.Type = ExtractedTypeExpense
		}

		if transaction.Related != nil {
			extracted.TransferAccount = transaction.Related.Name
			extracted.TransferCurrency = transaction.Related.Currency
		}

		result.Transactions = append(result.Transactions, extracted)
	}

	return result, nil
}
