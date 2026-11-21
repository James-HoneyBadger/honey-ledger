// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package validators

import (
	"github.com/go-playground/validator/v10"

	"github.com/James-HoneyBadger/honey-ledger/pkg/utils"
)

// ValidEmail returns whether the given email is valid
func ValidEmail(fl validator.FieldLevel) bool {
	if value, ok := fl.Field().Interface().(string); ok {
		if utils.IsValidEmail(value) {
			return true
		}
	}

	return false
}
