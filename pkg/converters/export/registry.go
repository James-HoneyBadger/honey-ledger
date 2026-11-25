// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package export

import (
	"fmt"
	"io"
	"sort"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/msmoney"
)

// Format describes one output format, for menus and help texts.
type Format struct {
	Key         string
	Name        string
	Extension   string
	Description string
}

var formats = map[string]Format{
	"csv":         {Key: "csv", Name: "CSV", Extension: ".csv", Description: "Spreadsheet-friendly, opens in Excel (comma-separated, UTF-8)"},
	"honeyledger": {Key: "honeyledger", Name: "Honey Ledger CSV", Extension: ".honeyledger.csv", Description: "Ready to import into Honey Ledger"},
	"tsv":         {Key: "tsv", Name: "TSV", Extension: ".tsv", Description: "Tab-separated values"},
	"xlsx":        {Key: "xlsx", Name: "Excel workbook", Extension: ".xlsx", Description: "Transactions and Accounts sheets"},
	"json":        {Key: "json", Name: "JSON", Extension: ".json", Description: "Accounts and transactions as structured data"},
	"qif":         {Key: "qif", Name: "QIF", Extension: ".qif", Description: "Quicken Interchange Format, for Quicken and most finance apps"},
}

// Formats returns every output format, ordered by key, so applications build
// their format menus from one list and never diverge.
func Formats() []Format {
	result := make([]Format, 0, len(formats))

	for _, format := range formats {
		result = append(result, format)
	}

	sort.Slice(result, func(left, right int) bool {
		return result[left].Key < result[right].Key
	})

	return result
}

// FormatByKey returns the format with the specified key, or an error.
func FormatByKey(key string) (Format, error) {
	format, exists := formats[key]

	if !exists {
		return Format{}, fmt.Errorf("unknown format %q", key)
	}

	return format, nil
}

// Write writes the data in the format with the specified key. source names the
// application in the JSON output, for example "HB SunRise 1.0.0".
func Write(key string, w io.Writer, source string, data *msmoney.ExtractedData) error {
	switch key {
	case "csv":
		return WriteCSV(w, data)
	case "tsv":
		return WriteTSV(w, data)
	case "honeyledger":
		return WriteHoneyLedgerCSV(w, data)
	case "xlsx":
		return WriteXLSX(w, data)
	case "json":
		return WriteJSON(w, source, data)
	case "qif":
		return WriteQIF(w, data)
	default:
		return fmt.Errorf("unknown format %q", key)
	}
}
