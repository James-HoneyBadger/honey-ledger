// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import (
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/msmoney/msisam"
	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
	"github.com/James-HoneyBadger/honey-ledger/pkg/log"
	"github.com/James-HoneyBadger/honey-ledger/pkg/models"
	"github.com/James-HoneyBadger/honey-ledger/pkg/validators"
)

const (
	// microsoftMoneyAmountScale is the number of decimal places a Microsoft
	// Money amount is stored with, while an import amount has two.
	microsoftMoneyAmountScale = 10000

	// microsoftMoneyNoHandle is the handle Microsoft Money stores when a field
	// refers to no object.
	microsoftMoneyNoHandle = 0

	// microsoftMoneyEarliestYear and microsoftMoneyLatestYear bound the dates a
	// transaction can hold. Money stores an empty date as a value far outside
	// this range.
	microsoftMoneyEarliestYear = 1900
	microsoftMoneyLatestYear   = 2200

	// microsoftMoneyByteOrderMark is the character Money leaves in front of the
	// text of some memos.
	microsoftMoneyByteOrderMark = "\u00ff\u00fe"
)

// microsoftMoneyAccount is an account as a Microsoft Money data file stores it.
type microsoftMoneyAccount struct {
	Name     string
	Currency string
}

// microsoftMoneyCategory is a category as a Microsoft Money data file stores it.
type microsoftMoneyCategory struct {
	Name   string
	Parent string
}

// microsoftMoneyTransaction is one transaction of a Microsoft Money data file,
// flattened so that every split of a transaction becomes a transaction of its
// own.
type microsoftMoneyTransaction struct {
	Time        time.Time
	Type        models.TransactionType
	Account     *microsoftMoneyAccount
	Related     *microsoftMoneyAccount
	Amount      int64
	Category    string
	SubCategory string
	Description string
}

// microsoftMoneyTransactionRow is the transaction row a data file stores, before
// its handles are resolved.
type microsoftMoneyTransactionRow struct {
	Handle   int64
	Account  int64
	Link     int64
	Category int64
	Security int64
	Amount   int64
	Time     time.Time
	Memo     string
}

// microsoftMoneyData is the content of a Microsoft Money data file that the
// importer can use.
type microsoftMoneyData struct {
	// Accounts holds every account of the file, including those without
	// transactions.
	Accounts     []*microsoftMoneyAccount
	Transactions []*microsoftMoneyTransaction
	// Skipped counts the rows the file holds that cannot become a transaction,
	// such as investment activity, which has no counterpart in the ledger.
	Skipped int
}

// readMicrosoftMoneyData reads the accounts, categories and transactions a
// Microsoft Money data file holds.
func readMicrosoftMoneyData(ctx core.Context, data []byte) (*microsoftMoneyData, error) {
	database, err := msisam.Open(data)

	if err != nil {
		unreadable := errs.ErrImportMicrosoftMoneyDataFileNotReadable

		if errors.Is(err, msisam.ErrEncryptedDatabase) {
			unreadable = errs.ErrImportMicrosoftMoneyDataFileEncrypted
		}

		log.Errorf(ctx, "[microsoft_money_data_reader.readMicrosoftMoneyData] cannot open the Microsoft Money data file, because %s", err.Error())
		return nil, errs.Or(err, unreadable)
	}

	currencies, err := readMicrosoftMoneyCurrencies(database)

	if err != nil {
		return nil, err
	}

	accounts, err := readMicrosoftMoneyAccounts(database, currencies)

	if err != nil {
		return nil, err
	}

	categories, err := readMicrosoftMoneyCategories(database)

	if err != nil {
		return nil, err
	}

	rows, err := readMicrosoftMoneyTransactionRows(database)

	if err != nil {
		return nil, err
	}

	parents, err := readMicrosoftMoneySplitParents(database)

	if err != nil {
		return nil, err
	}

	transactions, skipped := flattenMicrosoftMoneyTransactions(rows, parents, accounts, categories)

	allAccounts := make([]*microsoftMoneyAccount, 0, len(accounts))

	for _, account := range accounts {
		allAccounts = append(allAccounts, account)
	}

	sort.Slice(allAccounts, func(i, j int) bool { return allAccounts[i].Name < allAccounts[j].Name })

	return &microsoftMoneyData{Accounts: allAccounts, Transactions: transactions, Skipped: skipped}, nil
}

// openMicrosoftMoneyTable returns the named table of the data file, reporting a
// file that does not hold it as unreadable.
func openMicrosoftMoneyTable(ctx core.Context, database *msisam.Database, name string) (*msisam.Table, error) {
	table, err := database.Table(name)

	if err != nil {
		log.Errorf(ctx, "[microsoft_money_data_reader.openMicrosoftMoneyTable] the Microsoft Money data file has no \"%s\" table, because %s", name, err.Error())
		return nil, errs.Or(err, errs.ErrImportMicrosoftMoneyDataFileNotReadable)
	}

	return table, nil
}

// readMicrosoftMoneyCurrencies returns the ISO code of every currency, keyed by
// the handle the data file gives it. A currency the ledger does not know, or one
// without an ISO code, maps to an empty string.
func readMicrosoftMoneyCurrencies(database *msisam.Database) (map[int64]string, error) {
	table, err := database.Table("CRNC")

	if err != nil {
		// An older file may name its currencies differently; amounts then keep
		// the currency the user imports with.
		return map[int64]string{}, nil
	}

	currencies := make(map[int64]string, table.Definition.RowCount)

	err = database.EachRow(table.Definition, func(fields []msisam.Field) error {
		var handle int64
		code := ""

		for _, field := range fields {
			switch field.Column.Name {
			case "hcrnc":
				handle, _ = database.Int(field)
			case "szIsoCode":
				text, err := database.String(field)

				if err != nil {
					return nil
				}

				code = strings.ToUpper(strings.TrimSpace(text))
			}
		}

		if handle != microsoftMoneyNoHandle && validators.AllCurrencyNames[code] {
			currencies[handle] = code
		}

		return nil
	})

	if err != nil {
		return nil, errs.Or(err, errs.ErrImportMicrosoftMoneyDataFileNotReadable)
	}

	return currencies, nil
}

// readMicrosoftMoneyAccounts returns the accounts of the data file, keyed by
// their handle.
func readMicrosoftMoneyAccounts(database *msisam.Database, currencies map[int64]string) (map[int64]*microsoftMoneyAccount, error) {
	table, err := database.Table("ACCT")

	if err != nil {
		return nil, errs.Or(err, errs.ErrImportMicrosoftMoneyDataFileNotReadable)
	}

	accounts := make(map[int64]*microsoftMoneyAccount, table.Definition.RowCount)

	err = database.EachRow(table.Definition, func(fields []msisam.Field) error {
		var handle, currency int64
		name := ""

		for _, field := range fields {
			switch field.Column.Name {
			case "hacct":
				handle, _ = database.Int(field)
			case "hcrnc":
				currency, _ = database.Int(field)
			case "szFull":
				text, err := database.String(field)

				if err != nil {
					return nil
				}

				name = strings.TrimSpace(text)
			}
		}

		if handle == microsoftMoneyNoHandle || name == "" {
			return nil
		}

		accounts[handle] = &microsoftMoneyAccount{Name: name, Currency: currencies[currency]}

		return nil
	})

	if err != nil {
		return nil, errs.Or(err, errs.ErrImportMicrosoftMoneyDataFileNotReadable)
	}

	return accounts, nil
}

// readMicrosoftMoneyCategories returns the categories of the data file, keyed by
// their handle, with the grouping the ledger files them under.
func readMicrosoftMoneyCategories(database *msisam.Database) (map[int64]*microsoftMoneyCategory, error) {
	table, err := database.Table("CAT")

	if err != nil {
		return nil, errs.Or(err, errs.ErrImportMicrosoftMoneyDataFileNotReadable)
	}

	names := make(map[int64]string, table.Definition.RowCount)
	parentHandles := make(map[int64]int64, table.Definition.RowCount)

	err = database.EachRow(table.Definition, func(fields []msisam.Field) error {
		var handle, parent int64
		name := ""

		for _, field := range fields {
			switch field.Column.Name {
			case "hcat":
				handle, _ = database.Int(field)
			case "hcatParent":
				parent, _ = database.Int(field)
			case "szFull":
				text, err := database.String(field)

				if err != nil {
					return nil
				}

				name = strings.TrimSpace(text)
			}
		}

		if handle == microsoftMoneyNoHandle || name == "" {
			return nil
		}

		names[handle] = name
		parentHandles[handle] = parent

		return nil
	})

	if err != nil {
		return nil, errs.Or(err, errs.ErrImportMicrosoftMoneyDataFileNotReadable)
	}

	categories := make(map[int64]*microsoftMoneyCategory, len(names))

	for handle, name := range names {
		// Money keeps an income or expense root above every category. That root
		// is a direction rather than a category, so a transaction filed directly
		// against it stays uncategorized, and the categories beneath it are not
		// grouped under it.
		if parentHandles[handle] == microsoftMoneyNoHandle {
			continue
		}

		category := &microsoftMoneyCategory{Name: name}

		if parent, hasParent := names[parentHandles[handle]]; hasParent && parentHandles[parentHandles[handle]] != microsoftMoneyNoHandle {
			category.Parent = parent
		}

		categories[handle] = category
	}

	return categories, nil
}

// readMicrosoftMoneyTransactionRows returns every transaction row of the data
// file, in storage order.
func readMicrosoftMoneyTransactionRows(database *msisam.Database) ([]*microsoftMoneyTransactionRow, error) {
	table, err := database.Table("TRN")

	if err != nil {
		return nil, errs.Or(err, errs.ErrImportMicrosoftMoneyDataFileNotReadable)
	}

	rows := make([]*microsoftMoneyTransactionRow, 0, table.Definition.RowCount)

	err = database.EachRow(table.Definition, func(fields []msisam.Field) error {
		row := &microsoftMoneyTransactionRow{}

		for _, field := range fields {
			switch field.Column.Name {
			case "htrn":
				row.Handle, _ = database.Int(field)
			case "hacct":
				row.Account, _ = database.Int(field)
			case "hacctLink":
				row.Link, _ = database.Int(field)
			case "hcat":
				row.Category, _ = database.Int(field)
			case "hsec":
				row.Security, _ = database.Int(field)
			case "amt":
				row.Amount, _ = database.Int(field)
			case "mMemo":
				text, err := database.String(field)

				if err == nil {
					row.Memo = trimMicrosoftMoneyText(text)
				}
			case "dt":
				if when, ok := database.Time(field); ok && when.Year() >= microsoftMoneyEarliestYear && when.Year() <= microsoftMoneyLatestYear {
					row.Time = when
				}
			}
		}

		if row.Handle != microsoftMoneyNoHandle {
			rows = append(rows, row)
		}

		return nil
	})

	if err != nil {
		return nil, errs.Or(err, errs.ErrImportMicrosoftMoneyDataFileNotReadable)
	}

	return rows, nil
}

// readMicrosoftMoneySplitParents returns the handles of the transactions that
// hold splits. Money stores a transaction with splits as a parent row that holds
// the total and one row per split, and only the splits belong in the ledger.
func readMicrosoftMoneySplitParents(database *msisam.Database) (map[int64]bool, error) {
	parents := make(map[int64]bool)
	table, err := database.Table("TRN_SPLIT")

	if err != nil {
		// A file without splits has no table to read.
		return parents, nil
	}

	err = database.EachRow(table.Definition, func(fields []msisam.Field) error {
		for _, field := range fields {
			if field.Column.Name != "htrnParent" {
				continue
			}

			if parent, ok := database.Int(field); ok && parent != microsoftMoneyNoHandle {
				parents[parent] = true
			}
		}

		return nil
	})

	if err != nil {
		return nil, errs.Or(err, errs.ErrImportMicrosoftMoneyDataFileNotReadable)
	}

	return parents, nil
}

// flattenMicrosoftMoneyTransactions turns the stored rows into transactions,
// reporting the count of rows that cannot become one. A transfer is stored twice,
// once for each account, so only the row that takes the money out becomes a
// transaction, with the account that receives it as the related account.
func flattenMicrosoftMoneyTransactions(rows []*microsoftMoneyTransactionRow, parents map[int64]bool, accounts map[int64]*microsoftMoneyAccount, categories map[int64]*microsoftMoneyCategory) ([]*microsoftMoneyTransaction, int) {
	transactions := make([]*microsoftMoneyTransaction, 0, len(rows))
	skipped := 0

	for _, row := range rows {
		// The parent of a split only holds the total of the splits.
		// Investment activity has no counterpart in the ledger, and a
		// transaction without an amount moves no money.
		if parents[row.Handle] || row.Security != microsoftMoneyNoHandle || row.Amount == 0 {
			skipped++

			continue
		}

		account, exists := accounts[row.Account]

		if !exists || row.Time.IsZero() {
			skipped++

			continue
		}

		transaction := &microsoftMoneyTransaction{
			Time:        row.Time,
			Account:     account,
			Description: row.Memo,
		}

		if row.Link != microsoftMoneyNoHandle {
			// Money stores both sides of a transfer, so the side that receives
			// the money would repeat the transaction.
			if row.Amount > 0 {
				skipped++

				continue
			}

			related, exists := accounts[row.Link]

			if !exists {
				skipped++

				continue
			}

			transaction.Type = models.TRANSACTION_TYPE_TRANSFER
			transaction.Related = related
			transaction.Amount = convertMicrosoftMoneyAmount(-row.Amount)
		} else {
			if row.Amount > 0 {
				transaction.Type = models.TRANSACTION_TYPE_INCOME
				transaction.Amount = convertMicrosoftMoneyAmount(row.Amount)
			} else {
				transaction.Type = models.TRANSACTION_TYPE_EXPENSE
				transaction.Amount = convertMicrosoftMoneyAmount(-row.Amount)
			}

			if category, exists := categories[row.Category]; exists {
				transaction.Category = category.Parent
				transaction.SubCategory = category.Name
			}
		}

		transactions = append(transactions, transaction)
	}

	return transactions, skipped
}

// trimMicrosoftMoneyText removes the byte order mark Money leaves in front of
// the text of some memos.
func trimMicrosoftMoneyText(text string) string {
	return strings.TrimSpace(strings.TrimPrefix(text, microsoftMoneyByteOrderMark))
}

// convertMicrosoftMoneyAmount returns the amount in the two decimal places an
// import uses, rounded away from zero.
func convertMicrosoftMoneyAmount(amount int64) int64 {
	divisor := int64(microsoftMoneyAmountScale / 100)

	if amount < 0 {
		return -((-amount + divisor/2) / divisor)
	}

	return (amount + divisor/2) / divisor
}
