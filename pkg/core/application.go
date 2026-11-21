// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package core

import "fmt"

// ApplicationName represents the application name
const ApplicationName = "Honey Ledger"

// Version, CommitHash and BuildTime are set at build
var (
	Version    string
	CommitHash string
	BuildTime  string
)

func GetOutgoingUserAgent() string {
	if Version == "" {
		return ApplicationName
	}

	return fmt.Sprintf("%s/%s", ApplicationName, Version)
}
