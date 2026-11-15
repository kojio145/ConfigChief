package ui

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kojio145/configchief/internal/packs"
)

// rendered gives the text a bundled pack writes in its generation test: a
// configuration the pack reads and then writes back the same.
func rendered(t *testing.T, id string) string {
	t.Helper()
	fsys, err := packs.FS(id)
	if err != nil {
		t.Fatal(err)
	}
	data, err := fs.ReadFile(fsys, "tests/render/basic/expect.txt")
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func deviceOf(v *ResultView, host string) DeviceResult {
	for _, d := range v.Devices {
		if d.Host == host {
			return d
		}
	}
	return DeviceResult{}
}

func findingsOf(v *ResultView, rule string) []FindingView {
	var out []FindingView
	for _, f := range v.Findings {
		if f.Rule == rule {
			out = append(out, f)
		}
	}
	return out
}

func outDirs(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(filepath.Join(dir, "out"))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

func TestResult(t *testing.T) {
	s, dir := newSession(t, map[string]string{"input/new/rt-c01.txt": rendered(t, "cisco_ios")})
	// Each step a minute after the one before: each writes its own folder.
	clock := at
	s.now = func() time.Time {
		clock = clock.Add(time.Minute)
		return clock
	}
	file := func(rel string) bool { return rel != "" && exists(filepath.Join(dir, filepath.FromSlash(rel))) }

	if _, err := s.Generate(); err == nil {
		t.Error("generated without a parameter book")
	}
	if _, err := s.Approve("0000000000000000", "", "sample reason", "sample-user"); err == nil || !strings.Contains(err.Error(), "先に") {
		t.Errorf("err = %v", err)
	}
	if _, err := s.Unapprove("a-0001"); err == nil || !strings.Contains(err.Error(), "先に") {
		t.Errorf("err = %v", err)
	}
	if h, err := s.Home(); err != nil || h.LastGen != "" {
		t.Errorf("home before a generation = %+v, %v", h, err)
	}
	if _, err := s.MakeBook("new", false); err != nil {
		t.Fatal(err)
	}

	v, err := s.Generate()
	if err != nil {
		t.Fatal(err)
	}
	noNull(t, "result", v)
	d := deviceOf(v, "rt-c01")
	if len(v.Devices) != 1 || d.State != "written" || d.Word == "" || d.Pack != "cisco_ios" || d.Mark != "" || !file(d.File) {
		t.Fatalf("devices = %+v", v.Devices)
	}
	if v.Left != 0 || v.Notice != "" || v.Message != "" || !v.Configs || v.Unverified != "" || v.Changes.Has {
		t.Errorf("result = %+v", v)
	}
	if !file(v.Report.XLSX) || !file(v.Report.HTML) || !file(v.Dir) || len(v.Counts) == 0 || len(v.Checks.Rows) == 0 {
		t.Errorf("report %+v, dir %s, counts %d, checks %d", v.Report, v.Dir, len(v.Counts), len(v.Checks.Rows))
	}
	if !strings.Contains(v.Title, "生成とチェック") {
		t.Errorf("title = %s", v.Title)
	}
	if h, err := s.Home(); err != nil || h.LastGen == "" {
		t.Errorf("home after a generation = %+v, %v", h, err)
	}

	// A second device whose values are wrong: it is stopped, the other is
	// written, and the changes since the first time are told.
	if err := os.WriteFile(filepath.Join(dir, "input", "new", "ix-c01.txt"), []byte(rendered(t, "nec_ix")), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := s.MakeBook("new", true); err != nil {
		t.Fatal(err)
	}
	var seen []string
	s.Progress = func(done, total int, name string) {
		seen = append(seen, fmt.Sprintf("%d/%d %s", done, total, name))
	}
	if v, err = s.Generate(); err != nil {
		t.Fatal(err)
	}
	s.Progress = nil
	noNull(t, "result of two devices", v)
	if strings.Join(seen, ";") != "0/2 ix-c01;1/2 rt-c01" {
		t.Errorf("progress = %v", seen)
	}
	ix, rt := deviceOf(v, "ix-c01"), deviceOf(v, "rt-c01")
	if len(v.Devices) != 2 || ix.State != "blocked" || ix.Mark != "unknown" || ix.Errors == 0 || ix.File != "" || ix.Why == "" || rt.State != "written" {
		t.Fatalf("devices = %+v", v.Devices)
	}
	v106, v202 := findingsOf(v, "V-106"), findingsOf(v, "V-202")
	if len(v106) == 0 || len(v202) < 2 {
		t.Fatalf("V-106 %d, V-202 %d", len(v106), len(v202))
	}
	if f := v106[0]; f.State != "fix" || f.CanScope || f.Mark != "unknown" || f.Device != "ix-c01" || f.Title == "" || f.Word == "" || f.Table == "" || f.Row == "" {
		t.Errorf("V-106 = %+v", f)
	}
	if f := v202[0]; f.State != "open" || !f.CanScope || f.Mark != "warn" || f.Message == "" || len(f.Evidence) == 0 || f.At == "" {
		t.Errorf("V-202 = %+v", f)
	}
	if v.Left != len(v106)+len(v202) || !file(v.Unverified) || !v.Changes.Has || !file(v.Changes.Report.HTML) || len(v.Changes.Tables) == 0 {
		t.Errorf("left %d, mark %q, changes %+v", v.Left, v.Unverified, v.Changes)
	}
	wrote := outDirs(t, dir)

	// What stops a device is not approved, nor a finding the result lacks.
	if _, err := s.Approve(v106[0].Key, "", "sample reason", "sample-user"); err == nil {
		t.Error("approved what stops a device")
	}
	if _, err := s.Approve("0000000000000000", "", "sample reason", "sample-user"); err == nil || !strings.Contains(err.Error(), "今の結果にありません") {
		t.Errorf("err = %v", err)
	}
	if _, err := s.Approve(v202[0].Key, "half", "sample reason", "sample-user"); err == nil {
		t.Error("approved by a scope that is not there")
	}
	if _, err := s.Approve(v202[0].Key, "", " ", "sample-user"); err == nil {
		t.Error("approved without a reason")
	}

	// One finding. The checks run again and nothing is written.
	a, err := s.Approve(v202[0].Key, "", "sample reason", " sample-user ")
	if err != nil {
		t.Fatal(err)
	}
	noNull(t, "result after an approval", a)
	if a.Message != "承認しました（a-0001。当たった指摘 1 件）" || !strings.Contains(a.Notice, v.Dir) || !strings.Contains(a.Notice, "変更の前のまま") {
		t.Errorf("message %q, notice %q", a.Message, a.Notice)
	}
	if a.Left != v.Left-1 || a.By != "sample-user" || a.Dir != v.Dir || a.Report != v.Report || !a.Changes.Has || a.Unverified != v.Unverified {
		t.Errorf("after an approval = left %d, by %q, dir %s", a.Left, a.By, a.Dir)
	}
	got := findingsOf(a, "V-202")[0]
	if got.Key != v202[0].Key || got.State != "approved" || got.Approval != "a-0001" || got.Reason != "sample reason" || got.By != "sample-user" || got.When == "" {
		t.Errorf("approved finding = %+v", got)
	}
	if len(a.Approvals) != 1 || a.Approvals[0].ID != "a-0001" || a.Approvals[0].Hits != 1 || a.Approvals[0].Note != "" || a.Approvals[0].Target == "" {
		t.Errorf("approvals = %+v", a.Approvals)
	}
	if now := outDirs(t, dir); strings.Join(now, ",") != strings.Join(wrote, ",") {
		t.Errorf("an approval wrote a folder: %v → %v", wrote, now)
	}
	if data, err := os.ReadFile(s.settings); err != nil || !strings.Contains(string(data), `"name": "sample-user"`) {
		t.Errorf("settings = %s, %v", data, err)
	}

	// The rest of the check on every device, then that approval taken back.
	b, err := s.Approve(v202[1].Key, "all", "sample reason", "sample-user")
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Approvals) != 2 || b.Approvals[1].ID != "a-0002" || b.Approvals[1].Hits != len(v202) || b.Approvals[1].Devices != "*" {
		t.Errorf("approvals = %+v", b.Approvals)
	}
	if b.Left != len(v106) || deviceOf(b, "rt-c01").Warnings != 0 || deviceOf(b, "ix-c01").Approved+deviceOf(b, "rt-c01").Approved != len(v202) {
		t.Errorf("left %d, devices %+v", b.Left, b.Devices)
	}
	c, err := s.Unapprove("a-0002")
	if err != nil {
		t.Fatal(err)
	}
	if c.Message != "承認 a-0002 を取り消しました" || c.Notice == "" || len(c.Approvals) != 1 || c.Left != a.Left {
		t.Errorf("after taking back = %q, %d approvals, left %d", c.Message, len(c.Approvals), c.Left)
	}
	if _, err := s.Unapprove("a-9999"); err == nil {
		t.Error("took back an approval that is not there")
	}
	// The approvals of one device: the device of the finding only.
	e, err := s.Approve(v202[1].Key, "device", "sample reason", "sample-user")
	if err != nil {
		t.Fatal(err)
	}
	if last := e.Approvals[len(e.Approvals)-1]; last.Devices != v202[1].Device || last.Hits == 0 {
		t.Errorf("approval of a device = %+v", last)
	}
	if now := outDirs(t, dir); strings.Join(now, ",") != strings.Join(wrote, ",") {
		t.Errorf("the approvals wrote a folder: %v → %v", wrote, now)
	}

	// The checks alone write their results and no configuration.
	k, err := s.Check()
	if err != nil {
		t.Fatal(err)
	}
	noNull(t, "result of the checks", k)
	if k.Configs || k.Notice != "" || k.Dir == v.Dir || !file(k.Report.HTML) || file(k.Dir+"/configs") || !strings.Contains(k.Title, "書き出していません") {
		t.Errorf("checks = dir %s, configs %v, notice %q", k.Dir, k.Configs, k.Notice)
	}
	if deviceOf(k, "rt-c01").State != "checked" || deviceOf(k, "rt-c01").File != "" {
		t.Errorf("devices = %+v", k.Devices)
	}
	if n, err := s.Unapprove("a-0001"); err != nil || !strings.Contains(n.Notice, "チェック結果表") || strings.Contains(n.Notice, "UNVERIFIED") {
		t.Errorf("after the checks alone = %+v, %v", n, err)
	}

	// A generation that is stopped writes nothing.
	wrote = outDirs(t, dir)
	s.Progress = func(done, total int, name string) { s.Cancel() }
	if _, err := s.Generate(); err != errStopped {
		t.Fatalf("err = %v", err)
	}
	s.Progress = nil
	if now := outDirs(t, dir); strings.Join(now, ",") != strings.Join(wrote, ",") {
		t.Errorf("a generation that was stopped wrote a folder: %v → %v", wrote, now)
	}

	// Another case forgets the result.
	if _, err := s.Create(filepath.Join(t.TempDir(), "other"), "sample 2", "new"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Unapprove("a-0001"); err == nil || !strings.Contains(err.Error(), "先に") {
		t.Errorf("err = %v", err)
	}
}
