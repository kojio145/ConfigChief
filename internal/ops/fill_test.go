package ops

import (
	"os"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/listimport"
)

// A list of sites: two devices on a line (a pair), and a list of routes with
// two rows for one device.
const fillSitesCSV = "ホスト名,ホスト名 予備,拠点,WAN IP,WAN IP 予備,LAN,状態\n" +
	"rt-a01,rt-a02,site-a,192.0.2.1 255.255.255.252,192.0.2.5 255.255.255.252,198.51.100.1/24,\n" +
	"rt-b01,,site-b,,,198.51.100.65/26,\n" +
	"rt-c01,,site-c,192.0.2.9/30,,198.51.100.129/26,廃止\n"

const fillRoutesCSV = "機器,宛先,次\n" +
	"rt-a01,203.0.113.0 255.255.255.0,192.0.2.2\n" +
	"rt-a01,203.0.113.128 255.255.255.128,192.0.2.2\n" +
	"rt-a02,203.0.113.0 255.255.255.0,192.0.2.6\n"

const fillSitesRule = `version: 1
file: sites.csv
header_row: 1
exclude:
  - {column: 状態, op: equals, value: 廃止}
device: {column: ホスト名}
vars:
  - {name: site, column: 拠点}
  - {name: wan, column: WAN IP, steps: [{type: cidr}]}
  - {name: lan, column: LAN}
also:
  - device: {column: ホスト名 予備}
    vars:
      - {name: wan, column: WAN IP 予備, steps: [{type: cidr}]}
`

const fillRoutesRule = `version: 1
file: routes.csv
header_row: 1
device: {column: 機器}
vars:
  - {name: dst, column: 宛先}
  - {name: nh, column: 次}
`

const fillTemplate = `hostname {{devices.hostname}}
interface GigabitEthernet0/0
 description {{site}}
 ip address {{wan}}
{{#sync ip route}}
{{#each routes}}
ip route {{dst}} {{nh}}
{{/each}}
{{/sync}}
`

const fillNow = `hostname rt-a01
!
interface GigabitEthernet0/0
 description old
 ip address 192.0.2.1/30
!
ip route 0.0.0.0 0.0.0.0 192.0.2.2
end
`

func fillCase(t *testing.T) *Env {
	t.Helper()
	e := newCase(t, map[string]string{
		"input/customer/sites.csv":  fillSitesCSV,
		"input/customer/routes.csv": fillRoutesCSV,
		"vars/sites.yaml":           fillSitesRule,
		"vars/routes.yaml":          fillRoutesRule,
		"fill/branch.txt":           strings.ReplaceAll(fillTemplate, "{{devices.hostname}}", "{{devices.hostname?}}"),
		"fill/plain.txt":            "interface GigabitEthernet0/0\n ip address {{wan}}\n",
		"input/old/rt-a01.txt":      fillNow,
	})
	return e
}

func TestFillWhole(t *testing.T) {
	e := fillCase(t)
	o, err := e.Fill(FillOptions{Template: "plain"}, at)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, d := range o.Devices {
		got = append(got, d.Name+":"+map[bool]string{true: "made", false: "no"}[d.Made])
	}
	// rt-c01 is excluded; rt-b01 has no WAN address; rt-a02 takes its own.
	if strings.Join(got, " ") != "rt-a01:made rt-a02:made rt-b01:no" {
		t.Fatalf("devices: %v", got)
	}
	data, err := os.ReadFile(e.Proj.Path(o.Devices[1].File))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "interface GigabitEthernet0/0\n ip address 192.0.2.5/30\n" {
		t.Fatalf("rt-a02:\n%s", data)
	}
	if !strings.Contains(o.Devices[2].Why, "「wan」の値が空です") {
		t.Fatalf("rt-b01: %s", o.Devices[2].Why)
	}
	if o.NotMade() != 1 || o.Report.XLSX == "" {
		t.Fatalf("outcome: %+v", o)
	}
	if !strings.Contains(strings.Join(o.Notes, "\n"), "除外の条件に当たった 1 行") {
		t.Fatalf("notes: %v", o.Notes)
	}
	// One device only; a name the template is not for.
	if o, err = e.Fill(FillOptions{Template: "plain.txt", Devices: []string{"RT-A02"}}, at); err != nil || len(o.Devices) != 1 {
		t.Fatalf("one: %v %v", o, err)
	}
	if _, err = e.Fill(FillOptions{Template: "plain", Devices: []string{"rt-z99"}}, at); err == nil || !strings.Contains(err.Error(), "rt-z99") {
		t.Fatalf("unknown device: %v", err)
	}
	// The book is used but the case has none.
	if _, err = e.Fill(FillOptions{Template: "branch"}, at); err == nil || !strings.Contains(err.Error(), "パラメータブックがありません") {
		t.Fatalf("no book: %v", err)
	}
}

func TestFillDiff(t *testing.T) {
	e := fillCase(t)
	put(t, e, "fill/change.txt", strings.ReplaceAll(fillTemplate, "hostname {{devices.hostname}}\n", ""))
	o, err := e.Fill(FillOptions{Template: "change", Diff: "old"}, at)
	if err != nil {
		t.Fatal(err)
	}
	a := o.Devices[0]
	if !a.Made {
		t.Fatalf("rt-a01: %s", a.Why)
	}
	want := "configure terminal\ninterface GigabitEthernet0/0\n description site-a\n exit\nip route 203.0.113.0 255.255.255.0 192.0.2.2\nip route 203.0.113.128 255.255.255.128 192.0.2.2\nno ip route 0.0.0.0 0.0.0.0 192.0.2.2\nend\n"
	if a.Text != want {
		t.Fatalf("got\n%s", a.Text)
	}
	if len(a.Kept) != 1 || a.Kept[0].Text != "description old" {
		t.Fatalf("kept: %+v", a.Kept)
	}
	if !strings.Contains(o.Devices[1].Why, "今のコンフィグ（input/old）に、この機器のファイルがありません") {
		t.Fatalf("rt-a02: %s", o.Devices[1].Why)
	}
	// Nothing to change once it is in.
	put(t, e, "input/old/rt-a01.txt", "hostname rt-a01\ninterface GigabitEthernet0/0\n description site-a\n ip address 192.0.2.1/30\nip route 203.0.113.0 255.255.255.0 192.0.2.2\nip route 203.0.113.128 255.255.255.128 192.0.2.2\nend\n")
	if o, err = e.Fill(FillOptions{Template: "change", Diff: "old", Devices: []string{"rt-a01"}}, at); err != nil || !o.Devices[0].Unchanged {
		t.Fatalf("unchanged: %+v %v", o.Devices[0], err)
	}
}

func TestFillDraftAndSave(t *testing.T) {
	e := fillCase(t)
	d, err := e.FillDraft("rt-a01.txt", "rt-a01")
	if err != nil {
		t.Fatal(err)
	}
	// The description now is not the list's; the routes of the list are not
	// in the configuration, which the notes tell.
	if !strings.Contains(d.Text, " description old") || !strings.Contains(d.Text, " ip address {{wan}}") || !strings.Contains(strings.Join(d.Notes, ""), "変数ルール「routes」の 2 行") {
		t.Fatalf("draft:\n%s\n%v", d.Text, d.Notes)
	}
	rel, fault, err := e.SaveTemplate("", "branch", d.Text)
	if err != nil || fault != "" || rel != "fill/branch_2.txt" {
		t.Fatalf("save: %s %s %v", rel, fault, err)
	}
	if _, fault, _ = e.SaveTemplate(rel, "", "x {{#each}}\n"); fault == "" {
		t.Fatal("a fault should be told")
	}
	if _, _, err := e.SaveTemplate("fill/none.txt", "", "x"); err == nil {
		t.Fatal("only a template of the case is written over")
	}
	kept, err := e.DeleteTemplate(rel, at)
	if err != nil || !strings.HasPrefix(kept, "state/backup/fill_branch_2_") {
		t.Fatalf("delete: %s %v", kept, err)
	}
}

func TestVarRuleEdit(t *testing.T) {
	e := fillCase(t)
	r, err := listimport.ParseVars([]byte(fillRoutesRule))
	if err != nil {
		t.Fatal(err)
	}
	tr, err := e.TryVarRule(r)
	if err != nil || len(tr.Faults) > 0 {
		t.Fatalf("try: %v %v", tr, err)
	}
	if tr.Devices != 2 || tr.Total != 3 || strings.Join(tr.Header, ",") != "機器,一覧の行,dst,nh" {
		t.Fatalf("try: %+v", tr)
	}
	if _, err := e.SaveVarRule("", "routes", r); err == nil {
		t.Fatal("a name taken should be refused")
	}
	rel, err := e.SaveVarRule("", "routes2", r)
	if err != nil || rel != "vars/routes2.yaml" {
		t.Fatalf("save: %s %v", rel, err)
	}
	// Two rules with the same variables: a template using them is refused.
	if _, err := e.Fill(FillOptions{Template: "plain"}, at); err != nil {
		t.Fatalf("plain still works: %v", err)
	}
	put(t, e, "fill/routes.txt", "{{#each}}\nip route {{dst}} {{nh}}\n{{/each}}\n")
	if _, err := e.Fill(FillOptions{Template: "routes"}, at); err == nil || !strings.Contains(err.Error(), "2 つ以上の変数ルール") {
		t.Fatalf("same names: %v", err)
	}
	if _, err := e.DeleteVarRule(rel, at); err != nil {
		t.Fatal(err)
	}
	bad := *r
	bad.Vars = append(bad.Vars, listimport.Var{Name: "a.b"})
	if _, err := e.SaveVarRule("", "bad", &bad); err == nil {
		t.Fatal("a fault should stop the saving")
	}
}
