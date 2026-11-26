// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

// Package export writes ExtractedData in the formats the applications ship:
// CSV, TSV, Honey Ledger CSV, XLSX, JSON and QIF. Both the Honey Ledger HB
// Money Extractor and HB SunRise call these writers, so every format is
// maintained in exactly one place.
//
// Amount conventions, shared by every writer:
//
//   - ExtractedTransaction.Amount is in hundredths of the currency and is
//     always positive; Type tells whether the money came in, went out or
//     moved to the transfer account.
//   - signedAmount gives the amount from the account's point of view
//     everywhere except the Honey Ledger CSV, where the amount column always
//     holds the positive amount.
//   - a transfer is written once with the receiving account, except in QIF,
//     where it appears in both accounts so the books balance.
//   - the Honey Ledger CSV uses ExtractedTransaction.TransferAmount when it
//     is set, so a cross-currency transfer carries the amount the receiving
//     account got; zero means the receiving amount is unknown and the source
//     amount is used.
package export

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/msmoney"
)

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

// spreadsheetSafe returns the text with a leading apostrophe added when it
// starts with a character a spreadsheet would treat as a formula (=, +, -, @,
// tab or carriage return), so a memo like "=HYPERLINK(...)" displays as text
// instead of executing when the CSV is opened. Amounts are not passed through
// here: they are rendered by formatAmount and always start with a digit or -.
func spreadsheetSafe(text string) string {
	trimmed := strings.TrimLeft(text, " \t\r")

	if trimmed != "" {
		switch trimmed[0] {
		case '=', '+', '-', '@', '\t', '\r':
			return "'" + text
		}
	}

	return text
}

// xlsxSafeText removes the characters an Excel cell cannot hold: control
// characters other than tab and line breaks, and the two non-characters that
// are invalid in XML.
func xlsxSafeText(text string) string {
	return strings.Map(func(r rune) rune {
		if r < 0x20 && r != '\t' && r != '\n' && r != '\r' {
			return -1
		}

		if r == 0xFFFE || r == 0xFFFF {
			return -1
		}

		return r
	}, text)
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
		row := []string{formatDate(t.Time), t.Type, spreadsheetSafe(t.Account), t.Currency, formatAmount(signedAmount(t)), spreadsheetSafe(t.TransferAccount), spreadsheetSafe(t.Category), spreadsheetSafe(t.SubCategory), spreadsheetSafe(t.Memo)}

		if err := writer.Write(row); err != nil {
			return err
		}
	}

	writer.Flush()

	return writer.Error()
}

func WriteCSV(w io.Writer, data *msmoney.ExtractedData) error {
	return writeDelimited(w, ',', true, data)
}

func WriteTSV(w io.Writer, data *msmoney.ExtractedData) error {
	return writeDelimited(w, '\t', true, data)
}

// writeHoneyLedgerCSV writes the layout Honey Ledger exports and imports, so
// the file can be imported without mapping any column.
func WriteHoneyLedgerCSV(w io.Writer, data *msmoney.ExtractedData) error {
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
			if t.TransferAmount != 0 {
				account2Amount = formatAmount(t.TransferAmount)
			} else {
				account2Amount = amount
			}
		}

		row := []string{t.Time.Format("2006-01-02 15:04:05"), timezone, t.Type, spreadsheetSafe(t.Category), spreadsheetSafe(t.SubCategory), spreadsheetSafe(t.Account), t.Currency, amount, spreadsheetSafe(t.TransferAccount), t.TransferCurrency, account2Amount, "", "", spreadsheetSafe(t.Memo)}

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
	Closed   bool   `json:"closed,omitempty"`
	Hidden   bool   `json:"hidden,omitempty"`
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
	Payee           string `json:"payee,omitempty"`
}

func WriteJSON(w io.Writer, source string, data *msmoney.ExtractedData) error {
	document := struct {
		Source       string            `json:"source"`
		Accounts     []jsonAccount     `json:"accounts"`
		Transactions []jsonTransaction `json:"transactions"`
		Skipped      int               `json:"skippedRows"`
	}{
		Source:       source,
		Accounts:     make([]jsonAccount, 0, len(data.Accounts)),
		Transactions: make([]jsonTransaction, 0, len(data.Transactions)),
		Skipped:      data.Skipped.Total(),
	}

	for _, a := range data.Accounts {
		document.Accounts = append(document.Accounts, jsonAccount{Name: a.Name, Currency: a.Currency, Closed: a.Closed, Hidden: a.Hidden})
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
			Payee:           t.Payee,
		})
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)

	return encoder.Encode(document)
}

func WriteXLSX(w io.Writer, data *msmoney.ExtractedData) error {
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
			xlsxSafeText(t.Type), xlsxSafeText(t.Account), xlsxSafeText(t.Currency),
			excelize.Cell{StyleID: amountStyle, Value: float64(signedAmount(t)) / 100},
			xlsxSafeText(t.TransferAccount), xlsxSafeText(t.Category), xlsxSafeText(t.SubCategory), xlsxSafeText(t.Memo),
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

		if err := accountStream.SetRow(cell, []any{xlsxSafeText(a.Name), xlsxSafeText(a.Currency)}); err != nil {
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
func WriteQIF(w io.Writer, data *msmoney.ExtractedData) error {
	type entry struct {
		time     time.Time
		amount   int64
		memo     string
		payee    string
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
			entries[t.Account] = append(entries[t.Account], entry{t.Time, -t.Amount, t.Memo, t.Payee, "[" + t.TransferAccount + "]"})
			entries[t.TransferAccount] = append(entries[t.TransferAccount], entry{t.Time, t.Amount, t.Memo, t.Payee, "[" + t.Account + "]"})

			continue
		}

		entries[t.Account] = append(entries[t.Account], entry{t.Time, signedAmount(t), t.Memo, t.Payee, categoryPath(t)})
	}

	sort.Strings(names)

	var out strings.Builder

	for _, name := range names {
		out.WriteString("!Account\nN" + qifText(name) + "\nTBank\n^\n!Type:Bank\n")

		for _, e := range entries[name] {
			out.WriteString("D" + e.time.Format("01/02/2006") + "\n")
			out.WriteString("T" + formatAmount(e.amount) + "\n")

			if e.payee != "" {
				out.WriteString("P" + qifText(e.payee) + "\n")

				if e.memo != "" {
					out.WriteString("M" + qifText(e.memo) + "\n")
				}
			} else if e.memo != "" {
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
