package ops

import "testing"

// A switch goes from one vendor to the other, both ways: VLANs, access and
// trunk ports, a LAG, the interface of a VLAN, spanning tree and the
// management settings. A person only decides the names of the interfaces
// (the interface map); what the new pack writes in another way is told
// (G-614), and nothing is left to mend.
func TestCrossVendorQX(t *testing.T) {
	ports := map[string]string{
		"GigabitEthernet1/0/1": "GigabitEthernet1/0/1", "GigabitEthernet1/0/2": "GigabitEthernet1/0/2", "GigabitEthernet1/0/3": "GigabitEthernet1/0/3",
		"GigabitEthernet1/0/23": "GigabitEthernet1/0/23", "GigabitEthernet1/0/24": "GigabitEthernet1/0/24",
	}
	with := func(more map[string]string) map[string]string {
		out := map[string]string{}
		for k, v := range ports {
			out[k] = v
		}
		for k, v := range more {
			out[k] = v
		}
		return out
	}
	crossSites(t, "qx", []site{
		{
			old: "sw-c01.txt", oldHost: "sw-c01", newHost: "sw-q01", pack: "nec_qx",
			ifs:   with(map[string]string{"Port-channel1": "Bridge-Aggregation1", "Vlan10": "Vlan-interface10"}),
			first: 8,
			notes: []string{
				"「スパニングツリーを使う」が空なので、有効にして書きます",
				"「拡張システム ID」は書きません",
				"VTP の設定（transparent）は書きません",
				"「デフォルトゲートウェイ」192.0.2.126 を、デフォルト経路として書きます",
				"GigabitEthernet1/0/2: 「アクセスポートをすぐ転送にする」（全体の既定）は QX に無いので、このポートに stp edged-port を書きます",
				"GigabitEthernet1/0/2: ストーム制御 5.00% を、5% として書きます",
				"GigabitEthernet1/0/3: トランクの「許可 VLAN」が空なので、all として書きます",
				"ユーザ admin: 「使える接続方式」が空なので、terminal・ssh・telnet で書きます",
			},
		},
		{
			// The other way: what only this device says (spanning tree
			// switched on, a description of a VLAN, what a user may log in
			// by) is left out, and told. A list that counts from 0 is
			// counted anew.
			old: "sw-q11.txt", oldHost: "sw-q11", newHost: "sw-c11", pack: "cisco_ios",
			ifs:   with(map[string]string{"Bridge-Aggregation1": "Port-channel1", "Vlan-interface10": "Vlan10"}),
			first: 4,
			notes: []string{
				"ACL sample-in: 順番が 0 から始まっています（Cisco の順番は 1 以上です）。同じ並びのまま、10・20・30 … と振り直して書きます",
				"「スパニングツリーを使う」の「有効」は書きません",
				"ユーザ admin: 「使える接続方式」（ssh・terminal）は書きません",
				"VLAN 10 の説明「sample office floor」は書きません",
			},
		},
		{
			// A layer-3 switch with what the second round reads: numbered
			// access lists (the pack gives them numbers of the new device,
			// in the lists and in what names them), a remark, a filter that
			// drops what no entry matches, a watch of the line, a routing
			// policy with a community, port security, banners and the
			// console.
			old: "l3-c31.txt", oldHost: "l3-c31", newHost: "l3-q31", pack: "nec_qx",
			ifs:   map[string]string{"GigabitEthernet1/0/1": "GigabitEthernet1/0/1", "GigabitEthernet1/0/24": "GigabitEthernet1/0/24", "Vlan10": "Vlan-interface10"},
			first: 10,
			notes: []string{
				"ACL 10: 番号を QX の番号 2010 に付け替えて書きます",
				"ACL 110: 番号を QX の番号 3010 に付け替えて書きます",
				"ACL 3010 の 10 番: 注記の行を、次の行（20 番）の注記として書きます",
				"ACL 3010: IF に当てていますが、最後の行が「すべての通信」の行ではありません",
				"「スパニングツリーを使う」が空なので、有効にして書きます",
				"ポートセキュリティを使うポートがあるので、port-security enable を書きます",
				"覚えた MAC アドレスを設定に残します",
				"違反のときの動作 restrict を、protect として書きます",
				"回線監視 1: 「断と判断する回数」が空なので、1 回として書きます",
				"ユーザ admin: 「使える接続方式」が空なので、terminal・ssh・telnet で書きます",
			},
		},
		{
			// The other way: a hybrid port that is a trunk in effect, a
			// filter whose rest is passed (said in full by the import),
			// secrets the device saved (they stop the run until a person
			// writes them anew). A rate of storm suppression is written
			// as a rate.
			old: "l3-q41.txt", oldHost: "l3-q41", newHost: "l3-c41", pack: "cisco_ios",
			ifs: map[string]string{"GigabitEthernet1/0/1": "GigabitEthernet1/0/1", "GigabitEthernet1/0/24": "GigabitEthernet1/0/24", "Vlan-interface10": "Vlan10"},
			stops: []string{
				"SNMP コミュニティに、別メーカーの機器が暗号化した形の値があります",
				"の認証の鍵は、別メーカーの機器が暗号化した形です",
			},
			// The version and the key of VRRP, how often a probe may fail and
			// a comment tied to an entry of a list have no line in the pack
			// of the new vendor.
			unused: []string{"版", "断と判断する回数", "行の注記"},
			first:  14,
			mend: [][5]string{
				{"SNMP コミュニティ", "コミュニティ", "cipher $c$3$sampleSampleSampleSampleSample", "コミュニティ", "example-ro"},
				{"冗長化（VRRP・HSRP）", "グループ番号", "10", "鍵", ""},
				{"冗長化（VRRP・HSRP）", "グループ番号", "10", "認証", ""},
				{"冗長化（VRRP・HSRP）", "グループ番号", "10", "版", ""},
				{"回線監視", "番号・名前", "1", "断と判断する回数", ""},
				{"ACL 行", "ACL 名", "3000", "行の注記", ""},
			},
			notes: []string{
				"ACL 2000: 順番が 0 から始まっています",
				"ACL 3000: 順番が 0 から始まっています",
				"ACL 2000: この番号は、Cisco の standard の ACL の番号ではないので",
				"名前 ACL-3000 に付け替えて書きます",
				"「スパニングツリーを使う」の「有効」は書きません",
				"GigabitEthernet1/0/1: モード hybrid を、trunk として書きます",
				"ユーザ admin: 「使える接続方式」（ssh）は書きません",
			},
		},
		{
			// What the two vendors name alike since the columns were made
			// one (appendix A 3.22 and 3.24): the region and the instances
			// of MST, the timers and the root of the common tree, the
			// guards, the cost and the priority of a port, storm control as
			// a rate, the short LACP timeout, DHCP snooping, a port mirror.
			old: "sw-q51.txt", oldHost: "sw-q51", newHost: "sw-c51", pack: "cisco_ios",
			ifs:   with(map[string]string{"Bridge-Aggregation1": "Port-channel1"}),
			first: 2,
			notes: []string{
				"「スパニングツリーを使う」の「有効」は書きません",
				"「ルートブリッジの指定（機器全体）」primary を、MST のインスタンス 0（共通の木）の指定として書きます",
			},
		},
	})
}
