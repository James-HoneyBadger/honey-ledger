// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/export"
	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/msmoney"
)

// outputFormat is a file format the extracted data can be written in
type outputFormat struct {
	key         string
	name        string
	extension   string
	description string
	write       func(w io.Writer, data *msmoney.ExtractedData) error
}

// allFormats is built from the export package's registry, so the menu and the
// writers can never diverge.
var allFormats = buildAllFormats()

func buildAllFormats() []outputFormat {
	registry := export.Formats()
	formats := make([]outputFormat, 0, len(registry))

	for _, format := range registry {
		formats = append(formats, outputFormat{
			key:         format.Key,
			name:        format.Name,
			extension:   format.Extension,
			description: format.Description,
			write:       formatWriter(format.Key),
		})
	}

	return formats
}

// formatWriter returns the writer for one registry key.
func formatWriter(key string) func(w io.Writer, data *msmoney.ExtractedData) error {
	return func(w io.Writer, data *msmoney.ExtractedData) error {
		return export.Write(key, w, appName+" "+version, data)
	}
}

func formatKeys() string {
	keys := make([]string, len(allFormats))

	for i, f := range allFormats {
		keys[i] = f.key
	}

	return strings.Join(keys, ", ")
}

func printFormats(w io.Writer) {
	for i, f := range allFormats {
		fmt.Fprintf(w, "  %d) %-17s %-14s %s\n", i+1, f.name, "["+f.key+"]", f.description)
	}
}

// parseFormats turns a comma separated list of format keys into formats. An
// empty list returns no formats, which means the user is asked.
func parseFormats(list string) ([]outputFormat, error) {
	list = strings.TrimSpace(list)

	if list == "" {
		return nil, nil
	}

	var selected []outputFormat
	seen := map[string]bool{}

	for _, key := range strings.Split(list, ",") {
		key = strings.ToLower(strings.TrimSpace(key))

		if key == "all" {
			return allFormats, nil
		}

		found := false

		for _, f := range allFormats {
			if f.key == key {
				found = true

				if !seen[key] {
					seen[key] = true
					selected = append(selected, f)
				}
			}
		}

		if !found {
			return nil, fmt.Errorf("unknown format %q; choose from %s, or all", key, formatKeys())
		}
	}

	return selected, nil
}

// writeFormat writes the data to a new file in dir and returns its path. An
// existing file is never overwritten; a number is added to the name instead.
func writeFormat(f outputFormat, dir string, baseName string, data *msmoney.ExtractedData) (string, error) {
	path := uniquePath(filepath.Join(dir, baseName+f.extension))

	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)

	if err != nil {
		return "", err
	}

	buffered := bufio.NewWriterSize(file, 1<<20)
	err = f.write(buffered, data)

	if err == nil {
		err = buffered.Flush()
	}

	if closeErr := file.Close(); err == nil {
		err = closeErr
	}

	if err != nil {
		os.Remove(path)
		return "", err
	}

	return path, nil
}

func uniquePath(path string) string {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return path
	}

	// The extension can have two parts (".honeyledger.csv"), so the name is
	// split at the first dot of the file name instead of the last.
	dir, name := filepath.Split(path)
	stem, extension := name, ""

	if i := strings.Index(name, "."); i > 0 {
		stem, extension = name[:i], name[i:]
	}

	for n := 1; ; n++ {
		candidate := filepath.Join(dir, fmt.Sprintf("%s (%d)%s", stem, n, extension))

		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

// The format writers live in the export package next to the extraction, so
// the program and the server-side import share exactly one implementation of
// every output format.

func writeCSV(w io.Writer, data *msmoney.ExtractedData) error {
	return export.WriteCSV(w, data)
}

func writeTSV(w io.Writer, data *msmoney.ExtractedData) error {
	return export.WriteTSV(w, data)
}

func writeHoneyLedgerCSV(w io.Writer, data *msmoney.ExtractedData) error {
	return export.WriteHoneyLedgerCSV(w, data)
}

func writeXLSX(w io.Writer, data *msmoney.ExtractedData) error {
	return export.WriteXLSX(w, data)
}

func writeJSON(w io.Writer, data *msmoney.ExtractedData) error {
	return export.WriteJSON(w, appName+" "+version, data)
}

func writeQIF(w io.Writer, data *msmoney.ExtractedData) error {
	return export.WriteQIF(w, data)
}
