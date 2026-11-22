// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package cmd

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/James-HoneyBadger/honey-ledger/pkg/settings"
)

func TestGetConfigWithoutSensitiveData_RedactsCredentials(t *testing.T) {
	config := &settings.Config{
		DatabaseConfig: &settings.DatabaseConfig{
			DatabasePassword: "db-password",
		},
		SMTPConfig: &settings.SMTPConfig{
			SMTPPasswd: "smtp-password",
		},
		MinIOConfig: &settings.MinIOConfig{
			SecretAccessKey: "minio-secret",
		},
		WebDAVConfig: &settings.WebDAVConfig{
			Password: "webdav-password",
		},
		SecretKey:             "secret-key",
		OAuth2ClientSecret:    "oauth2-secret",
		AmapApplicationSecret: "amap-secret",
		AmapApplicationKey:    "amap-key",
		TomTomMapAPIKey:       "tomtom-key",
		TianDiTuAPIKey:        "tianditu-key",
		GoogleMapAPIKey:       "google-map-key",
		BaiduMapAK:            "baidu-key",
	}

	sanitized := getConfigWithoutSensitiveData(config)

	assert.NotNil(t, sanitized)
	assert.Equal(t, "****", sanitized.DatabaseConfig.DatabasePassword)
	assert.Equal(t, "****", sanitized.SMTPConfig.SMTPPasswd)
	assert.Equal(t, "****", sanitized.MinIOConfig.SecretAccessKey)
	assert.Equal(t, "****", sanitized.WebDAVConfig.Password)
	assert.Equal(t, "****", sanitized.SecretKey)
	assert.Equal(t, "****", sanitized.OAuth2ClientSecret)
	assert.Equal(t, "****", sanitized.AmapApplicationSecret)
	assert.Equal(t, "****", sanitized.AmapApplicationKey)
	assert.Equal(t, "****", sanitized.TomTomMapAPIKey)
	assert.Equal(t, "****", sanitized.TianDiTuAPIKey)
	assert.Equal(t, "****", sanitized.GoogleMapAPIKey)
	assert.Equal(t, "****", sanitized.BaiduMapAK)

	// The caller's configuration must not be modified.
	assert.Equal(t, "secret-key", config.SecretKey)
}

func TestGetConfigWithoutSensitiveData_NilConfig(t *testing.T) {
	assert.Nil(t, getConfigWithoutSensitiveData(nil))
}
