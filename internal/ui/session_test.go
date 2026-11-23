package ui

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kojio145/configchief/internal/packs"
)

var at = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

// sample gives the text of the import test of a bundled pack.
func sample(t *testing.T, id string) string {
	t.Helper()
	fsys, err := packs.FS(id)
	if err != nil {
		t.Fatal(err)
	}
	data, err := fs.ReadFile(fsys, "tests/import/basic/input.txt")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// newSession starts a session with a case that holds the given files (path
// from the case folder → text).
func newSession(t *testing.T, files map[string]string) (*Session, string) {
	t.Helper()
	s := New("1.2.3", filepath.Join(t.TempDir(), "settings", "settings.json"))
	s.now = func() time.Time { return at }
	dir := filepath.Join(t.TempDir(), "case")
	if _, err := s.Create(dir, "sample", "migration"); err != nil {
		t.Fatal(err)
	}
	for rel, text := range files {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(rel)), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return s, dir
}

// noNull tells when a view gives the page a null where it expects a list.
func noNull(t *testing.T, name string, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "null") {
		t.Errorf("%s holds a null: %s", name, data)
	}
	return string(data)
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func TestCases(t *testing.T) {
	settings := filepath.Join(t.TempDir(), "deep", "settings.json")
	s := New("1.2.3", settings)
	if st := s.State(); st.Case != nil || len(st.Recent) != 0 || st.Version != "1.2.3" {
		t.Fatalf("state at the start = %+v", st)
	}
	if _, err := s.Home(); err == nil || !strings.Contains(err.Error(), "案件を開いていません") {
		t.Errorf("err = %v", err)
	}
	if _, err := s.Create(" ", "", ""); err == nil || !strings.Contains(err.Error(), "フォルダを選んでください") {
		t.Errorf("err = %v", err)
	}

	base := t.TempDir()
	a := filepath.Join(base, "site-a")
	h, err := s.Create(a, "", "new")
	if err != nil {
		t.Fatal(err)
	}
	if h.Case.Name != "site-a" || h.Case.Kind != "新規" || h.Case.Book != "parameters.xlsx" || h.Old != 0 || h.New != 0 || h.HasBook || h.LastOut != "" {
		t.Errorf("home = %+v", h)
	}
	if len(h.Packs) != 3 || h.Packs[0].ID != "cisco_ios" || h.Packs[0].Version == "" {
		t.Errorf("packs = %+v", h.Packs)
	}
	noNull(t, "home", h)
	if _, err := s.Create(a, "", ""); err == nil || !strings.Contains(err.Error(), "すでに") {
		t.Errorf("making a case folder twice: %v", err)
	}
	b := filepath.Join(base, "site-b")
	if _, err := s.Create(b, "拠点 B の更改", ""); err != nil {
		t.Fatal(err)
	}
	st := s.State()
	if st.Case == nil || st.Case.Name != "拠点 B の更改" || st.Case.Kind != "更改" {
		t.Errorf("case = %+v", st.Case)
	}
	if len(st.Recent) != 2 || st.Recent[0].Dir != b || st.Recent[1].Dir != a || st.Recent[1].Name != "site-a" {
		t.Errorf("recent = %+v", st.Recent)
	}
	// Opening a case again moves it to the top; it is not listed twice.
	if _, err := s.Open(a); err != nil {
		t.Fatal(err)
	}
	if st = s.State(); len(st.Recent) != 2 || st.Recent[0].Dir != a {
		t.Errorf("recent = %+v", st.Recent)
	}
	if _, err := s.Open(base); err == nil || !strings.Contains(err.Error(), "案件フォルダではありません") {
		t.Errorf("err = %v", err)
	}
	if st = s.State(); st.Case == nil || st.Case.Dir != a {
		t.Errorf("a folder that could not be opened closed the case: %+v", st.Case)
	}

	// Another window finds the list; a folder that went away is marked.
	if err := os.RemoveAll(b); err != nil {
		t.Fatal(err)
	}
	s2 := New("1.2.3", settings)
	st = s2.State()
	if st.Case != nil || len(st.Recent) != 2 || st.Recent[0].Missing || !st.Recent[1].Missing {
		t.Errorf("state of a new window = %+v", st)
	}
	if st = s2.Forget(b); len(st.Recent) != 1 || st.Recent[0].Dir != a {
		t.Errorf("recent = %+v", st.Recent)
	}
	if st = New("1.2.3", settings).State(); len(st.Recent) != 1 {
		t.Errorf("recent after forgetting = %+v", st.Recent)
	}
	if st = s.Close(); st.Case != nil {
		t.Errorf("case after closing = %+v", st.Case)
	}

	// No more than ten are kept, and broken settings are as good as none.
	for i := 0; i < 12; i++ {
		if _, err := s2.Create(filepath.Join(base, fmt.Sprintf("more-%02d", i)), "", ""); err != nil {
			t.Fatal(err)
		}
	}
	if st = s2.State(); len(st.Recent) != maxRecent || filepath.Base(st.Recent[0].Dir) != "more-11" {
		t.Errorf("recent = %d, first %+v", len(st.Recent), st.Recent[0])
	}
	if err := os.WriteFile(settings, []byte("{broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if st = New("1.2.3", settings).State(); len(st.Recent) != 0 {
		t.Errorf("recent from broken settings = %+v", st.Recent)
	}
	if st = New("1.2.3", "").State(); len(st.Recent) != 0 {
		t.Errorf("recent without settings = %+v", st.Recent)
	}
}

func TestImport(t *testing.T) {
	s, dir := newSession(t, map[string]string{
		"input/old/a_cisco.txt": sample(t, "cisco_ios"),
		"input/old/d_short.txt": "hostname sample-short\n", // too short to tell the vendor from
	})
	// The reports of this case do not mask; the screen still does.
	cfg := filepath.Join(dir, "project.yaml")
	data, err := os.ReadFile(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte(strings.Replace(string(data), "mask_in_reports: true", "mask_in_reports: false", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Lines("old", "input/old/a_cisco.txt"); err == nil || !strings.Contains(err.Error(), "先に取り込んでください") {
		t.Errorf("err = %v", err)
	}
	if _, err := s.Import("logs"); err == nil {
		t.Error("no error for a source that cannot be imported")
	}
	if _, err := s.Import("new"); err == nil || !strings.Contains(err.Error(), "input/new にファイルがありません") {
		t.Errorf("err = %v", err)
	}

	v, err := s.Import("old")
	if err != nil {
		t.Fatal(err)
	}
	all := noNull(t, "import", v)
	if v.Source != "old" || v.Title != "取り込みの結果（旧コンフィグ）" || len(v.Files) != 2 || len(v.Counts) < 6 || v.Findings == 0 {
		t.Fatalf("import = %+v", v)
	}
	a, d := v.Files[0], v.Files[1]
	if !a.Imported || a.Device != "rt-a01" || a.Pack != "cisco_ios" || a.Version == "" || a.Chosen || a.NeedsPack || a.Total == 0 || a.Total != a.Taken+a.Ignored+a.Blank+a.Unknown || a.Ratio == "" {
		t.Errorf("the file that was read = %+v", a)
	}
	if d.Imported || !d.NeedsPack || len(d.Candidates) != 3 || !strings.Contains(d.Err, "メーカーを判定できません") || d.Mark != "warn" {
		t.Errorf("the file that was not read = %+v", d)
	}
	var names []string
	for _, tb := range v.Tables {
		names = append(names, tb.Name)
		if len(tb.Marks) != len(tb.Rows) {
			t.Errorf("%s: %d marks for %d rows", tb.Name, len(tb.Marks), len(tb.Rows))
		}
	}
	if strings.Join(names, ",") != "読めなかった行の形,読めなかった行,新旧対照表で決まっていないところ,注意" {
		t.Errorf("lists = %v", names)
	}
	if len(v.Messages) != 1 || !strings.Contains(v.Messages[0], "parameters.xlsx を作りました") {
		t.Errorf("messages = %v", v.Messages)
	}
	if v.Report.XLSX != "out/2026-01-02_030405/reports/import_old.xlsx" || !exists(filepath.Join(dir, filepath.FromSlash(v.Report.HTML))) {
		t.Errorf("report = %+v", v.Report)
	}

	lines, err := s.Lines("old", "input/old/a_cisco.txt")
	if err != nil {
		t.Fatal(err)
	}
	all += noNull(t, "lines", lines)
	if lines.Device != "rt-a01" || len(lines.Lines) < a.Total {
		t.Fatalf("lines = %d of %s, the file has %d", len(lines.Lines), lines.Device, a.Total)
	}
	masked, made, blank := 0, 0, 0
	for _, l := range lines.Lines {
		if strings.Contains(l.Text, "***") {
			masked++
		}
		made += len(l.Made)
		if l.Blank {
			blank++
			if l.Word != "無視（コメント・空行など）" {
				t.Errorf("line %d: %q", l.No, l.Word)
			}
		} else if l.Word == "" {
			t.Errorf("line %d has no state in words", l.No)
		}
	}
	if masked < 3 || made == 0 || blank == 0 {
		t.Errorf("masked %d, values %d, blank %d", masked, made, blank)
	}
	// The report follows the case and shows the values; the screen holds
	// none of the words the report shows where the screen masks.
	html, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(v.Report.HTML)))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(html), "***") {
		t.Error("the report masks though the case says not to")
	}
	for _, secret := range []string{"dummy-ro-comm", "dummy-bgp-key"} {
		if !strings.Contains(string(html), secret) {
			t.Fatalf("the sample does not hold %q any more: pick another value for this test", secret)
		}
		if strings.Contains(all, secret) {
			t.Errorf("the screen shows the secret %q", secret)
		}
	}
	if _, err := s.Lines("old", "input/old/d_short.txt"); err == nil || !strings.Contains(err.Error(), "取り込めていません") {
		t.Errorf("err = %v", err)
	}
	if _, err := s.Lines("old", "input/old/none.txt"); err == nil {
		t.Error("no error for a file that is not there")
	}

	// A person chooses the pack of the file the tool could not tell.
	if _, err := s.ChoosePack("old", "input/old/d_short.txt", "sample_os"); err == nil {
		t.Error("no error for a pack that is not there")
	}
	v, err = s.ChoosePack("old", "input/old/d_short.txt", "nec_ix")
	if err != nil {
		t.Fatal(err)
	}
	if d = v.Files[1]; !d.Imported || !d.Chosen || d.Pack != "nec_ix" || d.NeedsPack {
		t.Errorf("the file after the choice = %+v", d)
	}
	if _, err := s.Lines("old", "input/old/d_short.txt"); err != nil {
		t.Error(err)
	}

	h, err := s.Home()
	if err != nil {
		t.Fatal(err)
	}
	if h.Old != 2 || !h.HasBook || h.BookTime == "" || h.LastOut != "out/2026-01-02_030405" {
		t.Errorf("home = %+v", h)
	}
}

func TestCancel(t *testing.T) {
	s, dir := newSession(t, map[string]string{
		"input/old/a_cisco.txt": sample(t, "cisco_ios"),
		"input/old/b_nec.txt":   sample(t, "nec_ix"),
	})
	var seen []string
	s.Progress = func(done, total int, file string) {
		seen = append(seen, fmt.Sprintf("%d/%d %s", done, total, file))
		s.Cancel() // the user presses the button while the first file is read
	}
	if _, err := s.Import("old"); err != errStopped {
		t.Fatalf("err = %v", err)
	}
	if strings.Join(seen, ";") != "0/2 input/old/a_cisco.txt" {
		t.Errorf("progress = %v", seen)
	}
	entries, _ := os.ReadDir(filepath.Join(dir, "out"))
	if exists(filepath.Join(dir, "parameters.xlsx")) || len(entries) != 0 {
		t.Error("an import that was stopped wrote something")
	}
	if _, err := s.Lines("old", "input/old/a_cisco.txt"); err == nil {
		t.Error("an import that was stopped left lines to show")
	}
	// The next step runs again.
	s.Progress = nil
	if _, err := s.Import("old"); err != nil {
		t.Fatal(err)
	}
}

func TestMappingBookCompare(t *testing.T) {
	s, dir := newSession(t, map[string]string{
		"input/old/rt-a01.txt": sample(t, "cisco_ios"),
		"input/new/ix-a01.txt": sample(t, "nec_ix"),
	})
	m, err := s.Mapping()
	if err != nil {
		t.Fatal(err)
	}
	noNull(t, "mapping without a book", m)
	if m.HasBook || len(m.Devices.Rows) != 0 || len(m.Devices.Header) != 8 || len(m.Ifs.Header) != 5 {
		t.Errorf("mapping without a book = %+v", m)
	}
	if _, err := s.Compare("old", "new"); err == nil || !strings.Contains(err.Error(), "parameters.xlsx がありません") {
		t.Errorf("err = %v", err)
	}
	if _, err := s.Import("old"); err != nil {
		t.Fatal(err)
	}
	if m, err = s.Mapping(); err != nil {
		t.Fatal(err)
	}
	noNull(t, "mapping", m)
	if !m.HasBook || len(m.Devices.Rows) != 1 || m.Devices.Rows[0][1] != "rt-a01" || m.Devices.Marks[0] != "unknown" {
		t.Errorf("devices = %+v", m.Devices)
	}
	if len(m.Open.Rows) != 1 || m.Open.Rows[0][0] != "M-407" || len(m.Counts) != 8 || m.Counts[1].Value != "1" {
		t.Errorf("open = %v, counts = %v", m.Open.Rows, m.Counts)
	}

	b, err := s.MakeBook("new", false)
	if err != nil {
		t.Fatal(err)
	}
	noNull(t, "book", b)
	if len(b.Messages) < 2 || !strings.Contains(b.Messages[0], "に足しました（新コンフィグ から") || !strings.Contains(b.Messages[0], "控え: state/backup/") {
		t.Errorf("messages = %v", b.Messages)
	}
	if b, err = s.MakeBook("old", true); err != nil {
		t.Fatal(err)
	}
	noNull(t, "book from old", b)
	if !strings.Contains(b.Messages[0], "設計データのシートを空にしてから書きました") || len(b.Open.Rows) == 0 || b.Findings == 0 {
		t.Errorf("book from old = %+v", b)
	}
	if _, err := s.MakeBook("book", false); err == nil {
		t.Error("no error for a source the book cannot be made from")
	}

	if _, err := s.Compare("new", "new"); err == nil || !strings.Contains(err.Error(), "同じもの") {
		t.Errorf("err = %v", err)
	}
	c, err := s.Compare("new", "book")
	if err != nil {
		t.Fatal(err)
	}
	noNull(t, "compare", c)
	if c.Title != "比較の結果（新コンフィグ と パラメータブック）" || len(c.Counts) != 5 || len(c.Tables) != 4 || c.Tables[0].Name != "A にだけある行" || len(c.Tables[0].Rows) == 0 {
		t.Errorf("compare = %+v", c)
	}
	if !exists(filepath.Join(dir, filepath.FromSlash(c.Report.XLSX))) {
		t.Errorf("report = %+v", c.Report)
	}
}

func TestDiff(t *testing.T) {
	s := New("", "")
	dir := t.TempDir()
	var a, b []string
	for i := 1; i <= 30; i++ {
		a = append(a, fmt.Sprintf("line %d", i))
		b = append(b, fmt.Sprintf("line %d", i))
	}
	b[14] = "line fifteen"
	pa, pb := filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")
	os.WriteFile(pa, []byte(strings.Join(a, "\n")+"\n"), 0o644)
	os.WriteFile(pb, []byte(strings.Join(b, "\r\n")), 0o644)

	v, err := s.Diff(pa, pa)
	if err != nil {
		t.Fatal(err)
	}
	noNull(t, "diff of a file with itself", v)
	if !v.Equal || len(v.Lines) != 0 {
		t.Errorf("a file against itself = %+v", v)
	}
	if v, err = s.Diff(pa, pb); err != nil {
		t.Fatal(err)
	}
	if v.Equal || v.Added != 1 || v.Removed != 1 {
		t.Errorf("diff = %+v", v)
	}
	var got []string
	for _, l := range v.Lines {
		got = append(got, fmt.Sprintf("%s %d %d %s", l.Op, l.A, l.B, l.Text))
	}
	want := []string{
		"gap 0 0 同じ行 11 行",
		"same 12 12 line 12", "same 13 13 line 13", "same 14 14 line 14",
		"del 15 0 line 15", "add 0 15 line fifteen",
		"same 16 16 line 16", "same 17 17 line 17", "same 18 18 line 18",
		"gap 0 0 同じ行 12 行",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("lines =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if _, err := s.Diff(pa, ""); err == nil {
		t.Error("no error for one file")
	}
	if _, err := s.Diff(pa, filepath.Join(dir, "none.txt")); err == nil {
		t.Error("no error for a file that is not there")
	}
}

func TestPath(t *testing.T) {
	s := New("", "")
	if _, err := s.Path(""); err == nil {
		t.Error("no error without a case")
	}
	s, dir := newSession(t, nil)
	for rel, want := range map[string]string{
		"":          dir,
		"input/old": filepath.Join(dir, "input", "old"),
		"state":     filepath.Join(dir, "state"),
	} {
		if got, err := s.Path(rel); err != nil || got != want {
			t.Errorf("Path(%q) = %q, %v; want %q", rel, got, err, want)
		}
	}
	// Of the files, only a book or a report is given.
	for _, name := range []string{"parameters.xlsx", "tool.exe", "project.bat"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if got, err := s.Path("parameters.xlsx"); err != nil || got != filepath.Join(dir, "parameters.xlsx") {
		t.Errorf("the book: %q, %v", got, err)
	}
	outside := filepath.Dir(dir)
	for _, rel := range []string{"..", "../x", "input/../../x", "/tmp", `input\old`, `C:\Windows`, outside, "report.html", "out/none", "tool.exe", "project.bat", "project.yaml"} {
		if got, err := s.Path(rel); err == nil {
			t.Errorf("Path(%q) = %q, want an error", rel, got)
		}
	}
}
