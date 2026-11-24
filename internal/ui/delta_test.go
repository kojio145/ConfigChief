package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The part of the screen for the change commands made from the book: the
// sides and devices to choose from, a change and a device unchanged.
func TestDeltaRun(t *testing.T) {
	cfg := "hostname rt-d01\n!\ninterface GigabitEthernet0/1\n description sample lan\n ip address 198.51.100.1 255.255.255.0\n no shutdown\n!\nip route 0.0.0.0 0.0.0.0 198.51.100.254\n!\nend\n"
	s, dir := newSession(t, map[string]string{"input/new/rt-d01.txt": cfg})
	if _, err := s.MakeBook("new", false); err != nil {
		t.Fatal(err)
	}
	v, err := s.Delta()
	if err != nil || len(v.Sides) != 3 || v.Sides[1].Side != "new" || v.Sides[1].Files != 1 || len(v.Devices) != 1 || v.Devices[0] != "rt-d01" {
		t.Fatalf("delta view = %+v, %v", v, err)
	}
	noNull(t, "the delta view", v)
	if err := os.MkdirAll(filepath.Join(dir, "input", "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	now := strings.Replace(cfg, " description sample lan\n", " description old lan\n zz-unsupported-sub 1\n", 1)
	if err := writeTestFile(filepath.Join(dir, "input", "logs", "rt-d01.txt"), []byte(now), 0o644); err != nil {
		t.Fatal(err)
	}
	r, err := s.DeltaRun("logs", nil)
	if err != nil {
		t.Fatal(err)
	}
	noNull(t, "the delta run", r)
	if r.Made != 1 || len(r.Devices) != 1 || r.Report == "" || len(r.Table.Rows) != 1 || r.Table.Rows[0][1] != "作った" {
		t.Fatalf("run = %+v", r)
	}
	d := r.Devices[0]
	if d.Result != "made" || !strings.Contains(d.Text, "interface GigabitEthernet0/1\n description sample lan\n exit\n") || strings.Contains(d.Text, "zz-unsupported") ||
		len(d.Added) != 1 || len(d.Removed) != 0 || len(d.Replaced) != 1 || !strings.Contains(d.Replaced[0], "→ description sample lan") ||
		len(d.Kept) != 1 || !strings.Contains(d.Kept[0], "zz-unsupported-sub 1　（取り込めない行") {
		t.Fatalf("device = %+v", d)
	}
	// The device now as the book: unchanged.
	if r, err = s.DeltaRun("new", []string{"rt-d01"}); err != nil || r.Unchanged != 1 || r.Devices[0].Result != "unchanged" {
		t.Fatalf("unchanged = %+v, %v", r, err)
	}
}
