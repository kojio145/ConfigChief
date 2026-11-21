package ops

import (
	"io/fs"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/packs"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/termlog"
)

// sample gives the text of the import test of a bundled pack.
func sample(t *testing.T, id string) string {
	t.Helper()
	return sampleOf(t, id, "basic")
}

// sampleOf gives the text of one of the import tests of a bundled pack.
func sampleOf(t *testing.T, id, name string) string {
	t.Helper()
	fsys, err := packs.FS(id)
	if err != nil {
		t.Fatal(err)
	}
	data, err := fs.ReadFile(fsys, "tests/import/"+name+"/input.txt")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// newCase makes a case folder with the given files (path from the case
// folder → text) and opens it.
func newCase(t *testing.T, files map[string]string) *Env {
	t.Helper()
	dir := t.TempDir()
	p, err := project.Init(dir, "sample", "")
	if err != nil {
		t.Fatal(err)
	}
	for rel, text := range files {
		if err := os.WriteFile(p.Path(rel), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	e, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func TestOpen(t *testing.T) {
	e := newCase(t, nil)
	if want := []string{"cisco_ios", "nec_ix", "nec_qx"}; !reflect.DeepEqual(e.PackIDs, want) {
		t.Errorf("packs = %v, want %v", e.PackIDs, want)
	}
	if len(e.PackProblems) != 0 {
		t.Errorf("problems = %v", e.PackProblems)
	}
	if _, err := Open(t.TempDir()); err == nil {
		t.Error("no error for a folder that is not a case folder")
	}
}

func TestOpenCasePack(t *testing.T) {
	e := newCase(t, nil)
	// A pack of the case folder that cannot be read is told and left out.
	if err := os.MkdirAll(e.Proj.Path("packs/broken"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(e.Proj.Path("packs/broken/pack.yaml"), []byte("id: broken\ncolour: red\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	e, err := Open(e.Proj.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(e.PackProblems) != 1 || !strings.Contains(e.PackProblems[0], "packs/broken は使いません") {
		t.Errorf("problems = %v", e.PackProblems)
	}
	if e.Packs["broken"] != nil {
		t.Error("the broken pack was loaded")
	}
}

func TestImport(t *testing.T) {
	e := newCase(t, map[string]string{
		"input/old/a_cisco.txt": sample(t, "cisco_ios"),
		"input/old/b_nec.txt":   sample(t, "nec_ix"),
		"input/old/c_again.log": sample(t, "cisco_ios"),
		"input/old/d_short.txt": "hostname sample-short\n",
		"input/old/e_empty.txt": "\n\n",
	})
	src, err := e.Import("old", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(src.Files) != 5 {
		t.Fatalf("files = %d", len(src.Files))
	}
	// Two files have the host name rt-a01: until a person chooses one,
	// neither is taken (the tool does not pick one by itself).
	if got, want := src.Devices(), []string{"ix-a01"}; !reflect.DeepEqual(got, want) {
		t.Errorf("devices before the choice = %v, want %v", got, want)
	}
	both := []string{"input/old/a_cisco.txt", "input/old/c_again.log"}
	for _, i := range []int{0, 2} {
		f := src.Files[i]
		if f.Result != nil || f.Device != "rt-a01" || !reflect.DeepEqual(f.Same, both) || f.Left != "" ||
			!strings.Contains(f.Err, "ホスト名 rt-a01 のファイルが 2 つあります（input/old/a_cisco.txt、input/old/c_again.log）。どれを取り込むか選んでください") {
			t.Errorf("file %d before the choice = %+v", i, f)
		}
	}
	if err := e.ChooseFile("old", "input/old/b_nec.txt", at); err == nil || !strings.Contains(err.Error(), "重なっていません") {
		t.Errorf("choosing a file whose host name is its own: %v", err)
	}
	if err := e.ChooseFile("old", "input/old/nothing.txt", at); err == nil {
		t.Error("no error for a file that is not there")
	}
	// The later file is chosen: it is taken, and the first one is left out
	// without being a fault.
	if err := e.ChooseFile("old", "input/old/c_again.log", at); err != nil {
		t.Fatal(err)
	}
	src, err = e.Import("old", "")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := src.Devices(), []string{"ix-a01", "rt-a01"}; !reflect.DeepEqual(got, want) {
		t.Errorf("devices with the later file chosen = %v, want %v", got, want)
	}
	if f := src.Files[0]; f.Result != nil || f.Err != "" || !strings.Contains(f.Left, "ホスト名 rt-a01 は input/old/c_again.log を取り込みます") {
		t.Errorf("the file left out = %+v", f)
	}
	if src.File("rt-a01") != src.Files[2] {
		t.Error("the chosen file is not the one of the device")
	}
	// Chosen again: the first one.
	if err := e.ChooseFile("old", "input/old/a_cisco.txt", at); err != nil {
		t.Fatal(err)
	}
	src, err = e.Import("old", "")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := src.Devices(), []string{"rt-a01", "ix-a01"}; !reflect.DeepEqual(got, want) {
		t.Errorf("devices = %v, want %v", got, want)
	}
	a, b, c, d, empty := src.Files[0], src.Files[1], src.Files[2], src.Files[3], src.Files[4]
	if a.Pack != "cisco_ios" || a.Chosen || a.Err != "" || a.Result == nil {
		t.Errorf("a = %+v", a)
	}
	if b.Pack != "nec_ix" || b.Err != "" || b.Result == nil {
		t.Errorf("b = %+v", b)
	}
	if c.Result != nil || c.Err != "" || !strings.Contains(c.Left, "ホスト名 rt-a01 は input/old/a_cisco.txt を取り込みます") {
		t.Errorf("c = %+v", c)
	}
	if d.Result != nil || !strings.Contains(d.Err, "メーカーを判定できません（候補: cisco_ios 0 点、nec_ix 0 点、nec_qx 0 点）") {
		t.Errorf("d.Err = %q", d.Err)
	}
	if empty.Err != "中身が空です" {
		t.Errorf("empty.Err = %q", empty.Err)
	}

	// The rows of both devices are in one dataset.
	devs := src.Data.Tables["devices"]
	if devs == nil || len(devs.Rows) != 2 {
		t.Fatalf("devices rows = %+v", devs)
	}
	for i, want := range []string{"cisco_ios", "nec_ix"} {
		if got := devs.Rows[i].Get("vendor"); got != want {
			t.Errorf("vendor of row %d = %q, want %q", i, got, want)
		}
	}
	n := src.Count()
	ca, cb := a.Result.Count(), b.Result.Count()
	if n.Total != ca.Total+cb.Total || n.Unknown != ca.Unknown+cb.Unknown || n.Taken == 0 {
		t.Errorf("count = %+v", n)
	}
	if src.File("ix-a01") != b || src.File("rt-z99") != nil {
		t.Error("File does not find the file of a device")
	}

	// A chosen pack reads the file that could not be told, and the choice
	// stays for the next time.
	src, err = e.Import("old", "cisco_ios")
	if err != nil {
		t.Fatal(err)
	}
	d = src.Files[3]
	if d.Pack != "cisco_ios" || !d.Chosen || d.Result == nil || d.Result.Device != "sample-short" {
		t.Errorf("d with a chosen pack = %+v", d)
	}
	if src.Files[0].Chosen {
		t.Error("a file with a clear vendor is marked as chosen")
	}
	src, err = e.Import("old", "")
	if err != nil {
		t.Fatal(err)
	}
	if d = src.Files[3]; d.Pack != "cisco_ios" || !d.Chosen || d.Result == nil {
		t.Errorf("d the next time = %+v", d)
	}
	data, err := os.ReadFile(e.Proj.Path("state/sources.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if want := "sources:\n  - file: input/old/d_short.txt\n    pack: cisco_ios\nuse:\n  - source: old\n    device: rt-a01\n    file: input/old/a_cisco.txt\n"; string(data) != want {
		t.Errorf("sources.yaml:\n%s", data)
	}
}

func TestImportErrors(t *testing.T) {
	e := newCase(t, nil)
	if _, err := e.Import("logs", ""); err == nil {
		t.Error("no error for a source that cannot be imported")
	}
	if _, err := e.Import("old", "sample_os"); err == nil || !strings.Contains(err.Error(), "使えるパック: cisco_ios, nec_ix, nec_qx") {
		t.Errorf("err = %v", err)
	}
	src, err := e.Import("new", "")
	if err != nil || len(src.Files) != 0 {
		t.Errorf("empty source = %+v, %v", src, err)
	}
}

// Terminal logs: the vendor is told from the configuration in the log, the
// lines around it are counted apart, and what was done is told.
func TestImportLog(t *testing.T) {
	e := newCase(t, map[string]string{
		"input/old/a.log": sampleOf(t, "cisco_ios", "log"),
		"input/old/b.log": sampleOf(t, "nec_ix", "log"),
	})
	src, err := e.Import("old", "")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := src.Devices(), []string{"rt-b02", "ix-b02"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("devices = %v, want %v", got, want)
	}
	a, b := src.Files[0], src.Files[1]
	if a.Pack != "cisco_ios" || b.Pack != "nec_ix" {
		t.Errorf("packs = %s, %s", a.Pack, b.Pack)
	}
	for _, f := range src.Files {
		if len(f.Result.Notes) != 1 || !strings.Contains(f.Result.Notes[0], "端末ログとして読みました") {
			t.Errorf("%s: notes = %v", f.Path, f.Result.Notes)
		}
		first := f.Result.Lines[0]
		if !first.Outside || !first.Separator || first.Reason != termlog.ReasonOutside {
			t.Errorf("%s: line 1 = %+v", f.Path, first)
		}
	}
	// Marks in the lines of the configuration are a thing to look at; marks
	// on lines of their own are not.
	if w := a.Result.Warnings; len(w) != 1 || !strings.Contains(w[0], "ページ送りの跡と設定が同じ行にありました（25・34・41 行目）") {
		t.Errorf("a: warnings = %v", w)
	}
	if w := b.Result.Warnings; len(w) != 0 {
		t.Errorf("b: warnings = %v", w)
	}
	ca, cb := a.Result.Count(), b.Result.Count()
	if ca.Outside != 7 || cb.Outside != 6 {
		t.Errorf("outside = %d and %d, want 7 and 6", ca.Outside, cb.Outside)
	}
	if n := src.Count(); n.Outside != 13 || n.Separators != ca.Separators+cb.Separators {
		t.Errorf("count = %+v", n)
	}
}

// A log without the command is read as a whole, and that is told.
func TestImportLogWithoutCommand(t *testing.T) {
	text := sampleOf(t, "cisco_ios", "log")
	text = strings.Replace(text, "rt-b02#show running-config\n", "", 1)
	e := newCase(t, map[string]string{"input/old/a.log": text})
	src, err := e.Import("old", "cisco_ios")
	if err != nil {
		t.Fatal(err)
	}
	f := src.Files[0]
	if f.Result == nil {
		t.Fatalf("err = %s", f.Err)
	}
	found := false
	for _, w := range f.Result.Warnings {
		found = found || strings.Contains(w, "設定を表示するコマンドの行が見つかりません")
	}
	if !found || len(f.Result.Notes) != 0 || f.Result.Count().Outside != 0 {
		t.Errorf("warnings = %v, notes = %v", f.Result.Warnings, f.Result.Notes)
	}
}
