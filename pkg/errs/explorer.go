// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package errs

import "net/http"

// Error codes related to insights explorers
var (
	ErrInsightsExplorerIdInvalid   = NewNormalError(NormalSubcategoryInsightsExplorer, 0, http.StatusBadRequest, "explorer id is invalid")
	ErrInsightsExplorerNotFound    = NewNormalError(NormalSubcategoryInsightsExplorer, 1, http.StatusBadRequest, "explorer not found")
	ErrInsightsExplorerDataInvalid = NewNormalError(NormalSubcategoryInsightsExplorer, 2, http.StatusBadRequest, "explorer data is invalid")
)
