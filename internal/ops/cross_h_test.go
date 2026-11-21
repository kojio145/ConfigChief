package ops

import "testing"

// A site router whose IPsec has no tunnel interface for one peer (a crypto
// map on the outside interface; a policy map on it), candidates in both
// forms and a tunnel interface that tells its peer what it protects goes
// from one vendor to the other, both ways (appendix A 3.13 and 3.20). What
// a person has to write for the new vendor is listed here.
func TestCrossVendorIPsecPolicy(t *testing.T) {
	crossSites(t, "ipsec_h", []site{
		{
			old: "rt-p01.txt", oldHost: "rt-p01", newHost: "ix-p01", pack: "nec_ix",
			ifs: map[string]string{"GigabitEthernet0/0": "GigaEthernet0.0", "GigabitEthernet0/1": "GigaEthernet1.0", "Tunnel1": "Tunnel0.0"},
			// Nothing to mend. The entry of the crypto map becomes a policy
			// map on the outside interface, with a list made from the same
			// selectors and the second transform set as a second proposal.
			// The access list on the tunnel interface becomes the traffic
			// selectors of its IKEv2 profile, and the lists of the IKEv2
			// proposal the lists of that profile. The new vendor has no
			// second combination with IKE version 2: the values of the
			// second transform set join the lists, which is told.
			first: 1,
			notes: []string{"IPsec Tunnel1 の追加の提案 20 番（段 ipsec）: NEC IX の IKEv2 は、候補を値の並びで書きます"},
		},
		{
			old: "ix-p01.txt", oldHost: "ix-p01", newHost: "rt-p01", pack: "cisco_ios",
			ifs: map[string]string{"GigaEthernet0.0": "GigabitEthernet0/0", "GigaEthernet1.0": "GigabitEthernet0/1", "Tunnel0.0": "Tunnel1", "Tunnel1.0": "Tunnel2"},
			// Nothing to mend. The policy map on the outside interface
			// becomes an entry of a crypto map named after that interface.
			// What the maps and the IKEv2 profile tell the peer that they
			// protect becomes an access list on each tunnel interface. The
			// lists of ESP values become a transform set for each
			// combination, and the tunnel of IKE version 1, which has no
			// source on the old vendor, sends from the interface of the
			// default route (both are told).
			first: 2,
			notes: []string{"「ESP 暗号（追加）」「ESP ハッシュ（追加）」の値を、組み合わせごとのトランスフォームセット（全部で 2 個）として書きます",
				"トンネル Tunnel2: 「送信元」が空なので、デフォルト経路の出口の GigabitEthernet0/0 から送ると書きます"},
		},
	})
}
