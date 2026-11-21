package listimport

import (
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/sheet"
)

func varTable(t *testing.T, csv string, r *VarRule) *VarResult {
	t.Helper()
	bk, err := sheet.Read([]byte(csv))
	if err != nil {
		t.Fatal(err)
	}
	if faults := r.Validate(); len(faults) > 0 {
		t.Fatal(faults)
	}
	loc := r.Location()
	s, err := PickSheet(bk, loc)
	if err != nil {
		t.Fatal(err)
	}
	tb, err := ReadTable(s, loc)
	if err != nil {
		t.Fatal(err)
	}
	return ReadVars(tb, r)
}

func values(res *VarResult, dev, name string) string {
	var out []string
	for _, r := range res.Rows[DeviceKey(dev)] {
		out = append(out, r.Values[name])
	}
	return strings.Join(out, ",")
}

// A device named once for several rows: the rows below continue it; a
// device left out takes its rows with it; a row left out by a variable's
// value is that row only (PalaTerm's readFill).
func TestReadVarsContinuing(t *testing.T) {
	csv := "機器,状態,VLAN,アドレス\n" +
		"rt-a01,,10,192.0.2.1/24\n" +
		",,20,192.0.2.129/25\n" +
		",,,\n" +
		",,30,\n" +
		"rt-a02,廃止,10,198.51.100.1/24\n" +
		",,20,198.51.100.129/25\n" +
		"rt-a03,,10,203.0.113.1/24\n" +
		"rt-a01,,40,203.0.113.129/25\n"
	r := &VarRule{Version: 1, File: "x.csv", HeaderRow: 1, Device: Field{Column: "機器"},
		Exclude: []VarCond{{Column: "状態", Op: "equals", Value: "廃止"}, {Var: "addr", Op: "empty"}},
		Vars:    []Var{{Name: "vlan", Field: Field{Column: "VLAN"}}, {Name: "addr", Field: Field{Column: "アドレス"}}}}
	res := varTable(t, csv, r)
	if strings.Join(res.Devices, " ") != "rt-a01 rt-a03" {
		t.Fatalf("devices: %v", res.Devices)
	}
	if got := values(res, "rt-a01", "vlan"); got != "10,20,40" {
		t.Fatalf("rt-a01: %s", got)
	}
	// Lines: 3 of rt-a01, 1 of rt-a03; left out: vlan 30 (no address) and
	// rt-a02's own row; its continuing row goes with it unseen.
	if res.Lines != 4 || res.Excluded != 2 {
		t.Fatalf("lines %d excluded %d", res.Lines, res.Excluded)
	}
	if len(res.Sample) == 0 || !res.Sample[1].Cont || res.Sample[1].Device != "rt-a01" {
		t.Fatalf("sample: %+v", res.Sample)
	}
	if !strings.Contains(res.Sample[2].Excluded, "変数「addr」が空") {
		t.Fatalf("why: %+v", res.Sample[2])
	}
	// A list that starts with a row without a device name.
	res = varTable(t, "機器,VLAN,アドレス\n,10,192.0.2.1/24\nrt-a01,20,192.0.2.2/24\n", &VarRule{Version: 1, File: "x.csv", HeaderRow: 1,
		Device: Field{Column: "機器"}, Vars: []Var{{Name: "vlan", Field: Field{Column: "VLAN"}}}})
	if values(res, "rt-a01", "vlan") != "20" || len(res.Warnings) != 1 {
		t.Fatalf("first row: %v %v", values(res, "rt-a01", "vlan"), res.Warnings)
	}
}

// Two devices on a row: a row whose first device cell is empty continues
// both; one whose second device cell is empty has no second device.
func TestReadVarsAlso(t *testing.T) {
	csv := "主,予備,拠点,経路,主の次,予備の次\n" +
		"rt-a01,rt-a02,site-a,203.0.113.0/24,192.0.2.2,192.0.2.6\n" +
		",,,198.51.100.0/24,192.0.2.2,192.0.2.6\n" +
		"rt-b01,,site-b,203.0.113.0/24,192.0.2.10,\n" +
		",,,198.51.100.0/24,192.0.2.10,\n"
	r := &VarRule{Version: 1, File: "x.csv", HeaderRow: 1, Device: Field{Column: "主"},
		Vars: []Var{{Name: "site", Field: Field{Column: "拠点"}, Empty: "line"}, {Name: "dst", Field: Field{Column: "経路"}}, {Name: "nh", Field: Field{Column: "主の次"}}},
		Also: []AlsoDevice{{Label: "予備", Device: Field{Column: "予備"}, Vars: []Var{{Name: "nh", Field: Field{Column: "予備の次"}}}}}}
	res := varTable(t, csv, r)
	if strings.Join(res.Devices, " ") != "rt-a01 rt-a02 rt-b01" {
		t.Fatalf("devices: %v", res.Devices)
	}
	if values(res, "rt-a02", "nh") != "192.0.2.6,192.0.2.6" || values(res, "rt-a02", "dst") != "203.0.113.0/24,198.51.100.0/24" {
		t.Fatalf("rt-a02: %s %s", values(res, "rt-a02", "nh"), values(res, "rt-a02", "dst"))
	}
	if values(res, "rt-b01", "nh") != "192.0.2.10,192.0.2.10" {
		t.Fatalf("rt-b01: %s", values(res, "rt-b01", "nh"))
	}
	// The site is on the first row only: the continuing row's lines are
	// left out, not the device.
	rows := res.Rows["rt-a01"]
	if !rows[1].Lines["site"] || rows[1].Empty["site"] {
		t.Fatalf("site of a continuing row: %+v", rows[1])
	}
}

func TestMatchNameAndAlign(t *testing.T) {
	m := VarMatch{Width: true, Space: true, Domain: true, Dash: true}
	if MatchName(" ＲＴ_A01.example.com ", m) != "rt-a01" || MatchName("192.0.2.1", m) != "192.0.2.1" {
		t.Fatalf("%q %q", MatchName(" ＲＴ_A01.example.com ", m), MatchName("192.0.2.1", m))
	}
	res := &VarResult{Devices: []string{"RT_A01.example.com", "rt-a01", "rt-z99"}, Rows: map[string][]VarRow{
		"rt_a01.example.com": {{Line: 2}}, "rt-a01": {{Line: 3}}, "rt-z99": {{Line: 4}}}}
	res.Align([]string{"rt-a01", "rt-a02"}, m)
	if strings.Join(res.Devices, " ") != "rt-a01 rt-z99" || len(res.Rows["rt-a01"]) != 2 {
		t.Fatalf("aligned: %v %v", res.Devices, res.Rows)
	}
	// Nothing to leave out: names stay as the list has them.
	res = &VarResult{Devices: []string{"RT-A01 "}, Rows: map[string][]VarRow{"rt-a01": {{Line: 2}}}}
	res.Align([]string{"rt-a01"}, VarMatch{})
	if res.Devices[0] != "RT-A01 " {
		t.Fatalf("no match: %v", res.Devices)
	}
}

func TestVarRuleFaults(t *testing.T) {
	r := &VarRule{Version: 1, File: "x.csv", HeaderRow: 1, Device: Field{Column: "機器"},
		Exclude: []VarCond{{Var: "nope", Op: "empty"}, {Column: "a", Var: "v", Op: "empty"}, {Op: "empty"}},
		Vars:    []Var{{Name: "v", Field: Field{Column: "V"}, Empty: "value"}},
		Also:    []AlsoDevice{{Device: Field{Column: "機器2"}, Vars: []Var{{Name: "v", Field: Field{Column: "V2"}, Empty: "line"}}}}}
	all := strings.Join(r.Validate(), "\n")
	for _, want := range []string{"変数「nope」は vars にありません", "どちらか 1 つだけ", "column（一覧の見出し）か var", "empty「value」は使えません", "empty は 1 台目の変数に書きます"} {
		if !strings.Contains(all, want) {
			t.Errorf("want %q in\n%s", want, all)
		}
	}
}
