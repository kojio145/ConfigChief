package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// excelOf writes an xlsx whose sheets hold the given rows (sheet → rows).
func excelOf(t *testing.T, dir, name string, sheets map[string][][]string, order []string) string {
	t.Helper()
	f := excelize.NewFile()
	defer f.Close()
	for i, sh := range order {
		if i == 0 {
			if err := f.SetSheetName("Sheet1", sh); err != nil {
				t.Fatal(err)
			}
		} else if _, err := f.NewSheet(sh); err != nil {
			t.Fatal(err)
		}
		for r, row := range sheets[sh] {
			for c, v := range row {
				if v == "" {
					continue
				}
				cellName, _ := excelize.CoordinatesToCellName(c+1, r+1)
				if err := f.SetCellStr(sh, cellName, v); err != nil {
					t.Fatal(err)
				}
			}
		}
	}
	path := filepath.Join(dir, name)
	if err := f.SaveAs(path); err != nil {
		t.Fatal(err)
	}
	return path
}

// The configurations of two sites side by side, under a title row, with an
// empty cell and a note in the middle (spec 14.16.4). Every value is made up.
var siteRows = [][]string{
	{"拠点ルータの設定（見本）"},
	{"", "拠点A", "拠点B", ""},
	{"", "hostname rt-a01", "hostname rt-b01"},
	{"", "interface GigabitEthernet0/0", "interface GigabitEthernet0/0"},
	{"", " ip address 192.0.2.1 255.255.255.0", " ip address 198.51.100.1 255.255.255.0"},
	{"", "", "※ 回線は後日"},
	{"", "!", "!"},
	{"", "", "end"},
}

func TestReadExcel(t *testing.T) {
	dir := t.TempDir()
	path := excelOf(t, dir, "sites.xlsx", map[string][][]string{"表紙": {{"表紙"}}, "設定": siteRows}, []string{"表紙", "設定"})
	b, err := ReadExcel(path, "設定", 0)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(b.Sheets, ",") != "表紙,設定" || b.Sheet != "設定" {
		t.Errorf("sheets %v %s", b.Sheets, b.Sheet)
	}
	if b.Header != 2 {
		t.Errorf("header row %d, want 2", b.Header)
	}
	if len(b.Columns) != 2 {
		t.Fatalf("columns %+v", b.Columns)
	}
	a, c := b.Columns[0], b.Columns[1]
	if a.Letter != "B" || a.Header != "拠点A" || a.Name != "拠点A.txt" || a.Lines != 5 {
		t.Errorf("column B %+v", a)
	}
	if c.Letter != "C" || c.Lines != 6 || len(c.Sample) != 3 || c.Sample[2] != " ip address 198.51.100.1 255.255.255.0" {
		t.Errorf("column C %+v", c)
	}
	// Another header row is taken as it is asked.
	b, err = ReadExcel(path, "設定", 3)
	if err != nil {
		t.Fatal(err)
	}
	if b.Header != 3 || b.Columns[0].Header != "hostname rt-a01" || b.Columns[0].Name != "hostname rt-a01.txt" {
		t.Errorf("header 3: %+v", b.Columns)
	}
	// The first sheet when none is named; a sheet that is not there.
	if b, err := ReadExcel(path, "", 0); err != nil || b.Sheet != "表紙" {
		t.Errorf("first sheet: %v %+v", err, b)
	}
	if _, err := ReadExcel(path, "無い", 0); err == nil || !strings.Contains(err.Error(), "シート「無い」") {
		t.Errorf("no sheet: %v", err)
	}
	if _, err := ReadExcel(filepath.Join(dir, "a.txt"), "", 0); err == nil {
		t.Error("a text file is read as Excel")
	}
}

func TestExcelNames(t *testing.T) {
	cols := []ExcelColumn{{Letter: "A", Header: "拠点/1"}, {Letter: "B", Header: "拠点/1"}, {Letter: "C"}}
	names("設定", cols)
	if cols[0].Name != "拠点_1.txt" || cols[1].Name != "拠点_1_2.txt" || cols[2].Name != "設定_C.txt" {
		t.Errorf("names %+v", cols)
	}
}

func TestAddExcelColumns(t *testing.T) {
	e := newCase(t, nil)
	path := excelOf(t, t.TempDir(), "sites.xlsx", map[string][][]string{"設定": siteRows}, []string{"設定"})
	a, err := e.AddExcelColumns("old", path, "設定", 2, []int{3, 2}, false, at)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(a.Copied, ",") != "拠点A.txt,拠点B.txt" {
		t.Fatalf("copied %v (skipped %v)", a.Copied, a.Skipped)
	}
	got := read(t, e, "input/old/拠点B.txt")
	want := "hostname rt-b01\ninterface GigabitEthernet0/0\n ip address 198.51.100.1 255.255.255.0\n※ 回線は後日\n!\nend\n"
	if got != want {
		t.Errorf("拠点B.txt:\n%q\nwant\n%q", got, want)
	}
	// The empty cell is an empty line; the spaces in front stay.
	if got := read(t, e, "input/old/拠点A.txt"); got != "hostname rt-a01\ninterface GigabitEthernet0/0\n ip address 192.0.2.1 255.255.255.0\n\n!\n" {
		t.Errorf("拠点A.txt: %q", got)
	}
	// The same again: nothing is written. Another content: asked first.
	a, err = e.AddExcelColumns("old", path, "設定", 2, []int{2}, false, at)
	if err != nil || len(a.Same) != 1 || len(a.Copied) != 0 {
		t.Errorf("same again: %v %+v", err, a)
	}
	writeTestFile(e.Proj.Path("input/old/拠点A.txt"), []byte("hostname other\n"), 0o644)
	a, _ = e.AddExcelColumns("old", path, "設定", 2, []int{2}, false, at)
	if len(a.Clash) != 1 || read(t, e, "input/old/拠点A.txt") != "hostname other\n" {
		t.Errorf("clash: %+v", a)
	}
	a, _ = e.AddExcelColumns("old", path, "設定", 2, []int{2}, true, at)
	if len(a.Copied) != 1 || len(a.Backups) != 1 {
		t.Errorf("overwrite: %+v", a)
	}
	// The files import as two devices (the sample is too short for the
	// vendor to be told, so the pack is chosen), the note an unhandled line.
	src, err := e.Import("old", "cisco_ios")
	if err != nil {
		t.Fatal(err)
	}
	hosts := map[string]bool{}
	for _, f := range src.Files {
		hosts[f.Device] = true
	}
	if !hosts["rt-a01"] || !hosts["rt-b01"] {
		t.Errorf("devices %v", hosts)
	}
	// Not for the logs; a column with nothing is told.
	if _, err := e.AddExcelColumns("logs", path, "設定", 2, []int{2}, false, at); err == nil {
		t.Error("columns put into the logs")
	}
	if a, _ := e.AddExcelColumns("new", path, "設定", 2, []int{9}, false, at); len(a.Skipped) != 1 {
		t.Errorf("empty column: %+v", a)
	}
}

func TestAddInputsLeavesExcel(t *testing.T) {
	e := newCase(t, nil)
	dir := t.TempDir()
	xl := excelOf(t, dir, "sites.xlsx", map[string][][]string{"設定": siteRows}, []string{"設定"})
	txt := filepath.Join(dir, "rt-c01.txt")
	os.WriteFile(txt, []byte("hostname rt-c01\n"), 0o644)
	a, err := e.AddInputs("old", []string{xl, txt}, false, at)
	if err != nil {
		t.Fatal(err)
	}
	if len(a.Excel) != 1 || a.Excel[0] != xl || strings.Join(a.Copied, ",") != "rt-c01.txt" {
		t.Errorf("added %+v", a)
	}
	if _, err := os.Stat(e.Proj.Path("input/old/sites.xlsx")); err == nil {
		t.Error("the Excel file was copied")
	}
	// The list of the customer still takes Excel files.
	if a, _ := e.AddInputs("customer", []string{xl}, false, at); len(a.Copied) != 1 || len(a.Excel) != 0 {
		t.Errorf("customer %+v", a)
	}
	// An Excel file put into input/old by hand is told on import.
	writeTestFile(e.Proj.Path("input/old/hand.xlsx"), []byte("PK"), 0o644)
	src, err := e.Import("old", "")
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range src.Files {
		if strings.HasSuffix(f.Path, "hand.xlsx") {
			found = strings.Contains(f.Err, "列を選んでコンフィグにできます")
		}
	}
	if !found {
		t.Errorf("hand.xlsx not told: %+v", src.Files)
	}
}
