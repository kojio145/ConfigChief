package check_test

import (
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
	"github.com/kojio145/configchief/internal/schemas"
)

func defs(t *testing.T) *schema.Set {
	t.Helper()
	sc, err := schemas.Load()
	if err != nil {
		t.Fatal(err)
	}
	return sc
}

// put adds a row; cells are name, value, name, value … and the values of a
// list are written with "|" between.
func put(t *testing.T, sc *schema.Set, ds *model.Dataset, table string, cells ...string) *model.Row {
	t.Helper()
	tab := sc.Table(table)
	if tab == nil {
		t.Fatalf("no table %s", table)
	}
	n := 2
	if old := ds.Tables[table]; old != nil {
		n = len(old.Rows) + 2
	}
	row := &model.Row{Table: table, Cells: map[string]*model.Cell{}}
	row.Origins = []model.Origin{{Kind: "book", File: "book.xlsx", Sheet: tab.Title.JA, Addr: "A" + string(rune('0'+n))}}
	for i := 0; i+1 < len(cells); i += 2 {
		c := tab.Column(cells[i])
		if c == nil {
			t.Fatalf("no column %s.%s", table, cells[i])
		}
		row.Cells[c.Name] = &model.Cell{Values: strings.Split(cells[i+1], "|"), List: c.List}
	}
	for _, k := range tab.Key {
		row.Key = append(row.Key, model.KV{Name: k, Value: row.Get(k)})
	}
	if !ds.Adopt(row) {
		t.Fatalf("row twice: %s %s", table, row.KeyText())
	}
	return row
}

// good is the data of two routers with nothing to tell about.
func good(t *testing.T, sc *schema.Set) *model.Dataset {
	ds := model.NewDataset("book")
	put(t, sc, ds, "devices", "hostname", "rt-a01", "vendor", "cisco_ios", "enable_secret", "sample-enable-1")
	put(t, sc, ds, "devices", "hostname", "rt-a02", "vendor", "cisco_ios")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/0", "enabled", "true", "description", "sample uplink", "ipv4", "192.0.2.1/30")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/1", "enabled", "true", "description", "sample lan", "ipv4", "198.51.100.1/24", "mtu", "1500", "tcp_mss", "1460")
	put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "Loopback0", "enabled", "true", "description", "sample id", "ipv4", "203.0.113.1/32")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/0", "enabled", "true", "description", "sample uplink", "ipv4", "192.0.2.2/30")
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/1", "enabled", "false")
	put(t, sc, ds, "static_routes", "device", "rt-a01", "prefix", "0.0.0.0/0", "next_hop", "192.0.2.2")
	put(t, sc, ds, "static_routes", "device", "rt-a01", "prefix", "203.0.113.128/25", "next_hop", "198.51.100.254")
	put(t, sc, ds, "ntp_servers", "device", "rt-a01", "server", "192.0.2.123", "source_if", "Loopback0")
	return ds
}

func run(sc *schema.Set, ds, bk *model.Dataset) ([]check.Finding, *check.Tally) {
	tally := check.NewTally()
	fs := check.Values(check.Input{Schema: sc, Data: ds, Book: bk, HasPack: func(id string) bool { return id == "cisco_ios" }}, tally)
	return fs, tally
}

func ids(fs []check.Finding) string {
	var out []string
	for _, f := range fs {
		out = append(out, f.RuleID+"/"+f.Device+":"+f.Ref.Table+"."+f.Ref.Column)
	}
	return strings.Join(out, " ")
}

func TestValuesClean(t *testing.T) {
	sc := defs(t)
	fs, tally := run(sc, good(t, sc), nil)
	if len(fs) != 0 {
		for _, f := range fs {
			t.Errorf("%s %s %+v: %s", f.RuleID, f.Device, f.Ref, f.Message)
		}
	}
	for _, id := range []string{"V-101", "V-102", "V-103", "V-104", "V-106", "V-109", "V-110", "V-201", "V-202", "V-203", "V-204", "V-206", "V-208", "V-212", "V-218"} {
		if !tally.Ran(id) {
			t.Errorf("%s did not run", id)
		}
	}
	// What cannot be checked is told, not passed.
	for _, id := range []string{"V-111", "V-112", "V-114", "V-209"} {
		if tally.Ran(id) || tally.Skipped(id)[""] == "" {
			t.Errorf("%s: ran %v, skipped %q", id, tally.Ran(id), tally.Skipped(id)[""])
		}
	}
	if n := tally.Looked("V-206")["rt-a01"]; n != 1 {
		t.Errorf("V-206 looked at rt-a01 %d times", n)
	}
}

func TestValuesFindings(t *testing.T) {
	sc := defs(t)
	cases := []struct {
		name string
		add  func(ds *model.Dataset)
		want string
	}{
		{"required", func(ds *model.Dataset) {
			put(t, sc, ds, "ntp_servers", "device", "rt-a02")
		}, "V-101/rt-a02:ntp_servers.server"},
		{"explicit is left to the pack", func(ds *model.Dataset) {
			put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "Loopback0")
		}, ""},
		{"route without a way out", func(ds *model.Dataset) {
			put(t, sc, ds, "static_routes", "device", "rt-a02", "prefix", "198.51.100.0/24")
		}, "V-101/rt-a02:static_routes."},
		{"form", func(ds *model.Dataset) {
			put(t, sc, ds, "ntp_servers", "device", "rt-a02", "server", "192.0.2.124", "prefer", "sometimes")
		}, "V-102/rt-a02:ntp_servers.prefer"},
		{"range", func(ds *model.Dataset) {
			put(t, sc, ds, "static_routes", "device", "rt-a02", "prefix", "198.51.100.0/24", "next_hop", "192.0.2.1", "distance", "999")
		}, "V-103/rt-a02:static_routes.distance"},
		{"no number", func(ds *model.Dataset) {
			put(t, sc, ds, "static_routes", "device", "rt-a02", "prefix", "198.51.100.0/24", "next_hop", "192.0.2.1", "distance", "many")
		}, "V-102/rt-a02:static_routes.distance"},
		{"word", func(ds *model.Dataset) {
			put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "Loopback0", "enabled", "true", "description", "sample", "duplex", "sideways")
		}, "V-104/rt-a02:interfaces.duplex"},
		{"reference", func(ds *model.Dataset) {
			put(t, sc, ds, "ntp_servers", "device", "rt-a02", "server", "192.0.2.124", "source_if", "Loopback9")
		}, "V-106/rt-a02:ntp_servers.source_if"},
		{"reference to another table", func(ds *model.Dataset) {
			put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "Loopback0", "enabled", "true", "description", "sample", "acl_in", "SAMPLE-IN")
		}, "V-106/rt-a02:interfaces.acl_in"},
		{"interface of another device", func(ds *model.Dataset) {
			put(t, sc, ds, "ntp_servers", "device", "rt-a02", "server", "192.0.2.124", "source_if", "Loopback0")
		}, "V-106/rt-a02:ntp_servers.source_if"},
		{"vendor", func(ds *model.Dataset) {
			put(t, sc, ds, "devices", "hostname", "rt-a03", "vendor", "sample_os")
		}, "V-109/rt-a03:devices.vendor"},
		{"wide letters", func(ds *model.Dataset) {
			put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "Loopback0", "enabled", "true", "description", "ｓａｍｐｌｅ")
		}, "V-110/rt-a02:interfaces.description"},
		{"overlap", func(ds *model.Dataset) {
			put(t, sc, ds, "interfaces", "device", "rt-a01", "name", "GigabitEthernet0/2", "enabled", "true", "description", "sample", "ipv4", "198.51.100.129/25")
		}, "V-201/rt-a01:interfaces.ipv4"},
		{"same address", func(ds *model.Dataset) {
			put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "Loopback0", "enabled", "true", "description", "sample", "ipv4", "203.0.113.1/32")
		}, "V-202/rt-a02:interfaces.ipv4"},
		{"network address", func(ds *model.Dataset) {
			put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/2", "enabled", "true", "description", "sample", "ipv4", "203.0.113.64/26")
		}, "V-203/rt-a02:interfaces.ipv4"},
		{"broadcast address", func(ds *model.Dataset) {
			put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "GigabitEthernet0/2", "enabled", "true", "description", "sample", "ipv4", "203.0.113.127/26")
		}, "V-203/rt-a02:interfaces.ipv4"},
		{"next hop", func(ds *model.Dataset) {
			put(t, sc, ds, "static_routes", "device", "rt-a02", "prefix", "198.51.100.0/24", "next_hop", "203.0.113.200")
		}, "V-204/rt-a02:static_routes.next_hop"},
		{"next hop by another route", func(ds *model.Dataset) {
			put(t, sc, ds, "static_routes", "device", "rt-a02", "prefix", "203.0.113.0/24", "next_hop", "192.0.2.1")
			put(t, sc, ds, "static_routes", "device", "rt-a02", "prefix", "198.51.100.0/24", "next_hop", "203.0.113.200")
		}, ""},
		{"next hop of a dialer", func(ds *model.Dataset) {
			put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "Dialer1", "enabled", "true", "description", "sample", "ipv4", "negotiated")
			put(t, sc, ds, "static_routes", "device", "rt-a02", "prefix", "198.51.100.0/24", "next_hop", "203.0.113.200")
		}, ""},
		{"host name", func(ds *model.Dataset) {
			put(t, sc, ds, "devices", "hostname", "sample router", "vendor", "cisco_ios")
		}, "V-206/sample router:devices.hostname"},
		{"description", func(ds *model.Dataset) {
			put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "Loopback0", "enabled", "true")
		}, "V-208/rt-a02:interfaces.description"},
		{"mss", func(ds *model.Dataset) {
			put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "Loopback0", "enabled", "true", "description", "sample", "ip_mtu", "1454", "tcp_mss", "1420")
		}, "V-212/rt-a02:interfaces.tcp_mss"},
		{"placeholder", func(ds *model.Dataset) {
			put(t, sc, ds, "devices", "hostname", "rt-a03", "vendor", "cisco_ios", "enable_secret", "ChangeMe")
		}, "V-218/rt-a03:devices.enable_secret"},
		{"one letter over and over", func(ds *model.Dataset) {
			put(t, sc, ds, "devices", "hostname", "rt-a03", "vendor", "cisco_ios", "enable_secret", "xxxxxxxx")
		}, "V-218/rt-a03:devices.enable_secret"},
	}
	for _, c := range cases {
		ds := good(t, sc)
		c.add(ds)
		fs, _ := run(sc, ds, nil)
		if got := ids(fs); got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
			for _, f := range fs {
				t.Logf("  %s: %s", f.RuleID, f.Message)
			}
		}
	}
}

// A finding tells the cell of the book, never a secret, and keeps its key
// while nothing about it changes.
func TestValuesFindingShape(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	put(t, sc, ds, "devices", "hostname", "rt-a03", "vendor", "cisco_ios", "enable_secret", "ChangeMe")
	fs, _ := run(sc, ds, nil)
	if len(fs) != 1 {
		t.Fatalf("findings: %s", ids(fs))
	}
	f := fs[0]
	if f.Severity != check.Warning || f.At != "機器!A4" || f.Key == "" {
		t.Errorf("finding %+v", f)
	}
	for _, e := range f.Evidence {
		if strings.Contains(e.Value, "ChangeMe") {
			t.Errorf("the finding holds the secret: %+v", e)
		}
	}
	again, _ := run(sc, ds, nil)
	if again[0].Key != f.Key {
		t.Error("the key changed from run to run")
	}
}

// A row of the book without a device is told about: it is in no device's
// data.
func TestValuesNoDevice(t *testing.T) {
	sc := defs(t)
	bk := model.NewDataset("book")
	put(t, sc, bk, "ntp_servers", "server", "192.0.2.123")
	put(t, sc, bk, "ntp_servers", "device", "*", "server", "192.0.2.124")
	fs, _ := run(sc, good(t, sc), bk)
	if got := ids(fs); got != "V-101/:ntp_servers.device" {
		t.Errorf("got %q", got)
	}
}

// A name that points into a table the pack of the device does not read yet
// is a warning that does not stop the device: what it names may be in the
// lines the pack left to a person.
func TestValuesTableNotRead(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	put(t, sc, ds, "interfaces", "device", "rt-a02", "name", "Loopback0", "enabled", "true", "description", "sample", "acl_in", "SAMPLE-IN")
	put(t, sc, ds, "ntp_servers", "device", "rt-a02", "server", "192.0.2.124", "source_if", "Loopback9")
	tally := check.NewTally()
	fs := check.Values(check.Input{Schema: sc, Data: ds,
		HasPack: func(id string) bool { return id == "cisco_ios" },
		Reads:   func(pack, table string) bool { return pack == "cisco_ios" && table != "acls" },
	}, tally)
	if got := ids(fs); got != "V-115/rt-a02:interfaces.acl_in V-106/rt-a02:ntp_servers.source_if" {
		t.Errorf("got %q", got)
	}
	for _, f := range fs {
		if f.RuleID == "V-115" && (f.Severity != check.Warning || check.Blocks(f.RuleID) || !check.Approvable(f.RuleID)) {
			t.Errorf("V-115 = %+v", f)
		}
	}
	if n := tally.Looked("V-115")["rt-a02"]; n != 1 {
		t.Errorf("V-115 looked at rt-a02 %d times", n)
	}
}

// The words of the case replace the ones of the tool.
func TestValuesPlaceholders(t *testing.T) {
	sc := defs(t)
	ds := good(t, sc)
	put(t, sc, ds, "devices", "hostname", "rt-a03", "vendor", "cisco_ios", "enable_secret", "sample-later")
	tally := check.NewTally()
	fs := check.Values(check.Input{Schema: sc, Data: ds, Placeholders: []string{"sample-later"}}, tally)
	if got := ids(fs); got != "V-218/rt-a03:devices.enable_secret" {
		t.Errorf("got %q", got)
	}
}
