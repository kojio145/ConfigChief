package ops

import "testing"

// A site router on an IPoE line goes from one vendor to the other, both ways
// (appendix A 3.23): the address by router advertisements, a delegated prefix
// for the LAN, router advertisements and a DHCPv6 server on the LAN, an IPv6
// filter, a static route by a link-local next hop, OSPFv3, and a tunnel that
// carries IPv4 over IPv6 to a named gateway. What a person has to write for
// the new vendor is listed here.
func TestCrossVendorIPv6(t *testing.T) {
	crossSites(t, "ipv6", []site{
		{
			old: "rt-v11.txt", oldHost: "rt-v11", newHost: "ix-v11", pack: "nec_ix",
			ifs: map[string]string{"GigabitEthernet0/0": "GigaEthernet0.0", "GigabitEthernet0/1": "GigaEthernet1.0", "Loopback0": "Loopback0.0", "Tunnel1": "Tunnel0.0"},
			// The new vendor has no line that sets a link-local address by
			// hand. Its routes have no name either: that is told as a value
			// with no line (G-602) once the interfaces can be written, so
			// the cell is emptied here with the rest.
			stops: []string{"リンクローカルアドレスを手で決める書き方を、NEC IX では確かめていません"},
			first: 2,
			mend: [][5]string{
				{"IPv6 インタフェース", "IF 名", "GigaEthernet0.0", "リンクローカルアドレス", ""},
				{"IPv6 スタティック経路", "宛先", "2001:db8:100::/48", "説明", ""},
				// The old vendor sends router advertisements when nothing is
				// said; the new one does not (told as V-269, an information).
				{"IPv6 インタフェース", "IF 名", "GigaEthernet1.0", "RA を送る", "send"},
			},
			// The new vendor forwards IPv6 without a line that says so.
			notes: []string{"「IPv6 の転送」を有効にする行は NEC IX にありません"},
		},
		{
			old: "ix-v11.txt", oldHost: "ix-v11", newHost: "rt-v11", pack: "cisco_ios",
			ifs: map[string]string{"GigaEthernet0.0": "GigabitEthernet0/0", "GigaEthernet1.0": "GigabitEthernet0/1", "Loopback0.0": "Loopback0", "Tunnel0.0": "Tunnel1"},
			// The new vendor has no line for an EUI-64 address out of a
			// delegated prefix. It has none that asks for the DNS servers
			// either (G-602, once the interfaces can be written).
			stops: []string{"もらったプレフィックスに EUI-64 で付ける書き方を、Cisco では確かめていません"},
			first: 3,
			mend: [][5]string{
				{"IPv6 インタフェース", "IF 名", "GigabitEthernet0/1", "下 64 ビットを EUI-64 で作る", ""},
				{"IPv6 インタフェース", "IF 名", "GigabitEthernet0/0", "DHCPv6 でもらう情報", ""},
			},
			// The new vendor needs the switch of IPv6 routing, and a lifetime
			// on the line that advertises a DNS server.
			notes: []string{"「IPv6 の転送」を有効にして書きます", "「RA で配る DNS の生存時間」が空なので、1800 秒で書きます"},
		},
	})
}
