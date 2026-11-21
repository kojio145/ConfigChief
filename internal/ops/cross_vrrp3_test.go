package ops

import "testing"

// A router with an IPv4 and an IPv6 group of VRRP goes from one vendor to the
// other, both ways (appendix A 3.23, "IPv6 の VRRP"; made-up values). Cisco
// runs both on version 3 ("fhrp version vrrp v3"); the IPv4 VRRP of NEC IX is
// version 2. Nothing is left for a person to write: the packs write the
// versions in their own way and tell.
func TestCrossVendorVRRP3(t *testing.T) {
	crossSites(t, "vrrp3", []site{
		{
			old: "rt-w05.txt", oldHost: "rt-w05", newHost: "ix-w05", pack: "nec_ix",
			ifs: map[string]string{"GigabitEthernet0/0": "GigaEthernet0.0", "GigabitEthernet0/1": "GigaEthernet1.0"},
			// The IPv4 group goes to version 2 (told); "vrrpv2" is how the
			// new vendor works anyway (not told).
			first: 1,
			notes: []string{"NEC IX の IPv4 の VRRP は版 2 だけなので"},
		},
		{
			old: "ix-w05.txt", oldHost: "ix-w05", newHost: "rt-w05", pack: "cisco_ios",
			ifs: map[string]string{"GigaEthernet0.0": "GigabitEthernet0/0", "GigaEthernet1.0": "GigabitEthernet0/1"},
			// The device goes to version 3 for its IPv6 group, and the IPv4
			// group keeps meeting peers on version 2 ("vrrpv2").
			first: 2,
			notes: []string{
				"機器全体を VRRP の版 3 にして書きます",
				"GigabitEthernet0/1 の VRRP 1: IPv6 の VRRP を書くために",
			},
		},
	})
}
