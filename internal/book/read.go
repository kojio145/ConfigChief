package book

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
	"github.com/kojio145/configchief/internal/sheet"
)

// Problem is something in the book a person has to look at.
type Problem struct {
	Sheet string
	Addr  string // "C5"; "" = the whole sheet
	Text  string
	// Kind is the check a problem belongs to ("V-105", ...); "" for the ones
	// that are told as they are.
	Kind string
	// Device is what the "device" cell of the row says (a host name, "*" or
	// "@group"); "" if the problem is not about a row of a device.
	Device string
}

func (p Problem) String() string {
	if p.Addr == "" {
		return fmt.Sprintf("シート「%s」: %s", p.Sheet, p.Text)
	}
	return fmt.Sprintf("シート「%s」の %s: %s", p.Sheet, p.Addr, p.Text)
}

// SheetInfo tells where the columns of a sheet are, for adding rows to it.
type SheetInfo struct {
	Name  string         // the name of the sheet in the file
	Cols  map[string]int // internal name of a column → its place, from 0
	Free  []string       // the headings the definition does not have, as written
	Width int            // the cells of the header row, up to the last filled one
	Last  int            // the last row with a value, from 1 (the header is row 1)
}

// Book is a parameter book that was read.
type Book struct {
	File    string
	Data    *model.Dataset
	Devices []DeviceMap
	Ifs     []IfMap
	Common  []Common
	// Sheets is keyed by the internal name of the table; the sheets that are
	// not tables are "@devices", "@ifs" and "@common".
	Sheets   map[string]*SheetInfo
	Memo     []string // the sheets the tool does not know (left alone)
	Problems []Problem
}

func (b *Book) problemf(sheet, addr, format string, a ...any) {
	b.Problems = append(b.Problems, Problem{Sheet: sheet, Addr: addr, Text: fmt.Sprintf(format, a...)})
}

func addr(col, row int) string {
	s, _ := excelize.CoordinatesToCellName(col+1, row+1)
	return s
}

// Read reads a parameter book (spec 6). file is the name that goes into the
// origin of the values. A value that does not fit its column is kept as
// written and listed in Problems.
func Read(data []byte, file string, sc *schema.Set) (*Book, error) {
	src, err := sheet.Read(data)
	if err != nil {
		return nil, fmt.Errorf("%s を読めません: %v", file, err)
	}
	b := &Book{File: file, Data: model.NewDataset("book"), Sheets: map[string]*SheetInfo{}}

	tables := map[string]*schema.Table{}
	for _, t := range sc.Tables {
		for _, n := range []string{t.Title.JA, t.Title.EN, t.Name} {
			if n != "" {
				tables[foldName(n)] = t
			}
		}
	}
	others := map[string]*mgmt{}
	for _, m := range mgmtSheets {
		others[foldName(m.title.JA)] = m
		others[foldName(m.title.EN)] = m
	}

	for i := range src.Sheets {
		s := &src.Sheets[i]
		name := foldName(s.Name)
		switch {
		case others[name] != nil:
			m := others[name]
			if b.Sheets[m.id] != nil {
				b.problemf(s.Name, "", "シート「%s」と同じ内容のシートです。このシートは読みません", b.Sheets[m.id].Name)
				continue
			}
			b.readMgmt(s, m)
		case tables[name] != nil:
			t := tables[name]
			if b.Sheets[t.Name] != nil {
				b.problemf(s.Name, "", "シート「%s」と同じ表のシートです。このシートは読みません", b.Sheets[t.Name].Name)
				continue
			}
			b.readTable(s, t)
		default:
			b.Memo = append(b.Memo, s.Name)
		}
	}
	return b, nil
}

// header finds the columns of a sheet by the text of the header row. names
// gives the internal name for a folded heading.
func (b *Book) header(s *sheet.Sheet, names map[string]string) (info *SheetInfo, free map[int]string) {
	info = &SheetInfo{Name: s.Name, Cols: map[string]int{}}
	free = map[int]string{}
	if len(s.Rows) == 0 {
		return info, free
	}
	for c, h := range s.Rows[0] {
		h = strings.TrimSpace(h)
		if h == "" {
			continue
		}
		info.Width = c + 1
		name, ok := names[foldName(h)]
		if !ok {
			free[c] = h
			if _, dup := info.Cols[freeCol(h)]; !dup {
				info.Cols[freeCol(h)] = c
				info.Free = append(info.Free, h)
			}
			continue
		}
		if first, dup := info.Cols[name]; dup {
			b.problemf(s.Name, addr(c, 0), "見出し「%s」が 2 つあります。%s の列を使います", h, addr(first, 0))
			continue
		}
		info.Cols[name] = c
	}
	info.Last = 1
	for r := 1; r < len(s.Rows); r++ {
		if !blank(s.Rows[r]) {
			info.Last = r + 1
		}
	}
	return info, free
}

// freeCol is the name in SheetInfo.Cols of a column the definition does not
// have, by its heading.
func freeCol(heading string) string { return "free:" + foldName(heading) }

func blank(row []string) bool {
	for _, v := range row {
		if strings.TrimSpace(v) != "" {
			return false
		}
	}
	return true
}

func at(row []string, c int) string {
	if c < 0 || c >= len(row) {
		return ""
	}
	return row[c]
}

func (b *Book) readTable(s *sheet.Sheet, t *schema.Table) {
	names := map[string]string{}
	for _, c := range t.Columns {
		for _, n := range []string{c.Title.JA, c.Title.EN, c.Name} {
			if n != "" {
				names[foldName(n)] = c.Name
			}
		}
	}
	info, free := b.header(s, names)
	b.Sheets[t.Name] = info

	for r := 1; r < len(s.Rows); r++ {
		line := s.Rows[r]
		if blank(line) {
			continue
		}
		row := &model.Row{Table: t.Name, Cells: map[string]*model.Cell{}}
		row.Origins = []model.Origin{{Kind: "book", File: b.File, Sheet: s.Name, Addr: addr(0, r)}}
		for _, c := range t.Columns {
			ci, ok := info.Cols[c.Name]
			if !ok {
				continue
			}
			raw := at(line, ci)
			if strings.TrimSpace(raw) == "" {
				continue
			}
			cell := &model.Cell{List: c.List}
			cell.Origins = []model.Origin{{Kind: "book", File: b.File, Sheet: s.Name, Addr: addr(ci, r)}}
			for _, item := range split(c, raw) {
				v, err := c.Normalize(item)
				if err != nil {
					// The checks of the values tell about it again, by device (V-102 to V-104).
					b.Problems = append(b.Problems, Problem{Sheet: s.Name, Addr: addr(ci, r), Kind: "V-102", Text: err.Error()})
					v = strings.TrimSpace(item)
				}
				if v != "" {
					cell.Values = append(cell.Values, v)
				}
			}
			if len(cell.Values) > 0 {
				row.Cells[c.Name] = cell
			}
		}
		for _, k := range t.Key {
			row.Key = append(row.Key, model.KV{Name: k, Value: row.Get(k)})
		}
		for c := 0; c < len(line); c++ {
			if h, ok := free[c]; ok && strings.TrimSpace(line[c]) != "" {
				row.Extra = append(row.Extra, model.KV{Name: h, Value: strings.TrimSpace(line[c])})
			}
		}
		if !b.Data.Adopt(row) {
			first := b.Data.Tables[t.Name].Find(row.Key)
			b.Problems = append(b.Problems, Problem{Sheet: s.Name, Addr: addr(0, r), Kind: "V-105", Device: row.Get(t.DeviceColumn()),
				Text: fmt.Sprintf("%s の行とキーが同じです（%s）。この行は読みません", first.Origins[0].Addr, keyShown(t, row))})
		}
	}
}

// split gives the items of a cell: one for a plain column, the parts between
// commas and line breaks for a list column.
func split(c *schema.Column, raw string) []string {
	if !c.List {
		return []string{raw}
	}
	return strings.FieldsFunc(raw, func(r rune) bool { return r == ',' || r == '\n' || r == '\r' })
}

func keyShown(t *schema.Table, r *model.Row) string {
	parts := make([]string, len(r.Key))
	for i, k := range r.Key {
		parts[i] = k.Value
	}
	return strings.Join(parts, " / ")
}

func (b *Book) readMgmt(s *sheet.Sheet, m *mgmt) {
	names := map[string]string{}
	for _, c := range m.cols {
		names[foldName(c.title.JA)] = c.name
		names[foldName(c.title.EN)] = c.name
	}
	info, _ := b.header(s, names)
	b.Sheets[m.id] = info
	get := func(line []string, name string) string {
		ci, ok := info.Cols[name]
		if !ok {
			return ""
		}
		return strings.TrimSpace(at(line, ci))
	}
	for r := 1; r < len(s.Rows); r++ {
		line := s.Rows[r]
		if blank(line) {
			continue
		}
		switch m {
		case deviceSheet:
			d := DeviceMap{get(line, "old_host"), get(line, "old_vendor"), get(line, "new_host"), get(line, "new_vendor"),
				get(line, "new_model"), get(line, "new_os"), get(line, "note"), r + 1}
			if d.OldHost == "" {
				b.problemf(s.Name, addr(0, r), "旧ホスト名が空です。この行は使いません")
				continue
			}
			b.Devices = append(b.Devices, d)
		case ifSheet:
			f := IfMap{get(line, "new_host"), get(line, "old_if"), get(line, "new_if"), get(line, "basis"), r + 1}
			if f.NewHost == "" || f.OldIf == "" {
				b.problemf(s.Name, addr(0, r), "新ホスト名か旧 IF 名が空です。この行は使いません")
				continue
			}
			b.Ifs = append(b.Ifs, f)
		case commonSheet:
			b.Common = append(b.Common, Common{get(line, "scope"), get(line, "table"), get(line, "column"), get(line, "value"), r + 1})
		}
	}
}
