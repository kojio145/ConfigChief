package check_test

import (
	"testing"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
)

// vrrp3Base is two routers on one LAN with IPv4 and IPv6 (made-up values):
// rt-v01 is on VRRP version 3 (as Cisco has it), rt-v02 on version 2.
func vrrp3Base(t *testing.T, sc *schema.Set, v1 string) *model.Dataset {
	ds := model.NewDataset("book")
	put(t, sc, ds, "devices", "hostname", "rt-v01", "vendor", "cisco_ios", "vrrp_version", v1)
	put(t, sc, ds, "devices", "hostname", "rt-v02", "vendor", "nec_ix")
	put(t, sc, ds, "interfaces", "device", "rt-v01", "name", "GigabitEthernet0/1", "kind", "physical", "ipv4", "198.51.100.2/24", "ipv6", "2001:db8:0:1::2/64")
	put(t, sc, ds, "interfaces", "device", "rt-v02", "name", "GigaEthernet0.0", "kind", "physical", "ipv4", "198.51.100.3/24", "ipv6", "2001:db8:0:1::3/64")
	put(t, sc, ds, "interfaces", "device", "rt-v01", "name", "GigabitEthernet0/2", "kind", "physical", "ipv6", "2001:db8:0:2::2/64")
	return ds
}

// The versions of an IPv4 group (X-405) and "vrrpv2" where it does nothing
// (V-291).
func TestVRRPVersion(t *testing.T) {
	sc := defs(t)
	ds := vrrp3Base(t, sc, "3")
	put(t, sc, ds, "fhrp", "device", "rt-v01", "if_name", "GigabitEthernet0/1", "protocol", "vrrp", "group", "1", "virtual_ip", "198.51.100.1", "priority", "110")
	put(t, sc, ds, "fhrp", "device", "rt-v02", "if_name", "GigaEthernet0.0", "protocol", "vrrp", "group", "1", "virtual_ip", "198.51.100.1", "priority", "100")
	// With "vrrpv2" the version 3 side meets version 2.
	put(t, sc, ds, "fhrp", "device", "rt-v01", "if_name", "GigabitEthernet0/1", "protocol", "vrrp", "group", "2", "virtual_ip", "198.51.100.4", "priority", "110", "vrrpv2", "true")
	put(t, sc, ds, "fhrp", "device", "rt-v02", "if_name", "GigaEthernet0.0", "protocol", "vrrp", "group", "2", "virtual_ip", "198.51.100.4", "priority", "100", "vrrpv2", "true")
	same(t, told(cross(sc, ds), "X-405"), []string{"X-405 rt-v01 protocol"})
	fs, _ := run(sc, ds, nil)
	same(t, told(fs, "V-291"), []string{"V-291 rt-v02 vrrpv2"})

	// Both on version 3: nothing to tell.
	ds = vrrp3Base(t, sc, "3")
	put(t, sc, ds, "devices", "hostname", "rt-v03", "vendor", "cisco_ios", "vrrp_version", "3")
	put(t, sc, ds, "interfaces", "device", "rt-v03", "name", "GigabitEthernet0/1", "kind", "physical", "ipv4", "198.51.100.5/24")
	put(t, sc, ds, "fhrp", "device", "rt-v01", "if_name", "GigabitEthernet0/1", "protocol", "vrrp", "group", "3", "virtual_ip", "198.51.100.1", "priority", "110")
	put(t, sc, ds, "fhrp", "device", "rt-v03", "if_name", "GigabitEthernet0/1", "protocol", "vrrp", "group", "3", "virtual_ip", "198.51.100.1", "priority", "100")
	same(t, told(cross(sc, ds), "X-405"), nil)
}

// IPv6 VRRP: the virtual addresses (V-290) and the pairs (X-406, X-343,
// X-346, X-300).
func TestIPv6VRRP(t *testing.T) {
	sc := defs(t)
	ds := vrrp3Base(t, sc, "3")
	// Everything differs but the priority.
	put(t, sc, ds, "ipv6_fhrp", "device", "rt-v01", "if_name", "GigabitEthernet0/1", "group", "2", "link_local", "fe80::1", "addresses", "2001:db8:0:1::1", "priority", "100", "advert_interval", "1000")
	put(t, sc, ds, "ipv6_fhrp", "device", "rt-v02", "if_name", "GigaEthernet0.0", "group", "2", "link_local", "fe80::2", "addresses", "2001:db8:0:1::1|2001:db8:0:1::5", "priority", "100", "advert_interval", "1500")
	// A group the other device of the subnet does not have; a group with
	// nobody else in the subnet, its addresses wrong.
	put(t, sc, ds, "ipv6_fhrp", "device", "rt-v01", "if_name", "GigabitEthernet0/1", "group", "4", "link_local", "fe80::4")
	put(t, sc, ds, "ipv6_fhrp", "device", "rt-v01", "if_name", "GigabitEthernet0/2", "group", "5", "link_local", "2001:db8:0:2::9", "addresses", "2001:db8:0:3::1")
	same(t, told(cross(sc, ds), "X-3", "X-4"), []string{
		"X-300 rt-v01 group",
		"X-343 rt-v01 priority",
		"X-346 rt-v01 group",
		"X-406 rt-v01 advert_interval",
		"X-406 rt-v01 addresses",
		"X-406 rt-v01 link_local",
	})
	fs, _ := run(sc, ds, nil)
	same(t, told(fs, "V-290"), []string{"V-290 rt-v01 addresses", "V-290 rt-v01 link_local"})

	// A pair that agrees.
	ds = vrrp3Base(t, sc, "3")
	put(t, sc, ds, "ipv6_fhrp", "device", "rt-v01", "if_name", "GigabitEthernet0/1", "group", "2", "link_local", "fe80::1", "addresses", "2001:db8:0:1::1", "priority", "110", "advert_interval", "1000")
	put(t, sc, ds, "ipv6_fhrp", "device", "rt-v02", "if_name", "GigaEthernet0.0", "group", "2", "link_local", "fe80::1", "addresses", "2001:db8:0:1::1", "priority", "100", "advert_interval", "1000")
	same(t, told(cross(sc, ds), "X-"), nil)
}
