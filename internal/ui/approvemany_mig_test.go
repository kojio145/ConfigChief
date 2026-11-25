package ui

import (
	"strings"
	"testing"
)

// The findings of the mapping are picked and approved together, with no
// reason and no name (spec 14.22.10).
func TestApproveMigrationMany(t *testing.T) {
	s, _ := newSession(t, map[string]string{"input/old/rt-a01.txt": sample(t, "cisco_ios")})
	if _, err := s.Import("old"); err != nil {
		t.Fatal(err)
	}
	m, err := s.Mapping()
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for _, f := range m.Findings {
		if f.State == "open" {
			keys = append(keys, f.Key)
		}
	}
	if len(keys) == 0 {
		t.Fatal("no finding to approve")
	}
	if _, err := s.ApproveMigrationMany(nil, "", ""); err == nil {
		t.Error("nothing picked was taken")
	}
	msg, err := s.ApproveMigrationMany(keys, "", "")
	if err != nil || !strings.Contains(msg, "件を承認しました") {
		t.Fatalf("approve = %q, %v", msg, err)
	}
	if m, err = s.Mapping(); err != nil {
		t.Fatal(err)
	}
	for _, f := range m.Findings {
		if f.State == "open" {
			t.Errorf("left open: %+v", f)
		} else if f.Reason != "（理由なし）" {
			t.Errorf("reason shown: %q", f.Reason)
		}
	}
}
