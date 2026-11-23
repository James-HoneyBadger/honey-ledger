# HB Money Extractor

HB Money Extractor is a standalone program that reads a Microsoft Money data file
(`.mny`) or backup (`.mbf`) and writes its accounts and transactions in the format
you choose. It needs no Honey Ledger server, database or login, and it never
changes the file it reads.

## Using it

Double-click the program, or run it from a terminal, and answer the questions:

1. the Money file to extract (you can drag the file onto the window),
2. the output formats (one, several, or all),
3. the output folder (Enter keeps the folder of the input file).

For scripts, give everything on the command line:

```sh
hb-money-extractor -f csv,xlsx -o ./exported "2026_11_21_Backup.mbf"
```

| Option | Meaning |
|--------|---------|
| `-f`   | Formats, comma separated, or `all` |
| `-o`   | Output folder (default: next to the input file) |
| `-list` | List the formats |
| `-version` | Print the version |

Existing files are never overwritten; a number is added to the new file's name.

## Formats

| Key | File | Use it for |
|-----|------|------------|
| `csv` | `.csv` | Excel and other spreadsheets (UTF-8 with a byte order mark) |
| `honeyledger` | `.honeyledger.csv` | Importing into Honey Ledger without mapping any column |
| `tsv` | `.tsv` | Tab-separated tools |
| `xlsx` | `.xlsx` | An Excel workbook with Transactions and Accounts sheets |
| `json` | `.json` | Scripts and other programs |
| `qif` | `.qif` | Quicken and most finance applications |

In every format except `honeyledger`, amounts are signed from the account's point
of view: money coming in is positive, money going out or moving away is negative.
A transfer is listed once, in the account the money left, with the receiving
account in the *Transfer Account* column. In QIF it appears in both accounts so the
accounts balance.

## What is left out

Investment activity, the total line of a split transaction, and rows without an
amount, account or date cannot become ordinary transactions. The program reports
how many rows were left out. Password-protected Money files are not supported.

## Building it

```sh
go build -o hb-money-extractor ./cmd/hb-money-extractor
scripts/build-money-extractor.sh   # Linux, Windows and macOS builds in dist-extractor/
```
