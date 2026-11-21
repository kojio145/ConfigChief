package check_test

import (
	"testing"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/model"
)

// set puts a value into a cell of a row.
func set(r *model.Row, col, v string) { r.Set(col, v, model.Origin{}) }

// V-227: what a DHCPv6 server hands out (appendix A 3.23).
func TestValuesDHCPv6Delegation(t *testing.T) {
	sc := defs(t)
	ds := pair6(t, sc)
	// Nothing to tell: a fixed pool, a part of the received prefix, an
	// address prefix for the hosts.
	put(t, sc, ds, "ipv6_interfaces", "device", "rt-a01", "if_name", "GigabitEthernet0/0", "ra", "off", "pd_request", "pd-sample")
	put(t, sc, ds, "ipv6_interfaces", "device", "rt-a01", "if_name", "GigabitEthernet0/1", "ra", "send", "dhcp_server", "lan-sample",
		"dhcp_pd_from", "pd-sample", "dhcp_pd_pool", "::/56", "dhcp_pd_length", "60")
	put(t, sc, ds, "ipv6_interfaces", "device", "rt-a02", "if_name", "GigabitEthernet0/0", "ra", "off", "dhcp_server", "wan-sample",
		"dhcp_pd_pool", "2001:db8:1000::/40", "dhcp_pd_length", "56", "dhcp_pd_valid", "1800", "dhcp_pd_preferred", "600", "dhcp_ntp", "2001:db8::123")
	a2 := put(t, sc, ds, "ipv6_interfaces", "device", "rt-a02", "if_name", "GigabitEthernet0/1", "ra", "send", "dhcp_server", "lan-sample",
		"dhcp_address_prefix", "2001:db8:0:20::/64", "dhcp_pd_pool", "2001:db8:2000::/40", "dhcp_pd_length", "40", "dhcp_pd_valid", "infinite", "dhcp_pd_preferred", "infinite")
	fs, tally := run(sc, ds, nil)
	same(t, told(fs, "V-227"), nil)
	if !tally.Ran("V-227") {
		t.Error("V-227 did not run")
	}

	// A length shorter than the pool, and a preferred lifetime longer than
	// the valid one.
	set(a2, "dhcp_pd_length", "32")
	set(a2, "dhcp_pd_valid", "600")
	set(a2, "dhcp_pd_preferred", "1800")
	// A pool without its length; one lifetime; a name nobody asks for; a
	// row without a server.
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/2", "kind", "physical", "enabled", "true", "description", "sample")
	put(t, sc, ds, "ipv6_interfaces", "device", "rt-a01", "if_name", "GigabitEthernet0/2", "ra", "off", "dhcp_server", "lan-b",
		"dhcp_pd_pool", "2001:db8:3000::/40", "dhcp_pd_valid", "600")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/3", "kind", "physical", "enabled", "true", "description", "sample")
	put(t, sc, ds, "ipv6_interfaces", "device", "rt-a01", "if_name", "GigabitEthernet0/3", "ra", "off", "dhcp_server", "lan-c",
		"dhcp_pd_from", "pd-lost", "dhcp_pd_pool", "::/56", "dhcp_pd_length", "60")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/4", "kind", "physical", "enabled", "true", "description", "sample")
	put(t, sc, ds, "ipv6_interfaces", "device", "rt-a01", "if_name", "GigabitEthernet0/4", "ra", "off", "dhcp_ntp", "2001:db8::123")
	fs, _ = run(sc, ds, nil)
	same(t, told(fs, "V-227"), []string{
		"V-227 rt-a02 dhcp_pd_length",
		"V-227 rt-a02 dhcp_pd_preferred",
		"V-227 rt-a01 dhcp_pd_length",
		"V-227 rt-a01 dhcp_pd_preferred",
		"V-227 rt-a01 dhcp_pd_from",
		"V-227 rt-a01 dhcp_ntp",
	})
}

// V-228, X-363 and X-364: RIP (appendix A 3.27).
func TestRIP(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	// Two routers on the link of the sample pair, both with version 2 and
	// the same text key: nothing to tell.
	put(t, sc, ds, "rip_processes", "device", "rt-a01", "family", "ipv4", "version", "2")
	put(t, sc, ds, "rip_processes", "device", "rt-a02", "family", "ipv4")
	a := put(t, sc, ds, "rip_interfaces", "device", "rt-a01", "family", "ipv4", "if_name", "GigabitEthernet0/0", "auth_mode", "text", "auth_key", "example-rip-key")
	b := put(t, sc, ds, "rip_interfaces", "device", "rt-a02", "family", "ipv4", "if_name", "GigabitEthernet0/0", "send_version", "2", "receive_version", "2", "auth_mode", "text", "auth_key", "example-rip-key")
	put(t, sc, ds, "rip_redistribute", "device", "rt-a01", "family", "ipv4", "source", "static")
	both := func() []check.Finding {
		fs, _ := run(sc, ds, nil)
		return append(fs, cross(sc, ds)...)
	}
	fs, tally := run(sc, ds, nil)
	xt := check.NewTally()
	fs = append(fs, check.Cross(check.Input{Schema: sc, Data: ds}, xt)...)
	same(t, told(fs, "V-228", "X-363", "X-364"), nil)
	if !tally.Ran("V-228") || !xt.Ran("X-363") || !xt.Ran("X-364") {
		t.Error("V-228, X-363 or X-364 did not run")
	}

	// X-363: one side sends version 1 only. X-364: the keys differ.
	set(a, "send_version", "1")
	set(b, "auth_key", "example-other-key")
	fs = both()
	same(t, told(fs, "X-363", "X-364"), []string{"X-363 rt-a01 send_version", "X-364 rt-a01 auth_key", "X-364 rt-a02 auth_key"})

	// X-364: text against md5; then md5 with different key numbers.
	set(a, "send_version", "")
	set(b, "auth_key", "example-rip-key")
	set(b, "auth_mode", "md5")
	fs = both()
	same(t, told(fs, "X-363", "X-364"), []string{"X-364 rt-a01 auth_mode", "X-364 rt-a02 auth_mode"})
	set(a, "auth_mode", "md5")
	set(a, "auth_key_id", "5")
	fs = both()
	same(t, told(fs, "X-363", "X-364"), []string{"X-364 rt-a01 auth_key_id", "X-364 rt-a02 auth_key_id"})
	set(b, "auth_key_id", "5")

	// V-228: rows without a process, a key without a mode, a key number on
	// a text key, a version on RIPng, an OSPF process on a static source.
	put(t, sc, ds, "rip_interfaces", "device", "rt-a01", "family", "ipv6", "if_name", "GigabitEthernet0/0", "send_version", "2")
	put(t, sc, ds, "rip_redistribute", "device", "rt-a02", "family", "ipv6", "source", "static", "process_id", "1")
	put(t, sc, ds, "rip_interfaces", "device", "rt-a01", "family", "ipv4", "if_name", "GigabitEthernet0/1", "auth_key", "example-rip-key", "auth_key_id", "2")
	fs = both()
	same(t, told(fs, "V-228", "X-363", "X-364"), []string{
		"V-228 rt-a01 family", "V-228 rt-a01 send_version",
		"V-228 rt-a02 family", "V-228 rt-a02 process_id",
		"V-228 rt-a01 auth_mode", "V-228 rt-a01 auth_key_id",
	})
}

// X-300, X-301 and X-311 for a tunnel over IPv6: the far end is its IPv6
// destination, the source its interface's IPv6 address (appendix A 3.23).
func TestTunnelsOverIPv6(t *testing.T) {
	sc := defs(t)
	ds := pair6(t, sc)
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "Tunnel1", "kind", "tunnel", "enabled", "true", "description", "sample tunnel", "ipv4", "192.0.2.1/30")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "Tunnel1", "kind", "tunnel", "enabled", "true", "description", "sample tunnel", "ipv4", "192.0.2.2/30")
	a := put(t, sc, ds, "tunnels", "device", "rt-a01", "if_name", "Tunnel1", "type", "ipv4_over_ipv6", "source", "GigabitEthernet0/0", "destination_v6", "2001:db8:0:1::2")
	put(t, sc, ds, "tunnels", "device", "rt-a02", "if_name", "Tunnel1", "type", "ipv4_over_ipv6", "source", "GigabitEthernet0/0", "destination_v6", "2001:db8:0:1::1")
	tally := check.NewTally()
	fs := check.Cross(check.Input{Schema: sc, Data: ds}, tally)
	same(t, told(fs, "X-300", "X-301", "X-311"), nil)
	if !tally.Ran("X-311") {
		t.Error("X-311 did not run")
	}
	// The far end is the loopback of the other device: the sources do not
	// mirror the destinations.
	set(a, "destination_v6", "2001:db8::2")
	fs = cross(sc, ds)
	same(t, told(fs, "X-300", "X-301", "X-311"), []string{"X-311 rt-a01 destination_v6"})
	// Nobody has the address.
	set(a, "destination_v6", "2001:db8:ff::9")
	fs = cross(sc, ds)
	same(t, told(fs, "X-300"), []string{"X-300 rt-a01 destination_v6"})
}

// V-229: the family of the target of a watch and of its addresses.
func TestTrackFamily(t *testing.T) {
	sc := defs(t)
	ds := pair6(t, sc)
	put(t, sc, ds, "tracks", "device", "rt-a01", "id", "1", "type", "icmp", "target", "2001:db8:ff::1", "source_ip", "2001:db8:0:1::1", "interval", "5", "fail_count", "3")
	put(t, sc, ds, "tracks", "device", "rt-a01", "id", "2", "type", "route", "target", "2001:db8:100::/48", "interval", "5", "fail_count", "3")
	b := put(t, sc, ds, "tracks", "device", "rt-a01", "id", "3", "type", "icmp", "target", "192.0.2.9", "interval", "5", "fail_count", "3")
	c := put(t, sc, ds, "tracks", "device", "rt-a01", "id", "4", "type", "route", "target", "192.0.2.0/24", "interval", "5", "fail_count", "3")
	fs, tally := run(sc, ds, nil)
	same(t, told(fs, "V-229"), nil)
	if !tally.Ran("V-229") {
		t.Error("V-229 did not run")
	}
	set(b, "via_next_hop", "2001:db8:0:1::2")
	set(c, "target", "192.0.2.1/24")
	put(t, sc, ds, "tracks", "device", "rt-a01", "id", "5", "type", "icmp", "target", "sample-host", "interval", "5", "fail_count", "3")
	fs, _ = run(sc, ds, nil)
	same(t, told(fs, "V-229"), []string{"V-229 rt-a01 via_next_hop", "V-229 rt-a01 target", "V-229 rt-a01 target"})
}

// X-404: the authentication of OSPFv3 on the two ends of a link.
func TestOSPFv3Auth(t *testing.T) {
	sc := defs(t)
	ds := pair6(t, sc)
	for _, d := range []string{"rt-a01", "rt-a02"} {
		put(t, sc, ds, "ospfv3_processes", "device", d, "process_id", "1", "router_id", map[string]string{"rt-a01": "192.0.2.1", "rt-a02": "192.0.2.2"}[d])
		put(t, sc, ds, "ospfv3_areas", "device", d, "process_id", "1", "area", "0", "auth_hash", "md5", "auth_spi", "1000", "auth_key", "0123456789abcdef0123456789abcdef")
	}
	a := put(t, sc, ds, "ospfv3_interfaces", "device", "rt-a01", "process_id", "1", "if_name", "GigabitEthernet0/0", "area", "0")
	put(t, sc, ds, "ospfv3_interfaces", "device", "rt-a02", "process_id", "1", "if_name", "GigabitEthernet0/0", "area", "0",
		"auth_hash", "md5", "auth_spi", "1000", "auth_key", "0123456789abcdef0123456789abcdef")
	tally := check.NewTally()
	fs := check.Cross(check.Input{Schema: sc, Data: ds}, tally)
	same(t, told(fs, "X-404"), nil)
	if !tally.Ran("X-404") {
		t.Error("X-404 did not run")
	}
	set(a, "auth_hash", "none")
	same(t, told(cross(sc, ds), "X-404"), []string{"X-404 rt-a01 auth_hash"})
	set(a, "auth_hash", "sha1")
	set(a, "auth_spi", "1000")
	set(a, "auth_key", "0123456789abcdef0123456789abcdef01234567")
	same(t, told(cross(sc, ds), "X-404"), []string{"X-404 rt-a01 auth_hash"})
}
