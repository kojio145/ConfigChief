package check_test

import (
	"testing"

	"github.com/kojio145/configchief/internal/check"
)

// The checks of appendix A 3.26: V-283 to V-289.
func TestValuesBasics(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	// Nothing to tell: plain secrets, a user the device keeps the password
	// of, a tunnel of EtherIP in a bridge group with the interface it extends.
	put(t, sc, ds, "users", "device", "rt-a01", "name", "sample-user", "secret", "dummy-chap-password", "secret_form", "plain", "reversible", "true")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "Tunnel0", "enabled", "true", "description", "sample etherip", "bridge_group", "1")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/2", "enabled", "true", "description", "sample extended", "bridge_group", "1")
	put(t, sc, ds, "tunnels", "device", "rt-a01", "if_name", "Tunnel0", "type", "etherip", "source", "192.0.2.1", "destination", "192.0.2.2")
	// V-283: a secret in the form a device saved.
	put(t, sc, ds, "ospf_processes", "device", "rt-a01", "process_id", "1")
	put(t, sc, ds, "ospf_interfaces", "device", "rt-a01", "process_id", "1", "if_name", "GigabitEthernet0/0", "area", "0", "auth_type", "simple", "auth_key", "7 dummy7sample4C5D")
	// V-284: a hash that has to be turned back; the encrypted form on a
	// user that is not kept that way.
	put(t, sc, ds, "users", "device", "rt-a01", "name", "example-user", "secret", "dummy0hash0sample0user", "secret_form", "hashed", "reversible", "true")
	put(t, sc, ds, "users", "device", "rt-a01", "name", "operator", "secret", "dummy7sample8A9B", "secret_form", "encrypted")
	// V-285: two limits; a mode without a limit; WRED with a queue limit.
	put(t, sc, ds, "qos_classes", "device", "rt-a01", "name", "cm-sample-a", "match", "any")
	put(t, sc, ds, "qos_policies", "device", "rt-a01", "policy", "pm-sample", "class", "cm-sample-a", "seq", "10",
		"shape_bps", "1000000", "shape_percent", "30", "police_violate", "drop", "queue_limit", "64", "random_detect", "on")
	put(t, sc, ds, "qos_policies", "device", "rt-a01", "policy", "pm-sample", "class", "default", "seq", "20", "shape_mode", "peak", "queue_limit_unit", "bytes")
	// V-286: two conditions; "not" with nothing; the class itself. A list
	// with a CoS is one condition.
	put(t, sc, ds, "acls", "device", "rt-a01", "name", "ACL-SAMPLE-Q", "type", "extended")
	put(t, sc, ds, "qos_class_matches", "device", "rt-a01", "class", "cm-sample-a", "seq", "10", "acl", "ACL-SAMPLE-Q", "cos", "5")
	put(t, sc, ds, "qos_class_matches", "device", "rt-a01", "class", "cm-sample-a", "seq", "20", "cos", "1", "qos_group", "5")
	put(t, sc, ds, "qos_class_matches", "device", "rt-a01", "class", "cm-sample-a", "seq", "30", "negate", "true")
	put(t, sc, ds, "qos_class_matches", "device", "rt-a01", "class", "cm-sample-a", "seq", "40", "class_ref", "cm-sample-a")
	// V-287: a tunnel of EtherIP outside a bridge group; alone in its group.
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "Tunnel0", "enabled", "true", "description", "sample etherip")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "Tunnel1", "enabled", "true", "description", "sample etherip", "bridge_group", "7")
	put(t, sc, ds, "tunnels", "device", "rt-a02", "if_name", "Tunnel0", "type", "etherip", "source", "192.0.2.2", "destination", "192.0.2.1")
	put(t, sc, ds, "tunnels", "device", "rt-a02", "if_name", "Tunnel1", "type", "etherip", "source", "192.0.2.2", "destination", "203.0.113.1")
	// V-288: both ways without a method; an idle time on a line that stays up.
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "Dialer1", "enabled", "true", "description", "sample pppoe", "ipv4", "negotiated")
	put(t, sc, ds, "pppoe", "device", "rt-a01", "name", "Dialer1", "physical_if", "GigabitEthernet0/2", "logical_if", "Dialer1",
		"username", "example-user@example.net", "password", "dummy-ppp-password", "auth_both", "true", "idle_timeout", "300")
	// V-289: seconds without minutes; options without "datetime"; a size
	// without the buffer.
	put(t, sc, ds, "devices", "hostname", "rt-a03", "vendor", "cisco_ios", "console_timeout_sec", "30", "log_timestamp", "uptime", "log_timestamp_options", "msec", "log_buffer_size", "8192")
	put(t, sc, ds, "remote_access", "device", "rt-a03", "protocols", "ssh", "idle_timeout_sec", "30")

	fs, tally := run(sc, ds, nil)
	same(t, told(fs, "V-28"), []string{
		"V-283 rt-a01 auth_key",
		"V-284 rt-a01 reversible",  // example-user
		"V-284 rt-a01 secret_form", // operator
		"V-285 rt-a01 shape_percent",
		"V-285 rt-a01 police_violate",
		"V-285 rt-a01 random_detect",
		"V-285 rt-a01 shape_mode",       // default
		"V-285 rt-a01 queue_limit_unit", // default
		"V-286 rt-a01 qos_group",        // 20
		"V-286 rt-a01 negate",           // 30
		"V-286 rt-a01 class_ref",        // 40
		"V-287 rt-a02 type",             // Tunnel0
		"V-287 rt-a02 type",             // Tunnel1
		"V-288 rt-a01 auth_both",
		"V-288 rt-a01 idle_timeout",
		"V-289 rt-a03 console_timeout_sec",
		"V-289 rt-a03 log_timestamp_options",
		"V-289 rt-a03 log_buffer_size",
		"V-289 rt-a03 idle_timeout_sec",
	})
	for _, id := range []string{"V-283", "V-284", "V-285", "V-286", "V-287", "V-288", "V-289"} {
		if !tally.Ran(id) {
			t.Errorf("%s did not run", id)
		}
	}
}

// X-407: the two ends of a tunnel of EtherIP are the same kind of tunnel.
func TestCrossEtherIP(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	// rt-a01 and rt-a02 point at each other; one end is GRE.
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "Tunnel0", "enabled", "true", "bridge_group", "1")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "Tunnel0", "enabled", "true")
	put(t, sc, ds, "tunnels", "device", "rt-a01", "if_name", "Tunnel0", "type", "etherip", "source", "192.0.2.1", "destination", "192.0.2.2")
	put(t, sc, ds, "tunnels", "device", "rt-a02", "if_name", "Tunnel0", "type", "gre", "source", "192.0.2.2", "destination", "192.0.2.1")
	tally := check.NewTally()
	fs := check.Cross(check.Input{Schema: sc, Data: ds}, tally)
	same(t, told(fs, "X-407"), []string{"X-407 rt-a01 type"})
	if !tally.Ran("X-407") {
		t.Error("X-407 did not run")
	}
}
