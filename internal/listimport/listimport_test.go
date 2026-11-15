package listimport

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
	"github.com/kojio145/configchief/internal/schemas"
	"github.com/kojio145/configchief/internal/sheet"
)

func defs(t *testing.T) *schema.Set {
	t.Helper()
	sc, err := schemas.Load()
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

func rule(t *testing.T, text string) *Rule {
	t.Helper()
	r, err := Parse([]byte(text))
	if err != nil {
		t.Fatal(err)
	}
	if p := r.Validate(defs(t)); len(p) > 0 {
		t.Fatalf("the rule has faults:\n%s", strings.Join(p, "\n"))
	}
	return r
}

func run(t *testing.T, s *sheet.Sheet, r *Rule) (*model.Dataset, *Result) {
	t.Helper()
	tab, err := ReadTable(s, r)
	if err != nil {
		t.Fatal(err)
	}
	ds := model.NewDataset("customer")
	return ds, Convert(tab, r, defs(t), "input/customer/sites.xlsx", ds)
}

// One line per device, the roles of the interfaces as columns.
const wide = `
version: 1
file: sites.xlsx
sheet: 拠点一覧
header_row: 2
exclude:
  - {column: 状態, op: equals, value: 廃止}
rows:
  - table: devices
    columns:
      - {to: hostname, column: ホスト名, steps: [{type: trim}, {type: width}]}
      - {to: site, column: 拠点}
  - table: インタフェース
    label: WAN 側
    columns:
      - {to: 機器, column: ホスト名, steps: [{type: trim}, {type: width}]}
      - {to: IF 名, value: GigabitEthernet0/0}
      - {to: ipv4, template: "{WAN IP}/{WAN マスク}", steps: [{type: cidr}]}
      - {to: enabled, value: "true"}
  - table: interfaces
    label: LAN 側
    columns:
      - {to: device, column: ホスト名, steps: [{type: trim}, {type: width}]}
      - {to: name, value: GigabitEthernet0/1}
      - {to: ipv4, column: LAN IP}
      - {to: description, column: 用途, default: LAN}
`

func wideSheet() *sheet.Sheet {
	return &sheet.Sheet{Name: "拠点一覧", Rows: [][]string{
		{"拠点の一覧"},
		{"拠点", "ホスト名", "WAN IP", "WAN マスク", "LAN IP", "用途", "状態"},
		{"東京", " rt-a01 ", "192.0.2.1", "255.255.255.252", "198.51.100.1/24", "事務所", ""},
		{"", "ｒｔ－ａ０２", "192.0.2.5", "255.255.255.252", "", "", ""}, // 東京 again (merged), no LAN side
		{"大阪", "rt-b01", "192.0.2.9", "255.255.0.255", "198.51.100.300/24", "", ""},
		{"大阪", "rt-b09", "192.0.2.13", "255.255.255.252", "", "", "廃止"},
		{"", "", "", "", "198.51.100.129/25", "", ""}, // no host name
	}, Merges: []sheet.Range{{R1: 2, C1: 0, R2: 3, C2: 0}}}
}

func TestConvertWide(t *testing.T) {
	ds, res := run(t, wideSheet(), rule(t, wide))
	want := `devices:
  - hostname: "rt-a01"
    site: "東京"
  - hostname: "rt-a02"
    site: "東京"
  - hostname: "rt-b01"
    site: "大阪"
interfaces:
  - device: "rt-a01"
    name: "GigabitEthernet0/0"
    enabled: "true"
    ipv4: "192.0.2.1/30"
  - device: "rt-a01"
    name: "GigabitEthernet0/1"
    description: "事務所"
    ipv4: "198.51.100.1/24"
  - device: "rt-a02"
    name: "GigabitEthernet0/0"
    enabled: "true"
    ipv4: "192.0.2.5/30"
`
	if got := ds.Dump(); got != want {
		t.Errorf("the design data:\n%s\nwant:\n%s", got, want)
	}
	if res.Lines != 5 || res.Excluded != 1 || res.Rows != 6 {
		t.Errorf("lines %d, excluded %d, rows %d", res.Lines, res.Excluded, res.Rows)
	}
	// rt-b01: a mask that is none (WAN) and an address that is none (LAN);
	// the last line: an address without a host name. The line of rt-a02 has
	// no LAN address: no row, and no problem (the default alone says nothing).
	var got []string
	for _, p := range res.Problems {
		got = append(got, p.Addr+" "+p.Table+"."+p.Column+" "+p.Device)
	}
	wantP := []string{"C5 interfaces.ipv4 ", "E5 interfaces.ipv4 ", "E7 interfaces. "}
	if strings.Join(got, "|") != strings.Join(wantP, "|") {
		t.Errorf("problems %q, want %q", got, wantP)
		for _, p := range res.Problems {
			t.Log(p.Addr, p.Text)
		}
	}
	// The origin of a cell is its cell of the sheet.
	row := ds.Tables["interfaces"].Rows[1]
	if o := row.Cells["ipv4"].Origins[0]; o.Kind != "customer" || o.Sheet != "拠点一覧" || o.Addr != "E3" || o.Line != 3 {
		t.Errorf("origin %+v", o)
	}
}

// One line per interface: a device on several lines, and a second list that
// speaks of the same rows.
const long = `
version: 1
file: ports.csv
header_row: 1
names: old
rows:
  - table: devices
    columns:
      - {to: hostname, column: 機器}
      - {to: model, column: 機種, fallback: [型番]}
  - table: interfaces
    columns:
      - {to: device, column: 機器}
      - {to: name, column: ポート, steps: [{type: regex, pattern: "^GE", to: "GigabitEthernet"}]}
      - {to: ipv4, column: アドレス}
      - {to: speed, column: 速度, steps: [{type: map, rows: [{from: 1G, to: "1000"}, {from: 自動, to: auto}]}]}
      - {to: dhcp_relay, column: 中継先}
`

func TestConvertLong(t *testing.T) {
	s := &sheet.Sheet{Name: "Sheet1", Rows: [][]string{
		{"機器", "機種", "型番", "ポート", "アドレス", "速度", "中継先"},
		{"rt-a01", "", "model-x", "GE0/0", "192.0.2.1/30", "1G", ""},
		{"rt-a01", "model-y", "", "GE0/1", "198.51.100.1/24", "自動", "198.51.100.10、198.51.100.11"},
		{"rt-a01", "", "", "GE0/1", "198.51.100.2/24", "", ""},
	}}
	r := rule(t, long)
	if !r.Old() {
		t.Error("names: old was not read")
	}
	ds, res := run(t, s, r)
	want := `devices:
  - hostname: "rt-a01"
    model: "model-x"
interfaces:
  - device: "rt-a01"
    name: "GigabitEthernet0/0"
    ipv4: "192.0.2.1/30"
    speed: "1000"
  - device: "rt-a01"
    name: "GigabitEthernet0/1"
    dhcp_relay: ["198.51.100.10", "198.51.100.11"]
    ipv4: "198.51.100.1/24"
    speed: "auto"
`
	if got := ds.Dump(); got != want {
		t.Errorf("the design data:\n%s\nwant:\n%s", got, want)
	}
	// The model written twice with two values, and the address of Gi0/1.
	if len(res.Problems) != 2 || res.Problems[0].Addr != "B3" || res.Problems[1].Addr != "E4" || res.Problems[1].Device != "rt-a01" {
		t.Errorf("problems %+v", res.Problems)
	}
	if res.Rows != 3 {
		t.Errorf("rows %d, want 3", res.Rows)
	}
}

func TestMissingColumn(t *testing.T) {
	s := &sheet.Sheet{Name: "Sheet1", Rows: [][]string{
		{"機器", "ポート", "IP"},
		{"rt-a01", "GE0/0", "192.0.2.1/30"},
	}}
	ds, res := run(t, s, rule(t, long))
	if strings.Join(res.Missing, ",") != "アドレス,中継先,型番,機種,速度" {
		t.Errorf("missing %q", res.Missing)
	}
	// The device is read; the interface says nothing and gives no row.
	if got := ds.Dump(); got != "devices:\n  - hostname: \"rt-a01\"\n" {
		t.Errorf("the design data:\n%s", got)
	}
}

func TestValidate(t *testing.T) {
	r, err := Parse([]byte(`
version: 2
file: a/b.xlsx
header_row: 0
names: both
exclude:
  - {column: "", op: like}
rows:
  - table: nothing
    columns: [{to: x, column: y}]
  - table: interfaces
    columns:
      - {to: device, column: 機器, value: rt-a01}
      - {to: ipv4, column: IP, steps: [{type: guessOS}]}
      - {to: ipv4, column: IP}
      - {to: colour, column: 色}
      - {to: mtu, fallback: [MTU]}
`))
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(r.Validate(defs(t)), "\n")
	for _, w := range []string{"version は 1", "フォルダは書けません", "header_row", "names「both」", "exclude 1: column", "op「like」",
		"「nothing」という表はありません", "値の元を column・template・value のどれか 1 つ", "加工「guessOS」は使えません", "2 回書かれています",
		"「colour」という列はありません", "fallback は column と一緒に", "キーの列「IF 名」がありません"} {
		if !strings.Contains(got, w) {
			t.Errorf("no fault with %q in:\n%s", w, got)
		}
	}
	if _, err := Parse([]byte("version: 1\nfile: a.csv\nheader: 3\n")); err == nil {
		t.Error("an unknown word was read")
	}
}

func TestLoad(t *testing.T) {
	fsys := fstest.MapFS{
		"b.yaml":    {Data: []byte(wide)},
		"a.yaml":    {Data: []byte(long)},
		".x.yaml":   {Data: []byte("nothing")},
		"notes.txt": {Data: []byte("nothing")},
	}
	rules, problems := Load(fsys, "mappings", defs(t))
	if len(problems) > 0 || len(rules) != 2 || rules[0].Path != "mappings/a.yaml" || rules[1].Rows[1].Table != "interfaces" {
		t.Fatalf("rules %+v, problems %q", rules, problems)
	}
	tg := Targeted(rules)
	if !tg["devices"]["site"] || !tg["devices"]["model"] || !tg["interfaces"]["speed"] || tg["interfaces"]["mtu"] {
		t.Errorf("targeted %v", tg)
	}
	fsys["c.yml"] = &fstest.MapFile{Data: []byte(wide)}
	fsys["d.yaml"] = &fstest.MapFile{Data: []byte("version: 1\nfile: x.csv\nheader_row: 1\nrows: []\n")}
	rules, problems = Load(fsys, "mappings", defs(t))
	if rules != nil || len(problems) != 2 || !strings.Contains(problems[0], "c.yml") || !strings.Contains(problems[1], "mappings/d.yaml: rows") {
		t.Errorf("rules %v, problems %q", rules, problems)
	}
	// The text a rule is written as reads back the same.
	r := rule(t, wide)
	text, err := Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	back := rule(t, string(text))
	again, _ := Marshal(back)
	if string(text) != string(again) {
		t.Errorf("written:\n%s\nread and written again:\n%s", text, again)
	}
}

func TestCIDR(t *testing.T) {
	for in, want := range map[string]string{
		"192.0.2.1 255.255.255.0":       "192.0.2.1/24",
		"192.0.2.1/255.255.255.252":     "192.0.2.1/30",
		" 192.0.2.1 / 255.255.255.255 ": "192.0.2.1/32",
		"192.0.2.1/0.0.0.0":             "192.0.2.1/0",
		"192.0.2.1/24":                  "192.0.2.1/24",
		"192.0.2.1/255.0.255.0":         "192.0.2.1/255.0.255.0",
		"192.0.2.1":                     "192.0.2.1",
		"dhcp":                          "dhcp",
		"":                              "",
	} {
		if got := cidr(in); got != want {
			t.Errorf("cidr(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDraft(t *testing.T) {
	s := wideSheet()
	text := Draft(s, "sites.xlsx")
	for _, w := range []string{"header_row: 2\n", `file: "sites.xlsx"`, "#   C 列: WAN IP\n", `{to: hostname, column: "ホスト名", steps: [{type: trim}]}`, `{to: site, column: "拠点"`} {
		if !strings.Contains(text, w) {
			t.Errorf("the draft lacks %q:\n%s", w, text)
		}
	}
	// A draft is a rule that reads.
	r := rule(t, text)
	ds, _ := run(t, s, r)
	if n := len(ds.Tables["devices"].Rows); n != 4 {
		t.Errorf("%d devices, want 4", n)
	}
}
