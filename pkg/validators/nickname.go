// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package validators

import (
	"github.com/go-playground/validator/v10"

	"github.com/James-HoneyBadger/honey-ledger/pkg/utils"
)

// ValidNickname returns whether the given nick name is valid
func ValidNickname(fl validator.FieldLevel) bool {
	if value, ok := fl.Field().Interface().(string); ok {
		if utils.IsValidNickName(value) {
			return true
		}
	}

	return false
}
