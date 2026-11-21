// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/James-HoneyBadger/honey-ledger/pkg/settings"
)

func TestClone(t *testing.T) {
	expectedObject := &settings.Config{
		Mode: settings.MODE_PRODUCTION,
		DatabaseConfig: &settings.DatabaseConfig{
			DatabaseType:          settings.MySqlDbType,
			DatabaseHost:          "localhost",
			ConnectionMaxLifeTime: 60,
		},
		LogModes: []string{"console", "file"},
	}
	actualObject := &settings.Config{}
	err := Clone(expectedObject, actualObject)

	assert.Equal(t, nil, err)
	assert.EqualValues(t, expectedObject, actualObject)
}
