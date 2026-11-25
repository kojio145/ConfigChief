package convert

import (
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/model"
)

// switchData is an old device with a built-in switch (made-up values): the
// interface of VLAN 10, two access ports of it, a trunk and a routed port.
func switchData() *model.Dataset {
	d := model.NewDataset("old")
	put(d, "devices", k("hostname", "rt-a01"), "vendor", "cisco_ios")
	put(d, "interfaces", k("device", "rt-a01", "name", "Vlan10"), "enabled", "true", "ipv4", "192.0.2.1/25", "vlan_id", "10")
	put(d, "interfaces", k("device", "rt-a01", "name", "FastEthernet0/1"), "enabled", "true", "description", "sample pc")
	put(d, "interfaces", k("device", "rt-a01", "name", "FastEthernet0/2"), "enabled", "true")
	put(d, "interfaces", k("device", "rt-a01", "name", "FastEthernet0/3"), "enabled", "true")
	put(d, "interfaces", k("device", "rt-a01", "name", "FastEthernet0/4"), "enabled", "true")
	put(d, "switchports", k("device", "rt-a01", "if_name", "FastEthernet0/1"), "mode", "access", "access_vlan", "10", "voice_vlan", "30")
	put(d, "switchports", k("device", "rt-a01", "if_name", "FastEthernet0/2"), "mode", "access")
	put(d, "switchports", k("device", "rt-a01", "if_name", "FastEthernet0/3"), "mode", "trunk", "allowed_vlans", "10|20", "native_vlan", "1")
	put(d, "switchports", k("device", "rt-a01", "if_name", "FastEthernet0/4"), "mode", "routed")
	return d
}

// The settings of an old interface tell its switch port (spec 14.22.5).
func TestRefsSwitchport(t *testing.T) {
	sc := defs(t)
	devs := []book.DeviceMap{{OldHost: "rt-a01", NewHost: "ix-b01"}}
	ifs := []book.IfMap{
		{NewHost: "ix-b01", OldIf: "FastEthernet0/1"},
		{NewHost: "ix-b01", OldIf: "FastEthernet0/2"},
		{NewHost: "ix-b01", OldIf: "FastEthernet0/3"},
		{NewHost: "ix-b01", OldIf: "FastEthernet0/4"},
	}
	Refs(sc, switchData(), devs, ifs)
	for i, want := range []string{"説明: sample pc / アクセス VLAN 10 / 音声 VLAN 30", "アクセス VLAN 1（既定）", "トランク（許可 VLAN 10,20・ネイティブ 1）", "（設定なし）"} {
		if ifs[i].Now != want {
			t.Errorf("%s: %q, want %q", ifs[i].OldIf, ifs[i].Now, want)
		}
	}
}

// The ports of 足すポート become access ports of the VLAN of the row (spec
// 14.22.6); a row that cannot tell its VLAN is told, not dropped.
func TestAddPorts(t *testing.T) {
	sc := defs(t)
	dev := []book.DeviceMap{{OldHost: "rt-a01", NewHost: "ix-b01", NewVendor: "nec_ix"}}
	ifs := []book.IfMap{
		{NewHost: "ix-b01", OldIf: "Vlan10", NewIf: "GigaEthernet1:1.0", AddPorts: "GigaEthernet1:port5, GigaEthernet1:port6,GigaEthernet1:port1"},
		{NewHost: "ix-b01", OldIf: "FastEthernet0/1", NewIf: "GigaEthernet1:port1"},
		{NewHost: "ix-b01", OldIf: "FastEthernet0/2", NewIf: "-"},
		{NewHost: "ix-b01", OldIf: "FastEthernet0/3", NewIf: "GigaEthernet1:port7"},
		{NewHost: "ix-b01", OldIf: "FastEthernet0/4", NewIf: "GigaEthernet0.0", AddPorts: "GigaEthernet1:port8"},
	}
	a := Apply(sc, switchData(), dev, ifs)
	for _, p := range []string{"GigaEthernet1:port5", "GigaEthernet1:port6"} {
		sp := a.Data.Tables["switchports"].Find([]model.KV{{Name: "device", Value: "ix-b01"}, {Name: "if_name", Value: p}})
		if sp == nil || sp.Get("mode") != "access" || sp.Get("access_vlan") != "10" {
			t.Errorf("%s: switch port %+v", p, sp)
		}
		if ir := a.Data.Tables["interfaces"].Find([]model.KV{{Name: "device", Value: "ix-b01"}, {Name: "name", Value: p}}); ir == nil || ir.Get("enabled") != "true" {
			t.Errorf("%s: interface %+v", p, ir)
		}
	}
	if len(a.Ports) != 2 {
		t.Errorf("added: %+v", a.Ports)
	}
	// port1 has its row from the old FastEthernet0/1 already; port8 has a
	// row whose interface has no VLAN.
	var left []string
	for _, n := range a.PortsLeft {
		left = append(left, n.Value+": "+n.Key)
	}
	got := strings.Join(left, "\n")
	if len(left) != 2 || !strings.Contains(got, "GigaEthernet1:port1: スイッチポートの表に") || !strings.Contains(got, "GigaEthernet1:port8: 新 IF GigaEthernet0.0 の VLAN が分からない") {
		t.Errorf("left:\n%s", got)
	}
	// The number of a VLAN group's interface is the VLAN when the old
	// interface has none.
	b := Apply(sc, switchData(), dev, []book.IfMap{{NewHost: "ix-b01", OldIf: "FastEthernet0/4", NewIf: "GigaEthernet1:3.0", AddPorts: "GigaEthernet1:port2"}})
	if sp := b.Data.Tables["switchports"].Find([]model.KV{{Name: "device", Value: "ix-b01"}, {Name: "if_name", Value: "GigaEthernet1:port2"}}); sp == nil || sp.Get("access_vlan") != "3" {
		t.Errorf("group number: %+v %+v", sp, b.PortsLeft)
	}
}
