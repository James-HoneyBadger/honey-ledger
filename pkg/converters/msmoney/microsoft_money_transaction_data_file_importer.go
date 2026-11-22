// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import (
	"bytes"
	"time"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/converter"
	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/msmoney/msisam"
	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
	"github.com/James-HoneyBadger/honey-ledger/pkg/log"
	"github.com/James-HoneyBadger/honey-ledger/pkg/models"
)

// microsoftMoneyDatabaseSignature is the engine name a Microsoft Money data file
// carries at byte offset 4. Money keeps its tables in a Jet database, but names
// the engine MSISAM, which is why no Jet or Access driver opens the file.
var microsoftMoneyDatabaseSignature = []byte(msisam.Magic)

// microsoftMoneyTransactionDataFileImporter defines the structure of Microsoft Money data file importer
type microsoftMoneyTransactionDataFileImporter struct{}

// Initialize a Microsoft Money transaction data file importer singleton instance
var MicrosoftMoneyTransactionDataFileImporter = &microsoftMoneyTransactionDataFileImporter{}

// IsMicrosoftMoneyDataFile returns whether the data carries a Microsoft Money (MSISAM) header
func IsMicrosoftMoneyDataFile(data []byte) bool {
	return len(data) >= 4+len(microsoftMoneyDatabaseSignature) && bytes.Equal(data[4:4+len(microsoftMoneyDatabaseSignature)], microsoftMoneyDatabaseSignature)
}

// ParseImportedData returns the imported data by parsing the Microsoft Money
// transaction data
func (c *microsoftMoneyTransactionDataFileImporter) ParseImportedData(ctx core.Context, user *models.User, data []byte, defaultTimezone *time.Location, additionalOptions converter.TransactionDataImporterOptions, accountMap map[string]*models.Account, expenseCategoryMap map[string]map[string]*models.TransactionCategory, incomeCategoryMap map[string]map[string]*models.TransactionCategory, transferCategoryMap map[string]map[string]*models.TransactionCategory, tagMap map[string]*models.TransactionTag) (models.ImportedTransactionSlice, []*models.Account, []*models.TransactionCategory, []*models.TransactionCategory, []*models.TransactionCategory, []*models.TransactionTag, error) {
	if !IsMicrosoftMoneyDataFile(data) {
		log.Warnf(ctx, "[microsoft_money_transaction_data_file_importer.ParseImportedData] the import file of user \"uid:%d\" has no Microsoft Money database header", user.Uid)
		return nil, nil, nil, nil, nil, nil, errs.ErrImportFileTypeNotSupported
	}

	moneyData, err := readMicrosoftMoneyData(ctx, data)

	if err != nil {
		return nil, nil, nil, nil, nil, nil, err
	}

	if moneyData.Skipped > 0 {
		log.Warnf(ctx, "[microsoft_money_transaction_data_file_importer.ParseImportedData] skipped %d rows of the Microsoft Money data file of user \"uid:%d\", because they are investment activity, a split total, or have no amount, account or date", moneyData.Skipped, user.Uid)
	}

	if len(moneyData.Transactions) < 1 {
		return nil, nil, nil, nil, nil, nil, errs.ErrNotFoundTransactionDataInFile
	}

	log.Infof(ctx, "[microsoft_money_transaction_data_file_importer.ParseImportedData] parsed %d transactions from the Microsoft Money data file of user \"uid:%d\"", len(moneyData.Transactions), user.Uid)

	dataTableImporter := converter.CreateNewSimpleImporterWithTypeNameMapping(microsoftMoneyTransactionTypeNameMapping)

	return dataTableImporter.ParseImportedData(ctx, user, createNewMicrosoftMoneyTransactionDataTable(moneyData.Transactions), defaultTimezone, additionalOptions, accountMap, expenseCategoryMap, incomeCategoryMap, transferCategoryMap, tagMap)
}
