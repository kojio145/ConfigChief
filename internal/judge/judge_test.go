package judge

import (
	"reflect"
	"testing"

	"github.com/kojio145/configchief/internal/project"
)

func TestShape(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"ip sla 10", "ip sla <n>"},
		{"  ip   sla\t20 ", "ip sla <n>"},
		{`description "to core 1"`, "description <str>"},
		{`banner "one" and "two"`, "banner <str> and <str>"},
		{"ip helper-address 192.0.2.1", "ip helper-address <ip>"},
		{"ip route 198.51.100.0 255.255.255.0 192.0.2.254", "ip route <ip> <ip> <ip>"},
		{"network 203.0.113.0/24", "network <ip>/<n>"},
		{"ipv6 route 2001:db8::/32 Null0", "ipv6 route <ip6>/<n> Null<n>"},
		{"ipv6 address 2001:db8:0:1::1/64", "ipv6 address <ip6>/<n>"},
		{"ipv6 neighbor fe80::1 0000.5e00.5301", "ipv6 neighbor <ip6> <mac>"},
		{"mac 00:00:5e:00:53:01 and 00-00-5e-00-53-02", "mac <mac> and <mac>"},
		{"interface GigabitEthernet1/0/1", "interface GigabitEthernet<n>"},
		{"interface GigabitEthernet0/1.30", "interface GigabitEthernet<n>.<n>"},
		{"interface Vlan10", "interface Vlan<n>"},
		{"spanning-tree vlan 10,20,30-40 priority 4096", "spanning-tree vlan <n> priority <n>"},
		{"snmp-server enable traps oid 1.3.6.1.4.1", "snmp-server enable traps oid <n>.<n>.<n>.<n>.<n>.<n>"},
		{"clock set 12:30:00", "clock set <n>:<n>:<n>"},
		{"stamp 7 00AA11BB22CC", "stamp <n> <hex>"},
		{"stamp 7 deadbeef", "stamp <n> deadbeef"},
		{"stamp 7 0123456789", "stamp <n> <n>"},
		{"address-family ipv4", "address-family ipv4"},
		{"address-family ipv6 unicast", "address-family ipv6 unicast"},
		{"encapsulation dot1q 30", "encapsulation dot1q <n>"},
		{"hash sha256 group 14", "hash sha256 group <n>"},
		{"version 1.2.3.999", "version <n>.<n>.<n>.<n>"},
		{"route-map RM-SAMPLE permit 10", "route-map RM-SAMPLE permit <n>"},
		{"Route-map rm-sample permit 10", "Route-map rm-sample permit <n>"},
		{"", ""},
	} {
		if got := Shape(c.in); got != c.want {
			t.Errorf("Shape(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestIDs(t *testing.T) {
	a := Key{Source: "old", Pack: "p", Shape: "ip sla <n>"}
	b := Key{Source: "old", Pack: "p", Path: []string{"interface Vlan<n>"}, Shape: "ip sla <n>"}
	c := Key{Source: "new", Pack: "p", Shape: "ip sla <n>"}
	ids := IDs([]Key{a, b, c})
	if len(ids) != 3 || len(ids[0]) != 8 || ids[0] == ids[1] || ids[0] == ids[2] || ids[1] == ids[2] {
		t.Errorf("ids = %v", ids)
	}
	if again := IDs([]Key{c, a}); again[1] != ids[0] || again[0] != ids[2] {
		t.Errorf("the ID of a group changed: %v, %v", ids, again)
	}
	if len(IDs(nil)) != 0 {
		t.Error("no key, no ID")
	}
	// The path and the shape are apart: the same words cut at another place
	// are another group.
	if x, y := (Key{Path: []string{"a"}, Shape: "b"}), (Key{Shape: "a\x01b"}); x.Hash() == y.Hash() {
		t.Error("the hash does not tell the path from the shape")
	}
}

func j(devices, verdict string) project.Judgment {
	return project.Judgment{Pack: "p", Shape: "s", Devices: devices, Verdict: verdict, By: "admin"}
}

func TestFind(t *testing.T) {
	k := Key{Source: "old", Pack: "p", Shape: "s"}
	other := project.Judgment{Source: "new", Pack: "p", Shape: "s", Devices: "*", Verdict: "hold", By: "admin"}
	list := []project.Judgment{
		j("*", "hold"),
		j("rt-a01, rt-a02", "manual"),
		other,
		j("*", "not_needed"),
		j("rt-a02", "hold"),
	}
	for _, c := range []struct {
		device string
		at     int
		own    bool
	}{
		{"rt-a01", 1, true},
		{"rt-a02", 4, true},
		{"rt-a03", 3, false},
		{"", 3, false},
	} {
		if at, own := Find(list, k, c.device); at != c.at || own != c.own {
			t.Errorf("Find(%q) = %d, %v; want %d, %v", c.device, at, own, c.at, c.own)
		}
	}
	if at, _ := Find(list, Key{Source: "old", Pack: "p", Path: []string{"x"}, Shape: "s"}, "rt-a01"); at != -1 {
		t.Errorf("another path is another group: %d", at)
	}
	if at, _ := Find(list, Key{Source: "new", Pack: "p", Shape: "s"}, "rt-a01"); at != 2 {
		t.Errorf("the judgment of the new side: %d", at)
	}
}

func verdicts(list []project.Judgment) []string {
	var out []string
	for _, x := range list {
		out = append(out, x.Devices+"="+x.Verdict)
	}
	return out
}

func TestSetAndClear(t *testing.T) {
	k := Key{Source: "old", Pack: "p", Shape: "s"}
	list := []project.Judgment{j("*", "hold"), j("rt-a01,rt-a02", "manual")}

	// For every device: the one for every device goes, the named ones stay.
	got := Set(list, k, nil, project.Judgment{Verdict: "not_needed", Reason: "r", By: "admin"})
	if want := []string{"rt-a01,rt-a02=manual", "*=not_needed"}; !reflect.DeepEqual(verdicts(got), want) {
		t.Errorf("set for all = %v, want %v", verdicts(got), want)
	}
	if got[1].Source != "old" || got[1].Pack != "p" || got[1].Shape != "s" {
		t.Errorf("the key is not in the judgment: %+v", got[1])
	}

	// For a device: it leaves the judgment that named it.
	got = Set(list, k, []string{"rt-a02", "rt-a03"}, project.Judgment{Verdict: "hold", By: "admin"})
	if want := []string{"*=hold", "rt-a01=manual", "rt-a02,rt-a03=hold"}; !reflect.DeepEqual(verdicts(got), want) {
		t.Errorf("set for devices = %v, want %v", verdicts(got), want)
	}
	// A judgment with no device left goes.
	got = Set(list, k, []string{"rt-a01", "rt-a02"}, project.Judgment{Verdict: "hold", By: "admin"})
	if want := []string{"*=hold", "rt-a01,rt-a02=hold"}; !reflect.DeepEqual(verdicts(got), want) {
		t.Errorf("set for all the named devices = %v, want %v", verdicts(got), want)
	}

	got, missing := Clear(list, k, nil)
	if want := []string{"rt-a01,rt-a02=manual"}; !reflect.DeepEqual(verdicts(got), want) || missing != nil {
		t.Errorf("clear for all = %v, %v", verdicts(got), missing)
	}
	if _, missing = Clear(got, k, nil); !reflect.DeepEqual(missing, []string{"*"}) {
		t.Errorf("nothing for all to clear: %v", missing)
	}
	got, missing = Clear(list, k, []string{"rt-a01", "rt-a09"})
	if want := []string{"*=hold", "rt-a02=manual"}; !reflect.DeepEqual(verdicts(got), want) || !reflect.DeepEqual(missing, []string{"rt-a09"}) {
		t.Errorf("clear for devices = %v, %v", verdicts(got), missing)
	}
	// The list given is not changed.
	if want := []string{"*=hold", "rt-a01,rt-a02=manual"}; !reflect.DeepEqual(verdicts(list), want) {
		t.Errorf("the list was changed: %v", verdicts(list))
	}
}
