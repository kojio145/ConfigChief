package model

import (
	"sort"
	"strings"
)

// The M0 form of the design data: tables of rows of text cells with their
// origins. The table definitions (types, required columns, checks) join in M1.

// KV is a named value. Slices of KV keep the order they were written in.
type KV struct {
	Name  string
	Value string
}

// Ref names a cell of the design data: the table, the key of the row
// (Row.KeyText) and the column ("" = the row as a whole).
type Ref struct {
	Table  string
	Key    string
	Column string
}

// Origin tells where a value came from.
type Origin struct {
	Kind  string // "config" | "book" | "common" | "default" | "derived" | "customer"
	File  string
	Line  int    // line of the configuration; 0 if none
	Sheet string // sheet of the parameter book
	Addr  string // cell of the sheet ("C5")
}

// Cell is one value of a row. A column filled by `append` is a list.
type Cell struct {
	Values  []string
	List    bool
	Origins []Origin
}

// Text gives the cell as one string; the items of a list are joined by ",".
func (c *Cell) Text() string {
	if c == nil {
		return ""
	}
	return strings.Join(c.Values, ",")
}

// Row is one setting: an interface, a route, a user ...
type Row struct {
	Table   string
	Key     []KV // the columns that tell rows apart; "device" comes first
	Cells   map[string]*Cell
	Origins []Origin // the lines that opened the row
	Extra   []KV     // free columns of the parameter book: heading and value
}

// Get gives the text of a column ("" if empty).
func (r *Row) Get(col string) string { return r.Cells[col].Text() }

// Set puts a value into a column and reports whether it replaced a different
// value.
func (r *Row) Set(col, val string, o Origin) (replaced bool) {
	old := r.Cells[col]
	replaced = old != nil && old.Text() != val
	r.Cells[col] = &Cell{Values: []string{val}, Origins: []Origin{o}}
	return replaced
}

// Append adds a value to a list column.
func (r *Row) Append(col, val string, o Origin) {
	c := r.Cells[col]
	if c == nil {
		c = &Cell{}
		r.Cells[col] = c
	}
	c.List = true
	c.Values = append(c.Values, val)
	c.Origins = append(c.Origins, o)
}

// Columns gives the names of the filled columns: the key first, then the
// rest by name.
func (r *Row) Columns() []string {
	var out []string
	seen := map[string]bool{}
	for _, k := range r.Key {
		out = append(out, k.Name)
		seen[k.Name] = true
	}
	var rest []string
	for name := range r.Cells {
		if !seen[name] {
			rest = append(rest, name)
		}
	}
	sort.Strings(rest)
	return append(out, rest...)
}

// KeyText is the row's identity inside its table.
func (r *Row) KeyText() string { return keyText(r.Key) }

func keyText(key []KV) string {
	var sb strings.Builder
	for i, k := range key {
		if i > 0 {
			sb.WriteByte(',')
		}
		sb.WriteString(k.Name + "=" + k.Value)
	}
	return sb.String()
}

// Table is the rows of one kind of setting, in the order they were made.
type Table struct {
	Name  string
	Rows  []*Row
	index map[string]*Row
}

// Find gives the row with the key, or nil.
func (t *Table) Find(key []KV) *Row {
	if t == nil {
		return nil
	}
	return t.index[keyText(key)]
}

// Dataset is the design data from one source.
type Dataset struct {
	Source string // "old" | "book" | "new" | "customer" | "running"
	Tables map[string]*Table
}

// NewDataset makes an empty dataset.
func NewDataset(source string) *Dataset {
	return &Dataset{Source: source, Tables: map[string]*Table{}}
}

// Open gives the row with the key, making it if it is not there yet. The key
// columns are filled as cells too.
func (d *Dataset) Open(table string, key []KV, o Origin) *Row {
	t := d.Tables[table]
	if t == nil {
		t = &Table{Name: table, index: map[string]*Row{}}
		d.Tables[table] = t
	}
	kt := keyText(key)
	if r := t.index[kt]; r != nil {
		r.Origins = append(r.Origins, o)
		return r
	}
	r := &Row{Table: table, Key: append([]KV{}, key...), Cells: map[string]*Cell{}, Origins: []Origin{o}}
	for _, k := range key {
		r.Cells[k.Name] = &Cell{Values: []string{k.Value}, Origins: []Origin{o}}
	}
	t.Rows = append(t.Rows, r)
	t.index[kt] = r
	return r
}

// Adopt puts a row made elsewhere into the dataset and reports whether it
// went in: a row with the same key that is already there stays, and the new
// one is left out.
func (d *Dataset) Adopt(r *Row) bool {
	t := d.Tables[r.Table]
	if t == nil {
		t = &Table{Name: r.Table, index: map[string]*Row{}}
		d.Tables[r.Table] = t
	}
	kt := r.KeyText()
	if t.index[kt] != nil {
		return false
	}
	t.Rows = append(t.Rows, r)
	t.index[kt] = r
	return true
}

// Remove takes a row out of its table.
func (d *Dataset) Remove(r *Row) {
	t := d.Tables[r.Table]
	if t == nil {
		return
	}
	for i, have := range t.Rows {
		if have == r {
			t.Rows = append(t.Rows[:i:i], t.Rows[i+1:]...)
			delete(t.index, r.KeyText())
			return
		}
	}
}

// Drop removes a table.
func (d *Dataset) Drop(table string) { delete(d.Tables, table) }

// TableNames gives the names of the tables, sorted.
func (d *Dataset) TableNames() []string {
	var out []string
	for name := range d.Tables {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Dump writes the dataset as YAML text: tables by name, rows in the order
// they were made, the key columns first. The same data gives the same text.
func (d *Dataset) Dump() string {
	var sb strings.Builder
	for _, name := range d.TableNames() {
		sb.WriteString(name + ":\n")
		for _, r := range d.Tables[name].Rows {
			first := true
			for _, col := range r.Columns() {
				c := r.Cells[col]
				if c == nil || (len(c.Values) == 0) || (!c.List && c.Values[0] == "") {
					continue
				}
				lead := "    "
				if first {
					lead, first = "  - ", false
				}
				if c.List {
					items := make([]string, len(c.Values))
					for i, v := range c.Values {
						items[i] = quote(v)
					}
					sb.WriteString(lead + col + ": [" + strings.Join(items, ", ") + "]\n")
				} else {
					sb.WriteString(lead + col + ": " + quote(c.Values[0]) + "\n")
				}
			}
			if first {
				sb.WriteString("  - {}\n")
			}
		}
	}
	return sb.String()
}

// quote writes a value as a YAML double-quoted string, so that no value is
// read back as a number, a bool or a date.
func quote(s string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			sb.WriteString(`\"`)
		case '\\':
			sb.WriteString(`\\`)
		case '\n':
			sb.WriteString(`\n`)
		case '\t':
			sb.WriteString(`\t`)
		default:
			sb.WriteRune(r)
		}
	}
	sb.WriteByte('"')
	return sb.String()
}
