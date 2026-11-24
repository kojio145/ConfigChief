package convert

import (
	"reflect"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/book"
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

// put adds a row: key and cells are "column", "value" pairs; a value for a
// list is written with "|" between the items.
func put(d *model.Dataset, table string, key []string, cells ...string) {
	var kvs []model.KV
	for i := 0; i < len(key); i += 2 {
		kvs = append(kvs, model.KV{Name: key[i], Value: key[i+1]})
	}
	o := model.Origin{Kind: "config", File: "input/old/sample.txt", Line: 1}
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

func oldData() *model.Dataset {
	d := model.NewDataset("old")
	put(d, "devices", k("hostname", "rt-a01"), "vendor", "cisco_ios", "os_version", "15.1")
	put(d, "devices", k("hostname", "rt-a02"), "vendor", "cisco_ios")
	put(d, "devices", k("hostname", "rt-a03"), "vendor", "cisco_ios")
	put(d, "interfaces", k("device", "rt-a01", "name", "GigabitEthernet0/0"), "enabled", "true", "ipv4", "192.0.2.1/30")
	put(d, "interfaces", k("device", "rt-a01", "name", "GigabitEthernet0/1"), "enabled", "true", "ipv4", "unnumbered:Loopback0")
	put(d, "interfaces", k("device", "rt-a01", "name", "Loopback0"), "enabled", "true", "ipv4", "192.0.2.255/32")
	put(d, "interfaces", k("device", "rt-a01", "name", "Serial0/0"), "enabled", "false")
	put(d, "interfaces", k("device", "rt-a02", "name", "GigabitEthernet0/0"), "enabled", "true")
	put(d, "static_routes", k("device", "rt-a01", "vrf", "", "prefix", "198.51.100.0/24", "next_hop", "", "out_if", "Serial0/0"))
	put(d, "static_routes", k("device", "rt-a01", "vrf", "", "prefix", "203.0.113.0/24", "next_hop", "192.0.2.2", "out_if", "GigabitEthernet0/0"))
	put(d, "static_routes", k("device", "rt-a01", "vrf", "", "prefix", "203.0.113.128/25", "next_hop", "", "out_if", "Null0"))
	put(d, "ntp_servers", k("device", "rt-a01", "server", "192.0.2.123"), "source_if", "Loopback0")
	put(d, "ntp_servers", k("device", "rt-a01", "server", "192.0.2.124"), "source_if", "Serial0/0")
	put(d, "ntp_servers", k("device", "rt-a03", "server", "192.0.2.123"))
	put(d, "tracks", k("device", "rt-a01", "id", "1"), "type", "interface", "target", "GigabitEthernet0/0")
	put(d, "tracks", k("device", "rt-a01", "id", "2"), "type", "icmp", "target", "192.0.2.2", "source_if", "Loopback0")
	put(d, "tracks", k("device", "rt-a01", "id", "3"), "type", "interface", "target", "Serial0/0")
	put(d, "tracks", k("device", "rt-a01", "id", "4"), "type", "interface", "target", "Serial0/9")
	return d
}

func TestPropose(t *testing.T) {
	old := oldData()
	// Without a new side nothing is guessed.
	dev, ifs := Propose(old, nil, nil)
	want := []book.DeviceMap{
		{OldHost: "rt-a01", OldVendor: "cisco_ios"},
		{OldHost: "rt-a02", OldVendor: "cisco_ios"},
		{OldHost: "rt-a03", OldVendor: "cisco_ios"},
	}
	if !reflect.DeepEqual(dev, want) || len(ifs) != 0 {
		t.Errorf("no new side: dev = %+v, ifs = %+v", dev, ifs)
	}

	// A new configuration of the same host name gives the proposal; a host
	// name a person wrote into the book gives the rows of its interfaces.
	nw := model.NewDataset("new")
	put(nw, "devices", k("hostname", "rt-a01"), "vendor", "nec_ix", "model", "sample-model")
	put(nw, "interfaces", k("device", "rt-a01", "name", "Loopback0"), "enabled", "true")
	have := []book.DeviceMap{{OldHost: "rt-a02", NewHost: "rt-b02", NewVendor: "nec_ix", Row: 3}}
	dev, ifs = Propose(old, nw, have)
	wantDev := []book.DeviceMap{
		{OldHost: "rt-a01", OldVendor: "cisco_ios", NewHost: "rt-a01", NewVendor: "nec_ix", NewModel: "sample-model"},
		{OldHost: "rt-a03", OldVendor: "cisco_ios"},
	}
	if !reflect.DeepEqual(dev, wantDev) {
		t.Errorf("dev = %+v", dev)
	}
	wantIfs := []book.IfMap{
		{NewHost: "rt-a01", OldIf: "GigabitEthernet0/0"},
		{NewHost: "rt-a01", OldIf: "GigabitEthernet0/1"},
		{NewHost: "rt-a01", OldIf: "Loopback0", NewIf: "Loopback0", Basis: BasisSameName},
		{NewHost: "rt-a01", OldIf: "Serial0/0"},
		{NewHost: "rt-b02", OldIf: "GigabitEthernet0/0"},
	}
	if !reflect.DeepEqual(ifs, wantIfs) {
		t.Errorf("ifs = %+v", ifs)
	}
}

func values(d *model.Dataset, table string) []string {
	var out []string
	if d.Tables[table] == nil {
		return nil
	}
	for _, r := range d.Tables[table].Rows {
		var cells []string
		for _, c := range r.Columns() {
			cells = append(cells, c+"="+strings.Join(r.Cells[c].Values, "|"))
		}
		out = append(out, strings.Join(cells, " "))
	}
	return out
}

func TestApply(t *testing.T) {
	sc := defs(t)
	dev := []book.DeviceMap{
		{OldHost: "rt-a01", NewHost: "ix-b01", NewVendor: "nec_ix", NewModel: "sample-model"},
		{OldHost: "rt-a02", Out: true}, // "-": not carried over, no reason needed (spec 14.18.1)
	}
	ifs := []book.IfMap{
		{NewHost: "ix-b01", OldIf: "GigabitEthernet0/0", NewIf: "GigaEthernet0.0"},
		{NewHost: "ix-b01", OldIf: "Loopback0", NewIf: "Loopback0.0"},
		{NewHost: "ix-b01", OldIf: "Serial0/0", NewIf: "-"},
		{NewHost: "ix-b01", OldIf: "GigabitEthernet0/1"},
	}
	a := Apply(sc, oldData(), dev, ifs)

	if got, want := values(a.Data, "devices"), []string{"hostname=ix-b01 model=sample-model os_version=15.1 vendor=nec_ix"}; !reflect.DeepEqual(got, want) {
		t.Errorf("devices = %q", got)
	}
	wantIf := []string{
		"device=ix-b01 name=GigaEthernet0.0 enabled=true ipv4=192.0.2.1/30",
		"device=ix-b01 name=GigabitEthernet0/1 enabled=true ipv4=unnumbered:Loopback0.0",
		"device=ix-b01 name=Loopback0.0 enabled=true ipv4=192.0.2.255/32",
	}
	if got := values(a.Data, "interfaces"); !reflect.DeepEqual(got, wantIf) {
		t.Errorf("interfaces = %q", got)
	}
	wantRoutes := []string{
		"device=ix-b01 vrf= prefix=203.0.113.0/24 next_hop=192.0.2.2 out_if=GigaEthernet0.0",
		"device=ix-b01 vrf= prefix=203.0.113.128/25 next_hop= out_if=Null0",
	}
	if got := values(a.Data, "static_routes"); !reflect.DeepEqual(got, wantRoutes) {
		t.Errorf("static_routes = %q", got)
	}
	wantNTP := []string{
		"device=ix-b01 server=192.0.2.123 source_if=Loopback0.0",
		"device=ix-b01 server=192.0.2.124",
	}
	if got := values(a.Data, "ntp_servers"); !reflect.DeepEqual(got, wantNTP) {
		t.Errorf("ntp_servers = %q", got)
	}
	// A column that holds an interface or something else: a value is renamed
	// only when it is an interface of the device.
	wantTracks := []string{
		"device=ix-b01 id=1 target=GigaEthernet0.0 type=interface",
		"device=ix-b01 id=2 source_if=Loopback0.0 target=192.0.2.2 type=icmp",
		"device=ix-b01 id=3 type=interface",
		"device=ix-b01 id=4 target=Serial0/9 type=interface",
	}
	if got := values(a.Data, "tracks"); !reflect.DeepEqual(got, wantTracks) {
		t.Errorf("tracks = %q", got)
	}
	// The key follows the new names.
	if r := a.Data.Tables["interfaces"].Rows[0]; r.KeyText() != "device=ix-b01,name=GigaEthernet0.0" {
		t.Errorf("key = %s", r.KeyText())
	}
	// The old data is not changed.
	if got := oldData().Tables["interfaces"].Rows[0].KeyText(); got != "device=rt-a01,name=GigabitEthernet0/0" {
		t.Errorf("old key = %s", got)
	}

	if !reflect.DeepEqual(a.Hosts, map[string]string{"rt-a01": "ix-b01"}) {
		t.Errorf("hosts = %v", a.Hosts)
	}
	if !reflect.DeepEqual(a.Unmapped, []string{"rt-a02", "rt-a03"}) || !reflect.DeepEqual(a.NoReason, []string{"rt-a03"}) {
		t.Errorf("unmapped = %v, no reason = %v", a.Unmapped, a.NoReason)
	}
	if len(a.Undecided) != 1 || a.Undecided[0].Value != "GigabitEthernet0/1" || a.Undecided[0].Device != "rt-a01" {
		t.Errorf("undecided = %+v", a.Undecided)
	}
	var dropped []string
	for _, n := range a.Dropped {
		dropped = append(dropped, n.Table+" "+n.Key)
	}
	if want := []string{"interfaces device=rt-a01,name=Serial0/0", "static_routes device=rt-a01,vrf=,prefix=198.51.100.0/24,next_hop=,out_if=Serial0/0"}; !reflect.DeepEqual(dropped, want) {
		t.Errorf("dropped = %q", dropped)
	}
	var cleared []string
	for _, n := range a.Cleared {
		cleared = append(cleared, n.Table+" "+n.Column+" "+n.Value)
	}
	if want := []string{"ntp_servers source_if Serial0/0", "tracks target Serial0/0"}; !reflect.DeepEqual(cleared, want) {
		t.Errorf("cleared = %q", cleared)
	}
	if len(a.Clashes) != 0 || len(a.Shared) != 0 {
		t.Errorf("clashes = %+v, shared = %+v", a.Clashes, a.Shared)
	}
}

func TestApplySharedInterface(t *testing.T) {
	sc := defs(t)
	dev := []book.DeviceMap{{OldHost: "rt-a01", NewHost: "rt-a01"}}
	ifs := []book.IfMap{
		{NewHost: "rt-a01", OldIf: "GigabitEthernet0/0", NewIf: "GigabitEthernet0/0/0"},
		{NewHost: "rt-a01", OldIf: "GigabitEthernet0/1", NewIf: "GigabitEthernet0/0/0"},
		{NewHost: "rt-a01", OldIf: "Loopback0", NewIf: "Loopback0"},
		{NewHost: "rt-a01", OldIf: "Serial0/0", NewIf: "-"},
	}
	a := Apply(sc, oldData(), dev, ifs)
	if len(a.Shared) != 1 || a.Shared[0].Value != "GigabitEthernet0/0/0" || a.Shared[0].Key != "GigabitEthernet0/0, GigabitEthernet0/1" {
		t.Errorf("shared = %+v", a.Shared)
	}
	if len(a.Clashes) != 1 || a.Clashes[0].Key != "device=rt-a01,name=GigabitEthernet0/1" {
		t.Errorf("clashes = %+v", a.Clashes)
	}
	if n := len(a.Data.Tables["interfaces"].Rows); n != 2 {
		t.Errorf("interfaces rows = %d", n)
	}
}
