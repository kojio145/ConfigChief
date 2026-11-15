package ui

import (
	"path/filepath"
	"strings"
	"testing"
)

// The report to carry out, from the page of the unhandled lines.
func TestCarryOut(t *testing.T) {
	text := strings.Replace(sample(t, "cisco_ios"), "hostname rt-a01\n", "hostname rt-a01\nzzfeature HONSHA-GW mode 10\n", 1)
	s, dir := newSession(t, map[string]string{"input/old/rt-a01.txt": text})
	if _, err := s.CarryOut("book"); err == nil {
		t.Error("a source that is no configuration gave no error")
	}
	v, err := s.CarryOut("old")
	if err != nil {
		t.Fatal(err)
	}
	noNullValue(t, "the report", v)
	if !strings.HasSuffix(v.Report.XLSX, "/reports/unhandled_shapes.xlsx") || !exists(filepath.Join(dir, filepath.FromSlash(v.Report.HTML))) {
		t.Errorf("the report: %+v", v.Report)
	}
	var words []string
	for _, r := range v.Hidden.Rows {
		words = append(words, r[0])
	}
	if got := strings.Join(words, " "); !strings.Contains(got, "HONSHA-GW") || !strings.Contains(got, "zzfeature") {
		t.Errorf("the hidden words: %s", got)
	}
	if !strings.Contains(v.Summary, "目で確かめてください") {
		t.Errorf("the summary: %s", v.Summary)
	}
}
