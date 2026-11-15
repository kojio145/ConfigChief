package ops

import "testing"

// A site router with QoS (a shaped line, a priority class, shares of the
// bandwidth, marking) goes from one vendor to the other, both ways (the end
// of stage M6, spec 18). What the new pack writes in another way is told
// (G-614); what a person has to write for the new vendor is listed here.
func TestCrossVendorQoS(t *testing.T) {
	crossSites(t, "qos", []site{
		{
			old: "rt-q01.txt", oldHost: "rt-q01", newHost: "ix-q01", pack: "nec_ix",
			ifs: map[string]string{"GigabitEthernet0/0": "GigaEthernet0.0", "GigabitEthernet0/1": "GigaEthernet1.0"},
			// The new vendor has neither fair queueing nor policing: a
			// person decides what to do with them.
			unused: []string{"公平に送る", "上限・捨てる"},
			first:  3,
			mend: [][5]string{
				{"QoS のポリシー", "クラス名", "default", "公平に送る", ""},
				{"QoS のポリシー", "ポリシー名", "pm-q-lan", "上限・捨てる（bps）", ""},
				{"QoS のポリシー", "ポリシー名", "pm-q-lan", "超えたとき", ""},
			},
			notes: []string{"「帯域（残りの %）」は NEC IX に無いので"},
		},
		{
			old: "ix-q01.txt", oldHost: "ix-q01", newHost: "rt-q01", pack: "cisco_ios",
			ifs:   map[string]string{"GigaEthernet0.0": "GigabitEthernet0/0", "GigaEthernet1.0": "GigabitEthernet0/1"},
			first: 1,
			notes: []string{"キューごとのクラス（cm-q-pq-high など）に分けて書きます"},
		},
		{
			// The forms before MQC: a priority list, a custom queue list,
			// rate limits and a shaped interface.
			old: "rt-q02.txt", oldHost: "rt-q02", newHost: "ix-q02", pack: "nec_ix",
			ifs: map[string]string{"Serial0/0": "GigaEthernet0.0", "Serial0/1": "GigaEthernet1.0", "GigabitEthernet0/1": "GigaEthernet2.0"},
			// Policing is not of the new vendor: a person decides. The
			// shares of the custom queues come to 100 %, and the device
			// keeps 10 % for each of its own two classes: told, not changed.
			unused: []string{"上限・捨てる"},
			first:  3,
			mend: [][5]string{
				{"QoS のポリシー", "クラス名", "rl-acl101", "上限・捨てる（bps）", ""},
				{"QoS のポリシー", "クラス名", "rl-acl101", "捨てる側の通常バースト（バイト）", ""},
				{"QoS のポリシー", "クラス名", "rl-acl101", "捨てる側の超過バースト（バイト）", ""},
				{"QoS のポリシー", "クラス名", "rl-acl101", "超えたとき", ""},
				{"QoS のポリシー", "クラス名", "default", "上限・捨てる（bps）", ""},
				{"QoS のポリシー", "クラス名", "default", "捨てる側の通常バースト（バイト）", ""},
				{"QoS のポリシー", "クラス名", "default", "捨てる側の超過バースト（バイト）", ""},
				{"QoS のポリシー", "クラス名", "default", "超えたとき", ""},
				{"QoS のポリシー", "クラス名", "default", "超えたときに付ける値", ""},
			},
			notes: []string{"帯域（%）の合計が 120% で"},
		},
	})
}
