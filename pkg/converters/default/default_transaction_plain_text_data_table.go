// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package _default

import (
	"encoding/csv"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/datatable"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
)

// defaultPlainTextDataTable defines the structure of Honey Ledger default plain text data table
type defaultPlainTextDataTable struct {
	columnSeparator       string
	lineSeparator         string
	allRows               [][]string
	headerLineColumnNames []string
}

// defaultPlainTextDataRow defines the structure of Honey Ledger default plain text data row
type defaultPlainTextDataRow struct {
	allItems []string
}

// defaultPlainTextDataRowIterator defines the structure of Honey Ledger default plain text data row iterator
type defaultPlainTextDataRowIterator struct {
	dataTable    *defaultPlainTextDataTable
	currentIndex int
}

// defaultTransactionPlainTextDataTableBuilder defines the structure of Honey Ledger default transaction plain text data table builder
type defaultTransactionPlainTextDataTableBuilder struct {
	columnSeparator       string
	lineSeparator         string
	columns               []datatable.TransactionDataTableColumn
	dataColumnNameMapping map[datatable.TransactionDataTableColumn]string
	builder               *strings.Builder
}

// DataRowCount returns the total count of data row
func (t *defaultPlainTextDataTable) DataRowCount() int {
	if len(t.allRows) < 1 {
		return 0
	}

	return len(t.allRows) - 1
}

// HeaderColumnNames returns the header column name list
func (t *defaultPlainTextDataTable) HeaderColumnNames() []string {
	return t.headerLineColumnNames
}

// DataRowIterator returns the iterator of data row
func (t *defaultPlainTextDataTable) DataRowIterator() datatable.BasicDataTableRowIterator {
	return &defaultPlainTextDataRowIterator{
		dataTable:    t,
		currentIndex: 0,
	}
}

// ColumnCount returns the total count of column in this data row
func (r *defaultPlainTextDataRow) ColumnCount() int {
	return len(r.allItems)
}

// GetData returns the data in the specified column index
func (r *defaultPlainTextDataRow) GetData(columnIndex int) string {
	if columnIndex >= len(r.allItems) {
		return ""
	}

	return r.allItems[columnIndex]
}

// HasNext returns whether the iterator does not reach the end
func (t *defaultPlainTextDataRowIterator) HasNext() bool {
	return t.currentIndex+1 < len(t.dataTable.allRows)
}

// CurrentRowId returns current index
func (t *defaultPlainTextDataRowIterator) CurrentRowId() string {
	return fmt.Sprintf("line#%d", t.currentIndex)
}

// Next returns the next basic data row
func (t *defaultPlainTextDataRowIterator) Next() datatable.BasicDataTableRow {
	if t.currentIndex+1 >= len(t.dataTable.allRows) {
		return nil
	}

	t.currentIndex++

	return &defaultPlainTextDataRow{
		allItems: t.dataTable.allRows[t.currentIndex],
	}
}

// AppendTransaction appends the specified transaction to data builder
func (b *defaultTransactionPlainTextDataTableBuilder) AppendTransaction(data map[datatable.TransactionDataTableColumn]string) {
	for i := 0; i < len(b.columns); i++ {
		if i > 0 {
			b.builder.WriteString(b.columnSeparator)
		}

		b.builder.WriteString(escapePlainTextValue(data[b.columns[i]], b.columnSeparator))
	}

	b.builder.WriteString(b.lineSeparator)
}

// ReplaceDelimiters returns the text unchanged: AppendTransaction quotes any
// field that contains a separator, a quote or a line break, so the text
// survives a round trip instead of being silently altered.
func (b *defaultTransactionPlainTextDataTableBuilder) ReplaceDelimiters(text string) string {
	return text
}

// escapePlainTextValue quotes a field following RFC 4180.
func escapePlainTextValue(value string, columnSeparator string) string {
	if strings.Contains(value, columnSeparator) || strings.ContainsAny(value, "\"\r\n") {
		return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
	}

	return value
}

// String returns the textual representation of this data
func (b *defaultTransactionPlainTextDataTableBuilder) String() string {
	return b.builder.String()
}

func (b *defaultTransactionPlainTextDataTableBuilder) generateHeaderLine() string {
	var ret strings.Builder

	for i := 0; i < len(b.columns); i++ {
		if ret.Len() > 0 {
			ret.WriteString(b.columnSeparator)
		}

		dataColumn := b.columns[i]
		columnName := b.dataColumnNameMapping[dataColumn]

		ret.WriteString(escapePlainTextValue(columnName, b.columnSeparator))
	}

	ret.WriteString(b.lineSeparator)

	return ret.String()
}

func createNewDefaultPlainTextDataTable(content string, columnSeparator string, lineSeparator string) (*defaultPlainTextDataTable, error) {
	separator, size := utf8.DecodeRuneInString(columnSeparator)

	if size < 1 || separator == utf8.RuneError {
		return nil, errs.ErrNotFoundTransactionDataInFile
	}

	csvReader := csv.NewReader(strings.NewReader(content))
	csvReader.Comma = separator
	csvReader.FieldsPerRecord = -1
	// Files written by older versions did not quote fields, so a stray quote
	// must not make the whole file unreadable.
	csvReader.LazyQuotes = true

	allRows, err := csvReader.ReadAll()

	if err != nil || len(allRows) < 2 {
		return nil, errs.ErrNotFoundTransactionDataInFile
	}

	// A byte order mark left by an external editor would otherwise become part
	// of the first column name.
	allRows[0][0] = strings.TrimPrefix(allRows[0][0], "\ufeff")

	return &defaultPlainTextDataTable{
		columnSeparator:       columnSeparator,
		lineSeparator:         lineSeparator,
		allRows:               allRows,
		headerLineColumnNames: allRows[0],
	}, nil
}

func createNewDefaultTransactionPlainTextDataTableBuilder(transactionCount int, columns []datatable.TransactionDataTableColumn, dataColumnNameMapping map[datatable.TransactionDataTableColumn]string, columnSeparator string, lineSeparator string) *defaultTransactionPlainTextDataTableBuilder {
	var builder strings.Builder
	builder.Grow(transactionCount * 100)

	dataTableBuilder := &defaultTransactionPlainTextDataTableBuilder{
		columnSeparator:       columnSeparator,
		lineSeparator:         lineSeparator,
		columns:               columns,
		dataColumnNameMapping: dataColumnNameMapping,
		builder:               &builder,
	}

	dataTableBuilder.builder.WriteString(dataTableBuilder.generateHeaderLine())

	return dataTableBuilder
}
