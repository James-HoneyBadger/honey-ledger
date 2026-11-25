// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package export

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/msmoney"
)

func sampleData() *msmoney.ExtractedData {
	day := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)

	return &msmoney.ExtractedData{
		Accounts: []msmoney.ExtractedAccount{{Name: "Checking", Currency: "USD"}, {Name: "Savings", Currency: "USD"}, {Name: "Unused", Currency: "USD"}},
		Transactions: []msmoney.ExtractedTransaction{
			{Time: day, Type: msmoney.ExtractedTypeExpense, Account: "Checking", Currency: "USD", Amount: 1234, Category: "Food", SubCategory: "Groceries", Memo: "Milk, \"2%\"\nand eggs"},
			{Time: day, Type: msmoney.ExtractedTypeIncome, Account: "Checking", Currency: "USD", Amount: 500000, Category: "Salary", SubCategory: "Pay"},
			{Time: day, Type: msmoney.ExtractedTypeTransfer, Account: "Checking", Currency: "USD", Amount: 10000, TransferAccount: "Savings", TransferCurrency: "USD"},
		},
	}
}

func TestFormatAmount(t *testing.T) {
	assert.Equal(t, "12.34", formatAmount(1234))
	assert.Equal(t, "-0.05", formatAmount(-5))
	assert.Equal(t, "0.00", formatAmount(0))
}

func TestCSVKeepsAmountsSignedAndQuotesText(t *testing.T) {
	var out bytes.Buffer
	assert.NoError(t, WriteCSV(&out, sampleData()))

	rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(out.String(), "\ufeff"))).ReadAll()
	assert.NoError(t, err)
	assert.Equal(t, 4, len(rows))
	assert.Equal(t, "-12.34", rows[1][4])
	assert.Equal(t, "5000.00", rows[2][4])
	assert.Equal(t, "-100.00", rows[3][4])
	assert.Equal(t, "Savings", rows[3][5])
	assert.Equal(t, "Milk, \"2%\"\nand eggs", rows[1][8])
}

func TestHoneyLedgerCSVHasImportableLayout(t *testing.T) {
	var out bytes.Buffer
	assert.NoError(t, WriteHoneyLedgerCSV(&out, sampleData()))

	rows, err := csv.NewReader(&out).ReadAll()
	assert.NoError(t, err)
	assert.Equal(t, "Sub Category", rows[0][4])
	assert.Equal(t, "2024-03-05 00:00:00", rows[1][0])
	assert.Equal(t, "+00:00", rows[1][1])
	assert.Equal(t, "12.34", rows[1][7])
	assert.Equal(t, "Transfer", rows[3][2])
	assert.Equal(t, "Savings", rows[3][8])
	assert.Equal(t, "100.00", rows[3][10])
}

func TestHoneyLedgerCSVCarriesTheTransferAmountAcrossCurrencies(t *testing.T) {
	data := sampleData()
	data.Transactions[2].TransferCurrency = "EUR"
	data.Transactions[2].TransferAmount = 9160

	var out bytes.Buffer
	assert.NoError(t, WriteHoneyLedgerCSV(&out, data))

	rows, err := csv.NewReader(&out).ReadAll()
	assert.NoError(t, err)
	assert.Equal(t, "91.60", rows[3][10], "the receiving side's amount becomes the account2 amount")
}

func TestDelimitedWritersNeutralizeSpreadsheetFormulas(t *testing.T) {
	data := sampleData()
	data.Transactions[0].Memo = "=HYPERLINK(\"http://example.com\")"

	var out bytes.Buffer
	assert.NoError(t, WriteCSV(&out, data))

	rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(out.String(), "\ufeff"))).ReadAll()
	assert.NoError(t, err)
	assert.Equal(t, "'=HYPERLINK(\"http://example.com\")", rows[1][8], "a memo that a spreadsheet would run as a formula is neutralized")
}

func TestJSONHasAccountsAndTransactions(t *testing.T) {
	var out bytes.Buffer
	assert.NoError(t, WriteJSON(&out, "test 1.2.3", sampleData()))

	var document struct {
		Source       string           `json:"source"`
		Accounts     []map[string]any `json:"accounts"`
		Transactions []map[string]any `json:"transactions"`
		Skipped      int              `json:"skippedRows"`
	}

	assert.NoError(t, json.Unmarshal(out.Bytes(), &document))
	assert.Equal(t, "test 1.2.3", document.Source)
	assert.Equal(t, 3, len(document.Accounts))
	assert.Equal(t, 3, len(document.Transactions))
	assert.Equal(t, "-12.34", document.Transactions[0]["amount"])
}

func TestQIFWritesBothSidesOfATransfer(t *testing.T) {
	var out bytes.Buffer
	assert.NoError(t, WriteQIF(&out, sampleData()))
	text := out.String()

	assert.Equal(t, 3, strings.Count(text, "!Account"))
	assert.Contains(t, text, "D03/05/2024\nT-100.00\nL[Savings]\n^")
	assert.Contains(t, text, "D03/05/2024\nT100.00\nL[Checking]\n^")
	assert.Contains(t, text, "LFood:Groceries")
	assert.Contains(t, text, "PMilk, \"2%\" and eggs")
}

func TestXLSXHasBothSheets(t *testing.T) {
	var out bytes.Buffer
	assert.NoError(t, WriteXLSX(&out, sampleData()))

	file, err := excelize.OpenReader(&out)
	assert.NoError(t, err)

	defer file.Close()

	rows, err := file.GetRows("Transactions")
	assert.NoError(t, err)
	assert.Equal(t, 4, len(rows))

	accounts, err := file.GetRows("Accounts")
	assert.NoError(t, err)
	assert.Equal(t, 4, len(accounts))
}

func TestCSVNeutralizesFormulaPrefixes(t *testing.T) {
	day := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)
	data := &msmoney.ExtractedData{
		Accounts: []msmoney.ExtractedAccount{{Name: "Checking", Currency: "USD"}},
		Transactions: []msmoney.ExtractedTransaction{
			{Time: day, Type: msmoney.ExtractedTypeExpense, Account: "=cmd|' /C calc'!A0", Currency: "USD", Amount: 1234, Category: "+Food", SubCategory: "@Sub", Memo: "=HYPERLINK(\"http://evil\")"},
		},
	}

	var out bytes.Buffer
	assert.NoError(t, WriteCSV(&out, data))

	rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(out.String(), "\ufeff"))).ReadAll()
	assert.NoError(t, err)
	assert.Equal(t, "'=cmd|' /C calc'!A0", rows[1][2])
	assert.Equal(t, "-12.34", rows[1][4], "amounts keep their sign and are never prefixed")
	assert.Equal(t, "'+Food", rows[1][6])
	assert.Equal(t, "'@Sub", rows[1][7])
	assert.Equal(t, "'=HYPERLINK(\"http://evil\")", rows[1][8])
}

func TestHoneyLedgerCSVNeutralizesFormulaPrefixes(t *testing.T) {
	day := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)
	data := &msmoney.ExtractedData{
		Accounts: []msmoney.ExtractedAccount{{Name: "Checking", Currency: "USD"}},
		Transactions: []msmoney.ExtractedTransaction{
			{Time: day, Type: msmoney.ExtractedTypeExpense, Account: "=cmd|' /C calc'!A0", Currency: "USD", Amount: 1234, Category: "+Food", SubCategory: "@Sub", Memo: "=HYPERLINK(\"http://evil\")"},
		},
	}

	var out bytes.Buffer
	assert.NoError(t, WriteHoneyLedgerCSV(&out, data))

	rows, err := csv.NewReader(&out).ReadAll()
	assert.NoError(t, err)
	assert.Equal(t, "'+Food", rows[1][3])
	assert.Equal(t, "'@Sub", rows[1][4])
	assert.Equal(t, "'=cmd|' /C calc'!A0", rows[1][5])
	assert.Equal(t, "12.34", rows[1][7], "amounts are never prefixed")
	assert.Equal(t, "'=HYPERLINK(\"http://evil\")", rows[1][13])
}

func TestXLSXStripsInvalidCharacters(t *testing.T) {
	day := time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)
	data := &msmoney.ExtractedData{
		Accounts: []msmoney.ExtractedAccount{{Name: "Bad\x01Name\uFFFE", Currency: "USD"}},
		Transactions: []msmoney.ExtractedTransaction{
			{Time: day, Type: msmoney.ExtractedTypeExpense, Account: "Checking", Currency: "USD", Amount: 1234, Memo: "line1\x01line2\nline3\uFFFF"},
		},
	}

	var out bytes.Buffer
	assert.NoError(t, WriteXLSX(&out, data))

	file, err := excelize.OpenReader(&out)
	assert.NoError(t, err)

	defer file.Close()

	rows, err := file.GetRows("Transactions")
	assert.NoError(t, err)
	assert.Equal(t, "line1line2\nline3", rows[1][8])

	accounts, err := file.GetRows("Accounts")
	assert.NoError(t, err)
	assert.Equal(t, "BadName", accounts[1][0])
}

func TestRegistryListsEveryWriter(t *testing.T) {
	formats := Formats()
	require.Len(t, formats, 6)

	keys := make([]string, 0, len(formats))

	for _, format := range formats {
		keys = append(keys, format.Key)
	}

	assert.Equal(t, []string{"csv", "honeyledger", "json", "qif", "tsv", "xlsx"}, keys)

	var out bytes.Buffer
	require.NoError(t, Write("honeyledger", &out, "test", sampleData()))
	assert.Contains(t, out.String(), "Sub Category")

	err := Write("pdf", &out, "test", sampleData())
	assert.Error(t, err)
}
