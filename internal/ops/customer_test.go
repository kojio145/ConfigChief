package ops

import (
	"os"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/check"
)

// A list with one line per device: the LAN side and a VLAN as columns.
const sitesCSV = "拠点の一覧,,,,,\n" +
	"ホスト名,拠点,LAN IP,LAN マスク,VLAN10,回線\n" +
	"rt-a01,site-a,198.51.100.2,255.255.255.0,192.0.2.129/25,sample lan\n" +
	"rt-a09,site-z,198.51.100.66,255.255.255.192,,\n" +
	"rt-a01,site-b,,,,\n"

const sitesMap = `version: 1
file: sites*.csv
header_row: 2
rows:
  - table: devices
    columns:
      - {to: hostname, column: ホスト名}
      - {to: site, column: 拠点}
  - table: interfaces
    label: LAN
    columns:
      - {to: device, column: ホスト名}
      - {to: name, value: GigabitEthernet0/1}
      - {to: ipv4, template: "{LAN IP}/{LAN マスク}", steps: [{type: cidr}]}
      - {to: description, column: 回線}
  - table: interfaces
    label: VLAN 10
    columns:
      - {to: device, column: ホスト名}
      - {to: name, value: GigabitEthernet0/3.10}
      - {to: ipv4, column: VLAN10}
`

func put(t *testing.T, e *Env, rel, text string) {
	t.Helper()
	if err := os.WriteFile(e.Proj.Path(rel), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// listCase is a case whose book holds the sample device rt-a01, with a
// customer's list and its mapping.
func listCase(t *testing.T) *Env {
	t.Helper()
	e := newCase(t, map[string]string{
		"input/new/rt-a01.txt":         sample(t, "cisco_ios"),
		"input/customer/sites_v1.csv":  sitesCSV,
		"input/customer/memo.txt":      "nothing of a list\n",
		"mappings/sites.yaml":          sitesMap,
		"mappings/.hidden.yaml":        "nothing",
		"mappings/readme_not_yaml.txt": "nothing",
	})
	if _, err := e.MakeBook("new", false, at); err != nil {
		t.Fatal(err)
	}
	return e
}

func byRule(o *CustomerOutcome, rule string) []Judged {
	var out []Judged
	for _, f := range o.Verdicts.Items {
		if f.RuleID == rule {
			out = append(out, f)
		}
	}
	return out
}

func TestCustomer(t *testing.T) {
	e := listCase(t)
	o, err := e.Customer(true, at)
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Lists) != 1 || o.Lists[0].File != "input/customer/sites_v1.csv" || o.Lists[0].Err != "" {
		t.Fatalf("lists = %+v", o.Lists)
	}
	if r := o.Lists[0].Result; r.Lines != 3 || r.Rows != 5 {
		t.Errorf("lines %d, rows %d", r.Lines, r.Rows)
	}
	if len(o.Unmapped) != 1 || o.Unmapped[0] != "input/customer/memo.txt" || len(o.Notes) != 0 {
		t.Errorf("unmapped %v, notes %v", o.Unmapped, o.Notes)
	}
	// Only the list: the device rt-a09 and its LAN side.
	if s := byRule(o, "S-501"); len(s) != 2 || s[0].Device != "rt-a09" || s[0].Ref.Table != "devices" || s[1].Ref.Table != "interfaces" || s[0].Severity != check.Warning {
		t.Errorf("S-501 = %+v", s)
	}
	// Different: the site (the book says nothing) and the address of the
	// VLAN. The LAN address and its description are the same.
	s := byRule(o, "S-503")
	if len(s) != 2 || s[0].Ref.Column != "site" || s[0].Evidence[0].Value != "site-a" || s[0].Evidence[1].Value != "" ||
		s[1].Ref.Column != "ipv4" || s[1].Ref.Key != "name=GigabitEthernet0/3.10" || s[1].Evidence[1].Value != "192.0.2.130/25" || s[1].Severity != check.Error {
		t.Errorf("S-503 = %+v", s)
	}
	if !strings.Contains(s[1].At, "!") {
		t.Errorf("no cell of the book for %+v", s[1])
	}
	// Only the book: the other interfaces of rt-a01, and nothing else (the
	// users, the routes ... are no table the list speaks of).
	ifs := len(o.Design.Tables["interfaces"].Rows)
	if b := byRule(o, "S-502"); len(b) != ifs-2 || b[0].Ref.Table != "interfaces" || b[0].Severity != check.Info {
		t.Errorf("S-502: %d findings, %d interfaces in the book", len(b), ifs)
	}
	// Not read: the site of rt-a01 written a second time with another value.
	if p := byRule(o, "S-504"); len(p) != 1 || p[0].Device != "rt-a01" || p[0].Evidence[2].Value != "B5" || !strings.Contains(p[0].Message, "先に別の値") {
		t.Errorf("S-504 = %+v", p)
	}
	if c := o.Verdicts.Count(); c.Errors != 2 || c.Warnings != 3 || o.Findings() != 5 {
		t.Errorf("count = %+v, findings %d", c, o.Findings())
	}
	h := read(t, e, o.Report.HTML)
	for _, want := range []string{"外部一覧とパラメータブックの比較", "rt-a09", "192.0.2.129/25", "sites_v1.csv の「CSV」E3", "S-503", "memo.txt は、どの列対応ファイルにも当たらない"} {
		if !strings.Contains(h, want) {
			t.Errorf("the report lacks %q", want)
		}
	}

	// The same list again: nothing changed. Then the list changes.
	if o, err = e.Customer(true, at); err != nil || len(o.Changed) != 0 {
		t.Fatalf("read again: %v, changed %+v", err, o.Changed)
	}
	put(t, e, "input/customer/sites_v1.csv", strings.Replace(sitesCSV, "rt-a01,site-b,,,,\n", "", 1))
	if o, err = e.Customer(false, at); err != nil || len(o.Changed) != 1 || o.Changed[0].RuleID != "S-505" || len(o.Problems) != 0 {
		t.Fatalf("a changed list: %v, changed %+v, problems %+v", err, o.Changed, o.Problems)
	}
	// Without writing, the mark stays: the change is told again.
	if o, err = e.Customer(true, at); err != nil || len(o.Changed) != 1 {
		t.Fatalf("a changed list, written: %v, changed %+v", err, o.Changed)
	}
	if o, err = e.Customer(true, at); err != nil || len(o.Changed) != 0 {
		t.Fatalf("after the mark was written: %v, changed %+v", err, o.Changed)
	}
}

func TestCustomerApproveReflect(t *testing.T) {
	e := listCase(t)
	o, err := e.Customer(false, at)
	if err != nil {
		t.Fatal(err)
	}
	site, addr := byRule(o, "S-503")[0], byRule(o, "S-503")[1]

	// The book is right about the site: approved, by its fingerprint.
	a, err := e.Approve(ApproveRequest{Finding: site.Key[:8], Reason: "the list is old", By: "tester"}, at)
	if err != nil || !a.Source || a.Migration || a.Approval.Rule != "S-503" {
		t.Fatalf("approve: %v, %+v", err, a)
	}
	// The list is right about the address, and about the new device.
	r, err := e.Reflect(ReflectRequest{Findings: []string{addr.Key}}, at)
	if err != nil || r.Cells != 1 || r.Rows != 0 || r.Skipped() != 0 || r.Backup == "" {
		t.Fatalf("reflect a cell: %v, %+v", err, r)
	}
	if r, err = e.Reflect(ReflectRequest{Rule: "S-501", Devices: "rt-a09"}, at); err != nil || r.Rows != 2 || r.Cells != 0 {
		t.Fatalf("reflect rows: %v, %+v", err, r)
	}
	if o, err = e.Customer(false, at); err != nil {
		t.Fatal(err)
	}
	if len(o.OnlyList) != 0 || len(o.Cells) != 1 || o.Verdicts.Count().Errors != 0 || o.Verdicts.Count().Approved != 1 {
		t.Errorf("after: only list %d, cells %d, count %+v", len(o.OnlyList), len(o.Cells), o.Verdicts.Count())
	}
	_, bk, err := e.readBook()
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, row := range bk.Data.Tables["interfaces"].Rows {
		if row.Get("name") == "GigabitEthernet0/3.10" || row.Get("device") == "rt-a09" {
			got = append(got, row.Get("device")+" "+row.Get("ipv4"))
		}
	}
	if strings.Join(got, "|") != "rt-a01 192.0.2.129/25|rt-a09 198.51.100.66/26" {
		t.Errorf("the book: %q", got)
	}
	if n := len(bk.Data.Tables["devices"].Rows); n != 2 {
		t.Errorf("%d devices in the book", n)
	}
	// What is left can be approved by a scope: S-503 is the one error that
	// can, and the warnings can.
	if _, err := e.Approve(ApproveRequest{Rule: "S-504", Devices: "*", Reason: "known", By: "tester"}, at); err != nil {
		t.Fatal(err)
	}
	if o, err = e.Customer(false, at); err != nil || o.Verdicts.Open() != 0 || len(o.Verdicts.Approvals()) != 2 {
		t.Errorf("after the approvals: %v, open %d", err, o.Verdicts.Open())
	}

	// A fingerprint that is not there, and a check that is not for the book.
	if _, err := e.Reflect(ReflectRequest{Findings: []string{"0123456789"}}, at); err == nil {
		t.Error("no error for a fingerprint that is not there")
	}
	if _, err := e.Reflect(ReflectRequest{Rule: "S-504"}, at); err == nil {
		t.Error("no error for S-504")
	}
}

// A value of a row written for every device is not changed for one device.
func TestReflectSharedRow(t *testing.T) {
	e := listCase(t)
	addRow(t, e, "NTP サーバ", map[string]string{"機器": "*", "サーバ": "192.0.2.125"})
	put(t, e, "input/customer/ntp.csv", "機器,NTP,優先\nrt-a01,192.0.2.125,○\n")
	put(t, e, "mappings/ntp.yaml", `version: 1
file: ntp.csv
header_row: 1
rows:
  - table: NTP サーバ
    columns:
      - {to: 機器, column: 機器}
      - {to: サーバ, column: NTP}
      - {to: 優先, column: 優先}
`)
	o, err := e.Customer(false, at)
	if err != nil {
		t.Fatal(err)
	}
	var f *Judged
	for i, x := range o.Verdicts.Items {
		if x.RuleID == "S-503" && x.Ref.Table == "ntp_servers" {
			f = &o.Verdicts.Items[i]
		}
	}
	if f == nil || f.Ref.Column != "prefer" || f.Evidence[0].Value != "true" {
		t.Fatalf("no finding for the NTP server: %+v", o.Verdicts.Items)
	}
	r, err := e.Reflect(ReflectRequest{Findings: []string{f.Key}}, at)
	if err != nil || r.Cells != 0 || r.Skipped() != 1 || r.Backup != "" || !strings.Contains(r.Items[0].Skipped, "機器 rt-a01 の名前で書いた行がありません") {
		t.Fatalf("reflect: %v, %+v", err, r)
	}
}

// A list that holds the names of the old devices.
func TestCustomerOldNames(t *testing.T) {
	e := newCase(t, map[string]string{"input/old/rt-a01.txt": sample(t, "cisco_ios")})
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	setCell(t, e, "機器対応", "旧ホスト名", "rt-a01", "新ホスト名", "rt-n01")
	setCell(t, e, "機器対応", "旧ホスト名", "rt-a01", "新メーカー", "cisco_ios")
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	_, bk, err := e.readBook()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range bk.Ifs {
		to := m.OldIf
		if to == "GigabitEthernet0/1" {
			to = "GigabitEthernet0/0/1"
		}
		setCell(t, e, "IF 対応", "旧 IF 名", m.OldIf, "新 IF 名", to)
	}
	if _, err := e.MakeBook("old", false, at); err != nil {
		t.Fatal(err)
	}
	put(t, e, "input/customer/ports.csv", "機器,ポート,アドレス\nrt-a01,GigabitEthernet0/1,198.51.100.2/24\nrt-a01,GigabitEthernet0/9,198.51.100.129/25\nrt-z01,GigabitEthernet0/1,198.51.100.193/26\n")
	put(t, e, "mappings/ports.yaml", `version: 1
file: ports.csv
header_row: 1
names: old
rows:
  - table: interfaces
    columns:
      - {to: device, column: 機器}
      - {to: name, column: ポート}
      - {to: ipv4, column: アドレス}
`)
	o, err := e.Customer(false, at)
	if err != nil {
		t.Fatal(err)
	}
	// The LAN side is the same under the new names. The port the mapping
	// does not know stays as it is, and the device it does not know is left
	// out; both are told.
	if len(o.Cells) != 0 || len(o.OnlyList) != 1 || o.OnlyList[0].Device != "rt-n01" || o.OnlyList[0].Row.Get("name") != "GigabitEthernet0/9" {
		t.Errorf("cells %+v, only list %+v", o.Cells, o.OnlyList)
	}
	notes := strings.Join(o.Notes, "\n")
	if len(o.Notes) != 2 || !strings.Contains(notes, "機器 rt-z01 は、「機器対応」で新ホスト名が決まっていない") || !strings.Contains(notes, "IF GigabitEthernet0/9 は、「IF 対応」で新 IF 名が決まっていない") {
		t.Errorf("notes:\n%s", notes)
	}
}

func TestCustomerFaults(t *testing.T) {
	e := newCase(t, nil)
	if _, err := e.Customer(false, at); err == nil || !strings.Contains(err.Error(), "列対応ファイル（.yaml）がありません") {
		t.Errorf("no mapping: %v", err)
	}
	put(t, e, "mappings/a.yaml", "version: 1\nfile: a.csv\nheader_row: 1\nrows:\n  - table: devices\n    columns: [{to: site, column: 拠点}]\n")
	put(t, e, "mappings/b.yml", "version: 1\n")
	_, err := e.Customer(false, at)
	if err == nil || !strings.Contains(err.Error(), "キーの列「ホスト名」がありません") || !strings.Contains(err.Error(), "b.yml") {
		t.Errorf("faults: %v", err)
	}
	// A mapping without its list, a list without the header, and no book.
	os.Remove(e.Proj.Path("mappings/b.yml"))
	put(t, e, "mappings/a.yaml", "version: 1\nfile: a.csv\nheader_row: 1\nrows:\n  - table: devices\n    columns: [{to: hostname, column: 機器名}, {to: site, column: 拠点}]\n")
	put(t, e, "mappings/c.yaml", "version: 1\nfile: c.csv\nheader_row: 1\nrows:\n  - table: devices\n    columns: [{to: hostname, column: 機器名}]\n")
	put(t, e, "input/customer/a.csv", "機器名,場所\nrt-a01,site-a\n")
	o, err := e.Customer(true, at)
	if err != nil {
		t.Fatal(err)
	}
	notes := strings.Join(o.Notes, "\n")
	for _, want := range []string{"見出し「拠点」がありません", "「c.csv」に当たるファイルがありません", "parameters.xlsx がないので、一覧と比べていません"} {
		if !strings.Contains(notes, want) {
			t.Errorf("no note with %q in:\n%s", want, notes)
		}
	}
	if o.Rows() != 1 || o.Findings() != 3 || len(o.Verdicts.Items) != 0 {
		t.Errorf("rows %d, findings %d, items %+v", o.Rows(), o.Findings(), o.Verdicts.Items)
	}
	// A way to start: the draft of a mapping.
	put(t, e, "input/customer/list.csv", "No,ホスト名,設置場所\n1,rt-a01,site-a\n")
	path, err := e.DraftMapping("list.csv", "")
	if err != nil || path != "mappings/list.yaml" || !strings.Contains(read(t, e, path), `{to: site, column: "設置場所"`) {
		t.Fatalf("draft: %v, %s", err, path)
	}
	if _, err := e.DraftMapping("list.csv", ""); err == nil {
		t.Error("a draft was written over")
	}
}
