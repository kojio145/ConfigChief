package check_test

import (
	"sort"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
)

// told lists the findings of some checks as "ID device column".
func told(fs []check.Finding, prefix ...string) []string {
	var out []string
	for _, f := range fs {
		for _, p := range prefix {
			if strings.HasPrefix(f.RuleID, p) {
				out = append(out, f.RuleID+" "+f.Device+" "+f.Ref.Column)
			}
		}
	}
	sort.Strings(out)
	return out
}

func same(t *testing.T, got, want []string) {
	t.Helper()
	sort.Strings(want)
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("got:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
}

// The checks of the values of the routing and redundancy tables.
func TestValuesM3(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	// V-205: a virtual address outside the subnet of its interface.
	put(t, sc, ds, "fhrp", "device", "rt-a01", "if_name", "GigabitEthernet0/1", "protocol", "vrrp", "group", "1", "virtual_ip", "198.51.100.254", "track", "1")
	put(t, sc, ds, "fhrp", "device", "rt-a01", "if_name", "GigabitEthernet0/1", "protocol", "vrrp", "group", "2", "virtual_ip", "203.0.113.200")
	// V-207: a router ID on two devices; the same one for OSPF and BGP of a device is fine.
	put(t, sc, ds, "ospf_processes", "device", "rt-a01", "process_id", "1", "router_id", "203.0.113.1")
	put(t, sc, ds, "bgp_processes", "device", "rt-a01", "vrf", "", "asn", "64496", "router_id", "203.0.113.1")
	put(t, sc, ds, "ospf_processes", "device", "rt-a02", "process_id", "1", "router_id", "203.0.113.1")
	// V-210: defined and not used. V-211: no entry, no permit.
	put(t, sc, ds, "acls", "device", "rt-a01", "name", "ACL-SAMPLE-USED", "type", "extended")
	put(t, sc, ds, "acls", "device", "rt-a01", "name", "ACL-SAMPLE-EMPTY", "type", "extended")
	put(t, sc, ds, "acls", "device", "rt-a01", "name", "ACL-SAMPLE-DENY", "type", "extended")
	put(t, sc, ds, "acl_entries", "device", "rt-a01", "acl", "ACL-SAMPLE-USED", "seq", "10", "action", "permit", "protocol", "ip", "src", "any", "dst", "any")
	put(t, sc, ds, "acl_entries", "device", "rt-a01", "acl", "ACL-SAMPLE-DENY", "seq", "10", "action", "remark", "remark", "sample")
	put(t, sc, ds, "acl_entries", "device", "rt-a01", "acl", "ACL-SAMPLE-DENY", "seq", "20", "action", "deny", "protocol", "ip", "src", "any", "dst", "any")
	put(t, sc, ds, "snmp_communities", "device", "rt-a01", "community", "dummy-ro-comm", "acl", "ACL-SAMPLE-USED")
	put(t, sc, ds, "snmp_communities", "device", "rt-a01", "community", "dummy-rw-comm", "acl", "ACL-SAMPLE-DENY")
	put(t, sc, ds, "prefix_lists", "device", "rt-a01", "name", "PL-SAMPLE", "seq", "5", "action", "permit", "prefix", "0.0.0.0/0")
	put(t, sc, ds, "prefix_lists", "device", "rt-a01", "name", "PL-SAMPLE", "seq", "10", "action", "deny", "prefix", "192.0.2.0/24")
	// V-214: the watch a group relies on sends from an interface that is off.
	put(t, sc, ds, "tracks", "device", "rt-a01", "id", "1", "type", "icmp", "target", "192.0.2.2", "source_if", "GigabitEthernet0/2")
	put(t, sc, ds, "tracks", "device", "rt-a01", "id", "2", "type", "icmp", "target", "192.0.2.2")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/2", "enabled", "false")

	fs, tally := run(sc, ds, nil)
	same(t, told(fs, "V-205", "V-207", "V-21"), []string{
		"V-205 rt-a01 virtual_ip",
		"V-207 rt-a02 router_id",
		"V-210 rt-a01 name", // ACL-SAMPLE-EMPTY
		"V-210 rt-a01 name", // PL-SAMPLE, once
		"V-210 rt-a01 id",   // the watch 2
		"V-211 rt-a01 name", // ACL-SAMPLE-EMPTY
		"V-211 rt-a01 name", // ACL-SAMPLE-DENY
		"V-214 rt-a01 track",
	})
	for _, id := range []string{"V-205", "V-207", "V-210", "V-211", "V-214"} {
		if !tally.Ran(id) {
			t.Errorf("%s did not run", id)
		}
	}
}

func cross(sc *schema.Set, ds *model.Dataset) []check.Finding {
	return check.Cross(check.Input{Schema: sc, Data: ds}, check.NewTally())
}

// pairBase is two routers on one link and one LAN, with nothing to tell.
func pairBase(t *testing.T, sc *schema.Set) *model.Dataset {
	ds := model.NewDataset("book")
	put(t, sc, ds, "devices", "hostname", "rt-a01", "vendor", "cisco_ios")
	put(t, sc, ds, "devices", "hostname", "rt-a02", "vendor", "cisco_ios")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/0", "kind", "physical", "ipv4", "192.0.2.1/30")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/0", "kind", "physical", "ipv4", "192.0.2.2/30")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/1", "kind", "physical", "ipv4", "198.51.100.2/24")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/1", "kind", "physical", "ipv4", "198.51.100.3/24")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "Loopback0", "kind", "loopback", "ipv4", "203.0.113.1/32")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "Loopback0", "kind", "loopback", "ipv4", "203.0.113.2/32")
	return ds
}

func TestCrossQuiet(t *testing.T) {
	sc := defs(t)
	ds := pairBase(t, sc)
	for _, d := range []struct{ host, peer, lo, prio string }{{"rt-a01", "192.0.2.2", "203.0.113.1", "110"}, {"rt-a02", "192.0.2.1", "203.0.113.2", "100"}} {
		put(t, sc, ds, "bgp_processes", "device", d.host, "vrf", "", "asn", "64496")
		put(t, sc, ds, "bgp_neighbors", "device", d.host, "vrf", "", "neighbor", d.peer, "remote_as", "64496", "password", "dummy-bgp-key")
		put(t, sc, ds, "ospf_processes", "device", d.host, "process_id", "1")
		put(t, sc, ds, "ospf_interfaces", "device", d.host, "process_id", "1", "if_name", "GigabitEthernet0/0", "area", "0", "hello", "5", "dead", "20", "auth_type", "md5", "auth_key_id", "1", "auth_key", "dummy-ospf-key")
		put(t, sc, ds, "ospf_interfaces", "device", d.host, "process_id", "1", "if_name", "GigabitEthernet0/1", "area", "1", "passive", "true")
		put(t, sc, ds, "ospf_interfaces", "device", d.host, "process_id", "1", "if_name", "Loopback0", "area", "0")
		put(t, sc, ds, "ospf_areas", "device", d.host, "process_id", "1", "area", "1", "type", "stub")
		put(t, sc, ds, "fhrp", "device", d.host, "if_name", "GigabitEthernet0/1", "protocol", "vrrp", "group", "1", "virtual_ip", "198.51.100.1", "priority", d.prio)
	}
	if fs := cross(sc, ds); len(fs) != 0 {
		t.Errorf("findings for a pair that agrees: %v", told(fs, "X-"))
	}
}

func TestCrossBGP(t *testing.T) {
	sc := defs(t)
	ds := pairBase(t, sc)
	put(t, sc, ds, "bgp_processes", "device", "rt-a01", "vrf", "", "asn", "64496")
	put(t, sc, ds, "bgp_processes", "device", "rt-a02", "vrf", "", "asn", "64497")
	// The other side is AS 64497, and has no password, other timers and multihop.
	put(t, sc, ds, "bgp_neighbors", "device", "rt-a01", "vrf", "", "neighbor", "192.0.2.2", "remote_as", "64498", "password", "dummy-bgp-key", "keepalive", "10", "holdtime", "30", "update_source", "Loopback0")
	put(t, sc, ds, "bgp_neighbors", "device", "rt-a02", "vrf", "", "neighbor", "192.0.2.1", "remote_as", "64496", "keepalive", "20", "holdtime", "30", "ebgp_multihop", "2")
	// Nobody of the case has this address.
	put(t, sc, ds, "bgp_neighbors", "device", "rt-a01", "vrf", "", "neighbor", "203.0.113.99", "remote_as", "64499")
	// The other side has no row that points back.
	put(t, sc, ds, "bgp_neighbors", "device", "rt-a02", "vrf", "", "neighbor", "198.51.100.2", "remote_as", "64496")
	same(t, told(cross(sc, ds), "X-3"), []string{
		"X-300 rt-a01 neighbor",
		"X-321 rt-a01 remote_as",
		"X-322 rt-a02 neighbor",
		"X-323 rt-a01 password",
		"X-324 rt-a01 update_source",
		"X-324 rt-a01 ebgp_multihop",
		"X-325 rt-a01 keepalive",
	})
}

func TestCrossOSPF(t *testing.T) {
	sc := defs(t)
	ds := pairBase(t, sc)
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/2", "kind", "physical", "ipv4", "198.18.0.1/24", "mtu", "1500")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/2", "kind", "physical", "ipv4", "198.18.0.2/24", "mtu", "1400")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/3", "kind", "physical", "ipv4", "198.18.1.1/24")
	put(t, sc, ds, "ospf_processes", "device", "rt-a01", "process_id", "1", "passive_default", "true")
	put(t, sc, ds, "ospf_processes", "device", "rt-a02", "process_id", "1")
	// Area, timers, kind of network, authentication.
	put(t, sc, ds, "ospf_interfaces", "device", "rt-a01", "process_id", "1", "if_name", "GigabitEthernet0/0", "area", "0", "passive", "false", "hello", "5", "dead", "20", "network_type", "point_to_point", "auth_type", "md5", "auth_key_id", "1", "auth_key", "dummy-ospf-key")
	put(t, sc, ds, "ospf_interfaces", "device", "rt-a02", "process_id", "1", "if_name", "GigabitEthernet0/0", "area", "0.0.0.1", "hello", "10", "dead", "20")
	// The same area written two ways, of two kinds; the key differs; passive on one side.
	put(t, sc, ds, "ospf_interfaces", "device", "rt-a01", "process_id", "1", "if_name", "GigabitEthernet0/2", "area", "2", "auth_type", "simple", "auth_key", "dummy-ospf-pw-1")
	put(t, sc, ds, "ospf_interfaces", "device", "rt-a02", "process_id", "1", "if_name", "GigabitEthernet0/2", "area", "0.0.0.2", "auth_type", "simple", "auth_key", "dummy-ospf-pw-2")
	put(t, sc, ds, "ospf_areas", "device", "rt-a01", "process_id", "1", "area", "2", "type", "stub")
	// OSPF on one side only; nobody else in the subnet (passive: no finding, active: X-300).
	put(t, sc, ds, "ospf_interfaces", "device", "rt-a01", "process_id", "1", "if_name", "GigabitEthernet0/1", "area", "0")
	put(t, sc, ds, "ospf_interfaces", "device", "rt-a01", "process_id", "1", "if_name", "GigabitEthernet0/3", "area", "0", "passive", "false")
	put(t, sc, ds, "ospf_interfaces", "device", "rt-a01", "process_id", "1", "if_name", "Loopback0", "area", "0")
	same(t, told(cross(sc, ds), "X-3"), []string{
		"X-300 rt-a01 if_name", // GigabitEthernet0/3
		"X-331 rt-a01 area",
		"X-332 rt-a01 hello",
		"X-333 rt-a01 network_type",
		"X-334 rt-a01 auth_type",
		"X-334 rt-a01 auth_key",
		"X-335 rt-a01 area",
		"X-336 rt-a01 if_name",
		"X-337 rt-a01 passive", // GigabitEthernet0/2: passive by the default of the process
		"X-338 rt-a01 if_name", // GigabitEthernet0/1
	})
}

func TestCrossRedundancy(t *testing.T) {
	sc := defs(t)
	ds := pairBase(t, sc)
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/2", "kind", "physical", "ipv4", "198.18.0.1/24", "mtu", "1500")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/2", "kind", "physical", "ipv4", "198.18.0.2/25", "mtu", "1500")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/3", "kind", "physical", "ipv4", "198.18.2.1/24", "ip_mtu", "1400")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/3", "kind", "physical", "ipv4", "198.18.2.2/24", "mtu", "1500")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/4", "kind", "physical", "ipv4", "198.18.3.1/24")
	// Everything differs but the priority.
	put(t, sc, ds, "fhrp", "device", "rt-a01", "if_name", "GigabitEthernet0/1", "protocol", "vrrp", "group", "1", "virtual_ip", "198.51.100.1", "priority", "100", "advert_interval", "1", "auth_type", "simple", "auth_key", "dummy-vrrp-1")
	put(t, sc, ds, "fhrp", "device", "rt-a02", "if_name", "GigabitEthernet0/1", "protocol", "hsrp", "group", "1", "virtual_ip", "198.51.100.254", "priority", "100", "advert_interval", "3")
	// A group the other device of the subnet does not have; a group with nobody else in the subnet.
	put(t, sc, ds, "fhrp", "device", "rt-a01", "if_name", "GigabitEthernet0/1", "protocol", "vrrp", "group", "2", "virtual_ip", "198.51.100.5")
	put(t, sc, ds, "fhrp", "device", "rt-a01", "if_name", "GigabitEthernet0/4", "protocol", "vrrp", "group", "3", "virtual_ip", "198.18.3.254")
	same(t, told(cross(sc, ds), "X-3"), []string{
		"X-300 rt-a01 group",
		"X-341 rt-a01 virtual_ip",
		"X-342 rt-a01 protocol",
		"X-343 rt-a01 priority",
		"X-344 rt-a01 auth_type",
		"X-345 rt-a01 advert_interval",
		"X-346 rt-a01 group",
		"X-351 rt-a01 ipv4",   // /24 and /25
		"X-351 rt-a01 ip_mtu", // 1400 and 1500
	})
}
