// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

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

func TestVersionFlagReportsTheBuildVersion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	original := version
	version = "2.1.0"

	defer func() { version = original }()

	assert.Equal(t, 0, run([]string{"-version"}, strings.NewReader(""), &stdout, &stderr))
	assert.Equal(t, "HB Money Extractor 2.1.0\n", stdout.String())
}
