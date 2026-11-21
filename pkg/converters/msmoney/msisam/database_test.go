// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msisam

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func buildDatabase(pageTypes ...byte) []byte {
	data := make([]byte, PageSize*(len(pageTypes)+1))
	copy(data[magicOffset:], []byte(Magic))

	for index, pageType := range pageTypes {
		data[(index+1)*PageSize] = pageType
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

func TestOpenPages(t *testing.T) {
	data := buildDatabase(PageTypeData, PageTypeDefinition, PageTypeData)
	data[len(data)-16] = 0xFF // trailing bytes of a partial page

	database, err := Open(data)
	assert.NoError(t, err)
	assert.Equal(t, 4, database.PageCount())

	assert.Equal(t, PageTypeHeader, database.PageType(0))
	assert.Equal(t, PageTypeData, database.PageType(1))
	assert.Equal(t, PageTypeDefinition, database.PageType(2))
	assert.Equal(t, PageTypeData, database.PageType(3))
	assert.Nil(t, database.Page(4))
	assert.False(t, database.IsValidPageIndex(-1))

	assert.Equal(t, map[byte]int{PageTypeHeader: 1, PageTypeData: 2, PageTypeDefinition: 1}, database.PageTypeCounts())
}

// TestOpenRealDatabase runs only when a Microsoft Money data file is supplied,
// because the format cannot be committed to the repository.
func TestOpenRealDatabase(t *testing.T) {
	path := os.Getenv("MSISAM_SAMPLE_FILE")

	if path == "" {
		t.Skip("set MSISAM_SAMPLE_FILE to a Microsoft Money data file")
	}

	data, err := os.ReadFile(path)
	assert.NoError(t, err)

	database, err := Open(data)
	assert.NoError(t, err)
	assert.Greater(t, database.PageCount(), 100)

	counts := database.PageTypeCounts()
	assert.Greater(t, counts[PageTypeData], 10)
	assert.Greater(t, counts[PageTypeDefinition], 0)
}
