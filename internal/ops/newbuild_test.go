package ops

import (
	"strings"
	"testing"
)

// A new build (the end of stage M6, spec 18): the empty book, the first
// device written by hand, copies of it, and the configurations of all of
// them.
func TestNewBuild(t *testing.T) {
	e := newCase(t, nil)
	if _, err := e.CloneDevice("rt-n01", []string{"rt-n02"}, at); err == nil {
		t.Error("a copy without a book")
	}
	sum, err := e.NewBook(at)
	if err != nil {
		t.Fatal(err)
	}
	if !sum.Created || sum.Rows() != 0 {
		t.Errorf("the empty book: %+v", sum)
	}
	if _, err := e.NewBook(at); err == nil || !strings.Contains(err.Error(), "既にあります") {
		t.Errorf("a second empty book: %v", err)
	}

	// The first device.
	addRow(t, e, "機器", map[string]string{"ホスト名": "rt-n01", "メーカー": "cisco_ios"})
	addRow(t, e, "インタフェース", map[string]string{"機器": "rt-n01", "IF 名": "GigabitEthernet0/0", "IPv4 アドレス": "dhcp", "説明": "sample wan"})
	addRow(t, e, "スタティック経路", map[string]string{"機器": "rt-n01", "宛先": "0.0.0.0/0", "ネクストホップ": "203.0.113.2"})
	addRow(t, e, "NTP サーバ", map[string]string{"機器": "*", "サーバ": "198.51.100.123"})

	for _, bad := range [][]string{{"rt-n01"}, {"rt n02"}, {"rt-n02", "rt-n02"}, {""}, {"*"}} {
		if _, err := e.CloneDevice("rt-n01", bad, at); err == nil {
			t.Errorf("the copy to %q was made", bad)
		}
	}
	if _, err := e.CloneDevice("rt-none", []string{"rt-n02"}, at); err == nil {
		t.Error("a copy of a device that is not there")
	}

	o, err := e.CloneDevice("rt-n01", []string{"rt-n02", " rt-n03 "}, at)
	if err != nil {
		t.Fatal(err)
	}
	if o.Summary.Rows() != 6 || o.Summary.Added["devices"] != 2 || o.Summary.Added["interfaces"] != 2 || o.Summary.Added["static_routes"] != 2 || o.Shared != 1 || o.Backup == "" {
		t.Errorf("the copy: %+v, shared %d, backup %q", o.Summary.Added, o.Shared, o.Backup)
	}
	if _, err := e.CloneDevice("rt-n01", []string{"rt-n02"}, at); err == nil {
		t.Error("a second copy to the same name")
	}

	// The copies hold what the first device holds.
	g, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	if len(g.Devices) != 3 {
		t.Fatalf("%d devices", len(g.Devices))
	}
	for _, d := range g.Devices {
		if d.State != GenWritten {
			t.Errorf("%s: %s\n%s", d.Host, d.State, strings.Join(unsettled(g), "\n"))
			continue
		}
		text := read(t, e, d.File)
		for _, want := range []string{"hostname " + d.Host, "description sample wan", "ip address dhcp", "ip route 0.0.0.0 0.0.0.0 203.0.113.2", "ntp server 198.51.100.123"} {
			if !strings.Contains(text, want) {
				t.Errorf("%s lacks %q:\n%s", d.Host, want, text)
			}
		}
	}
}
