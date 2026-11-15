package cli

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// "diff --tree" with two files, and with two sides of a case.
func TestTreeDiffCommands(t *testing.T) {
	tick := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	now = func() time.Time {
		tick = tick.Add(time.Minute)
		return tick
	}
	defer func() { now = time.Now }()
	text := rendered(t, "cisco_ios")
	i := strings.Index(text, "\ninterface ")
	if i < 0 {
		t.Fatal("no interface in the sample")
	}
	j := i + 1 + strings.Index(text[i+1:], "\n")
	head := text[i+1 : j]
	changed := text[:j+1] + " load-interval 30\n" + text[j+1:]

	tmp := t.TempDir()
	a, b := filepath.Join(tmp, "a.txt"), filepath.Join(tmp, "b.txt")
	put(t, a, text)
	put(t, b, "Current configuration : 100 bytes\n"+changed)

	if code, out, errOut := do("diff", "--tree", a, a); code != 0 || !strings.Contains(out, "違いはありません") {
		t.Fatalf("the same file = %d: %s%s", code, out, errOut)
	}
	code, out, errOut := do("diff", "--tree", a, b)
	if code != 1 {
		t.Fatalf("two files = %d: %s%s", code, out, errOut)
	}
	for _, want := range []string{"@@ " + head + "\n", ":  load-interval 30\n", "B だけの行 1、A だけの行 0、変更 0、順序 0", "時刻の行 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("the output does not hold %q:\n%s", want, out)
		}
	}
	// With the stamps kept, the size of the listing is a line of B.
	if code, out, _ := do("diff", "--tree", "--keep-stamps", a, b); code != 1 || !strings.Contains(out, "B だけの行 2") {
		t.Errorf("--keep-stamps = %d: %s", code, out)
	}
	for _, args := range [][]string{
		{"diff", "--tree", a},
		{"diff", "--tree", a, b, "--other"},
		{"diff", "--tree", a, filepath.Join(tmp, "none.txt")},
		{"diff", "--tree", "--a", "old", "--b", "new"},
		{"diff", "--tree", "--project", tmp, "--a", "old", "--b", "new"},
	} {
		if code, _, errOut := do(args...); code != 2 || errOut == "" {
			t.Errorf("%v = %d: %s", args, code, errOut)
		}
	}

	// Every device of two sides.
	dir := filepath.Join(tmp, "sample-case")
	if code, _, errOut := do("init", dir); code != 0 {
		t.Fatalf("init = %d: %s", code, errOut)
	}
	put(t, filepath.Join(dir, "input", "old", "rt-c01.txt"), text)
	put(t, filepath.Join(dir, "input", "new", "rt-c01.txt"), changed)
	code, out, errOut = do("diff", "--tree", "--project", dir, "--a", "old", "--b", "new")
	if code != 1 {
		t.Fatalf("two sides = %d: %s%s", code, out, errOut)
	}
	for _, want := range []string{"rt-c01: B だけの行 1、A だけの行 0、変更 0、順序 0", "tree_diff.xlsx", "tree_diff.html"} {
		if !strings.Contains(out, want) {
			t.Errorf("the output does not hold %q:\n%s", want, out)
		}
	}
	put(t, filepath.Join(dir, "input", "new", "rt-c01.txt"), text)
	if code, out, errOut := do("diff", "--tree", "--project", dir, "--a", "old", "--b", "new"); code != 0 || !strings.Contains(out, "rt-c01: 違いはありません") {
		t.Fatalf("the same on both sides = %d: %s%s", code, out, errOut)
	}
	if code, _, errOut := do("diff", "--tree", "--project", dir, "--a", "old", "--b", "logs"); code != 2 || !strings.Contains(errOut, "input/logs") {
		t.Errorf("a side without files = %d: %s", code, errOut)
	}
}
