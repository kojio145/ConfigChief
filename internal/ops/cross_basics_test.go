package ops

import "testing"

// The basics of a device go from one vendor to the other, both ways
// (appendix A 3.26): the time stamps and the buffer of the log, the console,
// the switches only one vendor has, a PPPoE line that asks the peer too, and
// the conditions and marks of QoS both vendors have. A person only decides
// the names of the interfaces; what the new pack writes in another way, or
// leaves out, is told (G-614), and nothing is left to mend.
func TestCrossVendorBasics(t *testing.T) {
	crossSites(t, "basics", []site{
		{
			old: "rt-g01.txt", oldHost: "rt-g01", newHost: "ix-g02", pack: "nec_ix",
			ifs:   map[string]string{"GigabitEthernet0/0": "GigaEthernet0.0", "GigabitEthernet0/1": "GigaEthernet1.0", "Dialer1": "GigaEthernet0.1"},
			first: 4,
			notes: []string{
				"「ログの時刻に足すもの」（msec・localtime）は NEC IX にありません",
				"「デバッグの時刻の付け方」は書きません",
				"コンソールの無操作切断（6 分）は書きません",
				"「つなぐきっかけの通信」（ip）は NEC IX にありません",
			},
		},
		{
			old: "ix-g01.txt", oldHost: "ix-g01", newHost: "rt-g02", pack: "cisco_ios",
			ifs:   map[string]string{"GigaEthernet0.0": "GigabitEthernet0/0", "GigaEthernet1.0": "GigabitEthernet0/1", "Loopback0.0": "Loopback0"},
			first: 5,
			notes: []string{
				"ログの時刻の付け方 time（時刻だけ）は Cisco に無いので、datetime",
				"「NTP サーバとして動く」を有効にする行は Cisco にありません",
				"「転送のキャッシュ」を有効にする行は書きません",
				"TCP MSS 調整の auto は Cisco に無いので、IP MTU 1454 − 40 = 1414 として書きます",
				"「ポートの説明」を、IF の「説明」として書きます",
			},
		},
	})
}
