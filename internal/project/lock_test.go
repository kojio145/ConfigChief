package project

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLockRefusesASecondHolder(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("the mark is held on Windows only")
	}
	dir := t.TempDir()
	a, err := Lock(dir)
	if err != nil || a == nil {
		t.Fatalf("first Lock = %v, %v", a, err)
	}
	data, _ := os.ReadFile(filepath.Join(dir, LockFile))
	if strings.Count(string(data), "|") != 3 {
		t.Errorf("the holder line: %q", data)
	}
	_, err = Lock(dir)
	var inUse *InUseError
	if !errors.Is(err, ErrInUse) || !errors.As(err, &inUse) || !strings.Contains(err.Error(), "PC: ") {
		t.Fatalf("second Lock = %v", err)
	}
	a.Release()
	if data, _ := os.ReadFile(filepath.Join(dir, LockFile)); len(data) != 0 {
		t.Errorf("the mark is emptied when it is let go: %q", data)
	}
	b, err := Lock(dir)
	if err != nil || b == nil {
		t.Fatalf("Lock after Release = %v, %v", b, err)
	}
	b.Release()
	var none *Held
	none.Release() // safe on nil
}

func TestHolderGone(t *testing.T) {
	dead := func(uint32) bool { return false }
	alive := func(uint32) bool { return true }
	if !holderGoneOn("PC-A|u|123|2026-11-22 10:00", "pc-a", dead) {
		t.Error("a process of this PC that does not run")
	}
	if holderGoneOn("PC-A|u|123|2026-11-22 10:00", "pc-a", alive) {
		t.Error("a process of this PC that runs")
	}
	if holderGoneOn("PC-B|u|123|2026-11-22 10:00", "pc-a", dead) {
		t.Error("another PC is trusted")
	}
}

func TestNoteTool(t *testing.T) {
	dir := t.TempDir()
	if got := NoteTool(dir, "0.0.0-dev"); got != "" || ToolVersion(dir) != "" {
		t.Errorf("a development build records nothing: %q %q", got, ToolVersion(dir))
	}
	if got := NoteTool(dir, "1.3.0"); got != "" || ToolVersion(dir) != "1.3.0" {
		t.Errorf("first record: %q %q", got, ToolVersion(dir))
	}
	if got := NoteTool(dir, "1.2.0"); got != "1.3.0" || ToolVersion(dir) != "1.3.0" {
		t.Errorf("an older tool is told and records nothing: %q %q", got, ToolVersion(dir))
	}
	if got := NoteTool(dir, "1.4.0"); got != "" || ToolVersion(dir) != "1.4.0" {
		t.Errorf("a newer tool records itself: %q %q", got, ToolVersion(dir))
	}
	if got := NoteTool(dir, "0.0.0-dev"); got != "" {
		t.Errorf("a development build never refuses: %q", got)
	}
	e := &ReadOnlyError{Newer: "1.4.0", Mine: "1.2.0"}
	if !strings.Contains(e.Error(), "v1.4.0") || !strings.Contains(e.Error(), "v1.2.0") {
		t.Error(e.Error())
	}
}

func TestSharedPlace(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows paths")
	}
	if got := SharedPlace(`\\server\share\case`); got != "ネットワークの場所" {
		t.Errorf("UNC: %q", got)
	}
	if got := SharedPlace(t.TempDir()); got != "" {
		t.Errorf("a local folder: %q", got)
	}
	root := t.TempDir()
	t.Setenv("OneDrive", root)
	if got := SharedPlace(filepath.Join(root, "cases", "a")); got != "OneDrive" {
		t.Errorf("OneDrive: %q", got)
	}
	if got := SharedPlace(root + "-other"); got != "" {
		t.Errorf("beside OneDrive: %q", got)
	}
}
