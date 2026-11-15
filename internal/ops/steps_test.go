package ops

import (
	"bytes"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/kojio145/configchief/internal/convert"
	"github.com/kojio145/configchief/internal/model"
)

var at = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

const reports = "out/2026-01-02_030405/reports/"

func read(t *testing.T, e *Env, rel string) string {
	t.Helper()
	data, err := os.ReadFile(e.Proj.Path(rel))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// secretValues gives every value of a secret column of the data.
func secretValues(e *Env, d *model.Dataset) []string {
	var out []string
	for name, t := range d.Tables {
		for _, r := range t.Rows {
			for _, k := range r.Key {
				if e.secret(name, k.Name) && k.Value != "" {
					out = append(out, k.Value)
				}
			}
			for col, c := range r.Cells {
				if e.secret(name, col) {
					for _, v := range c.Values {
						if v != "" {
							out = append(out, v)
						}
					}
				}
			}
		}
	}
	return out
}

// setCell writes one cell of the mapping sheets of the book: the row that
// holds find in the column under the heading where gets value under the
// heading set.
func setCell(t *testing.T, e *Env, sheet, where, find, set, value string) {
	t.Helper()
	f, err := excelize.OpenFile(e.Proj.BookPath())
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := f.GetRows(sheet)
	if err != nil {
		t.Fatal(err)
	}
	head, wc, sc := -1, -1, -1
	for i, row := range rows {
		for c, v := range row {
			if v == where {
				head, wc = i, c
			}
		}
		if head == i {
			for c, v := range row {
				if v == set {
					sc = c
				}
			}
			break
		}
	}
	if head < 0 || sc < 0 {
		t.Fatalf("the sheet %s has no headings %s and %s", sheet, where, set)
	}
	done := false
	for i := head + 1; i < len(rows); i++ {
		if wc < len(rows[i]) && rows[i][wc] == find {
			addr, _ := excelize.CoordinatesToCellName(sc+1, i+1)
			if err := f.SetCellValue(sheet, addr, value); err != nil {
				t.Fatal(err)
			}
			done = true
		}
	}
	if !done {
		t.Fatalf("the sheet %s has no row with %s = %s", sheet, where, find)
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	if err := e.Proj.WriteBook(buf.Bytes()); err != nil {
		t.Fatal(err)
	}
}

func TestRunImportOld(t *testing.T) {
	e := newCase(t, map[string]string{"input/old/rt-a01.txt": sample(t, "cisco_ios")})
	e.Version = "1.2.3"
	o, err := e.RunImport("old", "", at)
	if err != nil {
		t.Fatal(err)
	}
	if o.New != nil {
		t.Error("a new side without files in input/new")
	}
	// The interfaces wait for a new host name: IF 対応 is by new device.
	if o.Book == nil || !o.Book.Created || o.Book.DeviceRows != 1 || o.Book.IfRows != 0 || o.Backup != "" {
		t.Errorf("book = %+v, backup %q", o.Book, o.Backup)
	}
	// Nobody wrote a new host name, and the tool does not guess one.
	if a := o.Applied; a == nil || !reflect.DeepEqual(a.Unmapped, []string{"rt-a01"}) || !reflect.DeepEqual(a.NoReason, []string{"rt-a01"}) {
		t.Errorf("applied = %+v", o.Applied)
	}
	if o.Findings() == 0 {
		t.Error("no findings")
	}
	if want := (Files{reports + "import_old.xlsx", reports + "import_old.html"}); o.Report != want {
		t.Errorf("report = %+v", o.Report)
	}
	h := read(t, e, o.Report.HTML)
	for _, want := range []string{
		"取り込みの結果（旧コンフィグ）",
		"ConfigChief 1.2.3",
		"M-407",
		"username backup password 0 ***",
		"crypto isakmp key *** address 203.0.113.20",
		"crypto isakmp key *** address 203.0.113.40",
		"snmp-server community *** RO 10",
		// A line that only stands beside a secret keeps its text.
		"crypto isakmp policy 10",
	} {
		if !strings.Contains(h, want) {
			t.Errorf("the report lacks %q", want)
		}
	}
	secrets := secretValues(e, o.Source.Data)
	if len(secrets) < 5 {
		t.Fatalf("secrets of the sample = %q", secrets)
	}
	for _, v := range secrets {
		if strings.Contains(h, v) {
			t.Errorf("the report shows the secret %q", v)
		}
	}
	f, err := excelize.OpenFile(e.Proj.Path(o.Report.XLSX))
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(f.GetSheetList(), ","); got != "まとめ,機器別,未対応の形,未対応の行,対応の未決,注意,行の一覧" {
		t.Errorf("sheets = %s", got)
	}
	rows, _ := f.GetRows("行の一覧")
	f.Close()
	if n := o.Source.Count().Total; len(rows) < n+1 {
		t.Errorf("the list of lines has %d rows for %d lines", len(rows), n)
	}
	if log := read(t, e, "state/log.txt"); !strings.Contains(log, "import old: 1 ファイル") {
		t.Errorf("log = %q", log)
	}

	// A second run adds nothing and leaves the book alone.
	before := read(t, e, e.Proj.Config.Book)
	o, err = e.RunImport("old", "", at)
	if err != nil {
		t.Fatal(err)
	}
	if o.Book.Changed() || o.Backup != "" || read(t, e, e.Proj.Config.Book) != before {
		t.Errorf("the second run changed the book: %+v, backup %q", o.Book, o.Backup)
	}

	// What a person writes into 機器対応 counts from then on.
	setCell(t, e, "機器対応", "旧ホスト名", "rt-a01", "新ホスト名", "rt-b01")
	o, err = e.RunImport("old", "", at)
	if err != nil {
		t.Fatal(err)
	}
	if o.Book.Created || o.Book.DeviceRows != 0 || o.Book.IfRows == 0 || o.Backup == "" {
		t.Errorf("book after the edit = %+v, backup %q", o.Book, o.Backup)
	}
	if a := o.Applied; len(a.Unmapped) != 0 || a.Hosts["rt-a01"] != "rt-b01" || len(a.Undecided) != o.Book.IfRows {
		t.Errorf("applied after the edit: unmapped %v, hosts %v, undecided %d", a.Unmapped, a.Hosts, len(a.Undecided))
	}
	if r := o.Applied.Data.Tables["devices"]; r == nil || len(r.Rows) != 1 || r.Rows[0].KeyText() != "hostname=rt-b01" {
		t.Errorf("devices after the edit = %+v", r)
	}
}

func TestRunImportMaskOff(t *testing.T) {
	e := newCase(t, map[string]string{"input/new/rt-a01.txt": sample(t, "cisco_ios")})
	e.Proj.Config.Secrets.MaskInReports = false
	o, err := e.RunImport("new", "", at)
	if err != nil {
		t.Fatal(err)
	}
	if o.Book != nil || o.Applied != nil {
		t.Error("the new side touched the book")
	}
	if _, err := os.Stat(e.Proj.BookPath()); err == nil {
		t.Error("the new side made a book")
	}
	h := read(t, e, o.Report.HTML)
	if !strings.Contains(h, "取り込みの結果（新コンフィグ）") || strings.Contains(h, "***") {
		t.Error("the report of the new side is not as it should be")
	}
	if v := secretValues(e, o.Source.Data)[0]; !strings.Contains(h, v) {
		t.Errorf("masking is off, but the report lacks %q", v)
	}
}

func TestRunImportErrors(t *testing.T) {
	e := newCase(t, nil)
	for _, source := range []string{"old", "new", "logs"} {
		if _, err := e.RunImport(source, "", at); err == nil {
			t.Errorf("no error for %s", source)
		}
	}
	if dirs, _ := os.ReadDir(e.Proj.Path("out")); len(dirs) != 0 {
		t.Error("a failed step wrote a report")
	}
}

// The same text as old and as new: the proposals fill the whole mapping, and
// the comparison finds nothing.
func TestProposeAndCompare(t *testing.T) {
	text := sample(t, "cisco_ios")
	e := newCase(t, map[string]string{
		"input/old/rt-a01.txt": text,
		"input/new/rt-a01.txt": strings.Replace(text, "description sample loopback", "description sample loopback 2", 1),
	})
	if _, err := e.Compare("old", "new", at); err == nil || !strings.Contains(err.Error(), "import --source old") {
		t.Errorf("comparing without a book: %v", err)
	}
	o, err := e.RunImport("old", "", at)
	if err != nil {
		t.Fatal(err)
	}
	if o.New == nil {
		t.Fatal("the new side was not read")
	}
	if a := o.Applied; len(a.Unmapped)+len(a.Undecided)+len(a.Shared) != 0 || a.Hosts["rt-a01"] != "rt-a01" {
		t.Errorf("applied = unmapped %v, undecided %v, shared %v", a.Unmapped, a.Undecided, a.Shared)
	}
	_, bk, err := e.readBook()
	if err != nil {
		t.Fatal(err)
	}
	if len(bk.Ifs) == 0 {
		t.Fatal("no rows in IF 対応")
	}
	for _, m := range bk.Ifs {
		if m.NewIf != m.OldIf || m.Basis != convert.BasisSameName {
			t.Errorf("IF 対応: %+v", m)
		}
	}

	c, err := e.Compare("old", "new", at)
	if err != nil {
		t.Fatal(err)
	}
	r := c.Result
	if len(r.OnlyA)+len(r.OnlyB) != 0 || len(r.Cells) != 1 || r.Cells[0].Table != "interfaces" || r.Cells[0].Column != "description" || r.Cells[0].B.Text() != "sample loopback 2" {
		t.Errorf("old and new: only A %d, only B %d, cells %+v", len(r.OnlyA), len(r.OnlyB), r.Cells)
	}
	if want := (Files{reports + "compare_old_new.xlsx", reports + "compare_old_new.html"}); c.Report != want {
		t.Errorf("report = %+v", c.Report)
	}
	h := read(t, e, c.Report.HTML)
	for _, want := range []string{"比較の結果（旧コンフィグ と 新コンフィグ）", "sample loopback 2", "M-405", "input/new/rt-a01.txt の "} {
		if !strings.Contains(h, want) {
			t.Errorf("the report lacks %q", want)
		}
	}
	for _, v := range secretValues(e, c.A.Data) {
		if strings.Contains(h, v) {
			t.Errorf("the report shows the secret %q", v)
		}
	}

	// An interface that is not carried over leaves the comparison.
	setCell(t, e, "IF 対応", "旧 IF 名", "Loopback0", "新 IF 名", "-")
	c, err = e.Compare("new", "old", at)
	if err != nil {
		t.Fatal(err)
	}
	if a := c.B.Applied; a == nil || len(a.Dropped) == 0 {
		t.Fatalf("nothing was left out: %+v", c.B.Applied)
	}
	found := false
	for _, d := range c.Result.OnlyA {
		if d.Table == "interfaces" && d.Row.Get("description") == "sample loopback 2" {
			found = true
		}
	}
	if !found || len(c.Result.OnlyB) != 0 {
		t.Errorf("new and old after the edit: only A %d, only B %d", len(c.Result.OnlyA), len(c.Result.OnlyB))
	}
	if h := read(t, e, c.Report.HTML); !strings.Contains(h, "対応のメモ") || !strings.Contains(h, "新 IF 名が - の IF の行です") {
		t.Error("the report does not tell what the mapping left out")
	}

	for _, pair := range [][2]string{{"old", "old"}, {"old", "logs"}, {"", "new"}} {
		if _, err := e.Compare(pair[0], pair[1], at); err == nil {
			t.Errorf("no error for %v", pair)
		}
	}
}

func TestMakeBook(t *testing.T) {
	text := sample(t, "cisco_ios")
	e := newCase(t, map[string]string{"input/old/rt-a01.txt": text, "input/new/rt-a01.txt": text})
	if _, err := e.MakeBook("logs", false, at); err == nil {
		t.Error("no error for --from logs")
	}
	if _, err := e.Compare("book", "new", at); err == nil {
		t.Error("no error for a book that is not there")
	}

	// Without a book: the mapping is proposed and put on in one go.
	o, err := e.MakeBook("old", false, at)
	if err != nil {
		t.Fatal(err)
	}
	s := o.Summary
	if !s.Created || s.Rows() == 0 || s.DeviceRows != 1 || s.IfRows == 0 || o.Backup != "" {
		t.Errorf("summary = %+v, backup %q", s, o.Backup)
	}
	if len(o.Problems) != 0 {
		t.Errorf("the book has problems: %v", o.Problems)
	}
	if o.Findings() != 0 {
		t.Errorf("findings = %d", o.Findings())
	}
	rows := 0
	for _, tb := range o.Applied.Data.Tables {
		rows += len(tb.Rows)
	}
	if s.Rows() != rows {
		t.Errorf("%d rows written for %d rows of data", s.Rows(), rows)
	}
	for _, pair := range [][2]string{{"old", "book"}, {"book", "new"}} {
		c, err := e.Compare(pair[0], pair[1], at)
		if err != nil {
			t.Fatal(err)
		}
		if !c.Result.Equal() || c.Result.Rows != rows {
			r := c.Result
			t.Errorf("%v: only A %d, only B %d, cells %d, rows %d", pair, len(r.OnlyA), len(r.OnlyB), len(r.Cells), r.Rows)
			for i, d := range r.Cells {
				if i < 5 {
					t.Logf("  %s %s %s: %q / %q", d.Table, d.ARow.KeyText(), d.Column, d.A.Text(), d.B.Text())
				}
			}
		}
	}

	// Again: every row is there already.
	o, err = e.MakeBook("old", false, at)
	if err != nil {
		t.Fatal(err)
	}
	if o.Summary.Changed() || o.Backup != "" {
		t.Errorf("the second run changed the book: %+v", o.Summary)
	}

	// With replace the sheets of the tables are written anew, after a copy.
	o, err = e.MakeBook("new", true, at)
	if err != nil {
		t.Fatal(err)
	}
	if !o.Summary.Replaced || o.Summary.Rows() != rows || o.Backup == "" || o.Applied != nil {
		t.Errorf("replace: %+v, backup %q", o.Summary, o.Backup)
	}
	if _, err := os.Stat(e.Proj.Path(o.Backup)); err != nil {
		t.Errorf("the copy is not there: %v", err)
	}
	_, bk, err := e.readBook()
	if err != nil {
		t.Fatal(err)
	}
	if len(bk.Devices) != 1 || len(bk.Ifs) == 0 {
		t.Errorf("replace emptied the mapping: %d devices, %d interfaces", len(bk.Devices), len(bk.Ifs))
	}
}

// A book made from the new side needs no mapping.
func TestMakeBookFromNew(t *testing.T) {
	e := newCase(t, map[string]string{"input/new/ix-a01.txt": sample(t, "nec_ix")})
	o, err := e.MakeBook("new", false, at)
	if err != nil {
		t.Fatal(err)
	}
	if !o.Summary.Created || o.Summary.Rows() == 0 || o.Summary.DeviceRows != 0 || o.Applied != nil {
		t.Errorf("summary = %+v", o.Summary)
	}
	c, err := e.Compare("new", "book", at)
	if err != nil {
		t.Fatal(err)
	}
	if !c.Result.Equal() {
		r := c.Result
		t.Errorf("new and book: only A %d, only B %d, cells %d", len(r.OnlyA), len(r.OnlyB), len(r.Cells))
	}
	// Nothing differs, but the lines that were not taken in are still open.
	if n := c.A.Source.Count().Unknown; n == 0 || c.Findings() != n {
		t.Errorf("findings = %d for %d unknown lines", c.Findings(), n)
	}
	if c.B.Book == nil || c.A.Source == nil {
		t.Error("the sides do not tell where they came from")
	}
}
