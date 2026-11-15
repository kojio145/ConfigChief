package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The structured comparison on the page of the comparisons: two files
// without a case, then every device of two sides and the rows of one device.
func TestTree(t *testing.T) {
	text := sample(t, "cisco_ios")
	i := strings.Index(text, "\ninterface ")
	if i < 0 {
		t.Fatal("no interface in the sample")
	}
	j := i + 1 + strings.Index(text[i+1:], "\n")
	head := text[i+1 : j]
	changed := text[:j+1] + " load-interval 30\n" + strings.Replace(text[j+1:], "ntp server 192.0.2.123 prefer", "ntp server 192.0.2.125 prefer", 1)

	s, dir := newSession(t, map[string]string{"input/old/rt-a01.txt": text, "input/new/rt-a01.txt": changed})
	a, b := filepath.Join(dir, "input", "old", "rt-a01.txt"), filepath.Join(dir, "input", "new", "rt-a01.txt")

	if _, err := s.Tree(a, "", true, true); err == nil {
		t.Error("one file gave no error")
	}
	v, err := s.Tree(a, a, true, true)
	if err != nil {
		t.Fatal(err)
	}
	noNullValue(t, "the same file", v)
	if !v.Equal || len(v.Rows) != 0 {
		t.Errorf("the same file: %+v", v)
	}
	v, err = s.Tree(a, b, true, true)
	if err != nil {
		t.Fatal(err)
	}
	noNullValue(t, "two files", v)
	if v.Equal || v.Added != 1 || v.Changed != 1 || v.Removed != 0 || v.Moved != 0 {
		t.Fatalf("two files: %+v", v)
	}
	var kinds []string
	gaps, hit := 0, ""
	for k, r := range v.Rows {
		switch r.Op {
		case "gap":
			gaps++
		case "add":
			kinds = append(kinds, "add")
			// The heading of the block is shown above the added line.
			found := false
			for _, up := range v.Rows[:k] {
				if up.Head && len(up.TB) == 1 && up.TB[0].Text == head {
					found = true
				}
			}
			if !found {
				t.Errorf("the heading %q is not shown above the added line", head)
			}
		case "chg":
			kinds = append(kinds, "chg")
			for _, p := range r.TB {
				if p.Hit {
					hit += p.Text
				}
			}
		}
	}
	if strings.Join(kinds, " ") != "add chg" || gaps == 0 || hit != "125" {
		t.Errorf("rows %v, %d gaps, the changed part %q", kinds, gaps, hit)
	}

	// Every device of two sides.
	if _, err := s.TreeOf(0); err == nil {
		t.Error("the rows of a device before a comparison gave no error")
	}
	all, err := s.TreeAll("old", "new", true, true)
	if err != nil {
		t.Fatal(err)
	}
	noNullValue(t, "every device", all)
	if len(all.Devices) != 1 || !all.Devices[0].Both || !all.Devices[0].Differs || all.Devices[0].Added != 1 || all.Devices[0].Changed != 1 {
		t.Fatalf("every device: %+v", all.Devices)
	}
	if !strings.HasSuffix(all.Report.XLSX, "/reports/tree_diff.xlsx") || !exists(filepath.Join(dir, filepath.FromSlash(all.Report.HTML))) {
		t.Errorf("the report: %+v", all.Report)
	}
	one, err := s.TreeOf(0)
	if err != nil {
		t.Fatal(err)
	}
	noNullValue(t, "one device", one)
	if one.Title != "rt-a01" || one.Added != 1 || one.Changed != 1 {
		t.Errorf("one device: %+v", one)
	}
	if _, err := s.TreeOf(1); err == nil {
		t.Error("a device that is not there gave no error")
	}

	// A folder as a side, with a file only it has.
	other := t.TempDir()
	for name, body := range map[string]string{"rt-a01.cfg": text, "rt-b02.cfg": strings.Replace(text, "hostname rt-a01\n", "hostname rt-b02\n", 1)} {
		if err := os.WriteFile(filepath.Join(other, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	all, err = s.TreeAll("old", other, true, true)
	if err != nil {
		t.Fatal(err)
	}
	noNullValue(t, "a folder", all)
	if len(all.Devices) != 2 || all.Devices[0].Differs || all.Devices[1].Both || all.Devices[1].A != "" {
		t.Errorf("a folder: %+v", all.Devices)
	}
	if _, err := s.TreeAll("old", "", true, true); err == nil {
		t.Error("one side gave no error")
	}
}
