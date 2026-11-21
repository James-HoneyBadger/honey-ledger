// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package validators

import (
	"github.com/go-playground/validator/v10"

	"github.com/James-HoneyBadger/honey-ledger/pkg/utils"
)

// ValidHexRGBColor returns whether the given hex reb color is valid
func ValidHexRGBColor(fl validator.FieldLevel) bool {
	if value, ok := fl.Field().Interface().(string); ok {
		if utils.IsValidHexRGBColor(value) {
			return true
		}
	}

	return false
}
