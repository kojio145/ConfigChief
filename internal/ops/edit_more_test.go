package ops

import (
	"bytes"
	"os"
	"testing"

	"github.com/xuri/excelize/v2"

	"github.com/kojio145/configchief/internal/book"
)

// withoutSheet takes a sheet out of the book of the case, as in a book that
// was made before the table was there.
func withoutSheet(t *testing.T, e *Env, name string) {
	t.Helper()
	path := e.Proj.Path(e.Proj.Config.Book)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := f.DeleteSheet(name); err != nil {
		t.Fatal(err)
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatal(err)
	}
	if err := writeTestFile(path, buf.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func entry(t *testing.T, e *Env, id string) EditEntry {
	t.Helper()
	list, err := e.EditSheets()
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range list.Sheets {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("no sheet %s in the list", id)
	return EditEntry{}
}

func TestEditMissingSheet(t *testing.T) {
	e := editCase(t)
	name := entry(t, e, "ntp_servers").Name
	withoutSheet(t, e, name)
	if s := entry(t, e, "ntp_servers"); !s.Missing || s.Rows != 0 || s.Name != name {
		t.Fatalf("entry = %+v", s)
	}
	if entry(t, e, "users").Missing {
		t.Error("a sheet of the book is told to be missing")
	}
	// The sheet opens with its headings alone.
	v := view(t, e, "ntp_servers")
	if !v.Missing || len(v.Rows) != 0 || len(v.Columns) == 0 {
		t.Fatalf("view = %+v", v)
	}
	draft(t, e, book.DraftSheet{Sheet: "ntp_servers", Added: []book.DraftAdd{{Cells: map[string]string{"device": "rt-a01", "server": "192.0.2.123"}}}})
	list, err := e.EditSheets()
	if err != nil || len(list.Lost) != 0 || list.Draft != 1 {
		t.Fatalf("list = %+v, %v", list, err)
	}
	copies := backups(t, e)
	saved, err := e.SaveEdits(nil, at)
	if err != nil || saved.Summary == nil || saved.Summary.Sheets != 1 || saved.Summary.Added != 1 || backups(t, e) != copies+1 {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	if s := entry(t, e, "ntp_servers"); s.Missing || s.Rows != 1 {
		t.Errorf("entry after the save = %+v", s)
	}
	// The step that reads the book finds the row.
	_, bk, err := e.readBook()
	if err != nil || bk == nil || len(bk.Data.Tables["ntp_servers"].Rows) != 1 {
		t.Errorf("the book after the save: %v", err)
	}
}

func TestEditReplaceAll(t *testing.T) {
	e := editCase(t)
	opt := book.FindOptions{}
	found, err := e.FindAll("sample", opt)
	if err != nil || len(found) != 1 || found[0].Sheet != "users" || found[0].Count != 1 {
		t.Fatalf("found = %+v, %v", found, err)
	}
	// The sheet the screen shows is left to the screen.
	out, err := e.ReplaceAll("sample", "example", opt, "users")
	if err != nil || len(out.After) != 0 || out.Draft != 0 || e.DraftCount() != 0 {
		t.Fatalf("out = %+v, %v", out, err)
	}
	out, err = e.ReplaceAll("sample", "example", opt, "devices")
	if err != nil || len(out.After) != 1 || len(out.Before) != 1 || out.Draft != 1 || e.DraftCount() != 1 {
		t.Fatalf("out = %+v, %v", out, err)
	}
	v := view(t, e, "users")
	if v.Rows[0].Cells[col(t, v, "name")] != "example-admin" || v.Rows[0].Cells[col(t, v, "secret")] != "sample-pass-1" {
		t.Errorf("row = %v", v.Rows[0].Cells)
	}
	// Putting the parts as they were back undoes it.
	for _, p := range out.Before {
		draft(t, e, p)
	}
	if e.DraftCount() != 0 {
		t.Errorf("the draft after the undo: %d", e.DraftCount())
	}
	if key, err := e.FreeColumn("users", "担当", []string{"機器"}); err != nil || key != "free:担当" {
		t.Errorf("key = %q, %v", key, err)
	}
}

func TestEditCacheFollowsTheBook(t *testing.T) {
	e := editCase(t)
	e.Edit = &EditCache{}
	if v := view(t, e, "users"); len(v.Rows) != 1 {
		t.Fatalf("rows = %d", len(v.Rows))
	}
	if e.Edit.sheets == nil {
		t.Fatal("the sheets were not kept")
	}
	// The packs of the step are the choices of the new vendor, whatever was kept.
	for _, packs := range [][]string{{"sample_a"}, {"sample_b", "sample_c"}} {
		e.PackIDs = packs
		v := view(t, e, "@devices")
		if got := v.Columns[col(t, v, "new_vendor")].Choices; len(got) != len(packs) || got[0] != packs[0] {
			t.Errorf("choices = %v, want %v", got, packs)
		}
	}
	// What the tool writes into the book is seen at once.
	draft(t, e, book.DraftSheet{Sheet: "users", Added: []book.DraftAdd{{Cells: map[string]string{"device": "rt-a01", "name": "sample-operator"}}}})
	if v := view(t, e, "users"); len(v.Rows) != 2 || v.Rows[1].Row != 0 {
		t.Errorf("the draft is not on the sheet that was kept: %+v", v.Rows)
	}
	if _, err := e.SaveEdits(nil, at); err != nil {
		t.Fatal(err)
	}
	if v := view(t, e, "users"); len(v.Rows) != 2 || v.Rows[1].Row != 3 {
		t.Errorf("rows after the save = %+v", v.Rows)
	}
	// And what a person saves in Excel: the file is another one by its time or its size.
	withoutSheet(t, e, entry(t, e, "users").Name)
	if s := entry(t, e, "users"); !s.Missing {
		t.Errorf("the change made elsewhere is not seen: %+v", s)
	}
	// A step without a cache works the same.
	e.Edit = nil
	if s := entry(t, e, "users"); !s.Missing {
		t.Errorf("without a cache: %+v", s)
	}
}
