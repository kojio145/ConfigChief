package ui

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/book"
)

func TestEditPrefsAreKeptWithTheSettings(t *testing.T) {
	s, dir := newSession(t, nil)
	if got := s.EditPrefs(); len(got) != 0 {
		t.Fatalf("prefs = %+v", got)
	}
	s.EditPref("interfaces", GridPref{Hidden: []string{"mtu", "description", "mtu", ""}, Widths: map[string]int{"name": 240, "device": 5}})
	s.EditPref("users", GridPref{Hidden: []string{"role"}})
	s.EditPref("users", GridPref{}) // nothing to keep takes the sheet out
	want := map[string]GridPref{"interfaces": {Hidden: []string{"description", "mtu"}, Widths: map[string]int{"name": 240}}}
	if got := s.EditPrefs(); !reflect.DeepEqual(got, want) {
		t.Errorf("prefs = %+v", got)
	}
	noNull(t, "the prefs", s.EditPrefs())
	// The next start has them; the case folder has nothing of them.
	again := New("1.2.3", s.settings)
	if got := again.EditPrefs(); !reflect.DeepEqual(got, want) {
		t.Errorf("prefs after a start = %+v", got)
	}
	text, err := os.ReadFile(s.settings)
	if err != nil || !strings.Contains(string(text), `"grid"`) {
		t.Errorf("settings = %s, %v", text, err)
	}
	for _, name := range []string{"project.yaml"} {
		if data, err := os.ReadFile(dir + "/" + name); err != nil || strings.Contains(string(data), "grid") {
			t.Errorf("%s = %s, %v", name, data, err)
		}
	}
	// The recent cases stay beside them.
	if st := again.State(); len(st.Recent) != 1 {
		t.Errorf("recent = %+v", st.Recent)
	}
}

func TestEditColumnsFindAndReplaceOnScreen(t *testing.T) {
	s, _ := newSession(t, nil)
	if _, err := s.NewBook(); err != nil {
		t.Fatal(err)
	}
	if key, err := s.EditColumn("users", "担当 者", []string{"機器", "ユーザ名"}); err != nil || key != "free:担当者" {
		t.Errorf("key = %q, %v", key, err)
	}
	if _, err := s.EditColumn("users", "name", nil); err == nil {
		t.Error("a name of the definition was taken as a free column")
	}
	if _, err := s.EditDraft(book.DraftSheet{Sheet: "users",
		Added:   []book.DraftAdd{{Cells: map[string]string{"device": "rt-a01", "name": "sample-admin", "secret": "sample-pass-1", "free:担当者": "sample owner"}}},
		Columns: []book.DraftCol{{Op: "add", Title: "担当者"}}}); err != nil {
		t.Fatal(err)
	}
	v, err := s.EditSheet("users")
	if err != nil || len(v.Rows) != 1 {
		t.Fatalf("view = %+v, %v", v, err)
	}
	text := noNull(t, "the sheet with a column of the draft", v)
	if !strings.Contains(text, `"added":true`) {
		t.Errorf("no added column in %s", text)
	}
	found, err := s.EditFind("SAMPLE", book.FindOptions{})
	if err != nil || len(found) != 1 || found[0].Count != 2 {
		t.Fatalf("found = %+v, %v", found, err)
	}
	noNull(t, "what was found", found)
	if none, err := s.EditFind("no such text", book.FindOptions{}); err != nil || none == nil || len(none) != 0 {
		t.Errorf("found = %+v, %v", none, err)
	}
	out, err := s.EditReplace("sample", "example", book.FindOptions{Secrets: true}, "")
	if err != nil || len(out.After) != 1 || out.Counts[0].Count != 3 || out.Draft != 2 {
		t.Fatalf("out = %+v, %v", out, err)
	}
	saved, err := s.EditSave(nil)
	if err != nil || !saved.Saved || !strings.Contains(saved.Message, "追加 1 行") {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	v, _ = s.EditSheet("users")
	got := strings.Join(v.Rows[0].Cells, "|")
	if !strings.Contains(got, "example-admin") || !strings.Contains(got, "example-pass-1") || !strings.Contains(got, "example owner") || v.Columns[len(v.Columns)-1].Title != "担当者" {
		t.Errorf("row = %s, columns = %+v", got, v.Columns[len(v.Columns)-1])
	}
}
