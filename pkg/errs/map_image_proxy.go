// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package errs

import "net/http"

// Error codes related to map image proxy
var (
	ErrMapProviderNotCurrent      = NewNormalError(NormalSubcategoryMapProxy, 0, http.StatusBadRequest, "specified map provider is not set")
	ErrImageExtensionNotSupported = NewNormalError(NormalSubcategoryMapProxy, 0, http.StatusNotFound, "specified image extension is not supported")
)
