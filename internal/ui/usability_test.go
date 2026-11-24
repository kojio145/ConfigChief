package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/book"
)

// The way through a migration with less to do by hand (spec 14.10): files
// dropped, one import, the proposals of 新旧対照表（ポート） after the save of 新旧対照表（機器）,
// the models to choose, the result after the case is opened again, and
// findings approved together.
func TestUsability(t *testing.T) {
	s, dir := newSession(t, nil)
	src := filepath.Join(t.TempDir(), "rt-c01.txt")
	if err := writeTestFile(src, []byte(rendered(t, "cisco_ios")), 0o644); err != nil {
		t.Fatal(err)
	}
	if n := s.Next(); n == nil || n.Act != "add-old" {
		t.Fatalf("next = %+v", n)
	}
	a, err := s.AddFiles("old", []string{src}, false)
	if err != nil || a.Copied != 1 || a.Files != 1 || len(a.Clash) != 0 || !strings.Contains(a.Message, "input/old に 1 ファイル") {
		t.Fatalf("add = %+v, %v", a, err)
	}
	noNull(t, "the files added", a)
	if n := s.Next(); n == nil || n.Act != "import-run" {
		t.Fatalf("next = %+v", n)
	}
	if _, err := s.Import("old"); err != nil {
		t.Fatal(err)
	}
	if n := s.Next(); n == nil || n.Act != "edit-devices" {
		t.Fatalf("next = %+v", n)
	}

	// 新旧対照表（機器） offers the models of the packs.
	v, err := s.EditSheet("@devices")
	if err != nil {
		t.Fatal(err)
	}
	var model *book.EditColumn
	for i := range v.Columns {
		if v.Columns[i].Key == "new_model" {
			model = &v.Columns[i]
		}
	}
	if model == nil || len(model.Choices) == 0 || model.ChoiceNotes[model.Choices[0]] == "" {
		t.Fatalf("the column of the model = %+v", model)
	}

	// The save of 新旧対照表（機器） makes the rows of 新旧対照表（ポート） without a second import.
	row := v.Rows[0]
	if _, err := s.EditDraft(book.DraftSheet{Sheet: "@devices", Changed: []book.DraftRow{{Row: row.Row,
		Key: map[string]string{"old_host": "rt-c01"},
		Old: map[string]string{"new_host": "", "new_vendor": ""},
		New: map[string]string{"new_host": "rt-c01", "new_vendor": "cisco_ios"}}}}); err != nil {
		t.Fatal(err)
	}
	saved, err := s.EditSave(nil)
	if err != nil || !saved.Saved || !strings.Contains(saved.Message, "新旧対照表（ポート）に") {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	if said, err := s.ProposeIfs(); err != nil || !strings.Contains(said, "ありませんでした") {
		t.Errorf("proposals again = %q, %v", said, err)
	}
	if n := s.Next(); n == nil || n.Act != "edit-ifs" {
		t.Fatalf("next = %+v", n)
	}

	// The result is there after the case is opened again.
	if _, err := s.MakeBook("old", false); err != nil {
		t.Fatal(err)
	}
	if r, err := s.LastResult(); err != nil || r != nil {
		t.Fatalf("a result before any generation: %+v, %v", r, err)
	}
	if _, err := s.Generate(); err != nil {
		t.Fatal(err)
	}
	s.Close()
	if _, err := s.Open(dir); err != nil {
		t.Fatal(err)
	}
	r, err := s.LastResult()
	if err != nil || r == nil {
		t.Fatalf("last result = %v", err)
	}
	noNull(t, "the last result", r)
	d := deviceOf(r, "rt-c01")
	if r.Dir == "" || !r.Configs || d.File == "" || !strings.Contains(r.Title, "前回の結果") || r.Notice != "" {
		t.Fatalf("last result = %+v, device %+v", r, d)
	}
	text, err := s.ReadFile(d.File)
	if err != nil || len(text.Lines) == 0 || !strings.HasPrefix(text.Lines[0], "hostname") && !strings.Contains(strings.Join(text.Lines, "\n"), "hostname rt-c01") {
		t.Errorf("the configuration = %+v, %v", text, err)
	}

	// Findings picked are approved together, one approval each.
	var keys []string
	for _, f := range r.Findings {
		if f.State == "open" {
			keys = append(keys, f.Key)
		}
	}
	if len(keys) > 0 {
		after, err := s.ApproveMany(append(keys, "nothing"), "まとめて確かめた", "sample-user")
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(after.Message, "件を承認しました") || !strings.Contains(after.Message, "1 件は承認できませんでした") {
			t.Errorf("message = %s", after.Message)
		}
		for _, f := range after.Findings {
			if f.State == "open" {
				t.Errorf("left open: %+v", f)
			}
		}
	}
	if _, err := s.ApproveMany(nil, "x", "y"); err == nil {
		t.Error("nothing picked was taken")
	}
	if !s.BookWritable() {
		t.Error("the book cannot be written")
	}
}

// Warnings and errors picked in the list are approved together, one
// approval each (spec 14.10.8).
func TestApproveMany(t *testing.T) {
	s, _ := newSession(t, map[string]string{
		"input/new/rt-c01.txt": rendered(t, "cisco_ios"),
		"input/new/ix-c01.txt": rendered(t, "nec_ix"),
	})
	if _, err := s.MakeBook("new", false); err != nil {
		t.Fatal(err)
	}
	if _, err := s.ApproveMany([]string{"x"}, "r", "b"); err == nil {
		t.Error("approved without a result")
	}
	v, err := s.Generate()
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	fix := ""
	for _, f := range v.Findings {
		switch f.State {
		case "open":
			keys = append(keys, f.Key)
		case "fix":
			fix = f.Key
		}
	}
	if len(keys) < 2 || fix == "" {
		t.Fatalf("open %d, fix %q", len(keys), fix)
	}
	after, err := s.ApproveMany(append(keys, fix), "まとめて確かめた", "sample-user")
	if err != nil {
		t.Fatal(err)
	}
	noNull(t, "after the approvals", after)
	if !strings.Contains(after.Message, "件を承認しました") || !strings.Contains(after.Message, "1 件は承認できませんでした") || after.Notice == "" {
		t.Errorf("message = %s / %s", after.Message, after.Notice)
	}
	if len(after.Approvals) != len(keys) {
		t.Errorf("approvals = %d, want %d", len(after.Approvals), len(keys))
	}
	for _, f := range after.Findings {
		if f.State == "open" {
			t.Errorf("left open: %+v", f)
		}
	}
	if s.name != "sample-user" {
		t.Errorf("the name is not kept: %q", s.name)
	}
}
