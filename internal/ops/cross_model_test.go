package ops

import (
	"os"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/convert"
)

// A small router with built-in switch ports goes to the other vendor with
// the model of the new device known: the interface map is proposed from the
// model (appendix B 6.3), nothing is left for a person to name. Then the way
// to the end: the configuration that was generated is what the device shows,
// and neither the comparison of the book with it nor the check after the
// configuration went in has anything to tell — also not the columns only a
// person fills in (the peer of a port).
func TestBuiltInSwitchByModel(t *testing.T) {
	e := newCase(t, map[string]string{"input/old/rt-m01.txt": siteFile(t, "model", "rt-m01.txt")})
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	setCell(t, e, "新旧対照表（機器）", "旧ホスト名", "rt-m01", "新ホスト名", "ix-m01")
	setCell(t, e, "新旧対照表（機器）", "旧ホスト名", "rt-m01", "新メーカー", "nec_ix")
	setCell(t, e, "新旧対照表（機器）", "旧ホスト名", "rt-m01", "新機種", "ix-2ge-8sw")
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	_, bk, err := e.readBook()
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"FastEthernet0": "GigaEthernet2:port1", "FastEthernet1": "GigaEthernet2:port2", "FastEthernet2": "GigaEthernet2:port3", "FastEthernet3": "GigaEthernet2:port4",
		"FastEthernet4": "-", "FastEthernet5": "-", "FastEthernet6": "-", "FastEthernet7": "-", "FastEthernet8": "-",
		"GigabitEthernet0": "GigaEthernet0.0", "GigabitEthernet0.100": "GigaEthernet0.1",
		"Vlan1": "GigaEthernet2:1.0", "Vlan20": "GigaEthernet2:3.0", "Loopback0": "Loopback0.0",
	}
	if len(bk.Ifs) != len(want) {
		t.Errorf("新旧対照表（ポート） has %d rows, not %d", len(bk.Ifs), len(want))
	}
	for _, m := range bk.Ifs {
		if m.NewIf != want[m.OldIf] || m.Basis != convert.BasisShape("ix-2ge-8sw") {
			t.Errorf("新旧対照表（ポート）: %s → %q (%s), not %q", m.OldIf, m.NewIf, m.Basis, want[m.OldIf])
		}
	}
	b, err := e.MakeBook("old", false, at)
	if err != nil {
		t.Fatal(err)
	}
	if a := b.Applied; len(a.Undecided)+len(a.Cleared)+len(a.Clashes)+len(b.Problems) != 0 {
		t.Fatalf("the mapping: %+v, problems %v", a, b.Problems)
	}
	setCell(t, e, "インタフェース", "IF 名", "GigaEthernet2:port1", "対向機器", "sw-b01")
	setCell(t, e, "インタフェース", "IF 名", "GigaEthernet2:port1", "対向 IF", "GigabitEthernet1/0/1")

	o, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	for _, l := range unsettled(o) {
		if !strings.HasPrefix(l, "G-614 ") {
			t.Errorf("left to a person: %s", l)
		}
	}
	if o.Devices[0].State != GenWritten {
		t.Fatalf("not written: %s %s", o.Devices[0].State, o.Devices[0].Why)
	}
	got := strings.ReplaceAll(read(t, e, o.Devices[0].File), "\r\n", "\n")
	for _, line := range []string{"  vlan-group 1 port 1 2", "  vlan-group 3 port 3 4", "interface GigaEthernet2:3.0", "interface GigaEthernet0.1", "  encapsulation dot1q 100"} {
		if !strings.Contains(got, line+"\n") {
			t.Errorf("the configuration lacks %q:\n%s", line, got)
		}
	}

	// The device shows the configuration: the book and the new side agree.
	if err := os.WriteFile(e.Proj.Path("input/new/ix-m01.txt"), []byte(got), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := e.RunImport("new", "", at); err != nil {
		t.Fatal(err)
	}
	c, err := e.Compare("book", "new", at)
	if err != nil {
		t.Fatal(err)
	}
	// The model stands in the book only (no line of a configuration tells
	// it): the one difference there is.
	if r := c.Result; len(r.OnlyA)+len(r.OnlyB) != 0 || len(r.Cells) != 1 || r.Cells[0].Column != "model" {
		t.Errorf("the book and the new configuration: only A %+v, only B %+v, cells %+v", r.OnlyA, r.OnlyB, r.Cells)
	}

	// And the log taken after the configuration went in.
	if err := os.MkdirAll(e.Proj.Path("input/logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(e.Proj.Path("input/logs/ix-m01.txt"), []byte(got), 0o644); err != nil {
		t.Fatal(err)
	}
	v, err := e.Verify(at)
	if err != nil {
		t.Fatal(err)
	}
	if kinds := verifyKinds(v); len(kinds) != 0 || v.Left() != 0 {
		t.Errorf("the check after: %v", kinds)
	}
}

// proposals gives the interface map a model proposes for one old device.
func proposals(t *testing.T, dir, file, oldHost, newHost, vendor, model string) map[string]string {
	t.Helper()
	e := newCase(t, map[string]string{"input/old/" + file: siteFile(t, dir, file)})
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	setCell(t, e, "新旧対照表（機器）", "旧ホスト名", oldHost, "新ホスト名", newHost)
	setCell(t, e, "新旧対照表（機器）", "旧ホスト名", oldHost, "新メーカー", vendor)
	setCell(t, e, "新旧対照表（機器）", "旧ホスト名", oldHost, "新機種", model)
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	_, bk, err := e.readBook()
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]string{}
	for _, m := range bk.Ifs {
		out[m.OldIf] = m.NewIf
	}
	return out
}

// The other way, and the cases of appendix B 6.3: a VLAN a trunk carries
// gets the form of "logical", a port that is shut down is not carried over,
// a subinterface follows its port, a PPPoE interface follows its port.
func TestInterfaceMapByModel(t *testing.T) {
	for _, c := range []struct {
		dir, file, oldHost, vendor, model string
		want                              map[string]string
	}{
		{"vlan", "ix-v01.txt", "ix-v01", "cisco_ios", "isr-8fe-1fe-1ge", map[string]string{
			"GigaEthernet2:port1": "FastEthernet0", "GigaEthernet2:port2": "FastEthernet1", "GigaEthernet2:port3": "FastEthernet2", "GigaEthernet2:port4": "-",
			"GigaEthernet0.0": "GigabitEthernet0", "GigaEthernet1.0": "FastEthernet8", "GigaEthernet1.1": "FastEthernet8.30",
			"GigaEthernet2:1.0": "Vlan1", "GigaEthernet2:2.0": "Vlan2"}},
		{"vlan", "rt-v01.txt", "rt-v01", "nec_ix", "ix-2ge-8sw", map[string]string{
			"FastEthernet0": "GigaEthernet2:port1", "FastEthernet1": "GigaEthernet2:port2", "FastEthernet2": "GigaEthernet2:port3", "FastEthernet3": "-",
			"FastEthernet4": "GigaEthernet2:port4", "GigabitEthernet8": "GigaEthernet0.0", "Vlan10": "BVI0", "Vlan20": "BVI1"}},
		{"model", "rt-m02.txt", "rt-m02", "nec_ix", "ix-1ge-4sw", map[string]string{
			"GigabitEthernet0": "GigaEthernet0.0", "Dialer1": "GigaEthernet0.1", "FastEthernet0": "GigaEthernet1:port1", "FastEthernet1": "GigaEthernet1:port2",
			"FastEthernet2": "GigaEthernet1:port3", "FastEthernet3": "GigaEthernet1:port4", "FastEthernet4": "", "FastEthernet8": "-",
			"Vlan1": "GigaEthernet1:1.0", "Tunnel1": "Tunnel1.0"}},
		// A switch to a switch: a port the new model has under the same
		// name keeps it; the interface of a LAG and of a VLAN.
		{"qx", "sw-c01.txt", "sw-c01", "nec_qx", "qx-24ge-4xge", map[string]string{
			"GigabitEthernet1/0/1": "GigabitEthernet1/0/1", "GigabitEthernet1/0/2": "GigabitEthernet1/0/2", "GigabitEthernet1/0/3": "-",
			"GigabitEthernet1/0/23": "GigabitEthernet1/0/23", "GigabitEthernet1/0/24": "GigabitEthernet1/0/24",
			"Port-channel1": "Bridge-Aggregation1", "Vlan10": "Vlan-interface10"}},
	} {
		got := proposals(t, c.dir, c.file, c.oldHost, "new-"+c.oldHost, c.vendor, c.model)
		if len(got) != len(c.want) {
			t.Errorf("%s: %d interfaces, not %d: %v", c.file, len(got), len(c.want), got)
		}
		for old, name := range c.want {
			if got[old] != name {
				t.Errorf("%s: %s → %q, not %q", c.file, old, got[old], name)
			}
		}
	}
}
