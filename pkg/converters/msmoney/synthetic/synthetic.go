// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

// Package synthetic builds minimal, valid Microsoft Money data files (.mny)
// and backups (.mbf) in memory. It exists so the extraction pipeline can be
// tested end to end without a private sample file: Microsoft Money databases
// cannot be redistributed, but one can be synthesized.
//
// The generated database holds:
//
//   - two USD accounts, Checking and Savings
//   - a nested category, Food > Groceries, and a top level category Rent
//   - one split transaction, whose total row is skipped, allocated to
//     Groceries and Rent
//   - one transfer from Checking to Savings, stored on both sides
//   - one investment row, which has no counterpart in the extracted ledger
//
// Extracting it yields 2 accounts, 3 transactions (two split parts and the
// transfer) and a skipped count of 3 (the split total, the receiving side of
// the transfer, and the investment row).
package synthetic

import (
	"encoding/binary"
	"io"
	"math"
	"time"
	"unicode/utf16"

	"github.com/abemedia/go-cabinet"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/msmoney/msisam"
)

// Offsets shared by every page, kept in sync with the msisam reader.
const (
	pageLinkOffset = 4
	rowCountOffset = 0x0C
	rowStartTable  = 0x0E
)

// Offsets of a table definition page.
const (
	definitionRowCountOffset            = 16
	definitionVariableColumnCountOffset = 43
	definitionColumnCountOffset         = 45
	definitionFirstDataPageOffset       = 56
	definitionColumnsOffset             = 63
	definitionColumnEntrySize           = 25
)

// Offsets of a single column entry.
const (
	columnTypeOffset          = 0
	columnNumberOffset        = 5
	columnVariableIndexOffset = 7
	columnFlagsOffset         = 15
	columnFlagFixed           = 0x01
	columnFixedOffsetOffset   = 21
	columnSizeOffset          = 23
)

// Header fields of a Money database page.
const (
	versionOffset     = 0x14
	codePageOffset    = 0x3C
	databaseKeyOffset = 0x3E
	languageOffset    = 0x6E

	// encryptedHeaderOffset and encryptedHeaderLength describe the region of
	// the header page that is stored scrambled, even without a password.
	encryptedHeaderOffset = 0x18
	encryptedHeaderLength = 128
)

// headerKey scrambles the header fields the engine keeps out of plain sight,
// the same key the msisam reader unscrambles them with.
var headerKey = [4]byte{0xC7, 0xDA, 0x39, 0x6B}

// longValueInline marks a memo whose text is stored in the row itself, behind
// its twelve byte long value header.
const longValueInline = 0x80000000

// longValueOverhead is the size of the long value header every memo carries.
const longValueOverhead = 12

// moneyScale is the number of decimal places a Money amount is stored with.
const moneyScale = 10000

var jetEpoch = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)

// sampleDay is the date every generated transaction carries.
var sampleDay = time.Date(2024, 3, 5, 0, 0, 0, 0, time.UTC)

// column describes one column of a synthetic table definition.
type column struct {
	name     string
	kind     msisam.ColumnType
	fixed    bool
	offset   int
	size     int
	variable int
}

// MoneyFile returns a minimal, valid Microsoft Money data file.
func MoneyFile() []byte {
	dayBits := dateBits(sampleDay)

	headerPage := make([]byte, msisam.PageSize)
	headerPage[0] = msisam.PageTypeHeader
	copy(headerPage[4:], msisam.Magic)
	headerPage[versionOffset] = 1 // Jet 4 layout
	putU16(headerPage, codePageOffset, 1252)
	putU16(headerPage, languageOffset, 0x0409)
	// databaseKeyOffset stays zero: the database is not encrypted.
	scrambleHeader(headerPage)

	// Page 2 is where a real database registers its catalog. The page itself
	// does not need to decode; the catalog data page names every table.
	catalogDefinition := make([]byte, msisam.PageSize)
	catalogDefinition[0] = msisam.PageTypeDefinition

	acct := definitionPage(2, 7, []column{
		{name: "hacct", kind: msisam.ColumnTypeLongInt, fixed: true, offset: 0, size: 4},
		{name: "hcrnc", kind: msisam.ColumnTypeLongInt, fixed: true, offset: 4, size: 4},
		{name: "szFull", kind: msisam.ColumnTypeText, size: 40, variable: 0},
	})

	cat := definitionPage(5, 8, []column{
		{name: "hcat", kind: msisam.ColumnTypeLongInt, fixed: true, offset: 0, size: 4},
		{name: "hcatParent", kind: msisam.ColumnTypeLongInt, fixed: true, offset: 4, size: 4},
		{name: "szFull", kind: msisam.ColumnTypeText, size: 40, variable: 0},
	})

	trn := definitionPage(6, 9, []column{
		{name: "htrn", kind: msisam.ColumnTypeLongInt, fixed: true, offset: 0, size: 4},
		{name: "hacct", kind: msisam.ColumnTypeLongInt, fixed: true, offset: 4, size: 4},
		{name: "hacctLink", kind: msisam.ColumnTypeLongInt, fixed: true, offset: 8, size: 4},
		{name: "hcat", kind: msisam.ColumnTypeLongInt, fixed: true, offset: 12, size: 4},
		{name: "hsec", kind: msisam.ColumnTypeLongInt, fixed: true, offset: 16, size: 4},
		{name: "amt", kind: msisam.ColumnTypeMoney, fixed: true, offset: 20, size: 8},
		{name: "mMemo", kind: msisam.ColumnTypeMemo, size: 40, variable: 0},
		{name: "dt", kind: msisam.ColumnTypeDateTime, fixed: true, offset: 28, size: 8},
	})

	trnSplit := definitionPage(2, 10, []column{
		{name: "htrnParent", kind: msisam.ColumnTypeLongInt, fixed: true, offset: 0, size: 4},
	})

	crnc := definitionPage(1, 11, []column{
		{name: "hcrnc", kind: msisam.ColumnTypeLongInt, fixed: true, offset: 0, size: 4},
		{name: "szIsoCode", kind: msisam.ColumnTypeText, size: 4, variable: 0},
	})

	acctData := dataPage(1, [][]byte{
		accountRow(1, 1, "Checking"),
		accountRow(2, 1, "Savings"),
	})

	catData := dataPage(3, [][]byte{
		categoryRow(10, 0, "Income"),
		categoryRow(20, 0, "Expense"),
		categoryRow(11, 10, "Food"),
		categoryRow(12, 11, "Groceries"),
		categoryRow(21, 20, "Rent"),
	})

	trnData := dataPage(4, [][]byte{
		transactionRow(101, 1, 0, 12, 0, amount(-3000), "Groceries", dayBits),
		transactionRow(102, 1, 0, 21, 0, amount(-2000), "Rent", dayBits),
		transactionRow(200, 1, 2, 0, 0, amount(-10000), "Move to savings", dayBits),
		transactionRow(100, 1, 0, 0, 0, amount(-5000), "", dayBits), // split total
		transactionRow(201, 2, 1, 0, 0, amount(10000), "", dayBits), // receiving side
		transactionRow(300, 2, 0, 0, 7, amount(-2500), "", dayBits), // investment
	})

	trnSplitData := dataPage(5, [][]byte{
		packRow(1, []byte{0x01}, longInt(100), nil),
		packRow(1, []byte{0x01}, longInt(100), nil),
	})

	crncData := dataPage(6, [][]byte{
		packRow(2, []byte{0x03}, longInt(1), [][]byte{utf16Bytes("USD")}),
	})

	catalogData := dataPage(2, [][]byte{
		catalogRow(1, "ACCT"),
		catalogRow(3, "CAT"),
		catalogRow(4, "TRN"),
		catalogRow(5, "TRN_SPLIT"),
		catalogRow(6, "CRNC"),
	})

	pages := [][]byte{
		headerPage,        // 0
		acct,              // 1
		catalogDefinition, // 2
		cat,               // 3
		trn,               // 4
		trnSplit,          // 5
		crnc,              // 6
		acctData,          // 7
		catData,           // 8
		trnData,           // 9
		trnSplitData,      // 10
		crncData,          // 11
		catalogData,       // 12
	}

	file := make([]byte, 0, len(pages)*msisam.PageSize)

	for _, page := range pages {
		file = append(file, page...)
	}

	return file
}

// BackupFile returns MoneyFile wrapped in a Microsoft Cabinet archive, the
// layout Microsoft Money writes backups (.mbf) in.
func BackupFile() []byte {
	var out memoryWriteSeeker

	writer := cabinet.NewWriter(&out)
	file, err := writer.Create("Money.MNY")

	if err != nil {
		panic(err)
	}

	if _, err := file.Write(MoneyFile()); err != nil {
		panic(err)
	}

	if err := writer.Close(); err != nil {
		panic(err)
	}

	return out.buffer
}

// memoryWriteSeeker is an in-memory io.WriteSeeker with file semantics, so the
// cabinet writer can position its headers without touching the disk.
type memoryWriteSeeker struct {
	buffer []byte
	offset int64
}

// Write writes p at the current offset, extending the buffer with zeros when
// the offset lies past its end, like a file does.
func (m *memoryWriteSeeker) Write(p []byte) (int, error) {
	if m.offset > int64(len(m.buffer)) {
		m.buffer = append(m.buffer, make([]byte, m.offset-int64(len(m.buffer)))...)
	}

	if m.offset == int64(len(m.buffer)) {
		m.buffer = append(m.buffer, p...)
		m.offset = int64(len(m.buffer))

		return len(p), nil
	}

	n := copy(m.buffer[m.offset:], p)
	m.offset += int64(n)

	if n < len(p) {
		m.buffer = append(m.buffer, p[n:]...)
		m.offset = int64(len(m.buffer))
	}

	return len(p), nil
}

// Seek moves the write offset relative to the start, the current offset, or
// the end of the buffer.
func (m *memoryWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		m.offset = offset
	case io.SeekCurrent:
		m.offset += offset
	case io.SeekEnd:
		m.offset = int64(len(m.buffer)) + offset
	default:
		return 0, io.ErrUnexpectedEOF
	}

	if m.offset < 0 {
		m.offset = 0
	}

	return m.offset, nil
}

// amount returns the value a Money amount field stores for the given amount in
// hundredths of the currency: Money keeps four decimal places.
func amount(minor int64) uint64 {
	return uint64(int64(minor) * (moneyScale / 100))
}

// dateBits stores a date as the engine does: an IEEE 754 double counting days
// from 1899-12-30.
func dateBits(day time.Time) uint64 {
	days := day.Sub(jetEpoch).Hours() / 24

	return math.Float64bits(days)
}

// accountRow lays out one ACCT row.
func accountRow(handle, currency int64, name string) []byte {
	fixed := make([]byte, 8)
	putU32(fixed, 0, uint32(handle))
	putU32(fixed, 4, uint32(currency))

	return packRow(3, []byte{0x07}, fixed, [][]byte{utf16Bytes(name)})
}

// categoryRow lays out one CAT row.
func categoryRow(handle, parent int64, name string) []byte {
	fixed := make([]byte, 8)
	putU32(fixed, 0, uint32(handle))
	putU32(fixed, 4, uint32(parent))

	return packRow(3, []byte{0x07}, fixed, [][]byte{utf16Bytes(name)})
}

// transactionRow lays out one TRN row.
func transactionRow(handle, account, link, category, security int64, amount uint64, memo string, day uint64) []byte {
	fixed := make([]byte, 36)
	putU32(fixed, 0, uint32(handle))
	putU32(fixed, 4, uint32(account))
	putU32(fixed, 8, uint32(link))
	putU32(fixed, 12, uint32(category))
	putU32(fixed, 16, uint32(security))
	putU64(fixed, 20, amount)
	putU64(fixed, 28, day)

	return packRow(8, []byte{0xFF}, fixed, [][]byte{memoBytes(memo)})
}

// catalogRow lays out one catalog row, which names the object stored on the
// specified definition page.
func catalogRow(objectPage int, name string) []byte {
	fixed := make([]byte, 4)
	putU32(fixed, 0, uint32(objectPage))

	return packRow(1, []byte{0x01}, fixed, [][]byte{utf16Bytes(name)})
}

// memoBytes stores a memo the way the engine does: a twelve byte long value
// header whose inline flag is set, followed by the UTF-16 text.
func memoBytes(text string) []byte {
	encoded := utf16Bytes(text)
	header := make([]byte, longValueOverhead)
	putU32(header, 0, longValueInline|uint32(len(encoded)))

	return append(header, encoded...)
}

// longInt returns the four little endian bytes of a long integer field.
func longInt(value int64) []byte {
	fixed := make([]byte, 4)
	putU32(fixed, 0, uint32(value))

	return fixed
}

// definitionPage lays out a table definition that fits on one page.
func definitionPage(rowCount, firstDataPage int, columns []column) []byte {
	variableCount := 0

	for _, col := range columns {
		if !col.fixed {
			variableCount++
		}
	}

	stream := make([]byte, definitionColumnsOffset)
	stream[0] = msisam.PageTypeDefinition
	putU32(stream, definitionRowCountOffset, uint32(rowCount))
	putU16(stream, definitionVariableColumnCountOffset, variableCount)
	putU16(stream, definitionColumnCountOffset, len(columns))
	putU16(stream, definitionFirstDataPageOffset, firstDataPage)

	for index, col := range columns {
		entry := make([]byte, definitionColumnEntrySize)
		entry[columnTypeOffset] = byte(col.kind)
		putU16(entry, columnNumberOffset, index)
		putU16(entry, columnVariableIndexOffset, col.variable)
		putU16(entry, columnFixedOffsetOffset, col.offset)
		putU16(entry, columnSizeOffset, col.size)

		if col.fixed {
			entry[columnFlagsOffset] = columnFlagFixed
		}

		stream = append(stream, entry...)
	}

	for _, col := range columns {
		encoded := utf16Bytes(col.name)
		length := make([]byte, 2)
		putU16(length, 0, len(encoded))
		stream = append(stream, length...)
		stream = append(stream, encoded...)
	}

	if len(stream) > msisam.PageSize {
		panic("synthetic: table definition does not fit on one page")
	}

	page := make([]byte, msisam.PageSize)
	copy(page, stream)

	return page
}

// dataPage lays out a data page that belongs to the specified definition page.
func dataPage(owner int, rows [][]byte) []byte {
	page := make([]byte, msisam.PageSize)
	page[0] = msisam.PageTypeData
	putU32(page, pageLinkOffset, uint32(owner))
	putU16(page, rowCountOffset, len(rows))

	position := msisam.PageSize

	for index, row := range rows {
		position -= len(row)
		copy(page[position:], row)
		putU16(page, rowStartTable+index*2, position)
	}

	return page
}

// packRow lays out one row: the column count, the fixed columns, the variable
// columns and, growing down from the end of the row, the variable column
// offsets, the variable column count and the not-null mask.
func packRow(columnCount int, notNull []byte, fixed []byte, variables [][]byte) []byte {
	row := make([]byte, 0, msisam.PageSize)
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

// utf16Bytes encodes text the way the engine stores it: UTF-16, little endian.
func utf16Bytes(text string) []byte {
	units := utf16.Encode([]rune(text))
	out := make([]byte, 0, len(units)*2)

	for _, unit := range units {
		out = append(out, byte(unit), byte(unit>>8))
	}

	return out
}

// scrambleHeader scrambles the region of the header page the engine keeps out
// of plain sight. The same operation restores it, so the reader can unscramble
// it with the identical key.
func scrambleHeader(page []byte) {
	region := page[encryptedHeaderOffset : encryptedHeaderOffset+encryptedHeaderLength]
	copy(region, rc4(headerKey[:], region))
}

// rc4 applies the RC4 stream cipher with the specified key, mirroring the
// msisam reader's unscrambling.
func rc4(key, data []byte) []byte {
	var state [256]byte

	for index := range state {
		state[index] = byte(index)
	}

	for index, swap := 0, 0; index < len(state); index++ {
		swap = (swap + int(state[index]) + int(key[index%len(key)])) & 0xFF
		state[index], state[swap] = state[swap], state[index]
	}

	output := make([]byte, len(data))
	left, right := 0, 0

	for index, value := range data {
		left = (left + 1) & 0xFF
		right = (right + int(state[left])) & 0xFF
		state[left], state[right] = state[right], state[left]
		output[index] = value ^ state[(int(state[left])+int(state[right]))&0xFF]
	}

	return output
}

// putU16 writes a little endian 16 bit integer.
func putU16(buffer []byte, offset, value int) {
	binary.LittleEndian.PutUint16(buffer[offset:], uint16(value))
}

// putU32 writes a little endian 32 bit integer.
func putU32(buffer []byte, offset int, value uint32) {
	binary.LittleEndian.PutUint32(buffer[offset:], value)
}

// putU64 writes a little endian 64 bit integer.
func putU64(buffer []byte, offset int, value uint64) {
	binary.LittleEndian.PutUint64(buffer[offset:], value)
}
