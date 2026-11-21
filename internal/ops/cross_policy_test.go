package ops

import "testing"

// A site router with route policies (route maps for the routes of BGP in and
// out, for a redistribution, and for policy routing) goes from one vendor to
// the other, both ways (appendix A 3.19). What the new pack writes in
// another way is told (G-614); what a person has to write for the new vendor
// is listed here.
func TestCrossVendorRoutePolicy(t *testing.T) {
	crossSites(t, "policy", []site{
		{
			old: "rt-r21.txt", oldHost: "rt-r21", newHost: "ix-r21", pack: "nec_ix",
			ifs: map[string]string{"GigabitEthernet0/0": "GigaEthernet0.0", "GigabitEthernet0/1": "GigaEthernet1.0", "Loopback0": "Loopback0.0"},
			// The new vendor has no weight, and no route that stays when its
			// interface goes down: a person decides what to do with them.
			unused: []string{"付ける値: weight", "IF が落ちても残す"},
			first:  3,
			mend: [][5]string{
				{"経路制御の方針", "方針名", "RM-R-IN", "付ける値: weight", ""},
				{"スタティック経路", "宛先", "192.0.2.128/26", "IF が落ちても残す", ""},
			},
			// "Send the communities" has no line there. The community list
			// of one line needs no telling: the import of the old side has
			// turned it into the values of the entry, as the new side
			// writes them.
			notes: []string{"「コミュニティを送る」に当たる行は NEC IX にありません"},
		},
		{
			old: "ix-r21.txt", oldHost: "ix-r21", newHost: "rt-r21", pack: "cisco_ios",
			ifs: map[string]string{"GigaEthernet0.0": "GigabitEthernet0/0", "GigaEthernet1.0": "GigabitEthernet0/1", "Loopback0.0": "Loopback0"},
			// What a metric with a sign means on the new vendor is not in its
			// reference; and it sends no communities unless told to.
			stops: []string{
				"メトリック「+10」は書けません",
				"「コミュニティを送る」が有効でないので、相手には届きません",
			},
			first: 2,
			mend: [][5]string{
				{"経路制御の方針", "方針名", "rm-r-out", "付ける値: メトリック（MED）", "10"},
				{"BGP ネイバー", "ネイバーアドレス", "203.0.113.2", "コミュニティを送る", "true"},
				// The metric of a static route has no line on the new vendor.
				{"スタティック経路", "宛先", "192.0.2.64/26", "メトリック", ""},
			},
		},
	})
}
