package ops

import "testing"

// A site router with access lists (ports by name, a range, the ICMP type,
// several lists on an interface), NAT (a pool, a port forwarded to another
// port), a watch with a source address or an interval of its own while the
// line is down, and DHCP with a lease in minutes goes from one vendor to
// the other, both ways (appendix A 3.25). What a person has to write for the
// new vendor is listed in the test.
func TestCrossVendorAclNat(t *testing.T) {
	ways := []site{
		{
			old: "rt-g01.txt", oldHost: "rt-g01", newHost: "ix-g01", pack: "nec_ix",
			ifs: map[string]string{"GigabitEthernet0/0": "GigaEthernet0.0", "GigabitEthernet0/1": "GigaEthernet1.0"},
			// The new vendor has no direction of NAT on an interface: the pack
			// leaves it out and tells. Nothing is left for a person to mend.
			first: 2,
			// The probe takes its source as an interface.
			notes: []string{
				"送信元アドレス 203.0.113.1 を、そのアドレスを持つ IF GigaEthernet0.0 にして書きます",
				"GigaEthernet0.0: 「NAT の向き」は書きません",
			},
		},
		{
			old: "ix-g01.txt", oldHost: "ix-g01", newHost: "rt-g01", pack: "cisco_ios",
			ifs: map[string]string{"GigaEthernet0.0": "GigabitEthernet0/0", "GigaEthernet1.0": "GigabitEthernet0/1"},
			// The new vendor marks the interfaces inside and outside of NAT,
			// and has no count of failed probes: the pack writes the marks
			// from the rules of NAT, and the count as a delay of the watch.
			// Nothing is left for a person to mend.
			first: 6,
			notes: []string{
				"GigabitEthernet0/0: 「NAT の向き」が空なので、outside として書きます",
				"GigabitEthernet0/1: 「NAT の向き」が空なので、inside として書きます",
				"「断と判断する回数」（6 回）は Cisco に無いので、「断の反映の待ち」25 秒（間隔 5 秒 × 5）にして書きます",
				"行を順につないだリスト Gi0-0-in を作って当てます",
				"外側 IF GigabitEthernet0/0 のネットワークの長さ（27）で書きます",
				"「断の間の監視間隔」（30 秒）は Cisco に無いので、書きません",
			},
		},
	}
	crossSites(t, "f1", ways)
}
