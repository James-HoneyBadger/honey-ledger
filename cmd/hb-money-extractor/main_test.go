// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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

func TestParseFormats(t *testing.T) {
	formats, err := parseFormats("csv, JSON,csv")
	assert.Nil(t, err)
	assert.Equal(t, 2, len(formats))

	formats, err = parseFormats("all")
	assert.Nil(t, err)
	assert.Equal(t, len(allFormats), len(formats))

	_, err = parseFormats("pdf")
	assert.NotNil(t, err)
}

func TestFormatAmount(t *testing.T) {
	assert.Equal(t, "12.34", formatAmount(1234))
	assert.Equal(t, "-0.05", formatAmount(-5))
	assert.Equal(t, "0.00", formatAmount(0))
}

func TestCSVKeepsAmountsSignedAndQuotesText(t *testing.T) {
	var out bytes.Buffer
	assert.Nil(t, writeCSV(&out, sampleData()))

	rows, err := csv.NewReader(strings.NewReader(strings.TrimPrefix(out.String(), "\ufeff"))).ReadAll()
	assert.Nil(t, err)
	assert.Equal(t, 4, len(rows))
	assert.Equal(t, "-12.34", rows[1][4])
	assert.Equal(t, "5000.00", rows[2][4])
	assert.Equal(t, "-100.00", rows[3][4])
	assert.Equal(t, "Savings", rows[3][5])
	assert.Equal(t, "Milk, \"2%\"\nand eggs", rows[1][8])
}

func TestHoneyLedgerCSVHasImportableLayout(t *testing.T) {
	var out bytes.Buffer
	assert.Nil(t, writeHoneyLedgerCSV(&out, sampleData()))

	rows, err := csv.NewReader(&out).ReadAll()
	assert.Nil(t, err)
	assert.Equal(t, "Sub Category", rows[0][4])
	assert.Equal(t, "2024-03-05 00:00:00", rows[1][0])
	assert.Equal(t, "+00:00", rows[1][1])
	assert.Equal(t, "12.34", rows[1][7])
	assert.Equal(t, "Transfer", rows[3][2])
	assert.Equal(t, "Savings", rows[3][8])
	assert.Equal(t, "100.00", rows[3][10])
}

func TestJSONHasAccountsAndTransactions(t *testing.T) {
	var out bytes.Buffer
	assert.Nil(t, writeJSON(&out, sampleData()))

	var document struct {
		Accounts     []map[string]any `json:"accounts"`
		Transactions []map[string]any `json:"transactions"`
	}

	assert.Nil(t, json.Unmarshal(out.Bytes(), &document))
	assert.Equal(t, 3, len(document.Accounts))
	assert.Equal(t, 3, len(document.Transactions))
	assert.Equal(t, "-12.34", document.Transactions[0]["amount"])
}

func TestQIFWritesBothSidesOfATransfer(t *testing.T) {
	var out bytes.Buffer
	assert.Nil(t, writeQIF(&out, sampleData()))
	text := out.String()

	assert.Equal(t, 3, strings.Count(text, "!Account"))
	assert.Contains(t, text, "D03/05/2024\nT-100.00\nL[Savings]\n^")
	assert.Contains(t, text, "D03/05/2024\nT100.00\nL[Checking]\n^")
	assert.Contains(t, text, "LFood:Groceries")
	assert.Contains(t, text, "PMilk, \"2%\" and eggs")
}

func TestXLSXHasBothSheets(t *testing.T) {
	var out bytes.Buffer
	assert.Nil(t, writeXLSX(&out, sampleData()))

	file, err := excelize.OpenReader(&out)
	assert.Nil(t, err)

	defer file.Close()

	rows, err := file.GetRows("Transactions")
	assert.Nil(t, err)
	assert.Equal(t, 4, len(rows))

	accounts, err := file.GetRows("Accounts")
	assert.Nil(t, err)
	assert.Equal(t, 4, len(accounts))
}

func TestWriteFormatNeverOverwrites(t *testing.T) {
	dir := t.TempDir()
	csvFormat := allFormats[0]

	first, err := writeFormat(csvFormat, dir, "money", sampleData())
	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(dir, "money.csv"), first)

	second, err := writeFormat(csvFormat, dir, "money", sampleData())
	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(dir, "money (1).csv"), second)

	hl, err := writeFormat(allFormats[1], dir, "money", sampleData())
	assert.Nil(t, err)
	hl2, err := writeFormat(allFormats[1], dir, "money", sampleData())
	assert.Nil(t, err)
	assert.Equal(t, filepath.Join(dir, "money.honeyledger.csv"), hl)
	assert.Equal(t, filepath.Join(dir, "money (1).honeyledger.csv"), hl2)
}

func TestRunRejectsInvalidInput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	dir := t.TempDir()

	assert.Equal(t, 2, run([]string{"-f", "pdf", "x.mny"}, strings.NewReader(""), &stdout, &stderr))
	assert.Equal(t, 1, run([]string{"-f", "csv", filepath.Join(dir, "missing.mny")}, strings.NewReader(""), &stdout, &stderr))

	notMoney := filepath.Join(dir, "not.mny")
	assert.Nil(t, os.WriteFile(notMoney, []byte("hello"), 0o644))
	stderr.Reset()
	assert.Equal(t, 1, run([]string{"-f", "csv", notMoney}, strings.NewReader(""), &stdout, &stderr))
	assert.Contains(t, stderr.String(), "not a Microsoft Money data file")
}
