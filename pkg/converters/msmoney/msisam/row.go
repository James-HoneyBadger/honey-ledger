// SPDX-FileCopyrightText: 2020-2026 Honey Badger Universe
// SPDX-License-Identifier: GPL-3.0-or-later

package msisam

// rowLayout describes the parts of a row that are shared by all of its columns.
type rowLayout struct {
	columnCount int
	nullMask    []byte
	// variables holds the offset table of the row's variable length columns.
	// Entry n is where column n starts and entry n+1 is where it ends, both
	// relative to the start of the row.
	variables []int
}

// variableCount returns the number of variable length columns in the row.
func (layout rowLayout) variableCount() int {
	if layout.variables == nil {
		return 0
	}

	return len(layout.variables) - 1
}

// rowLayoutAt reads the layout of the row that starts at the specified offset.
func rowLayoutAt(page []byte, start, size int) (rowLayout, bool) {
	if size < rowHeaderSize+4 || start < 0 || size > len(page)-start {
		return rowLayout{}, false
	}

	end := start + size - 1
	columnCount, ok := readU16(page, start)

	if !ok || columnCount == 0 || columnCount > maxColumnCount {
		return rowLayout{}, false
	}

	maskSize := (columnCount + 7) / 8
	maskOffset := end - maskSize + 1

	if maskOffset <= start+rowHeaderSize {
		return rowLayout{}, false
	}

	layout := rowLayout{
		columnCount: columnCount,
		nullMask:    page[maskOffset : end+1],
	}

	variableCount, ok := readU16(page, maskOffset-2)

	if !ok || variableCount > columnCount {
		return rowLayout{}, false
	}

	if variableCount > 0 {
		// The variable column offsets sit just below the null mask.
		table := maskOffset - 4
		layout.variables = make([]int, variableCount+1)

		for index := range layout.variables {
			offset, ok := readU16(page, table-index*2)

			if !ok {
				return rowLayout{}, false
			}

			layout.variables[index] = offset
		}
	}

	return layout, true
}

// crackRow splits the row that starts at the specified offset into the fields
// of the specified table definition.
func (d *Database) crackRow(definition *Definition, page []byte, start, size int) ([]Field, error) {
	layout, ok := rowLayoutAt(page, start, size)

	if !ok {
		return nil, ErrCorruptRow
	}

	variableCount := layout.variableCount()
	fixedCount := layout.columnCount - variableCount
	fixedFound := 0
	fields := make([]Field, len(definition.Columns))

	for index, column := range definition.Columns {
		field := Field{Column: column}
		byteIndex := column.Number / 8
		bit := byte(1) << (column.Number % 8)

		// A set bit means the column holds a value; a clear bit means it is
		// null, except for a bool, whose value is the bit itself.
		notNull := byteIndex < len(layout.nullMask) && layout.nullMask[byteIndex]&bit != 0
		field.Null = !notNull

		// A fixed column occupies its slot whether or not it holds a value, so
		// the slot counter advances for null columns too.
		fixed := column.Fixed && fixedFound < fixedCount

		if column.Fixed {
			fixedFound++
		}

		switch {
		case field.Null && column.Type != ColumnTypeBool:
		case fixed:
			begin := rowHeaderSize + column.Offset

			if column.Size > 0 && begin+column.Size <= size {
				field.Data = page[start+begin : start+begin+column.Size]
			}
		case !column.Fixed && column.VariableIndex < variableCount:
			begin := layout.variables[column.VariableIndex]
			end := layout.variables[column.VariableIndex+1]

			if begin >= 0 && end >= begin && end <= size {
				field.Data = page[start+begin : start+end]
			}
		}

		fields[index] = field
	}

	return fields, nil
}

// EachRow calls visit once for every live row of the table, in storage order.
// Rows the engine has deleted or left as pointers to a relocated row are
// resolved or skipped, and rows that do not decode are skipped. The fields
// passed to visit alias the database buffer and must not be modified.
func (d *Database) EachRow(definition *Definition, visit func(fields []Field) error) error {
	if definition == nil || definition.Page <= 0 {
		return ErrCorruptDefinition
	}

	for _, page := range d.dataPagesOf(definition.Page) {
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

			fields, err := d.crackRow(definition, rowPage, start, size)

			if err != nil {
				continue
			}

			if err := visit(fields); err != nil {
				return err
			}
		}
	}

	return nil
}

// resolveRow returns the page, offset and length of the live row in the
// specified slot, following a row that was relocated to another page.
func (d *Database) resolveRow(page []byte, slot int) ([]byte, int, int, bool) {
	start, size, ok := rowBounds(page, slot)

	if !ok || size <= 0 {
		return nil, 0, 0, false
	}

	flags, _ := readU16(page, rowStartTable+slot*2)

	if flags&rowDeletedFlag != 0 {
		return nil, 0, 0, false
	}

	if flags&rowLookupFlag == 0 {
		return page, start, size, true
	}

	if size < 4 {
		return nil, 0, 0, false
	}

	reference, ok := readU32(page, start)

	if !ok {
		return nil, 0, 0, false
	}

	index := int(reference >> 8)

	if !d.IsValidPageIndex(index) {
		return nil, 0, 0, false
	}

	target := d.Page(index)
	start, size, ok = rowBounds(target, int(reference&0xFF))

	if !ok || size <= 0 {
		return nil, 0, 0, false
	}

	return target, start, size, true
}

// dataPagesOf returns the pages that hold the rows of the specified table, in
// page order. A data page names the definition page of the table it belongs to.
func (d *Database) dataPagesOf(definition int) []int {
	if d.dataPages == nil {
		pages := make(map[int][]int)

		for page := 1; page < d.pageCount; page++ {
			if d.PageType(page) != PageTypeData {
				continue
			}

			owner, _ := readU32(d.Page(page), pageLinkOffset)
			pages[int(owner)] = append(pages[int(owner)], page)
		}

		d.dataPages = pages
	}

	return d.dataPages[definition]
}
