package book

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
)

// Change is what a step wants to put into the book.
type Change struct {
	Data    *model.Dataset // rows for the sheets of the tables (nil = none)
	Devices []DeviceMap    // rows for 機器対応
	Ifs     []IfMap        // rows for IF 対応
	// Replace empties the sheets of the tables before Data goes in. The
	// other sheets are never emptied.
	Replace bool
	// Cells are values for cells of rows that are in the book: the one way
	// the tool changes what a person wrote, and only when asked to (spec
	// 6.5.4).
	Cells []CellSet
	// IfFill are proposals for rows of IF 対応 that are in the book: the new
	// interface name and the basis go into a row of the same new host name
	// and old interface whose new interface name is empty (spec 10.2). A
	// name a person wrote, and "-", stay.
	IfFill []IfMap
}

// CellSet puts a value into one cell of a row of the book.
type CellSet struct {
	Table  string
	Key    []model.KV // the key of the row, as the book has it
	Column string
	Cell   *model.Cell // nil = empty the cell
}

// Summary tells what Update did.
type Summary struct {
	Created    bool           // the book was made
	Added      map[string]int // table → rows added
	Kept       map[string]int // table → rows that were in the book already (not changed)
	DeviceRows int            // rows added to 機器対応
	IfRows     int            // rows added to IF 対応
	Sheets     int            // sheets made in a book that was there
	Cells      int            // cells of rows of the book that were changed
	IfFilled   int            // rows of IF 対応 whose empty new interface name was filled
	Replaced   bool           // the sheets of the tables were emptied first
	Problems   []Problem      // what Read said about the book as it was
}

// Changed tells whether the bytes differ from the book as it was, so that the
// caller has something to write.
func (s *Summary) Changed() bool {
	return s.Created || s.Replaced || s.Sheets > 0 || s.DeviceRows > 0 || s.IfRows > 0 || s.Rows() > 0 || s.Cells > 0 || s.IfFilled > 0
}

// Rows is the number of rows added to the sheets of the tables.
func (s *Summary) Rows() int {
	n := 0
	for _, v := range s.Added {
		n += v
	}
	return n
}

// Update gives the bytes of the book with the change put in (spec 6.2, 10).
//
// old is the book as it is (nil = there is none: a new one is made, with a
// sheet for every table). In a book that is there, the tool adds rows at the
// end of a sheet and makes the sheets that are missing; it does not touch
// what a person wrote. A row whose key is in the book already is left out.
func Update(old []byte, file string, sc *schema.Set, lang string, ch Change) ([]byte, *Summary, error) {
	sum := &Summary{Added: map[string]int{}, Kept: map[string]int{}}
	rows := func(table string) []*model.Row {
		if ch.Data == nil || ch.Data.Tables[table] == nil {
			return nil
		}
		return ch.Data.Tables[table].Rows
	}

	if old == nil && len(ch.Cells) > 0 {
		return nil, nil, fmt.Errorf("%s がないので、セルを書き換えられません", file)
	}
	if old == nil {
		sum.Created = true
		var grids []Grid
		for _, t := range sc.Tables {
			grids = append(grids, TableGrid(t, rows(t.Name), lang))
			if n := len(rows(t.Name)); n > 0 {
				sum.Added[t.Name] = n
			}
		}
		dev, ifs := uniqueMaps(nil, ch)
		var devRows, ifRows [][]string
		for _, d := range dev {
			devRows = append(devRows, d.cells())
		}
		for _, m := range ifs {
			ifRows = append(ifRows, m.cells())
		}
		sum.DeviceRows, sum.IfRows = len(dev), len(ifs)
		grids = append(grids, deviceSheet.grid(lang, devRows), ifSheet.grid(lang, ifRows), commonSheet.grid(lang, nil))
		var buf bytes.Buffer
		if err := WriteGrids(&buf, grids); err != nil {
			return nil, nil, err
		}
		return buf.Bytes(), sum, nil
	}

	bk, err := Read(old, file, sc)
	if err != nil {
		return nil, nil, err
	}
	sum.Problems = bk.Problems
	f, err := excelize.OpenReader(bytes.NewReader(old))
	if err != nil {
		return nil, nil, fmt.Errorf("%s を開けません: %v", file, err)
	}
	defer f.Close()
	st, err := newStyles(f)
	if err != nil {
		return nil, nil, err
	}
	order := f.GetSheetList()

	// A sheet cannot be deleted while it is the only one, so a spare sheet
	// stands by until the new ones are made.
	const spare = "configchief_tmp"
	if ch.Replace {
		sum.Replaced = true
		if _, err := f.NewSheet(spare); err != nil {
			return nil, nil, err
		}
		for _, t := range sc.Tables {
			if info := bk.Sheets[t.Name]; info != nil {
				if err := f.DeleteSheet(info.Name); err != nil {
					return nil, nil, err
				}
				// The new sheet takes the name the person gave the old one.
				bk.Sheets[t.Name] = &SheetInfo{Name: info.Name, Last: -1}
				delete(bk.Data.Tables, t.Name)
			}
		}
	}

	for _, t := range sc.Tables {
		info := bk.Sheets[t.Name]
		if info == nil || info.Last < 0 {
			g := TableGrid(t, rows(t.Name), lang)
			if info != nil {
				g.Name = info.Name
			}
			if _, err := f.NewSheet(g.Name); err != nil {
				return nil, nil, fmt.Errorf("シート「%s」を作れませんでした（%v）", g.Name, err)
			}
			if err := putGrid(f, g, st); err != nil {
				return nil, nil, err
			}
			sum.Sheets++
			if n := len(rows(t.Name)); n > 0 {
				sum.Added[t.Name] = n
			}
			continue
		}
		for _, r := range rows(t.Name) {
			if bk.Data.Tables[t.Name].Find(r.Key) != nil {
				sum.Kept[t.Name]++
				continue
			}
			bk.Data.Adopt(r)
			line := info.Last // the row after the last one, from 0
			info.Last++
			for _, c := range t.Columns {
				v := Show(c, r.Cells[c.Name], lang)
				if v == "" {
					continue
				}
				ci, err := column(f, info, st, c.Name, heading(c, lang))
				if err != nil {
					return nil, nil, err
				}
				if err := f.SetCellStr(info.Name, addr(ci, line), v); err != nil {
					return nil, nil, err
				}
			}
			for _, kv := range r.Extra {
				ci, err := column(f, info, st, freeCol(kv.Name), kv.Name)
				if err != nil {
					return nil, nil, err
				}
				if err := f.SetCellStr(info.Name, addr(ci, line), kv.Value); err != nil {
					return nil, nil, err
				}
			}
			sum.Added[t.Name]++
		}
	}

	for _, cs := range ch.Cells {
		t, info := sc.Table(cs.Table), bk.Sheets[cs.Table]
		c := t.Column(cs.Column)
		row := bk.Data.Tables[cs.Table].Find(cs.Key)
		if c == nil || info == nil || row == nil {
			return nil, nil, fmt.Errorf("%s に、書き換える行がありません（表 %s）", file, cs.Table)
		}
		line := sheetRow(row)
		if line < 0 {
			return nil, nil, fmt.Errorf("%s の行の場所が分かりません（表 %s）", file, cs.Table)
		}
		ci, err := column(f, info, st, c.Name, heading(c, lang))
		if err != nil {
			return nil, nil, err
		}
		if err := f.SetCellStr(info.Name, addr(ci, line), Show(c, cs.Cell, lang)); err != nil {
			return nil, nil, err
		}
		sum.Cells++
	}

	if info := bk.Sheets[ifSheet.id]; info != nil && len(ch.IfFill) > 0 {
		empty := map[string]IfMap{}
		for _, m := range bk.Ifs {
			if m.NewIf == "" && m.Row > 0 {
				empty[m.NewHost+"\x00"+m.OldIf] = m
			}
		}
		for _, p := range ch.IfFill {
			k := p.NewHost + "\x00" + p.OldIf
			m, ok := empty[k]
			if !ok || p.NewIf == "" {
				continue
			}
			delete(empty, k)
			for _, cv := range []struct{ name, v string }{{"new_if", p.NewIf}, {"basis", p.Basis}} {
				if cv.v == "" {
					continue
				}
				c := ifSheet.cols[ifSheet.col(cv.name)]
				ci, err := column(f, info, st, c.name, c.title.In(lang))
				if err != nil {
					return nil, nil, err
				}
				if err := f.SetCellStr(info.Name, addr(ci, m.Row-1), cv.v); err != nil {
					return nil, nil, err
				}
			}
			sum.IfFilled++
		}
	}

	dev, ifs := uniqueMaps(bk, ch)
	sum.DeviceRows, sum.IfRows = len(dev), len(ifs)
	var devRows, ifRows [][]string
	for _, d := range dev {
		devRows = append(devRows, d.cells())
	}
	for _, m := range ifs {
		ifRows = append(ifRows, m.cells())
	}
	for _, part := range []struct {
		m    *mgmt
		rows [][]string
	}{{deviceSheet, devRows}, {ifSheet, ifRows}, {commonSheet, nil}} {
		info := bk.Sheets[part.m.id]
		if info == nil {
			g := part.m.grid(lang, part.rows)
			if _, err := f.NewSheet(g.Name); err != nil {
				return nil, nil, fmt.Errorf("シート「%s」を作れませんでした（%v）", g.Name, err)
			}
			if err := putGrid(f, g, st); err != nil {
				return nil, nil, err
			}
			sum.Sheets++
			continue
		}
		for _, cells := range part.rows {
			line := info.Last
			info.Last++
			for i, v := range cells {
				if v == "" {
					continue
				}
				ci, err := column(f, info, st, part.m.cols[i].name, part.m.cols[i].title.In(lang))
				if err != nil {
					return nil, nil, err
				}
				if err := f.SetCellStr(info.Name, addr(ci, line), v); err != nil {
					return nil, nil, err
				}
			}
		}
	}

	if ch.Replace {
		if err := f.DeleteSheet(spare); err != nil {
			return nil, nil, err
		}
		// The sheets that were made again go back to where they were.
		have := map[string]bool{}
		for _, n := range f.GetSheetList() {
			have[n] = true
		}
		next := ""
		for i := len(order) - 1; i >= 0; i-- {
			if !have[order[i]] {
				continue
			}
			if next != "" {
				if err := f.MoveSheet(order[i], next); err != nil {
					return nil, nil, err
				}
			}
			next = order[i]
		}
		if len(order) > 0 {
			if idx, err := f.GetSheetIndex(order[0]); err == nil && idx >= 0 {
				f.SetActiveSheet(idx)
			}
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, nil, err
	}
	return buf.Bytes(), sum, nil
}

// column gives the place of a column of a sheet, adding a header cell at the
// right end if the sheet does not have the column.
func column(f *excelize.File, info *SheetInfo, st styles, name, title string) (int, error) {
	if ci, ok := info.Cols[name]; ok {
		return ci, nil
	}
	ci := info.Width
	info.Width++
	info.Cols[name] = ci
	if info.Last < 1 {
		info.Last = 1
	}
	cell := addr(ci, 0)
	if err := f.SetCellStr(info.Name, cell, title); err != nil {
		return 0, err
	}
	if err := f.SetCellStyle(info.Name, cell, cell, st.head); err != nil {
		return 0, err
	}
	return ci, nil
}

// uniqueMaps gives the rows of the change that the book does not have yet:
// one row per old host name, and one per new host name and old interface.
func uniqueMaps(bk *Book, ch Change) (dev []DeviceMap, ifs []IfMap) {
	seenDev, seenIf := map[string]bool{}, map[string]bool{}
	if bk != nil {
		for _, d := range bk.Devices {
			seenDev[d.OldHost] = true
		}
		for _, m := range bk.Ifs {
			seenIf[m.NewHost+"\x00"+m.OldIf] = true
		}
	}
	for _, d := range ch.Devices {
		if !seenDev[d.OldHost] {
			seenDev[d.OldHost] = true
			dev = append(dev, d)
		}
	}
	for _, m := range ch.Ifs {
		if k := m.NewHost + "\x00" + m.OldIf; !seenIf[k] {
			seenIf[k] = true
			ifs = append(ifs, m)
		}
	}
	return dev, ifs
}

// sheetRow gives the row of the sheet a row of the book was read from, from
// 0; -1 if it was not read from a sheet.
func sheetRow(r *model.Row) int {
	for _, o := range r.Origins {
		if o.Kind != "book" || o.Addr == "" {
			continue
		}
		i := strings.IndexFunc(o.Addr, func(c rune) bool { return c >= '0' && c <= '9' })
		if i < 0 {
			continue
		}
		if n, err := strconv.Atoi(o.Addr[i:]); err == nil && n >= 1 {
			return n - 1
		}
	}
	return -1
}
