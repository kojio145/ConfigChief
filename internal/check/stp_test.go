package check_test

import "testing"

// The checks of the kind of spanning tree: settings per VLAN with one tree
// for the device (V-271), a priority of the device with a tree per VLAN
// (V-272). An empty mode is the default of the device and is not judged.
func TestValuesSTP(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	for _, d := range [][2]string{{"sw-a01", "rstp"}, {"sw-a02", "rapid_pvst"}, {"sw-a03", ""}, {"sw-a04", "mst"}, {"sw-a05", "pvst"}} {
		put(t, sc, ds, "devices", "hostname", d[0], "vendor", "nec_qx")
		put(t, sc, ds, "interfaces", "device", d[0], "name", "GigabitEthernet1/0/1", "enabled", "true", "description", "sample port")
		put(t, sc, ds, "l2_settings", "device", d[0], "stp_mode", d[1], "stp_priority", "4096")
		put(t, sc, ds, "stp_vlans", "device", d[0], "vlans", "10,20", "priority", "8192")
	}
	fs, tally := run(sc, ds, nil)
	same(t, told(fs, "V-27"), []string{
		"V-271 sw-a01 vlans",        // one tree, and a row per VLAN
		"V-272 sw-a02 stp_priority", // a tree per VLAN, and a priority of the device
		"V-271 sw-a04 vlans",
		"V-272 sw-a05 stp_priority",
	})
	for _, id := range []string{"V-271", "V-272"} {
		if !tally.Ran(id) {
			t.Errorf("%s did not run", id)
		}
	}
}

// The steps of the priorities of spanning tree (V-259): 4096 for a bridge,
// 16 for a port.
func TestValuesSTPSteps(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	put(t, sc, ds, "devices", "hostname", "sw-a01", "vendor", "cisco_ios")
	put(t, sc, ds, "interfaces", "device", "sw-a01", "name", "GigabitEthernet1/0/1", "enabled", "true", "description", "sample port")
	put(t, sc, ds, "interfaces", "device", "sw-a01", "name", "GigabitEthernet1/0/2", "enabled", "true", "description", "sample port")
	put(t, sc, ds, "l2_settings", "device", "sw-a01", "stp_mode", "mst", "stp_priority", "1000")
	put(t, sc, ds, "stp_instances", "device", "sw-a01", "instance", "1", "priority", "4096")
	put(t, sc, ds, "stp_instances", "device", "sw-a01", "instance", "2", "priority", "65535")
	put(t, sc, ds, "switchports", "device", "sw-a01", "if_name", "GigabitEthernet1/0/1", "stp_port_priority", "64")
	put(t, sc, ds, "switchports", "device", "sw-a01", "if_name", "GigabitEthernet1/0/2", "stp_port_priority", "20")
	fs, tally := run(sc, ds, nil)
	same(t, told(fs, "V-259"), []string{
		"V-259 sw-a01 stp_priority",
		"V-259 sw-a01 priority",
		"V-259 sw-a01 stp_port_priority",
	})
	if !tally.Ran("V-259") {
		t.Error("V-259 did not run")
	}
}
