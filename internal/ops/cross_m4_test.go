package ops

import "testing"

// A site router with PPPoE, IPsec on a tunnel interface, NAT and a DHCP
// server goes from one vendor to the other, both ways (the end of stage M4,
// spec 18). What a person has to write for the new vendor is listed here.
func TestCrossVendorVPN(t *testing.T) {
	crossSites(t, "vpn", []site{
		{
			old: "rt-w01.txt", oldHost: "rt-w01", newHost: "ix-w01", pack: "nec_ix",
			ifs: map[string]string{"GigabitEthernet0/0": "GigaEthernet0.0", "GigabitEthernet0/1": "GigaEthernet1.0", "Dialer1": "GigaEthernet0.1", "Tunnel1": "Tunnel0.0"},
			// The new vendor has no direction of NAT on an interface (the pack
			// leaves it out and tells). Its tunnel of IPsec has neither a
			// source nor a destination: the map holds the address of the other
			// side (the destination, which is the peer of the IPsec row, is
			// left out without a word), and that the source is not written is
			// told.
			first: 2,
			notes: []string{"GigaEthernet0.1: 「NAT の向き」は書きません", "「送信元」（GigaEthernet0.1）は書きません"},
			mend:  [][5]string{},
		},
		{
			old: "ix-w01.txt", oldHost: "ix-w01", newHost: "rt-w01", pack: "cisco_ios",
			ifs: map[string]string{"GigaEthernet0.0": "GigabitEthernet0/0", "GigaEthernet1.0": "GigabitEthernet0/1", "GigaEthernet0.1": "Dialer1", "Tunnel0.0": "Tunnel1"},
			// The marks of the interfaces inside and outside of NAT come from
			// the pack (every interface with an address that is not outside is
			// inside, the tunnel too). The tunnel says where it sends to (the
			// peer of the IPsec row) and from (the interface the default route
			// leaves by, which is told).
			first: 7,
			notes: []string{
				"Dialer1: 「NAT の向き」が空なので、outside として書きます",
				"GigabitEthernet0/1: 「NAT の向き」が空なので、inside として書きます",
				"Tunnel1: 「NAT の向き」が空なので、inside として書きます",
				"「送信元」が空なので、デフォルト経路の出口の Dialer1 から送ると書きます",
			},
			mend: [][5]string{
				// How often the other side is asked again has no line on the
				// new vendor.
				{"IPsec", "名前", "Tunnel0.0", "相手監視の再送回数", ""},
			},
		},
	})
}
