package judge

import (
	"strings"
	"testing"
)

func TestCarry(t *testing.T) {
	words := map[string]bool{}
	for _, w := range strings.Fields("ip sla access list group in out interface gigabitethernet vrf forwarding snmp server host version username privilege " +
		"crypto isakmp address route map permit match acl neighbor remote as service timestamps debug datetime msec enable cisco tunnel") {
		words[w] = true
	}
	known := func(w string) bool { return words[w] }
	for _, c := range []struct{ in, want, hidden string }{
		{"ip sla 10", "ip sla <n>", ""},
		{" ip access-group ACL-HONSHA-IN in", "ip access-group <name> in", "ACL-HONSHA-IN"},
		{"interface GigabitEthernet0/0/1", "interface GigabitEthernet<n>", ""},
		{"vrf forwarding tokyo", "vrf forwarding <name>", "tokyo"},
		{"service timestamps debug datetime msec", "service timestamps debug datetime msec", ""},
		// A secret stays hidden even when it is spelled like a known word.
		{"enable password cisco", "enable password <secret>", ""},
		{"crypto isakmp key cisco address 192.0.2.1", "crypto isakmp key <secret>", ""},
		{"snmp-server community sample-ro RO 10", "snmp-server community <secret>", ""},
		{"neighbor 192.0.2.1 password 7 0822455D0A16", "neighbor <ip> password <secret>", ""},
		// The word before the secret is not known: the secret is hidden all the same.
		{"tunnel pre-shared-key cisco", "tunnel <name> <secret>", "pre-shared-key"},
		// What a person wrote.
		{" description in service route", "description <text>", ""},
		{"description", "description", ""},
		{"access-list 10 remark permit in", "access-list <n> remark <text>", ""},
		// A word of another script, a word in quotes, a name with digits.
		{"username 管理者 privilege 15", "username <name> privilege <n>", "管理者"},
		{`snmp-server contact "to the desk"`, "snmp-server contact <text>", ""},
		{"route-map RM-OUT-64496 permit 10", "route-map <name> permit <n>", "RM-OUT-<n>"},
		{"neighbor 2001:db8::1 remote-as 64496", "neighbor <ip6> remote-as <n>", ""},
		{"snmp-server host 192.0.2.9 version 2c honsha", "snmp-server host <ip> version <name> <name>", "<n>c,honsha"},
	} {
		got, hidden := Carry(c.in, known)
		if got != c.want || strings.Join(hidden, ",") != c.hidden {
			t.Errorf("Carry(%q) = %q hidden %v, want %q hidden %q", c.in, got, hidden, c.want, c.hidden)
		}
		// A shape goes through unchanged but for its names.
		if again, _ := CarryShape(Shape(c.in), known); again != got {
			t.Errorf("CarryShape(Shape(%q)) = %q, want %q", c.in, again, got)
		}
	}
	// Nothing known: only the marks and the words that open a secret stay.
	got, _ := Carry("ip address 192.0.2.1 255.255.255.0 secondary", func(string) bool { return false })
	if got != "<name> <name> <ip> <ip> <name>" {
		t.Errorf("nothing known: %q", got)
	}
}
