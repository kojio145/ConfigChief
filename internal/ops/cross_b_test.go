package ops

import "testing"

// A site router with IPsec on a tunnel interface and IKE version 2 goes
// from one vendor to the other, both ways (appendix A 3.20). What a person
// has to write for the new vendor is listed here.
func TestCrossVendorIKEv2(t *testing.T) {
	crossSites(t, "ikev2", []site{
		{
			old: "rt-y01.txt", oldHost: "rt-y01", newHost: "ix-y01", pack: "nec_ix",
			ifs: map[string]string{"GigabitEthernet0/0": "GigaEthernet0.0", "GigabitEthernet0/1": "GigaEthernet1.0", "Tunnel1": "Tunnel0.0"},
			// Nothing to mend: the source and the destination of the tunnel,
			// the identities and the two keys all have a line on the new
			// vendor. It writes how often a message is sent again together
			// with the interval, and tells that it took its own count.
			first: 1,
			notes: []string{"相手監視の再送回数が空なので、NEC IX の既定の 3 回として書きます"},
		},
		{
			old: "ix-y01.txt", oldHost: "ix-y01", newHost: "rt-y01", pack: "cisco_ios",
			ifs: map[string]string{"GigaEthernet0.0": "GigabitEthernet0/0", "GigaEthernet1.0": "GigabitEthernet0/1", "Tunnel0.0": "Tunnel1"},
			// The new vendor writes the watch of the peer with the interval
			// between the retries, which the old one leaves to the device.
			stops: []string{"IPsec Tunnel0.0 の相手監視の再送間隔"},
			first: 1,
			mend: [][5]string{
				{"IPsec", "名前", "Tunnel0.0", "相手監視の再送間隔（秒）", "5"},
			},
		},
	})
}
