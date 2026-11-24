package convert

import (
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/book"
)

// The port map shows the old device and what its interface holds (spec
// 14.19.5).
func TestRefs(t *testing.T) {
	sc := defs(t)
	old := oldData()
	put(old, "interfaces", k("device", "rt-a01", "name", "GigabitEthernet0/0"), "description", "uplink to the sample peer", "acl_in", "ACL-IN")
	devs := []book.DeviceMap{{OldHost: "rt-a01", NewHost: "rt-b01"}, {OldHost: "rt-a02", NewHost: "rt-b01"}}
	ifs := []book.IfMap{
		{NewHost: "rt-b01", OldIf: "GigabitEthernet0/0"},
		{NewHost: "rt-b01", OldIf: "Serial0/0"},
		{NewHost: "rt-b01", OldIf: "Loopback0"},
		{NewHost: "rt-b01", OldIf: "GigabitEthernet9/9"},
	}
	Refs(sc, old, devs, ifs)
	for i, want := range []struct {
		host string
		has  []string
	}{
		{"rt-a01", []string{"説明: uplink to the sample peer", "192.0.2.1/30", "ACL 入: ACL-IN", "使っている所: ", "スタティック"}},
		{"rt-a01", []string{"無効（shutdown）"}},
		{"rt-a01", []string{"192.0.2.255/32", "使っている所: "}},
		{"", nil},
	} {
		m := ifs[i]
		if m.OldHost != want.host {
			t.Errorf("%s: old host %q, want %q", m.OldIf, m.OldHost, want.host)
		}
		for _, s := range want.has {
			if !strings.Contains(m.Now, s) {
				t.Errorf("%s: %q lacks %q", m.OldIf, m.Now, s)
			}
		}
		if want.host == "" && m.Now != "" {
			t.Errorf("%s: %q, want nothing", m.OldIf, m.Now)
		}
	}
	// An interface with nothing set says so.
	put(old, "interfaces", k("device", "rt-a02", "name", "GigabitEthernet0/5"))
	one := []book.IfMap{{NewHost: "rt-b01", OldIf: "GigabitEthernet0/5"}}
	Refs(sc, old, devs, one)
	if one[0].OldHost != "rt-a02" || one[0].Now != "（設定なし）" {
		t.Errorf("empty interface: %+v", one[0])
	}
}
