# Changelog

All notable changes to Honey Ledger are documented here.

The format follows [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and the project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Releases before 1.4.2 are not catalogued here; see the commit history for
details.

## [Unreleased]

### Changed

- Microsoft Money imports decode payees: a transaction without a memo uses its
  payee as the description instead of importing without one.
- Microsoft Money imports carry account closed/hidden flags, split-integrity
  validation, and a validation summary through the synchronized engine.

## [2.2.0] - 2026-11-24

### Added

- Microsoft Money imports preserve each side of a transfer in its own currency:
  a cross-currency transfer carries the amount the receiving account got, and
  falls back to the source amount when the receiving side cannot be matched.
- Backup restore writes the restored database to a temporary file and only
  replaces the live database after the bytes are complete and carry a SQLite
  header, so an interrupted or corrupt restore cannot leave the live database
  unusable.

### Changed

- The Microsoft Money packages (MSISAM engine, extraction, output writers,
  synthetic test files) are carried in-repo again, so Honey Ledger builds
  self-contained. The copy is kept in lockstep with the one HB SunRise carries.

### Fixed

- The MSISAM engine bounds the row count and long values a corrupt file can
  claim, and the extractor's CSV/XLSX writers neutralize spreadsheet formulas
  in imported text.
- The QIF reader fails an import whose lines exceed the supported length
  instead of silently dropping the rest of the file, and the SGML decoder
  honors the encoding declared by the document.
- The National Bank of Romania exchange-rate source points at the bank's new
  XML address on curs.bnr.ro.

### Security

- The extraction engine no longer trusts the sizes a crafted Microsoft Money
  file declares for buffers (row counts and chained long values are clamped).
## [2.1.1] - 2026-11-23

### Fixed

- The HB Money Extractor reported its version as 1.0.0 regardless of the release
  it shipped with. `scripts/build-money-extractor.sh` now stamps the binary with
  the version from `package.json` at build time, so its `-version` output and the
  `source` field of its JSON report the same number as the app it came with. A
  plain `go build` reports `dev`.

## [2.1.0] - 2026-11-23

### Added

- HB Money Extractor (`cmd/hb-money-extractor`), a standalone program that reads a
  Microsoft Money data file (`.mny`) or backup (`.mbf`) and writes CSV, Honey
  Ledger CSV, TSV, an Excel workbook, JSON or QIF. It needs no server, database or
  login, so a Money file can be converted on a machine that does not run Honey
  Ledger. See [docs/HB_MONEY_EXTRACTOR.md](docs/HB_MONEY_EXTRACTOR.md) and
  `scripts/build-money-extractor.sh`.
- **Create Missing Items** in the import review creates the accounts, categories
  and tags a file references but the ledger does not have, then maps the parsed
  rows onto them.
- **Data Transfer** entries in the sidebar, and an Import card in Settings → Data
  Management, so importing and exporting no longer require finding the button in
  the transaction list.
- An export dialog offering CSV or TSV. The desktop app writes the file through a
  native Save dialog instead of dropping it in the downloads folder.

### Changed

- CSV and TSV export quote fields that contain a comma, tab, quote or line break,
  following RFC 4180, so descriptions and tags survive an export and re-import
  unchanged. Files written by earlier versions still import.
- The import review now applies the same rules the server applies when it saves a
  transaction: a category must be a subcategory of the matching type, and hidden
  accounts, categories and tags are refused. Rows that cannot be imported are
  reported and skipped instead of rejecting the whole batch.
- `max_import_file_size` defaults to 64 MiB (was 10 MiB).
- `enable_import` and `enable_export` default to `true`, matching the documented
  defaults. A configuration file that sets them explicitly is unaffected.
- The request that parses an uploaded file may take five minutes, up from thirty
  seconds, so a large Money file is not cut off mid-parse.

### Fixed

- Importing more than 65,535 IDs — roughly 32,000 transfers, which need two each —
  overflowed a counter; IDs are now generated in chunks, so large ledgers import.
- Comments longer than 255 characters aborted an import; they are truncated to the
  limit instead.
- A transaction whose latitude or longitude is exactly 0 was rejected as missing
  the coordinate.
- The import dialog's help link pointed at a page that does not exist; it now opens
  the matching section of the user guide.

## [2.0.0] - 2026-11-22

The first release under the GNU General Public License, and the first with no
built-in AI: the large language model features were removed and replaced with a
local, rule-based analysis endpoint. The CLI, configuration format and database
schema are unchanged, so an existing installation upgrades in place; the major
version marks the license change and the removal of the AI API endpoints, both of
which affect anyone shipping or embedding Honey Ledger.

### Added

- Microsoft Money data files (`.mny`) now import directly. Money saves its ledger
  in a Jet database that names the MSISAM engine, and the importer reads it
  without an export step: accounts with their currencies, categories grouped as
  Money nests them, expense and income transactions, one transfer per pair of
  accounts, one transaction per split, and memos as descriptions. Investment
  activity has no counterpart in the ledger and is skipped. A password protected
  file is reported as such instead of failing silently.
- A reader for the Microsoft Money data file format
  (`pkg/converters/msmoney/msisam`). It decodes the database header, the table
  definitions, the catalog that names the tables and the values in their rows,
  and reads a real Money Sunset database end to end: 165 accounts, 193
  categories and 39 293 transaction rows, of which 28 057 become transactions
  (investment activity and split totals are skipped). The format is documented in
  [docs/MONEY_FILE_FORMAT.md](docs/MONEY_FILE_FORMAT.md).
- Import dialog entries for Microsoft Money (`.mny`) and Quicken (`.qdf`) data
  files. Quicken files cannot be decoded, so selecting one explains which
  QIF/OFX export to use instead, rather than failing with a generic "import file
  type not supported" error.
- A local financial analysis endpoint, `POST /api/v1/analysis.json`. It computes
  income, expense and net totals per currency, the top expense categories, the
  largest expenses, recurring (subscription-like) payments, unusual spending,
  account balances and shared expenses — entirely on the machine, with no model
  and no outbound call.
- Microsoft Money backups (`.mbf`) import directly. A backup is a Microsoft
  Cabinet around the same database, so instead of asking the user to run
  `cabextract`, the importer expands the archive in memory — stored, MSZIP and
  LZX members are all read — and finds the database by its `MSISAM` header rather
  than by name, because Money names the member `mbf…tmp`. The expansion is capped
  at 512 MiB and 32 members so a crafted cabinet cannot exhaust memory, and a
  damaged backup is reported as such. The import dialog offers
  **Microsoft Money Backup File (MBF)** next to the data file.

### Changed

- **Honey Ledger is now licensed under the GNU General Public License v3.0 or
  later** (`GPL-3.0-or-later`) instead of the MIT License. Every source file
  carries an SPDX header, the copyright and warranty notice moved to `NOTICE`,
  and the license ships with the standalone and Electron packages. Anyone
  redistributing Honey Ledger, or building on it, now does so under the terms of
  the GPL.
- The README and the user guide describe the complete feature set, grouped by
  what you do with it, and the user guide documents the local financial analysis
  and the statistics chart types.

### Removed

- Cloud settings sync. Preferences stay in the local database; the three
  `/api/v1/users/settings/cloud/*` endpoints and the Settings Sync tab are
  gone.

- The AI receipt recognition feature (`POST
  /api/v1/llm/transactions/recognize_receipt_image.json`) and all large language
  model provider integrations (OpenAI, OpenAI-compatible, Anthropic,
  Anthropic-compatible, OpenRouter, Ollama, LM Studio and Google AI).
- The AI assistant endpoint (`POST /api/v1/ai/ask.json`), replaced by the local
  `POST /api/v1/analysis.json` endpoint.

### Fixed

- `third-party-dependencies.json` no longer lists the unused `gin-contrib/gzip`
  middleware, and the bundled Freepik illustrations record their license and
  attribution instead of only a link to Freepik's legal page.

### Documentation

- New [Microsoft Money file format reference](docs/MONEY_FILE_FORMAT.md): the
  header, the scrambled block that hides the database key, page types, table
  definitions, the row layout, and how the importer turns the tables into
  transactions.

## [1.5.0] - 2026-10-31

### Added

- Shared expenses: split an expense transaction with up to 20 people, review what
  each person owes per currency, and drill into the individual expenses behind a
  total. Backed by `GET /api/v1/transactions/splits/{list,summary,person}.json`
  and `POST /api/v1/transactions/splits/update.json`.
- A `Shared Expenses` page under Basis Data, with a split editor that shows the
  transaction amount and the unallocated remainder while editing. Shared
  expenses are also marked directly in the transaction list, where a chip shows
  the amount others owe and opens the split editor, and expense rows offer the
  same editor next to the description.
- Versioned database migrations with a `schema_migration` ledger, applied at
  startup after the automatic schema sync. Each migration runs once in its own
  transaction together with the row that records it, so a failure rolls the
  change back instead of leaving a half-applied schema.
- Shared expense settlements: record a payment between the user and another
  person in either direction and see net balances ("owes you" / "you owe")
  instead of only what the user is owed. Splits can now point either way, so a
  transaction someone else paid for is recorded as a share the user owes.
- Person handling for shared expenses: names are suggested from previous
  entries, a rename rewrites every split and settlement so a typo does not
  create a second person, and the split editor can divide a transaction equally
  across the listed people.
- The first real database migration, which backfills the direction of splits
  created before directions existed.
- Transaction review findings can be dismissed and restored, so a known-fine
  duplicate charge stops reappearing. The duplicate and unusual thresholds are
  accepted as request parameters instead of being fixed at build time, and every
  finding links through to the transaction it came from.
- Backend error messages are translated: all 305 messages the API can return now
  have an `error.` entry in every locale, and `scripts/check-locales.mjs` fails
  when a new message is added without one.
- The import column mapping is remembered per file layout, so importing the same
  export twice no longer means repeating the mapping, with a menu action to
  forget it.
- The transaction list reports the shared expense state of each row in the same
  response, so marking split transactions no longer costs a second request per
  page. This covers both the paged list and the single-month list.
- A shared expense filter on the transaction list (`shared_filter=shared` or
  `unshared` on `GET /api/v1/transactions/list.json` and
  `.../list/by_month.json`) with a toggle next to the search box. The filter runs
  inside the query, so paging and the total count stay exact, and it survives a
  reload of the filtered address.
- An AI assistant API (`POST /api/v1/ai/ask.json`) that answers questions about
  the user's finances with a language model the user configures in
  `[llm_ai_assistant]`. It is disabled by default, and only an aggregated,
  de-identified summary is sent: totals per currency, expense totals per category
  name, account balances and shared expense balances labelled by position. Each
  response reports what was sent in its `privacyNote`, and the settings support
  every provider the receipt recognition feature already supports.
- Pull request CI covering `gofmt`, `go vet`, generated OpenAPI verification, a
  backend build, `go test -short -race`, frontend lint/tests/build, Electron
  entry-point syntax checks, a `govulncheck` vulnerability scan, and repository
  hygiene checks.
- `SECURITY.md` with scope, threat model, and private disclosure process.
- Dependabot configuration for the Go modules, `admin/`, npm packages, and
  GitHub Actions, with minor and patch updates grouped.
- Export tests for the admin tool, covering dump contents, SQL quote escaping,
  and error propagation.
- Background update checks for packaged builds: the app downloads a newer
  release and installs it on quit, notifying the user when it is ready. Only
  AppImage and Windows installer builds self-update; set
  `HONEY_DISABLE_UPDATE_CHECK=1` to skip the check.

### Changed

- Renamed the application from Ratel Ledger to **Honey Ledger**. The rename covers
  the product name, package names, the application id
  (`com.honeybadger.honeyledger`), installers, exported file names, the config
  file (`honey-ledger.ini`), the SQLite database (`honey-ledger.db`) and the app
  icons, and is translated across all 19 locales.
- Existing installations upgrade in place: the previous data directory
  (`Ratel Ledger`, and `HB Moxie` before it) is migrated on first launch, the
  `ratel-ledger.ini`/`hbmoxie.ini` config files are still found and adopted, and
  the `RATEL_*`/`HBM_*` environment variables and `RATEL_DISABLE_UPDATE_CHECK`
  remain supported alongside the new `HONEY_*` names.
- Import mapping files exported before the rename still load through the legacy
  `ratelLedger…` JSON root fields.
- The Windows API helper is now `scripts/honeytools.ps1` to match its shell
  counterpart, and reads `HONEYTOOL_SERVER_BASEURL`/`HONEYTOOL_TOKEN`; the
  pre-rename `RATELTOOL_*` names are still picked up when the new ones are unset.
- The GitHub repository moved to `James-HoneyBadger/honey-ledger`, and the Go
  module path followed it (`github.com/James-HoneyBadger/honey-ledger`), so
  `go install` and imports use the new location. GitHub redirects the previous
  repository and clone URLs.
- The remaining pre-rename identifiers follow the product name: application
  settings and the application lock state are stored under `honey_…` browser
  storage keys, the random element ids are prefixed `honey_`, and the web session
  cookie is `honey_auth_token`. Values saved under the `ratel_…`/`hbm_…` keys and
  the `ratel_auth_token`/`hbm_auth_token` cookies are adopted on launch, so no
  preferences or signed-in sessions are lost.
- Upgrade the admin tool to Fyne 2.8.2 (from 2.5.4). Building it on Linux now
  requires the Wayland development headers in addition to X11.
- Upgrade the desktop shell to Electron 45.1 (from 33, which reached
  end-of-life in April 2025) and electron-builder 26, dropping the unused
  `electron-is-dev` dependency. The Electron 45 toolchain requires Node 22.12+.
- Builds are reproducible: `go mod download` replaces `go get .`, `npm ci`
  replaces `npm install`, and the pre-test `go clean -cache` is gone.
- `npm run lint` no longer rewrites files; `npm run lint:fix` applies fixes.
- Minimum Go version is 1.26, required by the updated `golang.org/x/net`.

### Fixed

- The application logo and splash artwork ship under their new names. The
  interface asked for `honey-ledger-logo.svg` while the files were still
  `ratel-ledger-*.svg`, so the logo was missing, and the artwork itself still
  read "RATEL LEDGER"; both now say Honey Ledger.
- English was missing the translation keys for the Persian, Chinese, and Hebrew
  calendar display options and for the names of every language in the language
  picker, so those menus rendered raw identifiers. Every locale now defines the
  same key set, enforced by `scripts/check-locales.mjs` in CI.
- The backend shuts down gracefully on `SIGTERM`/`SIGINT`: it stops accepting
  connections, drains in-flight requests, stops the cron scheduler, and closes
  database connections instead of exiting mid-write.
- The admin tool reports database iteration, scan, export, and file-close
  errors instead of silently producing a partial SQL dump that appeared
  successful.
- Data race in the interval cron test, exposed by enabling the race detector.
- `SECURITY.md` now lists every outbound connection with the setting that turns
  it off, and the README no longer counts Cloud Settings Sync as network access:
  synced preferences are stored in the application database.
- The CLI reference matches the commands again: `--features` (not
  `--feature-types`), the `user-session-new` `--type`/`--expiresInSeconds`
  flags, and `user-session-revoke --token`.
- The API reference documents the real map proxy paths
  (`/proxy/map/{tile,annotation}/:zoomLevel/:coordinateX/:fileName`) and the
  `map_data_fetch_proxy` key.
- The configuration reference documents `static_root_path` and notes the parser
  fallbacks for `[database] type` and `[mail] smtp_host` when the keys are
  omitted.

### Security

- Release artifacts now carry a SLSA build-provenance attestation, and are
  signed with minisign when a `MINISIGN_SECRET_KEY` secret is configured. The
  verification commands are documented in `SECURITY.md`.
- The generated default-user password is printed to the console only and is no
  longer written to the log file.
- Updated `golang.org/x/net`, `golang.org/x/crypto`, `xuri/excelize/v2`,
  `quic-go`, and `go-jose/v4`, clearing all vulnerabilities reachable from the
  application code according to `govulncheck`.
- Updated `golang.org/x/image` in the admin tool, clearing the four reachable
  image-decoding vulnerabilities reported by `govulncheck`.
- Cleared the Dependabot backlog. The runtime dependencies move to `axios`
  1.20.0 (prototype pollution through the fetch adapter, header injection, and
  an unhandled HTTP/2 `error` event), `vue` 3.5.42 and `vue-i18n` 11.4.14 (XSS
  through `srcdoc`, and `escapeParameter` leaving named values unescaped),
  `echarts` 6.1.0, `moment` 2.31.0 (path traversal through a crafted locale
  name) and `moment-timezone` with it.
- The build and test tooling follows: `vite` 7.3.7 (launch-editor command
  injection and the `server.fs.allow` path bypass), `eslint` 10.14 with
  `eslint-plugin-vue` 10.12 and `@vue/eslint-config-typescript` 14.9,
  `jest` 30.5.2 with `ts-jest` 29.4.15, and `vite-plugin-checker` 0.14.5.
  `minimatch` 11 and `glob` 14 are pinned through `overrides`, since the
  packages that depend on them still declare the vulnerable range.
- `git-rev-sync` is gone: the Vite config reads the commit hash with
  `git rev-parse --short HEAD`, which drops the `shelljs`/`glob`/`brace-expansion`
  chain that came with it.
- ESLint 10 requires Node 22.13 on the 22 line, so the documented minimum moves
  from 22.12 to 22.13.
- One low-severity advisory stays: `deepmerge` 4.3.1 has no fixed release, and
  only `jest-config` uses it, to merge test configuration at build time.
- `govulncheck` reports no reachable vulnerabilities. The only advisory left in
  the module graph is `GO-2026-5932` for the unmaintained `x/crypto/openpgp`
  package, which has no fix and which the application does not call.
- The backend's Go dependencies are current: `go-jose/v4` 4.1.6,
  `klauspost/compress` 1.18.7 (out-of-bounds write in `compress/s2`),
  `xuri/excelize/v2` at the release that fixes the unrecoverable pivot-table
  panic, `buger/jsonparser` 1.1.2, `filippo.io/edwards25519` 1.1.1, and the
  `golang.org/x/{net,crypto,text,sys}` set. The admin tool moves to the same
  `golang.org/x` versions, which raises its `go` directive to 1.26 to match the
  backend.

## [1.4.3] - 2026-10-24

### Added

- Linux package metadata and consistent installer file names.
- The release workflow can publish a GitHub release for a version tag.

## [1.4.2] - 2026-10-24

### Added

- Multi-platform application icons for the Linux, macOS, and Windows builds.

[2.1.1]: https://github.com/James-HoneyBadger/honey-ledger/releases/tag/v2.1.1
[2.0.0]: https://github.com/James-HoneyBadger/honey-ledger/releases/tag/v2.0.0
[1.5.0]: https://github.com/James-HoneyBadger/honey-ledger/releases/tag/v1.5.0
[1.4.3]: https://github.com/James-HoneyBadger/honey-ledger/releases/tag/v1.4.3
[1.4.2]: https://github.com/James-HoneyBadger/honey-ledger/releases/tag/v1.4.2
