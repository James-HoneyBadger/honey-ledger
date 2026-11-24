// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

// Package msmoney imports Microsoft Money data files (.mny) and backups (.mbf)
// into Honey Ledger. The extraction itself lives in the shared money-engine
// module; this package adapts it to the ledger: it keeps the importer the
// converters registry knows, maps the engine's errors onto the ledger's error
// constants, and routes the engine's log lines through the ledger's logger.
package msmoney

import (
	"errors"

	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
	"github.com/James-HoneyBadger/honey-ledger/pkg/log"
	engine "github.com/James-HoneyBadger/money-engine/msmoney"
)

// Re-exported extraction API. The aliases keep the existing public surface of
// this package, so the command line tool and the converters registry consume
// the same types as before, now backed by the shared implementation.
type (
	ExtractedData        = engine.ExtractedData
	ExtractedAccount     = engine.ExtractedAccount
	ExtractedTransaction = engine.ExtractedTransaction
)

const (
	ExtractedTypeIncome   = engine.ExtractedTypeIncome
	ExtractedTypeExpense  = engine.ExtractedTypeExpense
	ExtractedTypeTransfer = engine.ExtractedTypeTransfer
)

// IsMicrosoftMoneyDataFile returns whether the data carries a Microsoft Money (MSISAM) header
func IsMicrosoftMoneyDataFile(data []byte) bool {
	return engine.IsMicrosoftMoneyDataFile(data)
}

// IsMicrosoftMoneyBackupFile returns whether the data is a Microsoft Cabinet archive
func IsMicrosoftMoneyBackupFile(data []byte) bool {
	return engine.IsMicrosoftMoneyBackupFile(data)
}

// ExtractMicrosoftMoneyData reads a Microsoft Money data file (.mny) or backup
// (.mbf) and returns its accounts and transactions without needing a ledger.
// Errors are reported as the ledger's error constants, so callers can match on
// them exactly as before the extraction moved into money-engine.
func ExtractMicrosoftMoneyData(ctx core.Context, data []byte) (*ExtractedData, error) {
	extracted, err := engine.ExtractMicrosoftMoneyData(ctx, data)

	if err != nil {
		return nil, mappedExtractionError(err)
	}

	return extracted, nil
}

// mappedExtractionError maps an error of the shared engine onto the ledger's
// error constants. The engine's messages match the constants' messages, so the
// error text the user and the logs see is unchanged.
func mappedExtractionError(err error) *errs.Error {
	switch {
	case errors.Is(err, engine.ErrFileTypeNotSupported):
		return errs.ErrImportFileTypeNotSupported
	case errors.Is(err, engine.ErrDataFileEncrypted):
		return errs.ErrImportMicrosoftMoneyDataFileEncrypted
	case errors.Is(err, engine.ErrBackupFileNotReadable):
		return errs.ErrImportMicrosoftMoneyBackupFileNotReadable
	case errors.Is(err, engine.ErrBackupFileTooLarge):
		return errs.ErrImportMicrosoftMoneyBackupFileTooLarge
	default:
		return errs.ErrImportMicrosoftMoneyDataFileNotReadable
	}
}

// honeyLedgerContext narrows the engine's context back to the ledger's context
// type, so the ledger's logger can read it. The applications always pass their
// own context, so the assertion holds; anything else logs without a context.
func honeyLedgerContext(ctx engine.Context) core.Context {
	if ctx == nil {
		return nil
	}

	if ledgerContext, ok := ctx.(core.Context); ok {
		return ledgerContext
	}

	return nil
}

func init() {
	engine.BindLog(engine.Logger{
		Errorf: func(ctx engine.Context, format string, args ...any) {
			log.Errorf(honeyLedgerContext(ctx), format, args...)
		},
		Infof: func(ctx engine.Context, format string, args ...any) {
			log.Infof(honeyLedgerContext(ctx), format, args...)
		},
		Warnf: func(ctx engine.Context, format string, args ...any) {
			log.Warnf(honeyLedgerContext(ctx), format, args...)
		},
	})
}
