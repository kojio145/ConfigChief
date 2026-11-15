package importer

import (
	"os"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/pack"
)

// The joins pack tests what ties far-apart lines together: lists on one line,
// a table for the whole device, the network that covers an address, and a
// derive rule that walks a table of the design data.
func loadJoins(t *testing.T) *Result {
	t.Helper()
	p, err := pack.Load(os.DirFS("testdata/joins"))
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	for _, w := range p.Warnings {
		t.Errorf("pack warning: %s", w)
	}
	text, err := os.ReadFile("testdata/joins/tests/import/basic/input.txt")
	if err != nil {
		t.Fatal(err)
	}
	res, err := Import(p, defs(t), "basic.txt", string(text))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	return res
}

func TestJoinsValues(t *testing.T) {
	res := loadJoins(t)
	gi1 := "device=rt-c01,name=GigabitEthernet0/1"
	want := []struct{ table, key, col, val string }{
		// Blanks and commas both part the values of a list.
		{"devices", "hostname=rt-c01", "name_servers", "192.0.2.53,192.0.2.54,192.0.2.55"},
		// So does a list that a derive rule appends.
		{"interfaces", gi1, "dhcp_relay", "192.0.2.67,192.0.2.68"},
		// Of two networks that hold the address, the smaller one wins.
		{"ospf_interfaces", "device=rt-c01,process_id=1,if_name=GigabitEthernet0/1", "area", "2"},
		{"ospf_interfaces", "device=rt-c01,process_id=1,if_name=GigabitEthernet0/2", "area", "0"},
		// A work table with an empty key holds one row for the device.
		{"ntp_servers", "device=rt-c01,server=192.0.2.123", "source_if", "Loopback0"},
		{"ntp_servers", "device=rt-c01,server=192.0.2.124", "source_if", "Loopback0"},
	}
	for _, w := range want {
		if got := cell(t, res, w.table, w.key, w.col); got != w.val {
			t.Errorf("%s[%s].%s = %q, want %q", w.table, w.key, w.col, got, w.val)
		}
	}
	// No network holds the address of Loopback0.
	if n := len(res.Data.Tables["ospf_interfaces"].Rows); n != 2 {
		t.Errorf("ospf_interfaces: %d rows, want 2", n)
	}
}

func TestJoinsStates(t *testing.T) {
	res := loadJoins(t)
	cases := []struct {
		text, state, reasonHas string
	}{
		{"common source Lo0", Taken, ""},
		// The same row of the work table, but nothing reads this value.
		{"common spare sample-text", Unknown, "spare"},
		// Shadowed by a smaller network, and holding no address at all.
		{"area-net 198.51.100.0/24 area 1", Unknown, "x_net"},
		{"area-net 198.51.100.0/25 area 2", Taken, ""},
		{"area-net 203.0.113.0/30 area 0", Taken, ""},
		{"area-net 192.0.2.128/25 area 9", Unknown, "x_net"},
		{"relay-servers Gi0/1 192.0.2.67 192.0.2.68", Taken, ""},
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
	if c := res.Count(); c.Unknown != 3 {
		t.Errorf("unknown = %d, want 3", c.Unknown)
	}
}

// A value that a derive rule adds to a row of the design data points back at
// the line it came from, not only at the line that made the row.
func TestJoinsOrigins(t *testing.T) {
	res := loadJoins(t)
	src := line(t, res, "common source Lo0")
	for _, r := range res.Data.Tables["ntp_servers"].Rows {
		c := r.Cells["source_if"]
		if c == nil || len(c.Origins) == 0 {
			t.Fatalf("%s: no origin for source_if", r.KeyText())
		}
		found := false
		for _, o := range c.Origins {
			if o.Kind != "derived" {
				t.Errorf("%s: origin kind %q", r.KeyText(), o.Kind)
			}
			if o.Line == src.Line {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: origins %+v do not have line %d", r.KeyText(), c.Origins, src.Line)
		}
	}
	// And the line knows the cells it filled.
	want := map[Ref]bool{
		{"ntp_servers", "device=rt-c01,server=192.0.2.123", "source_if"}: false,
		{"ntp_servers", "device=rt-c01,server=192.0.2.124", "source_if"}: false,
	}
	for _, r := range src.Refs {
		if _, ok := want[r]; ok {
			want[r] = true
		}
	}
	for r, ok := range want {
		if !ok {
			t.Errorf("line %d has no ref to %+v (refs: %+v)", src.Line, r, src.Refs)
		}
	}
}
