// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msisam

import (
	"fmt"
	"sort"
	"strings"
)

const (
	// catalogPage is the definition page the database catalog is registered
	// under. The page itself is not readable, but the rows that name the
	// objects point at it.
	catalogPage = 2

	// catalogObjectPageOffset is where a catalog row holds the definition page
	// of the object it names, after the row's column count.
	catalogObjectPageOffset = rowHeaderSize
)

// CatalogEntry is one object the database registers, such as a table.
type CatalogEntry struct {
	// Name is the name the catalog gives the object.
	Name string
	// Page is the definition page of the object.
	Page int
}

// Table is a table definition together with its name.
type Table struct {
	Name       string
	Definition *Definition
}

// Catalog returns the objects the database registers, in storage order. A
// catalog row holds the definition page of its object first and the object's
// name first among the variable length columns.
func (d *Database) Catalog() ([]CatalogEntry, error) {
	entries := make([]CatalogEntry, 0, 128)

	for _, page := range d.dataPagesOf(catalogPage) {
		buffer := d.Page(page)
		count, ok := readU16(buffer, rowCountOffset)

		if !ok || count > PageSize/2 {
			continue
		}

		for index := 0; index < count; index++ {
			rowPage, start, size, ok := d.resolveRow(buffer, index)

			if !ok {
				continue
			}

			layout, ok := rowLayoutAt(rowPage, start, size)

			if !ok || layout.variableCount() == 0 {
				continue
			}

			objectPage, ok := readU32(rowPage, start+catalogObjectPageOffset)

			if !ok || !d.IsValidPageIndex(int(objectPage)) {
				continue
			}

			begin, end := layout.variables[0], layout.variables[1]

			if begin < 0 || end < begin || end > size {
				continue
			}

			name := decodeText(rowPage[start+begin : start+end])

			if name == "" {
				continue
			}

			entries = append(entries, CatalogEntry{Name: name, Page: int(objectPage)})
		}
	}

	return entries, nil
}

// Tables returns every table in the database, sorted by name. Names come from
// the catalog where it holds an entry for the table, and otherwise from the
// table's identity column, which the engine names after the table. A table that
// cannot be named is reported as UNKNOWN_ followed by its definition page.
func (d *Database) Tables() ([]*Table, error) {
	definitions, err := d.Definitions()

	if err != nil {
		return nil, err
	}

	catalog, err := d.Catalog()

	if err != nil {
		return nil, err
	}

	names := make(map[int]string, len(catalog))

	for _, entry := range catalog {
		if _, exists := names[entry.Page]; !exists {
			names[entry.Page] = entry.Name
		}
	}

	tables := make([]*Table, 0, len(definitions))
	unnamed := make([]*Definition, 0, len(definitions))

	for _, definition := range definitions {
		if name, exists := names[definition.Page]; exists {
			tables = append(tables, &Table{Name: name, Definition: definition})
		} else {
			unnamed = append(unnamed, definition)
		}
	}

	// Only trust a derived name when exactly one table claims it.
	claimed := make(map[string]int, len(unnamed))

	for _, definition := range unnamed {
		if name := derivedTableName(definition); name != "" {
			claimed[name]++
		}
	}

	for _, definition := range unnamed {
		name := derivedTableName(definition)

		if name != "" && claimed[name] == 1 {
			tables = append(tables, &Table{Name: name, Definition: definition})

			continue
		}

		tables = append(tables, &Table{Name: fmt.Sprintf("UNKNOWN_%d", definition.Page), Definition: definition})
	}

	sort.Slice(tables, func(left, right int) bool {
		return tables[left].Name < tables[right].Name
	})

	return tables, nil
}

// Table returns the table with the specified name, matching case insensitively.
func (d *Database) Table(name string) (*Table, error) {
	tables, err := d.Tables()

	if err != nil {
		return nil, err
	}

	for _, table := range tables {
		if strings.EqualFold(table.Name, name) {
			return table, nil
		}
	}

	return nil, fmt.Errorf("%w: %s", ErrTableNotFound, name)
}

// derivedTableName returns the name a table's identity column implies. The
// engine names that column after the table, with a leading h.
func derivedTableName(definition *Definition) string {
	if len(definition.Columns) == 0 {
		return ""
	}

	identity := definition.Columns[0].Name

	if len(identity) < 2 || identity[0] != 'h' {
		return ""
	}

	name := identity[1:]

	for _, character := range name {
		if !(character >= 'a' && character <= 'z') && !(character >= 'A' && character <= 'Z') {
			return ""
		}
	}

	return strings.ToUpper(name)
}
