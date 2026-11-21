package cli

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/packs"
)

// "fill": a template filled from a list, a change, and a first template.
func TestFillCommand(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sample-case")
	if code, _, errOut := do("init", dir); code != 0 {
		t.Fatalf("init = %d: %s", code, errOut)
	}
	put(t, filepath.Join(dir, "input", "customer", "sites.csv"), "host,wan\nrt-a01,192.0.2.1/30\nrt-a02,192.0.2.5/30\nrt-a03,\n")
	put(t, filepath.Join(dir, "vars", "sites.yaml"), "version: 1\nfile: sites.csv\nheader_row: 1\ndevice: {column: host}\nvars:\n  - {name: wan, column: wan}\n")
	put(t, filepath.Join(dir, "fill", "wan.txt"), "interface GigabitEthernet0/0\n ip address {{wan}}\n")
	fsys, err := packs.FS("cisco_ios")
	if err != nil {
		t.Fatal(err)
	}
	sample, err := fs.ReadFile(fsys, "tests/import/basic/input.txt")
	if err != nil {
		t.Fatal(err)
	}
	put(t, filepath.Join(dir, "input", "old", "rt-a01.txt"), string(sample))

	code, out, errOut := do("fill", "--project", dir, "--list")
	if code != 0 || !strings.Contains(out, "fill/wan.txt") || !strings.Contains(out, "変数 wan") {
		t.Fatalf("list = %d: %s%s", code, out, errOut)
	}
	code, out, errOut = do("fill", "--project", dir, "--template", "wan")
	if code != 1 || !strings.Contains(out, "作った rt-a02") || !strings.Contains(out, "作らなかった rt-a03: ") || !strings.Contains(out, "作った 2、作らなかった 1") {
		t.Fatalf("template = %d: %s%s", code, out, errOut)
	}
	code, out, errOut = do("fill", "--project", dir, "--template", "wan", "--diff", "old", "--device", "rt-a01")
	if code != 0 || !strings.Contains(out, "作った rt-a01") {
		t.Fatalf("diff = %d: %s%s", code, out, errOut)
	}
	code, out, errOut = do("fill", "--project", dir, "--draft", "rt-a01.txt", "--device", "rt-a01", "--name", "first")
	if code != 0 || !strings.Contains(out, "fill/first.txt") {
		t.Fatalf("draft = %d: %s%s", code, out, errOut)
	}
	if code, _, _ = do("fill", "--project", dir); code != 2 {
		t.Fatalf("no mode = %d", code)
	}
}
