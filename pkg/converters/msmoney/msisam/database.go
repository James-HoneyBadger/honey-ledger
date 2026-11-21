// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

// Package msisam reads the page layout of Microsoft Money data files: a Jet 4
// database whose header names the MSISAM engine instead of the Access engine,
// with page-level metadata that Jet drivers reject.
package msisam

import (
	"errors"
)

const (
	// Magic is the engine name stored at the start of the database header.
	Magic = "MSISAM Database"

	// PageSize is the fixed page size of a Money database, in bytes.
	PageSize = 4096

	// magicOffset is where the engine name starts in the header page.
	magicOffset = 4
)

// Page type markers written in the first byte of a page.
const (
	// PageTypeHeader marks the header page at index 0.
	PageTypeHeader byte = 0x00
	// PageTypeData marks a page that holds table rows.
	PageTypeData byte = 0x01
	// PageTypeDefinition marks a page that holds table definitions.
	PageTypeDefinition byte = 0x04
)

var (
	// ErrFileTooSmall reports a file that cannot hold a header page.
	ErrFileTooSmall = errors.New("msisam: file is smaller than one page")
	// ErrNotMSISAMDatabase reports data that does not carry the MSISAM header.
	ErrNotMSISAMDatabase = errors.New("msisam: file is not a Microsoft Money database")
)

// Database provides read access to the pages of a Microsoft Money data file.
type Database struct {
	data      []byte
	pageCount int
}

// Open validates the database header and returns a reader over the pages.
func Open(data []byte) (*Database, error) {
	if len(data) < PageSize {
		return nil, ErrFileTooSmall
	}

	if string(data[magicOffset:magicOffset+len(Magic)]) != Magic {
		return nil, ErrNotMSISAMDatabase
	}

	// A database ends with a partial page when Money appended trailing data, so
	// only whole pages are addressable.
	return &Database{
		data:      data,
		pageCount: len(data) / PageSize,
	}, nil
}

// PageCount returns the count of whole pages in the database.
func (d *Database) PageCount() int {
	return d.pageCount
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
