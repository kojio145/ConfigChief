package cli

import (
	"io/fs"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/packs"
)

// "sites": the devices of one source side by side.
func TestSitesCommand(t *testing.T) {
	fsys, err := packs.FS("cisco_ios")
	if err != nil {
		t.Fatal(err)
	}
	data, err := fs.ReadFile(fsys, "tests/import/basic/input.txt")
	if err != nil {
		t.Fatal(err)
	}
	base := string(data)
	dir := filepath.Join(t.TempDir(), "sample-case")
	if code, _, errOut := do("init", dir); code != 0 {
		t.Fatalf("init = %d: %s", code, errOut)
	}
	for _, h := range []string{"rt-a01", "rt-a02", "rt-a03"} {
		text := strings.Replace(base, "hostname rt-a01\n", "hostname "+h+"\n", 1)
		if h == "rt-a02" {
			text = strings.Replace(text, " description sample lan\n", " description sample lan 2\n", 1)
		}
		put(t, filepath.Join(dir, "input", "old", h+".txt"), text)
	}
	code, out, errOut := do("sites", "--project", dir, "--source", "old")
	if code != 0 {
		t.Fatalf("sites = %d: %s%s", code, out, errOut)
	}
	for _, want := range []string{"機器 3 台", "外れ値 rt-a02 ", "= sample lan 2（3 台中 2 台は sample lan）", "外れ値 1 件", "sites.xlsx"} {
		if !strings.Contains(out, want) {
			t.Errorf("the output does not hold %q:\n%s", want, out)
		}
	}
	for _, args := range [][]string{
		{"sites"},
		{"sites", "--project", dir}, // no book yet
		{"sites", "--project", dir, "--source", "other"},
		{"sites", "--project", dir, "--source", "new"},
		{"sites", "--project", dir, "extra"},
	} {
		if code, _, errOut := do(args...); code != 2 || errOut == "" {
			t.Errorf("%v = %d: %s", args, code, errOut)
		}
	}
}
