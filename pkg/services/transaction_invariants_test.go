// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package services

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/datastore"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
	"github.com/James-HoneyBadger/honey-ledger/pkg/models"
	"github.com/James-HoneyBadger/honey-ledger/pkg/settings"
	"github.com/James-HoneyBadger/honey-ledger/pkg/utils"
	"github.com/James-HoneyBadger/honey-ledger/pkg/uuid"
)

// transactionInvariantEnvironment is a TransactionService wired to a fresh
// SQLite database with two accounts and the categories the service requires.
type transactionInvariantEnvironment struct {
	ctx           core.Context
	uid           int64
	service       *TransactionService
	accountA      int64
	accountB      int64
	expenseChild  int64
	incomeChild   int64
	transferChild int64
}

func newTransactionInvariantEnvironment(t *testing.T, currencyA string, currencyB string) *transactionInvariantEnvironment {
	t.Helper()

	config := &settings.Config{
		WorkingPath: t.TempDir(),
		DatabaseConfig: &settings.DatabaseConfig{
			DatabaseType:      settings.Sqlite3DbType,
			DatabasePath:      filepath.Join(t.TempDir(), "invariants.db"),
			MaxIdleConnection: 1,
			MaxOpenConnection: 1,
		},
		UuidGeneratorType: settings.InternalUuidGeneratorType,
		UuidServerId:      7,
	}

	previousConfig := settings.Container.GetCurrentConfig()
	settings.SetCurrentConfig(config)
	require.NoError(t, datastore.InitializeDataStore(config))

	t.Cleanup(func() {
		_ = datastore.Container.Close()
		settings.SetCurrentConfig(previousConfig)
	})

	require.NoError(t, datastore.Container.UserDataStore.SyncStructs(new(models.Account), new(models.Transaction), new(models.TransactionCategory), new(models.TransactionTag), new(models.TransactionTagIndex), new(models.TransactionPictureInfo)))
	require.NoError(t, uuid.InitializeUuidGenerator(config))

	ctx := core.NewNullContext()
	uid := uuid.Container.GenerateUuid(uuid.UUID_TYPE_USER)

	accountA := insertInvariantAccount(t, ctx, uid, "Checking", currencyA)
	accountB := insertInvariantAccount(t, ctx, uid, "Savings", currencyB)

	expenseParent := insertInvariantCategory(t, ctx, uid, models.CATEGORY_TYPE_EXPENSE, 0, "Expense")
	incomeParent := insertInvariantCategory(t, ctx, uid, models.CATEGORY_TYPE_INCOME, 0, "Income")
	transferParent := insertInvariantCategory(t, ctx, uid, models.CATEGORY_TYPE_TRANSFER, 0, "Transfer")

	expenseChild := insertInvariantCategory(t, ctx, uid, models.CATEGORY_TYPE_EXPENSE, expenseParent, "Food")
	incomeChild := insertInvariantCategory(t, ctx, uid, models.CATEGORY_TYPE_INCOME, incomeParent, "Salary")
	transferChild := insertInvariantCategory(t, ctx, uid, models.CATEGORY_TYPE_TRANSFER, transferParent, "Internal")

	return &transactionInvariantEnvironment{
		ctx:           ctx,
		uid:           uid,
		service:       NewTransactionService(datastore.Container, uuid.Container),
		accountA:      accountA,
		accountB:      accountB,
		expenseChild:  expenseChild,
		incomeChild:   incomeChild,
		transferChild: transferChild,
	}
}

func insertInvariantAccount(t *testing.T, ctx core.Context, uid int64, name string, currency string) int64 {
	t.Helper()

	account := &models.Account{
		AccountId:       uuid.Container.GenerateUuid(uuid.UUID_TYPE_ACCOUNT),
		Uid:             uid,
		Category:        models.ACCOUNT_CATEGORY_CHECKING_ACCOUNT,
		Type:            models.ACCOUNT_TYPE_SINGLE_ACCOUNT,
		Name:            name,
		Color:           "000000",
		Currency:        currency,
		CreatedUnixTime: time.Now().Unix(),
		UpdatedUnixTime: time.Now().Unix(),
	}

	_, err := datastore.Container.UserDataStore.Choose(uid).NewSession(ctx).Insert(account)
	require.NoError(t, err)

	return account.AccountId
}

func insertInvariantCategory(t *testing.T, ctx core.Context, uid int64, categoryType models.TransactionCategoryType, parentId int64, name string) int64 {
	t.Helper()

	category := &models.TransactionCategory{
		CategoryId:       uuid.Container.GenerateUuid(uuid.UUID_TYPE_CATEGORY),
		Uid:              uid,
		Type:             categoryType,
		ParentCategoryId: parentId,
		Name:             name,
		Color:            "000000",
		CreatedUnixTime:  time.Now().Unix(),
		UpdatedUnixTime:  time.Now().Unix(),
	}

	_, err := datastore.Container.UserDataStore.Choose(uid).NewSession(ctx).Insert(category)
	require.NoError(t, err)

	return category.CategoryId
}

func (e *transactionInvariantEnvironment) createTransfer(amount int64, relatedAmount int64) (*models.Transaction, error) {
	transaction := &models.Transaction{
		Uid:                  e.currentUid(),
		Type:                 models.TRANSACTION_DB_TYPE_TRANSFER_OUT,
		CategoryId:           e.transferChild,
		AccountId:            e.accountA,
		RelatedAccountId:     e.accountB,
		Amount:               amount,
		RelatedAccountAmount: relatedAmount,
		TransactionTime:      utils.GetMinTransactionTimeFromUnixTime(time.Now().Unix()),
	}

	return transaction, e.service.CreateTransaction(e.ctx, transaction, nil, nil)
}

func (e *transactionInvariantEnvironment) createExpense(amount int64) (*models.Transaction, error) {
	transaction := &models.Transaction{
		Uid:             e.currentUid(),
		Type:            models.TRANSACTION_DB_TYPE_EXPENSE,
		CategoryId:      e.expenseChild,
		AccountId:       e.accountA,
		Amount:          amount,
		TransactionTime: utils.GetMinTransactionTimeFromUnixTime(time.Now().Unix()),
	}

	return transaction, e.service.CreateTransaction(e.ctx, transaction, nil, nil)
}

func (e *transactionInvariantEnvironment) createIncome(amount int64) (*models.Transaction, error) {
	transaction := &models.Transaction{
		Uid:             e.currentUid(),
		Type:            models.TRANSACTION_DB_TYPE_INCOME,
		CategoryId:      e.incomeChild,
		AccountId:       e.accountB,
		Amount:          amount,
		TransactionTime: utils.GetMinTransactionTimeFromUnixTime(time.Now().Unix()),
	}

	return transaction, e.service.CreateTransaction(e.ctx, transaction, nil, nil)
}

func (e *transactionInvariantEnvironment) currentUid() int64 {
	return e.uid
}

func (e *transactionInvariantEnvironment) accountBalance(accountId int64) int64 {
	account := &models.Account{}
	has, err := datastore.Container.UserDataStore.Choose(0).NewSession(e.ctx).ID(accountId).Get(account)

	if err != nil || !has {
		return 0
	}

	return account.Balance
}

// TestTransactionInvariants_TransfersConserveMoney verifies that creating,
// modifying and deleting transfers never creates or destroys money and that
// account balances always agree with the stored transactions.
func TestTransactionInvariants_TransfersConserveMoney(t *testing.T) {
	env := newTransactionInvariantEnvironment(t, "USD", "USD")

	// A same-currency transfer with unequal amounts would change the money
	// stock, so it is rejected.
	_, err := env.createTransfer(10000, 9000)
	require.Error(t, err)
	assert.Equal(t, errs.ErrTransactionSourceAndDestinationAmountNotEqual, err)

	expense, err := env.createExpense(2500)
	require.NoError(t, err)
	assert.Equal(t, int64(-2500), env.accountBalance(env.accountA))

	income, err := env.createIncome(1000)
	require.NoError(t, err)
	assert.Equal(t, int64(1000), env.accountBalance(env.accountB))

	transfer, err := env.createTransfer(10000, 10000)
	require.NoError(t, err)
	assert.Equal(t, int64(-12500), env.accountBalance(env.accountA))
	assert.Equal(t, int64(11000), env.accountBalance(env.accountB))
	assert.Equal(t, int64(-1500), env.accountBalance(env.accountA)+env.accountBalance(env.accountB), "expense and income move money, but a transfer must not")

	// The receiving side is stored as its own row with mirrored fields.
	counterpart := &models.Transaction{}
	has, err := datastore.Container.UserDataStore.Choose(0).NewSession(env.ctx).Where("uid=? AND deleted=? AND type=?", env.currentUid(), false, models.TRANSACTION_DB_TYPE_TRANSFER_IN).Get(counterpart)
	require.NoError(t, err)
	require.True(t, has)
	assert.Equal(t, transfer.RelatedId, counterpart.TransactionId)
	assert.Equal(t, transfer.TransactionId, counterpart.RelatedId)
	assert.Equal(t, transfer.AccountId, counterpart.RelatedAccountId)
	assert.Equal(t, transfer.RelatedAccountId, counterpart.AccountId)
	assert.Equal(t, transfer.RelatedAccountAmount, counterpart.Amount)
	assert.Equal(t, transfer.Amount, counterpart.RelatedAccountAmount)

	// Modifying a transfer re-attributes the money on both sides.
	modified := &models.Transaction{
		TransactionId:        transfer.TransactionId,
		Uid:                  env.currentUid(),
		CategoryId:           transfer.CategoryId,
		TransactionTime:      transfer.TransactionTime,
		TimezoneUtcOffset:    transfer.TimezoneUtcOffset,
		AccountId:            transfer.AccountId,
		RelatedAccountId:     transfer.RelatedAccountId,
		Amount:               20000,
		RelatedAccountAmount: 20000,
		Comment:              transfer.Comment,
		GeoLongitude:         transfer.GeoLongitude,
		GeoLatitude:          transfer.GeoLatitude,
	}

	require.NoError(t, env.service.ModifyTransaction(env.ctx, modified, 0, nil, nil, nil, nil))
	assert.Equal(t, int64(-22500), env.accountBalance(env.accountA))
	assert.Equal(t, int64(21000), env.accountBalance(env.accountB))
	assert.Equal(t, int64(-1500), env.accountBalance(env.accountA)+env.accountBalance(env.accountB))

	// Deleting the transfer reverses it exactly.
	require.NoError(t, env.service.DeleteTransaction(env.ctx, env.currentUid(), transfer.TransactionId))
	assert.Equal(t, int64(-2500), env.accountBalance(env.accountA))
	assert.Equal(t, int64(1000), env.accountBalance(env.accountB))

	// The expense and income remain untouched by the transfer lifecycle.
	remaining := make([]*models.Transaction, 0)
	require.NoError(t, datastore.Container.UserDataStore.Choose(0).NewSession(env.ctx).Where("uid=? AND deleted=?", env.currentUid(), false).Find(&remaining))
	require.Len(t, remaining, 2)
	amounts := map[int64]int64{expense.Amount: 0, income.Amount: 0}

	for _, transaction := range remaining {
		amounts[transaction.Amount]++
	}

	assert.Equal(t, int64(1), amounts[expense.Amount])
	assert.Equal(t, int64(1), amounts[income.Amount])
}

// TestTransactionInvariants_CrossCurrencyTransfers verifies that a transfer
// between accounts of different currencies keeps each side's amount in its own
// currency, while a same-currency transfer with unequal amounts is rejected.
func TestTransactionInvariants_CrossCurrencyTransfers(t *testing.T) {
	env := newTransactionInvariantEnvironment(t, "USD", "EUR")

	transfer, err := env.createTransfer(10000, 16000)
	require.NoError(t, err)
	assert.Equal(t, int64(-10000), env.accountBalance(env.accountA), "the source amount stays in the source currency")
	assert.Equal(t, int64(16000), env.accountBalance(env.accountB), "the destination amount stays in the destination currency")

	require.NoError(t, env.service.DeleteTransaction(env.ctx, env.currentUid(), transfer.TransactionId))
	assert.Equal(t, int64(0), env.accountBalance(env.accountA))
	assert.Equal(t, int64(0), env.accountBalance(env.accountB))
}

// TestTransactionInvariants_BalanceModification verifies that a balance
// modification sets the account balance once and that modifying or deleting it
// keeps the balance consistent with the stored amount.
func TestTransactionInvariants_BalanceModification(t *testing.T) {
	env := newTransactionInvariantEnvironment(t, "USD", "USD")

	transaction := &models.Transaction{
		Uid:             env.currentUid(),
		Type:            models.TRANSACTION_DB_TYPE_MODIFY_BALANCE,
		AccountId:       env.accountA,
		Amount:          5000,
		TransactionTime: utils.GetMinTransactionTimeFromUnixTime(time.Now().Unix()),
	}

	require.NoError(t, env.service.CreateTransaction(env.ctx, transaction, nil, nil))
	assert.Equal(t, int64(5000), env.accountBalance(env.accountA))

	modified := &models.Transaction{
		TransactionId:        transaction.TransactionId,
		Uid:                  env.currentUid(),
		AccountId:            env.accountA,
		Amount:               7000,
		TransactionTime:      transaction.TransactionTime,
		TimezoneUtcOffset:    transaction.TimezoneUtcOffset,
		RelatedAccountId:     transaction.RelatedAccountId,
		RelatedAccountAmount: transaction.RelatedAccountAmount,
		Comment:              transaction.Comment,
		GeoLongitude:         transaction.GeoLongitude,
		GeoLatitude:          transaction.GeoLatitude,
	}

	require.NoError(t, env.service.ModifyTransaction(env.ctx, modified, 0, nil, nil, nil, nil))
	assert.Equal(t, int64(7000), env.accountBalance(env.accountA))

	require.NoError(t, env.service.DeleteTransaction(env.ctx, env.currentUid(), transaction.TransactionId))
	assert.Equal(t, int64(0), env.accountBalance(env.accountA))
}
