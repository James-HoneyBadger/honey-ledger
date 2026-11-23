# HB Money Extractor

HB Money Extractor is a standalone program that reads a Microsoft Money data file
(`.mny`) or backup (`.mbf`) and writes its accounts and transactions in the format
you choose. It needs no Honey Ledger server, database or login, and it never
changes the file it reads.

## Using it

Double-click the program and answer the questions in the window:

1. the Money file to extract (you can drag the file onto the window),
2. the output formats (one, several, or all),
3. the output folder (Enter keeps the folder of the input file).

On Linux, a file manager may start the program without opening a window. Run it in
a terminal instead, then drag the file onto that terminal to paste its path.

The window stays open after the work is done, until you press Enter, so the result
can be read before it closes.

For scripts, give everything on the command line and nothing is asked:

```sh
hb-money-extractor -f csv,xlsx -o ./exported "2026_11_21_Backup.mbf"
```

| Option | Meaning |
|--------|---------|
| `-f`   | Formats, comma separated, or `all` |
| `-o`   | Output folder, created if it does not exist (default: next to the input file) |
| `-list` | List the formats and their keys |
| `-version` | Print the version |
| `-h`   | Print the usage summary |

Existing files are never overwritten; a number is added to the new file's name
(`ledger (1).csv`), so running it twice never destroys an earlier export.

The exit status is `0` when every requested file was written, `1` when the file
could not be read, extracted or written, and `2` for a bad option or format name.

## Formats

| Key | File | Use it for |
|-----|------|------------|
| `csv` | `.csv` | Excel and other spreadsheets (UTF-8 with a byte order mark) |
| `honeyledger` | `.honeyledger.csv` | Importing into Honey Ledger without mapping any column |
| `tsv` | `.tsv` | Tab-separated tools (UTF-8 with a byte order mark) |
| `xlsx` | `.xlsx` | An Excel workbook with Transactions and Accounts sheets |
| `json` | `.json` | Scripts and other programs |
| `qif` | `.qif` | Quicken and most finance applications |

In every format except `honeyledger`, amounts are signed from the account's point
of view: money coming in is positive, money going out or moving away is negative.
A transfer is listed once, in the account the money left, with the receiving
account in the *Transfer Account* column. In QIF it appears in both accounts so the
accounts balance.

`honeyledger` writes the layout Honey Ledger itself exports, with the amount
unsigned and the type in its own column, so the file imports without column
mapping. Every field is quoted when it needs to be, as RFC 4180 requires, so a
memo containing a comma, a quote or a line break arrives intact.

## What is left out

Investment activity, the total line of a split transaction, and rows without an
amount, account or date cannot become ordinary transactions. The program reports
how many rows were left out, so the count is never hidden.

## Troubleshooting

| Message | Meaning |
|---------|---------|
| `this is not a Microsoft Money data file (.mny) or backup (.mbf)` | The file has no Money header. Check that you picked the ledger itself, not a report or a lost file named `.mny`. |
| `the file is password protected; remove the password in Microsoft Money and try again` | Money encrypted the database with its own password. Open it in Money once, remove the password, save, and extract again. |
| `the file is damaged, or is not a complete Microsoft Money database` | The header is present but a table could not be read, for example a copy that was cut short. Copy the file again from Money or from the original backup. |

Money 2000 and earlier use an older 2048-byte page layout, which is not read yet.

## Building it

```sh
go build -o hb-money-extractor ./cmd/hb-money-extractor
scripts/build-money-extractor.sh   # Linux, Windows and macOS builds in dist-extractor/
```

Each Honey Ledger release attaches the program for Linux (x86-64, arm64), Windows
(x86-64) and macOS (Intel, Apple silicon), named
`hb-money-extractor-<os>-<arch>` (`hb-money-extractor-windows-amd64.exe` on
Windows). Download the one for your machine, and run it: there is nothing to
install.

`scripts/build-money-extractor.sh` produces the same five files in
`dist-extractor/` with CGO disabled, so the result is a single file with no
dependencies.
