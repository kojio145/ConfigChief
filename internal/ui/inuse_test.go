package ui

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/project"
)

// A case open in one window is refused by another, and let go when it is
// closed (spec 14.8.2).
func TestCaseInUse(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the mark is held on Windows only")
	}
	lockCases = true
	defer func() { lockCases = false }()
	a, dir := newSession(t, nil)
	defer a.Close()
	b := New("1.2.3", "")
	if _, err := b.Open(dir); !errors.Is(err, project.ErrInUse) || !strings.Contains(err.Error(), "別の場所で開かれています") {
		t.Fatalf("a second window: %v", err)
	}
	if _, err := a.Open(dir); err != nil {
		t.Fatalf("the window that has the case opens it again: %v", err)
	}
	a.Close()
	if data, _ := os.ReadFile(filepath.Join(dir, project.LockFile)); len(data) != 0 {
		t.Errorf("closing empties the mark: %q", data)
	}
	if _, err := b.Open(dir); err != nil {
		t.Fatalf("after the first closed it: %v", err)
	}
	b.Close()
}

// A case used by a newer ConfigChief is only read (spec 14.8.3).
func TestCaseOfNewerVersion(t *testing.T) {
	s, dir := newSession(t, nil)
	if v := project.ToolVersion(dir); v != "1.2.3" {
		t.Fatalf("the version is recorded: %q", v)
	}
	writeTestFile(filepath.Join(dir, project.ToolFile), []byte("version: 9.0.0\n"), 0o644)
	if _, err := s.Open(dir); err != nil {
		t.Fatal(err)
	}
	if st := s.State(); !strings.Contains(st.ReadOnly, "v9.0.0") {
		t.Errorf("the frame says so: %q", st.ReadOnly)
	}
	var ro *project.ReadOnlyError
	if _, err := s.Generate(); !errors.As(err, &ro) {
		t.Errorf("Generate: %v", err)
	}
	if _, err := s.NewBook(); !errors.As(err, &ro) {
		t.Errorf("NewBook: %v", err)
	}
	if _, err := s.EditDiscard("devices"); !errors.As(err, &ro) {
		t.Errorf("EditDiscard: %v", err)
	}
	if _, err := s.Home(); err != nil {
		t.Errorf("reading is fine: %v", err)
	}
	if v := project.ToolVersion(dir); v != "9.0.0" {
		t.Errorf("an older tool leaves the record: %q", v)
	}
	s.Close()
	if st := s.State(); st.ReadOnly != "" {
		t.Errorf("no case, nothing read only: %q", st.ReadOnly)
	}
}

func TestZoom(t *testing.T) {
	settings := filepath.Join(t.TempDir(), "settings.json")
	s := New("1.2.3", settings)
	if st := s.State(); st.Zoom != 100 {
		t.Errorf("the zoom at first: %d", st.Zoom)
	}
	if z := s.SetZoom(123); z != 120 {
		t.Errorf("SetZoom(123) = %d", z)
	}
	if z := s.SetZoom(10); z != 70 {
		t.Errorf("SetZoom(10) = %d", z)
	}
	if z := New("1.2.3", settings).State().Zoom; z != 70 {
		t.Errorf("the zoom is kept: %d", z)
	}
}
