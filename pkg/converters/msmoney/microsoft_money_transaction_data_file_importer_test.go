// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/converter"
	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
	"github.com/James-HoneyBadger/honey-ledger/pkg/models"
)

// buildDatabaseHeader returns data whose first twenty bytes look like a Jet
// database header, which is what a Microsoft Money data file starts with.
func buildDatabaseHeader(signature string) []byte {
	data := make([]byte, 128)
	copy(data[4:], []byte(signature))
	return data
}

func TestIsMicrosoftMoneyDataFile(t *testing.T) {
	assert.True(t, IsMicrosoftMoneyDataFile(buildDatabaseHeader("Standard Jet DB")))
	assert.True(t, IsMicrosoftMoneyDataFile(buildDatabaseHeader("Standard ACE DB")))
	assert.False(t, IsMicrosoftMoneyDataFile([]byte("SQLite format 3\x00")))
	assert.False(t, IsMicrosoftMoneyDataFile([]byte("!Type:Bank\nD01/02/2024\nT-1.00\n^")))
	assert.False(t, IsMicrosoftMoneyDataFile(nil))
	assert.False(t, IsMicrosoftMoneyDataFile(buildDatabaseHeader("Standard Jet DB")[:18]))
}

func TestMicrosoftMoneyTransactionDataFileParseImportedData(t *testing.T) {
	importer := MicrosoftMoneyTransactionDataFileImporter
	context := core.NewNullContext()

	user := &models.User{
		Uid:             1234567890,
		DefaultCurrency: "USD",
	}

	_, _, _, _, _, _, err := importer.ParseImportedData(context, user, buildDatabaseHeader("Standard Jet DB"), nil, converter.DefaultImporterOptions, nil, nil, nil, nil, nil)
	assert.Equal(t, errs.ErrImportMicrosoftMoneyDataFileNotReadable, err)

	_, _, _, _, _, _, err = importer.ParseImportedData(context, user, []byte("!Type:Bank\nD01/02/2024\nT-1.00\n^"), nil, converter.DefaultImporterOptions, nil, nil, nil, nil, nil)
	assert.Equal(t, errs.ErrImportFileTypeNotSupported, err)
}
