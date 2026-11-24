// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msmoney

// Logger is the log surface the extraction code writes to. An application
// binds the logger it uses everywhere else with BindLog, so parser
// diagnostics appear where its own log lines do. When nothing is bound, the
// lines are dropped.
type Logger struct {
	Errorf func(ctx Context, format string, args ...any)
	Infof  func(ctx Context, format string, args ...any)
	Warnf  func(ctx Context, format string, args ...any)
}

var boundLogger Logger

// BindLog sets the logger the extraction code writes to. Call it once at
// startup, before any extraction begins.
func BindLog(logger Logger) {
	boundLogger = logger
}

func errorf(ctx Context, format string, args ...any) {
	if boundLogger.Errorf != nil {
		boundLogger.Errorf(ctx, format, args...)
	}
}

func infof(ctx Context, format string, args ...any) {
	if boundLogger.Infof != nil {
		boundLogger.Infof(ctx, format, args...)
	}
}

func warnf(ctx Context, format string, args ...any) {
	if boundLogger.Warnf != nil {
		boundLogger.Warnf(ctx, format, args...)
	}
}
