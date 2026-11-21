package check_test

import (
	"testing"
)

// The checks of the values that came with the rest of the access lists, NAT,
// line monitoring and DHCP (appendix A 3.25): V-276 to V-282.
func TestValuesF1(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	put(t, sc, ds, "devices", "hostname", "rt-f01", "vendor", "cisco_ios")
	put(t, sc, ds, "acls", "device", "rt-f01", "name", "ACL-SAMPLE-F", "type", "extended")
	entry := func(seq string, cells ...string) {
		put(t, sc, ds, "acl_entries", append([]string{"device", "rt-f01", "acl", "ACL-SAMPLE-F", "seq", seq, "action", "permit", "src", "any", "dst", "any"}, cells...)...)
	}
	// V-276: a TCP flag on UDP, two flags on one line; one flag is fine.
	entry("10", "protocol", "udp", "flags", "established")
	entry("20", "protocol", "tcp", "flags", "ack|syn")
	entry("30", "protocol", "tcp", "flags", "established|syn|log")
	// V-277: an ICMP type on TCP, a code with a name; a code with a number
	// is fine.
	entry("40", "protocol", "tcp", "icmp_type", "echo")
	entry("50", "protocol", "icmp", "icmp_type", "unreachable", "icmp_code", "4")
	entry("60", "protocol", "icmp", "icmp_type", "3", "icmp_code", "4")
	entry("70", "protocol", "ip", "icmp_code", "4")
	// V-278: the list of the row and the list of one protocol.
	put(t, sc, ds, "remote_access", "device", "rt-f01", "protocols", "ssh|telnet", "acl", "ACL-SAMPLE-F", "acl_ssh", "ACL-SAMPLE-F")
	// V-279: the time ranges.
	tr := func(name, seq string, cells ...string) {
		put(t, sc, ds, "time_ranges", append([]string{"device", "rt-f01", "name", name, "seq", seq}, cells...)...)
	}
	tr("TR-SAMPLE-A", "10", "kind", "periodic", "days", "weekdays", "start_time", "8:00", "end_time", "18:00")
	tr("TR-SAMPLE-A", "20", "kind", "periodic", "start_time", "8:00", "end_time", "18:00")                   // no day
	tr("TR-SAMPLE-A", "30", "kind", "periodic", "days", "monday", "start_time", "25:00", "end_time", "9:00") // a time that is none
	tr("TR-SAMPLE-A", "40", "kind", "periodic", "days", "monday", "start_time", "8:00")                      // no end
	tr("TR-SAMPLE-B", "10", "kind", "absolute", "start_time", "8:00", "start_date", "2026-01-01")
	tr("TR-SAMPLE-B", "20", "kind", "absolute", "end_time", "8:00", "end_date", "2026-12-31")     // a second one
	tr("TR-SAMPLE-C", "10", "kind", "absolute", "start_date", "2026/01/01", "start_time", "8:00") // a date that is none
	tr("TR-SAMPLE-D", "10", "kind", "absolute", "start_date", "2026-01-01")                       // a date without its time
	// V-280: NAT.
	nat := func(name, kind string, cells ...string) {
		put(t, sc, ds, "nat_rules", append([]string{"device", "rt-f01", "name", name, "type", kind}, cells...)...)
	}
	nat("nat1", "dynamic_pool", "pool_start", "203.0.113.22", "pool_end", "203.0.113.17", "pool_prefix", "29")
	nat("nat2", "static_network", "local_ip", "198.51.100.64", "global_ip", "203.0.113.64")
	nat("nat3", "static_network", "local_ip", "198.51.100.65", "global_ip", "203.0.113.64", "pool_prefix", "26")
	nat("nat4", "pat_interface", "extendable", "true")
	nat("nat5", "static_network", "local_ip", "198.51.100.64", "global_ip", "203.0.113.64", "pool_prefix", "26", "extendable", "true")
	// V-281: line monitoring.
	track := func(id, kind string, cells ...string) {
		put(t, sc, ds, "tracks", append([]string{"device", "rt-f01", "id", id, "type", kind}, cells...)...)
	}
	track("1", "icmp", "target", "203.0.113.2", "source_ip", "198.51.100.1", "threshold", "1000", "up_when", "within_threshold")
	track("2", "icmp", "target", "203.0.113.3", "source_ip", "198.51.100.1", "source_if", "GigabitEthernet0/0") // both sources
	track("3", "interface", "target", "GigabitEthernet0/0", "threshold", "1000")                                // not of this kind
	track("4", "icmp", "target", "203.0.113.4", "members", "1")                                                 // not a list
	track("10", "list", "list_rule", "all_up", "members", "1|2")
	track("11", "list", "list_rule", "all_up")                  // no members
	track("12", "list", "members", "1|12")                      // itself, and no rule
	track("13", "list", "list_rule", "any_up", "members", "14") // each other
	track("14", "list", "list_rule", "any_up", "members", "13")
	// V-282: DHCP.
	pool := func(name string, cells ...string) {
		put(t, sc, ds, "dhcp_pools", append([]string{"device", "rt-f01", "name", name, "network", "198.51.100.0/24"}, cells...)...)
	}
	pool("POOL-SAMPLE-A", "lease", "90m", "default_router", "198.51.100.1", "default_routers_more", "198.51.100.2")
	pool("POOL-SAMPLE-B", "lease", "infinite")
	pool("POOL-SAMPLE-C", "lease", "1.5h")
	pool("POOL-SAMPLE-D", "lease", "0d")
	pool("POOL-SAMPLE-E", "default_routers_more", "198.51.100.2")

	fs, tally := run(sc, ds, nil)
	same(t, told(fs, "V-27"), []string{
		"V-276 rt-f01 flags",      // 10
		"V-276 rt-f01 flags",      // 20
		"V-277 rt-f01 icmp_type",  // 40
		"V-277 rt-f01 icmp_code",  // 50
		"V-277 rt-f01 icmp_code",  // 70
		"V-278 rt-f01 acl_ssh",    //
		"V-279 rt-f01 days",       // A 20
		"V-279 rt-f01 start_time", // A 30
		"V-279 rt-f01 start_time", // A 40
		"V-279 rt-f01 kind",       // B 20
		"V-279 rt-f01 start_date", // C 10
		"V-279 rt-f01 start_time", // D 10
	})
	same(t, told(fs, "V-28"), []string{
		"V-280 rt-f01 pool_end",             // nat1
		"V-280 rt-f01 pool_prefix",          // nat2
		"V-280 rt-f01 local_ip",             // nat3
		"V-280 rt-f01 extendable",           // nat4
		"V-281 rt-f01 source_ip",            // 2
		"V-281 rt-f01 threshold",            // 3
		"V-281 rt-f01 members",              // 4
		"V-281 rt-f01 members",              // 11
		"V-281 rt-f01 members",              // 12: itself
		"V-281 rt-f01 list_rule",            // 12
		"V-281 rt-f01 members",              // 13
		"V-281 rt-f01 members",              // 14
		"V-282 rt-f01 lease",                // C
		"V-282 rt-f01 lease",                // D
		"V-282 rt-f01 default_routers_more", // E
	})
	for _, id := range []string{"V-276", "V-277", "V-278", "V-279", "V-280", "V-281", "V-282"} {
		if !tally.Ran(id) {
			t.Errorf("%s did not run", id)
		}
	}
}
