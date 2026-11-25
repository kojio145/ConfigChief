package cli

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/kojio145/configchief/internal/packs"
)

// rendered gives the text a bundled pack writes in its generation test.
func rendered(t *testing.T, id string) string {
	t.Helper()
	fsys, err := packs.FS(id)
	if err != nil {
		t.Fatal(err)
	}
	data, err := fs.ReadFile(fsys, "tests/render/basic/expect.txt")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestGenerateCommands(t *testing.T) {
	// Each run is a minute after the one before.
	tick := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	now = func() time.Time {
		tick = tick.Add(time.Minute)
		return tick
	}
	defer func() { now = time.Now }()
	dir := filepath.Join(t.TempDir(), "sample-case")
	if code, _, errOut := do("init", dir, "--kind", "new"); code != 0 {
		t.Fatalf("init = %d: %s", code, errOut)
	}
	want := rendered(t, "cisco_ios")
	put(t, filepath.Join(dir, "input", "new", "rt-c01.txt"), want)

	if code, _, errOut := do("generate", "--project", dir); code != 2 || !strings.Contains(errOut, "パラメータブック") {
		t.Errorf("generate without a book = %d: %s", code, errOut)
	}
	if code, out, errOut := do("book", "--project", dir, "--from", "new"); code != 0 {
		t.Fatalf("book = %d: %s%s", code, out, errOut)
	}
	for _, args := range [][]string{
		{"generate"},
		{"generate", "--project", dir, "--device"},
		{"generate", "--project", dir, "--device", "rt-z99"},
		{"generate", "--project", dir, "extra"},
		{"check", "--project", dir, "--device", "rt-c01"},
		{"approve", "--project", dir},
		{"approve", "--project", dir, "--finding", "0000000000000000", "--reason", "sample reason", "--by", "sample-user"},
		{"approve", "--project", dir, "--remove", "a-0001"},
	} {
		if code, _, errOut := do(args...); code != 2 || errOut == "" {
			t.Errorf("%v = %d, stderr %q", args, code, errOut)
		}
	}

	code, out, errOut := do("generate", "--project", dir)
	if code != 0 {
		t.Fatalf("generate = %d: %s%s", code, out, errOut)
	}
	const first = "out/2026-01-02_031105/"
	wants(t, "generate", out,
		"生成: 1 台（書いた 1、",
		"rt-c01: "+first+"configs/rt-c01.txt",
		"指摘: 未承認のエラー 0、未承認の警告 0、",
		"チェック結果表: "+first+"reports/check_results.xlsx",
	)
	if strings.Contains(out, "UNVERIFIED") || strings.Contains(out, "変更一覧") {
		t.Errorf("a first generation without findings:\n%s", out)
	}
	got, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(first+"configs/rt-c01.txt")))
	if err != nil || strings.ReplaceAll(string(got), "\r\n", "\n") != want {
		t.Errorf("the configuration is not the one the book came from (%v)", err)
	}
	code, out, _ = do("check", "--project", dir)
	if code != 0 {
		t.Errorf("check = %d:\n%s", code, out)
	}
	wants(t, "check", out, "チェック: 1 台（確かめた 1、", "reports/check_results.html")

	// A second device that names access lists it does not have, with the same
	// addresses as the first.
	put(t, filepath.Join(dir, "input", "new", "ix-c01.txt"), rendered(t, "nec_ix"))
	if code, out, errOut := do("book", "--project", dir, "--from", "new"); code != 0 {
		t.Fatalf("book = %d: %s%s", code, out, errOut)
	}
	code, out, _ = do("generate", "--project", dir)
	if code != 1 {
		t.Fatalf("generate with findings = %d:\n%s", code, out)
	}
	wants(t, "generate with findings", out,
		"生成: 2 台（書いた 1、値の誤りで止めた 1、",
		"ix-c01: 生成していません（値の誤り（V-106）",
		"エラー V-106 ix-c01 インタフェース IF 名=",
		"警告 V-202 ix-c01 インタフェース IF 名=",
		"UNVERIFIED.txt",
		"前回の生成（2026-01-02_031105）との違い: 変わったセル 0、増えた行 0、減った行 0",
		"reports/changes.xlsx",
	)
	keys := regexp.MustCompile(`警告 V-202 .* \[([0-9a-f]{16})\]`).FindAllStringSubmatch(out, -1)
	if len(keys) < 2 {
		t.Fatalf("no fingerprints in:\n%s", out)
	}
	key := regexp.MustCompile(`エラー V-106 .* \[([0-9a-f]{16})\]`).FindStringSubmatch(out)
	if key == nil {
		t.Fatalf("no fingerprint of an error in:\n%s", out)
	}
	who := []string{"--reason", "sample reason", "--by", "sample-user"}
	for _, args := range [][]string{
		append([]string{"approve", "--project", dir, "--finding", key[1]}, who...),
		append([]string{"approve", "--project", dir, "--rule", "V-106", "--devices", "*"}, who...),
		append([]string{"approve", "--project", dir, "--finding", keys[0][1], "--devices", "*"}, who...),
	} {
		if code, _, errOut := do(args...); code != 2 || errOut == "" {
			t.Errorf("%v = %d, stderr %q", args, code, errOut)
		}
	}
	code, out, errOut = do(append([]string{"approve", "--project", dir, "--finding", keys[0][1]}, who...)...)
	if code != 0 || !strings.Contains(out, "承認 a-0001 を足しました（当たった指摘 1 件）") {
		t.Errorf("approve = %d: %s%s", code, out, errOut)
	}
	code, out, errOut = do(append([]string{"approve", "--project", dir, "--rule", "V-202", "--devices", "*", "--table", "interfaces"}, who...)...)
	if code != 0 || !strings.Contains(out, "承認 a-0002 を足しました") {
		t.Errorf("approve a scope = %d: %s%s", code, out, errOut)
	}
	code, out, _ = do("check", "--project", dir)
	if code != 1 {
		t.Errorf("check with errors left = %d", code)
	}
	wants(t, "check after the approvals", out, "未承認の警告 0、承認済み ", "エラー V-106 ix-c01")
	if strings.Contains(out, "警告 V-202") {
		t.Errorf("approved findings are still listed:\n%s", out)
	}
	if code, out, errOut = do("approve", "--project", dir, "--remove", "a-0002"); code != 0 || !strings.Contains(out, "取り消しました") {
		t.Errorf("approve --remove = %d: %s%s", code, out, errOut)
	}
	_, out, _ = do("check", "--project", dir)
	wants(t, "check after taking an approval back", out, "承認済み 1、", "警告 V-202")
}
