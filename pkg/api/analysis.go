// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package api

import (
	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
	"github.com/James-HoneyBadger/honey-ledger/pkg/log"
	"github.com/James-HoneyBadger/honey-ledger/pkg/models"
	"github.com/James-HoneyBadger/honey-ledger/pkg/services"
)

// FinancialAnalysisApi represents the financial analysis api
type FinancialAnalysisApi struct {
	analysis *services.FinancialAnalysisService
}

// Initialize a financial analysis api singleton instance
var (
	FinancialAnalysis = &FinancialAnalysisApi{
		analysis: services.FinancialAnalysis,
	}
)

// AnalyzeHandler returns a local, rule-based analysis of the user's finances.
func (a *FinancialAnalysisApi) AnalyzeHandler(c *core.WebContext) (any, *errs.Error) {
	var request models.FinancialAnalysisRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		log.Warnf(c, "[analysis.AnalyzeHandler] parse request failed, because %s", err.Error())
		return nil, errs.NewIncompleteOrIncorrectSubmissionError(err)
	}

	uid := c.GetCurrentUid()
	response, err := a.analysis.Analyze(c, uid, &request)

	if err != nil {
		log.Errorf(c, "[analysis.AnalyzeHandler] failed to analyse data for user \"uid:%d\", because %s", uid, err.Error())
		return nil, errs.Or(err, errs.ErrOperationFailed)
	}

	return response, nil
}
