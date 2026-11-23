package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// generate --delta: the change against the configuration now, a device the
// same as the book, and a side that is not one.
func TestGenerateDelta(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "delta-case")
	if code, _, errOut := do("init", dir, "--kind", "new"); code != 0 {
		t.Fatalf("init = %d: %s", code, errOut)
	}
	cfg := "hostname rt-d01\n!\ninterface GigabitEthernet0/1\n description sample lan\n ip address 198.51.100.1 255.255.255.0\n!\nip route 0.0.0.0 0.0.0.0 198.51.100.254\n!\nend\n"
	put(t, filepath.Join(dir, "input/new/rt-d01.txt"), cfg)
	if code, _, errOut := do("book", "--project", dir, "--from", "new"); code != 0 {
		t.Fatalf("book = %d: %s", code, errOut)
	}
	if err := os.MkdirAll(filepath.Join(dir, "input/logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	now := strings.Replace(cfg, "!\nend\n", "ip route 203.0.113.0 255.255.255.0 198.51.100.254\n!\nend\n", 1)
	put(t, filepath.Join(dir, "input/logs/rt-d01.txt"), now)
	code, out, errOut := do("generate", "--project", dir, "--delta", "logs")
	if code == 2 {
		t.Fatalf("generate --delta = %d: %s", code, errOut)
	}
	wants(t, "generate --delta", out, "作った rt-d01: out/", "消す 1", "差分の投入（今のコンフィグ: input/logs）: 機器 1、作った 1", "レポート: out/")
	files, _ := filepath.Glob(filepath.Join(dir, "out", "*", "delta", "rt-d01.txt"))
	if len(files) != 1 {
		t.Fatalf("delta files: %v", files)
	}
	data, _ := os.ReadFile(files[0])
	if !strings.Contains(string(data), "no ip route 203.0.113.0 255.255.255.0 198.51.100.254") {
		t.Errorf("the change:\n%s", data)
	}
	if code, _, errOut := do("generate", "--project", dir, "--delta", "running"); code != 2 || !strings.Contains(errOut, "old・new・logs") {
		t.Errorf("a side that is not one = %d: %s", code, errOut)
	}
	if code, _, errOut := do("generate", "--project", dir, "--delta", "logs", "--device", "rt-z99"); code != 2 || !strings.Contains(errOut, "rt-z99") {
		t.Errorf("a device the book does not have = %d: %s", code, errOut)
	}
}
