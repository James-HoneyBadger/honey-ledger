// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package errs

import "net/http"

// Error codes related to the transaction review
var (
	ErrTransactionReviewDismissalInvalid = NewNormalError(NormalSubcategoryInsightsExplorer, 0, http.StatusBadRequest, "review dismissal is invalid")
)
