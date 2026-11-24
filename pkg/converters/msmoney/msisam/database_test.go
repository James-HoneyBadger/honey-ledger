// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msisam

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func putU16(buffer []byte, offset, value int) {
	buffer[offset] = byte(value)
	buffer[offset+1] = byte(value >> 8)
}

func putU32(buffer []byte, offset int, value uint32) {
	buffer[offset] = byte(value)
	buffer[offset+1] = byte(value >> 8)
	buffer[offset+2] = byte(value >> 16)
	buffer[offset+3] = byte(value >> 24)
}

func putU64(buffer []byte, offset int, value uint64) {
	putU32(buffer, offset, uint32(value))
	putU32(buffer, offset+4, uint32(value>>32))
}

// sealHeader writes the scrambled header fields that Open reads back, leaving a
// database key of zero unless one is requested.
func sealHeader(data []byte, key uint32) {
	header := make([]byte, encryptedHeaderLength)

	putU32(header, databaseKeyOffset-encryptedHeaderOffset, key)
	putU16(header, codePageOffset-encryptedHeaderOffset, 1252)
	putU16(header, languageOffset-encryptedHeaderOffset, 1033)
	copy(data[encryptedHeaderOffset:], rc4(headerKey[:], header))
}

// buildDatabase returns a database whose page 0 is a header followed by the
// specified page type markers.
func buildDatabase(pageTypes ...byte) []byte {
	data := make([]byte, PageSize*(len(pageTypes)+1))
	copy(data[magicOffset:], []byte(Magic))
	data[versionOffset] = versionJet4
	sealHeader(data, 0)

	for index, pageType := range pageTypes {
		data[(index+1)*PageSize] = pageType
	}

	return data
}

// buildDatabaseFromPages returns a database whose header is followed by the
// specified pages.
func buildDatabaseFromPages(pages ...[]byte) []byte {
	data := make([]byte, PageSize*(len(pages)+1))
	copy(data[magicOffset:], []byte(Magic))
	data[versionOffset] = versionJet4
	sealHeader(data, 0)

	for index, page := range pages {
		copy(data[(index+1)*PageSize:], page)
	}

	return data
}

func TestOpenRejectsShortData(t *testing.T) {
	_, err := Open(make([]byte, PageSize-1))
	assert.Equal(t, ErrFileTooSmall, err)
}

func TestOpenRejectsOtherDatabases(t *testing.T) {
	data := buildDatabase(PageTypeData)
	copy(data[magicOffset:], []byte("Standard Jet DB"))

	_, err := Open(data)
	assert.Equal(t, ErrNotMSISAMDatabase, err)
}

func TestOpenRejectsEncryptedDatabase(t *testing.T) {
	data := buildDatabase(PageTypeData)
	sealHeader(data, 0x1234)

	_, err := Open(data)
	assert.Equal(t, ErrEncryptedDatabase, err)
}

func TestOpenReadsHeader(t *testing.T) {
	database, err := Open(buildDatabase(PageTypeData))
	require.NoError(t, err)

	assert.Equal(t, byte(versionJet4), database.Version())
	assert.Equal(t, uint16(1252), database.CodePage())
	assert.Equal(t, uint16(1033), database.LanguageID())
}

func TestOpenPages(t *testing.T) {
	data := buildDatabase(PageTypeData, PageTypeDefinition, PageTypeData)
	data[len(data)-16] = 0xFF // trailing bytes of a partial page

	database, err := Open(data)
	require.NoError(t, err)
	assert.Equal(t, 4, database.PageCount())

	assert.Equal(t, PageTypeHeader, database.PageType(0))
	assert.Equal(t, PageTypeData, database.PageType(1))
	assert.Equal(t, PageTypeDefinition, database.PageType(2))
	assert.Equal(t, PageTypeData, database.PageType(3))
	assert.Nil(t, database.Page(4))
	assert.False(t, database.IsValidPageIndex(-1))

	assert.Equal(t, map[byte]int{PageTypeHeader: 1, PageTypeData: 2, PageTypeDefinition: 1}, database.PageTypeCounts())
}

func TestRc4RoundTrips(t *testing.T) {
	key := []byte{0xC7, 0xDA, 0x39, 0x6B}
	plain := []byte("the quick brown fox jumps over the lazy dog")

	assert.Equal(t, plain, rc4(key, rc4(key, plain)))
	assert.Len(t, rc4(key, plain), len(plain))
}

func TestExpandText(t *testing.T) {
	// Compressed text stores one byte per character until a zero byte switches
	// the engine back to UTF-16.
	assert.Equal(t, utf16Bytes("AB"), expandText([]byte{0x41, 0x00, 0x42, 0x00}), "uncompressed input")
	assert.Equal(t, utf16Bytes("AB"), expandText([]byte{0x41, 0x42}))
	assert.Equal(t, utf16Bytes("AB\u20ACC"), expandText([]byte{0x41, 0x42, 0x00, 0xAC, 0x20, 0x43, 0x00}))
}

// utf16Bytes returns the UTF-16 encoding of text with every high byte present.
func utf16Bytes(text string) []byte {
	units := []byte{}

	for _, character := range text {
		if character < 0x10000 {
			units = append(units, byte(character), byte(character>>8))

			continue
		}

		character -= 0x10000
		high := 0xD800 + (character >> 10)
		low := 0xDC00 + (character & 0x3FF)
		units = append(units, byte(high), byte(high>>8), byte(low), byte(low>>8))
	}

	return units
}

// TestOpenRealDatabase runs only when a Microsoft Money data file is supplied,
// because the format cannot be committed to the repository.
func TestOpenRealDatabase(t *testing.T) {
	path := os.Getenv("MSISAM_SAMPLE_FILE")

	if path == "" {
		t.Skip("set MSISAM_SAMPLE_FILE to a Microsoft Money data file")
	}

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	database, err := Open(data)
	require.NoError(t, err)

	assert.Greater(t, database.PageCount(), 100)
	assert.Equal(t, byte(versionJet4), database.Version())

	counts := database.PageTypeCounts()
	assert.Greater(t, counts[PageTypeData], 10)
	assert.Greater(t, counts[PageTypeDefinition], 0)
	assert.Greater(t, counts[PageTypeLeaf], 0)
}

// TestRealDatabaseTables reads the tables a Microsoft Money data file holds, and
// runs only when such a file is supplied.
func TestRealDatabaseTables(t *testing.T) {
	path := os.Getenv("MSISAM_SAMPLE_FILE")

	if path == "" {
		t.Skip("set MSISAM_SAMPLE_FILE to a Microsoft Money data file")
	}

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	database, err := Open(data)
	require.NoError(t, err)

	tables, err := database.Tables()
	require.NoError(t, err)
	assert.Greater(t, len(tables), 50, "a Money database holds dozens of tables")

	for _, name := range []string{"ACCT", "CAT", "TRN", "TRN_SPLIT"} {
		table, err := database.Table(name)
		require.NoError(t, err, "table %s", name)
		assert.Greater(t, table.Definition.RowCount, 0, "table %s", name)
	}

	accounts, err := database.Table("ACCT")
	require.NoError(t, err)

	names := []string{}
	visited := 0

	err = database.EachRow(accounts.Definition, func(fields []Field) error {
		visited++

		index := -1

		for position := range fields {
			if fields[position].Column.Name == "szFull" {
				index = position
			}
		}

		require.GreaterOrEqual(t, index, 0)

		name, err := database.String(fields[index])

		if err != nil {
			return err
		}

		names = append(names, name)

		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, accounts.Definition.RowCount, visited)
	assert.NotEmpty(t, names)

	for _, name := range names {
		assert.NotEmpty(t, name, "every account has a name")
	}

	transactions, err := database.Table("TRN")
	require.NoError(t, err)

	dates := 0

	err = database.EachRow(transactions.Definition, func(fields []Field) error {
		for position := range fields {
			if fields[position].Column.Name != "dt" {
				continue
			}

			date, ok := database.Time(fields[position])

			if !ok {
				continue
			}

			assert.False(t, date.IsZero())
			assert.True(t, date.Year() > 1980 && date.Year() < 2100, "date %s is plausible", date)

			dates++
		}

		return nil
	})

	require.NoError(t, err)
	assert.Greater(t, dates, 1000)
}
