package cli

import (
	"path/filepath"
	"regexp"
	"testing"
	"time"
)

// The customer's list from the command line: a draft of the mapping, the
// comparison with the book, the values into the book, an approval.
func TestCustomer(t *testing.T) {
	now = func() time.Time { return time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC) }
	defer func() { now = time.Now }()
	dir := filepath.Join(t.TempDir(), "sample-case")
	if code, _, errOut := do("init", dir, "--kind", "new"); code != 0 {
		t.Fatalf("init = %d: %s", code, errOut)
	}
	put(t, filepath.Join(dir, "input", "new", "rt-a01.txt"), sample(t, "cisco_ios"))
	if code, _, errOut := do("book", "--project", dir, "--from", "new"); code != 0 {
		t.Fatalf("book = %d: %s", code, errOut)
	}

	// No mapping yet.
	code, _, errOut := do("customer", "--project", dir)
	if code != 2 {
		t.Fatalf("customer without a mapping = %d", code)
	}
	wants(t, "the error", errOut, "列の割り当てファイル（.yaml）がありません")

	put(t, filepath.Join(dir, "input", "customer", "sites.csv"), "ホスト名,拠点,VLAN10\nrt-a01,site-a,192.0.2.129/25\nrt-a09,site-z,\n")
	code, out, errOut := do("customer", "--project", dir, "--draft", "sites.csv")
	if code != 0 {
		t.Fatalf("draft = %d: %s", code, errOut)
	}
	wants(t, "the draft", out, "mappings/sites.yaml")
	if code, _, _ = do("customer", "--project", dir, "--draft", "sites.csv"); code != 2 {
		t.Errorf("a second draft = %d", code)
	}
	put(t, filepath.Join(dir, "mappings", "sites.yaml"), `version: 1
file: sites.csv
header_row: 1
rows:
  - table: 機器
    columns:
      - {to: ホスト名, column: ホスト名}
      - {to: 拠点, column: 拠点}
  - table: インタフェース
    columns:
      - {to: 機器, column: ホスト名}
      - {to: IF 名, value: GigabitEthernet0/3.10}
      - {to: IPv4 アドレス, column: VLAN10}
`)
	code, out, errOut = do("customer", "--project", dir)
	if code != 1 {
		t.Fatalf("customer = %d: %s%s", code, out, errOut)
	}
	wants(t, "the output", out, "一覧にだけある行（S-501）: 1", "値が違うセル（S-503）: 2", "未承認のエラー 2 件、未承認の警告 1 件",
		"[一覧=192.0.2.129/25] [ブック=192.0.2.130/25]", "out/2026-01-02_030405/reports/customer.xlsx")
	m := regexp.MustCompile(`S-503 rt-a01 インタフェース.*（指紋 ([0-9a-f]+)）`).FindStringSubmatch(out)
	if m == nil {
		t.Fatalf("no fingerprint in:\n%s", out)
	}

	// The address goes into the book, the new device too; the site is approved.
	code, out, errOut = do("customer", "--project", dir, "--apply", m[1][:8])
	if code != 0 {
		t.Fatalf("apply = %d: %s%s", code, out, errOut)
	}
	wants(t, "the output", out, "足した行 0、書き換えたセル 1", "state/backup/")
	if code, out, errOut = do("customer", "--project", dir, "--apply-rule", "S-501"); code != 0 {
		t.Fatalf("apply-rule = %d: %s%s", code, out, errOut)
	}
	wants(t, "the output", out, "足した行 1、書き換えたセル 0")
	code, out, errOut = do("approve", "--project", dir, "--rule", "S-503", "--devices", "rt-a01", "--table", "機器", "--reason", "the list is old", "--by", "tester")
	if code != 0 {
		t.Fatalf("approve = %d: %s%s", code, out, errOut)
	}
	wants(t, "the output", out, "当たった指摘 1 件", "次に外部一覧を読んだとき")
	code, out, _ = do("customer", "--project", dir)
	if code != 0 {
		t.Fatalf("customer after = %d:\n%s", code, out)
	}
	wants(t, "the output", out, "未承認のエラー 0 件、未承認の警告 0 件、承認済み 1 件")

	// What cannot be asked for.
	for _, args := range [][]string{
		{"customer", "--project", dir, "--apply", "x", "--apply-rule", "S-501"},
		{"customer", "--project", dir, "--sheet", "x"},
		{"customer", "--project", dir, "--apply-rule", "S-504"},
		{"customer", "--project", dir, "--apply", "0123456789"},
	} {
		if code, _, _ := do(args...); code != 2 {
			t.Errorf("%v = %d, want 2", args[2:], code)
		}
	}
}
