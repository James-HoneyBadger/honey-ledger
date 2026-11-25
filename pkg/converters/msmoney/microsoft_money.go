// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

// microsoft_money.go adapts the in-repo extraction packages to the ledger: it
// maps the extraction's errors onto the ledger's error constants, so callers
// match them exactly as before the extraction became self-contained, and it
// routes the extraction's log lines through the ledger's logger.

package msmoney

import (
	"errors"

	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
	"github.com/James-HoneyBadger/honey-ledger/pkg/log"
)

// mappedExtractionError maps an error of the extraction onto the ledger's
// error constants. The extraction's messages match the constants' messages, so
// the error text the user and the logs see is unchanged.
func mappedExtractionError(err error) *errs.Error {
	switch {
	case errors.Is(err, ErrFileTypeNotSupported):
		return errs.ErrImportFileTypeNotSupported
	case errors.Is(err, ErrDataFileEncrypted):
		return errs.ErrImportMicrosoftMoneyDataFileEncrypted
	case errors.Is(err, ErrDataFileUnsupportedVersion):
		// The ledger has no dedicated constant for an unsupported engine
		// version, so it reports the file as not readable, with the reason in
		// the log line.
		return errs.ErrImportMicrosoftMoneyDataFileNotReadable
	case errors.Is(err, ErrBackupFileNotReadable):
		return errs.ErrImportMicrosoftMoneyBackupFileNotReadable
	case errors.Is(err, ErrBackupFileTooLarge):
		return errs.ErrImportMicrosoftMoneyBackupFileTooLarge
	default:
		return errs.ErrImportMicrosoftMoneyDataFileNotReadable
	}
}

// honeyLedgerContext narrows the extraction's context back to the ledger's
// context type, so the ledger's logger can read it. The ledger always passes
// its own context, so the assertion holds; anything else logs without a
// context.
func honeyLedgerContext(ctx Context) core.Context {
	if ctx == nil {
		return nil
	}

	if ledgerContext, ok := ctx.(core.Context); ok {
		return ledgerContext
	}

	return nil
}

func init() {
	BindLog(Logger{
		Errorf: func(ctx Context, format string, args ...any) {
			log.Errorf(honeyLedgerContext(ctx), format, args...)
		},
		Infof: func(ctx Context, format string, args ...any) {
			log.Infof(honeyLedgerContext(ctx), format, args...)
		},
		Warnf: func(ctx Context, format string, args ...any) {
			log.Warnf(honeyLedgerContext(ctx), format, args...)
		},
	})
}
