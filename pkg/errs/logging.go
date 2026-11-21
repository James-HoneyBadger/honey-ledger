// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package errs

import (
	"net/http"
)

// Error codes related to logging
var (
	ErrLoggingError = NewSystemError(SystemSubcategoryLogging, 0, http.StatusInternalServerError, "logging error")
)
