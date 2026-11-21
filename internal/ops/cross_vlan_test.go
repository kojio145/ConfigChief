package ops

import "testing"

// A small router with built-in switch ports goes from one vendor to the
// other, both ways. One vendor names a VLAN by its number on every port and
// has an interface for the VLAN; the other puts ports into VLAN groups and
// joins groups and tagged subinterfaces with a bridge group. A person only
// decides the names of the interfaces (the interface map); what the new pack
// writes in another way is told (G-614), and nothing is left to mend.
func TestCrossVendorVLAN(t *testing.T) {
	ports := map[string]string{"FastEthernet0": "GigaEthernet2:port1", "FastEthernet1": "GigaEthernet2:port2", "FastEthernet2": "GigaEthernet2:port3", "FastEthernet3": "GigaEthernet2:port4"}
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
	crossSites(t, "vlan", []site{
		{
			// A trunk carries the VLANs: their interfaces become BVIs, and
			// the pack bridges the ports and the tagged subinterfaces.
			old: "rt-v01.txt", oldHost: "rt-v01", newHost: "ix-v01", pack: "nec_ix",
			ifs:   with(map[string]string{"FastEthernet4": "GigaEthernet2:port5", "GigabitEthernet8": "GigaEthernet0.0", "Vlan10": "BVI0", "Vlan20": "BVI1"}),
			first: 14,
			notes: []string{
				"VLAN 10 の IF BVI0 を、ブリッジグループ 10 の BVI として書きます",
				"VLAN 20 の IF BVI1 を、ブリッジグループ 20 の BVI として書きます",
				"bridge irb enable を書きます",
				"VLAN 10 のアクセスポート（ポート 1・2）を、VLAN グループ 1（IF GigaEthernet2:1.0）にまとめ",
				"VLAN 20 のアクセスポート（ポート 3・4）を、VLAN グループ 3（IF GigaEthernet2:3.0）にまとめ",
				"トランクのポート GigaEthernet2:port5 の VLAN 10 を、タグ付きのサブ IF GigaEthernet2:5.1（ブリッジグループ 10）として書きます",
				"トランクのポート GigaEthernet2:port5 の VLAN 20 を、タグ付きのサブ IF GigaEthernet2:5.2（ブリッジグループ 20）として書きます",
				"トランクのポート GigaEthernet2:port5 を、ポート 1 つの VLAN グループ 5（IF GigaEthernet2:5.0）として書きます",
				"GigaEthernet2:port1: NEC IX にスパニングツリーは無いので",
				"GigaEthernet2:port2: NEC IX にスパニングツリーは無いので",
				"VLAN 10（sample-office） の定義は書きません",
				"VLAN 20（sample-guest） の定義は書きません",
				"説明「office desk 1」は書きません",
				"説明「to sw-b01」は書きません",
			},
		},
		{
			// No trunk: the interfaces of the VLANs become port VLANs. The
			// ports without a line of their own are in VLAN 1.
			old: "rt-v02.txt", oldHost: "rt-v02", newHost: "ix-v02", pack: "nec_ix",
			ifs: with(map[string]string{"GigabitEthernet8": "GigaEthernet0.0", "GigabitEthernet8.10": "GigaEthernet0.1", "GigabitEthernet8.20": "GigaEthernet0.2",
				"Vlan1": "GigaEthernet2:1.0", "Vlan20": "GigaEthernet2:2.0"}),
			first: 4,
			notes: []string{
				"GigaEthernet2:port1: スイッチポートの行が無いので、VLAN 1 のアクセスポートとして書きます",
				"GigaEthernet2:port2: スイッチポートの行が無いので、VLAN 1 のアクセスポートとして書きます",
				"GigaEthernet2:port3: アクセス VLAN 20 を、VLAN グループ 2（IF GigaEthernet2:2.0）のポートとして書きます",
				"GigaEthernet2:port4: アクセス VLAN 20 を、VLAN グループ 2（IF GigaEthernet2:2.0）のポートとして書きます",
			},
		},
		{
			// Two trunks carry VLANs that the device has no port and no
			// interface for: the tagged subinterfaces of both trunks stand
			// in one bridge group. A VLAN only one trunk carries goes
			// nowhere, and is told.
			old: "rt-v03.txt", oldHost: "rt-v03", newHost: "ix-v03", pack: "nec_ix",
			ifs:   with(map[string]string{"GigabitEthernet8": "GigaEthernet0.0", "Vlan10": "BVI0"}),
			first: 16,
			notes: []string{
				"VLAN 10 の IF BVI0 を、ブリッジグループ 10 の BVI として書きます",
				"bridge irb enable を書きます",
				"説明「to sw-b01」は書きません",
				"説明「to sw-b02」は書きません",
				"VLAN 10 のアクセスポート（ポート 1・2）を、VLAN グループ 1（IF GigaEthernet2:1.0）にまとめ",
				"トランクのポート GigaEthernet2:port3 の VLAN 10 を、タグ付きのサブ IF GigaEthernet2:3.1（ブリッジグループ 10）として書きます",
				"トランクのポート GigaEthernet2:port3 の VLAN 30 を、タグ付きのサブ IF GigaEthernet2:3.2（ブリッジグループ 30）として書きます",
				"トランクのポート GigaEthernet2:port3 の VLAN 40 を、タグ付きのサブ IF GigaEthernet2:3.3（ブリッジグループ 40）として書きます",
				"トランクのポート GigaEthernet2:port4 の VLAN 30 を、タグ付きのサブ IF GigaEthernet2:4.1（ブリッジグループ 30）として書きます",
				"トランクのポート GigaEthernet2:port4 の VLAN 40 を、タグ付きのサブ IF GigaEthernet2:4.2（ブリッジグループ 40）として書きます",
				"許可 VLAN のうち 50 は、この機器にポートも BVI も無く、ほかのトランクも通さないので書きません",
				"VLAN 10（sample-office） の定義は書きません",
				"VLAN 30（sample-transit） の定義は書きません",
				"VLAN 40（sample-camera） の定義は書きません",
				"VLAN 50 の定義は書きません",
				"トランクのポート GigaEthernet2:port3 を、ポート 1 つの VLAN グループ 3（IF GigaEthernet2:3.0）として書きます",
			},
		},
		{
			// The rest of a switch (2026-11-21): what the new vendor does
			// not have is taken away and told (spanning tree, LLDP that
			// was on), or taken away without a word where nothing changes
			// (CDP that was off); a port mirror is written the way of the
			// new vendor.
			old: "rt-v04.txt", oldHost: "rt-v04", newHost: "ix-v04", pack: "nec_ix",
			ifs:    with(map[string]string{"GigabitEthernet8": "GigaEthernet0.0", "Vlan10": "GigaEthernet2:1.0", "Vlan20": "GigaEthernet2:3.0"}),
			unused: []string{"表「L2 の全体設定」の 1 行は生成に使われませんでした"},
			first:  13,
			mend:   [][5]string{{"L2 の全体設定", "機器", "ix-v04", "VTP のモード", ""}},
			notes: []string{
				"GigaEthernet2:port3: CDP・LLDP は NEC IX に無いので書きません",
				"スパニングツリーの全体設定は書きません",
				"ループガード・BPDU フィルタの既定",
				"CDP・LLDP は NEC IX に無いので書きません（隣の機器からは、この機器が見えなくなります）",
				"止まったポートを自動で戻す設定（errdisable recovery）は NEC IX に無いので書きません",
				"GigaEthernet2:port1: NEC IX にスパニングツリーは無いので、スパニングツリーのガード",
				"GigaEthernet2:port1: アクセス VLAN 10 を、VLAN グループ 1",
				"GigaEthernet2:port2: アクセス VLAN 10 を、VLAN グループ 1",
				"GigaEthernet2:port3: アクセス VLAN 20 を、VLAN グループ 3",
				"GigaEthernet2:port4: アクセス VLAN 20 を、VLAN グループ 3",
				"VLAN 10 の定義は書きません",
				"VLAN 20 の定義は書きません",
			},
		},
		{
			// The other way: the number of a VLAN group is the number of
			// the VLAN.
			old: "ix-v01.txt", oldHost: "ix-v01", newHost: "rt-v01", pack: "cisco_ios",
			ifs: map[string]string{"GigaEthernet2:port1": "FastEthernet0", "GigaEthernet2:port2": "FastEthernet1", "GigaEthernet2:port3": "FastEthernet2", "GigaEthernet2:port4": "FastEthernet3",
				"GigaEthernet0.0": "GigabitEthernet8", "GigaEthernet1.0": "GigabitEthernet9", "GigaEthernet1.1": "GigabitEthernet9.30",
				"GigaEthernet2:1.0": "Vlan1", "GigaEthernet2:2.0": "Vlan2"},
			first: 1,
			notes: []string{"VLAN 2 を使っているのに「VLAN」の表に行が無いので、定義（vlan 2）を足して書きます"},
		},
	})
}
