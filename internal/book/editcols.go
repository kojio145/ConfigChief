package book

import (
	"fmt"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/kojio145/configchief/internal/schema"
)

// The free columns of a sheet in the table of a screen (spec 14.6): a
// person adds one, gives one another heading or takes one out. The draft
// keeps that (DraftSheet.Columns); the save writes it after the rows.

// withColumns gives the sheet with what the draft does to its free columns:
// a column to add stands at the right end (the cells of the rows grow with
// it), a column to rename shows its new heading and keeps its name, a column
// to take out is marked. The sheet itself is not changed.
func (es *EditSheet) withColumns(part *DraftSheet) *EditSheet {
	if part == nil || len(part.Columns) == 0 {
		return es
	}
	cp := *es
	cp.Columns = append([]EditColumn{}, es.Columns...)
	for _, c := range part.Columns {
		key := freeCol(c.Title)
		ci := cp.index(key)
		switch c.Op {
		case "add":
			if ci < 0 {
				cp.Columns = append(cp.Columns, EditColumn{Key: key, Title: c.Title, Col: -1, Free: true, Added: true, Choices: []string{}})
			}
		case "rename":
			if ci >= 0 && cp.Columns[ci].Free && !cp.Columns[ci].Added {
				cp.Columns[ci].Was, cp.Columns[ci].Title = cp.Columns[ci].Title, c.To
			}
		case "drop":
			if ci >= 0 && cp.Columns[ci].Free {
				cp.Columns[ci].Dropped = true
			}
		}
	}
	if n := len(cp.Columns); n > len(es.Columns) {
		cp.Rows = make([]EditRow, len(es.Rows))
		for i, r := range es.Rows {
			cells := make([]string, n)
			copy(cells, r.Cells)
			r.Cells = cells
			cp.Rows[i] = r
		}
	}
	return &cp
}

// columns writes what the draft does to the free columns of the sheet.
func (p *plan) columns(f *excelize.File, st styles, sum *EditSummary) error {
	info := p.es.place
	for _, c := range p.cols {
		key := freeCol(c.Title)
		ci, have := info.Cols[key]
		switch c.Op {
		case "add":
			if p.had[key] {
				continue // the sheet had it before the save
			}
			// A value written for it made the heading already.
			if _, err := column(f, info, st, key, c.Title); err != nil {
				return err
			}
			sum.Columns++
		case "rename":
			to := freeCol(c.To)
			if _, taken := info.Cols[to]; !have || taken {
				continue // the column is gone, or the heading belongs to another by now
			}
			if err := f.SetCellStr(info.Name, addr(ci, 0), c.To); err != nil {
				return err
			}
			delete(info.Cols, key)
			info.Cols[to] = ci
			sum.Columns++
		case "drop":
			if !have {
				continue
			}
			if err := p.drop(f, key, ci); err != nil {
				return err
			}
			sum.Columns++
		}
	}
	return nil
}

// drop takes a column out of the sheet. The columns right of it move left,
// but not the comments on their cells: with a comment right of it the column
// stays and is emptied, its heading too (a column without a heading is not
// read).
func (p *plan) drop(f *excelize.File, key string, ci int) error {
	info := p.es.place
	keep := false
	if comments, err := f.GetComments(info.Name); err == nil {
		for _, c := range comments {
			if col, _, err := excelize.CellNameToCoordinates(c.Cell); err == nil && col > ci+1 {
				keep = true
			}
		}
	}
	delete(info.Cols, key)
	if keep {
		rows, err := f.GetRows(info.Name)
		if err != nil {
			return err
		}
		for r := range rows {
			if err := emptyCell(f, info.Name, addr(ci, r)); err != nil {
				return err
			}
		}
		return nil
	}
	name, err := excelize.ColumnNumberToName(ci + 1)
	if err != nil {
		return err
	}
	if err := f.RemoveCol(info.Name, name); err != nil {
		return err
	}
	for k, c := range info.Cols {
		if c > ci {
			info.Cols[k] = c - 1
		}
	}
	if info.Width > ci {
		info.Width--
	}
	return nil
}

// frame gives the sheet of a table or of the mapping as the tool makes it:
// the headings, their comments and the lists to choose from, without a row.
func frame(sc *schema.Set, id, lang string) (Grid, bool) {
	for _, m := range mgmtSheets {
		if m.id == id {
			return m.grid(lang, nil), true
		}
	}
	if t := sc.Table(id); t != nil {
		return TableGrid(t, nil, lang), true
	}
	return Grid{}, false
}

// FreeColumn tells the name of a free column with the heading title for a
// sheet, or why the sheet cannot have such a column: the heading is one of
// the definition of the sheet (the tool would read the column as that one),
// or one of headings, the headings the sheet has.
func FreeColumn(sc *schema.Set, sheetID, title string, headings []string) (string, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", fmt.Errorf("列の見出しを書いてください")
	}
	if strings.ContainsAny(title, "\r\n\t") {
		return "", fmt.Errorf("列の見出しに、改行とタブは使えません")
	}
	if len([]rune(title)) > 100 {
		return "", fmt.Errorf("列の見出しが長すぎます（100 文字まで）")
	}
	fold := foldName(title)
	var defined []string
	for _, m := range mgmtSheets {
		if m.id == sheetID {
			for _, c := range m.cols {
				defined = append(defined, c.title.JA, c.title.EN)
			}
		}
	}
	if t := sc.Table(sheetID); t != nil {
		for _, c := range t.Columns {
			defined = append(defined, c.Title.JA, c.Title.EN, c.Name)
		}
	}
	for _, n := range defined {
		if n != "" && foldName(n) == fold {
			return "", fmt.Errorf("「%s」は、この表の定義にある列の名前です。別の見出しにしてください", title)
		}
	}
	for _, h := range headings {
		if foldName(h) == fold {
			return "", fmt.Errorf("見出し「%s」の列は、もうあります", title)
		}
	}
	return freeCol(title), nil
}
