// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package errs

import "net/http"

// Error codes related to model context protocol server
var (
	ErrMCPServerNotEnabled = NewNormalError(NormalSubcategoryModelContextProtocol, 0, http.StatusBadRequest, "mcp server is not enabled")
)
