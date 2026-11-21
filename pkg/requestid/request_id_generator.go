// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package requestid

// RequestIdGenerator is common request generator interface
type RequestIdGenerator interface {
	GenerateRequestId(clientIpAddr string, clientPort uint16) string
	GetCurrentServerUniqId() uint16
	GetCurrentInstanceUniqId() uint16
}
