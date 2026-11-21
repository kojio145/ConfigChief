package ops

import "testing"

// A hub router goes from one vendor to the other, both ways (appendix A
// 3.20, the third round of IPsec): a tunnel interface for peers whose
// address is not known (a virtual template on one vendor, a tunnel
// interface with a dynamic map on the other), a tunnel whose peer is
// written by its name, and a tunnel whose inside is in a VRF. What a person
// has to write for the new vendor is listed here.
func TestCrossVendorIPsecThird(t *testing.T) {
	crossSites(t, "ipsec_h2", []site{
		{
			old: "rt-q01.txt", oldHost: "rt-q01", newHost: "ix-q01", pack: "nec_ix",
			ifs: map[string]string{"GigabitEthernet0/0": "GigaEthernet0.0", "GigabitEthernet0/1": "GigaEthernet1.0", "GigabitEthernet0/2": "GigaEthernet2.0",
				"Tunnel1": "Tunnel1.0", "Tunnel2": "Tunnel2.0", "Virtual-Template1": "Tunnel0.0"},
			// Nothing to mend. The virtual template becomes a tunnel
			// interface with a dynamic map (one peer at a time on the new
			// vendor), the peer by its name a policy with "peer-fqdn-ipv4",
			// and the VRF of the tunnel interface stays what it is. The
			// tunnels of IKE version 1 have no source on the new vendor
			// (told).
			first: 2,
			notes: []string{"トンネル Tunnel1.0: 「送信元」（GigaEthernet0.0）は書きません", "トンネル Tunnel2.0: 「送信元」（GigaEthernet0.0）は書きません"},
		},
		{
			old: "ix-q01.txt", oldHost: "ix-q01", newHost: "rt-q01", pack: "cisco_ios",
			ifs: map[string]string{"GigaEthernet0.0": "GigabitEthernet0/0", "GigaEthernet1.0": "GigabitEthernet0/1", "GigaEthernet2.0": "GigabitEthernet0/2",
				"Tunnel0.0": "Virtual-Template1", "Tunnel1.0": "Tunnel1", "Tunnel2.0": "Tunnel2"},
			// The tunnel interface of the peer "any" is paired with a
			// virtual template on the sheet of interfaces. A key the old
			// device saved encrypted cannot go to another vendor: a person
			// writes the key itself. The tunnel to the peer by its name
			// gets that name as its destination, and the interface of the
			// default route as its source (told).
			stops: []string{"IPsec Tunnel1.0 の鍵は、別メーカーの機器が暗号化した形です"},
			first: 3,
			mend: [][5]string{
				{"IPsec", "名前", "Tunnel1.0", "事前共有鍵", "dummy-psk-sample-name"},
			},
			notes: []string{"トンネル Tunnel1: 「送信元」が空なので、デフォルト経路の出口の GigabitEthernet0/0 から送ると書きます"},
		},
	})
}
