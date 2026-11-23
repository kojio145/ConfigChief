package ops

import (
	"os"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/project"
	"github.com/xuri/excelize/v2"
)

// deltaCase makes a case whose book was made from a Cisco, an NEC IX and an
// NEC QX configuration, and gives the configurations generated from it.
func deltaCase(t *testing.T) (*Env, map[string]string) {
	t.Helper()
	read := func(p string) string {
		t.Helper()
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	e := genCase(t, map[string]string{
		"input/new/rt-g01.txt": read("testdata/basics/rt-g01.txt"),
		"input/new/ix-g01.txt": read("testdata/basics/ix-g01.txt"),
		"input/new/sw-q11.txt": read("testdata/qx/sw-q11.txt"),
	})
	gen, err := e.Generate(GenOptions{Dry: true}, at)
	if err != nil {
		t.Fatal(err)
	}
	texts := map[string]string{}
	for _, d := range gen.Devices {
		if d.Output == nil || d.Output.Failed {
			t.Fatalf("%s was not generated: %s", d.Host, d.Why)
		}
		texts[d.Host] = d.Output.Text
	}
	return e, texts
}

// nowFile writes a configuration now: the generated text with some lines
// replaced.
func nowFile(t *testing.T, e *Env, rel, text string, rs ...[2]string) {
	t.Helper()
	for _, r := range rs {
		if !strings.Contains(text, r[0]) {
			t.Fatalf("%s: the text has no %q", rel, r[0])
		}
		text = strings.Replace(text, r[0], r[1], 1)
	}
	if err := os.MkdirAll(e.Proj.Path(rel[:strings.LastIndex(rel, "/")]), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(e.Proj.Path(rel), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func deltaOf(t *testing.T, o *DeltaOutcome, host string) *DeltaDevice {
	t.Helper()
	for _, d := range o.Devices {
		if d.Name == host {
			return d
		}
	}
	t.Fatalf("no device %s", host)
	return nil
}

func keptTexts(d *DeltaDevice) map[string]string {
	out := map[string]string{}
	for _, it := range d.Kept {
		out[itemText(it)] = it.Why
	}
	return out
}

// On each of the three vendors: a value changed, a line the device has that
// the book does not (removed), a line of the book the device lacks (added),
// lines no rule reads (never removed, also inside a block whose row the book
// does not have), a whole block the book does not have and a physical port
// the book does not list (left alone).
func TestDelta(t *testing.T) {
	e, texts := deltaCase(t)
	nowFile(t, e, "input/logs/rt-g01.txt", texts["rt-g01"],
		[2]string{" description sample lan\n", " description old lan\n zz-unsupported-sub 1\n"},
		[2]string{"ip route 0.0.0.0 0.0.0.0 Dialer1\n", "ip route 0.0.0.0 0.0.0.0 Dialer1\nip route 203.0.113.0 255.255.255.0 192.0.2.254\nzz-unsupported-top 1\n"},
		[2]string{"logging buffered 65536 warnings\n", ""},
		[2]string{"secret 0 dummy-admin-secret-1\n", "secret 0 dummy-admin-secret-0\n"},
		[2]string{"dialer-list 1", "interface Loopback9\n ip address 203.0.113.9 255.255.255.255\n!\ninterface Loopback8\n ip address 203.0.113.8 255.255.255.255\n zz-unsupported-lo 1\n!\ninterface GigabitEthernet0/2\n description spare\n shutdown\n!\ndialer-list 1"},
	)
	nowFile(t, e, "input/logs/ix-g01.txt", texts["ix-g01"],
		[2]string{"  description sample-lan\n", "  description old-lan\n"},
		[2]string{"ntp server 192.0.2.123\n", "ntp server 192.0.2.123\nip route 203.0.113.0/24 192.0.2.254\nzz-unsupported-top 1\n"},
		[2]string{"syslog ip host 192.0.2.50\n", ""},
	)
	nowFile(t, e, "input/logs/sw-q11.txt", texts["sw-q11"],
		[2]string{" description office desk\n", " description old desk\n"},
		[2]string{" ip route-static 0.0.0.0 0 192.0.2.126\n", " ip route-static 0.0.0.0 0 192.0.2.126\n ip route-static 203.0.113.0 24 192.0.2.254\n zz-unsupported-top 1\n"},
		[2]string{" ntp-service unicast-server 192.0.2.10\n", ""},
		[2]string{"vlan 20\n", "vlan 30\n name sample-spare\n#\nvlan 20\n"},
	)
	o, err := e.Delta(DeltaOptions{Side: "logs"}, at)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"rt-g01": "configure terminal\n" +
			"username admin privilege 15 secret 0 dummy-admin-secret-1\n" +
			"interface GigabitEthernet0/1\n description sample lan\n exit\n" +
			"logging buffered 65536 warnings\n" +
			"no interface Loopback9\n" +
			"interface Loopback8\n no ip address 203.0.113.8 255.255.255.255\n exit\n" +
			"no ip route 203.0.113.0 255.255.255.0 192.0.2.254\n" +
			"end\n",
		"ix-g01": "configure\n" +
			"syslog ip host 192.0.2.50\n" +
			"interface GigaEthernet1.0\n  description sample-lan\n  exit\n" +
			"no ip route 203.0.113.0/24 192.0.2.254\n" +
			"exit\n",
		"sw-q11": "system-view\n" +
			"interface GigabitEthernet1/0/1\n description office desk\n quit\n" +
			" ntp-service unicast-server 192.0.2.10\n" +
			"undo vlan 30\n" +
			" undo ip route-static 203.0.113.0 24 192.0.2.254\n" +
			"return\n",
	}
	for host, text := range want {
		d := deltaOf(t, o, host)
		if !d.Made || d.Text != text {
			t.Errorf("%s (%s):\n%s\nwant\n%s", host, d.Why, d.Text, text)
		}
		// A line no rule reads is never removed.
		if strings.Contains(d.Text, "zz-unsupported") {
			t.Errorf("%s: a line no rule reads is in the change:\n%s", host, d.Text)
		}
		if got := keptTexts(d)["zz-unsupported-top 1"]; got != keptUnknown {
			t.Errorf("%s: the line no rule reads: %q (kept %v)", host, got, keptTexts(d))
		}
		// The old description is written over, not removed.
		found := false
		for _, it := range d.Replaced {
			if strings.HasPrefix(it.Text, "description old") && strings.HasPrefix(it.Why, "description ") {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: replaced %+v", host, d.Replaced)
		}
		if strings.Contains(d.Text, "no description") || strings.Contains(d.Text, "undo description") {
			t.Errorf("%s: the description is removed:\n%s", host, d.Text)
		}
		data, err := os.ReadFile(e.Proj.Path(d.File))
		if err != nil || !strings.Contains(strings.ReplaceAll(string(data), "\r\n", "\n"), text) {
			t.Errorf("%s: the file %s: %v", host, d.File, err)
		}
	}
	rt := keptTexts(deltaOf(t, o, "rt-g01"))
	for line, why := range map[string]string{
		"interface GigabitEthernet0/1 > zz-unsupported-sub 1": keptUnknown,
		"interface Loopback8":                       keptInside,
		"interface Loopback8 > zz-unsupported-lo 1": keptUnknown,
		"interface GigabitEthernet0/2":              keptPhysical,
	} {
		if rt[line] != why {
			t.Errorf("rt-g01 %q: %q, want %q", line, rt[line], why)
		}
	}
	if o.Made() != 3 || o.NotMade() != 0 || o.Dir != "out/"+project.Stamp(at)+"/delta" {
		t.Errorf("outcome: made %d, not made %d, dir %s", o.Made(), o.NotMade(), o.Dir)
	}
	// The report hides the secrets (of the book and of the device now).
	html, err := os.ReadFile(e.Proj.Path(o.Report.HTML))
	if err != nil {
		t.Fatal(err)
	}
	if !e.mask() || strings.Contains(string(html), "dummy-admin-secret") || !strings.Contains(string(html), masked) {
		t.Errorf("the report shows a secret")
	}
	if !strings.Contains(string(html), "zz-unsupported-top 1") || !strings.Contains(string(html), "no ip route 203.0.113.0 255.255.255.0 192.0.2.254") {
		t.Errorf("the report lacks the lines")
	}

	// One device; a name the book does not have; a side that is not one.
	if o, err = e.Delta(DeltaOptions{Side: "logs", Devices: []string{"SW-Q11"}}, at); err != nil || len(o.Devices) != 1 {
		t.Fatalf("one device: %v %v", o, err)
	}
	if _, err = e.Delta(DeltaOptions{Side: "logs", Devices: []string{"rt-z99"}}, at); err == nil || !strings.Contains(err.Error(), "rt-z99") {
		t.Fatalf("unknown device: %v", err)
	}
	if _, err = e.Delta(DeltaOptions{Side: "running"}, at); err == nil {
		t.Fatal("an unknown side")
	}
}

// appendRow writes a row at the end of a sheet of the book, by headings.
func appendRow(t *testing.T, e *Env, sheet string, values map[string]string) {
	t.Helper()
	f, err := excelize.OpenFile(e.Proj.BookPath())
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	rows, err := f.GetRows(sheet)
	if err != nil || len(rows) == 0 {
		t.Fatalf("sheet %s: %v", sheet, err)
	}
	n := 0
	for head, v := range values {
		for c, h := range rows[0] {
			if h == head {
				cell, _ := excelize.CoordinatesToCellName(c+1, len(rows)+1)
				f.SetCellStr(sheet, cell, v)
				n++
			}
		}
	}
	if n != len(values) {
		t.Fatalf("sheet %s has not every heading of %v: %v", sheet, values, rows[0])
	}
	if err := f.Save(); err != nil {
		t.Fatal(err)
	}
}

// A device the same as the book is unchanged; the configuration of another
// vendor, a device without a configuration now, a device of input/old under
// its old host name, and one whose interfaces the book renames.
func TestDeltaDevices(t *testing.T) {
	e, texts := deltaCase(t)
	nowFile(t, e, "input/old/rt-old.txt", texts["rt-g01"],
		[2]string{"hostname rt-g01\n", "hostname rt-old\n"},
		[2]string{" description sample wan\n", " description old wan\n"})
	nowFile(t, e, "input/old/ix-g01.txt", strings.ReplaceAll(texts["rt-g01"], "rt-g01", "ix-g01"))
	nowFile(t, e, "input/old/sw-q11.txt", texts["sw-q11"])
	appendRow(t, e, "機器対応", map[string]string{"旧ホスト名": "rt-old", "新ホスト名": "rt-g01"})
	o, err := e.Delta(DeltaOptions{Side: "old"}, at)
	if err != nil {
		t.Fatal(err)
	}
	rt := deltaOf(t, o, "rt-g01")
	want := "configure terminal\nhostname rt-g01\ninterface GigabitEthernet0/0\n description sample wan\n exit\nend\n"
	if !rt.Made || rt.Text != want || rt.NowHost != "rt-old" || len(rt.Removed) != 0 {
		t.Errorf("rt-g01 (%s) from rt-old:\n%s\nremoved %+v", rt.Why, rt.Text, rt.Removed)
	}
	if ix := deltaOf(t, o, "ix-g01"); ix.Made || !strings.Contains(ix.Why, "メーカーパック（cisco_ios）と、生成するメーカーパック（nec_ix）が違います") {
		t.Errorf("ix-g01: %+v", ix)
	}
	if sw := deltaOf(t, o, "sw-q11"); !sw.Unchanged || sw.Made || sw.File != "" {
		t.Errorf("sw-q11: %+v", sw)
	}
	if o.NotMade() != 1 {
		t.Errorf("not made: %d", o.NotMade())
	}

	// The book renames an interface of the device: no change.
	appendRow(t, e, "IF 対応", map[string]string{"新ホスト名": "rt-g01", "旧 IF 名": "GigabitEthernet0/0", "新 IF 名": "GigabitEthernet0/3"})
	if o, err = e.Delta(DeltaOptions{Side: "old"}, at); err != nil {
		t.Fatal(err)
	}
	if rt := deltaOf(t, o, "rt-g01"); rt.Made || !strings.Contains(rt.Why, "IF 名が変わる") {
		t.Errorf("renamed: %+v", rt)
	}
	// No configuration now.
	if err := os.Remove(e.Proj.Path("input/old/sw-q11.txt")); err != nil {
		t.Fatal(err)
	}
	if o, err = e.Delta(DeltaOptions{Side: "old"}, at); err != nil {
		t.Fatal(err)
	}
	if sw := deltaOf(t, o, "sw-q11"); sw.Made || !strings.Contains(sw.Why, "この機器のファイルがありません") {
		t.Errorf("sw-q11 without a file: %+v", sw)
	}
}
