// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import (
	"bytes"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/msmoney/msisam"
)

// microsoftMoneyDatabaseSignature is the engine name a Microsoft Money data file
// carries at byte offset 4. Money keeps its tables in a Jet database, but names
// the engine MSISAM, which is why no Jet or Access driver opens the file.
var microsoftMoneyDatabaseSignature = []byte(msisam.Magic)

// IsMicrosoftMoneyDataFile returns whether the data carries a Microsoft Money (MSISAM) header
func IsMicrosoftMoneyDataFile(data []byte) bool {
	return len(data) >= 4+len(microsoftMoneyDatabaseSignature) && bytes.Equal(data[4:4+len(microsoftMoneyDatabaseSignature)], microsoftMoneyDatabaseSignature)
}
