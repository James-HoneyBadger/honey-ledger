// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bufio"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/msmoney"
)

// outputFormat is a file format the extracted data can be written in
type outputFormat struct {
	key         string
	name        string
	extension   string
	description string
	write       func(w io.Writer, data *msmoney.ExtractedData) error
}

var allFormats = []outputFormat{
	{"csv", "CSV", ".csv", "Spreadsheet-friendly, opens in Excel (comma-separated, UTF-8)", writeCSV},
	{"honeyledger", "Honey Ledger CSV", ".honeyledger.csv", "Ready to import into Honey Ledger", writeHoneyLedgerCSV},
	{"tsv", "TSV", ".tsv", "Tab-separated values", writeTSV},
	{"xlsx", "Excel workbook", ".xlsx", "Transactions and Accounts sheets", writeXLSX},
	{"json", "JSON", ".json", "Accounts and transactions as structured data", writeJSON},
	{"qif", "QIF", ".qif", "Quicken Interchange Format, for Quicken and most finance apps", writeQIF},
}

func formatKeys() string {
	keys := make([]string, len(allFormats))

	for i, f := range allFormats {
		keys[i] = f.key
	}

	return strings.Join(keys, ", ")
}

func printFormats(w io.Writer) {
	for i, f := range allFormats {
		fmt.Fprintf(w, "  %d) %-17s %-14s %s\n", i+1, f.name, "["+f.key+"]", f.description)
	}
}

// parseFormats turns a comma separated list of format keys into formats. An
// empty list returns no formats, which means the user is asked.
func parseFormats(list string) ([]outputFormat, error) {
	list = strings.TrimSpace(list)

	if list == "" {
		return nil, nil
	}

	var selected []outputFormat
	seen := map[string]bool{}

	for _, key := range strings.Split(list, ",") {
		key = strings.ToLower(strings.TrimSpace(key))

		if key == "all" {
			return allFormats, nil
		}

		found := false

		for _, f := range allFormats {
			if f.key == key {
				found = true

				if !seen[key] {
					seen[key] = true
					selected = append(selected, f)
				}
			}
		}

		if !found {
			return nil, fmt.Errorf("unknown format %q; choose from %s, or all", key, formatKeys())
		}
	}

	return selected, nil
}

// writeFormat writes the data to a new file in dir and returns its path. An
// existing file is never overwritten; a number is added to the name instead.
func writeFormat(f outputFormat, dir string, baseName string, data *msmoney.ExtractedData) (string, error) {
	path := uniquePath(filepath.Join(dir, baseName+f.extension))

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)

	if err != nil {
		return "", err
	}

	buffered := bufio.NewWriterSize(file, 1<<20)
	err = f.write(buffered, data)

	if err == nil {
		err = buffered.Flush()
	}

	if closeErr := file.Close(); err == nil {
		err = closeErr
	}

	if err != nil {
		os.Remove(path)
		return "", err
	}

	return path, nil
}

func uniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	// The extension can have two parts (".honeyledger.csv"), so the name is
	// split at the first dot of the file name instead of the last.
	dir, name := filepath.Split(path)
	stem, extension := name, ""

	if i := strings.Index(name, "."); i > 0 {
		stem, extension = name[:i], name[i:]
	}

	for n := 1; ; n++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", stem, n, extension))

		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

// formatAmount renders an amount in hundredths as a decimal number
func formatAmount(minor int64) string {
	sign := ""

	if minor < 0 {
		sign = "-"
		minor = -minor
	}

	return fmt.Sprintf("%s%d.%02d", sign, minor/100, minor%100)
}

// signedAmount is the amount as the account of the transaction sees it: money
// that comes in is positive, money that goes out or moves away is negative.
func signedAmount(t msmoney.ExtractedTransaction) int64 {
	if t.Type == msmoney.ExtractedTypeIncome {
		return t.Amount
	}

	return -t.Amount
}

func formatDate(t time.Time) string {
	return t.Format("2006-01-02")
}

func categoryPath(t msmoney.ExtractedTransaction) string {
	if t.Category != "" && t.SubCategory != "" {
		return t.Category + ":" + t.SubCategory
	}

	if t.SubCategory != "" {
		return t.SubCategory
	}

	return t.Category
}

func writeDelimited(w io.Writer, comma rune, withByteOrderMark bool, data *msmoney.ExtractedData) error {
	if withByteOrderMark {
		// Excel needs the mark to read accented characters correctly.
		if _, err := io.WriteString(w, "\ufeff"); err != nil {
			return err
		}
	}

	writer := csv.NewWriter(w)
	writer.Comma = comma

	header := []string{"Date", "Type", "Account", "Currency", "Amount", "Transfer Account", "Category", "Subcategory", "Memo"}

	if err := writer.Write(header); err != nil {
		return err
	}

	for _, t := range data.Transactions {
		row := []string{formatDate(t.Time), t.Type, t.Account, t.Currency, formatAmount(signedAmount(t)), t.TransferAccount, t.Category, t.SubCategory, t.Memo}

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	writer.Flush()

	return writer.Error()
}

func writeCSV(w io.Writer, data *msmoney.ExtractedData) error {
	return writeDelimited(w, ',', true, data)
}

func writeTSV(w io.Writer, data *msmoney.ExtractedData) error {
	return writeDelimited(w, '\t', true, data)
}

// writeHoneyLedgerCSV writes the layout Honey Ledger exports and imports, so
// the file can be imported without mapping any column.
func writeHoneyLedgerCSV(w io.Writer, data *msmoney.ExtractedData) error {
	writer := csv.NewWriter(w)

	header := []string{"Time", "Timezone", "Type", "Category", "Sub Category", "Account", "Account Currency", "Amount", "Account2", "Account2 Currency", "Account2 Amount", "Geographic Location", "Tags", "Description"}

	if err := writer.Write(header); err != nil {
		return err
	}

	for _, t := range data.Transactions {
		_, offset := t.Time.Zone()
		timezone := fmt.Sprintf("%+03d:%02d", offset/3600, absInt(offset%3600)/60)
		amount := formatAmount(t.Amount)
		account2Amount := ""

		if t.Type == msmoney.ExtractedTypeTransfer {
			account2Amount = amount
		}

		row := []string{t.Time.Format("2006-01-02 15:04:05"), timezone, t.Type, t.Category, t.SubCategory, t.Account, t.Currency, amount, t.TransferAccount, t.TransferCurrency, account2Amount, "", "", t.Memo}

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	writer.Flush()

	return writer.Error()
}

func absInt(value int) int {
	if value < 0 {
		return -value
	}

	return value
}

type jsonAccount struct {
	Name     string `json:"name"`
	Currency string `json:"currency"`
}

type jsonTransaction struct {
	Date            string `json:"date"`
	Type            string `json:"type"`
	Account         string `json:"account"`
	Currency        string `json:"currency"`
	Amount          string `json:"amount"`
	TransferAccount string `json:"transferAccount,omitempty"`
	Category        string `json:"category,omitempty"`
	Subcategory     string `json:"subcategory,omitempty"`
	Memo            string `json:"memo,omitempty"`
}

func writeJSON(w io.Writer, data *msmoney.ExtractedData) error {
	document := struct {
		Source       string            `json:"source"`
		Accounts     []jsonAccount     `json:"accounts"`
		Transactions []jsonTransaction `json:"transactions"`
		Skipped      int               `json:"skippedRows"`
	}{
		Source:       appName + " " + appVersion,
		Accounts:     make([]jsonAccount, 0, len(data.Accounts)),
		Transactions: make([]jsonTransaction, 0, len(data.Transactions)),
		Skipped:      data.Skipped,
	}

	for _, a := range data.Accounts {
		document.Accounts = append(document.Accounts, jsonAccount{Name: a.Name, Currency: a.Currency})
	}

	for _, t := range data.Transactions {
		document.Transactions = append(document.Transactions, jsonTransaction{
			Date:            formatDate(t.Time),
			Type:            t.Type,
			Account:         t.Account,
			Currency:        t.Currency,
			Amount:          formatAmount(signedAmount(t)),
			TransferAccount: t.TransferAccount,
			Category:        t.Category,
			Subcategory:     t.SubCategory,
			Memo:            t.Memo,
		})
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(document)
}

func writeXLSX(w io.Writer, data *msmoney.ExtractedData) error {
	file := excelize.NewFile()

	defer file.Close()

	dateStyle, err := file.NewStyle(&excelize.Style{NumFmt: 14})

	if err != nil {
		return err
	}

	amountStyle, err := file.NewStyle(&excelize.Style{NumFmt: 4})

	if err != nil {
		return err
	}

	boldStyle, err := file.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})

	if err != nil {
		return err
	}

	const transactionsSheet, accountsSheet = "Transactions", "Accounts"

	if err := file.SetSheetName("Sheet1", transactionsSheet); err != nil {
		return err
	}

	if _, err := file.NewSheet(accountsSheet); err != nil {
		return err
	}

	stream, err := file.NewStreamWriter(transactionsSheet)

	if err != nil {
		return err
	}

	if err := stream.SetColWidth(1, 1, 12); err != nil {
		return err
	}

	if err := stream.SetColWidth(3, 3, 28); err != nil {
		return err
	}

	if err := stream.SetColWidth(6, 9, 24); err != nil {
		return err
	}

	headers := []string{"Date", "Type", "Account", "Currency", "Amount", "Transfer Account", "Category", "Subcategory", "Memo"}
	headerRow := make([]any, len(headers))

	for i, h := range headers {
		headerRow[i] = excelize.Cell{StyleID: boldStyle, Value: h}
	}

	if err := stream.SetRow("A1", headerRow); err != nil {
		return err
	}

	for i, t := range data.Transactions {
		cell, err := excelize.CoordinatesToCellName(1, i+2)

		if err != nil {
			return err
		}

		row := []any{
			excelize.Cell{StyleID: dateStyle, Value: time.Date(t.Time.Year(), t.Time.Month(), t.Time.Day(), 0, 0, 0, 0, time.UTC)},
			t.Type, t.Account, t.Currency,
			excelize.Cell{StyleID: amountStyle, Value: float64(signedAmount(t)) / 100},
			t.TransferAccount, t.Category, t.SubCategory, t.Memo,
		}

		if err := stream.SetRow(cell, row); err != nil {
			return err
		}
	}

	if err := stream.Flush(); err != nil {
		return err
	}

	accountStream, err := file.NewStreamWriter(accountsSheet)

	if err != nil {
		return err
	}

	if err := accountStream.SetColWidth(1, 1, 36); err != nil {
		return err
	}

	if err := accountStream.SetRow("A1", []any{excelize.Cell{StyleID: boldStyle, Value: "Account"}, excelize.Cell{StyleID: boldStyle, Value: "Currency"}}); err != nil {
		return err
	}

	for i, a := range data.Accounts {
		cell, err := excelize.CoordinatesToCellName(1, i+2)

		if err != nil {
			return err
		}

		if err := accountStream.SetRow(cell, []any{a.Name, a.Currency}); err != nil {
			return err
		}
	}

	if err := accountStream.Flush(); err != nil {
		return err
	}

	return file.Write(w)
}

// qifText makes a value safe for a QIF line, which cannot hold a line break.
func qifText(value string) string {
	value = strings.ReplaceAll(value, "\r\n", " ")
	value = strings.ReplaceAll(value, "\r", " ")

	return strings.ReplaceAll(value, "\n", " ")
}

// writeQIF writes one section per account. A transfer appears in both accounts,
// as an outflow in the source and an inflow in the destination, so the books
// balance when the file is imported.
func writeQIF(w io.Writer, data *msmoney.ExtractedData) error {
	type entry struct {
		time     time.Time
		amount   int64
		memo     string
		category string
	}

	entries := map[string][]entry{}
	names := make([]string, 0, len(data.Accounts))
	known := map[string]bool{}

	addAccount := func(name string) {
		if !known[name] {
			known[name] = true
			names = append(names, name)
		}
	}

	for _, a := range data.Accounts {
		addAccount(a.Name)
	}

	for _, t := range data.Transactions {
		addAccount(t.Account)

		if t.Type == msmoney.ExtractedTypeTransfer {
			addAccount(t.TransferAccount)
			entries[t.Account] = append(entries[t.Account], entry{t.Time, -t.Amount, t.Memo, "[" + t.TransferAccount + "]"})
			entries[t.TransferAccount] = append(entries[t.TransferAccount], entry{t.Time, t.Amount, t.Memo, "[" + t.Account + "]"})

			continue
		}

		entries[t.Account] = append(entries[t.Account], entry{t.Time, signedAmount(t), t.Memo, categoryPath(t)})
	}

	sort.Strings(names)

	var out strings.Builder

	for _, name := range names {
		out.WriteString("!Account\nN" + qifText(name) + "\nTBank\n^\n!Type:Bank\n")

		for _, e := range entries[name] {
			out.WriteString("D" + e.time.Format("01/02/2006") + "\n")
			out.WriteString("T" + formatAmount(e.amount) + "\n")

			if e.memo != "" {
				out.WriteString("P" + qifText(e.memo) + "\nM" + qifText(e.memo) + "\n")
			}

			if e.category != "" {
				out.WriteString("L" + qifText(e.category) + "\n")
			}

			out.WriteString("^\n")
		}

		if _, err := io.WriteString(w, out.String()); err != nil {
			return err
		}

		out.Reset()
	}

	return nil
}
