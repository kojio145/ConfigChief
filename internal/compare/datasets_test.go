package compare

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schemas"
)

// put adds a row: key and cells are "column", "value" pairs; a value for a
// list is written with "|" between the items.
func put(d *model.Dataset, table string, key []string, cells ...string) {
	var kvs []model.KV
	for i := 0; i < len(key); i += 2 {
		kvs = append(kvs, model.KV{Name: key[i], Value: key[i+1]})
	}
	o := model.Origin{Kind: "config", File: "input/" + d.Source + "/sample.txt", Line: 1}
	r := d.Open(table, kvs, o)
	for i := 0; i < len(cells); i += 2 {
		if strings.Contains(cells[i+1], "|") {
			for _, v := range strings.Split(cells[i+1], "|") {
				r.Append(cells[i], v, o)
			}
		} else {
			r.Set(cells[i], cells[i+1], o)
		}
	}
}

func k(pairs ...string) []string { return pairs }

func TestDatasets(t *testing.T) {
	sc, err := schemas.Load()
	if err != nil {
		t.Fatal(err)
	}
	a := model.NewDataset("old")
	put(a, "devices", k("hostname", "rt-a01"), "vendor", "nec_ix", "domain_name", "example.com", "name_servers", "192.0.2.53|192.0.2.54")
	put(a, "interfaces", k("device", "rt-a01", "name", "GigaEthernet0.0"), "enabled", "true", "ipv4", "192.0.2.1/30", "description", "")
	put(a, "interfaces", k("device", "rt-a01", "name", "GigaEthernet1.0"), "enabled", "true")
	put(a, "interfaces", k("device", "rt-a01", "name", "Loopback0.0"), "enabled", "true", "ipv4", "192.0.2.255/32")
	put(a, "ntp_servers", k("device", "rt-a01", "server", "192.0.2.123"))

	b := model.NewDataset("new")
	put(b, "devices", k("hostname", "rt-a01"), "vendor", "nec_ix", "domain_name", "example.net", "name_servers", "192.0.2.54|192.0.2.53")
	put(b, "interfaces", k("device", "rt-a01", "name", "GigaEthernet0.0"), "enabled", "true", "ipv4", "192.0.2.1/30")
	put(b, "interfaces", k("device", "rt-a01", "name", "GigaEthernet1.0"), "enabled", "false", "description", "sample uplink")
	put(b, "interfaces", k("device", "rt-a01", "name", "GigaEthernet2.0"), "enabled", "true")
	put(b, "syslog_servers", k("device", "rt-a01", "server", "192.0.2.51"))

	res := Datasets(sc, a, b)
	var onlyA, onlyB, cells []string
	for _, d := range res.OnlyA {
		onlyA = append(onlyA, d.Table+" "+d.Device+" "+d.Row.KeyText())
	}
	for _, d := range res.OnlyB {
		onlyB = append(onlyB, d.Table+" "+d.Device+" "+d.Row.KeyText())
	}
	for _, d := range res.Cells {
		s := d.Table + " " + d.Device + " " + d.ARow.KeyText() + " " + d.Column + ": " + d.A.Text() + " -> " + d.B.Text()
		if d.OrderOnly {
			s += " (order)"
		}
		cells = append(cells, s)
	}
	if want := []string{
		"interfaces rt-a01 device=rt-a01,name=Loopback0.0",
		"ntp_servers rt-a01 device=rt-a01,server=192.0.2.123",
	}; !reflect.DeepEqual(onlyA, want) {
		t.Errorf("only A = %q", onlyA)
	}
	if want := []string{
		"interfaces rt-a01 device=rt-a01,name=GigaEthernet2.0",
		"syslog_servers rt-a01 device=rt-a01,server=192.0.2.51",
	}; !reflect.DeepEqual(onlyB, want) {
		t.Errorf("only B = %q", onlyB)
	}
	if want := []string{
		"devices rt-a01 hostname=rt-a01 domain_name: example.com -> example.net",
		"devices rt-a01 hostname=rt-a01 name_servers: 192.0.2.53,192.0.2.54 -> 192.0.2.54,192.0.2.53 (order)",
		"interfaces rt-a01 device=rt-a01,name=GigaEthernet1.0 description:  -> sample uplink",
		"interfaces rt-a01 device=rt-a01,name=GigaEthernet1.0 enabled: true -> false",
	}; !reflect.DeepEqual(cells, want) {
		t.Errorf("cells = %q", cells)
	}
	if res.Rows != 3 || res.Same != 1 || res.Equal() {
		t.Errorf("rows = %d, same = %d, equal = %v", res.Rows, res.Same, res.Equal())
	}

	if r := Datasets(sc, a, a); !r.Equal() || r.Rows != 5 || r.Same != 5 {
		t.Errorf("a with itself: %+v", r)
	}
	if r := Datasets(sc, a, nil); len(r.OnlyA) != 5 || len(r.OnlyB) != 0 {
		t.Errorf("a with nothing: %d rows only in A", len(r.OnlyA))
	}
}

// A name the import makes up is not in the configuration: rows that hold the
// same are one row whatever their names, and the rest is matched by name.
func TestDatasetsMadeUpName(t *testing.T) {
	sc, err := schemas.Load()
	if err != nil {
		t.Fatal(err)
	}
	a := model.NewDataset("old")
	put(a, "nat_rules", k("device", "rt-a01", "name", "nat1"), "type", "pat_interface", "outside_if", "GigaEthernet0.0")
	put(a, "nat_rules", k("device", "rt-a01", "name", "nat2"), "type", "static", "local_ip", "192.0.2.10", "global_ip", "203.0.113.10")
	put(a, "nat_rules", k("device", "rt-a01", "name", "nat3"), "type", "static", "local_ip", "192.0.2.11", "global_ip", "203.0.113.11")
	put(a, "nat_rules", k("device", "rt-a01", "name", "nat4"), "type", "static", "local_ip", "192.0.2.13", "global_ip", "203.0.113.13")

	b := model.NewDataset("new")
	put(b, "nat_rules", k("device", "rt-a01", "name", "nat1"), "type", "static", "local_ip", "192.0.2.10", "global_ip", "203.0.113.10")
	put(b, "nat_rules", k("device", "rt-a01", "name", "nat2"), "type", "pat_interface", "outside_if", "GigaEthernet0.0")
	put(b, "nat_rules", k("device", "rt-a01", "name", "nat3"), "type", "static", "local_ip", "192.0.2.11", "global_ip", "203.0.113.12")
	put(b, "nat_rules", k("device", "rt-a01", "name", "nat5"), "type", "static", "local_ip", "192.0.2.14", "global_ip", "203.0.113.14")

	res := Datasets(sc, a, b)
	var got []string
	for _, d := range res.OnlyA {
		got = append(got, "A "+d.Row.Get("name"))
	}
	for _, d := range res.OnlyB {
		got = append(got, "B "+d.Row.Get("name"))
	}
	for _, d := range res.Cells {
		got = append(got, "cell "+d.ARow.Get("name")+"/"+d.BRow.Get("name")+" "+d.Column)
	}
	want := []string{"A nat4", "B nat5", "cell nat3/nat3 global_ip"}
	if !reflect.DeepEqual(got, want) || res.Rows != 3 || res.Same != 2 {
		t.Errorf("got %v (rows %d, same %d), want %v", got, res.Rows, res.Same, want)
	}
}

// A row that points at a made-up name (a ref column) points at the same row
// on both sides, whatever each side calls it: the rows that hang on a
// renamed row are one row, in the key and in a cell alike.
func TestDatasetsPointsAtMadeUpName(t *testing.T) {
	sc, err := schemas.Load()
	if err != nil {
		t.Fatal(err)
	}
	a := model.NewDataset("book")
	put(a, "ipsec_tunnels", k("device", "rt-a01", "name", "sample-ra"), "style", "policy", "peer_address", "any", "ike_version", "1", "psk", "dummy-psk-1")
	put(a, "ipsec_selectors", k("device", "rt-a01", "tunnel", "sample-ra", "seq", "10"), "local_net", "0.0.0.0/0", "remote_net", "0.0.0.0/0", "protocol", "udp", "local_port", "1701")
	put(a, "ipsec_selectors", k("device", "rt-a01", "tunnel", "sample-ra", "seq", "20"), "local_net", "192.0.2.0/24", "remote_net", "0.0.0.0/0")
	put(a, "l2tp_groups", k("device", "rt-a01", "name", "RA-SAMPLE"), "role", "lns", "ipsec_policy", "sample-ra")

	b := model.NewDataset("running")
	put(b, "ipsec_tunnels", k("device", "rt-a01", "name", "CMAP-SAMPLE/10"), "style", "policy", "peer_address", "any", "ike_version", "1", "psk", "dummy-psk-1")
	put(b, "ipsec_selectors", k("device", "rt-a01", "tunnel", "CMAP-SAMPLE/10", "seq", "10"), "local_net", "0.0.0.0/0", "remote_net", "0.0.0.0/0", "protocol", "udp", "local_port", "1701")
	put(b, "ipsec_selectors", k("device", "rt-a01", "tunnel", "CMAP-SAMPLE/10", "seq", "20"), "local_net", "192.0.2.0/24", "remote_net", "198.51.100.0/24")
	put(b, "l2tp_groups", k("device", "rt-a01", "name", "RA-SAMPLE"), "role", "lns", "ipsec_policy", "CMAP-SAMPLE/10")

	res := Datasets(sc, a, b)
	var got []string
	for _, d := range res.OnlyA {
		got = append(got, "A "+d.Table)
	}
	for _, d := range res.OnlyB {
		got = append(got, "B "+d.Table)
	}
	for _, d := range res.Cells {
		got = append(got, "cell "+d.Table+" "+d.ARow.KeyText()+" "+d.Column)
	}
	want := []string{"cell ipsec_selectors " + a.Tables["ipsec_selectors"].Rows[1].KeyText() + " remote_net"}
	if !reflect.DeepEqual(got, want) || res.Rows != 4 || res.Same != 3 {
		t.Errorf("got %v (rows %d, same %d), want %v", got, res.Rows, res.Same, want)
	}
}

// A number that only orders the rows (schema: order_only) is compared as
// the place it gives the row: 15 and 25 on one side are 20 and 30 on the
// other, and a row more on one side is told.
func TestDatasetsOrderOnly(t *testing.T) {
	sc, err := schemas.Load()
	if err != nil {
		t.Fatal(err)
	}
	a := model.NewDataset("book")
	put(a, "ipsec_proposals", k("device", "rt-a01", "tunnel", "Tunnel1", "phase", "ipsec", "seq", "25"), "encryption", "aes128", "hash", "sha1")
	put(a, "ipsec_proposals", k("device", "rt-a01", "tunnel", "Tunnel1", "phase", "ipsec", "seq", "15"), "encryption", "aes192", "hash", "sha1")
	put(a, "ipsec_proposals", k("device", "rt-a01", "tunnel", "Tunnel1", "phase", "ike", "seq", "15"), "encryption", "aes128", "hash", "sha1", "dh", "dh14")
	b := model.NewDataset("running")
	put(b, "ipsec_proposals", k("device", "rt-a01", "tunnel", "Tunnel1", "phase", "ipsec", "seq", "20"), "encryption", "aes192", "hash", "sha1")
	put(b, "ipsec_proposals", k("device", "rt-a01", "tunnel", "Tunnel1", "phase", "ipsec", "seq", "30"), "encryption", "aes128", "hash", "sha256")
	put(b, "ipsec_proposals", k("device", "rt-a01", "tunnel", "Tunnel1", "phase", "ipsec", "seq", "40"), "encryption", "3des", "hash", "sha1")
	put(b, "ipsec_proposals", k("device", "rt-a01", "tunnel", "Tunnel1", "phase", "ike", "seq", "20"), "encryption", "aes128", "hash", "sha1", "dh", "dh14")

	res := Datasets(sc, a, b)
	var got []string
	for _, d := range res.OnlyA {
		got = append(got, "A "+d.Row.Get("seq"))
	}
	for _, d := range res.OnlyB {
		got = append(got, "B "+d.Row.Get("phase")+" "+d.Row.Get("seq"))
	}
	for _, d := range res.Cells {
		got = append(got, "cell "+d.ARow.Get("seq")+"/"+d.BRow.Get("seq")+" "+d.Column)
	}
	want := []string{"B ipsec 40", "cell 25/30 hash"}
	if !reflect.DeepEqual(got, want) || res.Rows != 3 || res.Same != 2 {
		t.Errorf("got %v (rows %d, same %d), want %v", got, res.Rows, res.Same, want)
	}
}

// A column that only a person fills in (compare: false) is not a difference
// between the book and a configuration.
func TestDatasetsLeavesOutWhatNoConfigurationTells(t *testing.T) {
	sc, err := schemas.Load()
	if err != nil {
		t.Fatal(err)
	}
	a, b := model.NewDataset("book"), model.NewDataset("new")
	put(a, "interfaces", k("device", "rt-a01", "name", "GigabitEthernet0/1"), "peer_device", "sw-b01", "peer_if", "GigabitEthernet1/0/1", "description", "to sw-b01")
	put(b, "interfaces", k("device", "rt-a01", "name", "GigabitEthernet0/1"), "description", "to sw-b01")
	if res := Datasets(sc, a, b); !res.Equal() {
		t.Errorf("the peer columns were compared: %+v", res.Cells)
	}
}
