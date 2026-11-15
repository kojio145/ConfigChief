package caserule

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
	"github.com/kojio145/configchief/internal/schemas"
	"github.com/kojio145/configchief/internal/tmpl"
)

func load(t *testing.T) *schema.Set {
	t.Helper()
	sc, err := schemas.Load()
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

func parse(t *testing.T, sc *schema.Set, name string) ([]*Rule, []string) {
	t.Helper()
	data, err := os.ReadFile("testdata/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return Parse("rules/"+name, data, sc, tmpl.New(tmpl.Options{}))
}

func rules(t *testing.T, sc *schema.Set, name string) map[string]*Rule {
	t.Helper()
	rs, problems := parse(t, sc, name)
	if len(problems) != 0 {
		t.Fatalf("%s: %v", name, problems)
	}
	if d := Doubled(rs); len(d) != 0 {
		t.Fatalf("%s: %v", name, d)
	}
	out := map[string]*Rule{}
	for _, r := range rs {
		out[r.ID] = r
	}
	return out
}

func TestParse(t *testing.T) {
	sc := load(t)
	rs := rules(t, sc, "rows.yaml")
	for id, want := range map[string][2]string{"P-201": {Expr, "G-612"}, "P-202": {Unique, "G-612"}, "P-203": {Equal, "G-612"}} {
		if r := rs[id]; r == nil || r.Kind != want[0] || r.CheckID() != want[1] || r.OfLines() {
			t.Errorf("%s = %+v", id, r)
		}
	}
	if r := rs["P-204"]; !r.InDevice || !reflect.DeepEqual(r.Devices, []string{"rt-a01", "rt-a02"}) || r.Table != "users" || r.Column != "secret" {
		t.Errorf("P-204 = %+v", r)
	}
	if r := rs["P-201"]; !reflect.DeepEqual(r.Devices, []string{"*"}) || r.File != "rules/rows.yaml" {
		t.Errorf("P-201 = %+v", r)
	}
	ls := rules(t, sc, "lines.yaml")
	if r := ls["P-003"]; r.Kind != Require || r.CheckID() != "G-610" || !r.OfLines() || r.Under == "" {
		t.Errorf("P-003 = %+v", r)
	}
	if r := ls["P-004"]; r.Kind != Forbid || r.CheckID() != "G-611" || r.Severity != check.Error {
		t.Errorf("P-004 = %+v", r)
	}

	// An empty file and a file without rules are no fault.
	for _, text := range []string{"", "# nothing yet\n", "rules: []\n"} {
		if rs, problems := Parse("rules/empty.yaml", []byte(text), sc, tmpl.New(tmpl.Options{})); len(rs) != 0 || len(problems) != 0 {
			t.Errorf("%q: %v, %v", text, rs, problems)
		}
	}
}

func TestParseFaults(t *testing.T) {
	sc := load(t)
	rs, problems := parse(t, sc, "bad.yaml")
	if len(rs) != 0 {
		t.Errorf("%d rules read from a file where every rule is wrong", len(rs))
	}
	// Each rule of the file has its fault told, with the place.
	for _, want := range []string{
		"rules/bad.yaml の 1 件目: id が要ります",
		"rules/bad.yaml の G-610: id「G-610」は、ツールのチェックと同じ形",
		"P-103: title が要ります",
		"P-104: severity が要ります",
		"P-105: severity「fatal」は使えません",
		"P-106: 種類",
		"P-107: 種類",
		"P-108: forbid_line の正規表現を読めません",
		"P-109: under の正規表現を読めません",
		"P-110: require_line には table と where を書けません",
		"P-111: 表「no_such_table」は、表の定義にありません",
		"P-112: expr の式を読めません",
		"P-113: where の式を読めません",
		"P-114: 表「interfaces」に、列「no_such_column」はありません",
		"P-115: column が要ります",
		"P-116: scope「site」は使えません",
		"P-117: equal_across には scope を書けません",
		"P-118: expr には under を書けません",
		"P-119: unique では、table と where は unique の中に書きます",
		"P-120: require_line の正規表現が空です",
	} {
		found := false
		for _, p := range problems {
			found = found || strings.Contains(p, want)
		}
		if !found {
			t.Errorf("no fault told like %q", want)
		}
	}
	if t.Failed() {
		t.Log(strings.Join(problems, "\n"))
	}

	// A field the tool does not know, and a text that is no YAML.
	for _, text := range []string{
		"rules:\n  - id: P-001\n    title: x\n    severity: error\n    require: '^end$'\n",
		"rules: [\n",
	} {
		if rs, problems := Parse("rules/x.yaml", []byte(text), sc, tmpl.New(tmpl.Options{})); len(rs) != 0 || len(problems) != 1 || !strings.HasPrefix(problems[0], "rules/x.yaml: ") {
			t.Errorf("%q: %v, %v", text, rs, problems)
		}
	}

	// The same ID twice, in one file or in two.
	a, _ := Parse("rules/a.yaml", []byte("rules:\n  - {id: P-001, title: x, severity: error, require_line: '^end$'}\n"), sc, tmpl.New(tmpl.Options{}))
	b, _ := Parse("rules/b.yaml", []byte("rules:\n  - {id: P-001, title: y, severity: info, forbid_line: '^end$'}\n"), sc, tmpl.New(tmpl.Options{}))
	if d := Doubled(append(a, b...)); len(d) != 1 || !strings.Contains(d[0], "rules/b.yaml の P-001") || !strings.Contains(d[0], "rules/a.yaml") {
		t.Errorf("doubled = %v", d)
	}
}

const masked = "＊＊＊"

func sample(t *testing.T) Text {
	t.Helper()
	data, err := os.ReadFile("testdata/sample.txt")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimRight(string(data), "\n"), "\n")
	return Text{
		Lines: lines,
		Shown: func(n int) string {
			// As a generation does: the value of a secret cell is covered.
			return strings.ReplaceAll(lines[n-1], "dummy-admin-secret-sample", masked)
		},
		Comment: func(body string) bool { return strings.HasPrefix(body, "!") },
	}
}

func TestLines(t *testing.T) {
	sc := load(t)
	rs := rules(t, sc, "lines.yaml")
	text := sample(t)
	// Rule → the lines of its findings (0 = the configuration as a whole).
	want := map[string][]int{
		"P-001": nil,
		"P-002": {0},
		"P-003": {11, 15},
		"P-004": {24, 27},
		"P-005": nil,
		"P-006": nil,
		"P-007": {21},
		"P-008": {16},
		"P-009": nil,
		"P-010": {4},
		"P-011": nil,
		"P-012": nil,
	}
	if len(want) != len(rs) {
		t.Fatalf("%d rules, %d expected", len(rs), len(want))
	}
	keys := map[string]string{}
	for id, lines := range want {
		r := rs[id]
		fs := r.Lines("rt-a01", text)
		var got []int
		for _, f := range fs {
			got = append(got, f.Line)
			if f.RuleID != r.CheckID() || f.CaseRule != id || f.Severity != r.Severity || f.Device != "rt-a01" ||
				!strings.HasPrefix(f.Message, id+" "+r.Title+": ") || f.Evidence[0] != (model.KV{Name: "ルール", Value: id}) {
				t.Errorf("%s: %+v", id, f)
			}
			if other, dup := keys[f.Key]; dup {
				t.Errorf("%s line %d has the key of %s", id, f.Line, other)
			}
			keys[f.Key] = fmt.Sprintf("%s line %d", id, f.Line)
			if strings.Contains(fmt.Sprint(f), "dummy-admin-secret-sample") {
				t.Errorf("%s: the finding holds a secret: %+v", id, f)
			}
		}
		if !reflect.DeepEqual(got, lines) {
			t.Errorf("%s: findings at lines %v, want %v", id, got, lines)
		}
	}

	// What a finding shows of the text.
	f := rs["P-004"].Lines("rt-a01", text)[0]
	if !reflect.DeepEqual(f.Evidence[1:], []model.KV{{Name: "行", Value: "transport input telnet ssh"}, {Name: "上の行", Value: "line vty 0 4"}}) {
		t.Errorf("P-004: %+v", f.Evidence)
	}
	f = rs["P-003"].Lines("rt-a01", text)[0]
	if !reflect.DeepEqual(f.Evidence[1:], []model.KV{{Name: "正規表現", Value: "^no ip proxy-arp$"}, {Name: "上の行", Value: "interface GigabitEthernet0/1"}}) {
		t.Errorf("P-003: %+v", f.Evidence)
	}
	f = rs["P-010"].Lines("rt-a01", text)[0]
	if f.Evidence[1].Value != "username admin secret "+masked {
		t.Errorf("P-010: %+v", f.Evidence)
	}

	// The same line under the same parent is one finding; a finding stays
	// the same finding when lines above it come and go.
	twice := text
	twice.Lines = append([]string{"!", "!"}, text.Lines...)
	twice.Lines = append(twice.Lines[:26:26], " transport input telnet ssh", " transport input telnet ssh")
	twice.Shown = func(n int) string { return twice.Lines[n-1] }
	fs := rs["P-004"].Lines("rt-a01", twice)
	if len(fs) != 1 || fs[0].Line != 26 || fs[0].Key != rs["P-004"].Lines("rt-a01", text)[0].Key {
		t.Errorf("P-004 on the text with a line twice: %+v", fs)
	}
	// Another device, another finding.
	if a, b := rs["P-002"].Lines("rt-a01", text), rs["P-002"].Lines("rt-a02", text); a[0].Key == b[0].Key {
		t.Error("P-002: the same key on two devices")
	}
}

// dataset makes design data of lines "table col=value col=value ...". The
// items of a list are parted by "|"; "~" stands for a space.
func dataset(t *testing.T, sc *schema.Set, lines ...string) *model.Dataset {
	t.Helper()
	ds := model.NewDataset("book")
	for _, line := range lines {
		words := strings.Fields(line)
		tab := sc.Table(words[0])
		if tab == nil {
			t.Fatalf("no table %s", words[0])
		}
		vals := map[string]string{}
		for _, w := range words[1:] {
			k, v, _ := strings.Cut(w, "=")
			if tab.Column(k) == nil {
				t.Fatalf("no column %s.%s", words[0], k)
			}
			vals[k] = strings.ReplaceAll(v, "~", " ")
		}
		var key []model.KV
		for _, k := range tab.Key {
			key = append(key, model.KV{Name: k, Value: vals[k]})
		}
		at := model.Origin{Kind: "book", Sheet: words[0], Addr: fmt.Sprintf("A%d", len(ds.Tables)+2)}
		row := ds.Open(words[0], key, at)
		for k, v := range vals {
			if tab.Column(k).List {
				for _, item := range strings.Split(v, "|") {
					row.Append(k, item, at)
				}
			} else {
				row.Set(k, v, at)
			}
		}
	}
	return ds
}

func rowsData(t *testing.T, sc *schema.Set) *Data {
	ds := dataset(t, sc,
		"devices hostname=rt-a01 vendor=cisco_ios group=core",
		"devices hostname=rt-a02 vendor=cisco_ios group=core|edge",
		"devices hostname=rt-a03 vendor=nec_ix",
		"interfaces device=rt-a01 name=GigabitEthernet0/0 enabled=true description=to~rt-a02 ipv4=192.0.2.1/30",
		"interfaces device=rt-a01 name=GigabitEthernet0/1 enabled=true ipv4=198.51.100.1/24",
		"interfaces device=rt-a01 name=GigabitEthernet0/2 enabled=false",
		"interfaces device=rt-a02 name=GigabitEthernet0/0 enabled=true description=to~rt-a01 ipv4=192.0.2.2/30",
		"interfaces device=rt-a02 name=GigabitEthernet0/1 enabled=true description=sample~lan ipv4=198.51.100.1/24",
		"interfaces device=rt-a03 name=GigaEthernet0.0 enabled=true description=sample~wan ipv4=203.0.113.1/30 mtu=1400",
		"interfaces device=rt-a03 name=GigaEthernet1.0 enabled=true description=sample~lan~2",
		"ntp_servers device=rt-a01 server=192.0.2.123",
		"ntp_servers device=rt-a01 server=192.0.2.124",
		"ntp_servers device=rt-a02 server=192.0.2.124",
		"ntp_servers device=rt-a02 server=192.0.2.123",
		"ntp_servers device=rt-a03 server=192.0.2.123",
		"users device=rt-a01 name=admin secret=dummy-secret-sample-1",
		"users device=rt-a01 name=operator secret=dummy-secret-sample-1",
		"users device=rt-a02 name=admin secret=dummy-secret-sample-1",
		"users device=rt-a03 name=admin secret=dummy-secret-sample-1",
	)
	eng := tmpl.New(tmpl.Options{IfName: func(s string) (string, error) { return s, nil }})
	return &Data{Schema: sc, All: ds, Project: map[string]any{"name": "sample"},
		Engine: func(string) *tmpl.Engine { return eng }}
}

func vendorOf(ds *model.Dataset) func(string) string {
	return func(host string) string {
		for _, r := range ds.Tables["devices"].Rows {
			if r.Get("hostname") == host {
				return r.Get("vendor")
			}
		}
		return ""
	}
}

func TestRows(t *testing.T) {
	sc := load(t)
	rs := rules(t, sc, "rows.yaml")
	d := rowsData(t, sc)
	want := map[string]struct {
		targets  string
		looked   int
		findings []string // severity, device, key of the row, column
	}{
		"P-201": {"rt-a01 rt-a02 rt-a03", 6, []string{"warning rt-a01 name=GigabitEthernet0/1 "}},
		"P-202": {"rt-a01 rt-a02 rt-a03", 5, []string{"error rt-a02 name=GigabitEthernet0/1 ipv4"}},
		"P-203": {"rt-a01 rt-a02 rt-a03", 3, []string{"warning rt-a03  server"}},
		"P-204": {"rt-a01 rt-a02", 3, []string{"warning rt-a01 name=operator secret"}},
		"P-205": {"rt-a03", 2, []string{"info rt-a03 name=GigaEthernet0.0 "}},
		"P-206": {"rt-a01 rt-a02", 5, nil},
		"P-207": {"rt-a01 rt-a02 rt-a03", 1, []string{"error rt-a01 name=GigabitEthernet0/0 "}},
		"P-208": {"rt-a01", 3, nil},
		"P-209": {"", 0, nil},
		"P-210": {"rt-a03", 0, nil},
	}
	if len(want) != len(rs) {
		t.Fatalf("%d rules, %d expected", len(rs), len(want))
	}
	for id, w := range want {
		r := rs[id]
		hosts := r.Targets(d.All, vendorOf(d.All))
		if got := strings.Join(hosts, " "); got != w.targets {
			t.Errorf("%s: for %q, want %q", id, got, w.targets)
		}
		looked := 0
		fs := r.Rows(d, hosts, func(host string, n int) { looked += n })
		var got []string
		for _, f := range fs {
			got = append(got, fmt.Sprintf("%s %s %s %s", f.Severity, f.Device, f.Ref.Key, f.Ref.Column))
			if f.RuleID != "G-612" || f.CaseRule != id || !strings.HasPrefix(f.Message, id+" "+r.Title+": ") || f.Evidence[0].Value != id {
				t.Errorf("%s: %+v", id, f)
			}
			if strings.Contains(fmt.Sprint(f), "dummy-secret-sample-1") {
				t.Errorf("%s: the finding holds a secret: %+v", id, f)
			}
		}
		if !reflect.DeepEqual(got, w.findings) || looked != w.looked {
			t.Errorf("%s: looked at %d, findings %q; want %d, %q", id, looked, got, w.looked, w.findings)
			for _, f := range fs {
				t.Logf("  %s %+v", f.Message, f.Evidence)
			}
		}
	}

	look := func(string, int) {}
	all := []string{"rt-a01", "rt-a02", "rt-a03"}
	// What the findings tell.
	f := rs["P-202"].Rows(d, all, look)[0]
	if !strings.Contains(f.Message, "「IPv4 アドレス」の値が、rt-a01 GigabitEthernet0/1 にもあります") || f.At == "" ||
		!reflect.DeepEqual(f.Evidence[1:], []model.KV{{Name: "値", Value: "198.51.100.1/24"}, {Name: "先の行", Value: "rt-a01 GigabitEthernet0/1"}}) {
		t.Errorf("P-202: %s %+v at %q", f.Message, f.Evidence, f.At)
	}
	f = rs["P-203"].Rows(d, all, look)[0]
	if !strings.Contains(f.Message, "ほかの 2 台と違います") ||
		!reflect.DeepEqual(f.Evidence[1:], []model.KV{{Name: "この機器", Value: "192.0.2.123"}, {Name: "多い方", Value: "192.0.2.123, 192.0.2.124"}}) {
		t.Errorf("P-203: %s %+v", f.Message, f.Evidence)
	}
	f = rs["P-204"].Rows(d, all, look)[0]
	if f.Evidence[1].Value != check.Seal("dummy-secret-sample-1") {
		t.Errorf("P-204: %+v", f.Evidence)
	}
	// A rule that cannot be run is an error whatever it says, once.
	f = rs["P-207"].Rows(d, all, look)[0]
	if !f.Firm || !strings.Contains(f.Message, "expr の式を調べられません") {
		t.Errorf("P-207: %+v", f)
	}

	// Of two sets as many, the one of the device that comes first stands.
	fs := rs["P-203"].Rows(d, []string{"rt-a03", "rt-a01"}, look)
	if len(fs) != 1 || fs[0].Device != "rt-a01" {
		t.Errorf("P-203 on two devices: %+v", fs)
	}
	// A device without a row has the empty set.
	fs = rs["P-203"].Rows(d, []string{"rt-a01", "rt-a02", "rt-a03"}[:2], look)
	if len(fs) != 0 {
		t.Errorf("P-203 on the two that agree: %+v", fs)
	}
	d.All.Tables["ntp_servers"].Rows = d.All.Tables["ntp_servers"].Rows[:4]
	d2 := &Data{Schema: sc, All: d.All, Engine: d.Engine}
	fs = rs["P-203"].Rows(d2, all, look)
	if len(fs) != 1 || fs[0].Device != "rt-a03" || fs[0].Evidence[1].Value != "（なし）" {
		t.Errorf("P-203 with a device without a server: %+v", fs)
	}
	// The rows of a device that is left out are not looked at.
	if fs := rs["P-202"].Rows(d2, []string{"rt-a02", "rt-a03"}, look); len(fs) != 0 {
		t.Errorf("P-202 without the first device: %+v", fs)
	}
}
