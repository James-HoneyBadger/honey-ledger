// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package services

import (
	"sort"
	"time"

	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/datastore"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
	"github.com/James-HoneyBadger/honey-ledger/pkg/models"
	"github.com/James-HoneyBadger/honey-ledger/pkg/utils"
)

const (
	financialAnalysisTopCategoryCount          = 8
	financialAnalysisLargestExpenseCount       = 10
	financialAnalysisRecurringMinOccurrences   = 3
	financialAnalysisRecurringMinGapDays       = 20
	financialAnalysisRecurringMaxGapDays       = 45
	financialAnalysisUnusualMinAccountExpenses = 5
	financialAnalysisUnusualAmountMultiplier   = 3
)

// FinancialAnalysisService produces a local, rule-based analysis of the user's
// finances. It replaces the large language model assistant with deterministic
// heuristics, so nothing is sent off the machine.
type FinancialAnalysisService struct {
	ServiceUsingDB

	// sharedExpenses supplies the outstanding balances with other people.
	sharedExpenses func(c core.Context, uid int64) ([]*models.SharedExpenseSummaryItem, error)
}

// FinancialAnalysis is the application-wide financial analysis service.
var FinancialAnalysis = &FinancialAnalysisService{
	ServiceUsingDB: ServiceUsingDB{
		container: datastore.Container,
	},
	sharedExpenses: func(c core.Context, uid int64) ([]*models.SharedExpenseSummaryItem, error) {
		return TransactionSplits.GetSharedExpenseSummary(c, uid)
	},
}

// Analyze computes a report over the user's transactions for the given period.
func (s *FinancialAnalysisService) Analyze(c core.Context, uid int64, request *models.FinancialAnalysisRequest) (*models.FinancialAnalysisResponse, error) {
	if uid <= 0 {
		return nil, errs.ErrUserIdInvalid
	}

	if request == nil {
		request = &models.FinancialAnalysisRequest{}
	}

	months := request.Months

	if months <= 0 {
		months = models.FinancialAnalysisDefaultMonths
	} else if months > models.FinancialAnalysisMaxMonths {
		months = models.FinancialAnalysisMaxMonths
	}

	now := time.Now()
	startTime := now.AddDate(0, -months, 0)

	session := s.UserDataDB(uid).NewSession(c)
	defer session.Close()

	startTransactionTime := utils.GetMinTransactionTimeFromUnixTime(startTime.Unix())
	endTransactionTime := utils.GetMinTransactionTimeFromUnixTime(now.Unix()) + 999

	var transactions []*models.Transaction

	if err := session.
		Where("uid=? AND deleted=? AND transaction_time>=? AND transaction_time<=?", uid, false, startTransactionTime, endTransactionTime).
		Cols("transaction_id", "type", "amount", "category_id", "account_id", "transaction_time", "comment").
		Find(&transactions); err != nil {
		return nil, err
	}

	var accounts []*models.Account

	if err := session.Where("uid=? AND deleted=?", uid, false).Cols("account_id", "name", "category", "currency", "balance").Find(&accounts); err != nil {
		return nil, err
	}

	var categories []*models.TransactionCategory

	if err := session.Where("uid=? AND deleted=?", uid, false).Cols("category_id", "name").Find(&categories); err != nil {
		return nil, err
	}

	var sharedItems []*models.SharedExpenseSummaryItem

	if s.sharedExpenses != nil {
		items, err := s.sharedExpenses(c, uid)

		if err == nil {
			sharedItems = items
		}
	}

	return assembleFinancialAnalysis(financialAnalysisDataSource{
		startTime:    startTime,
		endTime:      now,
		transactions: transactions,
		accounts:     accounts,
		categories:   categories,
		sharedItems:  sharedItems,
	}), nil
}

// financialAnalysisDataSource is the raw data the report is assembled from.
// Keeping the assembly separate from the queries makes the heuristics testable.
type financialAnalysisDataSource struct {
	startTime    time.Time
	endTime      time.Time
	transactions []*models.Transaction
	accounts     []*models.Account
	categories   []*models.TransactionCategory
	sharedItems  []*models.SharedExpenseSummaryItem
}

func assembleFinancialAnalysis(source financialAnalysisDataSource) *models.FinancialAnalysisResponse {
	accountCurrency := make(map[int64]string, len(source.accounts))
	accountName := make(map[int64]string, len(source.accounts))
	accountCategory := make(map[int64]models.AccountCategory, len(source.accounts))

	for _, account := range source.accounts {
		accountCurrency[account.AccountId] = account.Currency
		accountName[account.AccountId] = account.Name
		accountCategory[account.AccountId] = account.Category
	}

	categoryNames := make(map[int64]string, len(source.categories))

	for _, category := range source.categories {
		categoryNames[category.CategoryId] = category.Name
	}

	response := &models.FinancialAnalysisResponse{
		StartTime:         source.startTime.Unix(),
		EndTime:           source.endTime.Unix(),
		Totals:            []*models.FinancialAnalysisCurrencyTotal{},
		TopCategories:     []*models.FinancialAnalysisCategoryTotal{},
		LargestExpenses:   []*models.FinancialAnalysisTransaction{},
		RecurringPayments: []*models.FinancialAnalysisRecurringPayment{},
		UnusualExpenses:   []*models.FinancialAnalysisTransaction{},
		Accounts:          []*models.FinancialAnalysisAccountSummary{},
		SharedExpenses:    []*models.FinancialAnalysisSharedExpense{},
	}

	totalsByCurrency := make(map[string]*models.FinancialAnalysisCurrencyTotal)
	categoryTotals := make(map[string]*models.FinancialAnalysisCategoryTotal)

	expenseTransactions := make([]*models.Transaction, 0)
	expensesByAccount := make(map[int64][]int64)

	for _, transaction := range source.transactions {
		currency := accountCurrency[transaction.AccountId]

		if currency == "" {
			continue
		}

		if transaction.Type != models.TRANSACTION_DB_TYPE_INCOME && transaction.Type != models.TRANSACTION_DB_TYPE_EXPENSE {
			continue
		}

		total, exists := totalsByCurrency[currency]

		if !exists {
			total = &models.FinancialAnalysisCurrencyTotal{Currency: currency}
			totalsByCurrency[currency] = total
		}

		total.Count++
		response.TransactionCount++

		if transaction.Type == models.TRANSACTION_DB_TYPE_INCOME {
			total.Income += transaction.Amount
			continue
		}

		total.Expense += transaction.Amount
		expenseTransactions = append(expenseTransactions, transaction)
		expensesByAccount[transaction.AccountId] = append(expensesByAccount[transaction.AccountId], transaction.Amount)

		categoryName := categoryNames[transaction.CategoryId]

		if categoryName == "" {
			continue
		}

		key := currency + "\x00" + categoryName
		categoryTotal, exists := categoryTotals[key]

		if !exists {
			categoryTotal = &models.FinancialAnalysisCategoryTotal{Category: categoryName, Currency: currency}
			categoryTotals[key] = categoryTotal
		}

		categoryTotal.Amount += transaction.Amount
		categoryTotal.Count++
	}

	for _, total := range totalsByCurrency {
		total.Net = total.Income - total.Expense
		response.Totals = append(response.Totals, total)
	}

	sort.SliceStable(response.Totals, func(i, j int) bool {
		return response.Totals[i].Currency < response.Totals[j].Currency
	})

	for _, categoryTotal := range categoryTotals {
		response.TopCategories = append(response.TopCategories, categoryTotal)
	}

	sort.SliceStable(response.TopCategories, func(i, j int) bool {
		if response.TopCategories[i].Amount != response.TopCategories[j].Amount {
			return response.TopCategories[i].Amount > response.TopCategories[j].Amount
		}

		return response.TopCategories[i].Category < response.TopCategories[j].Category
	})

	if len(response.TopCategories) > financialAnalysisTopCategoryCount {
		response.TopCategories = response.TopCategories[:financialAnalysisTopCategoryCount]
	}

	response.LargestExpenses = largestExpenses(expenseTransactions, accountName, accountCurrency, categoryNames)
	response.RecurringPayments = recurringPayments(expenseTransactions, accountName, accountCurrency, categoryNames)
	response.UnusualExpenses = unusualExpenses(expenseTransactions, expensesByAccount, accountName, accountCurrency, categoryNames)

	for _, account := range source.accounts {
		response.Accounts = append(response.Accounts, &models.FinancialAnalysisAccountSummary{
			Name:     account.Name,
			Category: financialAnalysisAccountCategoryName(account.Category),
			Currency: account.Currency,
			Balance:  account.Balance,
		})
	}

	sort.SliceStable(response.Accounts, func(i, j int) bool {
		return response.Accounts[i].Name < response.Accounts[j].Name
	})

	for _, item := range source.sharedItems {
		response.SharedExpenses = append(response.SharedExpenses, &models.FinancialAnalysisSharedExpense{
			PersonName: item.PersonName,
			Currency:   item.Currency,
			Total:      item.Total,
			OwedToMe:   item.OwedToMe,
			IOwe:       item.IOwe,
		})
	}

	return response
}

func largestExpenses(transactions []*models.Transaction, accountName map[int64]string, accountCurrency map[int64]string, categoryNames map[int64]string) []*models.FinancialAnalysisTransaction {
	sorted := make([]*models.Transaction, len(transactions))
	copy(sorted, transactions)

	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Amount > sorted[j].Amount
	})

	count := financialAnalysisLargestExpenseCount

	if len(sorted) < count {
		count = len(sorted)
	}

	result := make([]*models.FinancialAnalysisTransaction, 0, count)

	for _, transaction := range sorted[:count] {
		result = append(result, financialAnalysisTransactionView(transaction, accountName, accountCurrency, categoryNames))
	}

	return result
}

func recurringPayments(transactions []*models.Transaction, accountName map[int64]string, accountCurrency map[int64]string, categoryNames map[int64]string) []*models.FinancialAnalysisRecurringPayment {
	type groupKey struct {
		accountId  int64
		categoryId int64
		amount     int64
	}

	groups := make(map[groupKey][]*models.Transaction)

	for _, transaction := range transactions {
		if transaction.Amount <= 0 {
			continue
		}

		key := groupKey{accountId: transaction.AccountId, categoryId: transaction.CategoryId, amount: transaction.Amount}
		groups[key] = append(groups[key], transaction)
	}

	result := []*models.FinancialAnalysisRecurringPayment{}

	for _, group := range groups {
		if len(group) < financialAnalysisRecurringMinOccurrences {
			continue
		}

		sort.SliceStable(group, func(i, j int) bool {
			return group[i].TransactionTime < group[j].TransactionTime
		})

		if !hasMonthlyGaps(group) {
			continue
		}

		last := group[len(group)-1]

		result = append(result, &models.FinancialAnalysisRecurringPayment{
			Account:  accountName[last.AccountId],
			Category: categoryNames[last.CategoryId],
			Currency: accountCurrency[last.AccountId],
			Amount:   last.Amount,
			Count:    int64(len(group)),
			LastTime: last.TransactionTime,
		})
	}

	sort.SliceStable(result, func(i, j int) bool {
		if result[i].Amount != result[j].Amount {
			return result[i].Amount > result[j].Amount
		}

		return result[i].Category < result[j].Category
	})

	return result
}

func hasMonthlyGaps(transactions []*models.Transaction) bool {
	gaps := make([]int64, 0, len(transactions)-1)

	for i := 1; i < len(transactions); i++ {
		gapDays := (transactions[i].TransactionTime - transactions[i-1].TransactionTime) / (24 * 60 * 60 * 1000)
		gaps = append(gaps, gapDays)
	}

	sort.SliceStable(gaps, func(i, j int) bool {
		return gaps[i] < gaps[j]
	})

	medianGap := gaps[len(gaps)/2]

	return medianGap >= financialAnalysisRecurringMinGapDays && medianGap <= financialAnalysisRecurringMaxGapDays
}

func unusualExpenses(transactions []*models.Transaction, expensesByAccount map[int64][]int64, accountName map[int64]string, accountCurrency map[int64]string, categoryNames map[int64]string) []*models.FinancialAnalysisTransaction {
	accountMedian := make(map[int64]int64, len(expensesByAccount))

	for accountId, amounts := range expensesByAccount {
		if len(amounts) < financialAnalysisUnusualMinAccountExpenses {
			continue
		}

		accountMedian[accountId] = medianInt64(amounts)
	}

	result := []*models.FinancialAnalysisTransaction{}

	for _, transaction := range transactions {
		medianAmount, exists := accountMedian[transaction.AccountId]

		if !exists || medianAmount <= 0 || transaction.Amount <= medianAmount*financialAnalysisUnusualAmountMultiplier {
			continue
		}

		result = append(result, financialAnalysisTransactionView(transaction, accountName, accountCurrency, categoryNames))
	}

	sort.SliceStable(result, func(i, j int) bool {
		return result[i].Amount > result[j].Amount
	})

	return result
}

func financialAnalysisTransactionView(transaction *models.Transaction, accountName map[int64]string, accountCurrency map[int64]string, categoryNames map[int64]string) *models.FinancialAnalysisTransaction {
	return &models.FinancialAnalysisTransaction{
		TransactionId: transaction.TransactionId,
		Time:          transaction.TransactionTime,
		Account:       accountName[transaction.AccountId],
		Category:      categoryNames[transaction.CategoryId],
		Currency:      accountCurrency[transaction.AccountId],
		Amount:        transaction.Amount,
		Comment:       transaction.Comment,
	}
}

func medianInt64(values []int64) int64 {
	sorted := make([]int64, len(values))
	copy(sorted, values)

	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i] < sorted[j]
	})

	return sorted[len(sorted)/2]
}

func financialAnalysisAccountCategoryName(category models.AccountCategory) string {
	switch category {
	case models.ACCOUNT_CATEGORY_CASH:
		return "cash"
	case models.ACCOUNT_CATEGORY_CHECKING_ACCOUNT:
		return "checking account"
	case models.ACCOUNT_CATEGORY_CREDIT_CARD:
		return "credit card"
	case models.ACCOUNT_CATEGORY_VIRTUAL:
		return "virtual"
	case models.ACCOUNT_CATEGORY_DEBT:
		return "debt"
	case models.ACCOUNT_CATEGORY_RECEIVABLES:
		return "receivables"
	case models.ACCOUNT_CATEGORY_INVESTMENT:
		return "investment"
	case models.ACCOUNT_CATEGORY_SAVINGS_ACCOUNT:
		return "savings account"
	default:
		return "account"
	}
}
