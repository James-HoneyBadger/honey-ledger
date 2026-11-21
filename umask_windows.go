// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

//go:build windows

package main

// setRestrictiveUmask is a no-op on Windows, which uses ACLs instead of umask.
func setRestrictiveUmask() {}
