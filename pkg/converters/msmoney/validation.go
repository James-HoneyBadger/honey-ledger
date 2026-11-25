// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import (
	"fmt"
	"sort"
	"time"
)

// Severity tells how certain a validation finding is.
type Severity string

const (
	// SeverityConfirmed marks a finding computed directly from decoded data,
	// with no interpretation in between.
	SeverityConfirmed Severity = "confirmed"
	// SeverityUncertain marks a finding that depends on an interpretation,
	// such as a heuristic match.
	SeverityUncertain Severity = "uncertain"
	// SeverityInfo marks an observation that is neither an error nor a
	// discrepancy.
	SeverityInfo Severity = "info"
)

// FindingKind names the check that produced a finding.
type FindingKind string

const (
	// FindingDuplicateTransaction marks two or more identical transactions.
	FindingDuplicateTransaction FindingKind = "duplicate_transaction"
	// FindingTransferAmountFallback marks a transfer whose receiving side
	// could not be matched, so the source amount was used for both sides.
	FindingTransferAmountFallback FindingKind = "transfer_amount_fallback"
	// FindingAccountWithoutTransactions marks an account the file holds that
	// has no transactions after extraction.
	FindingAccountWithoutTransactions FindingKind = "account_without_transactions"
	// FindingOrphanTransferAccount marks a transfer whose receiving account is
	// not among the file's accounts.
	FindingOrphanTransferAccount FindingKind = "orphan_transfer_account"
	// FindingTransferAmountMismatch marks a transfer between two accounts of
	// the same currency whose sides carry different amounts.
	FindingTransferAmountMismatch FindingKind = "transfer_amount_mismatch"
	// FindingSplitTotalMismatch marks a split whose fragments do not add up to
	// the total the parent row stores.
	FindingSplitTotalMismatch FindingKind = "split_total_mismatch"
	// FindingSplitBrokenReference marks a split link whose transaction row the
	// file does not hold.
	FindingSplitBrokenReference FindingKind = "split_broken_reference"
)

// Finding is one observation of the validation report. Severity distinguishes
// confirmed findings from uncertain interpretations, so a consumer never
// mistakes one for the other.
type Finding struct {
	Severity Severity
	Kind     FindingKind
	// Account is the account the finding is about.
	Account string
	// TransferAccount is the receiving account of a transfer finding.
	TransferAccount string
	// Time is the transaction date of the finding, when it has one.
	Time time.Time
	// Amount is the transaction amount of the finding, in hundredths, when it
	// has one.
	Amount int64
	// Message explains the finding in words.
	Message string
}

// ValidationReport is the result of validating extracted data. Findings are
// reported, never repaired: the report never invents transactions to make the
// books balance.
type ValidationReport struct {
	Findings []Finding
	// Summary counts and totals the extracted data, for a reconciliation
	// overview. It is calculated from the transactions the file holds, not a
	// Money account balance.
	Summary ValidationSummary
}

// ValidationSummary counts and totals the extracted transactions.
type ValidationSummary struct {
	TransactionCount int
	IncomeCount      int
	ExpenseCount     int
	TransferCount    int
	IncomeTotal      int64
	ExpenseTotal     int64
	// Earliest and Latest are the first and last transaction dates, zero when
	// there are no transactions.
	Earliest time.Time
	Latest   time.Time
}

// IsClean reports whether the validation found nothing worth reporting.
func (r ValidationReport) IsClean() bool {
	return len(r.Findings) == 0
}

// Confirmed returns the findings computed directly from decoded data.
func (r ValidationReport) Confirmed() []Finding {
	return r.withSeverity(SeverityConfirmed)
}

// Uncertain returns the findings that depend on an interpretation.
func (r ValidationReport) Uncertain() []Finding {
	return r.withSeverity(SeverityUncertain)
}

func (r ValidationReport) withSeverity(severity Severity) []Finding {
	findings := make([]Finding, 0, len(r.Findings))

	for _, finding := range r.Findings {
		if finding.Severity == severity {
			findings = append(findings, finding)
		}
	}

	return findings
}

// transactionKey identifies a transaction by the fields that make it
// distinguishable in a statement: the same account, day, amount, category and
// memo appearing twice is a duplicate.
type transactionKey struct {
	account     string
	date        string
	amount      int64
	category    string
	subCategory string
	memo        string
	transferTo  string
}

func keyOf(transaction ExtractedTransaction) transactionKey {
	return transactionKey{
		account:     transaction.Account,
		date:        transaction.Time.Format("2006-01-02"),
		amount:      transaction.Amount,
		category:    transaction.Category,
		subCategory: transaction.SubCategory,
		memo:        transaction.Memo,
		transferTo:  transaction.TransferAccount,
	}
}

// ValidateTransactions checks the extracted data for the problems that can be
// seen from the data alone: duplicate transactions, transfer amounts that fell
// back to the source amount, and accounts without transactions. A finding's
// severity marks whether it is confirmed or an interpretation.
func ValidateTransactions(data *ExtractedData) ValidationReport {
	report := ValidationReport{Findings: make([]Finding, 0)}

	seen := make(map[transactionKey]int, len(data.Transactions))

	for _, transaction := range data.Transactions {
		seen[keyOf(transaction)]++
	}

	for _, transaction := range data.Transactions {
		report.Summary.TransactionCount++
		report.Summary.Earliest, report.Summary.Latest = extendDateRange(report.Summary.Earliest, report.Summary.Latest, transaction.Time)

		switch transaction.Type {
		case ExtractedTypeIncome:
			report.Summary.IncomeCount++
			report.Summary.IncomeTotal += transaction.Amount
		case ExtractedTypeExpense:
			report.Summary.ExpenseCount++
			report.Summary.ExpenseTotal += transaction.Amount
		case ExtractedTypeTransfer:
			report.Summary.TransferCount++
		}

		key := keyOf(transaction)

		if count := seen[key]; count > 1 {
			report.Findings = append(report.Findings, Finding{
				Severity: SeverityConfirmed,
				Kind:     FindingDuplicateTransaction,
				Account:  transaction.Account,
				Time:     transaction.Time,
				Amount:   transaction.Amount,
				Message:  fmt.Sprintf("the transaction appears %d times with the same account, date, amount, category and memo", count),
			})

			// One finding per duplicate group, not one per copy.
			delete(seen, key)
		}

		if transaction.Type == ExtractedTypeTransfer && transaction.TransferAmount == 0 {
			report.Findings = append(report.Findings, Finding{
				Severity:        SeverityUncertain,
				Kind:            FindingTransferAmountFallback,
				Account:         transaction.Account,
				TransferAccount: transaction.TransferAccount,
				Time:            transaction.Time,
				Amount:          transaction.Amount,
				Message:         "the receiving side of the transfer could not be matched, so the source amount is used for both sides",
			})
		}
	}

	known := make(map[string]string, len(data.Accounts))

	for _, account := range data.Accounts {
		known[account.Name] = account.Currency
	}

	for _, transaction := range data.Transactions {
		if transaction.Type != ExtractedTypeTransfer {
			continue
		}

		currency, exists := known[transaction.TransferAccount]

		if !exists {
			report.Findings = append(report.Findings, Finding{
				Severity:        SeverityConfirmed,
				Kind:            FindingOrphanTransferAccount,
				Account:         transaction.Account,
				TransferAccount: transaction.TransferAccount,
				Time:            transaction.Time,
				Amount:          transaction.Amount,
				Message:         fmt.Sprintf("the transfer points at account %q, which the file does not hold", transaction.TransferAccount),
			})
		}

		if exists && currency == transaction.Currency && transaction.TransferAmount != 0 && transaction.TransferAmount != transaction.Amount {
			report.Findings = append(report.Findings, Finding{
				Severity:        SeverityUncertain,
				Kind:            FindingTransferAmountMismatch,
				Account:         transaction.Account,
				TransferAccount: transaction.TransferAccount,
				Time:            transaction.Time,
				Amount:          transaction.Amount,
				Message:         "the two sides of a same-currency transfer carry different amounts",
			})
		}
	}

	active := make(map[string]bool, len(data.Transactions))

	for _, transaction := range data.Transactions {
		active[transaction.Account] = true

		if transaction.TransferAccount != "" {
			active[transaction.TransferAccount] = true
		}
	}

	for _, account := range data.Accounts {
		if !active[account.Name] {
			report.Findings = append(report.Findings, Finding{
				Severity: SeverityInfo,
				Kind:     FindingAccountWithoutTransactions,
				Account:  account.Name,
				Message:  "the account has no transactions",
			})
		}
	}

	reportSplitFindings(data, &report)

	return report
}

// reportSplitFindings compares the split links the file stores with its rows:
// the fragments' sum against the parent's stored total, and split links that
// point at a row the file does not hold.
func reportSplitFindings(data *ExtractedData, report *ValidationReport) {
	fragmentSums := make(map[int64]int64, len(data.splitTotals))

	for _, transaction := range data.Transactions {
		if transaction.splitParent != 0 {
			fragmentSums[transaction.splitParent] += transaction.Amount
		}
	}

	// A deterministic order keeps the report stable run to run.
	parents := make([]int64, 0, len(data.splitTotals))

	for parent := range data.splitTotals {
		parents = append(parents, parent)
	}

	sort.Slice(parents, func(i, j int) bool { return parents[i] < parents[j] })

	for _, parent := range parents {
		total := data.splitTotals[parent]

		if fragmentSums[parent] == 0 || fragmentSums[parent] == total {
			continue
		}

		account := splitAccountOf(data, parent)

		report.Findings = append(report.Findings, Finding{
			Severity: SeverityConfirmed,
			Kind:     FindingSplitTotalMismatch,
			Account:  account,
			Message:  fmt.Sprintf("the split fragments add up to %d.%02d but the parent row stores %d.%02d", fragmentSums[parent]/100, fragmentSums[parent]%100, total/100, total%100),
		})
	}

	if data.splitReferencesWithoutRows > 0 {
		report.Findings = append(report.Findings, Finding{
			Severity: SeverityConfirmed,
			Kind:     FindingSplitBrokenReference,
			Message:  fmt.Sprintf("%d split link(s) point at a transaction row the file does not hold", data.splitReferencesWithoutRows),
		})
	}
}

// splitAccountOf returns the account of the first fragment of the split whose
// parent has the given handle, or an empty string when no fragment is left.
func splitAccountOf(data *ExtractedData, parent int64) string {
	for _, transaction := range data.Transactions {
		if transaction.splitParent == parent {
			return transaction.Account
		}
	}

	return ""
}

// extendDateRange returns the range of the given dates widened to include the
// given time.
func extendDateRange(earliest, latest, when time.Time) (time.Time, time.Time) {
	if when.IsZero() {
		return earliest, latest
	}

	if earliest.IsZero() || when.Before(earliest) {
		earliest = when
	}

	if latest.IsZero() || when.After(latest) {
		latest = when
	}

	return earliest, latest
}
