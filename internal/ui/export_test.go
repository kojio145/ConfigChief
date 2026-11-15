package ui

import (
	"path/filepath"
	"strings"
	"testing"
)

// The deliverables from the home screen: the delivery sheets in the standard
// form, and the table for the other side.
func TestExport(t *testing.T) {
	s, dir := newSession(t, map[string]string{"input/new/rt-a01.txt": sample(t, "cisco_ios")})
	if _, err := s.Export("sheet", false); err == nil || !strings.Contains(err.Error(), "先にパラメータブックを作ってください") {
		t.Errorf("no book: %v", err)
	}
	if _, err := s.MakeBook("new", false); err != nil {
		t.Fatal(err)
	}
	v, err := s.Export("sheet", true)
	if err != nil {
		t.Fatal(err)
	}
	noNullValue(t, "the sheets", v)
	if len(v.Files) != 1 || !strings.HasSuffix(v.Files[0], "/sheets/parameter_sheet.xlsx") || !exists(filepath.Join(dir, filepath.FromSlash(v.Files[0]))) ||
		v.Dir == "" || !strings.Contains(v.Summary, "標準の形") || !strings.Contains(v.Summary, "*** に伏せています") {
		t.Errorf("the sheets: %+v", v)
	}
	// The files can be opened from the screen.
	if _, err := s.Path(v.Files[0]); err != nil {
		t.Errorf("the path of the sheet: %v", err)
	}

	v, err = s.Export("peer", false)
	if err != nil {
		t.Fatal(err)
	}
	noNullValue(t, "the table", v)
	// The sample has neighbours that are not of the case.
	if len(v.Files) != 2 || !strings.HasSuffix(v.Files[0], "/reports/peer_parameters.xlsx") || !strings.Contains(v.Summary, "そのまま書いています") {
		t.Errorf("the table: %+v", v)
	}
	if _, err := s.Export("report", false); err == nil {
		t.Errorf("an unknown thing to write gave no error")
	}
}
