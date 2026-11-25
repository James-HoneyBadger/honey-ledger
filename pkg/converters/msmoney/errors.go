// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

import "errors"

// Errors the extraction reports. The messages match the error messages Honey
// Ledger reports for the same failures, so an application maps them onto its
// own error constants with errors.Is and keeps its messages identical.
var (
	// ErrFileTypeNotSupported reports input that carries no Microsoft Money header
	ErrFileTypeNotSupported = errors.New("import file type not supported")

	// ErrDataFileNotReadable reports a database that cannot be read
	ErrDataFileNotReadable = errors.New("microsoft money data file cannot be read directly")

	// ErrDataFileEncrypted reports a database whose key block says it is password protected
	ErrDataFileEncrypted = errors.New("microsoft money data file is password protected")

	// ErrDataFileUnsupportedVersion reports a database whose engine version is
	// not the Jet 4 layout this package reads, such as Money 2000 and earlier.
	ErrDataFileUnsupportedVersion = errors.New("microsoft money data file uses an unsupported version")

	// ErrBackupFileNotReadable reports a backup that cannot be read
	ErrBackupFileNotReadable = errors.New("microsoft money backup file cannot be read")

	// ErrBackupFileTooLarge reports a backup that would expand past the supported size
	ErrBackupFileTooLarge = errors.New("microsoft money backup file is too large")
)
