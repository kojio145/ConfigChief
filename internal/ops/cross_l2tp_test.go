package ops

import "testing"

// A router that takes callers in by L2TP over IPsec (an LNS) goes from one
// vendor to the other, both ways; the one that comes from NEC IX is a LAC as
// well (appendix A 3.21). What a person has to write for the new vendor is
// listed here. Both samples hold the whole of L2TP over IPsec: the row of
// IPsec that protects the callers (a peer "any", the transport mode) goes
// over with the row of L2TP that names it. The PPPoE server behind a LAC is
// not read yet, and a sample with a line that is not read cannot go through
// this test.
func TestCrossVendorL2TP(t *testing.T) {
	crossSites(t, "l2tp", []site{
		{
			old: "rt-l01.txt", oldHost: "rt-l01", newHost: "ix-l01", pack: "nec_ix",
			ifs: map[string]string{"GigabitEthernet0/0": "GigaEthernet0.0", "Loopback0": "Loopback0.0", "Virtual-Template1": "Tunnel0.0"},
			// Nothing to mend. The crypto map of the outside interface
			// becomes a dynamic map on the tunnel interface of the LNS,
			// under the name of the row ("/" written as "-"), and the row of
			// L2TP, which the import pointed at that row, names it. The
			// access list that picked L2TP (UDP 1701) has no line: the list
			// of the new vendor lets everything through.
			// One Virtual-Template became one tunnel interface: one caller
			// at a time.
			first: 1,
			notes: []string{"「受け口の IF」が 1 つなので、同時につなげるのは 1 台です"},
		},
		{
			old: "ix-l01.txt", oldHost: "ix-l01", newHost: "rt-l01", pack: "cisco_ios",
			ifs: map[string]string{"GigaEthernet0.0": "GigabitEthernet0/0", "GigaEthernet2.0": "GigabitEthernet0/1", "GigaEthernet2.1": "GigabitEthernet0/1.1",
				"Loopback0.0": "Loopback0", "Tunnel0.0": "Virtual-Template1", "Tunnel2.0": "Virtual-Template2"},
			// The new vendor has no line for the idle time of a LAC, nor for
			// the address of one caller. The PPPoE interfaces of the LAC are
			// written (appendix A 3.26): the tunnel interface of the old
			// vendor is mapped to the Virtual-Template the callers land on.
			unused: []string{"無通信で切る時間（秒）", "固定で払い出すアドレス"},
			first:  3,
			mend: [][5]string{
				{"L2TP", "名前", "lac-sample", "無通信で切る時間（秒）", ""},
				{"L2TP 接続ユーザ", "接続 ID", "example-user", "固定で払い出すアドレス", ""},
			},
			// The dynamic map on the tunnel interfaces of the LNS becomes a
			// crypto map on the interface the default route leaves by (the
			// row of the old vendor has no interface). The interface of the
			// LAC's tunnel is mapped to a virtual template, which this
			// vendor writes.
			notes: []string{"「自分側 IF」が空なので、デフォルト経路の出口の GigabitEthernet0/0 にクリプトマップを当てます"},
		},
	})
}
