package ui

import (
	"path/filepath"
	"strings"
	"testing"
)

// The list of many sites on the page of the comparisons: the secrets are
// masked whatever the case says.
func TestSites(t *testing.T) {
	base := sample(t, "cisco_ios")
	other := strings.Replace(base, "hostname rt-a01\n", "hostname rt-a02\n", 1)
	other = strings.Replace(other, " description sample lan\n", " description sample lan 2\n", 1)
	other = strings.Replace(other, "SampleHashForTestsOnly000", "SampleHashForTestsOnly999", 1)
	third := strings.Replace(base, "hostname rt-a01\n", "hostname rt-a03\n", 1)
	s, dir := newSession(t, map[string]string{"input/old/rt-a01.txt": base, "input/old/rt-a02.txt": other, "input/old/rt-a03.txt": third})

	if _, err := s.Sites("book"); err == nil {
		t.Error("no book gave no error")
	}
	v, err := s.Sites("old")
	if err != nil {
		t.Fatal(err)
	}
	noNullValue(t, "the list", v)
	if len(v.Tables) < 3 || v.Tables[0].Name != "表" || v.Tables[1].Name != "外れ値" {
		t.Fatalf("the tables: %+v", v.Tables)
	}
	var all []string
	for _, tb := range v.Tables {
		for _, r := range tb.Rows {
			all = append(all, strings.Join(r, "\t"))
		}
	}
	text := strings.Join(all, "\n")
	if !strings.Contains(text, "sample lan 2") || !strings.Contains(text, "***（別の値）") || strings.Contains(text, "SampleHashForTestsOnly") {
		t.Errorf("the rows:\n%s", text)
	}
	if !strings.HasSuffix(v.Report.XLSX, "/reports/sites.xlsx") || !exists(filepath.Join(dir, filepath.FromSlash(v.Report.HTML))) {
		t.Errorf("the report: %+v", v.Report)
	}
}
