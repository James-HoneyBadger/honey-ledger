// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msisam

import (
	"fmt"
	"math"
	"time"
	"unicode/utf16"
)

// Offsets of the fields of a row on a data or leaf page.
const (
	rowCountOffset = 0x0C
	rowStartTable  = 0x0E
	rowEndTable    = 0x0C
	rowDeletedFlag = 0x8000
	rowLookupFlag  = 0x4000
	rowOffsetMask  = 0x3FFF
	rowHeaderSize  = 2
)

// Layout of a long value, which a memo or OLE column stores in place of the
// data itself.
const (
	longValueOverhead    = 12
	longValueInline      = 0x80000000
	longValueSinglePage  = 0x40000000
	longValueFlagsMask   = 0xFF000000
	longValuePageRow     = 4
	longValueChainHeader = 4

	// longValueMaxSize bounds how large a chained long value may grow: the
	// chain is followed for at most maxChainPages pages, so it cannot hold
	// more than one page per link.
	longValueMaxSize = maxChainPages * PageSize
)

// moneyScale is the number of decimal digits a money value is stored with.
const moneyScale = 10000

// numericBytes is the width of the magnitude of a numeric value, not counting
// its sign byte.
const numericBytes = 16

// jetEpoch is the date the engine counts days from.
var jetEpoch = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)

// readU16 reads a little endian 16 bit integer, reporting whether the buffer
// holds it.
func readU16(buffer []byte, offset int) (int, bool) {
	if offset < 0 || offset+2 > len(buffer) {
		return 0, false
	}

	return int(buffer[offset]) | int(buffer[offset+1])<<8, true
}

// readU32 reads a little endian 32 bit integer, reporting whether the buffer
// holds it.
func readU32(buffer []byte, offset int) (uint32, bool) {
	if offset < 0 || offset+4 > len(buffer) {
		return 0, false
	}

	return uint32(buffer[offset]) | uint32(buffer[offset+1])<<8 |
		uint32(buffer[offset+2])<<16 | uint32(buffer[offset+3])<<24, true
}

// readU64 reads a little endian 64 bit integer, reporting whether the buffer
// holds it.
func readU64(buffer []byte, offset int) (uint64, bool) {
	low, ok := readU32(buffer, offset)

	if !ok {
		return 0, false
	}

	high, ok := readU32(buffer, offset+4)

	if !ok {
		return 0, false
	}

	return uint64(low) | uint64(high)<<32, true
}

// decodeText converts stored text to a Go string. Text is UTF-16 unless a high
// byte of every character is zero, in which case the engine drops those bytes
// and a single byte toggles between the two encodings.
func decodeText(data []byte) string {
	expanded := expandText(data)

	codeUnits := make([]uint16, 0, len(expanded)/2)

	for offset := 0; offset+1 < len(expanded); offset += 2 {
		codeUnits = append(codeUnits, uint16(expanded[offset])|uint16(expanded[offset+1])<<8)
	}

	return string(utf16.Decode(codeUnits))
}

// expandText undoes the engine's text compression, returning UTF-16 bytes.
func expandText(data []byte) []byte {
	expanded := make([]byte, 0, len(data)*2)
	compressed := true

	for offset := 0; offset < len(data); {
		if data[offset] == 0 {
			compressed = !compressed
			offset++

			continue
		}

		if compressed {
			expanded = append(expanded, data[offset], 0)
			offset++

			continue
		}

		if offset+1 >= len(data) {
			break
		}

		expanded = append(expanded, data[offset], data[offset+1])
		offset += 2
	}

	return expanded
}

// Field is one column of a row, still encoded the way the page stores it.
type Field struct {
	// Column is the column the field belongs to.
	Column Column
	// Null reports whether the column holds no value for this row.
	Null bool
	// Data holds the stored bytes, or nil when the column is null. It aliases
	// the database buffer and must not be modified.
	Data []byte
}

// Value decodes a field into a Go value. Integers decode to int64, money,
// float, double and numeric decode to float64, dates decode to time.Time, text
// and memos decode to string, and binary and OLE decode to []byte. A GUID
// decodes to its dashed string form, and a null field decodes to nil.
func (d *Database) Value(field Field) (any, error) {
	// A bool stores its value in the null mask, so it is never null.
	if field.Column.Type == ColumnTypeBool {
		return !field.Null, nil
	}

	if field.Null {
		return nil, nil
	}

	switch field.Column.Type {
	case ColumnTypeByte, ColumnTypeInt, ColumnTypeLongInt:
		value, ok := d.Int(field)

		if !ok {
			return nil, ErrCorruptRow
		}

		return value, nil
	case ColumnTypeMoney:
		value, ok := d.money(field)

		if !ok {
			return nil, ErrCorruptRow
		}

		return float64(value) / moneyScale, nil
	case ColumnTypeFloat, ColumnTypeDouble:
		value, ok := d.Float(field)

		if !ok {
			return nil, ErrCorruptRow
		}

		return value, nil
	case ColumnTypeNumeric:
		value, ok := d.numeric(field)

		if !ok {
			return nil, ErrCorruptRow
		}

		return value, nil
	case ColumnTypeDateTime:
		value, ok := d.Time(field)

		if !ok {
			return nil, ErrCorruptRow
		}

		return value, nil
	case ColumnTypeText, ColumnTypeMemo:
		return d.String(field)
	case ColumnTypeGUID:
		value, ok := d.GUID(field)

		if !ok {
			return nil, ErrCorruptRow
		}

		return value, nil
	case ColumnTypeBinary, ColumnTypeOLE:
		return d.Bytes(field)
	default:
		return nil, fmt.Errorf("%w: unsupported column type %s", ErrCorruptRow, field.Column.Type)
	}
}

// Bool returns the value of a bool field.
func (d *Database) Bool(field Field) bool {
	return !field.Null
}

// Int returns the value of a byte, int or long integer field.
func (d *Database) Int(field Field) (int64, bool) {
	if field.Null {
		return 0, false
	}

	switch field.Column.Type {
	case ColumnTypeByte:
		if len(field.Data) < 1 {
			return 0, false
		}

		return int64(field.Data[0]), true
	case ColumnTypeInt:
		value, ok := readU16(field.Data, 0)

		return int64(int16(value)), ok
	case ColumnTypeLongInt:
		value, ok := readU32(field.Data, 0)

		return int64(int32(value)), ok
	case ColumnTypeMoney:
		return d.money(field)
	default:
		return 0, false
	}
}

// money returns the value of a money field scaled by 10000.
func (d *Database) money(field Field) (int64, bool) {
	value, ok := readU64(field.Data, 0)

	return int64(value), ok
}

// Float returns the value of a float or double field.
func (d *Database) Float(field Field) (float64, bool) {
	if field.Null {
		return 0, false
	}

	switch field.Column.Type {
	case ColumnTypeFloat:
		value, ok := readU32(field.Data, 0)

		return float64(math.Float32frombits(value)), ok
	case ColumnTypeDouble:
		value, ok := readU64(field.Data, 0)

		return math.Float64frombits(value), ok
	default:
		return 0, false
	}
}

// Numeric returns the value of a numeric field, which stores a sign byte
// followed by a little endian magnitude scaled by the column's scale. It is a
// float64 convenience for inspection; financial code must not treat it as an
// exact representation.
func (d *Database) Numeric(field Field) (float64, bool) {
	return d.numeric(field)
}

// numeric decodes a numeric field, which stores a sign byte followed by a
// little endian magnitude that is scaled by the column's scale.
func (d *Database) numeric(field Field) (float64, bool) {
	if len(field.Data) < 1+numericBytes {
		return 0, false
	}

	magnitude := uint64(0)

	for index := numericBytes - 1; index >= 0; index-- {
		magnitude = magnitude<<8 | uint64(field.Data[1+index])
	}

	value := float64(magnitude)

	for scale := 0; scale < field.Column.Scale; scale++ {
		value /= 10
	}

	if field.Data[0]&0x80 != 0 {
		value = -value
	}

	return value, true
}

// Time returns the value of a date field, which counts days from 1899-12-30.
func (d *Database) Time(field Field) (time.Time, bool) {
	if field.Null {
		return time.Time{}, false
	}

	days, ok := readU64(field.Data, 0)

	if !ok {
		return time.Time{}, false
	}

	microseconds := math.Round(math.Float64frombits(days) * 24 * 60 * 60 * 1e6)

	if math.Abs(microseconds) > math.MaxInt64/2 {
		return time.Time{}, false
	}

	return jetEpoch.Add(time.Duration(int64(microseconds)) * time.Microsecond), true
}

// String returns the value of a text or memo field.
func (d *Database) String(field Field) (string, error) {
	if field.Null {
		return "", nil
	}

	data := field.Data

	switch field.Column.Type {
	case ColumnTypeText:
	case ColumnTypeMemo:
		long, err := d.longValue(field.Data)

		if err != nil {
			return "", err
		}

		data = long
	default:
		return "", fmt.Errorf("%w: column %s is not text", ErrCorruptRow, field.Column.Name)
	}

	return decodeText(data), nil
}

// Bytes returns the value of a binary or OLE field.
func (d *Database) Bytes(field Field) ([]byte, error) {
	if field.Null {
		return nil, nil
	}

	switch field.Column.Type {
	case ColumnTypeBinary:
		return field.Data, nil
	case ColumnTypeOLE:
		return d.longValue(field.Data)
	default:
		return nil, fmt.Errorf("%w: column %s is not binary", ErrCorruptRow, field.Column.Name)
	}
}

// GUID returns the value of a GUID field in its dashed string form.
func (d *Database) GUID(field Field) (string, bool) {
	if field.Null || len(field.Data) < 16 {
		return "", false
	}

	data := field.Data
	data1, _ := readU32(data, 0)
	data2, _ := readU16(data, 4)
	data3, _ := readU16(data, 6)

	return fmt.Sprintf("%08X-%04X-%04X-%02X%02X-%02X%02X%02X%02X%02X%02X",
		data1, data2, data3,
		data[8], data[9], data[10], data[11], data[12], data[13], data[14], data[15]), true
}

// longValue returns the data a memo or OLE column stores, following the pages
// that hold the part of the value that did not fit in the row.
func (d *Database) longValue(header []byte) ([]byte, error) {
	if len(header) < longValueOverhead {
		return nil, ErrCorruptRow
	}

	lengthAndFlags, _ := readU32(header, 0)

	switch {
	case lengthAndFlags&longValueInline != 0:
		return header[longValueOverhead:], nil
	case lengthAndFlags&longValueSinglePage != 0:
		pageRow, _ := readU32(header, longValuePageRow)
		data, ok := d.pageRow(pageRow)

		if !ok {
			return nil, ErrCorruptRow
		}

		return data, nil
	case lengthAndFlags&longValueFlagsMask == 0:
		return d.chainedValue(lengthAndFlags, header), nil
	default:
		return nil, fmt.Errorf("%w: unknown long value flags 0x%08X", ErrCorruptRow, lengthAndFlags)
	}
}

// chainedValue collects a long value that is spread over several leaf pages.
// The claimed length is untrusted, so the buffer is never preallocated beyond
// what the chain can actually hold.
func (d *Database) chainedValue(length uint32, header []byte) []byte {
	capacity := longValueMaxSize

	if length < uint32(capacity) {
		capacity = int(length)
	}

	output := make([]byte, 0, capacity)

	pageRow, _ := readU32(header, longValuePageRow)

	for followed := 0; pageRow != 0 && followed < maxChainPages; followed++ {
		chunk, ok := d.pageRow(pageRow)

		if !ok || len(chunk) <= longValueChainHeader {
			break
		}

		output = append(output, chunk[longValueChainHeader:]...)

		// Each chunk starts with the page and row of the next chunk.
		pageRow, _ = readU32(chunk, 0)
	}

	if uint64(length) < uint64(len(output)) {
		output = output[:int(length)]
	}

	return output
}

// pageRow returns the bytes of the row a page and row reference addresses.
func (d *Database) pageRow(reference uint32) ([]byte, bool) {
	page := int(reference >> 8)
	row := int(reference & 0xFF)

	if !d.IsValidPageIndex(page) {
		return nil, false
	}

	start, size, ok := rowBounds(d.Page(page), row)

	if !ok {
		return nil, false
	}

	return d.Page(page)[start : start+size], true
}

// rowBounds returns the start offset and length of the specified row slot.
func rowBounds(page []byte, row int) (int, int, bool) {
	count, ok := readU16(page, rowCountOffset)

	if !ok || row < 0 || row >= count {
		return 0, 0, false
	}

	start, ok := readU16(page, rowStartTable+row*2)

	if !ok {
		return 0, 0, false
	}

	end := PageSize

	if row > 0 {
		previous, ok := readU16(page, rowEndTable+row*2)

		if !ok {
			return 0, 0, false
		}

		end = previous & rowOffsetMask
	}

	start &= rowOffsetMask

	if start > end || end > PageSize {
		return 0, 0, false
	}

	return start, end - start, true
}
