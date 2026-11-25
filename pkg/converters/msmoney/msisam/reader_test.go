// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msisam

import (
	"bytes"
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testColumn describes a column for the synthetic definition builder.
type testColumn struct {
	name     string
	kind     ColumnType
	fixed    bool
	offset   int
	size     int
	variable int
}

// definitionPages lays out a table definition, splitting it over continuation
// pages when it is larger than one page. The definition starts on the first
// page of the result.
func definitionPages(rowCount, firstDataPage int, columns []testColumn) [][]byte {
	variableCount := 0

	for _, column := range columns {
		if !column.fixed {
			variableCount++
		}
	}

	stream := make([]byte, definitionColumnsOffset)
	stream[0] = PageTypeDefinition
	putU32(stream, definitionRowCountOffset, uint32(rowCount))
	putU16(stream, definitionVariableColumnCountOffset, variableCount)
	putU16(stream, definitionColumnCountOffset, len(columns))
	putU16(stream, definitionFirstDataPageOffset, firstDataPage)

	for index, column := range columns {
		entry := make([]byte, definitionColumnEntrySize)
		entry[columnTypeOffset] = byte(column.kind)
		putU16(entry, columnNumberOffset, index)
		putU16(entry, columnVariableIndexOffset, column.variable)
		putU16(entry, columnFixedOffsetOffset, column.offset)
		putU16(entry, columnSizeOffset, column.size)

		if column.fixed {
			entry[columnFlagsOffset] = columnFlagFixed
		}

		stream = append(stream, entry...)
	}

	for _, column := range columns {
		encoded := utf16Bytes(column.name)
		length := make([]byte, 2)
		putU16(length, 0, len(encoded))
		stream = append(stream, length...)
		stream = append(stream, encoded...)
	}

	pages := [][]byte{make([]byte, PageSize)}
	copy(pages[0], stream[:min(PageSize, len(stream))])

	remaining := stream[min(PageSize, len(stream)):]

	for len(remaining) > 0 {
		page := make([]byte, PageSize)
		take := min(PageSize-pageContinuationHeaderSize, len(remaining))
		copy(page[pageContinuationHeaderSize:], remaining[:take])
		remaining = remaining[take:]
		pages = append(pages, page)
	}

	for index := 0; index+1 < len(pages); index++ {
		// Each page holds the absolute index of the page that continues it.
		putU32(pages[index], pageLinkOffset, uint32(index+2))
		pages[index+1][0] = PageTypeDefinition
	}

	return pages
}

// packRow lays out one row: the column count, the fixed columns, the variable
// columns and, growing down from the end of the row, the variable column
// offsets, the variable column count and the not-null mask.
func packRow(columnCount int, notNull []byte, fixed []byte, variables [][]byte) []byte {
	row := make([]byte, 0, PageSize)
	header := make([]byte, 2)
	putU16(header, 0, columnCount)
	row = append(row, header...)
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
		entry := make([]byte, 2)
		putU16(entry, 0, offsets[index])
		row = append(row, entry...)
	}

	count := make([]byte, 2)
	putU16(count, 0, len(variables))
	row = append(row, count...)
	row = append(row, notNull...)

	return row
}

// buildDataPage lays out a data or leaf page that belongs to the specified
// definition page.
func buildDataPage(owner int, rows [][]byte) []byte {
	page := make([]byte, PageSize)
	page[0] = PageTypeData
	putU32(page, pageLinkOffset, uint32(owner))
	putU16(page, rowCountOffset, len(rows))

	position := PageSize

	for index, row := range rows {
		position -= len(row)
		copy(page[position:], row)
		putU16(page, rowStartTable+index*2, position)
	}

	return page
}

func TestDefinition(t *testing.T) {
	pages := definitionPages(7, 2, []testColumn{
		{name: "id", kind: ColumnTypeLongInt, fixed: true, offset: 0, size: 4},
		{name: "label", kind: ColumnTypeText, size: 50, variable: 0},
	})

	database, err := Open(buildDatabaseFromPages(pages...))
	require.NoError(t, err)

	definition, err := database.Definition(1)
	require.NoError(t, err)

	assert.Equal(t, 1, definition.Page)
	assert.Equal(t, 7, definition.RowCount)
	assert.Equal(t, 2, definition.FirstDataPage)
	require.Len(t, definition.Columns, 2)

	assert.Equal(t, "id", definition.Columns[0].Name)
	assert.Equal(t, ColumnTypeLongInt, definition.Columns[0].Type)
	assert.True(t, definition.Columns[0].Fixed)
	assert.Equal(t, 0, definition.Columns[0].Offset)
	assert.Equal(t, 4, definition.Columns[0].Size)

	assert.Equal(t, "label", definition.Columns[1].Name)
	assert.Equal(t, ColumnTypeText, definition.Columns[1].Type)
	assert.False(t, definition.Columns[1].Fixed)
	assert.Equal(t, 0, definition.Columns[1].VariableIndex)

	require.NotNil(t, definition.Column("LABEL"))
	assert.Nil(t, definition.Column("missing"))
}

func TestDefinitionFollowsContinuationPages(t *testing.T) {
	columns := make([]testColumn, 150)

	for index := range columns {
		columns[index] = testColumn{
			name:   "column" + string(rune('a'+index%26)) + string(rune('a'+index/26)),
			kind:   ColumnTypeLongInt,
			fixed:  true,
			offset: index * 4,
			size:   4,
		}
	}

	pages := definitionPages(1, 2, columns)
	require.Greater(t, len(pages), 1, "the definition must span several pages")

	database, err := Open(buildDatabaseFromPages(pages...))
	require.NoError(t, err)

	definition, err := database.Definition(1)
	require.NoError(t, err)
	require.Len(t, definition.Columns, 150)
	assert.Equal(t, columns[149].name, definition.Columns[149].Name)
}

func TestDefinitionRejectsOtherPages(t *testing.T) {
	database, err := Open(buildDatabaseFromPages(buildDataPage(2, nil)))
	require.NoError(t, err)

	_, err = database.Definition(1)
	assert.ErrorIs(t, err, ErrNotDefinitionPage)

	_, err = database.Definition(99)
	assert.ErrorIs(t, err, ErrInvalidPage)
}

func TestDefinitionsSkipsContinuationPages(t *testing.T) {
	columns := make([]testColumn, 150)

	for index := range columns {
		columns[index] = testColumn{name: "c" + string(rune('a'+index%26)), kind: ColumnTypeLongInt, fixed: true, size: 4}
	}

	database, err := Open(buildDatabaseFromPages(definitionPages(1, 2, columns)...))
	require.NoError(t, err)

	definitions, err := database.Definitions()
	require.NoError(t, err)
	assert.Len(t, definitions, 1)
}

func TestEachRow(t *testing.T) {
	pages := definitionPages(2, 3, []testColumn{
		{name: "amount", kind: ColumnTypeMoney, fixed: true, offset: 0, size: 8},
		{name: "label", kind: ColumnTypeText, size: 20, variable: 0},
	})

	first := make([]byte, 8)
	putU64(first, 0, 0xFFFFFFFFFFF6918C)
	second := make([]byte, 8)
	putU64(second, 0, uint64(int64(2500)))

	data := buildDataPage(1, [][]byte{
		packRow(2, []byte{0x03}, first, [][]byte{[]byte("Rent")}),
		packRow(2, []byte{0x03}, second, [][]byte{[]byte{}}),
	})

	pages = append(pages, data)
	database, err := Open(buildDatabaseFromPages(pages...))
	require.NoError(t, err)

	definition, err := database.Definition(1)
	require.NoError(t, err)

	values := []string{}

	err = database.EachRow(context.Background(), definition, func(fields []Field) error {
		require.Len(t, fields, 2)

		amount, err := database.Value(fields[0])
		require.NoError(t, err)

		label, err := database.String(fields[1])
		require.NoError(t, err)

		values = append(values, label)
		assert.NotNil(t, amount)

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"Rent", ""}, values)
}

func TestEachRowSkipsDeletedRows(t *testing.T) {
	pages := definitionPages(1, 3, []testColumn{
		{name: "amount", kind: ColumnTypeLongInt, fixed: true, offset: 0, size: 4},
	})

	data := buildDataPage(1, [][]byte{packRow(1, []byte{0x01}, []byte{1, 0, 0, 0}, nil)})
	// Mark the row deleted.
	start, _ := readU16(data, rowStartTable)
	putU16(data, rowStartTable, start|rowDeletedFlag)

	pages = append(pages, data)
	database, err := Open(buildDatabaseFromPages(pages...))
	require.NoError(t, err)

	definition, err := database.Definition(1)
	require.NoError(t, err)

	visited := 0

	err = database.EachRow(context.Background(), definition, func([]Field) error {
		visited++

		return nil
	})

	require.NoError(t, err)
	assert.Zero(t, visited)
}

func TestEachRowFollowsRelocatedRows(t *testing.T) {
	pages := definitionPages(1, 3, []testColumn{
		{name: "amount", kind: ColumnTypeLongInt, fixed: true, offset: 0, size: 4},
	})

	overflow := buildDataPage(1, [][]byte{packRow(1, []byte{0x01}, []byte{42, 0, 0, 0}, nil)})

	pointer := make([]byte, 4)
	putU32(pointer, 0, 3<<8) // page 3, row 0
	slot := packRow(1, []byte{0x01}, pointer, nil)
	data := buildDataPage(1, [][]byte{slot})
	start, _ := readU16(data, rowStartTable)
	putU16(data, rowStartTable, start|rowLookupFlag)

	pages = append(pages, data, overflow)
	database, err := Open(buildDatabaseFromPages(pages...))
	require.NoError(t, err)

	definition, err := database.Definition(1)
	require.NoError(t, err)

	var amounts []int64

	err = database.EachRow(context.Background(), definition, func(fields []Field) error {
		amount, ok := database.Int(fields[0])
		require.True(t, ok)

		amounts = append(amounts, amount)

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, []int64{42}, amounts)
}

func TestValueDecoding(t *testing.T) {
	database, err := Open(buildDatabase(PageTypeData))
	require.NoError(t, err)

	money := make([]byte, 8)
	putU64(money, 0, 0xFFFFFFFFFFED2979)

	value, err := database.Value(Field{Column: Column{Type: ColumnTypeMoney}, Data: money})
	require.NoError(t, err)
	assert.InDelta(t, -123.4567, value, 1e-9)

	days := make([]byte, 8)
	putU64(days, 0, 0x40E34A0000000000) // 2008-02-26

	date, ok := database.Time(Field{Column: Column{Type: ColumnTypeDateTime}, Data: days})
	require.True(t, ok)
	assert.Equal(t, time.Date(2008, 2, 26, 0, 0, 0, 0, time.UTC), date)

	flag := Field{Column: Column{Type: ColumnTypeBool}, Null: false}
	assert.True(t, database.Bool(flag))
	assert.False(t, database.Bool(Field{Column: Column{Type: ColumnTypeBool}, Null: true}))

	guid := []byte{0xDD, 0xD7, 0xF8, 0x67, 0xB3, 0x03, 0x39, 0x4A, 0xAB, 0xC9, 0x85, 0xA5, 0xC0, 0x33, 0xC0, 0xCE}
	formatted, ok := database.GUID(Field{Column: Column{Type: ColumnTypeGUID}, Data: guid})
	require.True(t, ok)
	// A GUID stores its first three fields little endian, like a Windows GUID.
	assert.Equal(t, "67F8D7DD-03B3-4A39-ABC9-85A5C033C0CE", formatted)

	null, err := database.Value(Field{Column: Column{Type: ColumnTypeText}, Null: true})
	require.NoError(t, err)
	assert.Nil(t, null)
}

func TestValueDecodesInlineMemo(t *testing.T) {
	database, err := Open(buildDatabase(PageTypeData))
	require.NoError(t, err)

	header := make([]byte, longValueOverhead)
	putU32(header, 0, longValueInline|6)
	header = append(header, 'M', 'e', 'm', 'o', '!', '!')

	text, err := database.String(Field{Column: Column{Type: ColumnTypeMemo}, Data: header})
	require.NoError(t, err)
	assert.Equal(t, "Memo!!", text)
}

func TestValueDecodesChainedMemo(t *testing.T) {
	last := make([]byte, 4)
	leaf := buildDataPage(0, [][]byte{append(append([]byte{}, last...), 'B', 'o', 'd', 'y')})

	next := make([]byte, 4)
	putU32(next, 0, 2<<8) // page 2, row 0
	root := buildDataPage(0, [][]byte{append(append([]byte{}, next...), 'M', 'e', 'm', 'o')})

	database, err := Open(buildDatabaseFromPages(root, leaf))
	require.NoError(t, err)

	head := make([]byte, longValueOverhead)
	putU32(head, 0, 8)                   // eight bytes follow: "Memo" and "Body"
	putU32(head, longValuePageRow, 1<<8) // page 1, row 0

	text, err := database.String(Field{Column: Column{Type: ColumnTypeMemo}, Data: head})
	require.NoError(t, err)
	assert.Equal(t, "MemoBody", text)
}

func TestChainedValueBoundsClaimedLength(t *testing.T) {
	last := make([]byte, 4)
	leaf := buildDataPage(0, [][]byte{append(append([]byte{}, last...), []byte("Body")...)})

	next := make([]byte, 4)
	putU32(next, 0, 2<<8)
	root := buildDataPage(0, [][]byte{append(append([]byte{}, next...), []byte("Memo")...)})

	database, err := Open(buildDatabaseFromPages(root, leaf))
	require.NoError(t, err)

	head := make([]byte, longValueOverhead)
	putU32(head, 0, 0x00FFFFFF) // the largest chained length, with no flag bits
	putU32(head, longValuePageRow, 1<<8)

	value, err := database.String(Field{Column: Column{Type: ColumnTypeMemo}, Data: head})
	require.NoError(t, err)
	assert.Equal(t, "MemoBody", value, "the buffer must never be preallocated to the claimed length")

	putU32(head, 0, 2) // a small claimed length truncates the value

	value, err = database.String(Field{Column: Column{Type: ColumnTypeMemo}, Data: head})
	require.NoError(t, err)
	assert.Equal(t, "Me", value)
}

func TestDefinitionClampsRowCountHint(t *testing.T) {
	pages := definitionPages(0xFFFFFFFF, 2, []testColumn{
		{name: "id", kind: ColumnTypeLongInt, fixed: true, offset: 0, size: 4},
	})

	database, err := Open(buildDatabaseFromPages(pages...))
	require.NoError(t, err)

	definition, err := database.Definition(1)
	require.NoError(t, err)
	assert.Equal(t, maxRowCountHint, definition.RowCount)
}

func TestOpenNeverPanicsOnMalformedInput(t *testing.T) {
	seed := make([]byte, PageSize)
	copy(seed[magicOffset:], Magic)

	inputs := map[string][]byte{
		"empty":                {},
		"smaller than a page":  bytes.Repeat([]byte{0xFF}, PageSize-1),
		"zero filled":          make([]byte, PageSize),
		"magic only":           seed,
		"magic and zero pages": append(append([]byte{}, seed...), make([]byte, 16*PageSize)...),
	}

	// Flip one byte at every offset of a valid-looking header, to catch paths
	// that trust a single field without checking the rest of the page.
	for offset := 0; offset < PageSize; offset++ {
		mutated := append([]byte{}, seed...)
		mutated[offset] ^= 0xFF
		inputs[fmt.Sprintf("flipped byte at %d", offset)] = mutated
	}

	for name, input := range inputs {
		t.Run(name, func(t *testing.T) {
			database, err := Open(input)

			if err != nil {
				return
			}

			tables, err := database.Tables()

			if err != nil {
				return
			}

			for _, table := range tables {
				_ = database.EachRow(context.Background(), table.Definition, func([]Field) error { return nil })
			}
		})
	}
}

// FuzzOpen checks that no input can panic or loop forever while opening a
// database, reading its tables, and walking every row.
func FuzzOpen(f *testing.F) {
	f.Add([]byte{})
	f.Add(make([]byte, PageSize))

	seed := make([]byte, PageSize)
	copy(seed[magicOffset:], Magic)
	f.Add(seed)

	f.Fuzz(func(t *testing.T, data []byte) {
		database, err := Open(data)

		if err != nil {
			return
		}

		tables, err := database.Tables()

		if err != nil {
			return
		}

		for _, table := range tables {
			_ = database.EachRow(context.Background(), table.Definition, func([]Field) error { return nil })
		}
	})
}

func TestCatalogNamesTables(t *testing.T) {
	definition := definitionPages(3, 3, []testColumn{
		{name: "hthing", kind: ColumnTypeLongInt, fixed: true, offset: 0, size: 4},
		{name: "szFull", kind: ColumnTypeText, size: 40, variable: 0},
	})

	// A catalog row names the definition page of its object first and its name
	// first among the variable length columns.
	catalogRow := packRow(3, []byte{0x07}, []byte{1, 0, 0, 0}, [][]byte{[]byte("THING"), []byte{}})
	catalog := buildDataPage(catalogPage, [][]byte{catalogRow})

	pages := append(append([][]byte{}, definition...), catalog)

	database, err := Open(buildDatabaseFromPages(pages...))
	require.NoError(t, err)

	entries, err := database.Catalog()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, CatalogEntry{Name: "THING", Page: 1}, entries[0])

	table, err := database.Table("thing")
	require.NoError(t, err)
	assert.Equal(t, "THING", table.Name)
	assert.Equal(t, 1, table.Definition.Page)
}

func TestTablesDeriveNames(t *testing.T) {
	definition := definitionPages(1, 4, []testColumn{
		{name: "hwidget", kind: ColumnTypeLongInt, fixed: true, offset: 0, size: 4},
	})

	database, err := Open(buildDatabaseFromPages(definition...))
	require.NoError(t, err)

	tables, err := database.Tables()
	require.NoError(t, err)
	require.Len(t, tables, 1)
	assert.Equal(t, "WIDGET", tables[0].Name)

	_, err = database.Table("missing")
	assert.ErrorIs(t, err, ErrTableNotFound)
}

func TestTablesReportAmbiguousNames(t *testing.T) {
	first := definitionPages(1, 4, []testColumn{
		{name: "hwidget", kind: ColumnTypeLongInt, fixed: true, offset: 0, size: 4},
	})
	second := definitionPages(1, 4, []testColumn{
		{name: "hwidget", kind: ColumnTypeLongInt, fixed: true, offset: 0, size: 4},
	})

	pages := append(append([][]byte{}, first...), second...)

	database, err := Open(buildDatabaseFromPages(pages...))
	require.NoError(t, err)

	tables, err := database.Tables()
	require.NoError(t, err)
	require.Len(t, tables, 2)

	names := []string{tables[0].Name, tables[1].Name}
	assert.ElementsMatch(t, []string{"UNKNOWN_1", "UNKNOWN_2"}, names)
}

func TestOpenRejectsUnsupportedVersion(t *testing.T) {
	data := buildDatabase(PageTypeData)
	data[versionOffset] = 0 // Money 2000 and earlier layout

	_, err := Open(data)
	assert.ErrorIs(t, err, ErrUnsupportedVersion)

	_, err = Open(buildDatabase(PageTypeData))
	require.NoError(t, err)
}

func TestEachRowStopsOnCancelledContext(t *testing.T) {
	pages := definitionPages(2, 3, []testColumn{
		{name: "id", kind: ColumnTypeLongInt, fixed: true, offset: 0, size: 4},
	})

	row := make([]byte, 4)
	putU32(row, 0, 1)

	pages = append(pages, buildDataPage(1, [][]byte{packRow(1, []byte{0x01}, row, nil), packRow(1, []byte{0x01}, row, nil)}))

	database, err := Open(buildDatabaseFromPages(pages...))
	require.NoError(t, err)

	definition, err := database.Definition(1)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	visited := 0
	err = database.EachRow(ctx, definition, func([]Field) error {
		visited++

		return nil
	})

	assert.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, 0, visited, "a cancelled context must stop before the first row")
}

func TestDiagnosticsRecordsDecodingProblems(t *testing.T) {
	pages := definitionPages(2, 1, []testColumn{
		{name: "id", kind: ColumnTypeLongInt, fixed: true, offset: 0, size: 4},
	})

	// One valid data page and one whose slot table points past the end of the
	// page, so its single row cannot be resolved.
	row := make([]byte, 4)
	putU32(row, 0, 7)
	pages = append(pages, buildDataPage(1, [][]byte{packRow(1, []byte{0x01}, row, nil)}))

	corrupt := make([]byte, PageSize)
	corrupt[0] = PageTypeData
	putU32(corrupt, pageLinkOffset, 1)
	putU16(corrupt, rowCountOffset, 1)
	putU16(corrupt, rowStartTable, 0x3000)
	pages = append(pages, corrupt)

	database, err := Open(buildDatabaseFromPages(pages...))
	require.NoError(t, err)

	definition, err := database.Definition(1)
	require.NoError(t, err)

	diagnostics := &Diagnostics{}
	database.SetDiagnostics(diagnostics)

	visited := 0
	require.NoError(t, database.EachRow(context.Background(), definition, func([]Field) error {
		visited++

		return nil
	}))

	assert.Equal(t, 1, visited, "the valid row must still be visited")
	assert.Equal(t, 1, diagnostics.UnresolvedRows)
	assert.Len(t, diagnostics.RowSamples, 1)
	assert.Equal(t, "row slot cannot be resolved", diagnostics.RowSamples[0].Err)
	assert.NotEmpty(t, diagnostics.Summary())
}

func TestDiagnosticsStaysSilentWithoutSink(t *testing.T) {
	pages := definitionPages(1, 1, []testColumn{
		{name: "id", kind: ColumnTypeLongInt, fixed: true, offset: 0, size: 4},
	})

	corrupt := make([]byte, PageSize)
	corrupt[0] = PageTypeData
	putU32(corrupt, pageLinkOffset, 1)
	putU16(corrupt, rowCountOffset, 1)
	putU16(corrupt, rowStartTable, 0x3000)
	pages = append(pages, corrupt)

	database, err := Open(buildDatabaseFromPages(pages...))
	require.NoError(t, err)

	definition, err := database.Definition(1)
	require.NoError(t, err)

	require.NoError(t, database.EachRow(context.Background(), definition, func([]Field) error { return nil }))
	assert.Nil(t, database.Diagnostics())
}
