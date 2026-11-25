// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import (
	"time"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/converter"
	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
	"github.com/James-HoneyBadger/honey-ledger/pkg/log"
	"github.com/James-HoneyBadger/honey-ledger/pkg/models"
)

// microsoftMoneyTransactionDataFileImporter defines the structure of Microsoft Money data file importer
type microsoftMoneyTransactionDataFileImporter struct{}

// Initialize a Microsoft Money transaction data file importer singleton instance
var MicrosoftMoneyTransactionDataFileImporter = &microsoftMoneyTransactionDataFileImporter{}

// ParseImportedData returns the imported data by parsing the Microsoft Money
// transaction data
func (c *microsoftMoneyTransactionDataFileImporter) ParseImportedData(ctx core.Context, user *models.User, data []byte, defaultTimezone *time.Location, additionalOptions converter.TransactionDataImporterOptions, accountMap map[string]*models.Account, expenseCategoryMap map[string]map[string]*models.TransactionCategory, incomeCategoryMap map[string]map[string]*models.TransactionCategory, transferCategoryMap map[string]map[string]*models.TransactionCategory, tagMap map[string]*models.TransactionTag) (models.ImportedTransactionSlice, []*models.Account, []*models.TransactionCategory, []*models.TransactionCategory, []*models.TransactionCategory, []*models.TransactionTag, error) {
	// A backup (.mbf) is a cabinet wrapping the database, so the extraction
	// unwraps it and treats the database inside exactly like a selected .mny
	// file.
	moneyData, err := ExtractMicrosoftMoneyData(ctx, data)

	if err != nil {
		log.Errorf(ctx, "[microsoft_money_transaction_data_file_importer.ParseImportedData] cannot extract the Microsoft Money data of the file of user \"uid:%d\", because %s", user.Uid, err.Error())
		return nil, nil, nil, nil, nil, nil, mappedExtractionError(err)
	}

	if moneyData.Skipped > 0 {
		log.Warnf(ctx, "[microsoft_money_transaction_data_file_importer.ParseImportedData] skipped %d rows of the Microsoft Money data file of user \"uid:%d\", because they are investment activity, a split total, or have no amount, account or date", moneyData.Skipped, user.Uid)
	}

	if !moneyData.Diagnostics.IsEmpty() {
		log.Warnf(ctx, "[microsoft_money_transaction_data_file_importer.ParseImportedData] decoding problems in the Microsoft Money data file of user \"uid:%d\": %s", user.Uid, moneyData.Diagnostics.Summary())
	}

	if len(moneyData.Transactions) < 1 {
		return nil, nil, nil, nil, nil, nil, errs.ErrNotFoundTransactionDataInFile
	}

	log.Infof(ctx, "[microsoft_money_transaction_data_file_importer.ParseImportedData] parsed %d transactions from the Microsoft Money data file of user \"uid:%d\"", len(moneyData.Transactions), user.Uid)

	dataTableImporter := converter.CreateNewSimpleImporterWithTypeNameMapping(microsoftMoneyTransactionTypeNameMapping)

	return dataTableImporter.ParseImportedData(ctx, user, createNewMicrosoftMoneyTransactionDataTable(moneyData.Transactions), defaultTimezone, additionalOptions, accountMap, expenseCategoryMap, incomeCategoryMap, transferCategoryMap, tagMap)
}
