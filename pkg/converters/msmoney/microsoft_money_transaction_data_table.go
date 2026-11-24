// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import (
	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/datatable"
	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/models"
	"github.com/James-HoneyBadger/honey-ledger/pkg/utils"
	engine "github.com/James-HoneyBadger/money-engine/msmoney"
)

var microsoftMoneyTransactionTypeNameMapping = map[models.TransactionType]string{
	models.TRANSACTION_TYPE_INCOME:   utils.IntToString(int(models.TRANSACTION_TYPE_INCOME)),
	models.TRANSACTION_TYPE_EXPENSE:  utils.IntToString(int(models.TRANSACTION_TYPE_EXPENSE)),
	models.TRANSACTION_TYPE_TRANSFER: utils.IntToString(int(models.TRANSACTION_TYPE_TRANSFER)),
}

// microsoftMoneyExtractedTypeNameMapping maps the extracted transaction kinds
// onto the type names the generic importer parses, keeping the values the same
// as the transaction type numbers of the ledger.
var microsoftMoneyExtractedTypeNameMapping = map[string]string{
	engine.ExtractedTypeIncome:   utils.IntToString(int(models.TRANSACTION_TYPE_INCOME)),
	engine.ExtractedTypeExpense:  utils.IntToString(int(models.TRANSACTION_TYPE_EXPENSE)),
	engine.ExtractedTypeTransfer: utils.IntToString(int(models.TRANSACTION_TYPE_TRANSFER)),
}

var microsoftMoneyTransactionSupportedColumns = map[datatable.TransactionDataTableColumn]bool{
	datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TIME:         true,
	datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TYPE:         true,
	datatable.TRANSACTION_DATA_TABLE_CATEGORY:                 true,
	datatable.TRANSACTION_DATA_TABLE_SUB_CATEGORY:             true,
	datatable.TRANSACTION_DATA_TABLE_ACCOUNT_NAME:             true,
	datatable.TRANSACTION_DATA_TABLE_ACCOUNT_CURRENCY:         true,
	datatable.TRANSACTION_DATA_TABLE_AMOUNT:                   true,
	datatable.TRANSACTION_DATA_TABLE_RELATED_ACCOUNT_NAME:     true,
	datatable.TRANSACTION_DATA_TABLE_RELATED_ACCOUNT_CURRENCY: true,
	datatable.TRANSACTION_DATA_TABLE_RELATED_AMOUNT:           true,
	datatable.TRANSACTION_DATA_TABLE_DESCRIPTION:              true,
}

// microsoftMoneyTransactionDataTable defines the structure of the Microsoft Money
// transaction data table
type microsoftMoneyTransactionDataTable struct {
	allData []engine.ExtractedTransaction
}

// microsoftMoneyTransactionDataRow defines the structure of the Microsoft Money
// transaction data row
type microsoftMoneyTransactionDataRow struct {
	items map[datatable.TransactionDataTableColumn]string
}

// microsoftMoneyTransactionDataRowIterator defines the structure of the Microsoft
// Money transaction data row iterator
type microsoftMoneyTransactionDataRowIterator struct {
	dataTable    *microsoftMoneyTransactionDataTable
	currentIndex int
}

// HasColumn returns whether the transaction data table has specified column
func (t *microsoftMoneyTransactionDataTable) HasColumn(column datatable.TransactionDataTableColumn) bool {
	_, exists := microsoftMoneyTransactionSupportedColumns[column]
	return exists
}

// TransactionRowCount returns the total count of transaction data row
func (t *microsoftMoneyTransactionDataTable) TransactionRowCount() int {
	return len(t.allData)
}

// TransactionRowIterator returns the iterator of transaction data row
func (t *microsoftMoneyTransactionDataTable) TransactionRowIterator() datatable.TransactionDataRowIterator {
	return &microsoftMoneyTransactionDataRowIterator{
		dataTable:    t,
		currentIndex: -1,
	}
}

// IsValid returns whether this row is valid data for importing
func (r *microsoftMoneyTransactionDataRow) IsValid() bool {
	return true
}

// GetData returns the data in the specified column type
func (r *microsoftMoneyTransactionDataRow) GetData(column datatable.TransactionDataTableColumn) string {
	if _, exists := microsoftMoneyTransactionSupportedColumns[column]; !exists {
		return ""
	}

	return r.items[column]
}

// HasNext returns whether the iterator does not reach the end
func (t *microsoftMoneyTransactionDataRowIterator) HasNext() bool {
	return t.currentIndex+1 < len(t.dataTable.allData)
}

// Next returns the next transaction data row
func (t *microsoftMoneyTransactionDataRowIterator) Next(ctx core.Context, user *models.User) (daraRow datatable.TransactionDataRow, err error) {
	if t.currentIndex+1 >= len(t.dataTable.allData) {
		return nil, nil
	}

	t.currentIndex++

	return &microsoftMoneyTransactionDataRow{
		items: buildMicrosoftMoneyTransactionRowItems(&t.dataTable.allData[t.currentIndex]),
	}, nil
}

// buildMicrosoftMoneyTransactionRowItems converts a transaction into the values
// of the columns the ledger imports.
func buildMicrosoftMoneyTransactionRowItems(transaction *engine.ExtractedTransaction) map[datatable.TransactionDataTableColumn]string {
	items := make(map[datatable.TransactionDataTableColumn]string, len(microsoftMoneyTransactionSupportedColumns))

	// A Money transaction stores a date without a time, so it is formatted in
	// its own location rather than converted to the default timezone.
	items[datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TIME] = utils.FormatUnixTimeToLongDateTime(transaction.Time.Unix(), nil)
	items[datatable.TRANSACTION_DATA_TABLE_TRANSACTION_TYPE] = microsoftMoneyExtractedTypeNameMapping[transaction.Type]
	items[datatable.TRANSACTION_DATA_TABLE_AMOUNT] = utils.FormatAmount(transaction.Amount)
	items[datatable.TRANSACTION_DATA_TABLE_ACCOUNT_NAME] = transaction.Account
	items[datatable.TRANSACTION_DATA_TABLE_ACCOUNT_CURRENCY] = transaction.Currency

	if transaction.TransferAccount != "" || transaction.TransferCurrency != "" {
		items[datatable.TRANSACTION_DATA_TABLE_RELATED_ACCOUNT_NAME] = transaction.TransferAccount
		items[datatable.TRANSACTION_DATA_TABLE_RELATED_ACCOUNT_CURRENCY] = transaction.TransferCurrency
	}

	if transaction.TransferAmount != 0 {
		items[datatable.TRANSACTION_DATA_TABLE_RELATED_AMOUNT] = utils.FormatAmount(transaction.TransferAmount)
	}

	if transaction.Type != engine.ExtractedTypeTransfer {
		items[datatable.TRANSACTION_DATA_TABLE_CATEGORY] = transaction.Category
		items[datatable.TRANSACTION_DATA_TABLE_SUB_CATEGORY] = transaction.SubCategory
	}

	if transaction.Memo != "" {
		items[datatable.TRANSACTION_DATA_TABLE_DESCRIPTION] = transaction.Memo
	}

	return items
}

// createNewMicrosoftMoneyTransactionDataTable returns the data table that holds
// the transactions of a Microsoft Money data file
func createNewMicrosoftMoneyTransactionDataTable(allData []engine.ExtractedTransaction) *microsoftMoneyTransactionDataTable {
	return &microsoftMoneyTransactionDataTable{allData: allData}
}
