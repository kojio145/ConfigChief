package importer

import (
	"os"
	"testing"

	"github.com/kojio145/configchief/internal/pack"
)

// The borrow pack tests children_from: the rules written under one line are
// used under other lines too.
func loadBorrow(t *testing.T) *Result {
	t.Helper()
	p, err := pack.Load(os.DirFS("testdata/borrow"))
	if err != nil {
		t.Fatalf("pack: %v", err)
	}
	for _, w := range p.Warnings {
		t.Errorf("pack warning: %s", w)
	}
	text, err := os.ReadFile("testdata/borrow/tests/import/basic/input.txt")
	if err != nil {
		t.Fatal(err)
	}
	res, err := Import(p, defs(t), "basic.txt", string(text))
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	return res
}

func TestBorrowValues(t *testing.T) {
	res := loadBorrow(t)
	want := []struct{ table, key, col, val string }{
		{"bgp_neighbors", "device=rt-d01,vrf=sample-a,neighbor=192.0.2.1", "description", "first peer"},
		// Where the rule is borrowed, "zone" is not in sight and reads as "".
		{"bgp_neighbors", "device=rt-d01,vrf=,neighbor=192.0.2.2", "remote_as", "64497"},
		// The children of a borrowed rule come along.
		{"bgp_neighbors", "device=rt-d01,vrf=,neighbor=192.0.2.2", "description", "second peer"},
		// The rule of the place itself wins over a borrowed one.
		{"bgp_neighbors", "device=rt-d01,vrf=,neighbor=192.0.2.3", "enabled", "false"},
		// A borrowed "set" writes to the row that is open where it is borrowed.
		{"vrfs", "device=rt-d01,name=global", "description", "the global zone"},
		// A rule without a row of its own may borrow below a rule with one.
		{"bgp_neighbors", "device=rt-d01,vrf=,neighbor=192.0.2.4", "remote_as", "64499"},
	}
	for _, w := range want {
		if got := cell(t, res, w.table, w.key, w.col); got != w.val {
			t.Errorf("%s[%s].%s = %q, want %q", w.table, w.key, w.col, got, w.val)
		}
	}
}

func TestBorrowStates(t *testing.T) {
	res := loadBorrow(t)
	cases := []struct{ text, state, rule string }{
		// The id stays the one the rule was written with.
		{"peer 192.0.2.2 as 64497", Taken, "peer"},
		{"note second peer", Taken, "peer.note"},
		{"peer 192.0.2.3 as 64498 off", Taken, "global.peer.off"},
		{"note the global zone", Taken, "note"},
		{"peer 192.0.2.4 as 64499", Taken, "peer"},
		{"mystery 1", Unknown, ""},
	}
	for _, c := range cases {
		l := line(t, res, c.text)
		if l.State != c.state || l.RuleID != c.rule {
			t.Errorf("%q: state %s by %q, want %s by %q (reason: %s)", c.text, l.State, l.RuleID, c.state, c.rule, l.Reason)
		}
	}
	if c := res.Count(); c.Unknown != 1 {
		t.Errorf("unknown = %d, want 1", c.Unknown)
	}
}
