package book

import (
	"fmt"
	"io"

	"github.com/xuri/excelize/v2"
)

// Grid is one sheet to write: a header row and the value rows, all as text.
type Grid struct {
	Name    string
	Header  []string
	Rows    [][]string
	Notes   map[int]string   // column index → comment on the header cell
	Choices map[int][]string // column index → values of the drop-down list
}

// fixedTime goes into the document properties so that the same grids always
// give the same bytes (the tool's rule: same input, same output).
const fixedTime = "2000-01-01T00:00:00Z"

// maxChoiceLen is Excel's limit for the comma-joined values of an inline
// drop-down list.
const maxChoiceLen = 255

// WriteGrids writes the grids as one xlsx file.
//
// Every cell is written as a string with the text format, so that Excel shows
// "0010", "1/2" or "1e5" as typed and does not turn what the user types later
// into a number or a date. The header row is frozen and gets a filter.
func WriteGrids(w io.Writer, grids []Grid) error {
	if len(grids) == 0 {
		return fmt.Errorf("書き出すシートがありません")
	}
	f := excelize.NewFile()
	defer f.Close()

	st, err := newStyles(f)
	if err != nil {
		return err
	}
	for gi, g := range grids {
		if gi == 0 {
			err = f.SetSheetName("Sheet1", g.Name)
		} else {
			_, err = f.NewSheet(g.Name)
		}
		if err != nil {
			return fmt.Errorf("シート「%s」を作れませんでした（%v）", g.Name, err)
		}
		if err = putFrame(f, g, st); err != nil {
			return err
		}
	}
	if err = f.SetDocProps(&excelize.DocProperties{Creator: "ConfigChief", Created: fixedTime, Modified: fixedTime}); err != nil {
		return err
	}
	// The rows go in as text of their own (rows.go): handing some hundred
	// thousand cells to the library one by one takes seconds (spec 16).
	frame, err := f.WriteToBuffer()
	if err != nil {
		return err
	}
	return fillRows(w, frame.Bytes(), grids, st.text)
}

// styles are the cell styles of a sheet the tool writes.
type styles struct {
	text int // the text format
	head int // the header row
}

func newStyles(f *excelize.File) (styles, error) {
	text, err := f.NewStyle(&excelize.Style{NumFmt: 49})
	if err != nil {
		return styles{}, err
	}
	head, err := f.NewStyle(&excelize.Style{
		NumFmt: 49,
		Font:   &excelize.Font{Bold: true},
		Fill:   excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"D9E1F2"}},
	})
	return styles{text, head}, err
}

// putGrid fills an empty sheet of the name of the grid.
func putGrid(f *excelize.File, g Grid, st styles) (err error) {
	if err = putFrame(f, g, st); err != nil || len(g.Header) == 0 {
		return err
	}
	for r, row := range g.Rows {
		for c, v := range row {
			if v == "" {
				continue
			}
			cell, _ := excelize.CoordinatesToCellName(c+1, r+2)
			if err = f.SetCellStr(g.Name, cell, v); err != nil {
				return err
			}
		}
	}
	return nil
}

// putFrame gives an empty sheet of the name of the grid everything but its
// rows: the headings, the formats, the filter over the rows to come, the
// comments and the lists to choose from.
func putFrame(f *excelize.File, g Grid, st styles) (err error) {
	if len(g.Header) == 0 {
		return nil
	}
	lastCol, _ := excelize.ColumnNumberToName(len(g.Header))
	// The text format on whole columns covers the rows the user adds.
	if err = f.SetColStyle(g.Name, "A:"+lastCol, st.text); err != nil {
		return err
	}
	for c, v := range g.Header {
		cell, _ := excelize.CoordinatesToCellName(c+1, 1)
		if err = f.SetCellStr(g.Name, cell, v); err != nil {
			return err
		}
	}
	if err = f.SetCellStyle(g.Name, "A1", lastCol+"1", st.head); err != nil {
		return err
	}
	if err = f.SetPanes(g.Name, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return err
	}
	if err = f.AutoFilter(g.Name, fmt.Sprintf("A1:%s%d", lastCol, len(g.Rows)+1), nil); err != nil {
		return err
	}
	for c := 0; c < len(g.Header); c++ {
		col, _ := excelize.ColumnNumberToName(c + 1)
		if note := g.Notes[c]; note != "" {
			if err = f.AddComment(g.Name, excelize.Comment{Cell: col + "1", Author: "ConfigChief", Text: note}); err != nil {
				return err
			}
		}
		if list := g.Choices[c]; len(list) > 0 {
			n := len(list) - 1
			for _, s := range list {
				n += len(s)
			}
			if n > maxChoiceLen {
				return fmt.Errorf("シート「%s」の %s 列: 選択肢が長すぎます（Excel の上限 %d 文字）", g.Name, col, maxChoiceLen)
			}
			dv := excelize.NewDataValidation(true)
			dv.SetSqref(fmt.Sprintf("%s2:%s1048576", col, col))
			if err = dv.SetDropList(list); err != nil {
				return err
			}
			if err = f.AddDataValidation(g.Name, dv); err != nil {
				return err
			}
		}
	}
	return nil
}
