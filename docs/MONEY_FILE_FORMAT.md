# Microsoft Money Data File Format

Microsoft Money keeps its data in a single-file database whose header names the
MSISAM engine instead of the Access engine. That rename is why every Jet and
Access driver refuses to open a Money file, even though the page layout below is
Jet 4 with a few Money-specific structures.

Everything in this document was verified against a real Microsoft Money Sunset
database. The reader that implements it lives in
[`pkg/converters/msmoney/msisam`](../pkg/converters/msmoney/msisam).

## Containers

| Extension | Contents |
|-----------|----------|
| `.mny` | The database itself |
| `.mbf` | A Microsoft Cabinet (`MSCF`, usually LZX compressed) holding one `.mny` member |

A backup can be unpacked with `cabextract <file>.mbf`, which writes the database
under a temporary name.

## Header page

Page 0 is 4096 bytes and starts with:

| Offset | Size | Meaning |
|--------|------|---------|
| `0x00` | 1 | Page type, `0x00` for the header page |
| `0x04` | 15 | Engine name, `MSISAM Database` |
| `0x14` | 1 | Engine version, `1` for the Jet 4 layout |
| `0x18` | 128 | Scrambled block, described below |

The 128 bytes from `0x18` are scrambled with RC4 keyed by `C7 DA 39 6B`, whether
or not the database has a password, so a reader has to unscramble them before it
can read:

| Offset | Size | Meaning |
|--------|------|---------|
| `0x3C` | 2 | Windows code page of non-unicode data, typically 1252 |
| `0x3E` | 4 | Database key |
| `0x6E` | 2 | Windows language identifier, typically 1033 |

A non-zero database key means a password protects the database and RC4 scrambles
every page with a key derived from it. The reader reports such a file as
`ErrEncryptedDatabase` rather than reading nonsense.

A database may end with a partial page, because Money appends trailing data, so
only whole pages are addressable.

## Page types

The first byte of a page holds its type:

| Value | Type | Contents |
|-------|------|----------|
| `0x00` | Header | Page 0 |
| `0x01` | Data | Table rows |
| `0x02` | Definition | Table definitions |
| `0x03` | Index | Index pages |
| `0x04` | Leaf | The overflow of a long value |
| `0x05` | Usage map | Which pages a table owns |

Offset `0x04` of a page holds a page number whose meaning depends on the type. A
data page names the definition page of the table it belongs to, and a definition
or leaf page names the page that continues it.

## Table definitions

A definition page begins with:

| Offset | Size | Meaning |
|--------|------|---------|
| `0x04` | 4 | Page that continues the definition, or 0 |
| `0x10` | 4 | Row count |
| `0x2B` | 2 | Number of variable length columns |
| `0x2D` | 2 | Number of columns |
| `0x33` | 4 | Number of real indexes |
| `0x38` | 2 | First data page |
| `0x3F` | | Start of the column entries |

The column entries are 25 bytes each, starting at `0x3F + realIndexes × 12`:

| Offset | Size | Meaning |
|--------|------|---------|
| `0x00` | 1 | Column type |
| `0x05` | 2 | Column number, also its bit in the row's null mask |
| `0x07` | 2 | Index into the row's variable column offset table |
| `0x0B` | 1 | Scale |
| `0x0C` | 1 | Precision |
| `0x0F` | 1 | Flags; bit `0x01` marks a fixed width column |
| `0x15` | 2 | Byte offset of a fixed width column within the row |
| `0x17` | 2 | Width of a fixed column, or maximum length of a variable column |

The names follow the entries: a little endian 16 bit length, then that many bytes
of UTF-16.

A definition larger than one page continues on the page named at `0x04`. A
continuation page repeats the eight byte page header, so its content starts at
offset 8. Continuation pages also start with `0x02`, so a reader has to collect
the referenced pages first to tell a table from the rest of another table's
definition.

### Column types

| Value | Type | Stored as |
|-------|------|-----------|
| `0x01` | Bool | The null mask bit itself |
| `0x02` | Byte | 1 byte |
| `0x03` | Int | 2 byte little endian |
| `0x04` | LongInt | 4 byte little endian |
| `0x05` | Money | 8 byte little endian, scaled by 10 000 |
| `0x06` | Float | 4 byte IEEE 754 |
| `0x07` | Double | 8 byte IEEE 754 |
| `0x08` | DateTime | 8 byte IEEE 754, days since 1899-12-30 |
| `0x09` | Binary | Raw bytes |
| `0x0A` | Text | UTF-16, may be compressed |
| `0x0B` | OLE | A long value, returned as raw bytes |
| `0x0C` | Memo | A long value, returned as text |
| `0x0F` | GUID | 16 bytes, the first three fields little endian |
| `0x10` | Numeric | A sign byte and a 16 byte little endian magnitude |
| `0x12` | Complex | Unsupported |

## Rows

A data page holds the number of rows at offset `0x0C`, then the start offset of
each row at `0x0E`. Rows fill the page from the end backwards, so row 0 starts
highest and every entry is the end of the row above it.

The high bits of a start offset are flags:

| Bit | Meaning |
|-----|---------|
| `0x8000` | The row is deleted |
| `0x4000` | The slot holds a 4 byte page and row pointer to the real row |

A row is laid out as:

1. the column count, 2 bytes;
2. the fixed width columns, each at the offset its definition gives;
3. the variable length columns, packed in order;
4. then, growing downwards from the end of the row, the offset table of the
   variable columns (one more entry than there are variable columns), the count
   of variable columns, and the not-null mask.

A bit set in the not-null mask means the column holds a value. The exceptions are
a bool, whose value is the bit itself, and a column the row does not carry at
all, which reads as null.

## Text

Text is UTF-16, but Money compresses it: while the engine is in compressed mode
each byte is one character with a zero high byte, and a `0x00` byte toggles
between compressed and plain UTF-16. A string of plain ASCII therefore occupies
half the space, and a string with any non-latin character contains the toggle.

## Long values

Memo and OLE columns store a 12 byte header instead of the value:

| Offset | Size | Meaning |
|--------|------|---------|
| `0x00` | 4 | Length, with flags in the high bits |
| `0x04` | 4 | Page and row of the first leaf page |

When bit 31 of the length is set the value is inline and follows the header. When
bit 30 is set it occupies a single leaf page. Otherwise it is spread over a chain
of leaf pages, each of which starts with the page and row of the next.

## The catalog

The catalog is the table registered under definition page 2, although that page
itself is not readable. Its rows hold the definition page of an object first and
the object's name first among the variable columns. In a real Money database the
catalog only names the objects from the end of the alphabet; the rest are named
after their identity column, which the engine calls `h` followed by the lower
case table name, so the table holding `hacct` is `ACCT`.

Money names its core tables `ACCT`, `CAT`, `CRNC`, `TRN` and `TRN_SPLIT`. A
transaction is in `TRN`, a split of a transaction is in `TRN_SPLIT`, categories
are in `CAT`, accounts in `ACCT` and currencies in `CRNC`.

## Reading a ledger out of a data file

The importer in [`pkg/converters/msmoney`](../pkg/converters/msmoney) turns the
tables into transactions. Four rules matter:

- **A transfer is stored twice**, once in each account, with `hacctLink` naming
  the other account. Only the row that takes the money out becomes a transfer, so
  the pair is imported once.
- **A transaction with splits is stored as a parent row that holds the total**
  plus one row per split. `TRN_SPLIT` links a split to its parent, so the parent
  is dropped and every split becomes a transaction under the parent's date and
  account.
- **An investment row names a security** in `hsec`. It records shares rather than
  money, so it is skipped; the cash side of the activity still imports.
- **Categories form a tree.** The root is `INCOME` or `EXPENSE`, which is a
  direction rather than a category, so a category's grouping is its parent and a
  transaction filed straight against a root stays uncategorized.

An amount is stored with four decimals and imports with two, rounded away from
zero. Dates count days from 1899-12-30, and a date Money never set falls far
outside the range a transaction can hold.
