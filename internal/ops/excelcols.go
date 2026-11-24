package ops

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/listimport"
	"github.com/kojio145/configchief/internal/sheet"
)

// Configurations kept in Excel, one device in a column and one line in a
// cell (spec 14.16.4): a person picks the sheet, the header row and the
// columns, and each column becomes a text file in input/old or input/new.

// IsExcel tells whether a file name is one of an Excel book.
func IsExcel(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".xlsx", ".xlsm", ".xls":
		return true
	}
	return false
}

// ExcelColumn is one column that can be made into a configuration.
type ExcelColumn struct {
	Col    int      `json:"col"`    // 1-based
	Letter string   `json:"letter"` // A, B, …
	Header string   `json:"header"` // the value in the header row
	Lines  int      `json:"lines"`  // the lines under the header, to the last filled one
	Sample []string `json:"sample"` // the first filled cells under the header
	Name   string   `json:"name"`   // the file it is made into
}

// ExcelBook is what the page shows of one Excel file.
type ExcelBook struct {
	Path    string        `json:"path"`
	File    string        `json:"file"` // the name of the file
	Sheets  []string      `json:"sheets"`
	Sheet   string        `json:"sheet"`  // the sheet shown
	Header  int           `json:"header"` // the header row, 1-based
	Rows    int           `json:"rows"`   // the rows of the sheet
	Columns []ExcelColumn `json:"columns"`
}

// sampleLines is how many filled cells a column shows.
const sampleLines = 3

// ReadExcel reads an Excel file for the choice of the columns. An empty
// sheet takes the first one; header 0 takes the row that looks most like a
// header.
func ReadExcel(path, sheetName string, header int) (*ExcelBook, error) {
	bk, s, err := openExcel(path, sheetName)
	if err != nil {
		return nil, err
	}
	out := &ExcelBook{Path: path, File: filepath.Base(path), Sheet: s.Name, Rows: len(s.Rows), Columns: []ExcelColumn{}}
	for _, x := range bk.Sheets {
		out.Sheets = append(out.Sheets, x.Name)
	}
	if header <= 0 {
		header = listimport.GuessHeaderRow(s.Rows)
	}
	if header > len(s.Rows) && len(s.Rows) > 0 {
		header = len(s.Rows)
	}
	if header < 1 {
		header = 1
	}
	out.Header = header
	out.Columns = excelColumns(s, header)
	return out, nil
}

func openExcel(path, sheetName string) (*sheet.Book, *sheet.Sheet, error) {
	if !IsExcel(path) {
		return nil, nil, fmt.Errorf("%s は Excel のファイルではありません（xlsx・xlsm・xls）", filepath.Base(path))
	}
	bk, err := sheet.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("%s を読めません: %v", filepath.Base(path), err)
	}
	if len(bk.Sheets) == 0 {
		return nil, nil, fmt.Errorf("%s にシートがありません", filepath.Base(path))
	}
	if sheetName == "" {
		return bk, &bk.Sheets[0], nil
	}
	for i := range bk.Sheets {
		if bk.Sheets[i].Name == sheetName {
			return bk, &bk.Sheets[i], nil
		}
	}
	return nil, nil, fmt.Errorf("%s にシート「%s」がありません", filepath.Base(path), sheetName)
}

func cell(s *sheet.Sheet, r, c int) string {
	if r < 0 || r >= len(s.Rows) || c < 0 || c >= len(s.Rows[r]) {
		return ""
	}
	return s.Rows[r][c]
}

// excelColumns lists the columns that have a header or a value under it,
// with the name of the file each would be made into.
func excelColumns(s *sheet.Sheet, header int) []ExcelColumn {
	width := 0
	for _, row := range s.Rows {
		if len(row) > width {
			width = len(row)
		}
	}
	var out []ExcelColumn
	for c := 0; c < width; c++ {
		lines := columnLines(s, header, c)
		head := strings.TrimSpace(cell(s, header-1, c))
		if head == "" && len(lines) == 0 {
			continue
		}
		col := ExcelColumn{Col: c + 1, Letter: listimport.ColumnLetter(c + 1), Header: head, Lines: len(lines), Sample: []string{}}
		for _, l := range lines {
			if strings.TrimSpace(l) == "" {
				continue
			}
			col.Sample = append(col.Sample, l)
			if len(col.Sample) == sampleLines {
				break
			}
		}
		out = append(out, col)
	}
	names(s.Name, out)
	return out
}

// columnLines gives the cells of column c (0-based) under the header row,
// to the last filled one. The values are kept as they are, the spaces in
// front included (the indent tells the level of a line).
func columnLines(s *sheet.Sheet, header, c int) []string {
	last := -1
	for r := header; r < len(s.Rows); r++ {
		if strings.TrimSpace(cell(s, r, c)) != "" {
			last = r
		}
	}
	var out []string
	for r := header; r <= last; r++ {
		v := strings.ReplaceAll(cell(s, r, c), "\r\n", "\n")
		out = append(out, strings.ReplaceAll(v, "\r", "\n"))
	}
	return out
}

// names gives each column the name of its file: <header>.txt, or
// <sheet>_<letter>.txt when the header is empty; a name that comes twice
// takes _2, _3 ….
func names(sheetName string, cols []ExcelColumn) {
	seen := map[string]int{}
	for i := range cols {
		base := SafeFileName(cols[i].Header, "")
		if base == "" {
			base = SafeFileName(sheetName+"_"+cols[i].Letter, cols[i].Letter)
		}
		name := base
		for n := 2; seen[strings.ToLower(name)] > 0; n++ {
			name = fmt.Sprintf("%s_%d", base, n)
		}
		seen[strings.ToLower(name)]++
		cols[i].Name = name + ".txt"
	}
}

// AddExcelColumns makes the columns cols (1-based) of a sheet into text
// files in input/<source>, one file a column (spec 14.16.4). Files of the
// same name are handled as AddInputs does.
func (e *Env) AddExcelColumns(source, path, sheetName string, header int, cols []int, overwrite bool, now time.Time) (*Added, error) {
	if source != "old" && source != "new" {
		return nil, fmt.Errorf("Excel の列から置けるのは、旧コンフィグと新コンフィグだけです")
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("コンフィグにする列を選んでください")
	}
	_, s, err := openExcel(path, sheetName)
	if err != nil {
		return nil, err
	}
	if header < 1 || header > len(s.Rows) {
		return nil, fmt.Errorf("見出しの行 %d はシートにありません（1〜%d）", header, len(s.Rows))
	}
	all := excelColumns(s, header)
	byCol := map[int]ExcelColumn{}
	for _, c := range all {
		byCol[c.Col] = c
	}
	rel := "input/" + source
	out := newAdded(rel)
	if err := mkdirInput(e, rel); err != nil {
		return nil, err
	}
	sort.Ints(cols)
	var letters []string
	for _, n := range cols {
		c, ok := byCol[n]
		if !ok {
			out.Skipped = append(out.Skipped, fmt.Sprintf("%s 列: 見出しも値もありません", listimport.ColumnLetter(n)))
			continue
		}
		lines := columnLines(s, header, n-1)
		if len(lines) == 0 {
			out.Skipped = append(out.Skipped, fmt.Sprintf("%s 列（%s）: 見出しの下に値がありません", c.Letter, c.Header))
			continue
		}
		data := []byte(strings.Join(lines, "\n") + "\n")
		if err := e.putInput(out, source, c.Name, data, overwrite, now); err != nil {
			return nil, err
		}
		letters = append(letters, c.Letter)
	}
	sort.Strings(out.Copied)
	if len(out.Copied) > 0 {
		e.Proj.Log(now, "add: %s のシート「%s」（見出し %d 行目）の列 %s から、%s に %d ファイルを作りました（上書き %d）",
			filepath.Base(path), s.Name, header, strings.Join(letters, ","), rel, len(out.Copied), len(out.Backups))
	}
	return out, nil
}

func mkdirInput(e *Env, rel string) error {
	if err := os.MkdirAll(e.Proj.Path(rel), 0o755); err != nil {
		return fmt.Errorf("%s を作れません: %v", rel, err)
	}
	return nil
}
