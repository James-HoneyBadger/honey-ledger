// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import (
	"bytes"
	"time"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/converter"
	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
	"github.com/James-HoneyBadger/honey-ledger/pkg/log"
	"github.com/James-HoneyBadger/honey-ledger/pkg/models"
)

// microsoftMoneyDatabaseSignatures are the engine signatures a Microsoft Money
// data file carries at byte offset 4. Money keeps its tables in a Jet database,
// so the header is shaped like an Access database even though the page layout
// is Money's own MSISAM variant, which standard Jet drivers refuse to open.
var microsoftMoneyDatabaseSignatures = [][]byte{
	[]byte("Standard Jet DB"),
	[]byte("Standard ACE DB"),
}

// microsoftMoneyTransactionDataFileImporter defines the structure of Microsoft Money data file importer
type microsoftMoneyTransactionDataFileImporter struct{}

// Initialize a Microsoft Money transaction data file importer singleton instance
var MicrosoftMoneyTransactionDataFileImporter = &microsoftMoneyTransactionDataFileImporter{}

// IsMicrosoftMoneyDataFile returns whether the data carries a Microsoft Money (Jet/MSISAM) header
func IsMicrosoftMoneyDataFile(data []byte) bool {
	for _, signature := range microsoftMoneyDatabaseSignatures {
		if len(data) >= 4+len(signature) && bytes.Equal(data[4:4+len(signature)], signature) {
			return true
		}
	}

	return false
}

// ParseImportedData reports that a Microsoft Money data file cannot be read directly
func (c *microsoftMoneyTransactionDataFileImporter) ParseImportedData(ctx core.Context, user *models.User, data []byte, defaultTimezone *time.Location, additionalOptions converter.TransactionDataImporterOptions, accountMap map[string]*models.Account, expenseCategoryMap map[string]map[string]*models.TransactionCategory, incomeCategoryMap map[string]map[string]*models.TransactionCategory, transferCategoryMap map[string]map[string]*models.TransactionCategory, tagMap map[string]*models.TransactionTag) (models.ImportedTransactionSlice, []*models.Account, []*models.TransactionCategory, []*models.TransactionCategory, []*models.TransactionCategory, []*models.TransactionTag, error) {
	if !IsMicrosoftMoneyDataFile(data) {
		log.Warnf(ctx, "[microsoft_money_transaction_data_file_importer.ParseImportedData] the import file of user \"uid:%d\" has no Jet database header", user.Uid)
		return nil, nil, nil, nil, nil, nil, errs.ErrImportFileTypeNotSupported
	}

	log.Warnf(ctx, "[microsoft_money_transaction_data_file_importer.ParseImportedData] the Microsoft Money data file of user \"uid:%d\" uses the undocumented MSISAM page layout", user.Uid)

	return nil, nil, nil, nil, nil, nil, errs.ErrImportMicrosoftMoneyDataFileNotReadable
}
