package ui

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCaseExportImport(t *testing.T) {
	s, _ := newSession(t, map[string]string{"input/old/rt-a01.txt": "hostname rt-a01\n"})
	v, data, err := s.CaseExport(false, "correct horse")
	if err != nil || !v.Encrypted || v.Files == 0 || len(data) == 0 || !strings.HasSuffix(v.Name, ".ccase") {
		t.Fatalf("%+v %v", v, err)
	}
	path := filepath.Join(t.TempDir(), v.Name)
	writeTestFile(path, data, 0o644)
	f, err := CaseFile(path)
	if err != nil || !f.Sealed {
		t.Fatalf("%+v %v", f, err)
	}
	if err := CaseCheck(path, "wrong horse!"); err == nil {
		t.Error("a wrong password")
	}
	if err := CaseCheck(path, "correct horse"); err != nil {
		t.Error(err)
	}
	parent := t.TempDir()
	got, err := s.CaseImport(path, "correct horse", parent)
	if err != nil {
		t.Fatal(err)
	}
	if got.Home == nil || got.Home.Old != 1 || filepath.Dir(got.Dir) != parent {
		t.Errorf("%+v", got)
	}
	if st := s.State(); st.Case == nil || !sameDir(st.Case.Dir, got.Dir) {
		t.Errorf("the case read back is open: %+v", st.Case)
	}
	other := filepath.Join(t.TempDir(), "x.txt")
	writeTestFile(other, []byte("hello"), 0o644)
	if _, err := CaseFile(other); err == nil {
		t.Error("not a case file")
	}
}
