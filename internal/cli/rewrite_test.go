package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// generate and check write the rewriting report in a case of renewal, and
// tell its numbers (spec 13.4, 15).
func TestGenerateRewrite(t *testing.T) {
	tick := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	now = func() time.Time {
		tick = tick.Add(time.Minute)
		return tick
	}
	defer func() { now = time.Now }()
	dir := filepath.Join(t.TempDir(), "sample-case")
	if code, _, errOut := do("init", dir); code != 0 {
		t.Fatalf("init = %d: %s", code, errOut)
	}
	text := rendered(t, "cisco_ios")
	put(t, filepath.Join(dir, "input", "old", "rt-c01.txt"), text)
	put(t, filepath.Join(dir, "input", "new", "rt-c01.txt"), text)
	if code, _, errOut := do("import", "--project", dir, "--source", "old"); code == 2 {
		t.Fatalf("import = %d: %s", code, errOut)
	}
	if code, _, errOut := do("book", "--project", dir, "--from", "old"); code == 2 {
		t.Fatalf("book = %d: %s", code, errOut)
	}
	for _, cmd := range []string{"generate", "check"} {
		code, out, errOut := do(cmd, "--project", dir)
		if code == 2 || !strings.Contains(out, "書き換え（旧コンフィグ ") || !strings.Contains(out, "書き換えレポート: out/") {
			t.Fatalf("%s = %d: %s%s", cmd, code, out, errOut)
		}
		i := strings.Index(out, "書き換えレポート: ")
		rel := strings.TrimSpace(strings.SplitN(out[i+len("書き換えレポート: "):], "\n", 2)[0])
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("%s: the report %s: %v", cmd, rel, err)
		}
	}
}
