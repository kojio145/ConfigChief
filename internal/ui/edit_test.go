package ui

import (
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/book"
)

func TestEditOnScreen(t *testing.T) {
	s, _ := newSession(t, nil)
	// Without a book there is nothing to edit.
	list, err := s.EditSheets()
	if err != nil || list.HasBook {
		t.Fatalf("list = %+v, %v", list, err)
	}
	noNull(t, "the list without a book", list)
	if _, err := s.NewBook(); err != nil {
		t.Fatal(err)
	}
	list, err = s.EditSheets()
	if err != nil || !list.HasBook || len(list.Sheets) < 4 {
		t.Fatalf("list = %+v, %v", list, err)
	}
	noNull(t, "the list", list)

	// Rows typed into the table are a draft; the home screen counts them.
	n, err := s.EditDraft(book.DraftSheet{Sheet: "users", Added: []book.DraftAdd{{Cells: map[string]string{"device": "rt-a01", "name": "sample-admin", "secret": "sample-pass-1"}}}})
	if err != nil || n != 1 {
		t.Fatalf("draft = %d, %v", n, err)
	}
	if h, err := s.Home(); err != nil || h.Draft != 1 {
		t.Fatalf("home = %+v, %v", h, err)
	}
	v, err := s.EditSheet("users")
	if err != nil || len(v.Rows) != 1 || v.Rows[0].Row != 0 || v.Draft != 1 {
		t.Fatalf("view = %+v, %v", v, err)
	}
	text := noNull(t, "the sheet", v)
	// The page gets the column as a secret one, to mask it.
	if !strings.Contains(text, `"secret":true`) {
		t.Errorf("no secret column in %s", text)
	}
	if said, err := s.EditCheck("interfaces", "mtu", []string{"many"}); err != nil || said[0] == "" {
		t.Errorf("check = %q, %v", said, err)
	}

	saved, err := s.EditSave(nil)
	if err != nil || !saved.Saved || saved.Draft != 0 || !strings.Contains(saved.Message, "追加 1 行") || strings.Contains(saved.Message, "sample-pass-1") {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	noNull(t, "the save", saved)
	if h, err := s.Home(); err != nil || h.Draft != 0 {
		t.Fatalf("home = %+v, %v", h, err)
	}

	// A conflict reaches the page with the secret values masked.
	v, _ = s.EditSheet("users")
	row := v.Rows[0]
	if _, err := s.EditDraft(book.DraftSheet{Sheet: "users", Changed: []book.DraftRow{{Row: row.Row, Key: map[string]string{"device": "rt-a01", "name": "sample-admin"},
		Old: map[string]string{"secret": "sample-pass-0"}, New: map[string]string{"secret": "sample-pass-2"}}}}); err != nil {
		t.Fatal(err)
	}
	saved, err = s.EditSave(nil)
	if err != nil || saved.Saved || len(saved.Conflicts) != 1 {
		t.Fatalf("saved = %+v, %v", saved, err)
	}
	if text := noNull(t, "the conflicts", saved); strings.Contains(text, "sample-pass") {
		t.Errorf("a secret value in %s", text)
	}
	if n, err := s.EditDiscard(""); err != nil || n != 0 {
		t.Errorf("discard = %d, %v", n, err)
	}
}
