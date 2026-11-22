// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/abemedia/go-cabinet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/converter"
	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/msmoney/msisam"
	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
	"github.com/James-HoneyBadger/honey-ledger/pkg/models"
)

// memoryWriteSeeker backs a cabinet written entirely in memory, so the tests do
// not touch the filesystem.
type memoryWriteSeeker struct {
	buffer []byte
	offset int64
}

func (w *memoryWriteSeeker) Write(data []byte) (int, error) {
	end := w.offset + int64(len(data))

	if end > int64(len(w.buffer)) {
		w.buffer = append(w.buffer, make([]byte, end-int64(len(w.buffer)))...)
	}

	copy(w.buffer[w.offset:], data)
	w.offset = end

	return len(data), nil
}

func (w *memoryWriteSeeker) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		w.offset = offset
	case io.SeekCurrent:
		w.offset += offset
	case io.SeekEnd:
		w.offset = int64(len(w.buffer)) + offset
	}

	return w.offset, nil
}

// buildBackupArchive writes a cabinet holding the given members with the given
// compression, which is what a Microsoft Money backup file is.
func buildBackupArchive(t *testing.T, compression cabinet.Compression, members map[string][]byte, order []string) []byte {
	t.Helper()

	writer := &memoryWriteSeeker{}
	cabinetWriter := cabinet.NewWriter(writer)

	require.NoError(t, cabinetWriter.SetCompression(compression))

	for _, name := range order {
		fileWriter, err := cabinetWriter.Create(name)
		require.NoError(t, err)

		_, err = fileWriter.Write(members[name])
		require.NoError(t, err)
	}

	require.NoError(t, cabinetWriter.Close())

	return writer.buffer
}

// sampleMoneyDatabase returns bytes that begin like a Microsoft Money data file.
func sampleMoneyDatabase(t *testing.T) []byte {
	t.Helper()

	database := make([]byte, 4096)
	copy(database[4:], []byte(msisam.Magic))

	for index := 4 + len(msisam.Magic); index < len(database); index++ {
		database[index] = byte(index % 251)
	}

	return database
}

func TestIsMicrosoftMoneyBackupFile(t *testing.T) {
	testCases := []struct {
		name     string
		data     []byte
		expected bool
	}{
		{name: "cabinet signature", data: []byte("MSCF\x00\x00\x00\x00"), expected: true},
		{name: "money database", data: append([]byte("\x00\x01\x00\x00"), []byte(msisam.Magic)...), expected: false},
		{name: "zip file", data: []byte("PK\x03\x04"), expected: false},
		{name: "empty", data: []byte{}, expected: false},
		{name: "truncated signature", data: []byte("MSC"), expected: false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			assert.Equal(t, testCase.expected, IsMicrosoftMoneyBackupFile(testCase.data))
		})
	}
}

func TestExtractMicrosoftMoneyDatabaseFromBackup(t *testing.T) {
	database := sampleMoneyDatabase(t)

	compressions := map[string]cabinet.Compression{
		"mszip": cabinet.MSZIP,
		"lzx":   cabinet.LZXWindow(21),
	}

	for name, compression := range compressions {
		t.Run(name, func(t *testing.T) {
			// Money names the member after the database it holds, not with a .mny
			// extension, so extraction cannot rely on the name.
			archive := buildBackupArchive(t, compression, map[string][]byte{
				"mbf943E.tmp": database,
			}, []string{"mbf943E.tmp"})

			assert.True(t, IsMicrosoftMoneyBackupFile(archive))

			extracted, err := extractMicrosoftMoneyDatabaseFromBackup(core.NewNullContext(), archive)
			require.NoError(t, err)
			assert.Equal(t, database, extracted)
			assert.True(t, IsMicrosoftMoneyDataFile(extracted))
		})
	}
}

func TestExtractMicrosoftMoneyDatabaseFromBackupSkipsOtherMembers(t *testing.T) {
	database := sampleMoneyDatabase(t)

	archive := buildBackupArchive(t, cabinet.MSZIP, map[string][]byte{
		"readme.txt":  []byte("this member is not a database"),
		"mbf1A2B.tmp": database,
	}, []string{"readme.txt", "mbf1A2B.tmp"})

	extracted, err := extractMicrosoftMoneyDatabaseFromBackup(core.NewNullContext(), archive)
	require.NoError(t, err)
	assert.Equal(t, database, extracted)
}

func TestExtractMicrosoftMoneyDatabaseFromBackupWithoutDatabase(t *testing.T) {
	archive := buildBackupArchive(t, cabinet.MSZIP, map[string][]byte{
		"readme.txt": []byte("this member is not a database"),
	}, []string{"readme.txt"})

	_, err := extractMicrosoftMoneyDatabaseFromBackup(core.NewNullContext(), archive)
	assert.ErrorIs(t, err, errs.ErrImportMicrosoftMoneyBackupFileNotReadable)
}

func TestExtractMicrosoftMoneyDatabaseFromBackupRejectsCorruptArchive(t *testing.T) {
	archive := append([]byte("MSCF"), bytes.Repeat([]byte{0}, 32)...)

	_, err := extractMicrosoftMoneyDatabaseFromBackup(core.NewNullContext(), archive)
	assert.ErrorIs(t, err, errs.ErrImportMicrosoftMoneyBackupFileNotReadable)
}

func TestExtractMicrosoftMoneyDatabaseFromBackupRejectsOversizedDatabase(t *testing.T) {
	originalLimit := maxMicrosoftMoneyBackupDatabaseSize
	maxMicrosoftMoneyBackupDatabaseSize = 1024
	defer func() {
		maxMicrosoftMoneyBackupDatabaseSize = originalLimit
	}()

	// The member starts with a valid Money header, so only the size cap can
	// reject it.
	database := sampleMoneyDatabase(t)
	require.Greater(t, int64(len(database)), maxMicrosoftMoneyBackupDatabaseSize)

	archive := buildBackupArchive(t, cabinet.MSZIP, map[string][]byte{
		"mbf943E.tmp": database,
	}, []string{"mbf943E.tmp"})

	_, err := extractMicrosoftMoneyDatabaseFromBackup(core.NewNullContext(), archive)
	assert.ErrorIs(t, err, errs.ErrImportMicrosoftMoneyBackupFileTooLarge)
}

func TestExtractMicrosoftMoneyDatabaseFromBackupLeavesInputUntouched(t *testing.T) {
	database := sampleMoneyDatabase(t)

	archive := buildBackupArchive(t, cabinet.MSZIP, map[string][]byte{
		"mbf943E.tmp": database,
	}, []string{"mbf943E.tmp"})

	original := make([]byte, len(archive))
	copy(original, archive)

	extracted, err := extractMicrosoftMoneyDatabaseFromBackup(core.NewNullContext(), archive)
	require.NoError(t, err)

	// Extraction is in-memory and returns a copy: the archive is unchanged and
	// changing the result cannot corrupt it.
	assert.Equal(t, original, archive)

	extracted[0] = 0xFF
	assert.Equal(t, original, archive)
}

func TestParseImportedDataAcceptsBackupFile(t *testing.T) {
	// The backup wraps the same database the .mny tests use, so a backup must
	// produce exactly the same transactions as the data file inside it.
	database := moneyTestDatabase()

	archive := buildBackupArchive(t, cabinet.MSZIP, map[string][]byte{
		"mbf943E.tmp": database,
	}, []string{"mbf943E.tmp"})

	user := &models.User{
		Uid:             1234567890,
		DefaultCurrency: "USD",
	}

	imported, accounts, expenseCategories, incomeCategories, transferCategories, _, err := MicrosoftMoneyTransactionDataFileImporter.ParseImportedData(
		core.NewNullContext(), user, archive, time.UTC, converter.DefaultImporterOptions, nil, nil, nil, nil, nil,
	)

	require.NoError(t, err)
	require.Len(t, imported, 5)
	assert.NotEmpty(t, accounts)
	assert.NotEmpty(t, expenseCategories)
	assert.NotEmpty(t, incomeCategories)
	assert.NotEmpty(t, transferCategories)
}
