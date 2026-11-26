// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/msmoney/synthetic"
)

// engineMoneyHeader returns the first page of a database that has no password.
func engineMoneyHeader() []byte {
	page := make([]byte, 4096)
	copy(page[4:], []byte("MSISAM Database"))
	page[0x14] = 1

	for index, key := 0, 0; index < 128; index++ {
		page[0x18+index] = page[0x18+index] ^ byte(key)
	}

	return page
}

func TestIsMicrosoftMoneyDataFile(t *testing.T) {
	assert.True(t, IsMicrosoftMoneyDataFile(engineMoneyHeader()))
	assert.False(t, IsMicrosoftMoneyDataFile(append(make([]byte, 4), []byte("Standard Jet DB")...)))
	assert.False(t, IsMicrosoftMoneyDataFile([]byte("SQLite format 3\x00")))
	assert.False(t, IsMicrosoftMoneyDataFile([]byte("!Type:Bank\nD01/02/2024\nT-1.00\n^")))
	assert.False(t, IsMicrosoftMoneyDataFile(nil))
	assert.False(t, IsMicrosoftMoneyDataFile(engineMoneyHeader()[:18]))
}

func TestExtractMicrosoftMoneyDataRejectsUnknownFormat(t *testing.T) {
	_, err := ExtractMicrosoftMoneyData(newTestContext(), []byte("SQLite format 3\x00"))
	assert.ErrorIs(t, err, ErrFileTypeNotSupported)
}

func TestFlattenMicrosoftMoneyTransactions(t *testing.T) {
	checking := &microsoftMoneyAccount{Name: "Checking", Currency: "USD"}
	savings := &microsoftMoneyAccount{Name: "Savings", Currency: "EUR"}
	accounts := map[int64]*microsoftMoneyAccount{100: checking, 200: savings}
	categories := map[int64]*microsoftMoneyCategory{
		1001: {Name: "Groceries", Parent: "Food"},
		1002: {Name: "Wages"},
	}

	moment := time.Date(2011, time.May, 19, 0, 0, 0, 0, time.UTC)
	day2 := time.Date(2011, time.May, 20, 0, 0, 0, 0, time.UTC)
	day3 := time.Date(2011, time.May, 21, 0, 0, 0, 0, time.UTC)
	day4 := time.Date(2011, time.May, 22, 0, 0, 0, 0, time.UTC)
	day5 := time.Date(2011, time.May, 23, 0, 0, 0, 0, time.UTC)

	rows := []*microsoftMoneyTransactionRow{
		{Handle: 1, Account: 100, Category: 1001, Amount: -250000, Time: moment, Memo: "Lunch"},
		{Handle: 2, Account: 100, Category: 1002, Amount: 150000, Time: moment},
		// A transfer whose two sides agree, stored as Money stores them.
		{Handle: 3, Account: 100, Link: 200, Amount: -50000, Time: moment},
		{Handle: 4, Account: 200, Link: 100, Amount: 50000, Time: moment},
		// A cross-currency transfer: the receiving side gets a different amount.
		{Handle: 5, Account: 100, Link: 200, Amount: -100000, Time: day2},
		{Handle: 6, Account: 200, Link: 100, Amount: 160000, Time: day2},
		// Two transfers in the same direction on the same day with no memo:
		// the receiving rows cannot be told apart, so no related amount is set.
		{Handle: 7, Account: 100, Link: 200, Amount: -20000, Time: day3},
		{Handle: 8, Account: 100, Link: 200, Amount: -30000, Time: day3},
		{Handle: 9, Account: 200, Link: 100, Amount: 20000, Time: day3},
		{Handle: 10, Account: 200, Link: 100, Amount: 30000, Time: day3},
		// Two same-day transfers disambiguated by their memos.
		{Handle: 11, Account: 100, Link: 200, Amount: -40000, Time: day4, Memo: "Rent"},
		{Handle: 12, Account: 100, Link: 200, Amount: -50000, Time: day4, Memo: "Trip"},
		{Handle: 13, Account: 200, Link: 100, Amount: 40000, Time: day4, Memo: "Rent"},
		{Handle: 14, Account: 200, Link: 100, Amount: 50000, Time: day4, Memo: "Trip"},
		// A split fragment is never matched to a whole receiving row.
		{Handle: 15, Account: 100, Link: 200, Amount: -60000, Time: day5},
		{Handle: 16, Account: 200, Link: 100, Amount: 60000, Time: day5},
		// Rows that cannot become transactions.
		{Handle: 17, Account: 100, Amount: 0, Time: moment},
		{Handle: 18, Account: 100, Security: 9, Amount: -750000, Time: moment},
		{Handle: 19, Account: 100, Amount: -7218000, Time: moment},
		{Handle: 20, Account: 100, Amount: -10000, Time: time.Time{}},
		{Handle: 21, Account: 999, Amount: -10000, Time: moment},
	}

	parents := map[int64]bool{19: true}
	children := map[int64]int64{15: 19}

	transactions, skipped := flattenMicrosoftMoneyTransactions(rows, parents, children, accounts, categories, nil)
	require.Len(t, transactions, 9)
	assert.Equal(t, SkippedBreakdown{
		SplitTotals:           1, // the split parent
		Investments:           1, // the investment row
		NoAmount:              1, // the zero-amount row
		NoAccountOrDate:       2, // the dateless row and the unknown-account row
		TransferReceivingSide: 7, // every receiving row of a transfer
	}, skipped)

	assert.Equal(t, transactionTypeExpense, transactions[0].Type)
	assert.Equal(t, int64(2500), transactions[0].Amount)
	assert.Equal(t, "Food", transactions[0].Category)
	assert.Equal(t, "Groceries", transactions[0].SubCategory)
	assert.Equal(t, "Lunch", transactions[0].Description)

	assert.Equal(t, transactionTypeIncome, transactions[1].Type)
	assert.Equal(t, int64(1500), transactions[1].Amount)
	assert.Empty(t, transactions[1].Category)
	assert.Equal(t, "Wages", transactions[1].SubCategory)

	transfers := transactions[2:]

	assert.Equal(t, transactionTypeTransfer, transfers[0].Type)
	assert.Equal(t, int64(500), transfers[0].Amount)
	assert.Equal(t, savings, transfers[0].Related)
	assert.Equal(t, int64(500), transfers[0].RelatedAmount)
	assert.Empty(t, transfers[0].SubCategory)

	assert.Equal(t, int64(1000), transfers[1].Amount)
	assert.Equal(t, int64(1600), transfers[1].RelatedAmount, "the receiving side's amount is preserved across currencies")

	for _, transfer := range transfers[2:4] {
		assert.Equal(t, int64(0), transfer.RelatedAmount, "ambiguous same-day transfers keep the source amount instead of guessing")
	}

	byAmount := make(map[int64]int64, len(transfers))

	for _, transfer := range transfers[4:6] {
		byAmount[transfer.Amount] = transfer.RelatedAmount
	}

	assert.Equal(t, int64(400), byAmount[int64(400)])
	assert.Equal(t, int64(500), byAmount[int64(500)])

	assert.Equal(t, int64(600), transfers[6].Amount)
	assert.Equal(t, int64(0), transfers[6].RelatedAmount, "a split fragment cannot consume a whole receiving row")
	assert.True(t, transfers[6].Split, "a split fragment keeps its provenance")
	assert.False(t, transfers[0].Split, "a whole transfer is not a split")
}

func TestFlattenMicrosoftMoneyTransactionsWithoutReceivingRows(t *testing.T) {
	checking := &microsoftMoneyAccount{Name: "Checking", Currency: "USD"}
	savings := &microsoftMoneyAccount{Name: "Savings", Currency: "EUR"}
	accounts := map[int64]*microsoftMoneyAccount{100: checking, 200: savings}

	moment := time.Date(2011, time.May, 19, 0, 0, 0, 0, time.UTC)

	rows := []*microsoftMoneyTransactionRow{
		{Handle: 1, Account: 100, Link: 200, Amount: -50000, Time: moment},
	}

	transactions, skipped := flattenMicrosoftMoneyTransactions(rows, nil, nil, accounts, nil, nil)
	require.Len(t, transactions, 1)
	assert.Equal(t, SkippedBreakdown{}, skipped)

	assert.Equal(t, int64(500), transactions[0].Amount)
	assert.Equal(t, int64(0), transactions[0].RelatedAmount)
}

func TestConvertMicrosoftMoneyAmount(t *testing.T) {
	assert.Equal(t, int64(2500), convertMicrosoftMoneyAmount(250000))
	assert.Equal(t, int64(-2500), convertMicrosoftMoneyAmount(-250000))
	assert.Equal(t, int64(1), convertMicrosoftMoneyAmount(50))
	assert.Equal(t, int64(0), convertMicrosoftMoneyAmount(49))
	assert.Equal(t, int64(-1), convertMicrosoftMoneyAmount(-50))
	assert.Equal(t, int64(12346), convertMicrosoftMoneyAmount(1234567))
}

func TestTrimMicrosoftMoneyText(t *testing.T) {
	assert.Equal(t, "Principal", trimMicrosoftMoneyText(microsoftMoneyByteOrderMark+"Principal"))
	assert.Equal(t, "Interest", trimMicrosoftMoneyText(" Interest "))
	assert.Equal(t, "", trimMicrosoftMoneyText(""))
}

// TestRealDatabaseExtract reads a Microsoft Money data file, and runs only when
// such a file is supplied.
func TestRealDatabaseExtract(t *testing.T) {
	path := os.Getenv("MSISAM_SAMPLE_FILE")

	if path == "" {
		t.Skip("set MSISAM_SAMPLE_FILE to a Microsoft Money data file")
	}

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	moneyData, err := readMicrosoftMoneyData(newTestContext(), data, ExtractionOptions{})
	require.NoError(t, err)
	require.NotEmpty(t, moneyData.Transactions)

	types := map[transactionType]int{}
	categorized := 0

	for _, transaction := range moneyData.Transactions {
		types[transaction.Type]++
		assert.NotEmpty(t, transaction.Account.Name)
		assert.NotZero(t, transaction.Amount)
		assert.False(t, transaction.Time.IsZero())

		if transaction.Type == transactionTypeTransfer {
			require.NotNil(t, transaction.Related)
			assert.NotEmpty(t, transaction.Related.Name)
			assert.Empty(t, transaction.SubCategory)

			continue
		}

		if transaction.SubCategory != "" {
			categorized++
		}
	}

	// Money allows a transaction without a category, but most carry one.
	assert.Greater(t, categorized, len(moneyData.Transactions)/2)

	assert.Greater(t, types[transactionTypeExpense], 100)
	assert.Greater(t, types[transactionTypeIncome], 100)
	assert.Greater(t, types[transactionTypeTransfer], 100)
	assert.Greater(t, moneyData.Skipped.Total(), 0)
}

func TestExtractReportsProgress(t *testing.T) {
	data := synthetic.MoneyFile()

	var phases []string
	last := Progress{}

	err := func() error {
		_, err := ExtractMicrosoftMoneyDataWithOptions(newTestContext(), data, ExtractionOptions{
			Progress: func(progress Progress) {
				phases = append(phases, progress.Phase)
				last = progress
			},
		})

		return err
	}()

	require.NoError(t, err)
	assert.Contains(t, phases, "opening")
	assert.Contains(t, phases, "reading tables")
	assert.Contains(t, phases, "transactions")
	assert.Contains(t, phases, "flattening")
	assert.Equal(t, "flattening", last.Phase, "the final progress report names the last step")
}

func TestExtractSurfacesDiagnostics(t *testing.T) {
	// A corrupt transaction page row makes the engine record an unresolved row,
	// which must surface in the extracted data instead of being silent.
	data := corruptTransactionFile()

	extracted, err := ExtractMicrosoftMoneyData(newTestContext(), data)
	require.NoError(t, err)
	assert.False(t, extracted.Diagnostics.IsEmpty(), "a damaged row must be reported")
	// The slot table is shared between a row's start and the previous row's
	// end, so corrupting one entry breaks the two rows that touch it.
	assert.Equal(t, 2, extracted.Diagnostics.UnresolvedRows)
	assert.NotEmpty(t, extracted.Diagnostics.Summary())
}

// corruptTransactionFile returns the synthetic database with one transaction
// row slot pointing outside its page, so the row cannot be resolved.
func corruptTransactionFile() []byte {
	data := synthetic.MoneyFile()

	// The synthetic layout puts TRN data on page 9; its first slot table entry
	// sits at offset 0x0E (msisam.rowStartTable). Pointing it past the page end
	// makes the row unresolvable while keeping everything else intact.
	const trnDataPage = 9
	const firstSlot = trnDataPage*4096 + 0x0E

	data[firstSlot] = 0x00
	data[firstSlot+1] = 0x30

	return data
}
