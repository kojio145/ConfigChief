package importer

import (
	"os"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/pack"
)

// The sample lives in testdata (not in this file) so that the check of public
// files sees its values.
func load(t *testing.T) *Result {
	t.Helper()
	p, err := pack.Load(os.DirFS("testdata/mini"))
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	for _, w := range p.Warnings {
		t.Errorf("pack warning: %s", w)
	}
	text, err := os.ReadFile("testdata/mini/tests/import/basic/input.txt")
	if err != nil {
		t.Fatal(err)
	}
	res, err := Import(p, defs(t), "basic.txt", string(text))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	return res
}

// line finds the one line with the text (indentation left out).
func line(t *testing.T, res *Result, text string) LineState {
	t.Helper()
	var found []LineState
	for _, l := range res.Lines {
		if strings.TrimSpace(l.Text) == text {
			found = append(found, l)
		}
	}
	if len(found) != 1 {
		t.Fatalf("line %q: found %d times", text, len(found))
	}
	return found[0]
}

func cell(t *testing.T, res *Result, table, keyText, col string) string {
	t.Helper()
	tb := res.Data.Tables[table]
	if tb == nil {
		t.Fatalf("no table %s", table)
	}
	for _, r := range tb.Rows {
		if r.KeyText() == keyText {
			return r.Get(col)
		}
	}
	t.Fatalf("no row %s in %s", keyText, table)
	return ""
}

func TestStates(t *testing.T) {
	res := load(t)
	if res.Device != "rt-a01" {
		t.Errorf("device = %q", res.Device)
	}
	cases := []struct {
		text, state, reasonHas string
	}{
		{"! sample configuration of a made-up device", Ignored, "区切り"},
		{"hostname rt-a01", Taken, ""},
		{"banner login ^", Taken, ""},
		{"Sample banner", Taken, ""},
		{"! still the banner", Taken, ""},
		{"^", Taken, ""},
		{"interface GigabitEthernet0/1", Taken, ""},
		{"description to rt-b01", Taken, ""},
		{"helper 198.51.100.11", Taken, ""},
		{"shutdown", Taken, ""},
		{"alias", Taken, ""},
		{"mystery knob 5", Unknown, ""},
		{"deeper mystery", Unknown, "親の行が未対応"},
		{"bad address 192.0.2.77", Unknown, "規則 if.bad"},
		{"tunnel-profile PROF-SAMPLE-A", Taken, ""},
		{"interface Gi0/3", Taken, ""},
		{"tunnel-profile PROF-SAMPLE-MISSING", Unknown, "derive"},
		{"profile PROF-SAMPLE-A", Taken, ""},
		{"peer 203.0.113.1", Taken, ""},
		{"color blue", Unknown, "color"},
		{"profile PROF-SAMPLE-UNUSED", Unknown, "derive"},
		{"peer 203.0.113.2", Unknown, "derive"},
		{"ntp server 198.51.100.20", Taken, ""},
		{"stray child", Unknown, "子の行の規則がありません"},
		{"service nagle", Ignored, "自動"},
		{"line console", Ignored, ""},
		{"speed 9600", Ignored, ""},
		{"user admin role admin", Taken, ""},
		{"user monitor role guest", Unknown, ""},
		{"end", Ignored, "終わり"},
	}
	for _, c := range cases {
		l := line(t, res, c.text)
		if l.State != c.state {
			t.Errorf("%q: state %s, want %s (reason: %s)", c.text, l.State, c.state, l.Reason)
		}
		if c.reasonHas != "" && !strings.Contains(l.Reason, c.reasonHas) {
			t.Errorf("%q: reason %q does not have %q", c.text, l.Reason, c.reasonHas)
		}
	}
	if l := line(t, res, "! still the banner"); l.Separator {
		t.Error("a line inside a block is not a separator")
	}
}

func TestValues(t *testing.T) {
	res := load(t)
	gi1 := "device=rt-a01,name=GigabitEthernet0/1"
	want := []struct{ table, key, col, val string }{
		{"devices", "hostname=rt-a01", "banner", " Sample banner\n! still the banner"},
		{"devices", "hostname=rt-a01", "snmp_location", "Sample room"},
		{"devices", "hostname=rt-a01", "note", "line 33"},
		{"interfaces", gi1, "description", "to rt-b01"},
		{"interfaces", gi1, "ipv4", "192.0.2.5/30"}, // the later line wins
		{"interfaces", gi1, "dhcp_relay", "198.51.100.10,198.51.100.11"},
		{"interfaces", gi1, "enabled", "false"},
		{"interfaces", gi1, "note", "GigabitEthernet0/1/to rt-b01/first"},
		{"interfaces", gi1, "kind", "physical"},
		{"interfaces", gi1, "mtu", "1500"},
		{"interfaces", "device=rt-a01,name=GigabitEthernet0/2", "enabled", "true"},
		{"interfaces", "device=rt-a01,name=Loopback0", "kind", "loopback"},
		{"interfaces", "device=rt-a01,name=Loopback0", "mtu", ""},
		{"tunnels", "device=rt-a01,if_name=GigabitEthernet0/2", "destination", "203.0.113.1"},
		{"tunnels", "device=rt-a01,if_name=GigabitEthernet0/2", "type", "gre"},
		{"users", "device=rt-a01,name=admin", "privilege", "admin"},
	}
	for _, w := range want {
		if got := cell(t, res, w.table, w.key, w.col); got != w.val {
			t.Errorf("%s[%s].%s = %q, want %q", w.table, w.key, w.col, got, w.val)
		}
	}
	if n := len(res.Data.Tables["tunnels"].Rows); n != 1 {
		t.Errorf("tunnels: %d rows, want 1 (a bind without its profile makes no row)", n)
	}
	if n := len(res.Data.Tables["users"].Rows); n != 1 {
		t.Errorf("users: %d rows, want 1", n)
	}
	for name := range res.Data.Tables {
		if strings.HasPrefix(name, "x_") {
			t.Errorf("work table %s is in the design data", name)
		}
	}
	if res.Work.Tables["x_profiles"] == nil || res.Work.Tables["x_tunnel_bind"] == nil {
		t.Error("the work tables are not kept apart for looking into")
	}
}

func TestOriginsAndRefs(t *testing.T) {
	res := load(t)
	l := line(t, res, "ip address 192.0.2.9 255.255.255.252")
	if len(l.Refs) != 1 || l.Refs[0] != (Ref{"interfaces", "device=rt-a01,name=GigabitEthernet0/2", "ipv4"}) {
		t.Errorf("refs = %+v", l.Refs)
	}
	// A derived value points back at every line behind the rows it used.
	var row = res.Data.Tables["tunnels"].Rows[0]
	var lines []int
	for _, o := range row.Cells["destination"].Origins {
		if o.Kind != "derived" {
			t.Errorf("origin kind %q", o.Kind)
		}
		lines = append(lines, o.Line)
	}
	bind := line(t, res, "tunnel-profile PROF-SAMPLE-A").Line
	peer := line(t, res, "peer 203.0.113.1").Line
	has := func(n int) bool {
		for _, l := range lines {
			if l == n {
				return true
			}
		}
		return false
	}
	if !has(bind) || !has(peer) {
		t.Errorf("origins %v do not have lines %d and %d", lines, bind, peer)
	}
}

func TestWarningsAndCount(t *testing.T) {
	res := load(t)
	var twice int
	for _, w := range res.Warnings {
		if strings.Contains(w, "2 回目の値") {
			twice++
		}
	}
	if twice != 1 {
		t.Errorf("warnings about a second value: %d, want 1\n%s", twice, strings.Join(res.Warnings, "\n"))
	}
	c := res.Count()
	if c.Total != len(res.Lines) || c.Separators != 2 {
		t.Errorf("count = %+v", c)
	}
	if c.Taken+c.Ignored+c.Unknown+c.Separators != c.Total {
		t.Errorf("the states do not add up: %+v", c)
	}
	if c.Unknown != 9 {
		for _, l := range res.Lines {
			if l.State == Unknown {
				t.Logf("unknown: %d %s (%s)", l.Line, l.Text, l.Reason)
			}
		}
		t.Errorf("unknown = %d, want 9", c.Unknown)
	}
}

func TestNoEndMarker(t *testing.T) {
	p, err := pack.Load(os.DirFS("testdata/mini"))
	if err != nil {
		t.Fatal(err)
	}
	res, err := Import(p, defs(t), "cut-off.txt", "ntp server 198.51.100.20\n")
	if err != nil {
		t.Fatal(err)
	}
	if res.Device != "cut-off" {
		t.Errorf("device = %q, want the file name", res.Device)
	}
	found := false
	for _, w := range res.Warnings {
		if strings.Contains(w, "終わりの印") {
			found = true
		}
	}
	if !found {
		t.Error("no warning for the missing end marker")
	}
}
