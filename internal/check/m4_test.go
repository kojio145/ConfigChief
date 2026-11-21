package check_test

import (
	"testing"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
)

// The checks of the values of the VPN tables.
func TestValuesM4(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	// V-213: outside the network; the range holds an address of the device
	// (one that is kept back is fine); the ends the wrong way round.
	put(t, sc, ds, "dhcp_pools", "device", "rt-a01", "name", "pool-sample-a", "network", "198.51.100.0/24", "range_start", "198.51.100.1", "range_end", "198.51.100.100", "default_router", "192.0.2.1")
	put(t, sc, ds, "dhcp_pools", "device", "rt-a01", "name", "pool-sample-b", "network", "198.51.100.0/24", "range_start", "198.51.100.1", "range_end", "198.51.100.100", "excluded", "198.51.100.1-198.51.100.10", "default_router", "198.51.100.1")
	put(t, sc, ds, "dhcp_pools", "device", "rt-a01", "name", "pool-sample-c", "network", "198.51.100.0/24", "range_start", "198.51.100.200", "range_end", "198.51.100.150")
	put(t, sc, ds, "dhcp_pools", "device", "rt-a01", "name", "pool-sample-d", "network", "192.0.2.0/25", "range_start", "192.0.2.20", "range_end", "192.0.2.200")
	// V-215: no tunnel interface, one that is no tunnel, one of another type; no selector.
	sec := []string{"peer_address", "203.0.113.20", "ike_version", "1", "psk", "dummy-psk-sample", "ike_encryption", "aes256", "ike_hash", "sha256", "ike_dh", "dh14", "esp_encryption", "aes256", "esp_hash", "sha256"}
	row := func(cells ...string) { put(t, sc, ds, "ipsec_tunnels", append(cells, sec...)...) }
	put(t, sc, ds, "tunnels", "device", "rt-a01", "if_name", "Tunnel1", "type", "ipsec", "source", "GigabitEthernet0/0", "destination", "203.0.113.20")
	put(t, sc, ds, "tunnels", "device", "rt-a01", "if_name", "Tunnel2", "type", "gre", "source", "GigabitEthernet0/0", "destination", "203.0.113.20")
	row("device", "rt-a01", "name", "vpn-sample-ok", "style", "tunnel_if", "tunnel_if", "Tunnel1")
	row("device", "rt-a01", "name", "vpn-sample-none", "style", "tunnel_if")
	row("device", "rt-a01", "name", "vpn-sample-lost", "style", "tunnel_if", "tunnel_if", "Tunnel9")
	row("device", "rt-a01", "name", "vpn-sample-gre", "style", "tunnel_if", "tunnel_if", "Tunnel2")
	row("device", "rt-a01", "name", "CMAP-SAMPLE/10", "style", "policy", "local_if", "GigabitEthernet0/0")
	row("device", "rt-a01", "name", "CMAP-SAMPLE/20", "style", "policy", "local_if", "GigabitEthernet0/0")
	put(t, sc, ds, "ipsec_selectors", "device", "rt-a01", "tunnel", "CMAP-SAMPLE/10", "seq", "10", "local_net", "198.51.100.0/24", "remote_net", "203.0.113.128/25")
	// V-216: weak algorithms, one finding for a column.
	put(t, sc, ds, "ipsec_tunnels", "device", "rt-a02", "name", "vpn-sample-weak", "style", "policy", "peer_address", "203.0.113.30", "ike_version", "1", "psk", "dummy-psk-sample",
		"ike_encryption", "des", "ike_hash", "md5", "ike_dh", "dh2", "esp_encryption", "3des", "esp_hash", "sha1", "pfs", "dh5")
	put(t, sc, ds, "ipsec_selectors", "device", "rt-a02", "tunnel", "vpn-sample-weak", "seq", "10", "local_net", "203.0.113.128/25", "remote_net", "198.51.100.0/24")
	// V-217: an interface of a PPPoE row that is not on the sheet.
	put(t, sc, ds, "pppoe", "device", "rt-a01", "name", "wan-sample-a", "physical_if", "GigabitEthernet0/0", "logical_if", "Dialer1", "username", "sample-a@example.com", "password", "dummy-pppoe-secret-a")

	fs, tally := run(sc, ds, nil)
	same(t, told(fs, "V-213", "V-215", "V-216", "V-217"), []string{
		"V-213 rt-a01 default_router", // pool-sample-a
		"V-213 rt-a01 range_start",    // pool-sample-a: 198.51.100.1 is the interface
		"V-213 rt-a01 range_start",    // pool-sample-c: the wrong way round
		"V-213 rt-a01 range_end",      // pool-sample-d
		"V-215 rt-a01 tunnel_if",      // none
		"V-215 rt-a01 tunnel_if",      // lost
		"V-215 rt-a01 tunnel_if",      // gre
		"V-215 rt-a01 style",          // CMAP-SAMPLE/20
		"V-216 rt-a02 ike_encryption",
		"V-216 rt-a02 ike_hash",
		"V-216 rt-a02 ike_dh",
		"V-216 rt-a02 pfs",
		"V-217 rt-a01 logical_if",
	})
	for _, id := range []string{"V-213", "V-215", "V-216", "V-217"} {
		if !tally.Ran(id) {
			t.Errorf("%s did not run", id)
		}
	}
}

// vpnBase is two routers, each with a WAN address and a LAN.
func vpnBase(t *testing.T, sc *schema.Set) *model.Dataset {
	ds := model.NewDataset("book")
	put(t, sc, ds, "devices", "hostname", "rt-a01", "vendor", "cisco_ios")
	put(t, sc, ds, "devices", "hostname", "ix-a02", "vendor", "nec_ix")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/0", "kind", "physical", "ipv4", "203.0.113.9/30")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/1", "kind", "physical", "ipv4", "198.51.100.1/24")
	put(t, sc, ds, "interfaces", "device", "ix-a02", "name", "GigaEthernet0.0", "kind", "physical", "ipv4", "203.0.113.21/30")
	put(t, sc, ds, "interfaces", "device", "ix-a02", "name", "GigaEthernet1.0", "kind", "physical", "ipv4", "192.0.2.1/25")
	return ds
}

var ike = []string{"ike_version", "1", "psk", "dummy-psk-sample", "ike_encryption", "aes256", "ike_hash", "sha256", "ike_dh", "dh14", "ike_lifetime", "28800",
	"esp_encryption", "aes256", "esp_hash", "sha1", "pfs", "dh14", "ipsec_lifetime", "3600"}

// A tunnel of IPsec between two vendors: the one side has a source and a
// destination, the other has neither. Nothing to tell.
func TestCrossVPNQuiet(t *testing.T) {
	sc := defs(t)
	ds := vpnBase(t, sc)
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "Tunnel1", "kind", "tunnel", "ipv4", "192.0.2.209/30", "ip_mtu", "1400")
	put(t, sc, ds, "interfaces", "device", "ix-a02", "name", "Tunnel0.0", "kind", "tunnel", "ipv4", "192.0.2.210/30", "ip_mtu", "1400")
	put(t, sc, ds, "tunnels", "device", "rt-a01", "if_name", "Tunnel1", "type", "ipsec", "source", "GigabitEthernet0/0", "destination", "203.0.113.21")
	put(t, sc, ds, "tunnels", "device", "ix-a02", "if_name", "Tunnel0.0", "type", "ipsec")
	put(t, sc, ds, "ipsec_tunnels", append([]string{"device", "rt-a01", "name", "Tunnel1", "style", "tunnel_if", "tunnel_if", "Tunnel1", "peer_address", "203.0.113.21", "dpd_interval", "30"}, ike...)...)
	put(t, sc, ds, "ipsec_tunnels", append([]string{"device", "ix-a02", "name", "Tunnel0.0", "style", "tunnel_if", "tunnel_if", "Tunnel0.0", "peer_address", "203.0.113.9", "dpd_interval", "20", "ike_mode", "main"}, ike...)...)
	// A GRE tunnel that mirrors.
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "Tunnel2", "kind", "tunnel", "ipv4", "192.0.2.213/30")
	put(t, sc, ds, "interfaces", "device", "ix-a02", "name", "Tunnel1.0", "kind", "tunnel", "ipv4", "192.0.2.214/30")
	put(t, sc, ds, "tunnels", "device", "rt-a01", "if_name", "Tunnel2", "type", "gre", "source", "GigabitEthernet0/0", "destination", "203.0.113.21", "key", "100")
	put(t, sc, ds, "tunnels", "device", "ix-a02", "if_name", "Tunnel1.0", "type", "gre", "source", "203.0.113.21", "destination", "203.0.113.9", "key", "100")
	if fs := cross(sc, ds); len(fs) != 0 {
		t.Errorf("findings for a pair that agrees: %v", told(fs, "X-"))
	}
}

func TestCrossIPsec(t *testing.T) {
	sc := defs(t)
	ds := vpnBase(t, sc)
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "Tunnel1", "kind", "tunnel", "ipv4", "192.0.2.209/30", "ip_mtu", "1400", "tcp_mss", "1360")
	put(t, sc, ds, "interfaces", "device", "ix-a02", "name", "Tunnel0.0", "kind", "tunnel", "ipv4", "192.0.2.217/30", "ip_mtu", "1380", "tcp_mss", "1340")
	put(t, sc, ds, "tunnels", "device", "rt-a01", "if_name", "Tunnel1", "type", "ipsec", "source", "GigabitEthernet0/0", "destination", "203.0.113.21")
	put(t, sc, ds, "tunnels", "device", "ix-a02", "if_name", "Tunnel0.0", "type", "ipsec")
	// Everything differs between the two sides of the tunnel.
	put(t, sc, ds, "ipsec_tunnels", "device", "rt-a01", "name", "Tunnel1", "style", "tunnel_if", "tunnel_if", "Tunnel1", "peer_address", "203.0.113.21",
		"ike_version", "1", "ike_mode", "main", "psk", "dummy-psk-sample-1", "ike_encryption", "aes256", "ike_hash", "sha256", "ike_dh", "dh14", "ike_lifetime", "28800",
		"esp_encryption", "aes256", "esp_hash", "sha1", "pfs", "dh14", "ipsec_lifetime", "3600", "local_id", "fqdn:rt-a01.example.com", "remote_id", "fqdn:ix-a02.example.com", "dpd_interval", "30")
	put(t, sc, ds, "ipsec_tunnels", "device", "ix-a02", "name", "Tunnel0.0", "style", "tunnel_if", "tunnel_if", "Tunnel0.0", "peer_address", "203.0.113.9",
		"ike_version", "1", "ike_mode", "aggressive", "psk", "dummy-psk-sample-2", "ike_encryption", "aes128", "ike_hash", "sha256", "ike_dh", "dh14", "ike_lifetime", "86400",
		"esp_encryption", "aes256", "esp_hash", "sha256", "pfs", "none", "ipsec_lifetime", "3600", "local_id", "fqdn:ix-a02.example.com", "remote_id", "fqdn:rt-sample.example.com", "nat_traversal", "true")
	// Crypto maps on both sides: the selectors do not mirror.
	put(t, sc, ds, "ipsec_tunnels", append([]string{"device", "rt-a01", "name", "CMAP-SAMPLE/10", "style", "policy", "local_if", "GigabitEthernet0/1", "peer_address", "192.0.2.1"}, ike...)...)
	put(t, sc, ds, "ipsec_tunnels", append([]string{"device", "ix-a02", "name", "vpn-sample-lan", "style", "policy", "local_if", "GigaEthernet1.0", "peer_address", "198.51.100.1"}, ike...)...)
	put(t, sc, ds, "ipsec_selectors", "device", "rt-a01", "tunnel", "CMAP-SAMPLE/10", "seq", "10", "local_net", "198.51.100.0/24", "remote_net", "192.0.2.0/25")
	put(t, sc, ds, "ipsec_selectors", "device", "ix-a02", "tunnel", "vpn-sample-lan", "seq", "10", "local_net", "192.0.2.0/25", "remote_net", "198.51.100.0/25", "protocol", "ip")
	// Nobody of the case has this address; a peer that is not an address; the other side has no row.
	put(t, sc, ds, "ipsec_tunnels", append([]string{"device", "rt-a01", "name", "CMAP-SAMPLE/20", "style", "policy", "peer_address", "203.0.113.99"}, ike...)...)
	put(t, sc, ds, "ipsec_tunnels", append([]string{"device", "rt-a01", "name", "CMAP-SAMPLE/30", "style", "policy", "peer_address", "any"}, ike...)...)
	put(t, sc, ds, "interfaces", "device", "ix-a02", "name", "Loopback0.0", "kind", "loopback", "ipv4", "203.0.113.130/32")
	put(t, sc, ds, "ipsec_tunnels", append([]string{"device", "rt-a01", "name", "CMAP-SAMPLE/40", "style", "policy", "peer_address", "203.0.113.130"}, ike...)...)
	same(t, told(cross(sc, ds), "X-3"), []string{
		"X-300 rt-a01 peer_address", // CMAP-SAMPLE/20
		"X-300 rt-a01 peer_address", // CMAP-SAMPLE/30
		"X-301 rt-a01 peer_address", // CMAP-SAMPLE/40
		"X-302 rt-a01 ike_mode",
		"X-303 rt-a01 ike_encryption",
		"X-304 rt-a01 esp_hash",
		"X-304 rt-a01 pfs",
		"X-305 rt-a01 psk",
		"X-306 rt-a01 local_id", // the other side waits for rt-sample
		"X-307 rt-a01 name",
		"X-308 rt-a01 ike_lifetime",
		"X-309 rt-a01 dpd_interval",
		"X-309 rt-a01 nat_traversal",
		"X-313 rt-a01 if_name", // the tunnel interfaces of the IPsec pair
		"X-314 rt-a01 if_name", // MTU
		"X-314 rt-a01 if_name", // MSS
	})
}

func TestCrossTunnels(t *testing.T) {
	sc := defs(t)
	ds := vpnBase(t, sc)
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "Loopback0", "kind", "loopback", "ipv4", "203.0.113.129/32")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "Tunnel1", "kind", "tunnel", "ipv4", "192.0.2.209/30")
	put(t, sc, ds, "interfaces", "device", "ix-a02", "name", "Tunnel0.0", "kind", "tunnel", "ipv4", "192.0.2.210/30")
	// The one side sends from its loopback, the other points at the WAN address; type, keepalive.
	put(t, sc, ds, "tunnels", "device", "rt-a01", "if_name", "Tunnel1", "type", "gre", "source", "Loopback0", "destination", "203.0.113.21", "keepalive_interval", "10", "keepalive_retry", "3")
	put(t, sc, ds, "tunnels", "device", "ix-a02", "if_name", "Tunnel0.0", "type", "ipip", "source", "GigaEthernet1.0", "destination", "203.0.113.9", "keepalive_interval", "5", "keepalive_retry", "3")
	// The key on one side only.
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/2", "kind", "physical", "ipv4", "198.18.0.1/30")
	put(t, sc, ds, "interfaces", "device", "ix-a02", "name", "GigaEthernet2.0", "kind", "physical", "ipv4", "198.18.0.2/30")
	put(t, sc, ds, "tunnels", "device", "rt-a01", "if_name", "Tunnel2", "type", "gre", "source", "GigabitEthernet0/2", "destination", "198.18.0.2", "key", "100")
	put(t, sc, ds, "tunnels", "device", "ix-a02", "if_name", "Tunnel1.0", "type", "gre", "source", "GigaEthernet2.0", "destination", "198.18.0.1")
	// Nobody has the destination; the other side has no tunnel that points back.
	put(t, sc, ds, "tunnels", "device", "rt-a01", "if_name", "Tunnel3", "type", "gre", "source", "GigabitEthernet0/0", "destination", "203.0.113.99")
	put(t, sc, ds, "tunnels", "device", "rt-a01", "if_name", "Tunnel4", "type", "gre", "source", "GigabitEthernet0/1", "destination", "192.0.2.1")
	same(t, told(cross(sc, ds), "X-3"), []string{
		"X-300 rt-a01 destination",
		"X-301 rt-a01 destination",
		"X-311 rt-a01 source",      // Loopback0 and 203.0.113.9
		"X-311 rt-a01 destination", // 203.0.113.21 and GigaEthernet1.0
		"X-312 rt-a01 type",
		"X-312 rt-a01 key",
		"X-314 rt-a01 keepalive_interval",
	})
}
