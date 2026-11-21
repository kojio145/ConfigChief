package check_test

import (
	"testing"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
)

// The checks of the values of the L2TP tables (appendix A 3.21).
func TestValuesL2TP(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "Virtual-Template1", "enabled", "true", "description", "sample callers", "ipv4", "unnumbered:Loopback0")
	// Nothing to tell: an LNS with its pool and a caller, a LAC.
	put(t, sc, ds, "l2tp_groups", "device", "rt-a01", "name", "lns-sample-ok", "role", "lns", "logical_ifs", "Virtual-Template1", "tunnel_auth", "false",
		"auth", "chap", "pool_start", "198.51.100.100", "pool_end", "198.51.100.150")
	put(t, sc, ds, "l2tp_users", "device", "rt-a01", "group", "lns-sample-ok", "name", "example-user", "password", "dummy-l2tp-user-secret-1")
	put(t, sc, ds, "l2tp_groups", "device", "rt-a01", "name", "lac-sample-ok", "role", "lac", "peer_address", "192.0.2.2", "tunnel_auth", "true", "password", "dummy-l2tp-secret-a", "domains", "example.com")
	// V-241: a LAC without its LNS, an LNS without an interface.
	put(t, sc, ds, "l2tp_groups", "device", "rt-a01", "name", "lac-sample-none", "role", "lac", "tunnel_auth", "true")
	// V-242: a password without the authentication. V-243: one end of the
	// pool; the ends the wrong way round; a caller of a LAC.
	put(t, sc, ds, "l2tp_groups", "device", "rt-a01", "name", "lns-sample-none", "role", "lns", "tunnel_auth", "false", "password", "dummy-l2tp-secret-b", "pool_start", "198.51.100.200")
	put(t, sc, ds, "l2tp_groups", "device", "rt-a01", "name", "lns-sample-pool", "role", "lns", "logical_ifs", "Virtual-Template1", "tunnel_auth", "false", "pool_start", "198.51.100.90", "pool_end", "198.51.100.80")
	put(t, sc, ds, "l2tp_users", "device", "rt-a01", "group", "lac-sample-ok", "name", "sample-user", "password", "dummy-l2tp-user-secret-2")

	// V-235: the row of IPsec an LNS names is on a tunnel interface, has a
	// peer with an address and is in tunnel mode; nothing to tell where it
	// takes any peer in transport mode. (A name that is no row: V-106.)
	put(t, sc, ds, "ipsec_tunnels", "device", "rt-a01", "name", "CMAP-SAMPLE/10", "style", "policy", "local_if", "GigabitEthernet0/0", "peer_address", "any", "ike_version", "1", "psk", "dummy-psk-sample-1", "ipsec_mode", "transport")
	put(t, sc, ds, "ipsec_tunnels", "device", "rt-a01", "name", "CMAP-SAMPLE/20", "style", "policy", "local_if", "GigabitEthernet0/0", "peer_address", "203.0.113.2", "ike_version", "1", "psk", "dummy-psk-sample-2")
	put(t, sc, ds, "l2tp_groups", "device", "rt-a01", "name", "lns-sample-sec", "role", "lns", "logical_ifs", "Virtual-Template1", "tunnel_auth", "false", "ipsec_policy", "CMAP-SAMPLE/10")
	put(t, sc, ds, "l2tp_groups", "device", "rt-a01", "name", "lns-sample-sec-bad", "role", "lns", "logical_ifs", "Virtual-Template1", "tunnel_auth", "false", "ipsec_policy", "CMAP-SAMPLE/20")

	// Nothing to tell: negotiated, and set by hand.
	l2 := func(cells ...string) {
		put(t, sc, ds, "l2tpv3", append([]string{"device", "rt-a01", "peer_address", "203.0.113.2", "local_if", "Loopback0"}, cells...)...)
	}
	l2("if_name", "GigabitEthernet0/0", "vc_id", "101", "profile", "PW-SAMPLE-A", "signaling", "dynamic", "password", "dummy-l2tpv3-secret-a")
	l2("if_name", "GigabitEthernet0/1", "vc_id", "102", "profile", "PW-SAMPLE-M", "signaling", "manual", "local_session_id", "222", "remote_session_id", "111", "local_cookie", "4 54321", "remote_cookie", "8 12345 67890")
	// V-244: by hand without an ID, with a password; negotiated with an ID;
	// a cookie in another form.
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "Loopback0", "enabled", "true", "description", "sample id", "ipv4", "203.0.113.2/32")
	l3 := func(cells ...string) {
		put(t, sc, ds, "l2tpv3", append([]string{"device", "rt-a02", "peer_address", "203.0.113.1", "local_if", "Loopback0"}, cells...)...)
	}
	l3("if_name", "GigabitEthernet0/0", "vc_id", "201", "profile", "PW-SAMPLE-B", "signaling", "manual", "local_session_id", "5", "password", "dummy-l2tpv3-secret-b", "local_cookie", "16 1")
	// V-245: the same profile with another password; the same circuit twice.
	l3("if_name", "GigabitEthernet0/1", "vc_id", "201", "profile", "PW-SAMPLE-B", "signaling", "dynamic", "remote_session_id", "7")

	fs, tally := run(sc, ds, nil)
	same(t, told(fs, "V-235"), []string{
		"V-235 rt-a01 ipsec_policy", // the peer
		"V-235 rt-a01 ipsec_policy", // the mode
	})
	same(t, told(fs, "V-24"), []string{
		"V-241 rt-a01 peer_address", // lac-sample-none
		"V-241 rt-a01 logical_ifs",  // lns-sample-none
		"V-242 rt-a01 password",     // lns-sample-none
		"V-243 rt-a01 pool_end",     // lns-sample-none
		"V-243 rt-a01 pool_start",   // lns-sample-pool
		"V-243 rt-a01 group",        // sample-user
		"V-244 rt-a02 remote_session_id",
		"V-244 rt-a02 password",
		"V-244 rt-a02 local_cookie",
		"V-244 rt-a02 remote_session_id", // negotiated
		"V-245 rt-a02 vc_id",
		"V-245 rt-a02 signaling",
		"V-245 rt-a02 password",
	})
	for _, id := range []string{"V-235", "V-241", "V-242", "V-243", "V-244", "V-245"} {
		if !tally.Ran(id) {
			t.Errorf("%s did not run", id)
		}
	}
}

// l2tpBase is three routers that reach each other from their loopbacks.
func l2tpBase(t *testing.T, sc *schema.Set) *model.Dataset {
	ds := model.NewDataset("book")
	put(t, sc, ds, "devices", "hostname", "rt-a01", "vendor", "cisco_ios")
	put(t, sc, ds, "devices", "hostname", "rt-a02", "vendor", "cisco_ios")
	put(t, sc, ds, "devices", "hostname", "ix-a03", "vendor", "nec_ix")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "Loopback0", "kind", "loopback", "ipv4", "203.0.113.1/32")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/0", "kind", "physical", "ipv4", "192.0.2.1/30")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "Loopback0", "kind", "loopback", "ipv4", "203.0.113.2/32")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/0", "kind", "physical", "ipv4", "192.0.2.2/30")
	put(t, sc, ds, "interfaces", "device", "ix-a03", "name", "GigaEthernet0.0", "kind", "physical", "ipv4", "192.0.2.5/30")
	return ds
}

// Two ends that agree, negotiated and set by hand; a LAC and its LNS.
func TestCrossL2TPQuiet(t *testing.T) {
	sc := defs(t)
	ds := l2tpBase(t, sc)
	put(t, sc, ds, "l2tpv3", "device", "rt-a01", "if_name", "GigabitEthernet0/1", "peer_address", "203.0.113.2", "vc_id", "101", "local_if", "Loopback0", "signaling", "dynamic", "password", "dummy-l2tpv3-secret-a")
	put(t, sc, ds, "l2tpv3", "device", "rt-a02", "if_name", "GigabitEthernet0/1", "peer_address", "203.0.113.1", "vc_id", "101", "local_if", "Loopback0", "signaling", "dynamic", "password", "dummy-l2tpv3-secret-a")
	put(t, sc, ds, "l2tpv3", "device", "rt-a01", "if_name", "GigabitEthernet0/2", "peer_address", "203.0.113.2", "vc_id", "102", "local_if", "Loopback0", "signaling", "manual",
		"local_session_id", "222", "remote_session_id", "111", "local_cookie", "4 54321", "remote_cookie", "8 12345 67890")
	put(t, sc, ds, "l2tpv3", "device", "rt-a02", "if_name", "GigabitEthernet0/2", "peer_address", "203.0.113.1", "vc_id", "102", "local_if", "Loopback0", "signaling", "manual",
		"local_session_id", "111", "remote_session_id", "222", "local_cookie", "8 12345 67890", "remote_cookie", "4 54321")
	put(t, sc, ds, "l2tp_groups", "device", "ix-a03", "name", "lac-sample", "role", "lac", "peer_address", "192.0.2.1", "tunnel_auth", "true", "password", "dummy-l2tp-secret-a", "domains", "example.com")
	put(t, sc, ds, "l2tp_groups", "device", "rt-a01", "name", "lns-sample", "role", "lns", "tunnel_auth", "true", "password", "dummy-l2tp-secret-a")
	if fs := cross(sc, ds); len(fs) != 0 {
		t.Errorf("findings for pairs that agree: %v", told(fs, "X-"))
	}
}

func TestCrossL2TP(t *testing.T) {
	sc := defs(t)
	ds := l2tpBase(t, sc)
	l2 := func(dev, ifn, peer, vc string, cells ...string) {
		put(t, sc, ds, "l2tpv3", append([]string{"device", dev, "if_name", ifn, "peer_address", peer, "vc_id", vc}, cells...)...)
	}
	// The password differs; the one side sends from another interface than
	// the other points at.
	l2("rt-a01", "GigabitEthernet0/1", "203.0.113.2", "101", "local_if", "GigabitEthernet0/0", "signaling", "dynamic", "password", "dummy-l2tpv3-secret-a")
	l2("rt-a02", "GigabitEthernet0/1", "203.0.113.1", "101", "local_if", "Loopback0", "signaling", "dynamic", "password", "dummy-l2tpv3-secret-b")
	// Set by hand: the IDs do not mirror, a cookie on one side only.
	l2("rt-a01", "GigabitEthernet0/2", "203.0.113.2", "102", "local_if", "Loopback0", "signaling", "manual", "local_session_id", "222", "remote_session_id", "111", "local_cookie", "4 54321")
	l2("rt-a02", "GigabitEthernet0/2", "203.0.113.1", "102", "local_if", "Loopback0", "signaling", "manual", "local_session_id", "111", "remote_session_id", "333")
	// The way differs.
	l2("rt-a01", "GigabitEthernet0/3", "203.0.113.2", "103", "local_if", "Loopback0", "signaling", "dynamic")
	l2("rt-a02", "GigabitEthernet0/3", "203.0.113.1", "103", "local_if", "Loopback0", "signaling", "manual", "local_session_id", "1", "remote_session_id", "2")
	// No row with this circuit number on the other side; nobody of the case
	// has the address; the other device has no row that points back.
	l2("rt-a01", "GigabitEthernet0/4", "203.0.113.2", "104", "local_if", "Loopback0", "signaling", "dynamic")
	l2("rt-a01", "GigabitEthernet0/5", "203.0.113.99", "105", "local_if", "Loopback0", "signaling", "dynamic")
	l2("rt-a01", "GigabitEthernet0/6", "192.0.2.5", "106", "local_if", "Loopback0", "signaling", "dynamic")

	// A LAC whose LNS waits for another name, with another password; a LAC
	// whose LNS does not check the tunnel; a LAC that points at a device
	// without an LNS; one that points outside the case.
	put(t, sc, ds, "l2tp_groups", "device", "ix-a03", "name", "lac-sample-a", "role", "lac", "peer_address", "192.0.2.1", "local_name", "ix-a03", "tunnel_auth", "true", "password", "dummy-l2tp-secret-a")
	put(t, sc, ds, "l2tp_groups", "device", "rt-a01", "name", "lns-sample-a", "role", "lns", "peer_name", "ix-a09", "tunnel_auth", "true", "password", "dummy-l2tp-secret-b")
	put(t, sc, ds, "l2tp_groups", "device", "rt-a02", "name", "lac-sample-b", "role", "lac", "peer_address", "192.0.2.5", "tunnel_auth", "true", "password", "dummy-l2tp-secret-a")
	put(t, sc, ds, "l2tp_groups", "device", "ix-a03", "name", "lns-sample-b", "role", "lns", "tunnel_auth", "false")
	put(t, sc, ds, "l2tp_groups", "device", "rt-a01", "name", "lac-sample-c", "role", "lac", "peer_address", "203.0.113.2", "tunnel_auth", "false")
	put(t, sc, ds, "l2tp_groups", "device", "rt-a01", "name", "lac-sample-d", "role", "lac", "peer_address", "203.0.113.99", "tunnel_auth", "false")

	same(t, told(cross(sc, ds), "X-3"), []string{
		"X-300 rt-a01 peer_address",     // GigabitEthernet0/5
		"X-300 rt-a01 peer_address",     // lac-sample-d
		"X-371 rt-a01 peer_address",     // GigabitEthernet0/6
		"X-372 rt-a01 vc_id",            // GigabitEthernet0/4
		"X-372 rt-a01 local_session_id", // 222, and the other side waits for 333
		"X-372 rt-a01 local_cookie",
		"X-372 rt-a01 signaling",
		"X-373 rt-a01 password",     // GigabitEthernet0/1
		"X-373 ix-a03 password",     // lac-sample-a
		"X-373 rt-a02 tunnel_auth",  // lac-sample-b
		"X-374 rt-a01 local_if",     // sends from GigabitEthernet0/0
		"X-375 ix-a03 local_name",   // lac-sample-a
		"X-375 rt-a01 peer_address", // lac-sample-c
	})
}
