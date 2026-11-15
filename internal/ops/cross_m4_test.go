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
			// The new vendor has no direction of NAT on an interface, and its
			// tunnel of IPsec has neither a source nor a destination: the map
			// holds the address of the other side.
			unused: []string{"NAT の向き", "送信元、宛先"},
			first:  3,
			mend: [][5]string{
				{"インタフェース", "IF 名", "GigaEthernet0.1", "NAT の向き", ""},
				{"インタフェース", "IF 名", "GigaEthernet1.0", "NAT の向き", ""},
				{"トンネル", "トンネル IF", "Tunnel0.0", "送信元", ""},
				{"トンネル", "トンネル IF", "Tunnel0.0", "宛先", ""},
			},
		},
		{
			old: "ix-w01.txt", oldHost: "ix-w01", newHost: "rt-w01", pack: "cisco_ios",
			ifs: map[string]string{"GigaEthernet0.0": "GigabitEthernet0/0", "GigaEthernet1.0": "GigabitEthernet0/1", "GigaEthernet0.1": "Dialer1", "Tunnel0.0": "Tunnel1"},
			// The new vendor marks the interfaces inside and outside of NAT,
			// and its tunnel says where it sends from and to.
			stops: []string{
				"NAT の向きが「inside」と「outside」のインタフェースがそろっていません",
				"NAT nat1 の外側 IF Dialer1 は、NAT の向きが「outside」ではありません",
				"Tunnel1 のトンネル の送信元 が空です",
				"Tunnel1 のトンネル の宛先 が空です",
			},
			first: 4,
			mend: [][5]string{
				{"インタフェース", "IF 名", "Dialer1", "NAT の向き", "outside"},
				{"インタフェース", "IF 名", "GigabitEthernet0/1", "NAT の向き", "inside"},
				{"トンネル", "トンネル IF", "Tunnel1", "送信元", "Dialer1"},
				{"トンネル", "トンネル IF", "Tunnel1", "宛先", "203.0.113.20"},
				// How often the other side is asked again has no line on the
				// new vendor.
				{"IPsec", "名前", "Tunnel0.0", "相手監視の再送回数", ""},
			},
		},
	})
}
