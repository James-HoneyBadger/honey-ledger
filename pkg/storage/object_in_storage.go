// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package storage

import (
	"io"
)

// ObjectInStorage represents the object instance in the storage
type ObjectInStorage interface {
	io.ReadCloser
	io.Seeker
}
