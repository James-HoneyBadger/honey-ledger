// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/msmoney/msisam"
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
	// Closed and Hidden mirror the flags the file stores on the account row.
	Closed bool
	Hidden bool
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
	Time    time.Time
	Type    transactionType
	Account *microsoftMoneyAccount
	Related *microsoftMoneyAccount
	Amount  int64
	// Split marks a transaction that came from one fragment of a split
	// transaction, so its provenance is never mistaken for a standalone
	// entry.
	Split bool
	// SplitParent is the handle of the transaction row that holds the total
	// of the split this transaction is a fragment of.
	SplitParent int64
	// RelatedAmount is the amount the related account received, in the related
	// account's currency. Money stores both sides of a transfer; when the two
	// currencies differ, the receiving amount is not equal to Amount. Zero
	// means the receiving side could not be matched and Amount is used.
	RelatedAmount int64
	Category      string
	SubCategory   string
	Description   string
	Payee         string
}

// microsoftMoneyTransactionRow is the transaction row a data file stores, before
// its handles are resolved.
type microsoftMoneyTransactionRow struct {
	Handle   int64
	Account  int64
	Link     int64
	Category int64
	Security int64
	Payee    int64
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
	// SplitTotals holds the stored total of every split parent row, keyed by
	// the parent handle, in hundredths. Validation compares the fragments'
	// sum against it.
	SplitTotals map[int64]int64
	// SplitReferencesWithoutRows counts the split links whose referenced row
	// does not exist, so validation can report the file's broken references.
	SplitReferencesWithoutRows int
	// Diagnostics reports the decoding problems found while reading the file.
	Diagnostics ExtractedDiagnostics
}

// readMicrosoftMoneyData reads the accounts, categories and transactions a
// Microsoft Money data file holds, reporting progress through the options and
// decoding problems into the returned data's diagnostics.
func readMicrosoftMoneyData(ctx Context, data []byte, options ExtractionOptions) (*microsoftMoneyData, error) {
	reportProgress(options, "opening", 0, 0)

	database, err := msisam.Open(data)

	if err != nil {
		unreadable := ErrDataFileNotReadable

		if errors.Is(err, msisam.ErrEncryptedDatabase) {
			unreadable = ErrDataFileEncrypted
		} else if errors.Is(err, msisam.ErrUnsupportedVersion) {
			unreadable = ErrDataFileUnsupportedVersion
		}

		errorf(ctx, "[microsoft_money_data_reader.readMicrosoftMoneyData] cannot open the Microsoft Money data file, because %s", err.Error())
		return nil, fmt.Errorf("%w: %v", unreadable, err)
	}

	// Decoding problems are recorded so a damaged file never loses rows
	// silently; the report travels with the extracted data.
	diagnostics := &msisam.Diagnostics{}
	database.SetDiagnostics(diagnostics)

	reportProgress(options, "reading tables", 0, 0)

	currencies, err := readMicrosoftMoneyCurrencies(ctx, database)

	if err != nil {
		return nil, err
	}

	accounts, err := readMicrosoftMoneyAccounts(ctx, database, currencies)

	if err != nil {
		return nil, err
	}

	categories, err := readMicrosoftMoneyCategories(ctx, database)

	if err != nil {
		return nil, err
	}

	rows, err := readMicrosoftMoneyTransactionRows(ctx, database, options)

	if err != nil {
		return nil, err
	}

	payees, err := readMicrosoftMoneyPayees(ctx, database)

	if err != nil {
		return nil, err
	}

	parents, children, err := readMicrosoftMoneySplitHandles(ctx, database)

	if err != nil {
		return nil, err
	}

	splitTotals, missingReferences := collectMicrosoftMoneySplitIntegrity(rows, parents, children)

	reportProgress(options, "flattening", 0, 0)

	transactions, skipped := flattenMicrosoftMoneyTransactions(rows, parents, children, accounts, categories, payees)

	allAccounts := make([]*microsoftMoneyAccount, 0, len(accounts))

	for _, account := range accounts {
		allAccounts = append(allAccounts, account)
	}

	sort.Slice(allAccounts, func(i, j int) bool { return allAccounts[i].Name < allAccounts[j].Name })

	return &microsoftMoneyData{
		Accounts:                   allAccounts,
		Transactions:               transactions,
		Skipped:                    skipped,
		SplitTotals:                splitTotals,
		SplitReferencesWithoutRows: missingReferences,
		Diagnostics:                extractedDiagnostics(diagnostics),
	}, nil
}

// reportProgress calls the options' progress callback when one is set.
func reportProgress(options ExtractionOptions, phase string, done, total int) {
	if options.Progress != nil {
		options.Progress(Progress{Phase: phase, Done: done, Total: total})
	}
}

// extractedDiagnostics converts the engine's decoding diagnostics into the
// public report that travels with the extracted data.
func extractedDiagnostics(diagnostics *msisam.Diagnostics) ExtractedDiagnostics {
	if diagnostics == nil || diagnostics.IsEmpty() {
		return ExtractedDiagnostics{}
	}

	result := ExtractedDiagnostics{
		CorruptRows:           diagnostics.CorruptRows,
		UnresolvedRows:        diagnostics.UnresolvedRows,
		CorruptDefinitions:    diagnostics.CorruptDefinitions,
		UnknownCatalogEntries: diagnostics.UnknownCatalogEntries,
		Samples:               make([]string, 0, len(diagnostics.RowSamples)+len(diagnostics.DefinitionSamples)),
	}

	for _, sample := range diagnostics.RowSamples {
		result.Samples = append(result.Samples, fmt.Sprintf("row (definition page %d, page %d, slot %d): %s", sample.Definition, sample.Page, sample.Slot, sample.Err))
	}

	for _, sample := range diagnostics.DefinitionSamples {
		result.Samples = append(result.Samples, fmt.Sprintf("definition (page %d): %s", sample.Page, sample.Err))
	}

	return result
}

// openMicrosoftMoneyTable returns the named table of the data file, reporting a
// file that does not hold it as unreadable.
func openMicrosoftMoneyTable(ctx Context, database *msisam.Database, name string) (*msisam.Table, error) {
	table, err := database.Table(name)

	if err != nil {
		errorf(ctx, "[microsoft_money_data_reader.openMicrosoftMoneyTable] the Microsoft Money data file has no \"%s\" table, because %s", name, err.Error())
		return nil, fmt.Errorf("%w: %v", ErrDataFileNotReadable, err)
	}

	return table, nil
}

// readMicrosoftMoneyCurrencies returns the ISO code of every currency, keyed by
// the handle the data file gives it. A currency the ledger does not know, or one
// without an ISO code, maps to an empty string.
func readMicrosoftMoneyCurrencies(ctx Context, database *msisam.Database) (map[int64]string, error) {
	table, err := database.Table("CRNC")

	if err != nil {
		// An older file may name its currencies differently; amounts then keep
		// the currency the user imports with.
		return map[int64]string{}, nil
	}

	currencies := make(map[int64]string, table.Definition.RowCount)

	err = database.EachRow(ctx, table.Definition, func(fields []msisam.Field) error {
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

		if handle != microsoftMoneyNoHandle && allCurrencyNames[code] {
			currencies[handle] = code
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDataFileNotReadable, err)
	}

	return currencies, nil
}

// readMicrosoftMoneyAccounts returns the accounts of the data file, keyed by
// their handle.
func readMicrosoftMoneyAccounts(ctx Context, database *msisam.Database, currencies map[int64]string) (map[int64]*microsoftMoneyAccount, error) {
	table, err := database.Table("ACCT")

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDataFileNotReadable, err)
	}

	accounts := make(map[int64]*microsoftMoneyAccount, table.Definition.RowCount)

	err = database.EachRow(ctx, table.Definition, func(fields []msisam.Field) error {
		var handle, currency int64
		name := ""
		closed, hidden := false, false

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
			case "fClosed":
				closed = database.Bool(field)
			case "fHidden":
				hidden = database.Bool(field)
			}
		}

		if handle == microsoftMoneyNoHandle || name == "" {
			return nil
		}

		accounts[handle] = &microsoftMoneyAccount{Name: name, Currency: currencies[currency], Closed: closed, Hidden: hidden}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDataFileNotReadable, err)
	}

	return accounts, nil
}

// readMicrosoftMoneyCategories returns the categories of the data file, keyed by
// their handle, with the grouping the ledger files them under.
func readMicrosoftMoneyCategories(ctx Context, database *msisam.Database) (map[int64]*microsoftMoneyCategory, error) {
	table, err := database.Table("CAT")

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDataFileNotReadable, err)
	}

	names := make(map[int64]string, table.Definition.RowCount)
	parentHandles := make(map[int64]int64, table.Definition.RowCount)

	err = database.EachRow(ctx, table.Definition, func(fields []msisam.Field) error {
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
		return nil, fmt.Errorf("%w: %v", ErrDataFileNotReadable, err)
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

// readMicrosoftMoneyPayees returns the name of every payee, keyed by the handle
// the data file gives it. A file without a payee table leaves every payee
// unnamed rather than failing the whole file.
func readMicrosoftMoneyPayees(ctx Context, database *msisam.Database) (map[int64]string, error) {
	table, err := database.Table("PAY")

	if err != nil {
		return map[int64]string{}, nil
	}

	payees := make(map[int64]string, table.Definition.RowCount)

	err = database.EachRow(ctx, table.Definition, func(fields []msisam.Field) error {
		var handle int64
		name := ""

		for _, field := range fields {
			switch field.Column.Name {
			case "hpay":
				handle, _ = database.Int(field)
			case "szName":
				text, err := database.String(field)

				if err != nil {
					return nil
				}

				name = strings.TrimSpace(text)
			}
		}

		if handle != microsoftMoneyNoHandle && name != "" {
			payees[handle] = name
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDataFileNotReadable, err)
	}

	return payees, nil
}

// readMicrosoftMoneyTransactionRows returns every transaction row of the data
// file, in storage order.
func readMicrosoftMoneyTransactionRows(ctx Context, database *msisam.Database, options ExtractionOptions) ([]*microsoftMoneyTransactionRow, error) {
	table, err := database.Table("TRN")

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDataFileNotReadable, err)
	}

	rows := make([]*microsoftMoneyTransactionRow, 0, table.Definition.RowCount)
	decoded := 0

	err = database.EachRow(ctx, table.Definition, func(fields []msisam.Field) error {
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
			case "hpay":
				row.Payee, _ = database.Int(field)
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

		decoded++
		reportProgress(options, "transactions", decoded, table.Definition.RowCount)

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDataFileNotReadable, err)
	}

	return rows, nil
}

// readMicrosoftMoneySplitHandles returns the handles of the transactions that
// hold splits. Money stores a transaction with splits as a parent row that
// holds the total and one row per split, and only the splits belong in the
// ledger. Parents maps a parent handle; children maps a split row handle to its
// parent handle.
func readMicrosoftMoneySplitHandles(ctx Context, database *msisam.Database) (parents map[int64]bool, children map[int64]int64, err error) {
	parents = make(map[int64]bool)
	children = make(map[int64]int64)

	table, err := database.Table("TRN_SPLIT")

	if err != nil {
		// A file without splits has no table to read.
		return parents, children, nil
	}

	err = database.EachRow(ctx, table.Definition, func(fields []msisam.Field) error {
		var parent, child int64

		for _, field := range fields {
			switch field.Column.Name {
			case "htrnParent":
				if value, ok := database.Int(field); ok && value != microsoftMoneyNoHandle {
					parent = value
				}
			case "htrn":
				if value, ok := database.Int(field); ok && value != microsoftMoneyNoHandle {
					child = value
				}
			}
		}

		if parent != microsoftMoneyNoHandle {
			parents[parent] = true
		}

		if child != microsoftMoneyNoHandle && parent != microsoftMoneyNoHandle {
			children[child] = parent
		}

		return nil
	})

	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrDataFileNotReadable, err)
	}

	return parents, children, nil
}

// transferCounterpart is the receiving side of a transfer: the amount the
// receiving account got, stored with the memo and date the receiving row holds.
type transferCounterpart struct {
	amount int64
	memo   string
	time   time.Time
}

// collectMicrosoftMoneySplitIntegrity records what the file's split links
// claim so validation can check them: the stored total of every parent row and
// how many split links refer to a row the file does not hold.
func collectMicrosoftMoneySplitIntegrity(rows []*microsoftMoneyTransactionRow, parents map[int64]bool, children map[int64]int64) (map[int64]int64, int) {
	totals := make(map[int64]int64, len(parents))
	held := make(map[int64]bool, len(rows))

	for _, row := range rows {
		held[row.Handle] = true

		if parents[row.Handle] {
			if row.Amount < 0 {
				totals[row.Handle] = convertMicrosoftMoneyAmount(-row.Amount)
			} else {
				totals[row.Handle] = convertMicrosoftMoneyAmount(row.Amount)
			}
		}
	}

	missing := 0

	for child, parent := range children {
		if !held[child] || !held[parent] {
			missing++
		}
	}

	return totals, missing
}

// flattenMicrosoftMoneyTransactions turns the stored rows into transactions,
// reporting the count of rows that cannot become one. A transfer is stored twice,
// once for each account, so only the row that takes the money out becomes a
// transaction, with the account that receives it as the related account. The
// amount the receiving account got is taken from its own row, because it can
// differ from the outgoing amount when the two accounts use different
// currencies; when the receiving row cannot be matched unambiguously, the
// outgoing amount is used instead of inventing one.
func flattenMicrosoftMoneyTransactions(rows []*microsoftMoneyTransactionRow, parents map[int64]bool, children map[int64]int64, accounts map[int64]*microsoftMoneyAccount, categories map[int64]*microsoftMoneyCategory, payees map[int64]string) ([]*microsoftMoneyTransaction, int) {
	// Index the receiving side of every transfer: an outgoing row holds the
	// receiving account in Link, and the receiving row holds the outgoing
	// account in Link with the opposite sign. Split fragments never act as the
	// receiving side of a whole transfer.
	counterparts := make(map[[2]int64][]transferCounterpart)

	for _, row := range rows {
		if row.Link == microsoftMoneyNoHandle || row.Amount <= 0 || children[row.Handle] != 0 {
			continue
		}

		key := [2]int64{row.Account, row.Link}
		counterparts[key] = append(counterparts[key], transferCounterpart{amount: row.Amount, memo: row.Memo, time: row.Time})
	}

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
			Split:       children[row.Handle] != 0,
			SplitParent: children[row.Handle],
			Payee:       payees[row.Payee],
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

			transaction.Type = transactionTypeTransfer
			transaction.Related = related
			transaction.Amount = convertMicrosoftMoneyAmount(-row.Amount)
			transaction.RelatedAmount = matchTransferCounterpartAmount(row, children, counterparts)
		} else {
			if row.Amount > 0 {
				transaction.Type = transactionTypeIncome
				transaction.Amount = convertMicrosoftMoneyAmount(row.Amount)
			} else {
				transaction.Type = transactionTypeExpense
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

// matchTransferCounterpartAmount returns the amount the receiving account got
// for an outgoing transfer row, converted to hundredths, or zero when no single
// receiving row matches. A split fragment never receives a whole transfer, so
// no amount is taken from one.
func matchTransferCounterpartAmount(row *microsoftMoneyTransactionRow, children map[int64]int64, counterparts map[[2]int64][]transferCounterpart) int64 {
	if children[row.Handle] != 0 {
		return 0
	}

	candidates := counterparts[[2]int64{row.Link, row.Account}]
	sameDay := make([]*transferCounterpart, 0, len(candidates))

	for i := range candidates {
		if sameMoneyDay(candidates[i].time, row.Time) {
			sameDay = append(sameDay, &candidates[i])
		}
	}

	if len(sameDay) == 0 {
		return 0
	}

	if len(sameDay) == 1 && sameDay[0].amount > 0 {
		return convertMicrosoftMoneyAmount(sameDay[0].amount)
	}

	// Several receiving rows on the same day: only a memo that agrees with the
	// outgoing row tells them apart. Without it the source amount is kept
	// rather than guessing.
	if row.Memo == "" {
		return 0
	}

	for _, candidate := range sameDay {
		if candidate.memo == row.Memo && candidate.amount > 0 {
			return convertMicrosoftMoneyAmount(candidate.amount)
		}
	}

	return 0
}

// sameMoneyDay reports whether two Money rows fall on the same day, since a
// Money transaction stores a date without a time.
func sameMoneyDay(a time.Time, b time.Time) bool {
	return a.Year() == b.Year() && a.Month() == b.Month() && a.Day() == b.Day()
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
