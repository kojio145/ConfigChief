package check_test

import (
	"testing"
)

// The checks of the values that came with IKE version 2 and the rest of
// IPsec (appendix A 3.20): V-231 to V-234, and what V-215 and V-216 learned.
func TestValuesIPsecMore(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	base := []string{"psk", "dummy-psk-sample", "esp_encryption", "aes256", "esp_hash", "sha256"}
	row := func(cells ...string) { put(t, sc, ds, "ipsec_tunnels", append(cells, base...)...) }
	put(t, sc, ds, "tunnels", "device", "rt-a01", "if_name", "Tunnel1", "type", "ipsec", "source", "GigabitEthernet0/0", "destination", "203.0.113.20")
	// Nothing to tell: version 2 with identities, the proposal left to the device.
	row("device", "rt-a01", "name", "vpn-sample-ok", "style", "tunnel_if", "tunnel_if", "Tunnel1", "peer_address", "203.0.113.20", "ike_version", "2",
		"local_id", "fqdn:rt-a01.example.com", "remote_id", "keyid:sample-site-b", "psk_remote", "dummy-psk-sample-b", "ike_prf", "sha256", "dpd_interval", "30", "dpd_mode", "periodic")
	// A peer "any" of the style policy needs no selector (V-215); a name is a peer too.
	row("device", "rt-a01", "name", "CMAP-SAMPLE/80", "style", "policy", "local_if", "GigabitEthernet0/0", "peer_address", "any", "ike_version", "2")
	row("device", "rt-a01", "name", "CMAP-SAMPLE/20", "style", "policy", "local_if", "GigabitEthernet0/0", "peer_address", "rt-b01.example.com", "ike_version", "2")
	put(t, sc, ds, "ipsec_selectors", "device", "rt-a01", "tunnel", "CMAP-SAMPLE/20", "seq", "10", "local_net", "198.51.100.0/24", "remote_net", "203.0.113.128/25")
	// V-231: no kind, a kind that is none, an address that is none.
	row("device", "rt-a02", "name", "CMAP-SAMPLE/10", "style", "policy", "peer_address", "any", "ike_version", "2",
		"local_id", "rt-a02.example.com", "remote_id", "name:sample-site-a")
	row("device", "rt-a02", "name", "CMAP-SAMPLE/20", "style", "policy", "peer_address", "any", "ike_version", "2", "local_id", "address:sample-site-a")
	// V-232: what the version of the row does not use.
	row("device", "rt-a02", "name", "CMAP-SAMPLE/30", "style", "policy", "peer_address", "any", "ike_version", "1", "ike_encryption", "aes256", "ike_hash", "sha256", "ike_dh", "dh14",
		"ike_prf", "sha256", "psk_remote", "dummy-psk-sample-c")
	row("device", "rt-a02", "name", "CMAP-SAMPLE/40", "style", "policy", "peer_address", "any", "ike_version", "2", "ike_mode", "aggressive")
	// V-233: a watch of the peer without its interval. V-216: weak values of the new columns.
	row("device", "rt-a02", "name", "CMAP-SAMPLE/50", "style", "policy", "peer_address", "any", "ike_version", "2",
		"dpd_retry", "3", "dpd_retry_interval", "5", "dpd_mode", "periodic", "ike_prf", "md5", "ah_hash", "md5")
	// V-234: neither an address, nor "any", nor a name.
	row("device", "rt-a02", "name", "CMAP-SAMPLE/60", "style", "policy", "peer_address", "203.0.113", "ike_version", "2")
	put(t, sc, ds, "ipsec_selectors", "device", "rt-a02", "tunnel", "CMAP-SAMPLE/60", "seq", "10", "local_net", "203.0.113.128/25", "remote_net", "198.51.100.0/24")

	fs, tally := run(sc, ds, nil)
	same(t, told(fs, "V-215", "V-216", "V-231", "V-232", "V-233", "V-234"), []string{
		"V-216 rt-a02 ike_prf",
		"V-216 rt-a02 ah_hash",
		"V-231 rt-a02 local_id",  // CMAP-SAMPLE/10
		"V-231 rt-a02 remote_id", // CMAP-SAMPLE/10
		"V-231 rt-a02 local_id",  // CMAP-SAMPLE/20
		"V-232 rt-a02 ike_prf",
		"V-232 rt-a02 psk_remote",
		"V-232 rt-a02 ike_mode",
		"V-233 rt-a02 dpd_retry",
		"V-233 rt-a02 dpd_retry_interval",
		"V-233 rt-a02 dpd_mode",
		"V-234 rt-a02 peer_address",
	})
	for _, id := range []string{"V-231", "V-232", "V-233", "V-234"} {
		if !tally.Ran(id) {
			t.Errorf("%s did not run", id)
		}
	}
}

var ike2 = []string{"ike_version", "2", "ike_encryption", "aes256", "ike_hash", "sha256", "ike_dh", "dh14", "ike_prf", "sha256", "ike_lifetime", "86400",
	"esp_encryption", "aes256", "esp_hash", "sha256", "pfs", "dh14", "ipsec_lifetime", "3600"}

// IKE version 2 between two vendors, each side with its own key, and a
// site that the other side waits for with the peer "any": nothing to tell.
func TestCrossIPsec2Quiet(t *testing.T) {
	sc := defs(t)
	ds := vpnBase(t, sc)
	put(t, sc, ds, "devices", "hostname", "ix-a03", "vendor", "nec_ix", "domain_name", "example.com")
	put(t, sc, ds, "interfaces", "device", "ix-a03", "name", "GigaEthernet0.0", "kind", "physical", "ipv4", "dhcp")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "Tunnel1", "kind", "tunnel", "ipv4", "192.0.2.209/30")
	put(t, sc, ds, "interfaces", "device", "ix-a02", "name", "Tunnel0.0", "kind", "tunnel", "ipv4", "192.0.2.210/30")
	put(t, sc, ds, "tunnels", "device", "rt-a01", "if_name", "Tunnel1", "type", "ipsec", "source", "GigabitEthernet0/0", "destination", "203.0.113.21")
	put(t, sc, ds, "tunnels", "device", "ix-a02", "if_name", "Tunnel0.0", "type", "ipsec", "destination", "203.0.113.9")
	// The one side has no identity (its address), the other names itself.
	put(t, sc, ds, "ipsec_tunnels", append([]string{"device", "rt-a01", "name", "Tunnel1", "style", "tunnel_if", "tunnel_if", "Tunnel1", "peer_address", "203.0.113.21",
		"remote_id", "keyid:sample-site-b", "psk", "dummy-psk-sample-a", "psk_remote", "dummy-psk-sample-b", "dpd_interval", "30", "dpd_retry_interval", "5", "dpd_mode", "periodic"}, ike2...)...)
	put(t, sc, ds, "ipsec_tunnels", append([]string{"device", "ix-a02", "name", "Tunnel0.0", "style", "tunnel_if", "tunnel_if", "Tunnel0.0", "peer_address", "203.0.113.9",
		"local_id", "keyid:sample-site-b", "remote_id", "address:203.0.113.9", "psk", "dummy-psk-sample-b", "psk_remote", "dummy-psk-sample-a", "dpd_interval", "30", "dpd_mode", "periodic", "always_on", "true"}, ike2...)...)
	// A site without a fixed address points at the center; the center waits
	// for any peer that names itself so. Another site points at the center
	// by its name.
	put(t, sc, ds, "interfaces", "device", "ix-a03", "name", "Tunnel0.0", "kind", "tunnel", "ipv4", "unnumbered:GigaEthernet0.0")
	put(t, sc, ds, "tunnels", "device", "ix-a03", "if_name", "Tunnel0.0", "type", "ipsec")
	put(t, sc, ds, "ipsec_tunnels", append([]string{"device", "ix-a03", "name", "Tunnel0.0", "style", "tunnel_if", "tunnel_if", "Tunnel0.0", "peer_address", "203.0.113.9",
		"local_id", "fqdn:ix-a03.example.com", "psk", "dummy-psk-sample-c"}, ike2...)...)
	put(t, sc, ds, "ipsec_tunnels", append([]string{"device", "rt-a01", "name", "CMAP-SAMPLE/90", "style", "policy", "local_if", "GigabitEthernet0/0", "peer_address", "any",
		"remote_id", "fqdn:ix-a03.example.com", "psk", "dummy-psk-sample-c"}, ike2...)...)
	if fs := cross(sc, ds); len(fs) != 0 {
		t.Errorf("findings for pairs that agree: %v", told(fs, "X-"))
	}
}

func TestCrossIPsec2(t *testing.T) {
	sc := defs(t)
	ds := vpnBase(t, sc)
	put(t, sc, ds, "devices", "hostname", "ix-a03", "vendor", "nec_ix", "domain_name", "example.com")
	put(t, sc, ds, "interfaces", "device", "ix-a03", "name", "GigaEthernet0.0", "kind", "physical", "ipv4", "dhcp")
	// Version 2: the PRF, AH on one side, the mode, the keys each way, the
	// identity that only one side writes, the lifetime in kilobytes.
	put(t, sc, ds, "ipsec_tunnels", "device", "rt-a01", "name", "CMAP-SAMPLE/10", "style", "policy", "local_if", "GigabitEthernet0/0", "peer_address", "203.0.113.21",
		"ike_version", "2", "ike_encryption", "aes256", "ike_hash", "sha256", "ike_dh", "dh14", "ike_prf", "sha256",
		"esp_encryption", "aes256", "esp_hash", "sha256", "ah_hash", "sha1", "ipsec_mode", "transport", "ipsec_lifetime_kb", "100000",
		"local_id", "fqdn:rt-a01.example.com", "psk", "dummy-psk-sample-a", "psk_remote", "dummy-psk-sample-b")
	put(t, sc, ds, "ipsec_tunnels", "device", "ix-a02", "name", "vpn-sample-a", "style", "policy", "local_if", "GigaEthernet0.0", "peer_address", "203.0.113.9",
		"ike_version", "2", "ike_encryption", "aes256", "ike_hash", "sha256", "ike_dh", "dh14", "ike_prf", "sha384",
		"esp_encryption", "aes256", "esp_hash", "sha256", "ipsec_lifetime_kb", "200000",
		"psk", "dummy-psk-sample-b", "psk_remote", "dummy-psk-sample-x")
	put(t, sc, ds, "ipsec_selectors", "device", "rt-a01", "tunnel", "CMAP-SAMPLE/10", "seq", "10", "local_net", "198.51.100.0/24", "remote_net", "192.0.2.0/25")
	put(t, sc, ds, "ipsec_selectors", "device", "ix-a02", "tunnel", "vpn-sample-a", "seq", "10", "local_net", "192.0.2.0/25", "remote_net", "198.51.100.0/24")
	// A key in hexadecimal on one side: the keys are not compared. The site
	// points at the center by its host name; the center waits for any peer.
	put(t, sc, ds, "ipsec_tunnels", append([]string{"device", "ix-a03", "name", "vpn-sample-b", "style", "policy", "local_if", "GigaEthernet0.0", "peer_address", "rt-a01",
		"local_id", "keyid:sample-site-c", "psk", "64756d6d792d6b65792d61", "psk_form", "hex"}, ike2...)...)
	put(t, sc, ds, "ipsec_tunnels", append([]string{"device", "rt-a01", "name", "CMAP-SAMPLE/90", "style", "policy", "local_if", "GigabitEthernet0/0", "peer_address", "any",
		"remote_id", "keyid:sample-site-c", "psk", "dummy-key-a"}, ike2...)...)
	put(t, sc, ds, "ipsec_selectors", "device", "ix-a03", "tunnel", "vpn-sample-b", "seq", "10", "local_net", "203.0.113.128/25", "remote_net", "198.51.100.0/24")
	put(t, sc, ds, "ipsec_selectors", "device", "rt-a01", "tunnel", "CMAP-SAMPLE/90", "seq", "10", "local_net", "198.51.100.0/24", "remote_net", "203.0.113.128/25")
	// A row with the peer "any" that nobody points at stays a peer outside.
	put(t, sc, ds, "ipsec_tunnels", append([]string{"device", "ix-a02", "name", "vpn-sample-z", "style", "policy", "local_if", "GigaEthernet0.0", "peer_address", "any",
		"remote_id", "keyid:sample-site-z", "psk", "dummy-psk-sample-z"}, ike2...)...)
	same(t, told(cross(sc, ds), "X-3"), []string{
		"X-300 ix-a02 peer_address", // vpn-sample-z
		"X-303 rt-a01 ike_prf",
		"X-304 rt-a01 ah_hash",
		"X-304 rt-a01 ipsec_mode",
		"X-305 rt-a01 psk",      // the other side checks this one with another key
		"X-306 rt-a01 local_id", // the other side waits for the address
		"X-308 rt-a01 ipsec_lifetime_kb",
		"X-365 ix-a03 psk_form",
	})
}

// The candidates of the two ends (appendix A 3.20): lists that differ and
// have a candidate in common are a warning, lists with nothing in common an
// error; the ports of the selectors are compared too.
func TestCrossIPsecCandidates(t *testing.T) {
	sc := defs(t)
	ds := vpnBase(t, sc)
	// IKE version 2: the lists of the two ends overlap for IKE (aes256 with
	// sha256 and dh14) and have nothing in common for ESP; PFS overlaps.
	put(t, sc, ds, "ipsec_tunnels", "device", "rt-a01", "name", "CMAP-SAMPLE/10", "style", "policy", "local_if", "GigabitEthernet0/0", "peer_address", "203.0.113.21",
		"ike_version", "2", "ike_encryption", "aes256", "ike_encryption_more", "aes128", "ike_hash", "sha256", "ike_dh", "dh14",
		"esp_encryption", "aes256", "esp_hash", "sha256", "pfs", "dh14", "psk", "dummy-psk-sample-a")
	put(t, sc, ds, "ipsec_proposals", "device", "rt-a01", "tunnel", "CMAP-SAMPLE/10", "phase", "ipsec", "seq", "20", "encryption", "aes128", "hash", "sha256")
	put(t, sc, ds, "ipsec_tunnels", "device", "ix-a02", "name", "vpn-sample-a", "style", "policy", "local_if", "GigaEthernet0.0", "peer_address", "203.0.113.9",
		"ike_version", "2", "ike_encryption", "aes256", "ike_hash", "sha256", "ike_dh", "dh14",
		"esp_encryption", "aes192", "esp_hash", "sha256", "esp_hash_more", "sha1", "pfs", "dh14", "pfs_more", "dh5", "psk", "dummy-psk-sample-a")
	// The selectors mirror each other but for a port.
	put(t, sc, ds, "ipsec_selectors", "device", "rt-a01", "tunnel", "CMAP-SAMPLE/10", "seq", "10", "local_net", "198.51.100.0/24", "remote_net", "192.0.2.0/25", "protocol", "udp", "local_port", "1701")
	put(t, sc, ds, "ipsec_selectors", "device", "ix-a02", "tunnel", "vpn-sample-a", "seq", "10", "local_net", "192.0.2.0/25", "remote_net", "198.51.100.0/24", "protocol", "udp")
	same(t, told(cross(sc, ds), "X-3"), []string{
		"X-366 rt-a01 ike_encryption",
		"X-304 rt-a01 esp_encryption",
		"X-366 rt-a01 pfs",
		"X-307 rt-a01 name",
	})
}

// The "(more)" columns belong to IKE version 2 and follow a first value
// (V-236); a port of a selector needs a protocol with ports (V-237).
func TestValuesIPsecLists(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	put(t, sc, ds, "ipsec_tunnels", "device", "rt-a01", "name", "CMAP-SAMPLE/10", "style", "policy", "peer_address", "any", "ike_version", "1",
		"ike_encryption", "aes256", "ike_encryption_more", "aes128", "ike_hash", "sha256", "ike_dh", "dh14", "esp_encryption", "aes256", "esp_hash", "sha256", "psk", "dummy-psk-sample-a")
	put(t, sc, ds, "ipsec_tunnels", "device", "rt-a01", "name", "CMAP-SAMPLE/20", "style", "policy", "peer_address", "any", "ike_version", "2",
		"ike_encryption", "aes256", "ike_encryption_more", "aes128", "esp_hash_more", "sha1", "pfs", "none", "pfs_more", "dh14", "psk", "dummy-psk-sample-b")
	put(t, sc, ds, "ipsec_selectors", "device", "rt-a01", "tunnel", "CMAP-SAMPLE/10", "seq", "10", "local_net", "0.0.0.0/0", "remote_net", "0.0.0.0/0", "protocol", "udp", "local_port", "1701")
	put(t, sc, ds, "ipsec_selectors", "device", "rt-a01", "tunnel", "CMAP-SAMPLE/10", "seq", "20", "local_net", "0.0.0.0/0", "remote_net", "0.0.0.0/0", "remote_port", "1701")
	fs, tally := run(sc, ds, nil)
	same(t, told(fs, "V-236"), []string{
		"V-236 rt-a01 ike_encryption_more", // version 1
		"V-236 rt-a01 esp_hash_more",       // no first value
		"V-236 rt-a01 pfs_more",            // PFS "none"
	})
	same(t, told(fs, "V-237"), []string{"V-237 rt-a01 remote_port"})
	for _, id := range []string{"V-236", "V-237"} {
		if !tally.Ran(id) {
			t.Errorf("%s did not run", id)
		}
	}
}

// The inner VRF of a tunnel interface is the VRF of that interface (V-238);
// the outer VRF of a row on an interface is the VRF of that interface
// (V-239).
func TestValuesIPsecVRF(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	put(t, sc, ds, "vrfs", "device", "rt-a01", "name", "wan-sample")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/5", "vrf", "wan-sample")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/6")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "Tunnel91")
	put(t, sc, ds, "tunnels", "device", "rt-a01", "if_name", "Tunnel91", "type", "ipsec")
	put(t, sc, ds, "ipsec_tunnels", "device", "rt-a01", "name", "Tunnel91", "style", "tunnel_if", "tunnel_if", "Tunnel91", "peer_address", "203.0.113.91", "ike_version", "2",
		"psk", "dummy-psk-sample-a", "outer_vrf", "wan-sample", "inner_vrf", "wan-sample")
	put(t, sc, ds, "ipsec_tunnels", "device", "rt-a01", "name", "CMAP-SAMPLE/91", "style", "policy", "local_if", "GigabitEthernet0/5", "peer_address", "any", "ike_version", "2",
		"psk", "dummy-psk-sample-b", "outer_vrf", "wan-sample", "inner_vrf", "wan-sample")
	put(t, sc, ds, "ipsec_tunnels", "device", "rt-a01", "name", "CMAP-SAMPLE/92", "style", "policy", "local_if", "GigabitEthernet0/5", "peer_address", "any", "ike_version", "2",
		"psk", "dummy-psk-sample-b")
	put(t, sc, ds, "ipsec_tunnels", "device", "rt-a01", "name", "CMAP-SAMPLE/93", "style", "policy", "local_if", "GigabitEthernet0/6", "peer_address", "any", "ike_version", "2",
		"psk", "dummy-psk-sample-b", "outer_vrf", "wan-sample")
	fs, tally := run(sc, ds, nil)
	same(t, told(fs, "V-238"), []string{"V-238 rt-a01 inner_vrf"})
	same(t, told(fs, "V-239"), []string{"V-239 rt-a01 outer_vrf", "V-239 rt-a01 outer_vrf"})
	for _, id := range []string{"V-238", "V-239"} {
		if !tally.Ran(id) {
			t.Errorf("%s did not run", id)
		}
	}
}
