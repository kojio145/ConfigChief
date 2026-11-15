package listimport

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kojio145/configchief/internal/sheet"
)

// Copied from PalaTerm internal/listimport/table.go (see ORIGIN.txt); the parts
// about PalaTerm's device fields are left out.

// Header is one column of the table: its header text (spaces and line
// breaks folded) and its column number (1 = A).
type Header struct {
	Name string `json:"name"`
	Col  int    `json:"col"`
}

// Table is the part of a sheet a rule reads: the headers and the value rows.
type Table struct {
	Sheet   string   `json:"sheet"`
	Headers []Header `json:"headers"`
	Rows    []SrcRow `json:"-"`
	// Warnings say what reading the table could not do as the sheet asks
	// (merged areas too many to fill): shown with the file's own.
	Warnings []string `json:"-"`
}

// SrcRow is one value row: its row number in the sheet (1-based) and its
// cells by column number.
type SrcRow struct {
	Line  int
	Cells map[int]string
}

var reSpaces = regexp.MustCompile(`\s+`)

// foldHeader makes a header comparable: Excel headers often carry a line
// break or a double space ("管理\nIP").
func foldHeader(s string) string { return strings.TrimSpace(reSpaces.ReplaceAllString(s, " ")) }

// PickSheet chooses the rule's sheet: by name when it has one and the book
// has it, else by number (0 = the first).
func PickSheet(b *sheet.Book, r *Rule) (*sheet.Sheet, error) {
	if b == nil || len(b.Sheets) == 0 {
		return nil, fmt.Errorf("ファイルに表がありません")
	}
	if r.Sheet != "" {
		for i := range b.Sheets {
			if strings.EqualFold(strings.TrimSpace(b.Sheets[i].Name), strings.TrimSpace(r.Sheet)) {
				return &b.Sheets[i], nil
			}
		}
		if r.SheetIndex == 0 {
			if len(b.Sheets) == 1 {
				return &b.Sheets[0], nil // a CSV has one sheet, whatever its name
			}
			return nil, fmt.Errorf("シート「%s」がありません", r.Sheet)
		}
	}
	i := r.SheetIndex
	if i < 1 {
		i = 1
	}
	if i > len(b.Sheets) {
		return nil, fmt.Errorf("%d 枚目のシートがありません（%d 枚）", i, len(b.Sheets))
	}
	return &b.Sheets[i-1], nil
}

// cellGrid is a sheet with merged areas filled in (unless the rule asks
// for the top-left cell only), and a warning when some areas were left
// unfilled to stay within sheet.MaxCells.
func cellGrid(s *sheet.Sheet, onlyFirst bool) ([][]string, string) {
	if onlyFirst || len(s.Merges) == 0 {
		return s.Rows, ""
	}
	rows := make([][]string, len(s.Rows))
	// A merged area is filled only where the sheet has cells: up to its
	// last row and its widest row (a merge reaching column ALL of row
	// 100,000 in a small file must not become a hundred million cells),
	// and all the filling together within sheet.MaxCells — counted as the
	// cells the rows actually grow by, not the area's size.
	width, cells := 0, 0
	for i, r := range s.Rows {
		rows[i] = append([]string(nil), r...)
		width = max(width, len(r))
		cells += len(r)
	}
	skipped := 0
	for _, m := range s.Merges {
		if m.R1 < 0 || m.C1 < 0 || m.R1 >= len(rows) || m.C1 >= width || m.R2 < m.R1 || m.C2 < m.C1 {
			continue
		}
		m.R2 = min(m.R2, len(rows)-1)
		m.C2 = min(m.C2, width-1)
		grow := 0
		for r := m.R1; r <= m.R2; r++ {
			grow += max(0, m.C2+1-len(rows[r]))
		}
		if cells+grow > sheet.MaxCells {
			skipped++ // left as it is; a smaller area further on may still fit
			continue
		}
		cells += grow
		v := ""
		if m.C1 < len(rows[m.R1]) {
			v = rows[m.R1][m.C1]
		}
		for r := m.R1; r <= m.R2; r++ {
			for len(rows[r]) <= m.C2 {
				rows[r] = append(rows[r], "")
			}
			for c := m.C1; c <= m.C2; c++ {
				if r == m.R1 && c == m.C1 {
					continue
				}
				rows[r][c] = v
			}
		}
	}
	if skipped > 0 {
		return rows, fmt.Sprintf("シート「%s」の結合セルが多すぎるため、%d 個の結合範囲は左上のセルだけに値があるものとして読みました", s.Name, skipped)
	}
	return rows, ""
}

// FilledRows is the sheet's grid as the rule reads it: merged areas
// filled in unless the rule takes their top-left cell only.
func FilledRows(s *sheet.Sheet, r *Rule) [][]string {
	rows, _ := cellGrid(s, r.MergeOnlyFirst)
	return rows
}

// ReadTable cuts the rule's table out of the sheet: the header row from the
// start column to the last filled header, and every value row with any
// cell filled under those headers.
func ReadTable(s *sheet.Sheet, r *Rule) (*Table, error) {
	rows, warn := cellGrid(s, r.MergeOnlyFirst)
	hr := r.HeaderRow
	if hr < 1 {
		hr = 1
	}
	if hr > len(rows) {
		return nil, fmt.Errorf("見出しの行（%d 行目）がありません。シートは %d 行です", hr, len(rows))
	}
	head := rows[hr-1]
	start := ColumnNumber(r.StartCol)
	if start == 0 {
		for c, v := range head {
			if strings.TrimSpace(v) != "" {
				start = c + 1
				break
			}
		}
		if start == 0 {
			return nil, fmt.Errorf("見出しの行（%d 行目）が空です", hr)
		}
	}
	t := &Table{Sheet: s.Name, Warnings: []string{}}
	if warn != "" {
		t.Warnings = append(t.Warnings, warn)
	}
	last := 0
	for c := start; c <= len(head); c++ {
		if strings.TrimSpace(head[c-1]) != "" {
			last = c
		}
	}
	if last == 0 {
		return nil, fmt.Errorf("見出しの行（%d 行目）の %s 列から右が空です", hr, ColumnLetter(start))
	}
	// The same header twice (a pair's primary and backup blocks, or a
	// repeated group, both with "Router") is two columns: the second and
	// later ones are named "Router (2)"…, so each is chosen — and read —
	// as itself rather than all as the first.
	seen := map[string]int{}
	for c := start; c <= last; c++ {
		name := foldHeader(head[c-1])
		if name != "" {
			seen[name]++
			if n := seen[name]; n > 1 {
				name = fmt.Sprintf("%s (%d)", name, n)
			}
		}
		t.Headers = append(t.Headers, Header{Name: name, Col: c})
	}
	dr := r.DataRow
	if dr == 0 {
		dr = hr + 1
	}
	for i := dr - 1; i < len(rows); i++ {
		row := rows[i]
		cells := map[int]string{}
		any := false
		for _, h := range t.Headers {
			if h.Col-1 < len(row) {
				v := row[h.Col-1]
				cells[h.Col] = v
				any = any || strings.TrimSpace(v) != ""
			}
		}
		if any {
			t.Rows = append(t.Rows, SrcRow{Line: i + 1, Cells: cells})
		}
	}
	return t, nil
}

// column finds a column by header text (the first one with it). Only a
// field without a header text goes by its number: a header renamed or
// moved at the source must show as missing, not quietly read whatever
// column now stands at the old place.
func (t *Table) column(name string, idx int) int {
	if n := foldHeader(name); n != "" {
		for _, h := range t.Headers {
			if h.Name == n {
				return h.Col
			}
		}
		for _, h := range t.Headers {
			if strings.EqualFold(h.Name, n) {
				return h.Col
			}
		}
		return 0
	}
	if idx > 0 {
		for _, h := range t.Headers {
			if h.Col == idx {
				return idx
			}
		}
	}
	return 0
}

var reTemplate = regexp.MustCompile(`\{([^{}]+)\}`)

// sourceOf is a field's value before its steps, the header of the column it
// was read from ("" for a fixed value; for a template the first column it
// names), the columns it names that the table does not have (they read
// empty), and whether anything of it came from the sheet. A template whose
// columns are all empty gives "" (not the bare separators between them).
func (t *Table) sourceOf(f *Field, row SrcRow) (v string, col int, missing []string, read bool) {
	switch f.Source() {
	case "column":
		col = t.column(f.Column, 0)
		if col == 0 {
			missing = append(missing, f.Column)
		} else {
			v = row.Cells[col]
		}
		// Empty: the next column that has the value.
		for _, fb := range f.Fallback {
			if strings.TrimSpace(fb) == "" {
				continue
			}
			c := t.column(fb, 0)
			if c == 0 {
				missing = append(missing, fb)
				continue
			}
			if strings.TrimSpace(v) == "" && strings.TrimSpace(row.Cells[c]) != "" {
				v, col = row.Cells[c], c
			}
		}
		return v, col, missing, strings.TrimSpace(v) != ""
	case "template":
		out := reTemplate.ReplaceAllStringFunc(f.Template, func(m string) string {
			name := m[1 : len(m)-1]
			c := t.column(name, 0)
			if c == 0 {
				missing = append(missing, name)
				return ""
			}
			if col == 0 {
				col = c
			}
			cell := strings.TrimSpace(row.Cells[c])
			read = read || cell != ""
			return cell
		})
		if !read {
			return "", col, missing, false
		}
		return out, col, missing, true
	case "value":
		return f.Value, 0, nil, false
	}
	return "", 0, nil, false
}

// Sources are the headers a field reads: its column and the fallbacks, or
// the ones its template names.
func (f *Field) Sources() []string {
	var out []string
	switch f.Source() {
	case "column":
		out = append(out, f.Column)
		out = append(out, f.Fallback...)
	case "template":
		for _, m := range reTemplate.FindAllStringSubmatch(f.Template, -1) {
			out = append(out, m[1])
		}
	}
	return out
}
