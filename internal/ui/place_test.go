package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// placeSession starts a session whose place next to the exe is a folder of
// the test (spec 14.15.1).
func placeSession(t *testing.T) (*Session, string, string) {
	t.Helper()
	settings := filepath.Join(t.TempDir(), "settings", "settings.json")
	s := New("1.2.3", settings)
	s.now = func() time.Time { return at }
	s.base = filepath.Join(t.TempDir(), PlaceFolder)
	return s, s.base, settings
}

func TestDefaultPlace(t *testing.T) {
	exe, err := os.Executable()
	if err != nil {
		t.Skip(err)
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	if got := DefaultPlace(); got != filepath.Join(filepath.Dir(exe), "projects") {
		t.Errorf("default place %s for %s", got, exe)
	}
}

func TestNewCaseInPlace(t *testing.T) {
	s, place, _ := placeSession(t)
	if v := s.Place(); v.Dir != place || !v.Default || len(v.Cases) != 0 {
		t.Errorf("empty place: %+v", v)
	}
	h, err := s.NewCase("  拠点ルータ更改 ", "migration")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(place, "拠点ルータ更改")
	if !sameDir(h.Case.Dir, want) || h.Case.Name != "拠点ルータ更改" {
		t.Errorf("case %+v, want %s", h.Case, want)
	}
	for _, f := range []string{"project.yaml", "input/old"} {
		if _, err := os.Stat(filepath.Join(want, filepath.FromSlash(f))); err != nil {
			t.Errorf("%s: %v", f, err)
		}
	}
	for _, f := range []string{"out", "extra", "templates", "input/new"} {
		if _, err := os.Stat(filepath.Join(want, filepath.FromSlash(f))); err == nil {
			t.Errorf("%s is made before it is used", f)
		}
	}
	// Nothing is made right in the place.
	entries, _ := os.ReadDir(place)
	if len(entries) != 1 {
		t.Errorf("the place holds %d entries", len(entries))
	}
	// The same name again, and a name a folder cannot take.
	if _, err := s.NewCase("拠点ルータ更改", "new"); err == nil || !strings.Contains(err.Error(), "同じ名前のフォルダ") {
		t.Errorf("same name: %v", err)
	}
	if _, err := s.NewCase("a/b", "new"); err == nil || !strings.Contains(err.Error(), "「/」") {
		t.Errorf("bad name: %v", err)
	}
	// An empty folder of the name is taken.
	os.MkdirAll(filepath.Join(place, "空のフォルダ"), 0o755)
	if _, err := s.NewCase("空のフォルダ", "new"); err != nil {
		t.Errorf("empty folder: %v", err)
	}
	// The place lists its cases by name; the recent list leaves them out.
	os.MkdirAll(filepath.Join(place, "not a case"), 0o755)
	v := s.Place()
	if len(v.Cases) != 2 || v.Cases[0].Name != "拠点ルータ更改" || v.Cases[1].Name != "空のフォルダ" {
		t.Errorf("cases: %+v", v.Cases)
	}
	outside := filepath.Join(t.TempDir(), "outside")
	if _, err := s.Create(outside, "外の案件", "new"); err != nil {
		t.Fatal(err)
	}
	st := s.State()
	if len(st.Recent) != 1 || !sameDir(st.Recent[0].Dir, outside) {
		t.Errorf("recent: %+v", st.Recent)
	}
}

func TestSampleAndImportInPlace(t *testing.T) {
	s, place, _ := placeSession(t)
	for _, want := range []string{"sample", "sample_2"} {
		h, said, err := s.NewSample()
		if err != nil || said == "" {
			t.Fatalf("sample: %v", err)
		}
		if !sameDir(h.Case.Dir, filepath.Join(place, want)) || h.Old != 2 {
			t.Errorf("sample at %s (old %d), want %s", h.Case.Dir, h.Old, want)
		}
	}
	v, data, err := s.CaseExport(false, "")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), v.Name)
	writeTestFile(path, data, 0o644)
	got, err := s.ImportHere(path, "")
	if err != nil {
		t.Fatal(err)
	}
	if !sameDir(filepath.Dir(got.Dir), place) {
		t.Errorf("read back at %s, want under %s", got.Dir, place)
	}
}

func TestSetPlace(t *testing.T) {
	s, place, settings := placeSession(t)
	other := filepath.Join(t.TempDir(), "cases")
	v, err := s.SetPlace(other)
	if err != nil || !sameDir(v.Dir, other) || v.Default {
		t.Fatalf("set: %+v %v", v, err)
	}
	h, err := s.NewCase("A", "new")
	if err != nil || !sameDir(filepath.Dir(h.Case.Dir), other) {
		t.Fatalf("case in the place chosen: %v", err)
	}
	// A case folder is not a place.
	if _, err := s.SetPlace(h.Case.Dir); err == nil || !strings.Contains(err.Error(), "案件フォルダです") {
		t.Errorf("case folder as a place: %v", err)
	}
	// The place is kept for the next start.
	s.Close()
	s2 := New("1.2.3", settings)
	s2.base = place
	if v := s2.Place(); !sameDir(v.Dir, other) {
		t.Errorf("place after a start: %s", v.Dir)
	}
	if v, err := s2.SetPlace(""); err != nil || !v.Default || v.Dir != place {
		t.Errorf("back to the default: %+v %v", v, err)
	}
}

// A place that cannot be made is told plainly.
func TestPlaceNotWritable(t *testing.T) {
	s, _, _ := placeSession(t)
	file := filepath.Join(t.TempDir(), "a file")
	writeTestFile(file, []byte("x"), 0o644)
	s.base = filepath.Join(file, "projects")
	if _, err := s.NewCase("A", "new"); err == nil || !strings.Contains(err.Error(), "書けません") {
		t.Errorf("not writable: %v", err)
	}
}

func TestDeliveryNamesOnHome(t *testing.T) {
	s, _ := newSession(t, nil)
	h, err := s.Home()
	if err != nil || h.DeliveryNames != "ja" {
		t.Fatalf("home: %v %+v", err, h)
	}
	if h, err = s.SetDeliveryNames("en"); err != nil || h.DeliveryNames != "en" {
		t.Errorf("en: %v", err)
	}
	if _, err = s.SetDeliveryNames("xx"); err == nil {
		t.Error("a language that is not there")
	}
}

// A folder of the case is made when the page opens it (spec 14.15.5).
func TestOpenFolderMakesIt(t *testing.T) {
	s, dir := newSession(t, nil)
	got, err := s.Path("templates")
	if err != nil || got != filepath.Join(dir, "templates") {
		t.Fatalf("templates: %q %v", got, err)
	}
	if st, err := os.Stat(got); err != nil || !st.IsDir() {
		t.Error("templates was not made")
	}
	if _, err := s.Path("somewhere"); err == nil {
		t.Error("a folder that is not of a case was made")
	}
	if _, err := os.Stat(filepath.Join(dir, "somewhere")); err == nil {
		t.Error("somewhere is there")
	}
}

// Files are put from the page into each input folder of a case just made,
// the folder being made then (spec 14.10.1, 14.15.5).
func TestAddFilesMakesTheFolder(t *testing.T) {
	s, _, _ := placeSession(t)
	h, err := s.NewCase("足す見本", "new")
	if err != nil {
		t.Fatal(err)
	}
	text := filepath.Join(t.TempDir(), "rt-a01.txt")
	os.WriteFile(text, []byte("hostname rt-a01\n"), 0o644)
	csv := filepath.Join(t.TempDir(), "sites.csv")
	os.WriteFile(csv, []byte("site,host\nA,rt-a01\n"), 0o644)
	for _, source := range []string{"old", "new", "logs", "customer"} {
		if _, err := os.Stat(filepath.Join(h.Case.Dir, "input", source)); err == nil {
			t.Errorf("input/%s is there before it is used", source)
		}
		src := text
		if source == "customer" {
			src = csv // a list of another system: xlsx, xls or csv
		}
		v, err := s.AddFiles(source, []string{src}, false)
		if err != nil || v.Copied != 1 || v.Files != 1 {
			t.Errorf("%s: %+v %v", source, v, err)
		}
	}
}
