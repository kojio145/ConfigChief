package cli

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kojio145/configchief/internal/packs"
)

// sample gives the text of the import test of a bundled pack.
func sample(t *testing.T, id string) string {
	t.Helper()
	fsys, err := packs.FS(id)
	if err != nil {
		t.Fatal(err)
	}
	data, err := fs.ReadFile(fsys, "tests/import/basic/input.txt")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// do runs the command and gives the exit code and both outputs.
func do(args ...string) (code int, out, errOut string) {
	var o, e bytes.Buffer
	code = Run(args, &o, &e)
	return code, o.String(), e.String()
}

func put(t *testing.T, path, text string) {
	t.Helper()
	if err := writeTestFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func wants(t *testing.T, what, got string, parts ...string) {
	t.Helper()
	for _, p := range parts {
		if !strings.Contains(got, p) {
			t.Errorf("%s lacks %q:\n%s", what, p, got)
		}
	}
}

func TestCase(t *testing.T) {
	now = func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }
	defer func() { now = time.Now }()
	dir := filepath.Join(t.TempDir(), "sample-case")
	const reports = "out/2026-01-02_030405/reports/"

	code, out, errOut := do("init", dir, "--kind", "migration")
	if code != 0 {
		t.Fatalf("init = %d: %s", code, errOut)
	}
	wants(t, "init", out, "案件名 sample-case", "input/old")
	if code, _, errOut = do("init", dir); code != 2 || !strings.Contains(errOut, "すでに") {
		t.Errorf("a second init = %d: %s", code, errOut)
	}
	if code, _, errOut = do("init", filepath.Join(t.TempDir(), "x"), "--kind", "other"); code != 2 || !strings.Contains(errOut, "migration か new") {
		t.Errorf("init with a wrong kind = %d: %s", code, errOut)
	}

	// Nothing to read yet.
	if code, _, errOut = do("import", "--project", dir); code != 2 || !strings.Contains(errOut, "input/old にファイルがありません") {
		t.Errorf("import without files = %d: %s", code, errOut)
	}
	for _, args := range [][]string{
		{"import"},
		{"import", "--project", dir, "--colour", "red"},
		{"import", "--project", dir, "--source"},
		{"import", "--project", dir, "extra"},
		{"import", "--project", filepath.Join(dir, "nothing")},
		{"book", "--project", dir, "--from", "logs"},
		{"compare", "--project", dir, "--a", "old"},
		{"compare", "--project", dir, "--a", "old", "--b", "old"},
		{"diff", "one"},
		{"diff", filepath.Join(dir, "nothing-a.txt"), filepath.Join(dir, "nothing-b.txt")},
	} {
		if code, _, errOut = do(args...); code != 2 || errOut == "" {
			t.Errorf("%v = %d, stderr %q", args, code, errOut)
		}
	}

	// The sample of the pack is read whole by now: a line no rule takes in
	// is put into it, for the steps that tell of such lines.
	text := strings.Replace(sample(t, "cisco_ios"), "hostname rt-a01\n", "hostname rt-a01\nservice tcp-keepalives-in\n", 1)
	oldFile, newFile := filepath.Join(dir, "input", "old", "rt-a01.txt"), filepath.Join(dir, "input", "new", "rt-a01.txt")
	put(t, oldFile, text)
	put(t, newFile, strings.Replace(text, "description sample loopback", "description sample loopback 2", 1))

	// The sample has lines no rule takes in: the step says so with its code.
	code, out, errOut = do("import", "--project", dir)
	if code != 1 {
		t.Fatalf("import = %d: %s%s", code, out, errOut)
	}
	wants(t, "import", out,
		"取り込み（旧コンフィグ）: 1 ファイル",
		"input/old/rt-a01.txt: rt-a01 cisco_ios ",
		"読めなかった行: ",
		"パラメータブック parameters.xlsx を作りました（新旧対照表（機器） 1 行、",
		"レポート: "+reports+"import_old.xlsx",
		reports+"import_old.html",
	)
	if strings.Contains(out, "決まっていない") {
		t.Errorf("the mapping is open although the new side has the same names:\n%s", out)
	}
	for _, rel := range []string{"parameters.xlsx", reports + "import_old.xlsx", reports + "import_old.html", "state/log.txt"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(rel))); err != nil {
			t.Errorf("%s is not there: %v", rel, err)
		}
	}

	code, out, _ = do("import", "--project", dir, "--source", "new", "--pack", "cisco_ios")
	if code != 1 {
		t.Errorf("import new = %d", code)
	}
	// --pack is for the files whose vendor is not clear; this one is.
	wants(t, "import new", out, "取り込み（新コンフィグ）", "rt-a01 cisco_ios 2", reports+"import_new.html")
	if code, _, errOut = do("import", "--project", dir, "--pack", "nothing"); code != 2 || !strings.Contains(errOut, "パック nothing はありません") {
		t.Errorf("import with a pack that is not there = %d: %s", code, errOut)
	}

	code, out, errOut = do("book", "--project", dir)
	if code != 0 {
		t.Fatalf("book = %d: %s%s", code, out, errOut)
	}
	wants(t, "book", out, "パラメータブック parameters.xlsx に足しました（旧コンフィグ から ", "控え: state/backup/parameters_2026-01-02_030405.xlsx", "インタフェース: 足した ")
	code, out, _ = do("book", "--project", dir)
	if code != 0 {
		t.Errorf("a second book = %d", code)
	}
	wants(t, "a second book", out, "は変えていません")
	code, out, _ = do("book", "--project", dir, "--from", "new", "--replace")
	if code != 0 {
		t.Errorf("book --replace = %d:\n%s", code, out)
	}
	wants(t, "book --replace", out, "設計データのシートを空にしてから書きました（新コンフィグ から ", "控え: state/backup/parameters_2026-01-02_030405_2.xlsx")

	code, out, errOut = do("compare", "--project", dir, "--a", "old", "--b", "new")
	if code != 1 {
		t.Fatalf("compare = %d: %s%s", code, out, errOut)
	}
	wants(t, "compare", out,
		"比較（A = 旧コンフィグ、B = 新コンフィグ）",
		"A にだけある行: 0",
		"B にだけある行: 0",
		"値が違うセル: 1",
		"旧コンフィグ に読めなかった行が ",
		reports+"compare_old_new.xlsx",
	)
	code, out, _ = do("compare", "--project", dir, "--a", "book", "--b", "new")
	if code != 1 {
		t.Errorf("compare book new = %d", code)
	}
	wants(t, "compare book new", out, "比較（A = パラメータブック、B = 新コンフィグ）", "値が違うセル: 0")

	code, out, _ = do("diff", oldFile, oldFile)
	if code != 0 || !strings.Contains(out, "違いはありません") {
		t.Errorf("diff of a file with itself = %d: %s", code, out)
	}
	code, out, _ = do("diff", oldFile, newFile)
	if code != 1 {
		t.Errorf("diff = %d", code)
	}
	wants(t, "diff", out, "--- "+oldFile, "+++ "+newFile, "@@ -", "- description sample loopback\n", "+ description sample loopback 2\n", "減った行 1、増えた行 1")
}
