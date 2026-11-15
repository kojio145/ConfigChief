package book

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// The rows that WriteGrids puts into the file as text read back as the
// rows the library writes cell by cell (putGrid), whatever the text.
func TestFillRowsReadsBack(t *testing.T) {
	long := strings.Repeat("あ", maxCellChars+10)
	odd := []string{
		"plain", " lead", "trail ", "\ttab", "two\nlines", "cr\r\nlf", "<&>\"'",
		"_x0041_", "_x005F_", "_x005F_x0041_", "a_x000D_b", "_xZZZZ_", "__x0041__", "_x0041_x0042_",
		"bell\x07", "日本語", "0010", "1/2", "1e5", "=A1", long, "😀" + long,
	}
	var rows [][]string
	for _, s := range odd {
		rows = append(rows, []string{s, "", s, "beyond the headings"})
	}
	rows = append(rows, []string{}, []string{"", ""}, []string{"last"})
	grids := []Grid{
		{Name: "一覧", Header: []string{"a", "b", "c"}, Rows: rows, Notes: map[int]string{1: "note"}, Choices: map[int][]string{2: {"p", "q"}}},
		{Name: "empty", Header: []string{"h"}},
		{Name: "third", Header: []string{"h", "i"}, Rows: [][]string{{"x", "y"}}},
	}

	var fast bytes.Buffer
	if err := WriteGrids(&fast, grids); err != nil {
		t.Fatal(err)
	}
	var again bytes.Buffer
	if err := WriteGrids(&again, grids); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(fast.Bytes(), again.Bytes()) {
		t.Error("the same grids gave different bytes")
	}

	// The same grids, every cell through the library.
	f := excelize.NewFile()
	defer f.Close()
	st, err := newStyles(f)
	if err != nil {
		t.Fatal(err)
	}
	for i, g := range grids {
		if i == 0 {
			err = f.SetSheetName("Sheet1", g.Name)
		} else {
			_, err = f.NewSheet(g.Name)
		}
		if err != nil {
			t.Fatal(err)
		}
		if err := putGrid(f, g, st); err != nil {
			t.Fatal(err)
		}
	}
	slow, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}

	read := func(data []byte) map[string][][]string {
		t.Helper()
		f, err := excelize.OpenReader(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		defer f.Close()
		out := map[string][][]string{}
		for _, name := range f.GetSheetList() {
			rows, err := f.GetRows(name)
			if err != nil {
				t.Fatal(err)
			}
			out[name] = rows
			// The format of the cells is the same too.
			for r := range rows {
				for c := range rows[r] {
					cell, _ := excelize.CoordinatesToCellName(c+1, r+1)
					id, _ := f.GetCellStyle(name, cell)
					out[name+" styles"] = append(out[name+" styles"], []string{cell, string(rune('0' + id))})
				}
			}
		}
		return out
	}
	a, b := read(fast.Bytes()), read(slow.Bytes())
	// Every cell reads back as it was written (cut at the limit of a cell;
	// a character the format cannot hold becomes the mark for one).
	for _, g := range grids {
		for r, row := range g.Rows {
			for c, v := range row {
				want := strings.ReplaceAll(clip(v), "", "�")
				got := ""
				if r+1 < len(a[g.Name]) && c < len(a[g.Name][r+1]) {
					got = a[g.Name][r+1][c]
				}
				if got != want {
					t.Errorf("%s row %d column %d reads %.60q, want %.60q", g.Name, r+2, c+1, got, want)
				}
			}
		}
	}
	// And as the library writes it, cell by cell: the same rows, the same
	// formats. (A text like "_x0041_" is left out: the library writes it so
	// that it reads back as the character of that number.)
	for name := range b {
		if strings.HasSuffix(name, " styles") {
			if !reflect.DeepEqual(a[name], b[name]) {
				t.Errorf("%s: %v, want %v", name, a[name], b[name])
			}
			continue
		}
		if len(a[name]) != len(b[name]) {
			t.Errorf("%s has %d rows, want %d", name, len(a[name]), len(b[name]))
			continue
		}
		for r := range b[name] {
			if strings.Contains(strings.Join(b[name][r], ""), "x00") || strings.Contains(strings.Join(a[name][r], ""), "_x") {
				continue
			}
			if !reflect.DeepEqual(a[name][r], b[name][r]) {
				t.Errorf("%s row %d: %.80q, want %.80q", name, r+1, a[name][r], b[name][r])
			}
		}
	}
	if utf16Len(clip(long)) != maxCellChars || utf16Len(clip("😀"+long)) != maxCellChars {
		t.Error("a long text is not cut at the limit of a cell")
	}
}

func utf16Len(s string) int {
	n := 0
	for _, r := range s {
		n++
		if r > 0xFFFF {
			n++
		}
	}
	return n
}
