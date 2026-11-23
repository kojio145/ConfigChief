package packs

import (
	"testing"

	"github.com/kojio145/configchief/internal/pack"
)

// The names of interfaces a device of each bundled model may have (V-111,
// appendix C 1.1): the ports of the model, the logical interfaces the pack
// knows, and their subinterfaces. Upper and lower case do not matter.
func TestFitsModel(t *testing.T) {
	cases := []struct {
		pack, model string
		fit, not    []string
	}{
		{"cisco_ios", "isr-xe-8ge-2ge",
			[]string{"GigabitEthernet0/0/0", "gigabitethernet0/0/1", "GigabitEthernet0/1/7", "GigabitEthernet0/0/0.100", "Vlan10", "Loopback0", "Tunnel5", "Dialer1", "Port-channel1", "Port-channel1.20", "BVI1", "Virtual-Template1", "Null0"},
			[]string{"GigabitEthernet0/1/8", "GigabitEthernet0/0/2", "GigabitEthernet0/2/0.10", "FastEthernet0", "Serial0/1/0", "Unknown1"}},
		{"cisco_ios", "isr-4fe-1fe",
			[]string{"FastEthernet0", "FastEthernet3", "FastEthernet4", "FastEthernet4.10", "Vlan1"},
			[]string{"FastEthernet5", "GigabitEthernet0"}},
		{"nec_ix", "ix-2ge-8sw",
			[]string{"GigaEthernet0.0", "GigaEthernet0.1", "GigaEthernet1.3", "GigaEthernet2:port8", "GigaEthernet2.0", "GigaEthernet2:3.0", "GigaEthernet2:3.1", "Loopback0.0", "Tunnel10.0", "BVI0", "Null0.0", "USB-Serial0.0"},
			[]string{"GigaEthernet2:port9", "GigaEthernet3.0", "GigaEthernet1:port1", "GigaEthernet1:2.0"}},
		{"nec_ix", "ix-4ge",
			[]string{"GigaEthernet3.0", "GigaEthernet3.2"},
			[]string{"GigaEthernet4.0", "GigaEthernet2:port1", "GigaEthernet2:1.0", "GigaEthernet2.0x"}},
		{"nec_qx", "qx-24ge-4xge",
			[]string{"GigabitEthernet1/0/1", "GigabitEthernet1/0/24", "Ten-GigabitEthernet1/0/28", "Vlan-interface100", "LoopBack0", "Bridge-Aggregation1", "NULL0", "Null0"},
			[]string{"GigabitEthernet1/0/25", "Ten-GigabitEthernet1/0/49", "GigabitEthernet2/0/1", "Route-Aggregation1"}},
	}
	loaded := map[string]*pack.Pack{}
	for _, c := range cases {
		p := loaded[c.pack]
		if p == nil {
			fsys, err := FS(c.pack)
			if err != nil {
				t.Fatal(err)
			}
			if p, err = pack.Load(fsys); err != nil {
				t.Fatal(err)
			}
			loaded[c.pack] = p
		}
		m := p.Model(c.model)
		if m == nil {
			t.Fatalf("%s has no model %s", c.pack, c.model)
		}
		for _, n := range c.fit {
			if !p.FitsModel(m, n) {
				t.Errorf("%s %s: %s does not fit", c.pack, c.model, n)
			}
		}
		for _, n := range c.not {
			if p.FitsModel(m, n) {
				t.Errorf("%s %s: %s fits", c.pack, c.model, n)
			}
		}
	}
	// ix-4ge is the one model that says what it does not have (G-607).
	p := loaded["nec_ix"]
	if u := p.Model("ix-4ge").Unsupported; len(u) != 1 || u[0].Table != "switchports" {
		t.Errorf("ix-4ge unsupported = %+v", u)
	}
}
