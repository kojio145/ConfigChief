package check_test

import (
	"testing"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
)

// The checks of the values of the tables of VLANs, switch ports and LAGs.
func TestValuesVLAN(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	put(t, sc, ds, "devices", "hostname", "sw-a01", "vendor", "cisco_ios")
	put(t, sc, ds, "vlans", "device", "sw-a01", "vlan_id", "10", "name", "sample-office")
	put(t, sc, ds, "vlans", "device", "sw-a01", "vlan_id", "20")
	port := func(name string, cells ...string) {
		put(t, sc, ds, "interfaces", "device", "sw-a01", "name", name, "enabled", "true", "description", "sample port")
		if len(cells) > 0 {
			put(t, sc, ds, "switchports", append([]string{"device", "sw-a01", "if_name", name}, cells...)...)
		}
	}
	// V-251: VLAN 30 has no row; VLAN 1 needs none. V-252: the native VLAN
	// is not among the allowed ones.
	port("GigabitEthernet1/0/1", "mode", "access", "access_vlan", "30")
	port("GigabitEthernet1/0/2", "mode", "access", "access_vlan", "1")
	port("GigabitEthernet1/0/3", "mode", "trunk", "native_vlan", "20", "allowed_vlans", "10|30-31")
	port("GigabitEthernet1/0/4", "mode", "trunk", "native_vlan", "20")
	// V-257: a list that cannot be read. V-258: a routed port with a VLAN.
	port("GigabitEthernet1/0/5", "mode", "trunk", "allowed_vlans", "10|all")
	port("GigabitEthernet1/0/6", "mode", "trunk", "allowed_vlans", "10-5000")
	port("GigabitEthernet1/0/7", "mode", "routed", "access_vlan", "10")
	// V-253: LACP and static in one LAG. V-254: the ports differ.
	port("GigabitEthernet1/0/11", "mode", "trunk", "allowed_vlans", "10|20")
	port("GigabitEthernet1/0/12", "mode", "trunk", "allowed_vlans", "10")
	port("GigabitEthernet1/0/13", "mode", "trunk", "allowed_vlans", "10|20")
	put(t, sc, ds, "lags", "device", "sw-a01", "if_name", "GigabitEthernet1/0/11", "group", "1", "mode", "active")
	put(t, sc, ds, "lags", "device", "sw-a01", "if_name", "GigabitEthernet1/0/12", "group", "1", "mode", "on")
	put(t, sc, ds, "lags", "device", "sw-a01", "if_name", "GigabitEthernet1/0/13", "group", "1", "mode", "passive")
	// V-255: the same VLAN twice on a port; two native VLANs.
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/1.10", "enabled", "true", "description", "sample a", "encap_vlan", "10", "encap_native", "true")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/1.11", "enabled", "true", "description", "sample b", "encap_vlan", "10")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/1.20", "enabled", "true", "description", "sample c", "encap_vlan", "20", "encap_native", "true")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/2.10", "enabled", "true", "description", "sample d", "encap_vlan", "10")
	// V-256: a bridge group of one interface; one with its BVI only; one that joins two.
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/3", "enabled", "true", "description", "sample e", "bridge_group", "1")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "BVI2", "enabled", "true", "description", "sample f", "bridge_group", "2")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/4", "enabled", "true", "description", "sample g", "bridge_group", "3")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/5", "enabled", "true", "description", "sample h", "bridge_group", "3")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "BVI3", "enabled", "true", "description", "sample i", "bridge_group", "3")

	fs, tally := run(sc, ds, nil)
	same(t, told(fs, "V-25"), []string{
		"V-251 sw-a01 access_vlan",   // 1/0/1: VLAN 30
		"V-251 sw-a01 allowed_vlans", // 1/0/3: 30-31
		"V-252 sw-a01 native_vlan",   // 1/0/3
		"V-253 sw-a01 mode",          // 1/0/12: on, the first is LACP
		"V-254 sw-a01 allowed_vlans", // 1/0/12
		"V-255 rt-a02 encap_vlan",    // 0/1.11
		"V-255 rt-a02 encap_native",  // 0/1.20
		"V-256 rt-a02 bridge_group",  // group 1
		"V-256 rt-a02 bridge_group",  // group 2
		"V-257 sw-a01 allowed_vlans", // 1/0/5
		"V-257 sw-a01 allowed_vlans", // 1/0/6
		"V-258 sw-a01 access_vlan",   // 1/0/7
	})
	for _, id := range []string{"V-251", "V-252", "V-253", "V-254", "V-255", "V-256", "V-257", "V-258"} {
		if !tally.Ran(id) {
			t.Errorf("%s did not run", id)
		}
	}
}

// l2Base is two switches and a router; the ports name their other ends.
func l2Base(t *testing.T, sc *schema.Set) (*model.Dataset, func(dev, name, peerDev, peerIf string, cells ...string)) {
	ds := model.NewDataset("book")
	put(t, sc, ds, "devices", "hostname", "sw-a01", "vendor", "cisco_ios")
	put(t, sc, ds, "devices", "hostname", "sw-a02", "vendor", "cisco_ios")
	put(t, sc, ds, "devices", "hostname", "rt-a03", "vendor", "cisco_ios")
	return ds, func(dev, name, peerDev, peerIf string, cells ...string) {
		put(t, sc, ds, "interfaces", "device", dev, "name", name, "kind", "physical", "peer_device", peerDev, "peer_if", peerIf)
		if len(cells) > 0 {
			put(t, sc, ds, "switchports", append([]string{"device", dev, "if_name", name}, cells...)...)
		}
	}
}

// Two ends that agree: nothing to tell.
func TestCrossL2Quiet(t *testing.T) {
	sc := defs(t)
	ds, port := l2Base(t, sc)
	port("sw-a01", "GigabitEthernet1/0/1", "sw-a02", "GigabitEthernet1/0/1", "mode", "trunk", "allowed_vlans", "10|20-21")
	port("sw-a02", "GigabitEthernet1/0/1", "sw-a01", "GigabitEthernet1/0/1", "mode", "trunk", "allowed_vlans", "10,20|21", "native_vlan", "1")
	port("sw-a01", "GigabitEthernet1/0/2", "sw-a02", "GigabitEthernet1/0/2", "mode", "access", "access_vlan", "10")
	port("sw-a02", "GigabitEthernet1/0/2", "", "", "mode", "access", "access_vlan", "10")
	put(t, sc, ds, "lags", "device", "sw-a01", "if_name", "GigabitEthernet1/0/1", "group", "1", "mode", "active")
	put(t, sc, ds, "lags", "device", "sw-a02", "if_name", "GigabitEthernet1/0/1", "group", "3", "mode", "passive")
	// A router with subinterfaces on a trunk; a peer outside the case.
	port("sw-a01", "GigabitEthernet1/0/3", "rt-a03", "GigabitEthernet0/1", "mode", "trunk", "allowed_vlans", "10|20", "native_vlan", "99")
	port("rt-a03", "GigabitEthernet0/1", "sw-a01", "GigabitEthernet1/0/3")
	put(t, sc, ds, "interfaces", "device", "rt-a03", "name", "GigabitEthernet0/1.10", "kind", "subif", "encap_vlan", "10")
	put(t, sc, ds, "interfaces", "device", "rt-a03", "name", "GigabitEthernet0/1.99", "kind", "subif", "encap_vlan", "99", "encap_native", "true")
	port("sw-a01", "GigabitEthernet1/0/4", "sw-sample", "GigabitEthernet1/0/1", "mode", "trunk")
	put(t, sc, ds, "vlans", "device", "sw-a01", "vlan_id", "10", "name", "sample-office")
	put(t, sc, ds, "vlans", "device", "sw-a02", "vlan_id", "10", "name", "sample-office")
	put(t, sc, ds, "vlans", "device", "sw-a02", "vlan_id", "20")
	if fs := cross(sc, ds); len(fs) != 0 {
		t.Errorf("findings for ends that agree: %v", told(fs, "X-"))
	}
}

func TestCrossL2(t *testing.T) {
	sc := defs(t)
	ds, port := l2Base(t, sc)
	// X-381: no such interface; the other end names another port.
	port("sw-a01", "GigabitEthernet1/0/1", "sw-a02", "GigabitEthernet1/0/9")
	port("sw-a01", "GigabitEthernet1/0/2", "sw-a02", "GigabitEthernet1/0/2")
	port("sw-a02", "GigabitEthernet1/0/2", "sw-a01", "GigabitEthernet1/0/3")
	// X-382 and X-383: the trunks differ. X-387: the speed. X-385: LACP meets static.
	port("sw-a01", "GigabitEthernet1/0/3", "sw-a02", "GigabitEthernet1/0/3", "mode", "trunk", "allowed_vlans", "10|20", "native_vlan", "99")
	port("sw-a02", "GigabitEthernet1/0/3", "", "", "mode", "trunk", "allowed_vlans", "10|30")
	put(t, sc, ds, "interfaces", "device", "sw-a01", "name", "GigabitEthernet1/0/4", "kind", "physical", "peer_device", "sw-a02", "peer_if", "GigabitEthernet1/0/4", "speed", "1000", "duplex", "full")
	put(t, sc, ds, "interfaces", "device", "sw-a02", "name", "GigabitEthernet1/0/4", "kind", "physical", "peer_device", "sw-a01", "peer_if", "GigabitEthernet1/0/4", "speed", "100", "duplex", "full")
	put(t, sc, ds, "lags", "device", "sw-a01", "if_name", "GigabitEthernet1/0/4", "group", "1", "mode", "active")
	put(t, sc, ds, "lags", "device", "sw-a02", "if_name", "GigabitEthernet1/0/4", "group", "1", "mode", "on")
	// X-385: both wait; one side only.
	port("sw-a01", "GigabitEthernet1/0/5", "sw-a02", "GigabitEthernet1/0/5")
	port("sw-a02", "GigabitEthernet1/0/5", "sw-a01", "GigabitEthernet1/0/5")
	put(t, sc, ds, "lags", "device", "sw-a01", "if_name", "GigabitEthernet1/0/5", "group", "2", "mode", "passive")
	put(t, sc, ds, "lags", "device", "sw-a02", "if_name", "GigabitEthernet1/0/5", "group", "2", "mode", "passive")
	port("sw-a01", "GigabitEthernet1/0/6", "sw-a02", "GigabitEthernet1/0/6")
	port("sw-a02", "GigabitEthernet1/0/6", "sw-a01", "GigabitEthernet1/0/6")
	put(t, sc, ds, "lags", "device", "sw-a02", "if_name", "GigabitEthernet1/0/6", "group", "3", "mode", "on")
	// X-384: access meets trunk; two access VLANs.
	port("sw-a01", "GigabitEthernet1/0/7", "sw-a02", "GigabitEthernet1/0/7", "mode", "access", "access_vlan", "10")
	port("sw-a02", "GigabitEthernet1/0/7", "sw-a01", "GigabitEthernet1/0/7", "mode", "trunk")
	port("sw-a01", "GigabitEthernet1/0/8", "sw-a02", "GigabitEthernet1/0/8", "mode", "access", "access_vlan", "10")
	port("sw-a02", "GigabitEthernet1/0/8", "sw-a01", "GigabitEthernet1/0/8", "mode", "access", "access_vlan", "20")
	// A router on a trunk: VLAN 30 is not allowed, and the native VLANs differ.
	port("sw-a01", "GigabitEthernet1/0/10", "rt-a03", "GigabitEthernet0/1", "mode", "trunk", "allowed_vlans", "10|20")
	port("rt-a03", "GigabitEthernet0/1", "sw-a01", "GigabitEthernet1/0/10")
	put(t, sc, ds, "interfaces", "device", "rt-a03", "name", "GigabitEthernet0/1.10", "kind", "subif", "encap_vlan", "10")
	put(t, sc, ds, "interfaces", "device", "rt-a03", "name", "GigabitEthernet0/1.30", "kind", "subif", "encap_vlan", "30")
	put(t, sc, ds, "interfaces", "device", "rt-a03", "name", "GigabitEthernet0/1.99", "kind", "subif", "encap_vlan", "99", "encap_native", "true")
	// X-386: one VLAN, two names.
	put(t, sc, ds, "vlans", "device", "sw-a01", "vlan_id", "10", "name", "sample-office")
	put(t, sc, ds, "vlans", "device", "sw-a02", "vlan_id", "10", "name", "sample-guest")
	// X-388: UDLD on one end only; two ways of it.
	port("sw-a01", "GigabitEthernet1/0/11", "sw-a02", "GigabitEthernet1/0/11", "udld", "aggressive")
	port("sw-a02", "GigabitEthernet1/0/11", "sw-a01", "GigabitEthernet1/0/11", "udld", "disable")
	port("sw-a01", "GigabitEthernet1/0/12", "sw-a02", "GigabitEthernet1/0/12", "udld", "aggressive")
	port("sw-a02", "GigabitEthernet1/0/12", "sw-a01", "GigabitEthernet1/0/12", "udld", "enable")
	same(t, told(cross(sc, ds), "X-38"), []string{
		"X-381 sw-a01 peer_if",       // 1/0/1
		"X-381 sw-a01 peer_if",       // 1/0/2
		"X-381 sw-a02 peer_if",       // 1/0/2 names 1/0/3, which names another port
		"X-382 sw-a01 allowed_vlans", // 1/0/3
		"X-382 sw-a01 allowed_vlans", // 1/0/10: VLAN 30 of the router
		"X-383 sw-a01 native_vlan",   // 1/0/3
		"X-383 sw-a01 native_vlan",   // 1/0/10: 99 on the router
		"X-384 sw-a01 mode",          // 1/0/7
		"X-384 sw-a01 access_vlan",   // 1/0/8
		"X-385 sw-a01 mode",          // 1/0/4
		"X-385 sw-a01 mode",          // 1/0/5
		"X-385 sw-a02 group",         // 1/0/6
		"X-386 sw-a02 name",
		"X-387 sw-a01 speed",
		"X-388 sw-a01 udld", // 1/0/11
		"X-388 sw-a01 udld", // 1/0/12
	})
}
