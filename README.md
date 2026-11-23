# Honey Ledger

## Open your Microsoft Money file on any computer

**No Windows. No copy of Money. No converter. No upload.** Honey Ledger reads the `.mny` database directly — including ledgers written decades ago.

[![CI](https://github.com/James-HoneyBadger/honey-ledger/actions/workflows/ci.yml/badge.svg)](https://github.com/James-HoneyBadger/honey-ledger/actions/workflows/ci.yml)
**Free software** (GPL-3.0-or-later) · **Windows**, **macOS**, **Linux** · **19 languages** · **159 currencies**

A local-first personal finance desktop app that reads the file formats your bank and your old software give you, reports on them properly, and never phones home.

Honey Ledger is a self-contained Electron desktop application for individuals who want full control over their financial data. The Go backend runs inside the app and everything you enter lives in a database you control — SQLite on your own machine by default, or a MySQL/PostgreSQL server you point it at. There is no account to create and no server to run. Network access is limited to the features that need it: exchange-rate downloads, map tiles, SMTP mail, your own object storage, and update checks. None of them carry telemetry and every one is optional: leave them unconfigured or switch them off and the app runs fully offline. Financial analysis is local and rule-based; there is no built-in AI and no model endpoint. [SECURITY.md](SECURITY.md) lists every outbound connection with the setting that turns it off, and the [documentation](docs/) describes the switches in detail.

## Microsoft Money, read natively

Microsoft Money was discontinued in 2009 and support ended in 2011. *Money Plus Sunset* still opens the files, but only on Windows — and several international editions of Sunset could never read non-US data files at all. Fifteen years on, a great many people still hold a `.mny` file containing a decade or two of accounts, transactions, splits and categories, with nowhere modern to put it.

The obstacle is the file. A `.mny` is a Jet 4 database in a variant the engine calls **MSISAM**, and that closes every ordinary door:

- **Jet and Access drivers refuse it**, because the header does not name their engine.
- **LibreOffice Base refuses it too** — it reads Access databases through those very drivers.
- **Quicken, GnuCash, HomeBank and Moneydance expect QIF, OFX or CSV**, not the database itself.
- **Producing that interchange file normally means running Money**, which is exactly what someone on macOS, on Linux, or holding only a backup file cannot do.

Honey Ledger removes the step instead of documenting it. In the transaction list click **Import**, choose **Microsoft Money File Format** and then **Microsoft Money Data File (MNY)**, pick the file, and it reads the database itself, in-process, on every platform the app runs on:

| What comes across | Detail |
|---|---|
| **Accounts** | with their currencies and account types |
| **Categories** | primary and secondary, nested the way Money nests them |
| **Transactions** | expense and income entries, with memos carried over as descriptions |
| **Transfers** | one transaction per pair of accounts, not two unrelated rows |
| **Splits** | one transaction per split, allocated to the right categories |
| **Backups (`.mbf`)** | Money's backups are Microsoft Cabinet archives around the same database; Honey Ledger unpacks one itself and imports the database inside |

**Verified end to end against a real Money Sunset database: 165 accounts, 193 categories and 39,293 transaction rows, of which 28,057 become transactions** — the remainder are investment activity and split totals, which have no counterpart in the ledger. Decoded from the binary and imported with no external program, no Windows, and no network involved.

Being straight about the limits, because an import you can trust matters more than one that claims everything: investment activity (share lots and prices) has no counterpart in the target ledger and is skipped; Money 2000 and earlier use an older 2048-byte page layout that is not read yet; and a password-protected file is reported as exactly that rather than failing silently. Quicken's own `.qdf` is likewise proprietary and still out of reach — selecting one explains which QIF/OFX export to use instead of returning a generic "unsupported file type".

The entire format — the header, the scrambled block that hides the database key, page types, table definitions and the row layout — is written down in [docs/MONEY_FILE_FORMAT.md](docs/MONEY_FILE_FORMAT.md), because a format this close to extinction deserves a record that outlives the software that made it.

If you want the data out of Money without using Honey Ledger, [HB Money Extractor](docs/HB_MONEY_EXTRACTOR.md) is a small standalone program built from this repository that reads the same files and writes CSV, Excel, JSON or QIF.

## What it does best

| | |
|---|---|
| **Microsoft Money files, natively** | The headline feature — see [above](#microsoft-money-read-natively). A `.mny` database read in-process on Windows, macOS and Linux: accounts, currencies, categories, transactions, transfers, splits, and Money's `.mbf` backups unpacked for you, with no export step and no Windows install. Plus QFX, QIF, OFX, IIF, CSV/TSV/SSV, Excel, Camt.052/053, MT940, GnuCash, Firefly III, Beancount, Alipay, WeChat Pay, JD.com and Feidee — 24 distinct import formats in the dialog. |
| **Your data stays yours** | SQLite on your disk by default, no cloud settings sync, no telemetry. Every outbound connection is documented with the switch that turns it off; unconfigure them all and the app is fully offline. |
| **Reporting that answers questions** | 18 chart types over categorical, trend and asset analyses — spend by category, income by account, cash flow, net worth over time — every figure converted into your base currency. An insights explorer builds your own queries, and transaction review flags probable duplicates, odd amounts and subscriptions. |
| **Multi-currency for real** | 159 currencies with a currency per account and per user, 15 central-bank and public rate feeds, manual overrides, and reports that add unlike currencies up correctly. |
| **Analysis without handing over your ledger** | A local, rule-based analysis endpoint (`POST /api/v1/analysis.json`) reports your totals, top categories, largest expenses, recurring payments, unusual spending and balances — all computed on your machine, with no AI and no model endpoint. |
| **Free software, and stays that way** | GPL-3.0-or-later, an SPDX header on every source file, and a CI check that fails the build if one goes missing. |
| **Desktop-native, not a web page** | System tray, global shortcuts, desktop notifications, app lock, automatic updates, dark mode, and native installers for all three platforms. |
| **Runs how you want** | SQLite, MySQL or PostgreSQL; a full REST API plus MCP; 19 languages and 139 timezones, each user with their own. |
| **Secure by default, restrictable per user** | Two-factor authentication, OAuth2/OIDC single sign-on, session management, API tokens, and 16 capabilities an administrator can switch off for a given user. |

## What's new in 2.0

- **Microsoft Money files, read natively — the headline.** Point Honey Ledger at a Money `.mny` file and it decodes the Jet/MSISAM database itself: accounts with their currencies, categories as Money nests them, expense and income transactions, one transfer per pair of accounts, and one transaction per split. Verified against a real Money Sunset database — 165 accounts, 193 categories and 39,293 transaction rows, 28,057 of them importable. No Windows, no copy of Money, no converter, no upload. Money's `.mbf` backups work too — they are Microsoft Cabinet archives around the same database, and Honey Ledger unpacks one itself, so there is nothing to extract by hand. Format reference: [docs/MONEY_FILE_FORMAT.md](docs/MONEY_FILE_FORMAT.md).
- **Local-only analysis** — the AI assistant and LLM receipt recognition are gone. In their place, a deterministic analysis endpoint reports totals, top categories, largest expenses, recurring payments and unusual spending, computed entirely on your machine.
- **Free software** — Honey Ledger moved from MIT to the GNU GPL v3.0 or later.

## Features

### Ledger

- **Accounts** — checking, savings, credit card, cash, investment, loan and more, with sub-accounts, hiding, reordering, and a per-account currency.
- **Transaction Types** — expense, income, transfer, and balance modification.
- **Categories** — primary and secondary levels, with presets for common categories and custom trees.
- **Tags & Tag Groups** — cross-cutting labels with colour grouping.
- **Transaction Pictures** — attach receipt images to a transaction.
- **Geotagged Transactions** — record where a transaction happened and see it on a map.
- **Draft Auto-Save** — the transaction form is preserved if you navigate away.
- **Account Reconciliation** — statement support for balancing an account.
- **Shared Expenses** — split an expense with up to 20 people, see who owes whom in either direction, record settlements, and filter the transaction list down to shared expenses.

### Automation

- **Scheduled Transactions** — recurring transactions with built-in cron scheduling.
- **Transaction Templates** — save and reuse common transaction patterns.
- **Automatic Backups** — scheduled SQLite snapshots with configurable retention.

### Reporting & Analysis

- **Overview Home Page** — balances, recent activity, and month-to-date totals at a glance.
- **Statistics** — 18 chart types over three analyses: expense and income by account, primary or secondary category; total and net income, expense, inflows, outflows and cash flow over time; and asset trends including total assets, total liabilities and net worth. Every figure is converted into your base currency.
- **Insights Explorer** — build custom queries over your own data and visualise the result as a chart or table.
- **Transaction Review** — points out possible duplicate charges, unusual amounts, and repeating subscriptions, with dismissible findings that link to the transaction they came from.

### Import & Export

- **Microsoft Money** — `.mny` data files read directly (see [Migrating from Microsoft Money](docs/USER_GUIDE.md#migrating-from-microsoft-money-or-quicken)).
- **Quicken** — QFX, QIF (Y/M/D, M/D/Y, D/M/Y), IIF; `.qdf` files are recognised and explained.
- **Generic** — CSV, TSV, SSV, Excel (`.xlsx`, `.xls`), with column mapping and custom transformation scripts.
- **Open formats** — OFX, Camt.052/053, MT940 (SWIFT), GnuCash XML, Firefly III, Beancount.
- **Regional exports** — Alipay (app and web CSV), WeChat Pay (app and export CSV/XLSX), JD.com Finance, Feidee MyMoney.
- **Export** — filtered transactions to CSV or TSV.

### Currencies & Rates

- **Multi-Currency** — 159 currencies, with a per-account and per-user currency.
- **Exchange Rates** — 15 central-bank and public rate feeds, automatic updates, manual overrides, and on-demand lookup.

### Analysis & Integrations

- **Financial Analysis** — a local, rule-based report (`POST /api/v1/analysis.json`): totals per currency, top categories, largest expenses, recurring payments, unusual spending, account balances and shared expenses. No AI, no model endpoint, nothing sent off the machine.
- **MCP Support** — Model Context Protocol server so external tools can read your ledger with a scoped token.
- **REST API & Tokens** — the whole app is an HTTP API; generate tokens for automation.

### Desktop

- **Desktop Only** — native Electron application for Windows, macOS and Linux. No browser needed.
- **System Tray** — minimise to tray, with desktop notifications.
- **Global Shortcuts** — quick-add a transaction (`Ctrl/Cmd+Shift+N`) or focus the window (`Ctrl/Cmd+Shift+M`).
- **Auto-Login** — single-user mode skips the login screen.
- **Application Lock** — a PIN lock over a running session.
- **Dark Mode** — full dark theme with automatic system detection.
- **Automatic Updates** — packaged builds check for new releases and install them on quit.
- **Admin Tool** — a standalone Fyne desktop tool for database management, users and diagnostics.

### Security

- **Two-Factor Authentication** — TOTP with recovery codes.
- **OAuth2 / OIDC** — single sign-on with an external identity provider, linkable and unlinkable per user.
- **Sessions** — see and revoke active sessions.
- **Per-User Feature Restrictions** — 16 capabilities an administrator can switch off for a user, from changing a password to importing or exporting.

### Data & Storage

- **Multiple Databases** — SQLite (default), MySQL, or PostgreSQL.
- **Backups** — create, download, restore and delete consistent snapshots.
- **Data Management** — clear all data, and choose where avatars and receipts are stored, including your own S3-compatible object storage.

### Internationalisation

- **Multi-Language** — 19 languages including English, German, French, Spanish, Italian, Japanese, Korean, and Chinese.
- **Multi-Timezone** — 139 timezones, with per-user timezone configuration and timezone-aware dates and statistics.


## Quick Start

### Run the Desktop App

```bash
./run.sh
```

The script installs missing frontend/Electron dependencies, builds the Go backend, builds the frontend when needed, and launches the Electron app. Use `./run.sh --rebuild` for a forced frontend rebuild, `./run.sh --server` for the backend only, or `./run.sh --help` for all options.

To launch Electron manually instead, build both halves first:

```bash
go build -o honey-ledger .
npm install && npm run build
cd electron && npm install && npm start
```

### Build from Source

```bash
./scripts/build.sh package
```

Then extract the archive and run the binary. See [docs/INSTALLATION.md](docs/INSTALLATION.md) for complete instructions. Release tags also build native Electron installers (AppImage/.deb, .dmg, NSIS) through the GitHub Actions workflow.

## Default Credentials

On first launch with no existing database, Honey Ledger creates a default administrator account (`admin`) with a random password. The password is printed once to the console on that first start and is never written to the log file. If auto-login is enabled (default), the login screen is skipped entirely; you can set a new password at any time with `honey-ledger userdata user-modify-password --username admin --password <new-password>`.

## Keyboard Shortcuts

| Shortcut | Action |
|----------|--------|
| `Ctrl+Shift+N` / `Cmd+Shift+N` | Quick-add transaction |
| `Ctrl+Shift+M` / `Cmd+Shift+M` | Show/focus main window |

## Admin Tool

A standalone Fyne-based desktop admin tool is available in the `admin/` directory for database management, user administration, and diagnostics.

```bash
cd admin
go run .
```

## Documentation

| Document | Description |
|----------|-------------|
| [Installation Guide](docs/INSTALLATION.md) | Installation methods: binary, source, Electron packaging |
| [Configuration Reference](docs/CONFIGURATION.md) | Complete configuration file reference |
| [User Guide](docs/USER_GUIDE.md) | End-user guide for all application features, including shared expenses and transaction review |
| [API Reference](docs/API_REFERENCE.md) | REST API endpoint documentation |
| [Architecture Overview](docs/ARCHITECTURE.md) | System design, project structure, and technology stack |
| [Microsoft Money File Format](docs/MONEY_FILE_FORMAT.md) | How a Money data file is laid out, and how the importer reads it |
| [HB Money Extractor](docs/HB_MONEY_EXTRACTOR.md) | Standalone tool that extracts a Money file to CSV, Excel, JSON or QIF without Honey Ledger |
| [Development Guide](docs/DEVELOPMENT.md) | Setting up a development environment, building, testing |
| [Electron Desktop App](docs/ELECTRON.md) | Desktop application build, packaging, and usage |
| [CLI Reference](docs/CLI_REFERENCE.md) | Command-line interface for server, database, and user management |
| [Contributing](docs/CONTRIBUTING.md) | Contribution guidelines, code style, and workflow |
| [Security Policy](SECURITY.md) | Supported versions, threat model, and private vulnerability reporting |
| [Changelog](CHANGELOG.md) | Release history |

## Technology Stack

| Layer    | Technology |
|----------|------------|
| Backend  | Go 1.26, Gin, xorm |
| Frontend | Vue 3, Vuetify 3, TypeScript, Vite |
| Desktop  | Electron 45 |
| Database | SQLite 3 / MySQL / PostgreSQL |
| Build    | Vite 7, Go toolchain, electron-builder |

## License

[GNU General Public License v3.0 or later](LICENSE) — Copyright (c) 2020-2026 Honey Badger Universe

Honey Ledger is free software: you can redistribute it and/or modify it under the terms of the GNU General Public License as published by the Free Software Foundation, either version 3 of the License, or (at your option) any later version.

This program is distributed in the hope that it will be useful, but WITHOUT ANY WARRANTY; without even the implied warranty of MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the [GNU General Public License](LICENSE) for more details.
