package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/book"
)

// editCase is a case with an empty book and two rows of the draft saved into
// it: a device and a user with a password.
func editCase(t *testing.T) *Env {
	t.Helper()
	e := newCase(t, nil)
	if _, err := e.NewBook(at); err != nil {
		t.Fatal(err)
	}
	draft(t, e, book.DraftSheet{Sheet: "devices", Added: []book.DraftAdd{{Cells: map[string]string{"hostname": "rt-a01", "vendor": "cisco_ios"}}}})
	draft(t, e, book.DraftSheet{Sheet: "users", Added: []book.DraftAdd{{Cells: map[string]string{"device": "rt-a01", "name": "sample-admin", "secret": "sample-pass-1"}}}})
	saved, err := e.SaveEdits(nil, at)
	if err != nil || saved.Summary == nil || saved.Summary.Added != 2 {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	return e
}

func draft(t *testing.T, e *Env, part book.DraftSheet) int {
	t.Helper()
	n, err := e.SaveDraft(part)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func backups(t *testing.T, e *Env) int {
	t.Helper()
	list, _ := filepath.Glob(e.Proj.Path("state/backup/*.xlsx"))
	return len(list)
}

func view(t *testing.T, e *Env, id string) *book.View {
	t.Helper()
	v, err := e.EditSheet(id)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

func col(t *testing.T, v *book.View, key string) int {
	t.Helper()
	for i, c := range v.Columns {
		if c.Key == key {
			return i
		}
	}
	t.Fatalf("no column %s", key)
	return -1
}

func TestEditDraftAndSave(t *testing.T) {
	e := editCase(t)
	if e.DraftCount() != 0 || exists(e, "state/draft.yaml") {
		t.Fatal("the draft is still there after the save")
	}
	before := read(t, e, e.Proj.Config.Book)
	copies := backups(t, e)

	// A change is a draft first: the book stays as it is.
	v := view(t, e, "devices")
	host, model := col(t, v, "hostname"), col(t, v, "model")
	if len(v.Rows) != 1 || v.Rows[0].Cells[host] != "rt-a01" {
		t.Fatalf("rows = %+v", v.Rows)
	}
	row := v.Rows[0]
	part := book.DraftSheet{Sheet: "devices", Changed: []book.DraftRow{{Row: row.Row, Key: map[string]string{"hostname": "rt-a01"},
		Old: map[string]string{"model": row.Cells[model]}, New: map[string]string{"model": "sample-model"}}}}
	if n := draft(t, e, part); n != 1 || e.DraftCount() != 1 {
		t.Fatalf("draft = %d", n)
	}
	if read(t, e, e.Proj.Config.Book) != before {
		t.Fatal("the draft changed the book")
	}
	text := read(t, e, "state/draft.yaml")
	for _, want := range []string{"version: 1", "sheet: devices", "row: 2", "hostname: rt-a01", "model: sample-model"} {
		if !strings.Contains(text, want) {
			t.Errorf("state/draft.yaml lacks %q:\n%s", want, text)
		}
	}
	// The case opened again shows the draft on the sheet.
	e2, err := Open(e.Proj.Dir)
	if err != nil {
		t.Fatal(err)
	}
	v = view(t, e2, "devices")
	if v.Draft != 1 || v.Rows[0].Cells[model] != "sample-model" || v.Rows[0].Base[model] != "" {
		t.Fatalf("view = %+v", v.Rows[0])
	}
	list, err := e2.EditSheets()
	if err != nil || list.Draft != 1 || !list.HasBook {
		t.Fatalf("list = %+v, %v", list, err)
	}
	for _, s := range list.Sheets {
		if (s.ID == "devices") != (s.Draft == 1) {
			t.Errorf("sheet %s: draft %d", s.ID, s.Draft)
		}
	}

	// The draft of the sheet thrown away: the book shows again.
	if n, err := e.DiscardDraft("devices"); err != nil || n != 0 || exists(e, "state/draft.yaml") {
		t.Fatalf("discard: %d, %v", n, err)
	}
	if v = view(t, e, "devices"); v.Rows[0].Cells[model] != "" {
		t.Errorf("after the discard: %q", v.Rows[0].Cells[model])
	}

	// Saved: the book has the value, a copy was taken, the draft is gone.
	draft(t, e, part)
	saved, err := e.SaveEdits(nil, at)
	if err != nil || saved.Summary == nil || saved.Summary.Cells != 1 || saved.Backup == "" {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	if backups(t, e) != copies+1 || exists(e, "state/draft.yaml") {
		t.Errorf("backups %d → %d, draft there: %v", copies, backups(t, e), exists(e, "state/draft.yaml"))
	}
	if v = view(t, e, "devices"); v.Rows[0].Cells[model] != "sample-model" || v.Draft != 0 {
		t.Errorf("after the save: %+v", v.Rows[0])
	}
	// The steps read what was saved.
	_, bk, err := e.readBook()
	if err != nil || bk.Data.Tables["devices"].Rows[0].Get("model") != "sample-model" {
		t.Errorf("the book: %v", err)
	}
	// The log tells of the save without a value.
	if log := read(t, e, "state/log.txt"); !strings.Contains(log, "edit: ") || strings.Contains(log, "sample-pass-1") {
		t.Errorf("log:\n%s", log)
	}
}

func TestEditConflictKeepsDraft(t *testing.T) {
	e := editCase(t)
	v := view(t, e, "devices")
	row := v.Rows[0]
	draft(t, e, book.DraftSheet{Sheet: "devices", Changed: []book.DraftRow{{Row: row.Row, Key: map[string]string{"hostname": "rt-a01"},
		Old: map[string]string{"model": ""}, New: map[string]string{"model": "mine"}}}})
	// Excel writes the same cell meanwhile.
	setCell(t, e, v.Name, v.Columns[col(t, v, "hostname")].Title, "rt-a01", v.Columns[col(t, v, "model")].Title, "theirs")
	copies := backups(t, e)

	saved, err := e.SaveEdits(nil, at)
	if err != nil || saved.Summary != nil || len(saved.Conflicts) != 1 || saved.Conflicts[0].Theirs != "theirs" {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	if e.DraftCount() != 1 || backups(t, e) != copies {
		t.Errorf("a save that waits for a choice wrote: draft %d, backups %d → %d", e.DraftCount(), copies, backups(t, e))
	}
	saved, err = e.SaveEdits(map[string]string{saved.Conflicts[0].ID: "theirs"}, at)
	if err != nil || saved.Summary == nil || saved.Summary.Changed() || e.DraftCount() != 0 {
		t.Fatalf("theirs: %+v, %v", saved, err)
	}
	if v = view(t, e, "devices"); v.Rows[0].Cells[col(t, v, "model")] != "theirs" || backups(t, e) != copies {
		t.Errorf("theirs: the book has %q", v.Rows[0].Cells[col(t, v, "model")])
	}
}

func TestEditMappingSheets(t *testing.T) {
	e := editCase(t)
	// The sheets of the mapping are tables too; the new vendor is chosen
	// from the packs.
	v := view(t, e, "@devices")
	if c := v.Columns[col(t, v, "new_vendor")]; strings.Join(c.Choices, ",") != "cisco_ios,nec_ix,nec_qx" {
		t.Errorf("new vendor: %v", c.Choices)
	}
	draft(t, e, book.DraftSheet{Sheet: "@devices", Added: []book.DraftAdd{{Cells: map[string]string{"old_host": "rt-old01", "new_host": "rt-a01", "new_vendor": "nec_ix"}}}})
	draft(t, e, book.DraftSheet{Sheet: "@ifs", Added: []book.DraftAdd{{Cells: map[string]string{"new_host": "rt-a01", "old_if": "GigabitEthernet0/0", "new_if": "GigaEthernet0.0"}}}})
	draft(t, e, book.DraftSheet{Sheet: "@common", Added: []book.DraftAdd{{Cells: map[string]string{"scope": "*", "table": "devices", "column": "domain_name", "value": "example.com"}}}})
	if _, err := e.SaveEdits(nil, at); err != nil {
		t.Fatal(err)
	}
	_, bk, err := e.readBook()
	if err != nil || len(bk.Devices) != 1 || bk.Devices[0].NewVendor != "nec_ix" || len(bk.Ifs) != 1 || bk.Ifs[0].NewIf != "GigaEthernet0.0" || len(bk.Common) != 1 || bk.Common[0].Value != "example.com" {
		t.Fatalf("book = %+v %+v %+v, %v", bk.Devices, bk.Ifs, bk.Common, err)
	}
	// The sheet opens by its name in the file as well (a finding names it so).
	if byName, err := e.EditSheet(v.Name); err != nil || byName.ID != "@devices" || len(byName.Rows) != 1 {
		t.Errorf("by the name of the sheet: %v", err)
	}
	if _, err := e.EditSheet("no such sheet"); err == nil {
		t.Error("no error for a sheet that is not there")
	}
	// A value that does not fit its column is told, as reading the book tells it.
	if said := e.CheckValues("interfaces", "mtu", []string{"1500", "many", ""}); said[0] != "" || said[1] == "" || said[2] != "" {
		t.Errorf("check = %q", said)
	}
}

func TestEditWithoutBook(t *testing.T) {
	e := newCase(t, nil)
	list, err := e.EditSheets()
	if err != nil || list.HasBook || len(list.Sheets) != 0 {
		t.Fatalf("list = %+v, %v", list, err)
	}
	if _, err := e.EditSheet("devices"); err == nil {
		t.Error("no error without a book")
	}
	if err := os.WriteFile(e.Proj.Path("state/draft.yaml"), []byte("version: 1\nsheets: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if saved, err := e.SaveEdits(nil, at); err != nil || saved.Summary != nil {
		t.Errorf("an empty draft: %+v, %v", saved, err)
	}
}
