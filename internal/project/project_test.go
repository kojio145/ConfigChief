package project

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"golang.org/x/text/encoding/japanese"
)

var day = time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

func TestInitAndOpen(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sample-case")
	p, err := Init(dir, "見本の案件: 拠点ルータ", "")
	if err != nil {
		t.Fatal(err)
	}
	if p.Config.Name != "見本の案件: 拠点ルータ" || p.Config.Kind != "migration" || p.Config.Book != "parameters.xlsx" ||
		p.Config.Language != "ja" || !p.Config.Secrets.MaskInReports || p.Config.Output.Filename != "{{ device.hostname }}.txt" {
		t.Errorf("config = %+v", p.Config)
	}
	// Only what is filled first is made (spec 14.15.5).
	if entries, _ := os.ReadDir(dir); len(entries) != 2 {
		t.Errorf("the case holds %d entries, want project.yaml and input", len(entries))
	}
	if st, err := os.Stat(p.Path("input/old")); err != nil || !st.IsDir() {
		t.Error("input/old is not there")
	}
	if entries, _ := os.ReadDir(p.Path("input")); len(entries) != 1 {
		t.Errorf("input holds %d folders, want old only", len(entries))
	}
	if _, err := Init(dir, "", ""); err == nil || !strings.Contains(err.Error(), "すでに") {
		t.Errorf("second Init: err = %v", err)
	}
	if _, err := Init(filepath.Join(t.TempDir(), "x"), "", "renewal"); err == nil {
		t.Error("no error for an unknown kind")
	}

	// Without a name the folder gives it.
	q, err := Init(filepath.Join(t.TempDir(), "sample-two"), "", "new")
	if err != nil {
		t.Fatal(err)
	}
	if q.Config.Name != "sample-two" || q.Config.Kind != "new" {
		t.Errorf("config = %+v", q.Config)
	}
	// A new build has nothing to put first: project.yaml only.
	if entries, _ := os.ReadDir(q.Dir); len(entries) != 1 {
		t.Errorf("a new build holds %d entries, want project.yaml only", len(entries))
	}
}

func TestOpenErrors(t *testing.T) {
	if _, err := Open(t.TempDir()); err == nil || !strings.Contains(err.Error(), "案件フォルダではありません") {
		t.Errorf("err = %v", err)
	}
	cases := []struct{ name, yaml, want string }{
		{"version", "version: 2\n", "version は 1"},
		{"kind", "version: 1\nkind: renewal\n", "kind「renewal」"},
		{"book in a folder", "version: 1\nbook: sub/parameters.xlsx\n", "直下の .xlsx"},
		{"book xls", "version: 1\nbook: parameters.xls\n", "直下の .xlsx"},
		{"language", "version: 1\nlanguage: fr\n", "language「fr」"},
		{"newline", "version: 1\noutput: {newline: cr}\n", "output.newline「cr」"},
		{"unknown field", "version: 1\ncolour: red\n", "colour"},
	}
	for _, c := range cases {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, ConfigFile), []byte(c.yaml), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := Open(dir); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want %q", c.name, err, c.want)
		}
	}
}

func TestInputs(t *testing.T) {
	p, err := Init(t.TempDir(), "sample", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"rt-b01.txt", "rt-a01.log", ".hidden", "~$open.txt"} {
		if err := os.WriteFile(p.Path("input/old/"+name), []byte("x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(p.Path("input/old/sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := p.Inputs("old")
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"input/old/rt-a01.log", "input/old/rt-b01.txt"}; !reflect.DeepEqual(got, want) {
		t.Errorf("inputs = %v, want %v", got, want)
	}
	if got, err := p.Inputs("logs"); err != nil || len(got) != 0 {
		t.Errorf("logs = %v, %v", got, err)
	}
}

func TestDecode(t *testing.T) {
	sjis, err := japanese.ShiftJIS.NewEncoder().Bytes([]byte("description 見本\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		name string
		in   []byte
		want string
	}{
		{"utf-8", []byte("description 見本\n"), "description 見本\n"},
		{"utf-8 with a mark", append([]byte{0xEF, 0xBB, 0xBF}, "a\r\nb\r\n"...), "a\nb\n"},
		{"shift_jis", sjis, "description 見本\n"},
		{"utf-16 little endian", []byte{0xFF, 0xFE, 'a', 0, '\r', 0, '\n', 0}, "a\n"},
		{"utf-16 big endian", []byte{0xFE, 0xFF, 0, 'a', 0, '\n'}, "a\n"},
		{"old mac line ends", []byte("a\rb\r"), "a\nb\n"},
		{"empty", nil, ""},
	}
	for _, c := range cases {
		if got := Decode(c.in); got != c.want {
			t.Errorf("%s: %q, want %q", c.name, got, c.want)
		}
	}
}

func TestSources(t *testing.T) {
	p, err := Init(t.TempDir(), "sample", "")
	if err != nil {
		t.Fatal(err)
	}
	s, err := p.Sources()
	if err != nil {
		t.Fatal(err)
	}
	if s.Pack("input/old/rt-a01.txt") != "" {
		t.Error("a choice in an empty state")
	}
	s.Choose("input/old/rt-b01.txt", "nec_ix")
	s.Choose("input/old/rt-a01.txt", "cisco_ios")
	if err := p.SaveSources(s); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(p.Path("state/sources.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	want := "sources:\n  - file: input/old/rt-a01.txt\n    pack: cisco_ios\n  - file: input/old/rt-b01.txt\n    pack: nec_ix\n"
	if string(data) != want {
		t.Errorf("sources.yaml:\n%s\nwant:\n%s", data, want)
	}
	s, err = p.Sources()
	if err != nil {
		t.Fatal(err)
	}
	if s.Pack("input/old/rt-b01.txt") != "nec_ix" {
		t.Errorf("pack = %q", s.Pack("input/old/rt-b01.txt"))
	}
}

func TestBackupAndWriteBook(t *testing.T) {
	p, err := Init(t.TempDir(), "sample", "")
	if err != nil {
		t.Fatal(err)
	}
	if rel, err := p.Backup(day); err != nil || rel != "" {
		t.Errorf("backup without a book = %q, %v", rel, err)
	}
	if err := p.WriteBook([]byte("first")); err != nil {
		t.Fatal(err)
	}
	rel, err := p.Backup(day)
	if err != nil || rel != "state/backup/parameters_2026-01-02_030405.xlsx" {
		t.Fatalf("backup = %q, %v", rel, err)
	}
	if err := p.WriteBook([]byte("second")); err != nil {
		t.Fatal(err)
	}
	if got, _ := os.ReadFile(p.Path(rel)); string(got) != "first" {
		t.Errorf("the copy holds %q", got)
	}
	if got, _ := os.ReadFile(p.BookPath()); string(got) != "second" {
		t.Errorf("the book holds %q", got)
	}
	// A second copy in the same second does not replace the first.
	rel2, err := p.Backup(day)
	if err != nil || rel2 != "state/backup/parameters_2026-01-02_030405_2.xlsx" {
		t.Fatalf("second backup = %q, %v", rel2, err)
	}
	if got, _ := os.ReadFile(p.Path(rel)); string(got) != "first" {
		t.Errorf("the first copy now holds %q", got)
	}
	if got, _ := os.ReadFile(p.Path(rel2)); string(got) != "second" {
		t.Errorf("the second copy holds %q", got)
	}

	p.Log(day, "取り込み %d 台", 2)
	p.Log(day, "二行目\nは一行に")
	if got, _ := os.ReadFile(p.Path("state/log.txt")); string(got) != "2026-01-02 03:04:05 取り込み 2 台\n2026-01-02 03:04:05 二行目 は一行に\n" {
		t.Errorf("log = %q", got)
	}
}
