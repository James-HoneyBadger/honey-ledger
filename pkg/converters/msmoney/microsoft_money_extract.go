// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

// Package msmoney reads Microsoft Money data files (.mny) and backups (.mbf)
// into plain structs: accounts and transactions, with every split of a
// transaction flattened into a transaction of its own. The applications carry
// a synchronized copy of this package, so no application depends on this
// repository being present.
package msmoney

import (
	"fmt"
	"time"
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
	// TransferAmount is the amount the transfer account received, in its own
	// currency. Zero means the receiving side was not known and Amount applies.
	TransferAmount int64
	Category       string
	SubCategory    string
	Memo           string
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
func ExtractMicrosoftMoneyData(ctx Context, data []byte) (*ExtractedData, error) {
	if IsMicrosoftMoneyBackupFile(data) {
		database, err := extractMicrosoftMoneyDatabaseFromBackup(ctx, data)

		if err != nil {
			return nil, fmt.Errorf("%w: %v", ErrBackupFileNotReadable, err)
		}

		data = database
	}

	if !IsMicrosoftMoneyDataFile(data) {
		return nil, ErrFileTypeNotSupported
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
		case transactionTypeIncome:
			extracted.Type = ExtractedTypeIncome
		case transactionTypeTransfer:
			extracted.Type = ExtractedTypeTransfer
			extracted.Category, extracted.SubCategory = "", ""
		default:
			extracted.Type = ExtractedTypeExpense
		}

		if transaction.Related != nil {
			extracted.TransferAccount = transaction.Related.Name
			extracted.TransferCurrency = transaction.Related.Currency
			extracted.TransferAmount = transaction.RelatedAmount
		}

		result.Transactions = append(result.Transactions, extracted)
	}

	return result, nil
}
