package fill

import (
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/listimport"
	"github.com/kojio145/configchief/internal/pack"
	"github.com/kojio145/configchief/internal/packs"
)

func row(line int, kv ...string) listimport.VarRow {
	r := listimport.VarRow{Line: line, Values: map[string]string{}, Lines: map[string]bool{}, Empty: map[string]bool{}}
	for i := 0; i+1 < len(kv); i += 2 {
		r.Values[kv[i]] = kv[i+1]
		if kv[i+1] == "" {
			r.Empty[kv[i]] = true
		}
	}
	return r
}

func sites() *List {
	return &List{Name: "sites", Vars: []string{"wan_ip", "desc", "lan"},
		Devices: []string{"rt-a01", "rt-a02", "rt-a03"},
		Rows: map[string][]listimport.VarRow{
			"rt-a01": {row(2, "wan_ip", "192.0.2.1/30", "desc", "branch a01", "lan", "198.51.100.1/24")},
			"rt-a02": {row(3, "wan_ip", "192.0.2.5/30", "desc", "", "lan", "198.51.100.65/26")},
			"rt-a03": {row(4, "wan_ip", "", "desc", "branch a03", "lan", "198.51.100.129/26")},
		}}
}

func routes() *List {
	return &List{Name: "routes", Vars: []string{"dst", "nh"},
		Devices: []string{"rt-a01", "rt-a02"},
		Rows: map[string][]listimport.VarRow{
			"rt-a01": {row(2, "dst", "203.0.113.0 255.255.255.0", "nh", "192.0.2.2"), row(3, "dst", "203.0.113.128 255.255.255.128", "nh", "192.0.2.2")},
			"rt-a02": {row(4, "dst", "203.0.113.0 255.255.255.0", "nh", "192.0.2.6")},
		}}
}

// fakeBook is a book with a devices table and an interfaces table.
type fakeBook struct {
	rows map[string]map[string][]map[string]string
}

func (b *fakeBook) Table(n string) (string, bool) {
	switch strings.ToLower(n) {
	case "devices", "機器":
		return "devices", true
	case "interfaces", "インタフェース":
		return "interfaces", true
	}
	return "", false
}
func (b *fakeBook) Column(t, n string) (string, bool) {
	cols := map[string][]string{"devices": {"hostname", "site"}, "interfaces": {"name", "ipv4", "description"}}
	for _, c := range cols[t] {
		if strings.EqualFold(c, n) {
			return c, true
		}
	}
	return "", false
}
func (b *fakeBook) Has(d string) bool                    { _, ok := b.rows[d]; return ok }
func (b *fakeBook) Rows(d, t string) []map[string]string { return b.rows[d][t] }
func (b *fakeBook) Devices() []string                    { return []string{"rt-a01", "rt-a02"} }
func (b *fakeBook) Secret(t, c string) bool              { return false }

func book() *fakeBook {
	return &fakeBook{rows: map[string]map[string][]map[string]string{
		"rt-a01": {"devices": {{"hostname": "rt-a01", "site": "site-a"}},
			"interfaces": {{"name": "GigabitEthernet0/0", "ipv4": "192.0.2.1/30"}, {"name": "GigabitEthernet0/1", "ipv4": "198.51.100.1/24"}}},
		"rt-a02": {"devices": {{"hostname": "rt-a02", "site": "site-b"}}},
	}}
}

func mustBind(t *testing.T, text string, src *Sources) *Bound {
	t.Helper()
	tp, err := Parse("t", text)
	if err != nil {
		t.Fatal(err)
	}
	b, errs := Bind(tp, src)
	if len(errs) > 0 {
		t.Fatal(errs)
	}
	return b
}

func TestFillValuesAndStops(t *testing.T) {
	b := mustBind(t, `hostname {{devices.hostname}}
interface GigabitEthernet0/0
 description {{desc?}}
 ip address {{wan_ip}}
{{! a note}}
{{#each routes}}
ip route {{dst}} {{nh}}
{{/each}}
`, &Sources{Lists: []*List{sites(), routes()}, Book: book()})
	devs := b.Devices()
	if len(devs) != 3 || devs[0].Name != "rt-a01" || devs[2].Name != "rt-a03" {
		t.Fatalf("devices: %+v", devs)
	}
	o, err := b.Fill("rt-a01")
	if err != nil {
		t.Fatal(err)
	}
	want := "hostname rt-a01\ninterface GigabitEthernet0/0\n description branch a01\n ip address 192.0.2.1/30\nip route 203.0.113.0 255.255.255.0 192.0.2.2\nip route 203.0.113.128 255.255.255.128 192.0.2.2\n"
	if got := o.Text(false); got != want {
		t.Fatalf("got\n%s", got)
	}
	// desc? empty: the line is left out.
	o, err = b.Fill("RT-A02")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(o.Text(false), "description") || len(o.Left) != 1 {
		t.Fatalf("rt-a02:\n%s %v", o.Text(false), o.Left)
	}
	// rt-a03: not in the book, and its wan_ip is empty.
	if _, err := b.Fill("rt-a03"); err == nil || !strings.Contains(err.Error(), "ブックにこの機器がありません") {
		t.Fatalf("rt-a03: %v", err)
	}
}

func TestFillEmptyStopsAndMissingRows(t *testing.T) {
	b := mustBind(t, "ip address {{wan_ip}}\n{{#each}}\nip route {{dst}} {{nh}}\n{{/each}}\n", &Sources{Lists: []*List{sites(), routes()}})
	if _, err := b.Fill("rt-a03"); err == nil || !strings.Contains(err.Error(), "「wan_ip」の値が空です") || !strings.Contains(err.Error(), "4 行目") {
		t.Fatalf("empty: %v", err)
	}
	b = mustBind(t, "{{#each}}\nip route {{dst}} {{nh}}\n{{/each}}\nip address {{lan}}\n", &Sources{Lists: []*List{sites(), routes()}})
	if _, err := b.Fill("rt-a03"); err == nil || !strings.Contains(err.Error(), "「routes」の一覧に、この機器の行がありません") {
		t.Fatalf("no rows: %v", err)
	}
}

func TestFillIfAndBookEach(t *testing.T) {
	b := mustBind(t, `{{#if desc}}
! has {{desc}}
{{#else}}
! none
{{/if}}
{{#if devices.site=SITE-A}}
! site a
{{/if}}
{{#each interfaces}}
interface {{name}}
 ip address {{interfaces.ipv4}}
{{/each}}
`, &Sources{Lists: []*List{sites()}, Book: book()})
	o, err := b.Fill("rt-a01")
	if err != nil {
		t.Fatal(err)
	}
	want := "! has branch a01\n! site a\ninterface GigabitEthernet0/0\n ip address 192.0.2.1/30\ninterface GigabitEthernet0/1\n ip address 198.51.100.1/24\n"
	if o.Text(false) != want {
		t.Fatalf("got\n%s", o.Text(false))
	}
	o, err = b.Fill("rt-a02")
	if err != nil {
		t.Fatal(err)
	}
	if o.Text(false) != "! none\n" {
		t.Fatalf("rt-a02:\n%s", o.Text(false))
	}
}

func TestParseAndBindFaults(t *testing.T) {
	for _, c := range []struct{ text, want string }{
		{"{{#each}}\n{{#each}}\n{{/each}}\n{{/each}}\n", "2 行目: {{#each}} の中に {{#each}} は書けません"},
		{"{{#if x}}\nа\n", "1 行目: {{#if}} に当たる {{/if}} がありません"},
		{"{{/sync}}\n", "{{/sync}} に当たる {{#sync}} がありません"},
		{"a {{#each}}\n", "1 行目"},
		{"x {{ wan ip }}\n", "読めません"},
		{"{{#else}}\n", "{{#else}} に当たる {{#if}} がありません"},
	} {
		if _, err := Parse("t", c.text); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%q: %v", c.text, err)
		}
	}
	tp, _ := Parse("t", "ip address {{wan_ipp}}\n{{#each}}\n{{dst}} {{desc}}\n{{/each}}\n{{x.y}}\n")
	_, errs := Bind(tp, &Sources{Lists: []*List{sites(), routes()}})
	all := ""
	for _, e := range errs {
		all += e.Error() + "\n"
	}
	for _, want := range []string{"1 行目: 「wan_ipp」という変数はありません（読み込んだ変数: ", "2 行目: {{#each}} の間に 2 つ以上の変数ルール", "5 行目: 「x.y」: パラメータブックがありません"} {
		if !strings.Contains(all, want) {
			t.Errorf("want %q in\n%s", want, all)
		}
	}
	tp, _ = Parse("t", "hostname x\n")
	if _, errs := Bind(tp, &Sources{}); len(errs) != 1 || !strings.Contains(errs[0].Error(), "変数もブックの値もありません") {
		t.Errorf("no names: %v", errs)
	}
}

func loadPack(t *testing.T, id string) *pack.Pack {
	t.Helper()
	fsys, err := packs.FS(id)
	if err != nil {
		t.Fatal(err)
	}
	p, err := pack.Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestDeltaCisco(t *testing.T) {
	p := loadPack(t, "cisco_ios")
	b := mustBind(t, `interface GigabitEthernet0/0
 description {{desc}}
 ip address {{wan_ip}}
{{#sync}}
ip access-list standard MGMT
 permit 198.51.100.0 0.0.0.255
{{/sync}}
interface Loopback0
 ip address 203.0.113.1 255.255.255.255
{{#sync ip route}}
{{#each routes}}
ip route {{dst}} {{nh}}
{{/each}}
{{/sync}}
router bgp 64496
 address-family ipv4
  network 198.51.100.0 mask 255.255.255.0
`, &Sources{Lists: []*List{sites(), routes()}})
	o, err := b.Fill("rt-a01")
	if err != nil {
		t.Fatal(err)
	}
	now := `hostname rt-a01
!
interface GigabitEthernet0/0
 description old text
 ip address 192.0.2.1/30
 no ip redirects
!
ip access-list standard MGMT
 permit 198.51.100.0 0.0.0.255
 permit 192.0.2.0 0.0.0.255
!
ip route 203.0.113.0 255.255.255.0 192.0.2.2
ip route 0.0.0.0 0.0.0.0 192.0.2.2
ip domain name example.com
router bgp 64496
 address-family ipv4
 exit-address-family
end
`
	d, err := MakeDelta(o, now, p)
	if err != nil {
		t.Fatal(err)
	}
	want := `configure terminal
interface GigabitEthernet0/0
 description branch a01
 exit
ip access-list standard MGMT
 no permit 192.0.2.0 0.0.0.255
 exit
interface Loopback0
 ip address 203.0.113.1 255.255.255.255
 exit
ip route 203.0.113.128 255.255.255.128 192.0.2.2
router bgp 64496
 address-family ipv4
  network 198.51.100.0 mask 255.255.255.0
  exit-address-family
 exit
no ip route 0.0.0.0 0.0.0.0 192.0.2.2
end`
	if got := strings.Join(d.Commands, "\n"); got != want {
		t.Fatalf("got\n%s", got)
	}
	// Kept: the old description and "no ip redirects" stay (no sync there).
	var kept []string
	for _, k := range d.Kept {
		kept = append(kept, k.Text)
	}
	if strings.Join(kept, "|") != "description old text|no ip redirects|exit-address-family" {
		t.Fatalf("kept: %v", kept)
	}
	if len(d.Removed) != 2 || d.Removed[1].Cmd != "no ip route 0.0.0.0 0.0.0.0 192.0.2.2" {
		t.Fatalf("removed: %+v", d.Removed)
	}
	// Nothing to change.
	d, err = MakeDelta(&Output{Lines: []Line{{Text: "ip domain name example.com"}}}, now, p)
	if err != nil || !d.Empty() {
		t.Fatalf("empty: %v %v", d, err)
	}
}

func TestDeltaUnnegateAndQX(t *testing.T) {
	p := loadPack(t, "cisco_ios")
	o := &Output{Lines: []Line{{Text: "interface GigabitEthernet0/1", Sync: 1}, {Text: " ip address 192.0.2.9 255.255.255.252", Sync: 1}}, Syncs: []Sync{{Start: 0}}}
	d, err := MakeDelta(o, "interface GigabitEthernet0/1\n ip address 192.0.2.9 255.255.255.252\n shutdown\n no cdp enable\n", p)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(d.Commands, "\n"); got != "configure terminal\ninterface GigabitEthernet0/1\n no shutdown\n cdp enable\n exit\nend" {
		t.Fatalf("got\n%s", got)
	}
	q := loadPack(t, "nec_qx")
	o = &Output{Lines: []Line{{Text: "interface GigabitEthernet1/0/1", Sync: 1}, {Text: " port access vlan 10", Sync: 1}}, Syncs: []Sync{{Start: 0}}}
	d, err = MakeDelta(o, "#\ninterface GigabitEthernet1/0/1\n description to-a01\n port access vlan 20\n#\n", q)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Join(d.Commands, "\n"); got != "system-view\ninterface GigabitEthernet1/0/1\n port access vlan 10\n undo description\n undo port access vlan 20\n quit\nreturn" {
		t.Fatalf("qx got\n%s", got)
	}
}

func TestDraft(t *testing.T) {
	cfg := `hostname rt-a01
interface GigabitEthernet0/0
 description branch a01
 ip address 192.0.2.1 255.255.255.252
interface GigabitEthernet0/1
 ip address 192.0.2.10 255.255.255.0
ip route 203.0.113.0 255.255.255.0 192.0.2.2
ip route 203.0.113.128 255.255.255.128 192.0.2.2
ip domain name example.com
`
	d := MakeDraft(cfg, []DraftValue{{"host", "rt-a01"}, {"wan", "192.0.2.1"}, {"desc", "branch a01"}, {"vlan", "10"}, {"same", "rt-a01"}},
		[]DraftEach{{Source: "routes", Names: []string{"dst", "nh"}, Rows: []map[string]string{
			{"dst": "203.0.113.0 255.255.255.0", "nh": "192.0.2.2"}, {"dst": "203.0.113.128 255.255.255.128", "nh": "192.0.2.2"}}}})
	want := `hostname {{host}}
interface GigabitEthernet0/0
 description {{desc}}
 ip address {{wan}} 255.255.255.252
interface GigabitEthernet0/1
 ip address 192.0.2.10 255.255.255.0
{{#each routes}}
ip route {{dst}} {{nh}}
{{/each}}
ip domain name example.com
`
	if d.Text != want {
		t.Fatalf("got\n%s\nnotes %v", d.Text, d.Notes)
	}
	notes := strings.Join(d.Notes, "\n")
	if !strings.Contains(notes, "「vlan」の値「10」は短い") || !strings.Contains(notes, "「same」の値「rt-a01」は「host」と同じ") {
		t.Fatalf("notes: %s", notes)
	}
	// The draft reads back as a template.
	if _, err := Parse("d", d.Text); err != nil {
		t.Fatal(err)
	}
}
