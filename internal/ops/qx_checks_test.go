package ops

import (
	"strings"
	"testing"
)

// The checks of VLANs, switch ports and LAGs (V-25x, X-38x) see the data of
// a QX-type switch like any other: its trunk, its LAG and its VLANs are
// compared with those of the switch of the other vendor at the far end.
func TestQXChecks(t *testing.T) {
	e := newCase(t, map[string]string{
		"input/old/sw-q21.txt": siteFile(t, "qx", "sw-q21.txt"),
		"input/old/sw-c21.txt": siteFile(t, "qx", "sw-c21.txt"),
	})
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	for _, d := range [][2]string{{"sw-q21", "nec_qx"}, {"sw-c21", "cisco_ios"}} {
		setCell(t, e, "新旧対照表（機器）", "旧ホスト名", d[0], "新ホスト名", d[0])
		setCell(t, e, "新旧対照表（機器）", "旧ホスト名", d[0], "新メーカー", d[1])
	}
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	// The interfaces keep their names.
	for _, n := range []string{"Bridge-Aggregation1", "GigabitEthernet1/0/1", "GigabitEthernet1/0/21", "GigabitEthernet1/0/22", "GigabitEthernet1/0/24",
		"Port-channel1", "GigabitEthernet1/0/45", "GigabitEthernet1/0/48"} {
		setCell(t, e, "新旧対照表（ポート）", "旧 IF 名", n, "新 IF 名", n)
	}
	b, err := e.MakeBook("old", false, at)
	if err != nil {
		t.Fatal(err)
	}
	if a := b.Applied; len(a.Undecided)+len(a.Dropped)+len(a.Cleared)+len(a.Clashes)+len(b.Problems) != 0 {
		t.Fatalf("the mapping: %+v, problems %v", a, b.Problems)
	}
	// The two cables: a single trunk and a member of the LAG.
	setCell(t, e, "インタフェース", "IF 名", "GigabitEthernet1/0/24", "対向機器", "sw-c21")
	setCell(t, e, "インタフェース", "IF 名", "GigabitEthernet1/0/24", "対向 IF", "GigabitEthernet1/0/48")
	setCell(t, e, "インタフェース", "IF 名", "GigabitEthernet1/0/21", "対向機器", "sw-c21")
	setCell(t, e, "インタフェース", "IF 名", "GigabitEthernet1/0/21", "対向 IF", "GigabitEthernet1/0/45")
	o, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	told := strings.Join(unsettled(o), "\n")
	for _, want := range []string{
		"V-251 switchports if_name=GigabitEthernet1/0/1: ",  // VLAN 40 has no row
		"V-252 switchports if_name=GigabitEthernet1/0/24: ", // the native VLAN 99 is not allowed
		"V-254 switchports if_name=GigabitEthernet1/0/22: ", // the members of the LAG differ
		"X-382 switchports if_name=GigabitEthernet1/0/24: ", // the allowed VLANs differ at the two ends
		"X-383 switchports if_name=GigabitEthernet1/0/24: ", // native VLAN 99 here, 1 there
		"X-385 lags if_name=GigabitEthernet1/0/21: ",        // static here, LACP there
		"X-386 vlans vlan_id=20: ",                          // VLAN 20 has two names
		"X-387 interfaces name=GigabitEthernet1/0/24: ",     // speed 1000 and 100
	} {
		if !strings.Contains(told, want) {
			t.Errorf("not told: %q:\n%s", want, told)
		}
	}
	t.Logf("\n%s", told)
}
