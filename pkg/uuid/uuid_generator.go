// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package uuid

// UuidGenerator is common uuid generator interface
type UuidGenerator interface {
	GenerateUuid(uuidType UuidType) int64
	GenerateUuids(uuidType UuidType, count uint16) []int64
}
