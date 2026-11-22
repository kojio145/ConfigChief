package errlog

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPanic(t *testing.T) {
	dir := t.TempDir()
	was := Path
	Path = func() string { return filepath.Join(dir, "sub", "error.log") }
	defer func() { Path = was }()
	msg := Panic("Generate", "index out of range")
	if !strings.Contains(msg, "Generate: index out of range") || !strings.Contains(msg, "error.log") {
		t.Errorf("message: %q", msg)
	}
	data, err := os.ReadFile(Path())
	if err != nil || !strings.Contains(string(data), "panic in Generate: index out of range") || !strings.Contains(string(data), "goroutine") {
		t.Fatalf("log: %v %q", err, data)
	}
	os.WriteFile(Path(), make([]byte, maxSize+1), 0o600)
	Panic("Check", 1)
	if _, err := os.Stat(Path() + ".old"); err != nil {
		t.Error("a large log is moved aside")
	}
}
