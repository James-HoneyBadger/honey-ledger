// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package models

// FinancialAnalysisDefaultMonths is the period analysed when a request does not
// specify one.
const FinancialAnalysisDefaultMonths = 12

// FinancialAnalysisMaxMonths is the longest period an analysis may cover.
const FinancialAnalysisMaxMonths = 60

// FinancialAnalysisRequest represents all parameters of a financial analysis.
type FinancialAnalysisRequest struct {
	// Months bounds the period the analysis covers; 0 means the default.
	Months int `json:"months" binding:"min=0,max=60"`
}

// FinancialAnalysisResponse represents a local, rule-based analysis of the
// user's finances. No data leaves the machine to produce it.
type FinancialAnalysisResponse struct {
	StartTime         int64                                `json:"startTime"`
	EndTime           int64                                `json:"endTime"`
	TransactionCount  int64                                `json:"transactionCount"`
	Totals            []*FinancialAnalysisCurrencyTotal    `json:"totals"`
	TopCategories     []*FinancialAnalysisCategoryTotal    `json:"topCategories"`
	LargestExpenses   []*FinancialAnalysisTransaction      `json:"largestExpenses"`
	RecurringPayments []*FinancialAnalysisRecurringPayment `json:"recurringPayments"`
	UnusualExpenses   []*FinancialAnalysisTransaction      `json:"unusualExpenses"`
	Accounts          []*FinancialAnalysisAccountSummary   `json:"accounts"`
	SharedExpenses    []*FinancialAnalysisSharedExpense    `json:"sharedExpenses"`
}

// FinancialAnalysisCurrencyTotal represents income and expense totals for one
// currency, in the smallest currency unit.
type FinancialAnalysisCurrencyTotal struct {
	Currency string `json:"currency"`
	Income   int64  `json:"income"`
	Expense  int64  `json:"expense"`
	Net      int64  `json:"net"`
	Count    int64  `json:"count"`
}

// FinancialAnalysisCategoryTotal represents the expense total of one category.
type FinancialAnalysisCategoryTotal struct {
	Category string `json:"category"`
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
	Count    int64  `json:"count"`
}

// FinancialAnalysisTransaction is one transaction referenced by a finding.
type FinancialAnalysisTransaction struct {
	TransactionId int64  `json:"transactionId,string"`
	Time          int64  `json:"time"`
	Account       string `json:"account"`
	Category      string `json:"category"`
	Currency      string `json:"currency"`
	Amount        int64  `json:"amount"`
	Comment       string `json:"comment"`
}

// FinancialAnalysisRecurringPayment is a detected subscription-like expense.
type FinancialAnalysisRecurringPayment struct {
	Account  string `json:"account"`
	Category string `json:"category"`
	Currency string `json:"currency"`
	Amount   int64  `json:"amount"`
	Count    int64  `json:"count"`
	LastTime int64  `json:"lastTime"`
}

// FinancialAnalysisAccountSummary represents one account balance.
type FinancialAnalysisAccountSummary struct {
	Name     string `json:"name"`
	Category string `json:"category"`
	Currency string `json:"currency"`
	Balance  int64  `json:"balance"`
}

// FinancialAnalysisSharedExpense represents the outstanding balance with one
// person.
type FinancialAnalysisSharedExpense struct {
	PersonName string `json:"personName"`
	Currency   string `json:"currency"`
	Total      int64  `json:"total"`
	OwedToMe   int64  `json:"owedToMe"`
	IOwe       int64  `json:"iOwe"`
}
