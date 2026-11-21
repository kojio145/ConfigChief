package ops

import "testing"

// A site router with the rest of BGP and OSPF (peer groups, timers,
// distances, aggregates, a virtual link, what goes into the routing table,
// the communities a policy takes off) goes from one vendor to the other,
// both ways (appendix A 3.19, the part added on 2026-11-21). What the new
// pack writes in another way is told (G-614); what a person has to write
// for the new vendor is listed here.
func TestCrossVendorRoutingRest(t *testing.T) {
	crossSites(t, "routing2", []site{
		{
			old: "rt-u01.txt", oldHost: "rt-u01", newHost: "ix-u01", pack: "nec_ix",
			ifs: map[string]string{"GigabitEthernet0/0": "GigaEthernet0.0", "GigabitEthernet0/1": "GigaEthernet1.0", "Loopback0": "Loopback0.0"},
			// The neighbors of the new vendor have no limit of prefixes: a
			// person leaves it out. The peer group needs nothing (the old
			// side was read neighbor by neighbor), and one distance for
			// every OSPF route is written kind by kind without a word.
			unused: []string{"受け取る経路数の上限、上限の警告を出す割合（%）"},
			first:  1,
			mend: [][5]string{
				{"BGP ネイバー", "ネイバーアドレス", "203.0.113.2", "受け取る経路数の上限", ""},
				{"BGP ネイバー", "ネイバーアドレス", "203.0.113.2", "上限の警告を出す割合（%）", ""},
			},
		},
		{
			old: "ix-u01.txt", oldHost: "ix-u01", newHost: "rt-u01", pack: "cisco_ios",
			ifs: map[string]string{"GigaEthernet0.0": "GigabitEthernet0/0", "GigaEthernet1.0": "GigabitEthernet0/1", "Loopback0.0": "Loopback0"},
		},
	})
}
