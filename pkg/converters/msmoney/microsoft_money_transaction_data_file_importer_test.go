// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/converter"
	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
	"github.com/James-HoneyBadger/honey-ledger/pkg/models"
)

// Column types of the Microsoft Money data file format.
const (
	moneyColumnLongInt  = 0x04
	moneyColumnMoney    = 0x05
	moneyColumnDateTime = 0x08
	moneyColumnText     = 0x0A
	moneyColumnMemo     = 0x0C
)

func moneyPutU16(buffer []byte, offset, value int) {
	buffer[offset] = byte(value)
	buffer[offset+1] = byte(value >> 8)
}

func moneyPutU32(buffer []byte, offset int, value uint32) {
	buffer[offset] = byte(value)
	buffer[offset+1] = byte(value >> 8)
	buffer[offset+2] = byte(value >> 16)
	buffer[offset+3] = byte(value >> 24)
}

func moneyPutU64(buffer []byte, offset int, value uint64) {
	moneyPutU32(buffer, offset, uint32(value))
	moneyPutU32(buffer, offset+4, uint32(value>>32))
}

// moneyScramble applies the stream cipher the format uses to hide the header
// fields.
func moneyScramble(key, data []byte) []byte {
	var state [256]byte

	for index := range state {
		state[index] = byte(index)
	}

	for index, swap := 0, 0; index < len(state); index++ {
		swap = (swap + int(state[index]) + int(key[index%len(key)])) & 0xFF
		state[index], state[swap] = state[swap], state[index]
	}

	output := make([]byte, len(data))
	left, right := 0, 0

	for index, value := range data {
		left = (left + 1) & 0xFF
		right = (right + int(state[left])) & 0xFF
		state[left], state[right] = state[right], state[left]
		output[index] = value ^ state[(int(state[left])+int(state[right]))&0xFF]
	}

	return output
}

// moneyHeader returns the first page of a database that has no password.
func moneyHeader() []byte {
	page := make([]byte, 4096)
	copy(page[4:], []byte("MSISAM Database"))
	page[0x14] = 1
	copy(page[0x18:], moneyScramble([]byte{0xC7, 0xDA, 0x39, 0x6B}, make([]byte, 128)))

	return page
}

// moneyUTF16 returns the UTF-16 encoding of text.
func moneyUTF16(text string) []byte {
	encoded := make([]byte, 0, len(text)*2)

	for _, character := range text {
		encoded = append(encoded, byte(character), byte(character>>8))
	}

	return encoded
}

func moneyInt32(value int64) []byte {
	buffer := make([]byte, 4)
	moneyPutU32(buffer, 0, uint32(value))

	return buffer
}

// moneyAmount returns an amount as the file stores it, with four decimals.
func moneyAmount(value int64) []byte {
	buffer := make([]byte, 8)
	moneyPutU64(buffer, 0, uint64(value))

	return buffer
}

// moneyMemo returns a memo as the file stores it: a header that marks the value
// as inline, followed by the text.
func moneyMemo(text []byte) []byte {
	header := make([]byte, 12)
	moneyPutU32(header, 0, 0x80000000|uint32(len(text)))

	return append(header, text...)
}

// moneyDate returns a date as the file stores it, counting days from 1899-12-30.
func moneyDate(year int, month time.Month, day int) []byte {
	days := time.Date(year, month, day, 0, 0, 0, 0, time.UTC).Sub(time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)).Hours() / 24
	buffer := make([]byte, 8)
	moneyPutU64(buffer, 0, math.Float64bits(days))

	return buffer
}

// moneyTestColumn describes a column of a table in the test database.
type moneyTestColumn struct {
	name     string
	kind     byte
	fixed    bool
	offset   int
	size     int
	variable int
}

// moneyDefinitionPage lays out a table definition page.
func moneyDefinitionPage(rowCount int, columns []moneyTestColumn) []byte {
	variableCount := 0

	for _, column := range columns {
		if !column.fixed {
			variableCount++
		}
	}

	page := make([]byte, 4096)
	page[0] = 2
	moneyPutU32(page, 16, uint32(rowCount))
	moneyPutU16(page, 43, variableCount)
	moneyPutU16(page, 45, len(columns))

	offset := 63

	for index, column := range columns {
		entry := page[offset : offset+25]
		entry[0] = column.kind
		moneyPutU16(entry, 5, index)
		moneyPutU16(entry, 7, column.variable)
		moneyPutU16(entry, 21, column.offset)
		moneyPutU16(entry, 23, column.size)

		if column.fixed {
			entry[15] = 1
		}

		offset += 25
	}

	for _, column := range columns {
		name := moneyUTF16(column.name)
		moneyPutU16(page, offset, len(name))
		offset += 2
		copy(page[offset:], name)
		offset += len(name)
	}

	return page
}

// moneyTestRow lays out one row: the column count, the fixed columns, the
// variable columns and, growing down from the end, their offsets, their count
// and the not-null mask.
func moneyTestRow(columnCount int, notNull []byte, fixed []byte, variables ...[]byte) []byte {
	row := []byte{byte(columnCount), byte(columnCount >> 8)}
	row = append(row, fixed...)

	offsets := make([]int, len(variables)+1)
	position := len(row)

	for index, variable := range variables {
		offsets[index] = position
		row = append(row, variable...)
		position += len(variable)
	}

	offsets[len(variables)] = position

	for index := len(offsets) - 1; index >= 0; index-- {
		row = append(row, byte(offsets[index]), byte(offsets[index]>>8))
	}

	row = append(row, byte(len(variables)), byte(len(variables)>>8))

	return append(row, notNull...)
}

// moneyDataPage lays out a page that holds the rows of the specified table.
func moneyDataPage(owner int, rows ...[]byte) []byte {
	page := make([]byte, 4096)
	page[0] = 1
	moneyPutU32(page, 4, uint32(owner))
	moneyPutU16(page, 0x0C, len(rows))

	position := 4096

	for index, row := range rows {
		position -= len(row)
		copy(page[position:], row)
		moneyPutU16(page, 0x0E+index*2, position)
	}

	return page
}

// moneyCatalogPage lays out the page that names the tables of the database.
func moneyCatalogPage(entries map[string]int) []byte {
	rows := make([][]byte, 0, len(entries))

	for name, page := range entries {
		rows = append(rows, moneyTestRow(2, []byte{0x03}, moneyInt32(int64(page)), moneyUTF16(name)))
	}

	return moneyDataPage(2, rows...)
}

// moneyTestDatabase returns a database that holds two accounts, five categories
// and the transactions that cover every rule of the importer.
func moneyTestDatabase() []byte {
	const expenseRoot, incomeRoot, food, bills, groceries, wages, principal, interest = 131, 130, 2001, 2002, 1001, 1002, 1003, 1004

	pages := [][]byte{
		moneyDefinitionPage(1, []moneyTestColumn{ // 1: CRNC
			{name: "hcrnc", kind: moneyColumnLongInt, fixed: true, offset: 0, size: 4},
			{name: "szIsoCode", kind: moneyColumnText, size: 3, variable: 0},
		}),
		nil, // 2: the page the catalog is registered under is unreadable
		moneyDataPage(1, moneyTestRow(2, []byte{0x03}, moneyInt32(45), moneyUTF16("USD"))), // 3: CRNC rows
		moneyDefinitionPage(2, []moneyTestColumn{ // 4: ACCT
			{name: "hacct", kind: moneyColumnLongInt, fixed: true, offset: 0, size: 4},
			{name: "hcrnc", kind: moneyColumnLongInt, fixed: true, offset: 4, size: 4},
			{name: "szFull", kind: moneyColumnText, size: 40, variable: 0},
		}),
		moneyDataPage(4, // 5: ACCT rows
			moneyTestRow(3, []byte{0x07}, append(moneyInt32(100), moneyInt32(45)...), moneyUTF16("Checking")),
			moneyTestRow(3, []byte{0x07}, append(moneyInt32(200), moneyInt32(45)...), moneyUTF16("Savings")),
		),
		moneyDefinitionPage(6, []moneyTestColumn{ // 6: CAT
			{name: "hcat", kind: moneyColumnLongInt, fixed: true, offset: 0, size: 4},
			{name: "hcatParent", kind: moneyColumnLongInt, fixed: true, offset: 4, size: 4},
			{name: "szFull", kind: moneyColumnText, size: 40, variable: 0},
		}),
		moneyDataPage(6, // 7: CAT rows
			moneyTestRow(3, []byte{0x07}, append(moneyInt32(expenseRoot), moneyInt32(0)...), moneyUTF16("EXPENSE")),
			moneyTestRow(3, []byte{0x07}, append(moneyInt32(incomeRoot), moneyInt32(0)...), moneyUTF16("INCOME")),
			moneyTestRow(3, []byte{0x07}, append(moneyInt32(food), moneyInt32(expenseRoot)...), moneyUTF16("Food")),
			moneyTestRow(3, []byte{0x07}, append(moneyInt32(bills), moneyInt32(expenseRoot)...), moneyUTF16("Bills")),
			moneyTestRow(3, []byte{0x07}, append(moneyInt32(groceries), moneyInt32(food)...), moneyUTF16("Groceries")),
			moneyTestRow(3, []byte{0x07}, append(moneyInt32(wages), moneyInt32(incomeRoot)...), moneyUTF16("Wages")),
			moneyTestRow(3, []byte{0x07}, append(moneyInt32(principal), moneyInt32(bills)...), moneyUTF16("Principal")),
			moneyTestRow(3, []byte{0x07}, append(moneyInt32(interest), moneyInt32(bills)...), moneyUTF16("Interest")),
		),
	}

	trnColumns := []moneyTestColumn{ // 8: TRN
		{name: "htrn", kind: moneyColumnLongInt, fixed: true, offset: 0, size: 4},
		{name: "hacct", kind: moneyColumnLongInt, fixed: true, offset: 4, size: 4},
		{name: "hacctLink", kind: moneyColumnLongInt, fixed: true, offset: 8, size: 4},
		{name: "hsec", kind: moneyColumnLongInt, fixed: true, offset: 12, size: 4},
		{name: "amt", kind: moneyColumnMoney, fixed: true, offset: 16, size: 8},
		{name: "hcat", kind: moneyColumnLongInt, fixed: true, offset: 24, size: 4},
		{name: "dt", kind: moneyColumnDateTime, fixed: true, offset: 28, size: 8},
		{name: "mMemo", kind: moneyColumnMemo, size: 200, variable: 0},
	}

	// moneyTransaction lays out one transaction row. A handle of zero leaves the
	// column out of the row, so it reads as null.
	moneyTransaction := func(handle, account, link, security, amount, category int64, memo []byte, notNull byte) []byte {
		fixed := make([]byte, 0, 36)
		fixed = append(fixed, moneyInt32(handle)...)
		fixed = append(fixed, moneyInt32(account)...)
		fixed = append(fixed, moneyInt32(link)...)
		fixed = append(fixed, moneyInt32(security)...)
		fixed = append(fixed, moneyAmount(amount)...)
		fixed = append(fixed, moneyInt32(category)...)
		fixed = append(fixed, moneyDate(2011, time.May, 19)...)

		return moneyTestRow(8, []byte{notNull}, fixed, memo)
	}

	notNull := func(columns ...int) byte {
		mask := byte(0)

		for _, column := range columns {
			mask |= 1 << column
		}

		return mask
	}

	pages = append(pages,
		moneyDefinitionPage(10, trnColumns),
		moneyDataPage(8, // 9: TRN rows
			moneyTransaction(1, 100, 0, 0, -250000, groceries, moneyMemo(moneyUTF16("Lunch")), notNull(0, 1, 4, 5, 6, 7)),
			moneyTransaction(2, 100, 0, 0, 150000, wages, nil, notNull(0, 1, 4, 5, 6)),
			moneyTransaction(3, 100, 200, 0, -50000, 0, nil, notNull(0, 1, 2, 4, 6)),
			moneyTransaction(4, 200, 100, 0, 50000, 0, nil, notNull(0, 1, 2, 4, 6)),
			moneyTransaction(5, 100, 0, 0, 0, groceries, nil, notNull(0, 1, 4, 5, 6)),
			moneyTransaction(6, 100, 0, 9, -750000, 0, nil, notNull(0, 1, 3, 4, 6)),
			moneyTransaction(7, 100, 0, 0, -7218000, 0, nil, notNull(0, 1, 4, 6)),
			moneyTransaction(8, 100, 200, 0, -1817300, principal, moneyMemo(append([]byte{0xFF, 0xFE}, moneyUTF16("Principal")...)), notNull(0, 1, 2, 4, 5, 6, 7)),
			moneyTransaction(9, 100, 0, 0, -5400700, interest, moneyMemo(moneyUTF16("Interest")), notNull(0, 1, 4, 5, 6, 7)),
			moneyTransaction(10, 999, 0, 0, -10000, groceries, nil, notNull(0, 1, 4, 5, 6)),
		),
		moneyDefinitionPage(2, []moneyTestColumn{ // 10: TRN_SPLIT
			{name: "htrn", kind: moneyColumnLongInt, fixed: true, offset: 0, size: 4},
			{name: "htrnParent", kind: moneyColumnLongInt, fixed: true, offset: 4, size: 4},
			{name: "iSplit", kind: moneyColumnLongInt, fixed: true, offset: 8, size: 4},
		}),
		moneyDataPage(10, // 11: TRN_SPLIT rows
			moneyTestRow(3, []byte{0x07}, append(append(moneyInt32(8), moneyInt32(7)...), moneyInt32(2)...)),
			moneyTestRow(3, []byte{0x07}, append(append(moneyInt32(9), moneyInt32(7)...), moneyInt32(3)...)),
		),
		moneyCatalogPage(map[string]int{ // 12
			"CRNC": 1, "ACCT": 4, "CAT": 6, "TRN": 8, "TRN_SPLIT": 10,
		}),
	)

	data := make([]byte, 4096*(len(pages)+1))
	copy(data, moneyHeader())

	for index, page := range pages {
		copy(data[(index+1)*4096:], page)
	}

	return data
}

func TestIsMicrosoftMoneyDataFile(t *testing.T) {
	assert.True(t, IsMicrosoftMoneyDataFile(moneyHeader()))
	assert.False(t, IsMicrosoftMoneyDataFile(append(make([]byte, 4), []byte("Standard Jet DB")...)))
	assert.False(t, IsMicrosoftMoneyDataFile([]byte("SQLite format 3\x00")))
	assert.False(t, IsMicrosoftMoneyDataFile([]byte("!Type:Bank\nD01/02/2024\nT-1.00\n^")))
	assert.False(t, IsMicrosoftMoneyDataFile(nil))
	assert.False(t, IsMicrosoftMoneyDataFile(moneyHeader()[:18]))
}

func TestMicrosoftMoneyTransactionDataFileParseImportedData(t *testing.T) {
	importer := MicrosoftMoneyTransactionDataFileImporter
	context := core.NewNullContext()

	user := &models.User{
		Uid:             1234567890,
		DefaultCurrency: "USD",
	}

	imported, accounts, expenseCategories, incomeCategories, transferCategories, tags, err := importer.ParseImportedData(context, user, moneyTestDatabase(), time.UTC, converter.DefaultImporterOptions, nil, nil, nil, nil, nil)
	require.NoError(t, err)
	require.Len(t, imported, 5)
	assert.Empty(t, tags)

	byDescription := make(map[string]*models.ImportTransaction, len(imported))

	for _, transaction := range imported {
		byDescription[transaction.Comment] = transaction
	}

	require.Len(t, byDescription, 4, "two transactions share an empty description")

	expense := byDescription["Lunch"]
	require.NotNil(t, expense, "the memo becomes the description")
	assert.Equal(t, models.TRANSACTION_DB_TYPE_EXPENSE, expense.Type)
	assert.Equal(t, int64(2500), expense.Amount)
	assert.Equal(t, "Groceries", expense.OriginalCategoryName)
	assert.Equal(t, "Checking", expense.OriginalSourceAccountName)

	var transfers []*models.ImportTransaction

	for _, transaction := range imported {
		if transaction.Type == models.TRANSACTION_DB_TYPE_TRANSFER_OUT {
			transfers = append(transfers, transaction)
		}
	}

	require.Len(t, transfers, 2)

	byAmount := make(map[int64]int64, len(transfers))

	for _, transfer := range transfers {
		byAmount[transfer.Amount] = transfer.RelatedAccountAmount
	}

	assert.Equal(t, int64(500), byAmount[int64(500)], "the receiving row's amount becomes the destination amount")
	assert.Equal(t, int64(18173), byAmount[int64(18173)], "a split fragment falls back to the source amount")

	assert.NotEmpty(t, accounts)
	assert.NotEmpty(t, expenseCategories)
	assert.NotEmpty(t, incomeCategories)
	assert.NotEmpty(t, transferCategories)
}

func TestMicrosoftMoneyTransactionDataFileParseImportedDataRejectsUnknownFormat(t *testing.T) {
	importer := MicrosoftMoneyTransactionDataFileImporter
	context := core.NewNullContext()

	user := &models.User{Uid: 1234567890, DefaultCurrency: "USD"}

	_, _, _, _, _, _, err := importer.ParseImportedData(context, user, []byte("!Type:Bank\nD01/02/2024\nT-1.00\n^"), time.UTC, converter.DefaultImporterOptions, nil, nil, nil, nil, nil)
	assert.Equal(t, errs.ErrImportFileTypeNotSupported, err)

	corrupt := moneyHeader()
	corrupt = append(corrupt, make([]byte, 4096)...)

	_, _, _, _, _, _, err = importer.ParseImportedData(context, user, corrupt, time.UTC, converter.DefaultImporterOptions, nil, nil, nil, nil, nil)
	assert.ErrorIs(t, err, errs.ErrImportMicrosoftMoneyDataFileNotReadable)
}
