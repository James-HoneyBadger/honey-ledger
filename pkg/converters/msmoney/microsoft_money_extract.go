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
	// Closed and Hidden mirror the flags the file stores on the account row.
	// An account may stay in the file after the user closed or hid it, with
	// its history intact.
	Closed bool
	Hidden bool
}

// ExtractedTransaction is one transaction of a Microsoft Money file. The amount
// is in hundredths of the currency and is always positive; Type tells whether
// the money came in, went out or moved to TransferAccount.
type ExtractedTransaction struct {
	Time     time.Time
	Type     string
	Account  string
	Currency string
	Amount   int64
	// Split marks a transaction that came from one fragment of a split
	// transaction, preserving its provenance.
	Split            bool
	TransferAccount  string
	TransferCurrency string
	// TransferAmount is the amount the transfer account received, in its own
	// currency. Zero means the receiving side was not known and Amount applies.
	TransferAmount int64
	Category       string
	SubCategory    string
	Memo           string
	// Payee is the payee the file names on the transaction, when it names one.
	Payee string
	// splitParent carries the handle of the split parent row this transaction
	// is a fragment of. It stays inside the engine so validation can compare
	// the fragments' sum with the stored total.
	splitParent int64
}

// SkippedBreakdown counts, by reason, the rows that cannot become a
// transaction, so a skipped row is never just part of an opaque total. Every
// field is a reason; Total is their sum.
type SkippedBreakdown struct {
	// SplitTotals counts the parent rows that only hold the total of their
	// splits; each split is exported as a transaction of its own.
	SplitTotals int
	// Investments counts investment activity, which has no counterpart in a
	// plain ledger.
	Investments int
	// NoAmount counts rows whose amount is zero, which move no money.
	NoAmount int
	// NoAccountOrDate counts rows whose account or date is missing.
	NoAccountOrDate int
	// TransferReceivingSide counts the receiving rows of transfers; the
	// transfer is exported once, from the side the money left.
	TransferReceivingSide int
	// TransferWithoutAccount counts transfers whose receiving account the
	// file does not hold.
	TransferWithoutAccount int
}

// Total returns the number of rows that cannot become a transaction.
func (s SkippedBreakdown) Total() int {
	return s.SplitTotals + s.Investments + s.NoAmount + s.NoAccountOrDate +
		s.TransferReceivingSide + s.TransferWithoutAccount
}

// Reasons returns the non-zero reasons in a fixed order, phrased for a user,
// or nil when nothing was skipped.
func (s SkippedBreakdown) Reasons() []string {
	type entry struct {
		count int
		one   string
		many  string
	}

	entries := []entry{
		{s.SplitTotals, "1 split total", "%d split totals"},
		{s.Investments, "1 investment row", "%d investment rows"},
		{s.NoAmount, "1 row without an amount", "%d rows without an amount"},
		{s.NoAccountOrDate, "1 row without an account or date", "%d rows without an account or date"},
		{s.TransferReceivingSide, "1 receiving side of a transfer", "%d receiving sides of transfers"},
		{s.TransferWithoutAccount, "1 transfer whose receiving account is unknown", "%d transfers whose receiving account is unknown"},
	}

	var reasons []string

	for _, reason := range entries {
		if reason.count == 1 {
			reasons = append(reasons, reason.one)
		} else if reason.count > 1 {
			reasons = append(reasons, fmt.Sprintf(reason.many, reason.count))
		}
	}

	return reasons
}

// ExtractedData is everything of a Microsoft Money file that can be exported
type ExtractedData struct {
	Accounts     []ExtractedAccount
	Transactions []ExtractedTransaction
	// Skipped counts the rows that cannot be exported, by reason. Total is
	// the number of rows left out; the fields name why.
	Skipped SkippedBreakdown
	// Diagnostics reports the decoding problems found while reading the file:
	// rows, definitions and catalog entries that could not be decoded. An
	// empty report means everything that was found decoded cleanly.
	Diagnostics ExtractedDiagnostics
	// splitTotals and splitReferencesWithoutRows stay inside the engine so
	// validation can check the file's split links against its rows.
	splitTotals                map[int64]int64
	splitReferencesWithoutRows int
}

// ExtractedDiagnostics summarizes the decoding problems found while reading a
// file. Counts are authoritative; Samples is a bounded selection of the first
// problems found, for humans.
type ExtractedDiagnostics struct {
	CorruptRows           int
	UnresolvedRows        int
	CorruptDefinitions    int
	UnknownCatalogEntries int
	Samples               []string
}

// IsEmpty reports whether no decoding problem was found.
func (d ExtractedDiagnostics) IsEmpty() bool {
	return d.CorruptRows == 0 && d.UnresolvedRows == 0 && d.CorruptDefinitions == 0 && d.UnknownCatalogEntries == 0
}

// Summary returns a one line human-readable description of the decoding
// problems, or an empty string when none were found.
func (d ExtractedDiagnostics) Summary() string {
	if d.IsEmpty() {
		return ""
	}

	return fmt.Sprintf("%d rows undecodable, %d rows unresolved, %d definitions undecodable, %d catalog entries unreadable",
		d.CorruptRows, d.UnresolvedRows, d.CorruptDefinitions, d.UnknownCatalogEntries)
}

// Progress describes one step of an extraction, for a progress display.
type Progress struct {
	// Phase names the step: opening, reading tables, transactions, flattening.
	Phase string
	// Done is how many units of the step are complete, and Total how many
	// there are; both are zero for steps that do not count units.
	Done  int
	Total int
}

// ExtractionOptions configures an extraction. The zero value extracts without
// progress reporting; cancellation is always honoured through the context.
type ExtractionOptions struct {
	// Progress, when set, is called as the extraction advances. The calls come
	// from the goroutine that runs the extraction.
	Progress func(progress Progress)
}

// ExtractMicrosoftMoneyData reads a Microsoft Money data file (.mny) or backup
// (.mbf) and returns its accounts and transactions without needing a ledger.
func ExtractMicrosoftMoneyData(ctx Context, data []byte) (*ExtractedData, error) {
	return ExtractMicrosoftMoneyDataWithOptions(ctx, data, ExtractionOptions{})
}

// ExtractMicrosoftMoneyDataWithOptions reads a Microsoft Money data file or
// backup like ExtractMicrosoftMoneyData, reporting progress and honouring
// cancellation through the context.
func ExtractMicrosoftMoneyDataWithOptions(ctx Context, data []byte, options ExtractionOptions) (*ExtractedData, error) {
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

	moneyData, err := readMicrosoftMoneyData(ctx, data, options)

	if err != nil {
		return nil, err
	}

	result := &ExtractedData{
		Accounts:                   make([]ExtractedAccount, 0, len(moneyData.Accounts)),
		Transactions:               make([]ExtractedTransaction, 0, len(moneyData.Transactions)),
		Skipped:                    moneyData.Skipped,
		Diagnostics:                moneyData.Diagnostics,
		splitTotals:                moneyData.SplitTotals,
		splitReferencesWithoutRows: moneyData.SplitReferencesWithoutRows,
	}

	for _, account := range moneyData.Accounts {
		result.Accounts = append(result.Accounts, ExtractedAccount{
			Name:     account.Name,
			Currency: account.Currency,
			Closed:   account.Closed,
			Hidden:   account.Hidden,
		})
	}

	for _, transaction := range moneyData.Transactions {
		extracted := ExtractedTransaction{
			Time:        transaction.Time,
			Account:     transaction.Account.Name,
			Currency:    transaction.Account.Currency,
			Amount:      transaction.Amount,
			Split:       transaction.Split,
			splitParent: transaction.SplitParent,
			Category:    transaction.Category,
			SubCategory: transaction.SubCategory,
			Memo:        transaction.Description,
			Payee:       transaction.Payee,
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
