package ui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/ops"
)

// The tab 書き換え of the result: the kinds, the devices and the lines,
// narrowed to a device and a kind (spec 13.4).
func TestRewriteView(t *testing.T) {
	old := rendered(t, "cisco_ios")
	s, dir := newSession(t, map[string]string{
		"input/old/rt-c01.txt": old + "sample-unknown-command 1\n",
		"input/new/rt-c01.txt": old,
	})
	if _, err := s.Rewrite("", ""); err == nil || !strings.Contains(err.Error(), "先に") {
		t.Errorf("err = %v", err)
	}
	// The renewal to the next model of the same vendor: the new host has
	// the same name, and the proposals fill in the mapping.
	if _, err := s.Import("old"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.MakeBook("old", false); err != nil {
		t.Fatal(err)
	}
	r, err := s.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if !r.Rewrite || r.RewriteReport.XLSX == "" || !exists(filepath.Join(dir, filepath.FromSlash(r.RewriteReport.XLSX))) {
		t.Fatalf("rewrite %v, report %+v", r.Rewrite, r.RewriteReport)
	}
	v, err := s.Rewrite("", "")
	if err != nil {
		t.Fatal(err)
	}
	noNull(t, "rewrite", v)
	lines := strings.Count(strings.TrimSuffix(old, "\n"), "\n") + 2
	if !v.Has || v.Total != lines || v.Matched != lines || len(v.Table.Rows) != lines || len(v.Kinds) != len(ops.RewriteKinds) || len(v.Devices) != 1 {
		t.Fatalf("view: has %v, total %d (want %d), matched %d, rows %d, kinds %d, devices %d", v.Has, v.Total, lines, v.Matched, len(v.Table.Rows), len(v.Kinds), len(v.Devices))
	}
	if len(v.Table.Header) != len(ops.RewriteHeader) || v.Report != r.RewriteReport {
		t.Errorf("header %v, report %+v", v.Table.Header, v.Report)
	}
	sum := 0
	for _, k := range v.Kinds {
		sum += k.Count
	}
	if sum != lines {
		t.Errorf("the kinds add up to %d, want %d", sum, lines)
	}

	u, err := s.Rewrite("rt-c01", ops.RwUnread)
	if err != nil {
		t.Fatal(err)
	}
	if u.Matched != 1 || len(u.Table.Rows) != 1 || !strings.Contains(u.Table.Rows[0][3], "sample-unknown-command") || u.Table.Marks[0] != "unknown" {
		t.Errorf("the unknown line: %d, %v", u.Matched, u.Table.Rows)
	}
	if _, err := s.Rewrite("rt-x99", ""); err == nil {
		t.Error("an old device that is not there was taken")
	}
	if _, err := s.Rewrite("", "sample"); err == nil {
		t.Error("a kind that is not there was taken")
	}

	// After an approval the checks run again and the tab is still there.
	f := r.Findings
	if len(f) > 0 {
		a, err := s.Approve(f[0].Key, "", "sample reason", "sample-user")
		if err == nil && !a.Rewrite {
			t.Error("the rewriting report is gone after an approval")
		}
	}
}

// A case of a new build has no tab 書き換え.
func TestRewriteViewNone(t *testing.T) {
	s, _ := newSession(t, map[string]string{"input/new/rt-c01.txt": rendered(t, "cisco_ios")})
	if _, err := s.Import("new"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.MakeBook("new", false); err != nil {
		t.Fatal(err)
	}
	r, err := s.Generate()
	if err != nil {
		t.Fatal(err)
	}
	if r.Rewrite || r.RewriteReport.XLSX != "" {
		t.Errorf("rewrite %v, report %+v", r.Rewrite, r.RewriteReport)
	}
	v, err := s.Rewrite("", "")
	if err != nil {
		t.Fatal(err)
	}
	noNull(t, "no rewrite", v)
	if v.Has {
		t.Error("a case of a new build has a rewriting report")
	}
}
