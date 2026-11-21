// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package errs

import "net/http"

// Error codes related to cron jobs
var (
	ErrCronJobNameIsEmpty           = NewSystemError(SystemSubcategoryCron, 0, http.StatusInternalServerError, "cron job name is empty")
	ErrCronJobNotExistsOrNotEnabled = NewSystemError(SystemSubcategoryCron, 1, http.StatusInternalServerError, "cron job not exists or not enabled")
)
