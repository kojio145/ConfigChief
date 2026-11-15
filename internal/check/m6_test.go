package check_test

import (
	"sort"
	"testing"

	"github.com/kojio145/configchief/internal/check"
)

// The checks of the values of the QoS tables (stage M6): a policy names a
// class that is not there (V-106; "default" and "local" are always there),
// and a class, a policy or a list that nothing uses (V-210). A list a
// condition names is used.
func TestValuesM6(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	put(t, sc, ds, "acls", "device", "rt-a01", "name", "ACL-SAMPLE-QOS", "type", "extended")
	put(t, sc, ds, "acl_entries", "device", "rt-a01", "acl", "ACL-SAMPLE-QOS", "seq", "10", "action", "permit", "protocol", "ip", "src", "any", "dst", "any")
	put(t, sc, ds, "qos_classes", "device", "rt-a01", "name", "cm-sample-used", "match", "any")
	put(t, sc, ds, "qos_classes", "device", "rt-a01", "name", "cm-sample-idle", "match", "any")
	put(t, sc, ds, "qos_class_matches", "device", "rt-a01", "class", "cm-sample-used", "seq", "10", "acl", "ACL-SAMPLE-QOS")
	put(t, sc, ds, "qos_class_matches", "device", "rt-a01", "class", "cm-sample-idle", "seq", "10", "dscp", "46")
	put(t, sc, ds, "qos_policies", "device", "rt-a01", "policy", "pm-sample-out", "class", "cm-sample-used", "seq", "10", "priority", "true")
	put(t, sc, ds, "qos_policies", "device", "rt-a01", "policy", "pm-sample-out", "class", "default", "seq", "20", "fair_queue", "true")
	put(t, sc, ds, "qos_policies", "device", "rt-a01", "policy", "pm-sample-out", "class", "cm-sample-none", "seq", "30", "set_dscp", "0")
	put(t, sc, ds, "qos_policies", "device", "rt-a01", "policy", "pm-sample-idle", "class", "local", "seq", "10", "set_dscp", "48")
	put(t, sc, ds, "qos_bindings", "device", "rt-a01", "if_name", "GigabitEthernet0/1", "direction", "output", "policy", "pm-sample-out")
	put(t, sc, ds, "qos_bindings", "device", "rt-a01", "if_name", "GigabitEthernet0/1", "direction", "input", "policy", "pm-sample-gone")

	fs := check.Values(check.Input{Schema: sc, Data: ds}, check.NewTally())
	var got []string
	for _, f := range fs {
		if (f.RuleID == "V-106" || f.RuleID == "V-210") && (f.Ref.Table == "acls" || len(f.Ref.Table) > 4 && f.Ref.Table[:4] == "qos_") {
			got = append(got, f.RuleID+" "+f.Ref.Table+" "+f.Ref.Key)
		}
	}
	sort.Strings(got)
	same(t, got, []string{
		"V-106 qos_bindings if_name=GigabitEthernet0/1,direction=input",
		"V-106 qos_policies policy=pm-sample-out,class=cm-sample-none",
		"V-210 qos_classes name=cm-sample-idle",
		"V-210 qos_policies policy=pm-sample-idle,class=local",
	})
}
