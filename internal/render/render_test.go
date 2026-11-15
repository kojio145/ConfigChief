package render_test

import (
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/pack"
	"github.com/kojio145/configchief/internal/packs"
	"github.com/kojio145/configchief/internal/packtest"
	"github.com/kojio145/configchief/internal/render"
	"github.com/kojio145/configchief/internal/schemas"
)

// cases runs the generation tests of a bundled pack (the sample data is in
// the folder of the pack).
func cases(t *testing.T, id string) map[string]packtest.RenderCase {
	t.Helper()
	sc, err := schemas.Load()
	if err != nil {
		t.Fatal(err)
	}
	fsys, err := packs.FS(id)
	if err != nil {
		t.Fatal(err)
	}
	p, err := pack.Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	list, err := packtest.RunRender(p, sc, fsys)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]packtest.RenderCase{}
	for _, c := range list {
		out[c.Name] = c
	}
	return out
}

func find(t *testing.T, o *render.Output, prefix string) (int, render.Line) {
	t.Helper()
	for i, l := range o.Lines {
		if strings.HasPrefix(l.Text, prefix) {
			return i + 1, l
		}
	}
	t.Fatalf("no line starts with %q", prefix)
	return 0, render.Line{}
}

func refs(list []render.CellRef) []string {
	var out []string
	for _, r := range list {
		out = append(out, r.Row.Table+"."+r.Column)
	}
	return out
}

func has(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// Each line tells the template that wrote it and the cells it came from
// (spec 9.3).
func TestLineOrigins(t *testing.T) {
	o := cases(t, "cisco_ios")["basic"].Output
	if o == nil {
		t.Fatal("no test \"basic\"")
	}
	_, l := find(t, o, "hostname ")
	if l.Section != "system" || l.Template != "system.j2" || l.TLine != 3 || l.Extra {
		t.Errorf("hostname line: %+v", l)
	}
	if got := refs(l.Values); len(got) != 1 || got[0] != "devices.hostname" {
		t.Errorf("hostname values = %v", got)
	}

	_, l = find(t, o, " ip address 203.0.113.9 ")
	if l.Section != "interfaces" || l.Template != "interfaces.j2" {
		t.Errorf("address line: %+v", l)
	}
	if got := refs(l.Values); len(got) != 1 || got[0] != "interfaces.ipv4" {
		t.Errorf("address values = %v", got)
	}
	if l.Values[0].Row.Get("name") != "GigabitEthernet0/0" {
		t.Errorf("address row = %s", l.Values[0].Row.KeyText())
	}

	// A line without a value has the cell that decided it as a condition.
	_, l = find(t, o, " no ip proxy-arp")
	if len(l.Values) != 0 || !has(refs(l.Conds), "interfaces.proxy_arp") {
		t.Errorf("proxy-arp line: values %v, conds %v", refs(l.Values), refs(l.Conds))
	}
}

// A line shown in a report covers the values of the secret columns.
func TestShownCoversSecrets(t *testing.T) {
	o := cases(t, "cisco_ios")["basic"].Output
	n, l := find(t, o, "username backup ")
	if !has(refs(l.Values), "users.secret") {
		t.Fatalf("values = %v", refs(l.Values))
	}
	secret := l.Values[len(l.Values)-1].Row.Get("secret")
	if secret == "" || !strings.Contains(l.Text, secret) {
		t.Fatalf("the line does not hold the secret: %q", l.Text)
	}
	shown := o.Shown(n)
	if strings.Contains(shown, secret) || !strings.Contains(shown, render.Masked) || !strings.HasPrefix(shown, "username backup ") {
		t.Errorf("shown = %q", shown)
	}
	n, _ = find(t, o, "snmp-server community ")
	if s := o.Shown(n); !strings.HasSuffix(s, render.Masked+" RO") {
		t.Errorf("shown = %q", s)
	}
	n, l = find(t, o, "interface Loopback0")
	if o.Shown(n) != l.Text {
		t.Errorf("a line without a secret changed: %q", o.Shown(n))
	}
}

// The lines of an extra configuration are marked, and go where the first
// line of the file says.
func TestExtraLines(t *testing.T) {
	c := cases(t, "cisco_ios")["extra"]
	o := c.Output
	if o == nil {
		t.Fatal("no test \"extra\"")
	}
	_, l := find(t, o, "ip multicast-routing")
	if !l.Extra || l.Template != "extra/_all.j2" || l.TLine != 2 || l.Section != "interfaces" {
		t.Errorf("line of _all.j2: %+v", l)
	}
	_, l = find(t, o, "service timestamps ")
	if !l.Extra || l.Template != "extra/@sample-edge.j2" || l.TLine != 1 || l.Section != "extra" {
		t.Errorf("line of the group: %+v", l)
	}
	_, l = find(t, o, "interface ")
	if l.Extra {
		t.Errorf("line of the pack: %+v", l)
	}
	for _, f := range c.Findings {
		switch f.RuleID {
		case "G-609":
			if f.Line == 0 || o.Lines[f.Line-1].Text != "ntp server 192.0.2.124" || !o.Lines[f.Line-1].Extra {
				t.Errorf("G-609 at line %d", f.Line)
			}
		case "G-602":
			if f.Ref.Table != "acls" || f.Ref.Key == "" {
				t.Errorf("G-602 ref = %+v", f.Ref)
			}
		}
	}
}

// A failed device keeps the lines that were written whole, and is not read
// back.
func TestFailed(t *testing.T) {
	c := cases(t, "cisco_ios")["failed"]
	if c.Output == nil || !c.Output.Failed {
		t.Fatal("the test \"failed\" did not fail")
	}
	for _, l := range c.Output.Lines {
		if strings.HasPrefix(l.Text, "snmp-server host") || strings.HasPrefix(l.Text, "username") {
			t.Errorf("a line that was not written whole: %q", l.Text)
		}
	}
	for _, f := range c.Findings {
		if f.RuleID == "G-602" || f.RuleID == "G-604" || f.RuleID == "G-605" {
			t.Errorf("%s on a failed device: %s", f.RuleID, f.Message)
		}
		if f.Device != "rt-c05" || f.Key == "" {
			t.Errorf("finding %+v", f)
		}
	}
}
