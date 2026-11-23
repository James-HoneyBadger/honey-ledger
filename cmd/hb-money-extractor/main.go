// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

// HB Money Extractor reads a Microsoft Money data file (.mny) or backup (.mbf)
// and writes its accounts and transactions in the formats the user selects. It
// is a standalone program: it needs no Honey Ledger server, database or login.
package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/James-HoneyBadger/honey-ledger/pkg/converters/msmoney"
	"github.com/James-HoneyBadger/honey-ledger/pkg/core"
	"github.com/James-HoneyBadger/honey-ledger/pkg/errs"
	"github.com/James-HoneyBadger/honey-ledger/pkg/log"
)

const (
	appName    = "HB Money Extractor"
	appVersion = "1.0.0"
)

func main() {
	log.SetQuiet()
	os.Exit(run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("hb-money-extractor", flag.ContinueOnError)
	flags.SetOutput(stderr)

	formatList := flags.String("f", "", "output format(s), comma separated: "+formatKeys()+", or all")
	outputDir := flags.String("o", "", "output folder (default: next to the input file)")
	listFormats := flags.Bool("list", false, "list the supported output formats and exit")
	showVersion := flags.Bool("version", false, "print the version and exit")

	flags.Usage = func() {
		fmt.Fprintf(stderr, "%s %s\n\n", appName, appVersion)
		fmt.Fprintf(stderr, "Extracts a Microsoft Money data file (.mny) or backup (.mbf).\n\n")
		fmt.Fprintf(stderr, "Usage:\n  hb-money-extractor [options] [file.mny|file.mbf]\n\n")
		fmt.Fprintf(stderr, "Run without a file (or without -f) to be asked what to extract.\n\nOptions:\n")
		flags.PrintDefaults()
	}

	if err := flags.Parse(args); err != nil {
		if err == flag.ErrHelp {
			return 0
		}

		return 2
	}

	if *showVersion {
		fmt.Fprintf(stdout, "%s %s\n", appName, appVersion)
		return 0
	}

	if *listFormats {
		printFormats(stdout)
		return 0
	}

	reader := bufio.NewReader(stdin)
	interactive := flags.NArg() == 0 || *formatList == ""
	inputPath := ""

	if flags.NArg() > 1 {
		fmt.Fprintln(stderr, "Give one input file at a time.")
		return 2
	} else if flags.NArg() == 1 {
		inputPath = flags.Arg(0)
	}

	selected, err := parseFormats(*formatList)

	if err != nil {
		fmt.Fprintln(stderr, err)
		return 2
	}

	if interactive {
		fmt.Fprintf(stdout, "%s %s\n\n", appName, appVersion)
	}

	if inputPath == "" {
		inputPath = ask(reader, stdout, "Microsoft Money file to extract (.mny or .mbf): ")
	}

	inputPath = cleanPath(inputPath)

	if inputPath == "" {
		fmt.Fprintln(stderr, "No file selected.")
		return finish(interactive, reader, stdout, 2)
	}

	data, err := os.ReadFile(inputPath)

	if err != nil {
		fmt.Fprintf(stderr, "Cannot read %q: %v\n", inputPath, err)
		return finish(interactive, reader, stdout, 1)
	}

	fmt.Fprintf(stdout, "Reading %s ...\n", filepath.Base(inputPath))

	extracted, err := msmoney.ExtractMicrosoftMoneyData(core.NewNullContext(), data)

	if err != nil {
		fmt.Fprintf(stderr, "Cannot extract %q: %s\n", inputPath, describeError(err))
		return finish(interactive, reader, stdout, 1)
	}

	printSummary(stdout, extracted)

	if len(extracted.Transactions) < 1 && len(extracted.Accounts) < 1 {
		fmt.Fprintln(stderr, "The file holds no accounts or transactions to extract.")
		return finish(interactive, reader, stdout, 1)
	}

	if len(selected) < 1 {
		selected = chooseFormats(reader, stdout)
	}

	targetDir := *outputDir

	if targetDir == "" && interactive {
		targetDir = cleanPath(ask(reader, stdout, "Output folder [same folder as the input file]: "))
	}

	if targetDir == "" {
		targetDir = filepath.Dir(inputPath)
	}

	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		fmt.Fprintf(stderr, "Cannot create the output folder %q: %v\n", targetDir, err)
		return finish(interactive, reader, stdout, 1)
	}

	baseName := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))
	failed := false

	for _, selectedFormat := range selected {
		path, err := writeFormat(selectedFormat, targetDir, baseName, extracted)

		if err != nil {
			fmt.Fprintf(stderr, "  %-22s FAILED: %v\n", selectedFormat.name, err)
			failed = true
			continue
		}

		fmt.Fprintf(stdout, "  %-22s %s\n", selectedFormat.name, path)
	}

	if failed {
		return finish(interactive, reader, stdout, 1)
	}

	fmt.Fprintln(stdout, "\nDone.")

	return finish(interactive, reader, stdout, 0)
}

// finish keeps the window open when the program was started by double-clicking
// it, so the messages can be read before it closes.
func finish(interactive bool, reader *bufio.Reader, stdout io.Writer, code int) int {
	if interactive && isTerminal(os.Stdin) {
		ask(reader, stdout, "\nPress Enter to exit. ")
	}

	return code
}

func printSummary(stdout io.Writer, data *msmoney.ExtractedData) {
	fmt.Fprintf(stdout, "Found %d accounts and %d transactions", len(data.Accounts), len(data.Transactions))

	if len(data.Transactions) > 0 {
		first, last := data.Transactions[0].Time, data.Transactions[0].Time

		for _, transaction := range data.Transactions {
			if transaction.Time.Before(first) {
				first = transaction.Time
			}

			if transaction.Time.After(last) {
				last = transaction.Time
			}
		}

		fmt.Fprintf(stdout, " (%s to %s)", first.Format("2006-01-02"), last.Format("2006-01-02"))
	}

	fmt.Fprintln(stdout, ".")

	if data.Skipped > 0 {
		fmt.Fprintf(stdout, "%d rows were left out: investment activity, split totals, and rows without an amount, account or date.\n", data.Skipped)
	}

	fmt.Fprintln(stdout)
}

func chooseFormats(reader *bufio.Reader, stdout io.Writer) []outputFormat {
	for {
		fmt.Fprintln(stdout, "Output formats:")
		printFormats(stdout)
		fmt.Fprintf(stdout, "  %d) All of the above\n\n", len(allFormats)+1)

		answer := ask(reader, stdout, "Choose one or more, separated by commas [1]: ")

		if answer == "" {
			return []outputFormat{allFormats[0]}
		}

		var chosen []outputFormat
		valid := true

		for _, item := range strings.Split(answer, ",") {
			number, err := strconv.Atoi(strings.TrimSpace(item))

			if err != nil || number < 1 || number > len(allFormats)+1 {
				fmt.Fprintf(stdout, "%q is not one of the choices.\n\n", strings.TrimSpace(item))
				valid = false
				break
			}

			if number == len(allFormats)+1 {
				return allFormats
			}

			chosen = append(chosen, allFormats[number-1])
		}

		if valid && len(chosen) > 0 {
			return chosen
		}
	}
}

func ask(reader *bufio.Reader, stdout io.Writer, prompt string) string {
	fmt.Fprint(stdout, prompt)

	line, _ := reader.ReadString('\n')

	return strings.TrimSpace(line)
}

// cleanPath removes the quotes a terminal adds when a file is dragged onto it.
func cleanPath(path string) string {
	path = strings.TrimSpace(path)

	if len(path) >= 2 && (path[0] == '"' || path[0] == '\'') && path[len(path)-1] == path[0] {
		path = path[1 : len(path)-1]
	}

	return path
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()

	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// describeError explains a failure in terms of the file the user picked.
func describeError(err error) string {
	if err.Error() == errs.ErrImportFileTypeNotSupported.Message {
		return "this is not a Microsoft Money data file (.mny) or backup (.mbf)"
	}

	return err.Error()
}
