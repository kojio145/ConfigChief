package ops

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

// addRow adds a row to a sheet of the book: heading → value.
func addRow(t *testing.T, e *Env, sheet string, cells map[string]string) {
	t.Helper()
	f, err := excelize.OpenFile(e.Proj.BookPath())
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		t.Fatalf("the sheet %s: %v", sheet, err)
	}
	for head, v := range cells {
		col := index(rows[0], head)
		if col < 0 {
			t.Fatalf("the sheet %s has no heading %s", sheet, head)
		}
		addr, _ := excelize.CoordinatesToCellName(col+1, len(rows)+1)
		if err := f.SetCellValue(sheet, addr, v); err != nil {
			t.Fatal(err)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		t.Fatal(err)
	}
	if err := e.Proj.WriteBook(buf.Bytes()); err != nil {
		t.Fatal(err)
	}
}

func messages(o *GenerateOutcome, rule string) string {
	var sb strings.Builder
	for _, f := range itemsOf(o, rule) {
		sb.WriteString(f.Message + "\n")
	}
	return sb.String()
}

// The configuration of one vendor becomes the configuration of another
// (spec 10, 19): the sample the Cisco pack writes is imported, the mapping is
// put on, and the NEC pack writes it. What the new vendor cannot write is
// told, never dropped (spec 10.5).
func TestCrossVendor(t *testing.T) {
	e := newCase(t, map[string]string{"input/old/rt-c01.txt": rendered(t, "cisco_ios")})
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	setCell(t, e, "新旧対照表（機器）", "旧ホスト名", "rt-c01", "新ホスト名", "ix-c01")
	setCell(t, e, "新旧対照表（機器）", "旧ホスト名", "rt-c01", "新メーカー", "nec_ix")
	// The import after the new host name is known asks for the interfaces.
	in, err := e.RunImport("old", "", at)
	if err != nil {
		t.Fatal(err)
	}
	if in.Book.IfRows != 6 || len(in.Applied.Undecided) != 6 {
		t.Fatalf("interfaces to map: %d rows, %d not decided", in.Book.IfRows, len(in.Applied.Undecided))
	}
	for old, name := range map[string]string{
		"GigabitEthernet0/0":    "GigaEthernet0.0",
		"GigabitEthernet0/1":    "GigaEthernet1.0",
		"GigabitEthernet0/2":    "-", // the port of a sub-interface has no row of its own on the new device
		"GigabitEthernet0/2.30": "GigaEthernet1.1",
		"GigabitEthernet0/3":    "GigaEthernet2.0",
		"Loopback0":             "Loopback0.0",
	} {
		setCell(t, e, "新旧対照表（ポート）", "旧 IF 名", old, "新 IF 名", name)
	}
	b, err := e.MakeBook("old", false, at)
	if err != nil {
		t.Fatal(err)
	}
	if a := b.Applied; len(a.Undecided) != 0 || len(a.Dropped) != 1 || len(a.Cleared) != 0 || len(a.Clashes) != 0 || len(b.Problems) != 0 {
		t.Fatalf("the mapping: %d not decided, %d dropped, %d cleared, %d clashes, problems %v",
			len(a.Undecided), len(a.Dropped), len(a.Cleared), len(a.Clashes), b.Problems)
	}

	// The values in a form the new vendor does not have stop the templates,
	// and one run tells them all.
	o, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Devices) != 1 || o.Devices[0].Pack != "nec_ix" || o.Devices[0].State != GenFailed || !exists(e, outDir+"configs/_failed/ix-c01.txt") {
		t.Fatalf("devices = %+v", o.Devices)
	}
	stops := messages(o, "G-601")
	for _, want := range []string{
		"タイムゾーンは「+9」「+5:30」の形で書きます",
		"ユーザ admin のパスワードは、別メーカーのハッシュです",
		"ユーザ monitor のパスワードは、別メーカーのハッシュです",
		"GigaEthernet1.0 の代理 ARP「無効」は書けません",
	} {
		if !strings.Contains(stops, want) {
			t.Errorf("G-601 lacks %q:\n%s", want, stops)
		}
	}
	if got := messages(o, "G-603"); !strings.Contains(got, "user_role の語「1」に当たるメーカーの語がありません") {
		t.Errorf("G-603 = %q", got)
	}
	if n := len(itemsOf(o, "G-602")); n != 0 {
		t.Errorf("%d findings of G-602 for a device whose templates failed", n)
	}

	// A person writes them the way of the new vendor.
	setCell(t, e, "機器", "ホスト名", "ix-c01", "時刻帯", "+9")
	setCell(t, e, "インタフェース", "IF 名", "GigaEthernet1.0", "代理 ARP", "")
	for _, name := range []string{"admin", "monitor"} {
		setCell(t, e, "ユーザ", "ユーザ名", name, "パスワード", "dummy-"+name+"-secret-sample")
		setCell(t, e, "ユーザ", "ユーザ名", name, "パスワードの形", "plain")
	}
	setCell(t, e, "ユーザ", "ユーザ名", "monitor", "権限", "monitor")
	if o, err = e.Generate(GenOptions{}, at); err != nil {
		t.Fatal(err)
	}
	if o.Devices[0].State != GenWritten {
		t.Fatalf("after the values were mended: %+v\n%s%s", o.Devices[0], messages(o, "G-601"), messages(o, "G-603"))
	}
	got := strings.ReplaceAll(read(t, e, o.Devices[0].File), "\r\n", "\n")
	if os.Getenv("CROSS_WRITE") != "" {
		writeTestFile(filepath.Join("testdata", "cross", "ix-c01.txt"), []byte(got), 0o644)
	}
	want, err := os.ReadFile(filepath.Join("testdata", "cross", "ix-c01.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if got != strings.ReplaceAll(string(want), "\r\n", "\n") {
		t.Errorf("the configuration for the new vendor:\n%s", got)
	}
	// Read back by the pack of the new vendor it gives the book again.
	for _, id := range []string{"G-604", "G-605"} {
		if n := len(itemsOf(o, id)); n != 0 {
			t.Errorf("%s: %d findings: %s", id, n, messages(o, id))
		}
	}
	// What the pack of the new vendor has no line for is an error, row by row.
	unused := messages(o, "G-602")
	for _, want := range []string{
		"ドメイン名、特権パスワード、パスワードの形、バナー",
		"MTU",
		"NAT の向き",
		"優先",
		"SSH の版、認証",
		"説明",
	} {
		if !strings.Contains(unused, want) {
			t.Errorf("G-602 lacks %q:\n%s", want, unused)
		}
	}
	if n := len(itemsOf(o, "G-602")); n != 6 || o.Unverified == "" {
		t.Errorf("%d findings of G-602, mark %q:\n%s", n, o.Unverified, unused)
	}
	// They are approved one by one, not by a scope.
	if _, err := e.Approve(ApproveRequest{Rule: "G-602", Devices: "*", Reason: "sample reason", By: "admin"}, at); err == nil {
		t.Error("a scope for G-602 was taken")
	}

	// The settings that are not carried over: the cells are emptied.
	for _, c := range [][4]string{
		{"機器", "ホスト名", "ix-c01", "ドメイン名"},
		{"機器", "ホスト名", "ix-c01", "特権パスワード"},
		{"機器", "ホスト名", "ix-c01", "パスワードの形"},
		{"機器", "ホスト名", "ix-c01", "バナー"},
		{"インタフェース", "IF 名", "GigaEthernet0.0", "MTU"},
		{"インタフェース", "IF 名", "GigaEthernet1.0", "NAT の向き"},
		{"NTP サーバ", "機器", "ix-c01", "優先"},
		{"遠隔ログイン", "機器", "ix-c01", "SSH の版"},
		{"遠隔ログイン", "機器", "ix-c01", "認証"},
		{"スタティック経路", "機器", "ix-c01", "説明"},
	} {
		setCell(t, e, c[0], c[1], c[2], c[3], "")
	}
	if o, err = e.Generate(GenOptions{}, at); err != nil {
		t.Fatal(err)
	}
	// What is left is told as written in another way (appendix A 3.26): the
	// level of the syslog destinations became the level of the device's log,
	// and the source interface is written as its address. Besides it, only
	// that the device has no model (V-114, information).
	var items []Judged
	for _, f := range o.Items {
		if f.RuleID != "V-114" {
			items = append(items, f)
		}
	}
	if o.Devices[0].State != GenWritten || len(items) != 1 || items[0].RuleID != "G-614" || !strings.Contains(items[0].Message, "ログに残す重要度") || o.Unverified == "" || len(items) == len(o.Items) {
		t.Errorf("after the cells were emptied: %+v, %d findings, mark %q", o.Devices[0], len(o.Items), o.Unverified)
		for _, f := range o.Items {
			t.Logf("  %s %s: %s", f.RuleID, f.Severity, f.Message)
		}
	}
	if text := strings.ReplaceAll(read(t, e, o.Devices[0].File), "\r\n", "\n"); text != got {
		t.Errorf("emptying cells no template reads changed the configuration:\n%s", text)
	}
	// The comparison with the old side keeps what was changed on the way.
	c, err := e.Compare("old", "book", at)
	if err != nil {
		t.Fatal(err)
	}
	changed := map[string]bool{}
	for _, d := range c.Result.Cells {
		changed[d.Table+"."+d.Column] = true
	}
	for _, want := range []string{"devices.timezone", "devices.domain_name", "devices.banner", "interfaces.proxy_arp", "interfaces.nat_role",
		"users.secret", "users.privilege", "ntp_servers.prefer", "remote_access.ssh_version"} {
		if !changed[want] {
			t.Errorf("the comparison old - book lacks %s: %v", want, changed)
		}
	}
	if len(c.Result.OnlyA) != 0 || len(c.Result.OnlyB) != 0 {
		t.Errorf("rows on one side only: %d, %d", len(c.Result.OnlyA), len(c.Result.OnlyB))
	}
}

// The failures of one template are told together, up to a number (spec 9.6).
func TestGenerateManyFailures(t *testing.T) {
	e := genCase(t, map[string]string{"input/new/rt-c01.txt": rendered(t, "cisco_ios")})
	text := strings.Repeat("{{ fail('sample stop') }}\n", 3)
	if err := writeTestFile(e.Proj.Path("extra/rt-c01.j2"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	o, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	if got := messages(o, "G-608"); stateOf(o, "rt-c01") != GenFailed || strings.Count(got, "sample stop") != 3 || strings.Contains(got, "ほかにも誤りがあります") {
		t.Errorf("three failures: %s\n%s", stateOf(o, "rt-c01"), got)
	}
	for i, f := range itemsOf(o, "G-608") {
		if !strings.Contains(f.Message, fmt.Sprintf("%d 行目", i+1)) {
			t.Errorf("failure %d: %s", i+1, f.Message)
		}
	}

	text = strings.Repeat("{{ fail('sample stop') }}\n", 25)
	if err := writeTestFile(e.Proj.Path("extra/rt-c01.j2"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	if o, err = e.Generate(GenOptions{}, at); err != nil {
		t.Fatal(err)
	}
	got := messages(o, "G-608")
	if strings.Count(got, "sample stop") != 20 || strings.Count(got, "ほかにも誤りがあります（1 つのテンプレートで調べるのは 20 件までです") != 1 {
		t.Errorf("25 failures:\n%s", got)
	}
}

// A row for every device holds a value one of the vendors cannot write: the
// devices of that vendor get a row of their own without it (spec 10.5).
func TestCrossVendorSharedRow(t *testing.T) {
	e := genCase(t, map[string]string{
		"input/new/rt-c01.txt": rendered(t, "cisco_ios"),
		"input/new/ix-c01.txt": rendered(t, "nec_ix"),
	})
	// The NEC sample names access lists it does not define (V-106).
	for _, c := range [][2]string{
		{"インタフェース", "入力 ACL"},
		{"インタフェース", "出力 ACL"},
		{"遠隔ログイン", "接続元の制限"},
		{"SNMP コミュニティ", "接続元の制限"},
	} {
		setCell(t, e, c[0], "機器", "ix-c01", c[1], "")
	}
	route := map[string]string{"機器": "*", "宛先": "198.18.50.0/24", "ネクストホップ": "203.0.113.10", "説明": "sample-shared"}
	addRow(t, e, "スタティック経路", route)
	o, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	var hit []string
	for _, f := range itemsOf(o, "G-602") {
		if strings.HasSuffix(f.Message, ": 説明") {
			hit = append(hit, f.Device)
		}
	}
	if len(hit) != 1 || hit[0] != "ix-c01" || stateOf(o, "rt-c01") != GenWritten || stateOf(o, "ix-c01") != GenWritten {
		t.Fatalf("G-602 for the row: %v, devices %s %s\n%s%s", hit, stateOf(o, "rt-c01"), stateOf(o, "ix-c01"), messages(o, "G-602"), messages(o, "G-601"))
	}

	setCell(t, e, "機器", "ホスト名", "ix-c01", "グループ", "sample-ix")
	route["機器"], route["説明"] = "@sample-ix", ""
	addRow(t, e, "スタティック経路", route)
	if o, err = e.Generate(GenOptions{}, at); err != nil {
		t.Fatal(err)
	}
	for _, f := range itemsOf(o, "G-602") {
		if strings.HasSuffix(f.Message, ": 説明") {
			t.Errorf("G-602 although the group has its own row: %s %s", f.Device, f.Message)
		}
	}
	for host, want := range map[string]string{
		"ix-c01": "ip route 198.18.50.0/24 203.0.113.10\n",
		"rt-c01": "ip route 198.18.50.0 255.255.255.0 203.0.113.10 name sample-shared\n",
	} {
		if text := strings.ReplaceAll(read(t, e, outDir+"configs/"+host+".txt"), "\r\n", "\n"); !strings.Contains(text, want) {
			t.Errorf("%s lacks %q:\n%s", host, want, text)
		}
	}
}
