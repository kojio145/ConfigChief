package compare

import (
	"sort"
	"strconv"
	"strings"

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
	names := tableNames(sc, a, b)
	// The made-up names that differ between the two sides, by the column
	// they stand in ("table.column") and by device: a row of another table
	// that points at such a name (a ref column) points at the same row on
	// both sides, and is compared as that.
	renamed := map[string]map[string]string{}
	for _, name := range names {
		t := sc.Table(name)
		if t == nil || t.AutoName() == "" {
			continue
		}
		auto, devCol := t.AutoName(), t.DeviceColumn()
		pair, _ := twins(t, rows(table(a, name)), rows(table(b, name)))
		for ra, rb := range pair {
			if ra.Get(auto) == rb.Get(auto) {
				continue
			}
			k := name + "." + auto
			if renamed[k] == nil {
				renamed[k] = map[string]string{}
			}
			renamed[k][ra.Get(devCol)+"\x00"+ra.Get(auto)] = rb.Get(auto)
		}
	}
	for _, name := range names {
		t := sc.Table(name)
		devCol := ""
		if t != nil {
			devCol = t.DeviceColumn()
		}
		ta, tb := table(a, name), table(b, name)
		as, bs := rows(ta), rows(tb)
		// seen is a row as it is compared: a row of side A as side B names
		// what it points at, and on both sides a number that only orders
		// the rows (schema: order_only) as the place it gives the row.
		seen := map[*model.Row]*model.Row{}
		if t != nil && len(renamed) > 0 {
			for _, ra := range as {
				if v := pointed(t, ra, devCol, renamed); v != nil {
					seen[ra] = v
				}
			}
		}
		view := func(r *model.Row) *model.Row {
			if v := seen[r]; v != nil {
				return v
			}
			return r
		}
		vas := make([]*model.Row, len(as))
		for i, ra := range as {
			vas[i] = view(ra)
		}
		vbs := bs
		byKey := func(key []model.KV) *model.Row { return tb.Find(key) }
		if col := orderOnly(t); col != "" {
			for i, v := range placed(t, vas, col) {
				seen[as[i]], vas[i] = v, v
			}
			vbs = placed(t, bs, col)
			index := map[string]*model.Row{}
			for _, v := range vbs {
				index[v.KeyText()] = v
			}
			byKey = func(key []model.KV) *model.Row { return index[(&model.Row{Key: key}).KeyText()] }
		}
		orig := map[*model.Row]*model.Row{}
		for i, v := range vbs {
			orig[v] = bs[i]
		}
		vpair, taken := twins(t, vas, vbs)
		for _, ra := range as {
			va := view(ra)
			vb := vpair[va]
			if vb == nil {
				if vb = byKey(va.Key); taken[vb] {
					vb = nil
				}
				taken[vb] = true
			}
			rb := orig[vb]
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
				if equal(filled(va.Cells[col]), cb) {
					continue
				}
				// A value filled in as the default on one side and nothing on
				// the other: neither side wrote it, so they do not differ.
				if (ca == nil && isDefault(cb)) || (cb == nil && isDefault(ca)) {
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
		for i, rb := range bs {
			if !taken[vbs[i]] {
				res.OnlyB = append(res.OnlyB, RowDiff{name, rb.Get(devCol), rb})
			}
		}
	}
	return res
}

// orderOnly gives the key column of a table whose number only orders the
// rows ("" = none).
func orderOnly(t *schema.Table) string {
	if t == nil {
		return ""
	}
	for _, k := range t.Key {
		if c := t.Column(k); c != nil && c.OrderOnly {
			return k
		}
	}
	return ""
}

// placed gives copies of the rows in which the column that only orders them
// holds the place of the row among the rows with the same other key columns
// (1, 2, 3 and so on, by the number): 15 and 25 on one side are 20 and 30 on
// the other.
func placed(t *schema.Table, rs []*model.Row, col string) []*model.Row {
	group := func(r *model.Row) string {
		var sb strings.Builder
		for _, k := range r.Key {
			if k.Name != col {
				sb.WriteString(k.Value)
				sb.WriteByte(0)
			}
		}
		return sb.String()
	}
	byGroup := map[string][]*model.Row{}
	for _, r := range rs {
		g := group(r)
		byGroup[g] = append(byGroup[g], r)
	}
	place := map[*model.Row]int{}
	for _, g := range byGroup {
		sort.SliceStable(g, func(i, j int) bool {
			x, e1 := strconv.Atoi(g[i].Get(col))
			y, e2 := strconv.Atoi(g[j].Get(col))
			if e1 == nil && e2 == nil {
				return x < y
			}
			return g[i].Get(col) < g[j].Get(col)
		})
		for i, r := range g {
			place[r] = i + 1
		}
	}
	out := make([]*model.Row, len(rs))
	for i, r := range rs {
		n := &model.Row{Table: r.Table, Origins: r.Origins, Extra: r.Extra, Cells: map[string]*model.Cell{}}
		for name, c := range r.Cells {
			n.Cells[name] = c
		}
		n.Cells[col] = &model.Cell{Values: []string{strconv.Itoa(place[r])}}
		for _, k := range r.Key {
			n.Key = append(n.Key, model.KV{Name: k.Name, Value: n.Get(k.Name)})
		}
		out[i] = n
	}
	return out
}

// pointed gives a copy of a row of side A in which the made-up names it
// points at (ref columns) are those of side B, or nil if it points at none
// that differs.
func pointed(t *schema.Table, r *model.Row, devCol string, renamed map[string]map[string]string) *model.Row {
	var out *model.Row
	for _, c := range t.Columns {
		m := renamed[c.Ref]
		cell := r.Cells[c.Name]
		if c.Ref == "" || m == nil || cell == nil {
			continue
		}
		for i, v := range cell.Values {
			to, ok := m[r.Get(devCol)+"\x00"+v]
			if !ok {
				continue
			}
			if out == nil {
				out = &model.Row{Table: r.Table, Origins: r.Origins, Cells: map[string]*model.Cell{}}
				for name, c := range r.Cells {
					out.Cells[name] = c
				}
			}
			if out.Cells[c.Name] == cell {
				n := *cell
				n.Values = append([]string{}, cell.Values...)
				out.Cells[c.Name] = &n
			}
			out.Cells[c.Name].Values[i] = to
		}
	}
	if out == nil {
		return nil
	}
	for _, k := range r.Key {
		out.Key = append(out.Key, model.KV{Name: k.Name, Value: out.Get(k.Name)})
	}
	return out
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
// order (but those it keeps out of a comparison: compare: false), then the other filled columns of the two rows by name.
func columns(t *schema.Table, ra, rb *model.Row) []string {
	var out []string
	seen := map[string]bool{}
	if t != nil {
		for _, c := range t.Columns {
			seen[c.Name] = true
			if c.Compare {
				out = append(out, c.Name)
			}
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

// isDefault tells that the cell holds a default filled in, not a value
// that was written.
func isDefault(c *model.Cell) bool {
	return c != nil && len(c.Origins) > 0 && c.Origins[0].Kind == "default"
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
