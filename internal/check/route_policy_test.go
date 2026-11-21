package check_test

import (
	"testing"
)

// The checks of the values of the route policies (V-221 to V-226).
func TestValuesRoutePolicies(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	put(t, sc, ds, "prefix_lists", "device", "rt-a01", "name", "PL-SAMPLE-A", "seq", "5", "action", "permit", "prefix", "198.51.100.0/24")
	put(t, sc, ds, "acls", "device", "rt-a01", "name", "ACL-SAMPLE-A", "type", "extended")
	put(t, sc, ds, "acl_entries", "device", "rt-a01", "acl", "ACL-SAMPLE-A", "seq", "10", "action", "permit", "protocol", "ip", "src", "any", "dst", "any")
	put(t, sc, ds, "bgp_processes", "device", "rt-a01", "asn", "64496")
	// V-221: a name that is in no row of the policies. RM-SAMPLE-OK is there.
	put(t, sc, ds, "bgp_neighbors", "device", "rt-a01", "neighbor", "192.0.2.2", "remote_as", "64497", "filter_in", "RM-SAMPLE-OK", "filter_out", "RM-SAMPLE-LOST")
	put(t, sc, ds, "bgp_redistribute", "device", "rt-a01", "source", "static", "filter", "RM-SAMPLE-DENY")
	// A good policy of routes; the lists it names count as used (V-210, V-223).
	put(t, sc, ds, "as_path_lists", "device", "rt-a01", "name", "10", "seq", "10", "action", "permit", "regex", "^64497_")
	put(t, sc, ds, "community_lists", "device", "rt-a01", "name", "CL-SAMPLE-A", "seq", "10", "action", "permit", "type", "expanded", "regex", "64497:.*")
	put(t, sc, ds, "route_policies", "device", "rt-a01", "name", "RM-SAMPLE-OK", "seq", "10", "action", "permit",
		"match_prefix_list", "PL-SAMPLE-A", "match_as_path", "10", "match_community_list", "CL-SAMPLE-A", "set_local_preference", "200")
	put(t, sc, ds, "route_policies", "device", "rt-a01", "name", "RM-SAMPLE-OK", "seq", "20", "action", "deny")
	// V-223: lists that no policy names.
	put(t, sc, ds, "as_path_lists", "device", "rt-a01", "name", "20", "seq", "10", "action", "permit", "regex", "^$")
	put(t, sc, ds, "community_lists", "device", "rt-a01", "name", "CL-SAMPLE-B", "seq", "10", "action", "permit", "type", "standard", "communities", "64497:100")
	// V-224: nothing gets through. V-226: a value on a row that drops.
	put(t, sc, ds, "route_policies", "device", "rt-a01", "name", "RM-SAMPLE-DENY", "seq", "10", "action", "deny", "match_tag", "100", "set_metric", "50")
	// V-222: nothing names it. V-226: values that do not go together.
	put(t, sc, ds, "route_policies", "device", "rt-a01", "name", "RM-SAMPLE-ALONE", "seq", "10", "action", "permit",
		"match_community", "64497:100", "match_community_list", "CL-SAMPLE-A", "set_community", "none|64496:1")
	put(t, sc, ds, "route_policies", "device", "rt-a01", "name", "RM-SAMPLE-ALONE", "seq", "20", "action", "permit", "match_community_exact", "true", "set_community_additive", "true")
	// V-225: a policy for packets with a column of routes; one for routes
	// with a column of packets. A policy used for both is left alone.
	put(t, sc, ds, "route_policies", "device", "rt-a01", "name", "RM-SAMPLE-PBR", "seq", "10", "action", "permit", "match_acl", "ACL-SAMPLE-A", "set_next_hop", "192.0.2.2")
	put(t, sc, ds, "route_policies", "device", "rt-a01", "name", "RM-SAMPLE-PBR", "seq", "20", "action", "permit", "match_prefix_list", "PL-SAMPLE-A", "set_local_preference", "100")
	put(t, sc, ds, "route_policies", "device", "rt-a01", "name", "RM-SAMPLE-BOTH", "seq", "10", "action", "permit", "set_interface", "Null0", "set_tag", "5")
	put(t, sc, ds, "route_policies", "device", "rt-a01", "name", "RM-SAMPLE-ROUTE", "seq", "10", "action", "permit", "set_default_next_hop", "192.0.2.2")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/2", "enabled", "true", "description", "sample pbr", "policy_route", "RM-SAMPLE-PBR")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/3", "enabled", "true", "description", "sample pbr", "policy_route", "RM-SAMPLE-BOTH")
	put(t, sc, ds, "ospf_processes", "device", "rt-a01", "process_id", "1", "router_id", "203.0.113.1")
	put(t, sc, ds, "ospf_redistribute", "device", "rt-a01", "process_id", "1", "source", "static", "filter", "RM-SAMPLE-BOTH")
	put(t, sc, ds, "ospf_redistribute", "device", "rt-a01", "process_id", "1", "source", "connected", "filter", "RM-SAMPLE-ROUTE")

	fs, tally := run(sc, ds, nil)
	same(t, told(fs, "V-106", "V-210", "V-22"), []string{
		"V-221 rt-a01 filter_out",
		"V-222 rt-a01 name", // RM-SAMPLE-ALONE
		"V-223 rt-a01 name", // the AS path list 20
		"V-223 rt-a01 name", // CL-SAMPLE-B
		"V-224 rt-a01 name", // RM-SAMPLE-DENY
		"V-225 rt-a01 name", // RM-SAMPLE-PBR 20
		"V-225 rt-a01 name", // RM-SAMPLE-ROUTE 10
		"V-226 rt-a01 action",
		"V-226 rt-a01 match_community_list",
		"V-226 rt-a01 set_community",
		"V-226 rt-a01 match_community_exact",
		"V-226 rt-a01 set_community_additive",
	})
	for _, id := range []string{"V-221", "V-222", "V-223", "V-224", "V-225", "V-226"} {
		if !tally.Ran(id) {
			t.Errorf("%s did not run", id)
		}
	}
}

// Two routers that both hand routes over between OSPF and BGP (X-361), and a
// policy that waits for a community the other side never sends (X-362).
func TestCrossRoutePolicies(t *testing.T) {
	sc := defs(t)
	ds := pairBase(t, sc)
	for _, d := range []struct{ host, peer, asn, ras string }{{"rt-a01", "192.0.2.2", "64496", "64497"}, {"rt-a02", "192.0.2.1", "64497", "64496"}} {
		put(t, sc, ds, "bgp_processes", "device", d.host, "vrf", "", "asn", d.asn)
		put(t, sc, ds, "ospf_processes", "device", d.host, "process_id", "1")
		put(t, sc, ds, "bgp_neighbors", "device", d.host, "vrf", "", "neighbor", d.peer, "remote_as", d.ras, "filter_in", "RM-SAMPLE-IN", "filter_out", "RM-SAMPLE-OUT")
	}
	// rt-a01 hands over both ways with a policy; rt-a02 has none towards OSPF.
	put(t, sc, ds, "ospf_redistribute", "device", "rt-a01", "process_id", "1", "source", "bgp", "filter", "RM-SAMPLE-B2O")
	put(t, sc, ds, "bgp_redistribute", "device", "rt-a01", "vrf", "", "source", "ospf", "filter", "RM-SAMPLE-O2B")
	put(t, sc, ds, "ospf_redistribute", "device", "rt-a02", "process_id", "1", "source", "bgp")
	put(t, sc, ds, "bgp_redistribute", "device", "rt-a02", "vrf", "", "source", "ospf", "filter", "RM-SAMPLE-O2B")
	// rt-a01 waits for 64497:100 and for 64497:200; rt-a02 puts 64497:100
	// and 64497:300 in place of whatever a route had.
	put(t, sc, ds, "route_policies", "device", "rt-a01", "name", "RM-SAMPLE-IN", "seq", "10", "action", "permit", "match_community", "64497:100")
	put(t, sc, ds, "route_policies", "device", "rt-a01", "name", "RM-SAMPLE-IN", "seq", "20", "action", "permit", "match_community", "64497:100|64497:200")
	put(t, sc, ds, "route_policies", "device", "rt-a02", "name", "RM-SAMPLE-OUT", "seq", "10", "action", "permit", "set_community", "64497:100")
	put(t, sc, ds, "route_policies", "device", "rt-a02", "name", "RM-SAMPLE-OUT", "seq", "20", "action", "permit", "set_community", "64497:300")
	put(t, sc, ds, "route_policies", "device", "rt-a02", "name", "RM-SAMPLE-OUT", "seq", "30", "action", "deny")
	// The other way round the sender adds its communities to those of the
	// route: what arrives is not known, nothing to tell.
	put(t, sc, ds, "route_policies", "device", "rt-a02", "name", "RM-SAMPLE-IN", "seq", "10", "action", "permit", "match_community", "64496:900")
	put(t, sc, ds, "route_policies", "device", "rt-a01", "name", "RM-SAMPLE-OUT", "seq", "10", "action", "permit", "set_community", "64496:100", "set_community_additive", "true")
	same(t, told(cross(sc, ds), "X-36"), []string{
		"X-361 rt-a02 filter",
		"X-362 rt-a01 match_community", // the row 20
	})

	// One device alone that hands over both ways is not told.
	ds = pairBase(t, sc)
	put(t, sc, ds, "ospf_redistribute", "device", "rt-a01", "process_id", "1", "source", "bgp")
	put(t, sc, ds, "bgp_redistribute", "device", "rt-a01", "vrf", "", "source", "ospf")
	if fs := told(cross(sc, ds), "X-36"); len(fs) != 0 {
		t.Errorf("one device alone: %v", fs)
	}
}
