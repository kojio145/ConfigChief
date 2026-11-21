package check_test

import (
	"testing"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
)

// pair6 is two routers on one IPv6 link, each with a LAN and a loopback, and
// nothing to tell.
func pair6(t *testing.T, sc *schema.Set) *model.Dataset {
	ds := model.NewDataset("book")
	put(t, sc, ds, "devices", "hostname", "rt-a01", "vendor", "cisco_ios")
	put(t, sc, ds, "devices", "hostname", "rt-a02", "vendor", "cisco_ios")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/0", "kind", "physical", "enabled", "true", "description", "sample link", "ipv6", "2001:db8:0:1::1/64")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/0", "kind", "physical", "enabled", "true", "description", "sample link", "ipv6", "2001:db8:0:1::2/64")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/1", "kind", "physical", "enabled", "true", "description", "sample lan", "ipv6", "2001:db8:0:10::1/64")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/1", "kind", "physical", "enabled", "true", "description", "sample lan", "ipv6", "2001:db8:0:20::1/64")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "Loopback0", "kind", "loopback", "enabled", "true", "description", "sample id", "ipv6", "2001:db8::1/128")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "Loopback0", "kind", "loopback", "enabled", "true", "description", "sample id", "ipv6", "2001:db8::2/128")
	return ds
}

// The checks of the values of the IPv6 tables: V-261 to V-268.
func TestValuesIPv6(t *testing.T) {
	sc := defs(t)
	ds := pair6(t, sc)
	// Nothing to tell: a route by a neighbor on the link, one by a link-local
	// address with its interface, a delegated prefix that is asked for.
	put(t, sc, ds, "ipv6_routes", "device", "rt-a01", "prefix", "::/0", "next_hop", "2001:db8:0:1::2")
	put(t, sc, ds, "ipv6_routes", "device", "rt-a01", "prefix", "2001:db8:100::/48", "next_hop", "fe80::2", "out_if", "GigabitEthernet0/0")
	// Every Ethernet interface says whether it sends router advertisements.
	v6 := map[string]*model.Row{}
	for _, r := range ds.Tables["interfaces"].Rows {
		if r.Get("kind") == "physical" {
			v6[r.Get("device")+" "+r.Get("name")] = put(t, sc, ds, "ipv6_interfaces", "device", r.Get("device"), "if_name", r.Get("name"), "ra", "off")
		}
	}
	v6["rt-a02 GigabitEthernet0/0"].Set("pd_request", "pd-sample", model.Origin{})
	v6["rt-a02 GigabitEthernet0/1"].Set("pd_from", "pd-sample", model.Origin{})
	v6["rt-a02 GigabitEthernet0/1"].Set("pd_suffix", "::1/64", model.Origin{})
	put(t, sc, ds, "tunnels", "device", "rt-a01", "if_name", "Tunnel1", "type", "ipv4_over_ipv6", "source", "GigabitEthernet0/0", "destination_v6", "2001:db8:ffff::1")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "Tunnel1", "kind", "tunnel", "enabled", "true", "description", "sample tunnel")
	fs, tally := run(sc, ds, nil)
	same(t, told(fs, "V-26"), nil)
	for _, id := range []string{"V-261", "V-262", "V-263", "V-264", "V-265", "V-266", "V-267", "V-268", "V-269"} {
		if !tally.Ran(id) {
			t.Errorf("%s did not run", id)
		}
	}

	// V-261: a fixed address inside the prefix another interface takes its
	// EUI-64 address from. V-262: the address of the other router.
	v6["rt-a01 GigabitEthernet0/0"].Append("eui64", "2001:db8:0:10::/64", model.Origin{})
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/2", "kind", "physical", "enabled", "true", "description", "sample twin", "ipv6", "2001:db8::1/128")
	// V-263: a next hop on no link of the device. V-264: a link-local one
	// without its interface.
	put(t, sc, ds, "ipv6_routes", "device", "rt-a01", "prefix", "2001:db8:200::/48", "next_hop", "2001:db8:0:99::1")
	put(t, sc, ds, "ipv6_routes", "device", "rt-a01", "prefix", "2001:db8:300::/48", "next_hop", "fe80::9")
	// V-265: interfaces that have no row.
	put(t, sc, ds, "ipv6_interfaces", "device", "rt-a01", "if_name", "GigabitEthernet0/9", "enabled", "true")
	put(t, sc, ds, "ipv6_ra_prefixes", "device", "rt-a01", "if_name", "GigabitEthernet0/9", "prefix", "2001:db8:0:99::/64")
	// V-266: a prefix nobody asks for.
	put(t, sc, ds, "ipv6_interfaces", "device", "rt-a02", "if_name", "GigabitEthernet0/2", "pd_from", "pd-lost", "pd_suffix", "::1/64", "ra", "off")
	// V-267: a process that has no row.
	put(t, sc, ds, "ospfv3_interfaces", "device", "rt-a01", "process_id", "7", "if_name", "Loopback0", "area", "0")
	put(t, sc, ds, "ospfv3_areas", "device", "rt-a01", "process_id", "7", "area", "1", "type", "stub")
	// V-268: no destination, both, one on a tunnel the carrier provisions,
	// and an IPv6 one on a GRE tunnel.
	put(t, sc, ds, "tunnels", "device", "rt-a02", "if_name", "Tunnel1", "type", "ipv4_over_ipv6", "source", "GigabitEthernet0/0")
	put(t, sc, ds, "tunnels", "device", "rt-a02", "if_name", "Tunnel2", "type", "ipv4_over_ipv6", "source", "GigabitEthernet0/0", "destination_v6", "2001:db8:ffff::1", "destination_fqdn", "aftr.example.jp")
	put(t, sc, ds, "tunnels", "device", "rt-a02", "if_name", "Tunnel3", "type", "map_e", "destination_fqdn", "aftr.example.jp")
	put(t, sc, ds, "tunnels", "device", "rt-a02", "if_name", "Tunnel4", "type", "gre", "source", "GigabitEthernet0/0", "destination", "192.0.2.1", "destination_v6", "2001:db8:ffff::1")
	fs, _ = run(sc, ds, nil)
	same(t, told(fs, "V-26"), []string{
		"V-261 rt-a01 eui64",
		"V-262 rt-a02 ipv6",
		"V-263 rt-a01 next_hop",
		"V-264 rt-a01 next_hop",
		"V-265 rt-a01 if_name", // ipv6_interfaces
		"V-265 rt-a01 if_name", // ipv6_ra_prefixes
		"V-266 rt-a02 pd_from",
		"V-267 rt-a01 process_id", // ospfv3_interfaces
		"V-267 rt-a01 process_id", // ospfv3_areas
		"V-268 rt-a02 destination_v6",
		"V-268 rt-a02 destination_fqdn",
		"V-268 rt-a02 destination_fqdn",
		"V-268 rt-a02 destination_v6",
	})
}

// A device that gets its address from outside is not told about its next hops.
func TestValuesIPv6Dynamic(t *testing.T) {
	sc := defs(t)
	ds := pair6(t, sc)
	put(t, sc, ds, "ipv6_interfaces", "device", "rt-a01", "if_name", "GigabitEthernet0/0", "autoconfig", "true")
	put(t, sc, ds, "ipv6_routes", "device", "rt-a01", "prefix", "2001:db8:200::/48", "next_hop", "2001:db8:0:99::1")
	fs, _ := run(sc, ds, nil)
	same(t, told(fs, "V-263"), nil)
}

// OSPFv3 on a link of two routers: X-391 to X-396.
func TestCrossOSPFv3(t *testing.T) {
	sc := defs(t)
	ds := pair6(t, sc)
	put(t, sc, ds, "ospfv3_processes", "device", "rt-a01", "process_id", "1", "router_id", "192.0.2.1")
	put(t, sc, ds, "ospfv3_processes", "device", "rt-a02", "process_id", "1", "router_id", "192.0.2.2")
	a := put(t, sc, ds, "ospfv3_interfaces", "device", "rt-a01", "process_id", "1", "if_name", "GigabitEthernet0/0", "area", "0", "hello", "10", "dead", "40")
	b := put(t, sc, ds, "ospfv3_interfaces", "device", "rt-a02", "process_id", "1", "if_name", "GigabitEthernet0/0", "area", "0.0.0.0", "hello", "10", "dead", "40")
	same(t, told(cross(sc, ds), "X-39"), nil)

	set := func(r *model.Row, col, val string) { r.Set(col, val, model.Origin{}) }
	set(b, "hello", "5")
	set(a, "network_type", "point_to_point")
	set(b, "network_type", "broadcast")
	set(b, "passive", "true")
	same(t, told(cross(sc, ds), "X-39"), []string{
		"X-392 rt-a01 hello",
		"X-393 rt-a01 network_type",
		"X-395 rt-a01 passive",
	})

	set(b, "hello", "10")
	set(b, "network_type", "point_to_point")
	set(b, "passive", "")
	set(b, "area", "1")
	same(t, told(cross(sc, ds), "X-39"), []string{"X-391 rt-a01 area"})

	set(a, "area", "1")
	put(t, sc, ds, "ospfv3_areas", "device", "rt-a02", "process_id", "1", "area", "1", "type", "stub")
	same(t, told(cross(sc, ds), "X-39"), []string{"X-394 rt-a01 area"})

	// One side only runs OSPFv3 on the link.
	ds = pair6(t, sc)
	put(t, sc, ds, "ospfv3_processes", "device", "rt-a01", "process_id", "1")
	put(t, sc, ds, "ospfv3_interfaces", "device", "rt-a01", "process_id", "1", "if_name", "GigabitEthernet0/0", "area", "0")
	same(t, told(cross(sc, ds), "X-39"), []string{"X-396 rt-a01 if_name"})
}

// BGP over IPv6 between two routers: X-397 to X-399.
func TestCrossBGPIPv6(t *testing.T) {
	sc := defs(t)
	ds := pair6(t, sc)
	put(t, sc, ds, "bgp_processes", "device", "rt-a01", "asn", "64496")
	put(t, sc, ds, "bgp_processes", "device", "rt-a02", "asn", "64497")
	a := put(t, sc, ds, "bgp_ipv6_neighbors", "device", "rt-a01", "neighbor", "2001:db8:0:1::2", "remote_as", "64497", "password", "dummy-bgp-secret")
	b := put(t, sc, ds, "bgp_ipv6_neighbors", "device", "rt-a02", "neighbor", "2001:db8:0:1::1", "remote_as", "64496", "password", "dummy-bgp-secret")
	same(t, told(cross(sc, ds), "X-39"), nil)

	a.Set("remote_as", "64499", model.Origin{})
	b.Set("password", "dummy-bgp-other", model.Origin{})
	same(t, told(cross(sc, ds), "X-39"), []string{
		"X-397 rt-a01 remote_as",
		"X-399 rt-a01 password",
	})

	// The other side has no row that points back.
	ds = pair6(t, sc)
	put(t, sc, ds, "bgp_processes", "device", "rt-a01", "asn", "64496")
	put(t, sc, ds, "bgp_processes", "device", "rt-a02", "asn", "64497")
	put(t, sc, ds, "bgp_ipv6_neighbors", "device", "rt-a01", "neighbor", "2001:db8:0:1::2", "remote_as", "64497")
	same(t, told(cross(sc, ds), "X-39"), []string{"X-398 rt-a01 neighbor"})
}

// An Ethernet interface with an IPv6 subnet that does not say whether it
// sends router advertisements is told (V-269): the vendors differ.
func TestValuesIPv6RouterAdvertisement(t *testing.T) {
	sc := defs(t)
	ds := pair6(t, sc)
	put(t, sc, ds, "ipv6_interfaces", "device", "rt-a02", "if_name", "GigabitEthernet0/1", "ra", "send")
	fs, _ := run(sc, ds, nil)
	same(t, told(fs, "V-26"), []string{"V-269 rt-a01 name", "V-269 rt-a01 name", "V-269 rt-a02 name"})
}
