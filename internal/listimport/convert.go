package listimport

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
)

// Problem is something of a list that could not be taken (S-504): a value
// that does not fit its column, a line without its key, a second value for a
// cell that has one.
type Problem struct {
	File  string // the list, from the case folder
	Sheet string
	Addr  string // the cell ("C5"), or the line ("5 行目") when no one cell is at fault
	Line  int
	// Table and Column name the place in the design data ("" = the row), and
	// Device the device when the line told it.
	Table, Column, Device string
	Text                  string
}

// Made is one row of the design data a line gave, for the preview of the
// rule editor.
type Made struct {
	Line   int
	Target int // the place of the target in the rule, from 0
	Row    *model.Row
}

// Result is what running a rule over a table gave.
type Result struct {
	Sheet string
	// Lines counts the value lines of the table, Excluded those the rule
	// leaves out, and Rows the rows of the design data they gave (a row two
	// lines speak of counts once).
	Lines, Excluded, Rows int
	// Missing names the columns the rule refers to that the table does not
	// have: they read empty.
	Missing  []string
	Problems []Problem
	Warnings []string
	Made     []Made
}

func addr(col, line int) string {
	if col < 1 {
		return strconv.Itoa(line) + " 行目"
	}
	return ColumnLetter(col) + strconv.Itoa(line)
}

// items gives the values of a cell: one for a plain column, the parts
// between commas and line breaks for a list column.
func items(c *schema.Column, raw string) []string {
	if !c.List {
		return []string{raw}
	}
	return strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' || r == '、' })
}

type value struct {
	field *Field
	col   *schema.Column
	vals  []string // normalized; nil = empty
	src   int      // the column of the sheet it was read from, 0 if none
	// read tells that the value came from the sheet (not a fixed value, not
	// the default).
	read bool
}

// Convert runs a rule over the table of a list and puts the rows it gives
// into the design data (spec 6.5.2). file names the list in the origins and
// the problems. The rule was validated before.
func Convert(t *Table, r *Rule, sc *schema.Set, file string, into *model.Dataset) *Result {
	res := &Result{Sheet: t.Sheet, Warnings: append([]string{}, t.Warnings...)}
	missing := map[string]bool{}
	problem := func(line, col int, table, column, device, format string, a ...any) {
		res.Problems = append(res.Problems, Problem{File: file, Sheet: t.Sheet, Addr: addr(col, line), Line: line,
			Table: table, Column: column, Device: device, Text: fmt.Sprintf(format, a...)})
	}
	for _, src := range t.Rows {
		res.Lines++
		if excluded(t, r, src, missing) {
			res.Excluded++
			continue
		}
		for ti := range r.Rows {
			tg := &r.Rows[ti]
			tab := sc.Table(tg.Table)
			if tab == nil {
				continue
			}
			devCol := tab.DeviceColumn()
			var vals []value
			byName := map[string]*value{}
			for fi := range tg.Columns {
				f := &tg.Columns[fi]
				c := tab.Column(f.To)
				if c == nil {
					continue
				}
				raw, col, miss, read := t.sourceOf(f, src)
				for _, m := range miss {
					missing[m] = true
				}
				v := ApplySteps(raw, f.Steps)
				if strings.TrimSpace(v) == "" {
					v, read = "", false
					if f.Default != "" {
						v = f.Default
					}
				}
				val := value{field: f, col: c, src: col, read: read}
				for _, item := range items(c, v) {
					n, err := c.Normalize(item)
					if err != nil {
						val.vals, val.read = nil, false
						problem(src.Line, col, tab.Name, c.Name, "", "%s の列「%s」: %v", tab.Title.JA, c.Title.JA, err)
						break
					}
					if n != "" {
						val.vals = append(val.vals, n)
					}
				}
				vals = append(vals, val)
			}
			for i := range vals {
				byName[vals[i].col.Name] = &vals[i]
			}
			var key []model.KV
			var lacking []string
			for _, k := range tab.Key {
				v := byName[k]
				if v == nil || len(v.vals) == 0 {
					lacking = append(lacking, tab.Column(k).Title.JA)
					continue
				}
				key = append(key, model.KV{Name: k, Value: v.vals[0]})
			}
			device := ""
			if v := byName[devCol]; v != nil && len(v.vals) > 0 {
				device = v.vals[0]
			}
			// What the line says besides the key, and whether the target has
			// such columns at all.
			said, others, first := false, false, 0
			for i := range vals {
				if tab.IsKey(vals[i].col.Name) {
					continue
				}
				others = true
				if vals[i].read && len(vals[i].vals) > 0 {
					if !said {
						first = vals[i].src
					}
					said = true
				}
			}
			if len(lacking) > 0 {
				if said {
					problem(src.Line, first, tab.Name, "", device, "%s の行を作れません。キーの列「%s」が空です", tab.Title.JA, strings.Join(lacking, "」「"))
				}
				continue
			}
			if others && !said && tab.Name != "devices" {
				continue // nothing but the key: no empty row
			}
			o := model.Origin{Kind: "customer", File: file, Line: src.Line, Sheet: t.Sheet, Addr: addr(0, src.Line)}
			have := into.Tables[tab.Name].Find(key)
			row := into.Open(tab.Name, key, o)
			if have == nil {
				res.Rows++
			}
			res.Made = append(res.Made, Made{Line: src.Line, Target: ti, Row: row})
			for i := range vals {
				v := &vals[i]
				if tab.IsKey(v.col.Name) || len(v.vals) == 0 {
					continue
				}
				co := o
				co.Addr = addr(v.src, src.Line)
				old := row.Cells[v.col.Name]
				if old != nil && old.Text() != "" {
					if old.Text() != strings.Join(v.vals, ",") {
						at := ""
						if len(old.Origins) > 0 {
							at = "（" + where(old.Origins[0]) + "）"
						}
						problem(src.Line, v.src, tab.Name, v.col.Name, device,
							"%s の列「%s」: 同じ行に、先に別の値があります%s。先の値を使います", tab.Title.JA, v.col.Title.JA, at)
					}
					continue
				}
				cell := &model.Cell{Values: v.vals, List: v.col.List}
				for range v.vals {
					cell.Origins = append(cell.Origins, co)
				}
				row.Cells[v.col.Name] = cell
			}
		}
	}
	for m := range missing {
		res.Missing = append(res.Missing, m)
	}
	sort.Strings(res.Missing)
	return res
}

// where tells the place of a value of a list.
func where(o model.Origin) string {
	s := o.File
	if o.Sheet != "" {
		s += " のシート「" + o.Sheet + "」"
	}
	return s + "の " + o.Addr
}

// excluded tells whether a line meets an exclusion condition. A condition on
// a column the table does not have is not judged: the empty value it would
// read says nothing, and "状態 が空" would leave out every line of a list
// whose 状態 column was renamed. The column is reported missing instead.
func excluded(t *Table, r *Rule, src SrcRow, missing map[string]bool) bool {
	for _, c := range r.Exclude {
		col := t.column(c.Column, 0)
		if col == 0 {
			missing[c.Column] = true
			continue
		}
		if condMet(c, src.Cells[col]) {
			return true
		}
	}
	return false
}

func condMet(c Cond, v string) bool {
	v = strings.ToLower(strings.TrimSpace(v))
	w := strings.ToLower(strings.TrimSpace(c.Value))
	switch c.Op {
	case "empty":
		return v == ""
	case "notempty":
		return v != ""
	case "contains":
		return w != "" && strings.Contains(v, w)
	case "notcontains":
		return w != "" && !strings.Contains(v, w)
	case "equals":
		return v == w
	case "notequals":
		return v != w
	}
	return false
}

// Targeted gives the columns of each table the rules fill: what a comparison
// with the lists looks at.
func Targeted(rules []*Rule) map[string]map[string]bool {
	out := map[string]map[string]bool{}
	for _, r := range rules {
		for _, tg := range r.Rows {
			if out[tg.Table] == nil {
				out[tg.Table] = map[string]bool{}
			}
			for _, f := range tg.Columns {
				out[tg.Table][f.To] = true
			}
		}
	}
	return out
}
