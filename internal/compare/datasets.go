package compare

import (
	"sort"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
)

// RowDiff is a row that only one side has.
type RowDiff struct {
	Table  string
	Device string // "" for a table that is not per device
	Row    *model.Row
}

// CellDiff is a cell of a row both sides have, with different values.
type CellDiff struct {
	Table  string
	Device string
	Column string
	ARow   *model.Row
	BRow   *model.Row
	A, B   *model.Cell // nil = empty
	// OrderOnly tells that both sides hold the same items of a list, in a
	// different order.
	OrderOnly bool
}

// Result is the outcome of comparing two sources (spec 12.3).
type Result struct {
	OnlyA []RowDiff
	OnlyB []RowDiff
	Cells []CellDiff
	// Rows counts the rows both sides have, and Same those of them without a
	// different cell.
	Rows, Same int
}

// Equal tells that nothing differs.
func (r *Result) Equal() bool {
	return len(r.OnlyA) == 0 && len(r.OnlyB) == 0 && len(r.Cells) == 0
}

// Datasets compares two sets of design data table by table, row by row (by
// key) and column by column. Host names and interface names are compared as
// they stand: the mapping is put on the old side before (convert.Apply).
//
// A table with a name the import makes up (schema: auto_name) is matched by
// what the rows hold first: two rows that differ in that name only are one
// row. What is left is matched by key (spec 12.3).
//
// The order of the result is fixed: the tables in the order of the
// definitions, the rows in the order of side A, then the rows only B has. An
// empty cell and a missing cell are the same; the free columns of the book are
// not compared.
func Datasets(sc *schema.Set, a, b *model.Dataset) *Result {
	res := &Result{}
	for _, name := range tableNames(sc, a, b) {
		t := sc.Table(name)
		devCol := ""
		if t != nil {
			devCol = t.DeviceColumn()
		}
		ta, tb := table(a, name), table(b, name)
		pair, taken := twins(t, rows(ta), rows(tb))
		for _, ra := range rows(ta) {
			rb := pair[ra]
			if rb == nil {
				if rb = tb.Find(ra.Key); taken[rb] {
					rb = nil
				}
				taken[rb] = true
			}
			if rb == nil {
				res.OnlyA = append(res.OnlyA, RowDiff{name, ra.Get(devCol), ra})
				continue
			}
			res.Rows++
			same := true
			for _, col := range columns(t, ra, rb) {
				if isKey(ra, col) || isKey(rb, col) {
					continue
				}
				ca, cb := filled(ra.Cells[col]), filled(rb.Cells[col])
				if equal(ca, cb) {
					continue
				}
				same = false
				res.Cells = append(res.Cells, CellDiff{
					Table: name, Device: ra.Get(devCol), Column: col,
					ARow: ra, BRow: rb, A: ca, B: cb,
					OrderOnly: sameItems(ca, cb),
				})
			}
			if same {
				res.Same++
			}
		}
		for _, rb := range rows(tb) {
			if !taken[rb] {
				res.OnlyB = append(res.OnlyB, RowDiff{name, rb.Get(devCol), rb})
			}
		}
	}
	return res
}

// twins pairs the rows of a table with a made-up name that hold the same
// besides that name: each row of a with the first such row of b that is free.
// taken tells the rows of b that have their row of a.
func twins(t *schema.Table, as, bs []*model.Row) (map[*model.Row]*model.Row, map[*model.Row]bool) {
	pair, taken := map[*model.Row]*model.Row{}, map[*model.Row]bool{}
	auto := t.AutoName()
	if auto == "" {
		return pair, taken
	}
	for _, ra := range as {
		for _, rb := range bs {
			if taken[rb] || !sameBut(t, ra, rb, auto) {
				continue
			}
			pair[ra], taken[rb] = rb, true
			break
		}
	}
	return pair, taken
}

// sameBut tells that two rows hold the same in every column but one.
func sameBut(t *schema.Table, ra, rb *model.Row, but string) bool {
	for _, col := range columns(t, ra, rb) {
		if col != but && !equal(filled(ra.Cells[col]), filled(rb.Cells[col])) {
			return false
		}
	}
	return true
}

// tableNames gives the tables to compare: those of the definitions in their
// order, then any other table of the two sides by name.
func tableNames(sc *schema.Set, a, b *model.Dataset) []string {
	var out []string
	seen := map[string]bool{}
	for _, t := range sc.Tables {
		out = append(out, t.Name)
		seen[t.Name] = true
	}
	var rest []string
	for _, d := range []*model.Dataset{a, b} {
		if d == nil {
			continue
		}
		for _, n := range d.TableNames() {
			if !seen[n] {
				seen[n] = true
				rest = append(rest, n)
			}
		}
	}
	sort.Strings(rest)
	return append(out, rest...)
}

func table(d *model.Dataset, name string) *model.Table {
	if d == nil {
		return nil
	}
	return d.Tables[name]
}

func rows(t *model.Table) []*model.Row {
	if t == nil {
		return nil
	}
	return t.Rows
}

// columns gives the columns to look at: those of the definition in their
// order, then the other filled columns of the two rows by name.
func columns(t *schema.Table, ra, rb *model.Row) []string {
	var out []string
	seen := map[string]bool{}
	if t != nil {
		for _, c := range t.Columns {
			out = append(out, c.Name)
			seen[c.Name] = true
		}
	}
	var rest []string
	for _, r := range []*model.Row{ra, rb} {
		for name := range r.Cells {
			if !seen[name] {
				seen[name] = true
				rest = append(rest, name)
			}
		}
	}
	sort.Strings(rest)
	return append(out, rest...)
}

func isKey(r *model.Row, col string) bool {
	for _, k := range r.Key {
		if k.Name == col {
			return true
		}
	}
	return false
}

// filled gives the cell, or nil if it holds nothing.
func filled(c *model.Cell) *model.Cell {
	if c == nil {
		return nil
	}
	for _, v := range c.Values {
		if v != "" {
			return c
		}
	}
	return nil
}

func equal(a, b *model.Cell) bool {
	if a == nil || b == nil {
		return a == b
	}
	if len(a.Values) != len(b.Values) {
		return false
	}
	for i := range a.Values {
		if a.Values[i] != b.Values[i] {
			return false
		}
	}
	return true
}

func sameItems(a, b *model.Cell) bool {
	if a == nil || b == nil || len(a.Values) != len(b.Values) {
		return false
	}
	x := append([]string{}, a.Values...)
	y := append([]string{}, b.Values...)
	sort.Strings(x)
	sort.Strings(y)
	for i := range x {
		if x[i] != y[i] {
			return false
		}
	}
	return true
}
