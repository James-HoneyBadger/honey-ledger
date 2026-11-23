// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package log

import "io"

// SetQuiet discards all log output. Command line tools that report their own
// results use it so the log lines of the libraries they call do not clutter
// the output.
func SetQuiet() {
	for _, logger := range []interface{ SetOutput(io.Writer) }{bootLogger, cliLogger, defaultLogger, requestLogger, sqlQueryLogger} {
		logger.SetOutput(io.Discard)
	}
}
