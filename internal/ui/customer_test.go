package ui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/listimport"
)

// noNullValue is noNull for what holds the word "null" as a value of its own
// (a cipher of that name is a choice of a column).
func noNullValue(t *testing.T, name string, v any) {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), ":null") {
		t.Errorf("%s holds a null: %s", name, data)
	}
}

// The screen of the lists of the customer: a mapping is written, tried and
// saved; the lists are compared with the book; a value goes into the book
// and a finding is approved.
func TestCustomer(t *testing.T) {
	s, dir := newSession(t, map[string]string{
		"input/new/rt-a01.txt":     sample(t, "cisco_ios"),
		"input/customer/sites.csv": "機器の一覧,,,\nホスト名,拠点,VLAN10,接続パスワード\nrt-a01,site-a,192.0.2.129/25,sample-secret-1\nrt-a09,site-z,,\n",
	})
	if _, err := s.MakeBook("new", false); err != nil {
		t.Fatal(err)
	}
	m, err := s.Mappings()
	if err != nil {
		t.Fatal(err)
	}
	noNullValue(t, "the mappings", m)
	if len(m.Mappings) != 0 || len(m.Files) != 1 || m.Files[0] != "sites.csv" || len(m.Tables) < 30 || m.Tables[0].Columns[0].Name != "hostname" || !m.Tables[0].Columns[0].Key {
		t.Fatalf("mappings = %+v, files %v, %d tables", m.Mappings, m.Files, len(m.Tables))
	}
	if _, err := s.Customer(); err == nil || !strings.Contains(err.Error(), "列の割り当てファイル（.yaml）がありません") {
		t.Errorf("no mapping: %v", err)
	}

	// A new rule: the header row and the columns of the devices are guessed.
	rule := listimport.Rule{Version: 1, File: "sites.csv"}
	sm, err := s.ListSample(rule)
	if err != nil {
		t.Fatal(err)
	}
	noNull(t, "the sample", sm)
	if sm.HeaderRow != 2 || sm.Sheet != "CSV" || len(sm.Headers) != 4 || len(sm.Suggest) != 2 || sm.Suggest[0].To != "hostname" || sm.Suggest[0].Column != "ホスト名" || len(sm.Grid) != 4 {
		t.Fatalf("sample = %+v", sm)
	}
	rule.HeaderRow = sm.HeaderRow
	rule.Rows = []listimport.Target{
		{Table: "devices", Columns: sm.Suggest},
		{Table: "interfaces", Columns: []listimport.Field{{To: "device", Column: "ホスト名"}, {To: "name", Value: "GigabitEthernet0/3.10"}, {To: "ipv4", Column: "VLAN10"}}},
		{Table: "pppoe", Columns: []listimport.Field{{To: "device", Column: "ホスト名"}, {To: "name", Value: "wan"}, {To: "password", Column: "接続パスワード"}}},
	}
	// The column that goes to a secret is masked in the sample and in the try.
	if sm, err = s.ListSample(rule); err != nil || sm.Grid[2][3] != "***" || sm.Grid[1][3] != "接続パスワード" || sm.Grid[2][0] != "rt-a01" {
		t.Fatalf("masked sample: %v, %+v", err, sm.Grid)
	}
	tv, err := s.ListTry(rule)
	if err != nil {
		t.Fatal(err)
	}
	noNull(t, "the try", tv)
	if len(tv.Faults) != 0 || tv.Lines != 2 || tv.Rows != 4 || len(tv.Tables) != 3 || !strings.HasPrefix(tv.Tables[1].Name, "インタフェース: 1 行") || tv.Tables[2].Rows[0][3] != "***" {
		t.Fatalf("try = %+v", tv)
	}
	if exists(filepath.Join(dir, "mappings", "sites.yaml")) {
		t.Error("the try wrote a mapping")
	}
	// A rule with a fault is told, and is not saved.
	bad := rule
	bad.Rows = append([]listimport.Target{}, rule.Rows...)
	bad.Rows[1] = listimport.Target{Table: "interfaces", Columns: []listimport.Field{{To: "device", Column: "ホスト名"}}}
	if tv, err = s.ListTry(bad); err != nil || len(tv.Faults) != 1 || !strings.Contains(tv.Faults[0], "キーの列「IF 名」がありません") {
		t.Errorf("faults: %v, %+v", err, tv)
	}
	if _, err := s.ListSave("", bad); err == nil {
		t.Error("a rule with a fault was saved")
	}

	rule.Rows = rule.Rows[:2]
	path, err := s.ListSave("", rule)
	if err != nil || path != "mappings/sites.yaml" {
		t.Fatalf("save: %v, %s", err, path)
	}
	if again, err := s.ListSave("", rule); err != nil || again != "mappings/sites_2.yaml" {
		t.Errorf("a second new mapping: %v, %s", err, again)
	}
	os.Remove(filepath.Join(dir, "mappings", "sites_2.yaml"))
	if _, err := s.ListSave("project.yaml", rule); err == nil {
		t.Error("a file that is no mapping was written over")
	}
	rule.Names = "new"
	if again, err := s.ListSave(path, rule); err != nil || again != path {
		t.Errorf("save over: %v, %s", err, again)
	}
	if m, err = s.Mappings(); err != nil || len(m.Mappings) != 1 || m.Mappings[0].Rule == nil || len(m.Mappings[0].Problems) != 0 || m.Mappings[0].Rule.Rows[1].Columns[1].Value != "GigabitEthernet0/3.10" {
		t.Fatalf("after saving: %v, %+v", err, m.Mappings)
	}
	noNullValue(t, "the mappings", m)
	// The value after each step.
	if got := s.ListTrace(" ＲＴ－a01/x ", []listimport.Step{{Type: "trim"}, {Type: "width"}, {Type: "split", Sep: "/", Keep: "left"}}); len(got) != 3 || got[0] != "ＲＴ－a01/x" || got[1] != "RT-a01/x" || got[2] != "RT-a01" {
		t.Errorf("trace = %q", got)
	}
	if got := s.ListTrace("x", nil); got == nil || len(got) != 0 {
		t.Errorf("trace of no steps = %#v", got)
	}
	// A mapping is taken out into state/backup; a file that is no mapping is not.
	if _, err := s.ListDelete("project.yaml"); err == nil {
		t.Error("a file that is no mapping was taken out")
	}
	if _, err := s.ListSave("", rule); err != nil {
		t.Fatal(err)
	}
	kept, err := s.ListDelete("mappings/sites_2.yaml")
	if err != nil || !strings.HasPrefix(kept, "state/backup/mappings_sites_2_") || !exists(filepath.Join(dir, filepath.FromSlash(kept))) || exists(filepath.Join(dir, "mappings", "sites_2.yaml")) {
		t.Errorf("delete: %v, %s", err, kept)
	}
	if m, err = s.Mappings(); err != nil || len(m.Mappings) != 1 {
		t.Fatalf("after deleting: %v, %+v", err, m.Mappings)
	}

	// The comparison: the device rt-a09 is new, the site and the address differ.
	v, err := s.Customer()
	if err != nil {
		t.Fatal(err)
	}
	noNull(t, "the comparison", v)
	if len(v.Tables) != 7 || v.Tables[1].Name != "一覧にだけある行" || len(v.Tables[1].Rows) != 1 || len(v.Tables[2].Rows) != 2 || v.Findings != 3 || !exists(filepath.Join(dir, filepath.FromSlash(v.Report.XLSX))) {
		t.Fatalf("comparison: findings %d, tables %d", v.Findings, len(v.Tables))
	}
	var site, addr, row FindingView
	for _, f := range v.Items {
		switch {
		case f.Rule == "S-503" && f.Column == "拠点":
			site = f
		case f.Rule == "S-503":
			addr = f
		case f.Rule == "S-501":
			row = f
		}
	}
	if site.Key == "" || addr.Key == "" || row.Key == "" || !addr.CanScope || addr.State != "open" || addr.At == "" {
		t.Fatalf("findings = %+v", v.Items)
	}
	msg, err := s.Reflect([]string{addr.Key}, "")
	if err != nil || !strings.Contains(msg, "足した行 0、書き換えたセル 1") || !strings.Contains(msg, "state/backup/") {
		t.Fatalf("reflect a cell: %v, %s", err, msg)
	}
	if msg, err = s.Reflect(nil, "S-501"); err != nil || !strings.Contains(msg, "足した行 1、書き換えたセル 0") {
		t.Fatalf("reflect the rows: %v, %s", err, msg)
	}
	if v, err = s.Customer(); err != nil || len(v.Tables[1].Rows) != 0 || len(v.Tables[2].Rows) != 1 {
		t.Fatalf("after reflecting: %v", err)
	}
	if msg, err = s.ApproveMigration(site.Key, "", "the list is old", "sample-user"); err != nil || !strings.Contains(msg, "承認しました") {
		t.Fatalf("approve: %v, %s", err, msg)
	}
	if v, err = s.Customer(); err != nil || v.Findings != 0 || v.By != "sample-user" || v.Items[0].State != "approved" {
		t.Fatalf("after approving: %v, findings %d, %+v", err, v.Findings, v.Items)
	}
}
