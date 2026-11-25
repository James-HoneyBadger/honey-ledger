// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msisam

import "fmt"

// maxDiagnosticSamples bounds how many per-kind samples a Diagnostics keeps,
// so a heavily damaged file cannot make the report grow without limit.
const maxDiagnosticSamples = 16

// Diagnostics records the problems found while decoding a database. A Database
// reports into its Diagnostics only when one is attached with SetDiagnostics;
// without one it stays silent, exactly as before the diagnostics existed.
//
// Counts are authoritative; Samples are a bounded selection for humans. A
// Diagnostics is not safe for concurrent use, like the Database it belongs to.
type Diagnostics struct {
	// CorruptRows counts rows that could not be decoded into fields.
	CorruptRows int
	// UnresolvedRows counts row slots that could not be resolved to live rows:
	// out-of-range slot tables or relocated rows that point nowhere.
	UnresolvedRows int
	// CorruptDefinitions counts table definitions that could not be decoded.
	CorruptDefinitions int
	// UnknownCatalogEntries counts catalog rows that could not be read as an
	// object name and page reference.
	UnknownCatalogEntries int

	RowSamples         []RowDiagnostic
	DefinitionSamples  []DefinitionDiagnostic
	CatalogEntrySample []CatalogDiagnostic

	// definitionPagesRecorded keeps one corrupt definition counted once,
	// however often its page is decoded.
	definitionPagesRecorded map[int]bool
}

// RowDiagnostic describes one row that failed to decode.
type RowDiagnostic struct {
	// Definition is the page of the table definition the row belongs to.
	Definition int
	// Page is the page that holds the row.
	Page int
	// Slot is the row's slot in the page's row table.
	Slot int
	// Err describes the failure.
	Err string
}

// DefinitionDiagnostic describes one table definition that failed to decode.
type DefinitionDiagnostic struct {
	// Page is the definition page.
	Page int
	// Err describes the failure.
	Err string
}

// CatalogDiagnostic describes one catalog entry that could not be read.
type CatalogDiagnostic struct {
	// Page is the page that holds the entry.
	Page int
	// Slot is the entry's slot in the page's row table.
	Slot int
	// Err describes the failure.
	Err string
}

// Summary returns a one line human-readable description of the recorded
// problems, or an empty string when none were recorded.
func (d *Diagnostics) Summary() string {
	if d == nil || d.IsEmpty() {
		return ""
	}

	return fmt.Sprintf("%d rows undecodable, %d rows unresolved, %d definitions undecodable, %d catalog entries unreadable",
		d.CorruptRows, d.UnresolvedRows, d.CorruptDefinitions, d.UnknownCatalogEntries)
}

// IsEmpty reports whether no problem was recorded.
func (d *Diagnostics) IsEmpty() bool {
	return d == nil || (d.CorruptRows == 0 && d.UnresolvedRows == 0 && d.CorruptDefinitions == 0 && d.UnknownCatalogEntries == 0)
}

func (d *Diagnostics) recordRow(definition, page, slot int, err error) {
	d.CorruptRows++

	if len(d.RowSamples) < maxDiagnosticSamples {
		d.RowSamples = append(d.RowSamples, RowDiagnostic{Definition: definition, Page: page, Slot: slot, Err: err.Error()})
	}
}

func (d *Diagnostics) recordUnresolvedRow(definition, page, slot int) {
	d.UnresolvedRows++

	if len(d.RowSamples) < maxDiagnosticSamples {
		d.RowSamples = append(d.RowSamples, RowDiagnostic{Definition: definition, Page: page, Slot: slot, Err: "row slot cannot be resolved"})
	}
}

func (d *Diagnostics) recordDefinition(page int, err error) {
	// The same definition is decoded once per table lookup; one corrupt page
	// must count once, however often it is read.
	if d.definitionPagesRecorded == nil {
		d.definitionPagesRecorded = make(map[int]bool)
	}

	if d.definitionPagesRecorded[page] {
		return
	}

	d.definitionPagesRecorded[page] = true
	d.CorruptDefinitions++

	if len(d.DefinitionSamples) < maxDiagnosticSamples {
		d.DefinitionSamples = append(d.DefinitionSamples, DefinitionDiagnostic{Page: page, Err: err.Error()})
	}
}

func (d *Diagnostics) recordCatalogEntry(page, slot int, err error) {
	d.UnknownCatalogEntries++

	if len(d.CatalogEntrySample) < maxDiagnosticSamples {
		d.CatalogEntrySample = append(d.CatalogEntrySample, CatalogDiagnostic{Page: page, Slot: slot, Err: err.Error()})
	}
}
