package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/project"
)

// A case open in the window stops a command; a case used by a newer
// ConfigChief stops a command that writes it (spec 14.8.2, 14.8.3).
func TestCaseInUseAndNewer(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "case")
	var out, errOut bytes.Buffer
	if code := Run([]string{"init", dir}, &out, &errOut); code != 0 {
		t.Fatalf("init: %d %s", code, errOut.String())
	}
	if runtime.GOOS == "windows" {
		h, err := project.Lock(dir)
		if err != nil || h == nil {
			t.Fatalf("Lock: %v", err)
		}
		out.Reset()
		errOut.Reset()
		if code := Run([]string{"generate", "--project", dir}, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "画面など別の場所で開かれています") {
			t.Errorf("in use: %d %s", code, errOut.String())
		}
		h.Release()
	}
	os.WriteFile(filepath.Join(dir, project.ToolFile), []byte("version: 9.0.0\n"), 0o644)
	was := Version
	Version = "1.2.0"
	defer func() { Version = was }()
	errOut.Reset()
	if code := Run([]string{"generate", "--project", dir}, &out, &errOut); code != 2 || !strings.Contains(errOut.String(), "v9.0.0") {
		t.Errorf("newer: %d %s", code, errOut.String())
	}
	errOut.Reset()
	if code := Run([]string{"sites", "--project", dir}, &out, &errOut); strings.Contains(errOut.String(), "v9.0.0") {
		t.Errorf("a command that only reads runs: %d %s", code, errOut.String())
	}
	if h, err := project.Lock(dir); err != nil {
		t.Errorf("the command let the case go: %v", err)
	} else {
		h.Release()
	}
}

func TestCaseCommands(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "case")
	var out, errOut bytes.Buffer
	Run([]string{"init", dir, "--name", "sample"}, &out, &errOut)
	os.WriteFile(filepath.Join(dir, "input", "old", "rt-a01.txt"), []byte("hostname rt-a01\n"), 0o644)
	file := filepath.Join(t.TempDir(), "x")
	t.Setenv("CC_TEST_PW", "correct horse")
	out.Reset()
	errOut.Reset()
	if code := Run([]string{"case", "export", "--project", dir, "--out", file, "--password-env", "CC_TEST_PW"}, &out, &errOut); code != 0 || !strings.Contains(out.String(), "パスワードで暗号化しました") {
		t.Fatalf("export: %d %s %s", code, out.String(), errOut.String())
	}
	parent := t.TempDir()
	if code := Run([]string{"case", "import", file + ".ccase", "--to", parent}, &out, &errOut); code != 2 {
		t.Errorf("a sealed file without its password: %d", code)
	}
	out.Reset()
	if code := Run([]string{"case", "import", file + ".ccase", "--to", parent, "--password-env", "CC_TEST_PW"}, &out, &errOut); code != 0 {
		t.Fatalf("import: %d %s", code, errOut.String())
	}
	if _, err := os.Stat(filepath.Join(parent, "sample", "input", "old", "rt-a01.txt")); err != nil {
		t.Errorf("the case is spread: %v %s", err, out.String())
	}
}
