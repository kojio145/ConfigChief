package schemas

import (
	"strings"
	"testing"
)

// The bundled definitions load, in the order of appendix A 2.
func TestLoad(t *testing.T) {
	s, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tb := range s.Tables {
		names = append(names, tb.Name)
	}
	want := "devices interfaces static_routes users remote_access ntp_servers syslog_servers snmp_communities snmp_hosts " +
		"vrfs ospf_processes ospf_interfaces ospf_areas ospf_ranges ospf_redistribute " +
		"bgp_processes bgp_neighbors bgp_networks bgp_redistribute fhrp acls acl_entries prefix_lists nat_rules tracks " +
		"pppoe ipsec_tunnels ipsec_selectors tunnels dhcp_pools connections " +
		"qos_classes qos_class_matches qos_policies qos_bindings"
	if got := strings.Join(names, " "); got != want {
		t.Errorf("tables:\n got %s\nwant %s", got, want)
	}
	for _, tb := range s.Tables {
		dev := tb.DeviceColumn()
		if dev == "" && tb.Name != "connections" {
			t.Errorf("%s: no device column", tb.Name)
		}
		if dev != "" && tb.Key[0] != dev {
			t.Errorf("%s: the key does not start with the device", tb.Name)
		}
		for _, c := range tb.Columns {
			if c.Note.JA != "" && c.Note.EN == "" {
				t.Errorf("%s.%s: note without en", tb.Name, c.Name)
			}
		}
	}
	if c := s.Table("ipsec_tunnels").Column("esp_encryption"); c == nil || len(c.Values) != 8 || c.Values[7] != "null" {
		t.Errorf("esp_encryption = %+v", c)
	}
	if c := s.Table("users").Column("secret"); c == nil || !c.Secret {
		t.Error("users.secret is not secret")
	}
}
