// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msisam

import (
	"fmt"
	"strings"
)

// Page-relative offsets of the fields of a table definition page.
const (
	definitionRowCountOffset            = 16
	definitionVariableColumnCountOffset = 43
	definitionColumnCountOffset         = 45
	definitionIndexCountOffset          = 47
	definitionRealIndexCountOffset      = 51
	definitionFirstDataPageOffset       = 56
	definitionColumnsOffset             = 63
	definitionColumnEntrySize           = 25
	definitionRealIndexEntrySize        = 12

	// maxColumnCount bounds a definition so that a corrupt column count cannot
	// make the reader walk off the end of the page chain.
	maxColumnCount = 1024

	// maxChainPages bounds the continuation pages a single definition may use.
	maxChainPages = 64

	// maxRowCountHint bounds the row count a definition reports. The count is
	// only used to size buffers, so a corrupt count cannot demand a huge
	// allocation; a real database holds far fewer rows.
	maxRowCountHint = 1 << 20
)

// Page-relative offsets of the fields of a single column entry.
const (
	columnTypeOffset          = 0
	columnNumberOffset        = 5
	columnVariableIndexOffset = 7
	columnScaleOffset         = 11
	columnPrecisionOffset     = 12
	columnFlagsOffset         = 15
	columnFixedOffsetOffset   = 21
	columnSizeOffset          = 23
)

// Column flags.
const (
	columnFlagFixed = 0x01
)

// ColumnType is the data type code stored in a table definition.
type ColumnType byte

// Column type codes written by the database engine.
const (
	ColumnTypeBool     ColumnType = 0x01
	ColumnTypeByte     ColumnType = 0x02
	ColumnTypeInt      ColumnType = 0x03
	ColumnTypeLongInt  ColumnType = 0x04
	ColumnTypeMoney    ColumnType = 0x05
	ColumnTypeFloat    ColumnType = 0x06
	ColumnTypeDouble   ColumnType = 0x07
	ColumnTypeDateTime ColumnType = 0x08
	ColumnTypeBinary   ColumnType = 0x09
	ColumnTypeText     ColumnType = 0x0A
	ColumnTypeOLE      ColumnType = 0x0B
	ColumnTypeMemo     ColumnType = 0x0C
	ColumnTypeGUID     ColumnType = 0x0F
	ColumnTypeNumeric  ColumnType = 0x10
	ColumnTypeComplex  ColumnType = 0x12
)

// String returns the name of the column type, for diagnostics.
func (t ColumnType) String() string {
	switch t {
	case ColumnTypeBool:
		return "bool"
	case ColumnTypeByte:
		return "byte"
	case ColumnTypeInt:
		return "int"
	case ColumnTypeLongInt:
		return "longint"
	case ColumnTypeMoney:
		return "money"
	case ColumnTypeFloat:
		return "float"
	case ColumnTypeDouble:
		return "double"
	case ColumnTypeDateTime:
		return "datetime"
	case ColumnTypeBinary:
		return "binary"
	case ColumnTypeText:
		return "text"
	case ColumnTypeOLE:
		return "ole"
	case ColumnTypeMemo:
		return "memo"
	case ColumnTypeGUID:
		return "guid"
	case ColumnTypeNumeric:
		return "numeric"
	case ColumnTypeComplex:
		return "complex"
	default:
		return fmt.Sprintf("unknown(0x%02X)", byte(t))
	}
}

// isKnownColumnType reports whether the type code is one the engine writes.
func (t ColumnType) isKnownColumnType() bool {
	switch t {
	case ColumnTypeBool, ColumnTypeByte, ColumnTypeInt, ColumnTypeLongInt,
		ColumnTypeMoney, ColumnTypeFloat, ColumnTypeDouble, ColumnTypeDateTime,
		ColumnTypeBinary, ColumnTypeText, ColumnTypeOLE, ColumnTypeMemo,
		ColumnTypeGUID, ColumnTypeNumeric, ColumnTypeComplex:
		return true
	default:
		return false
	}
}

// Column describes one column of a table.
type Column struct {
	// Name is the column name as stored in the definition.
	Name string
	// Type is the data type of the column.
	Type ColumnType
	// Number is the bit position of the column in the row's null mask. It is
	// also the bit position of a bool column's value.
	Number int
	// VariableIndex is the index of the column in the row's variable length
	// offset table. It is only meaningful when Fixed is false.
	VariableIndex int
	// Fixed reports whether the column is stored inline in every row.
	Fixed bool
	// Offset is the byte offset of a fixed column within the row.
	Offset int
	// Size is the byte width of a fixed column, or the maximum length of a
	// variable length column.
	Size int
	// Precision and Scale describe numeric columns.
	Precision int
	Scale     int
}

// Definition is a table definition read from a definition page.
type Definition struct {
	// Page is the index of the page that starts the definition.
	Page int
	// RowCount is the number of rows the table holds.
	RowCount int
	// FirstDataPage is the first page of the table's data.
	FirstDataPage int
	// Columns are the table's columns, ordered by column number.
	Columns []Column
}

// Column returns the column with the specified name, matching case
// insensitively, or nil when the table has no such column.
func (definition *Definition) Column(name string) *Column {
	for index := range definition.Columns {
		if strings.EqualFold(definition.Columns[index].Name, name) {
			return &definition.Columns[index]
		}
	}

	return nil
}

// Definition reads the table definition stored at the specified page.
func (d *Database) Definition(page int) (*Definition, error) {
	if !d.IsValidPageIndex(page) {
		return nil, ErrInvalidPage
	}

	if d.PageType(page) != PageTypeDefinition {
		return nil, ErrNotDefinitionPage
	}

	header := d.Page(page)
	columnCount, ok := readU16(header, definitionColumnCountOffset)
	if !ok || columnCount == 0 || columnCount > maxColumnCount {
		return nil, ErrCorruptDefinition
	}

	realIndexCount, _ := readU32(header, definitionRealIndexCountOffset)
	if realIndexCount > maxColumnCount {
		return nil, ErrCorruptDefinition
	}

	definition := &Definition{
		Page:    page,
		Columns: make([]Column, 0, columnCount),
	}

	rowCount, _ := readU32(header, definitionRowCountOffset)

	if rowCount > maxRowCountHint {
		definition.RowCount = maxRowCountHint
	} else {
		definition.RowCount = int(rowCount)
	}

	firstDataPage, _ := readU16(header, definitionFirstDataPageOffset)
	definition.FirstDataPage = firstDataPage

	chain := d.newPageChain(page)
	chain.offset = definitionColumnsOffset + int(realIndexCount)*definitionRealIndexEntrySize

	for index := 0; index < columnCount; index++ {
		entry, err := chain.read(definitionColumnEntrySize)

		if err != nil {
			return nil, err
		}

		column, err := parseColumnEntry(entry)

		if err != nil {
			return nil, err
		}

		definition.Columns = append(definition.Columns, *column)
	}

	for index := range definition.Columns {
		length, err := chain.readUInt16()

		if err != nil {
			return nil, err
		}

		if length > PageSize {
			return nil, ErrCorruptDefinition
		}

		name, err := chain.read(int(length))

		if err != nil {
			return nil, err
		}

		columnName, err := decodeText(name)

		if err != nil {
			return nil, fmt.Errorf("%w: column %d name is truncated", ErrCorruptDefinition, index)
		}

		definition.Columns[index].Name = columnName
	}

	return definition, nil
}

// parseColumnEntry decodes the fixed width part of a column entry.
func parseColumnEntry(entry []byte) (*Column, error) {
	columnType := ColumnType(entry[columnTypeOffset])

	if !columnType.isKnownColumnType() {
		return nil, fmt.Errorf("%w: unknown column type 0x%02X", ErrCorruptDefinition, byte(columnType))
	}

	columnNumber, _ := readU16(entry, columnNumberOffset)
	variableIndex, _ := readU16(entry, columnVariableIndexOffset)
	fixedOffset, _ := readU16(entry, columnFixedOffsetOffset)
	size, _ := readU16(entry, columnSizeOffset)
	flags := entry[columnFlagsOffset]

	column := &Column{
		Type:          columnType,
		Number:        columnNumber,
		VariableIndex: variableIndex,
		Fixed:         flags&columnFlagFixed != 0,
		Offset:        fixedOffset,
		Size:          size,
		Scale:         int(entry[columnScaleOffset]),
		Precision:     int(entry[columnPrecisionOffset]),
	}

	// A bool stores its value in the null mask, so it has no width of its own.
	if columnType == ColumnTypeBool {
		column.Size = 0
	}

	return column, nil
}

// Definitions returns every table definition in the database, skipping any that
// cannot be decoded. Definition pages that merely continue an earlier definition
// are folded into their table.
func (d *Database) Definitions() ([]*Definition, error) {
	starts, err := d.definitionPages()

	if err != nil {
		return nil, err
	}

	definitions := make([]*Definition, 0, len(starts))

	for _, page := range starts {
		definition, err := d.Definition(page)

		if err != nil {
			// A database whose catalog is in use holds a bare definition page
			// at the catalog page: it carries no readable definition because
			// the catalog is read from its data pages. That is not corruption.
			catalogInUse := page == catalogPage && len(d.dataPagesOf(catalogPage)) > 0

			if d.diagnostics != nil && !catalogInUse {
				d.diagnostics.recordDefinition(page, err)
			}

			continue
		}

		definitions = append(definitions, definition)
	}

	return definitions, nil
}

// definitionPages returns the pages that start a table definition, in page
// order.
func (d *Database) definitionPages() ([]int, error) {
	if d.encrypted() {
		return nil, ErrEncryptedDatabase
	}

	pages := make([]int, 0, 128)
	continuations := make(map[int]bool)

	for page := 1; page < d.pageCount; page++ {
		if d.PageType(page) != PageTypeDefinition {
			continue
		}

		pages = append(pages, page)
	}

	for _, page := range pages {
		next, _ := readU32(d.Page(page), pageLinkOffset)

		if d.IsValidPageIndex(int(next)) && d.PageType(int(next)) == PageTypeDefinition {
			continuations[int(next)] = true
		}
	}

	starts := make([]int, 0, len(pages))

	for _, page := range pages {
		if !continuations[page] {
			starts = append(starts, page)
		}
	}

	return starts, nil
}

// pageChain reads the bytes of a table definition, following the chain of
// continuation pages that holds a definition larger than one page.
type pageChain struct {
	database *Database
	page     int
	offset   int
	followed int
}

// newPageChain returns a reader positioned at the start of the specified page.
func (d *Database) newPageChain(page int) *pageChain {
	return &pageChain{database: d, page: page}
}

// read consumes the specified number of bytes from the chain.
func (c *pageChain) read(length int) ([]byte, error) {
	output := make([]byte, 0, length)

	for len(output) < length {
		if c.offset >= PageSize {
			next, ok := readU32(c.database.Page(c.page), pageLinkOffset)

			if !ok || !c.database.IsValidPageIndex(int(next)) {
				return nil, ErrCorruptDefinition
			}

			c.followed++

			if c.followed > maxChainPages {
				return nil, ErrCorruptDefinition
			}

			// Continuation pages repeat the eight byte page header.
			c.page = int(next)
			c.offset = pageContinuationHeaderSize
		}

		page := c.database.Page(c.page)
		take := min(length-len(output), PageSize-c.offset)

		output = append(output, page[c.offset:c.offset+take]...)
		c.offset += take
	}

	return output, nil
}

// readUInt16 consumes one little endian 16 bit integer from the chain.
func (c *pageChain) readUInt16() (int, error) {
	data, err := c.read(2)

	if err != nil {
		return 0, err
	}

	value, _ := readU16(data, 0)

	return value, nil
}
