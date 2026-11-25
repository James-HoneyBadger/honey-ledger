// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

// Package msisam reads Microsoft Money data files: a Jet 4 database whose
// header names the MSISAM engine instead of the Access engine, which is why no
// Jet or Access driver opens them.
//
// The package decodes the header, the table definitions, the catalog that names
// the tables and the values stored in their rows, so a Money file can be read
// without a copy of Money. It reads only; it never writes to a database.
//
// A database protected with a password encrypts every page, and Open reports it
// as ErrEncryptedDatabase.
package msisam

import (
	"errors"
	"fmt"
)

const (
	// Magic is the engine name stored at the start of the database header.
	Magic = "MSISAM Database"

	// PageSize is the fixed page size of a Money database, in bytes.
	PageSize = 4096

	// magicOffset is where the engine name starts in the header page.
	magicOffset = 4

	// versionOffset holds the engine version; 1 is the Jet 4 layout.
	versionOffset = 0x14

	// codePageOffset holds the Windows code page of non-unicode data.
	codePageOffset = 0x3C

	// databaseKeyOffset holds the key that encrypts every page when non-zero.
	databaseKeyOffset = 0x3E

	// languageOffset holds the Windows language identifier.
	languageOffset = 0x6E

	// encryptedHeaderOffset and encryptedHeaderLength describe the region of
	// the header page that is scrambled with headerKey, even in an unencrypted
	// database.
	encryptedHeaderOffset = 0x18
	encryptedHeaderLength = 128

	// pageLinkOffset holds the page that continues the current page's
	// definition or long value.
	pageLinkOffset = 4

	// pageContinuationHeaderSize is the header repeated at the start of a
	// continuation page.
	pageContinuationHeaderSize = 8

	// versionJet4 is the engine version of the supported layout.
	versionJet4 = 1
)

// headerKey scrambles the header fields that the engine keeps out of plain
// sight, whether or not the database is encrypted.
var headerKey = [4]byte{0xC7, 0xDA, 0x39, 0x6B}

// Page type markers written in the first byte of a page.
const (
	// PageTypeHeader marks the header page at index 0.
	PageTypeHeader byte = 0x00
	// PageTypeData marks a page that holds table rows.
	PageTypeData byte = 0x01
	// PageTypeDefinition marks a page that holds table definitions.
	PageTypeDefinition byte = 0x02
	// PageTypeIndex marks a page that holds an index.
	PageTypeIndex byte = 0x03
	// PageTypeLeaf marks a page that holds the overflow of a long value.
	PageTypeLeaf byte = 0x04
	// PageTypeUsageMap marks a page that records which pages a table owns.
	PageTypeUsageMap byte = 0x05
)

var (
	// ErrFileTooSmall reports a file that cannot hold a header page.
	ErrFileTooSmall = errors.New("msisam: file is smaller than one page")
	// ErrNotMSISAMDatabase reports data that does not carry the MSISAM header.
	ErrNotMSISAMDatabase = errors.New("msisam: file is not a Microsoft Money database")
	// ErrEncryptedDatabase reports a password protected database, whose pages
	// are scrambled with a key derived from the password.
	ErrEncryptedDatabase = errors.New("msisam: database is encrypted")
	// ErrUnsupportedVersion reports a database whose engine version is not the
	// Jet 4 layout this package reads, such as Money 2000 and earlier files
	// with 2048 byte pages.
	ErrUnsupportedVersion = errors.New("msisam: unsupported database version")
	// ErrInvalidPage reports a page index outside the file.
	ErrInvalidPage = errors.New("msisam: page index is outside the file")
	// ErrNotDefinitionPage reports a page that does not hold a table
	// definition.
	ErrNotDefinitionPage = errors.New("msisam: page does not hold a table definition")
	// ErrCorruptDefinition reports a table definition that cannot be decoded.
	ErrCorruptDefinition = errors.New("msisam: table definition is corrupt")
	// ErrCorruptRow reports a row that cannot be decoded.
	ErrCorruptRow = errors.New("msisam: row is corrupt")
	// ErrTableNotFound reports a table name the database does not hold.
	ErrTableNotFound = errors.New("msisam: table not found")
)

// Database provides read access to the pages of a Microsoft Money data file. A
// Database is not safe for concurrent use, because it records which pages belong
// to each table the first time it reads one.
type Database struct {
	data        []byte
	header      []byte
	pageCount   int
	dataPages   map[int][]int
	diagnostics *Diagnostics
}

// Open validates the database header and returns a reader over the pages.
func Open(data []byte) (*Database, error) {
	if len(data) < PageSize {
		return nil, ErrFileTooSmall
	}

	if string(data[magicOffset:magicOffset+len(Magic)]) != Magic {
		return nil, ErrNotMSISAMDatabase
	}

	// Only the Jet 4 layout is read. Money 2000 and earlier files keep the
	// same engine name but use 2048 byte pages and a different version byte;
	// reading them as Jet 4 would produce garbage, so they are reported
	// instead.
	if data[versionOffset] != versionJet4 {
		return nil, fmt.Errorf("%w: engine version %d", ErrUnsupportedVersion, data[versionOffset])
	}

	// A database ends with a partial page when Money appended trailing data, so
	// only whole pages are addressable.
	database := &Database{
		data:      data,
		header:    unscrambleHeader(data[:PageSize]),
		pageCount: len(data) / PageSize,
	}

	if database.encrypted() {
		return nil, ErrEncryptedDatabase
	}

	return database, nil
}

// unscrambleHeader returns a copy of the header page with its scrambled fields
// restored.
func unscrambleHeader(page []byte) []byte {
	header := make([]byte, len(page))
	copy(header, page)
	copy(header[encryptedHeaderOffset:], rc4(headerKey[:], header[encryptedHeaderOffset:encryptedHeaderOffset+encryptedHeaderLength]))

	return header
}

// rc4 applies the RC4 stream cipher with the specified key.
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

// Version returns the engine version of the database.
func (d *Database) Version() byte {
	return d.header[versionOffset]
}

// CodePage returns the Windows code page of the database.
func (d *Database) CodePage() uint16 {
	codePage, _ := readU16(d.header, codePageOffset)

	return uint16(codePage)
}

// LanguageID returns the Windows language identifier of the database.
func (d *Database) LanguageID() uint16 {
	language, _ := readU16(d.header, languageOffset)

	return uint16(language)
}

// encrypted reports whether the pages of the database are scrambled with a key
// derived from a password.
func (d *Database) encrypted() bool {
	key, _ := readU32(d.header, databaseKeyOffset)

	return key != 0
}

// PageCount returns the count of whole pages in the database.
func (d *Database) PageCount() int {
	return d.pageCount
}

// SetDiagnostics attaches a diagnostics sink the database reports decoding
// problems into. Without one, problems are silently skipped, as before the
// sink existed. Attach it before the first read.
func (d *Database) SetDiagnostics(diagnostics *Diagnostics) {
	d.diagnostics = diagnostics
}

// Diagnostics returns the attached sink, or nil.
func (d *Database) Diagnostics() *Diagnostics {
	return d.diagnostics
}

// IsValidPageIndex returns whether the index addresses a whole page.
func (d *Database) IsValidPageIndex(index int) bool {
	return index >= 0 && index < d.pageCount
}

// Page returns the bytes of the page at the specified index, or nil when the
// index is out of range.
func (d *Database) Page(index int) []byte {
	if !d.IsValidPageIndex(index) {
		return nil
	}

	return d.data[index*PageSize : (index+1)*PageSize]
}

// PageType returns the page type marker of the page at the specified index.
func (d *Database) PageType(index int) byte {
	page := d.Page(index)

	if page == nil {
		return 0
	}

	return page[0]
}

// PageTypeCounts returns the count of pages per page type marker.
func (d *Database) PageTypeCounts() map[byte]int {
	counts := make(map[byte]int)

	for index := 0; index < d.pageCount; index++ {
		counts[d.PageType(index)]++
	}

	return counts
}
