// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package quicken

import (
	"bytes"
	"time"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/converter"
	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
	"github.com/James-HoneyBadger/honey-ledger/pkg/log"
	"github.com/James-HoneyBadger/honey-ledger/pkg/models"
)

// sqliteDatabaseSignature marks a SQLite database, which is what Quicken for
// Mac stores its data in. Quicken for Windows uses an undocumented B-tree
// layout with no published signature.
var sqliteDatabaseSignature = []byte("SQLite format 3\x00")

// quickenTransactionDataFileImporter defines the structure of Quicken data file importer
type quickenTransactionDataFileImporter struct{}

// Initialize a Quicken transaction data file importer singleton instance
var QuickenTransactionDataFileImporter = &quickenTransactionDataFileImporter{}

// IsSQLiteDatabase returns whether the data is a SQLite database
func IsSQLiteDatabase(data []byte) bool {
	return bytes.HasPrefix(data, sqliteDatabaseSignature)
}

// ParseImportedData reports that a Quicken data file cannot be read directly
func (c *quickenTransactionDataFileImporter) ParseImportedData(ctx core.Context, user *models.User, data []byte, defaultTimezone *time.Location, additionalOptions converter.TransactionDataImporterOptions, accountMap map[string]*models.Account, expenseCategoryMap map[string]map[string]*models.TransactionCategory, incomeCategoryMap map[string]map[string]*models.TransactionCategory, transferCategoryMap map[string]map[string]*models.TransactionCategory, tagMap map[string]*models.TransactionTag) (models.ImportedTransactionSlice, []*models.Account, []*models.TransactionCategory, []*models.TransactionCategory, []*models.TransactionCategory, []*models.TransactionTag, error) {
	if IsSQLiteDatabase(data) {
		log.Warnf(ctx, "[quicken_transaction_data_file_importer.ParseImportedData] the Quicken data file of user \"uid:%d\" is a SQLite database with a private schema", user.Uid)
	} else {
		log.Warnf(ctx, "[quicken_transaction_data_file_importer.ParseImportedData] the Quicken data file of user \"uid:%d\" uses Quicken's undocumented database layout", user.Uid)
	}

	return nil, nil, nil, nil, nil, nil, errs.ErrImportQuickenDataFileNotReadable
}
