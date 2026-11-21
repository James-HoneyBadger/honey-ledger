// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package errs

import "net/http"

// Error codes related to transaction tag groups
var (
	ErrTransactionTagGroupIdInvalid            = NewNormalError(NormalSubcategoryTagGroup, 0, http.StatusBadRequest, "transaction tag group id is invalid")
	ErrTransactionTagGroupNotFound             = NewNormalError(NormalSubcategoryTagGroup, 1, http.StatusBadRequest, "transaction tag group not found")
	ErrTransactionTagGroupInUseCannotBeDeleted = NewNormalError(NormalSubcategoryTagGroup, 2, http.StatusBadRequest, "transaction tag group is in use and cannot be deleted")
)
