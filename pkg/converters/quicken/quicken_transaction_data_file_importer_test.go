// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package quicken

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/converter"
	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
	"github.com/James-HoneyBadger/honey-ledger/pkg/models"
)

func TestIsSQLiteDatabase(t *testing.T) {
	assert.True(t, IsSQLiteDatabase([]byte("SQLite format 3\x00\x10\x00\x01\x01")))
	assert.False(t, IsSQLiteDatabase([]byte("SQLite format 2\x00")))
	assert.False(t, IsSQLiteDatabase(nil))
}

func TestQuickenTransactionDataFileParseImportedData(t *testing.T) {
	importer := QuickenTransactionDataFileImporter
	context := core.NewNullContext()

	user := &models.User{
		Uid:             1234567890,
		DefaultCurrency: "USD",
	}

	_, _, _, _, _, _, err := importer.ParseImportedData(context, user, []byte("SQLite format 3\x00\x10\x00\x01\x01"), nil, converter.DefaultImporterOptions, nil, nil, nil, nil, nil)
	assert.Equal(t, errs.ErrImportQuickenDataFileNotReadable, err)

	_, _, _, _, _, _, err = importer.ParseImportedData(context, user, []byte("\x00\x01\x00\x00QDF\x00\x00"), nil, converter.DefaultImporterOptions, nil, nil, nil, nil, nil)
	assert.Equal(t, errs.ErrImportQuickenDataFileNotReadable, err)
}
