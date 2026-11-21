package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// judgeFile gives a sample with lines no rule takes in.
func judgeFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join("..", "ops", "testdata", "judge", name))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func judgeGroup(t *testing.T, v *JudgeView, shape string) JudgeGroup {
	t.Helper()
	for _, g := range v.Groups {
		if g.Shape == shape {
			return g
		}
	}
	t.Fatalf("no group %q in %+v", shape, v.Groups)
	return JudgeGroup{}
}

func TestJudge(t *testing.T) {
	s, dir := newSession(t, map[string]string{
		"input/old/rt-a01.txt": judgeFile(t, "rt-a01.txt"),
		"input/old/rt-a02.txt": judgeFile(t, "rt-a02.txt"),
		"input/old/short.txt":  "hostname sample-short\n",
	})
	if _, err := s.Unhandled("logs"); err == nil {
		t.Error("no error for a source that has no configurations")
	}
	if _, err := s.Unhandled("new"); err == nil || !strings.Contains(err.Error(), "input/new にファイルがありません") {
		t.Errorf("err = %v", err)
	}

	v, err := s.Unhandled("old")
	if err != nil {
		t.Fatal(err)
	}
	noNull(t, "the unhandled lines", v)
	if v.Source != "old" || v.Title != "未対応の行（旧コンフィグ）" || v.Message != "" || len(v.Groups) != 3 || v.Left != 12 || v.By != "" {
		t.Fatalf("view = %+v", v)
	}
	if len(v.Notes) != 1 || !strings.Contains(v.Notes[0], "input/old/short.txt は取り込めなかった") {
		t.Errorf("notes = %q", v.Notes)
	}
	if exists(filepath.Join(dir, "state", "judgments.yaml")) {
		t.Error("looking at the lines wrote the judgments")
	}
	sla := judgeGroup(t, v, "ip sla <n>")
	if sla.Pack != "cisco_ios" || sla.Path != "" || sla.Heads != 3 || sla.Below != 4 || !sla.Open || sla.Verdict != "未判定" || len(sla.Devices) != 2 {
		t.Errorf("group = %+v", sla)
	}
	if d := sla.Devices[0]; d.Device != "rt-a01" || d.File != "input/old/rt-a01.txt" || d.Lines != 5 || len(d.Heads) != 2 || d.Heads[0].Text != "ip sla 10" || d.Heads[0].Below != 2 || d.Verdict != "" || d.Word != "未判定" {
		t.Errorf("device = %+v", d)
	}
	if sla.Example.Text != "ip sla 10" || sla.ExampleFile != "input/old/rt-a01.txt" || len(sla.ExampleText) != 2 || sla.ExampleMore != 0 {
		t.Errorf("example = %+v, %q", sla.Example, sla.ExampleText)
	}
	policy := judgeGroup(t, v, "no mop enabled")
	if policy.Path != "interface GigabitEthernet<n>" || policy.Heads != 3 || policy.Below != 0 {
		t.Errorf("group below a line = %+v", policy)
	}

	// What is refused is not written.
	for _, c := range []struct{ want, group, devices, verdict, reason, by string }{
		{"名前を書いてください", sla.ID, "", "manual", "", " "},
		{"理由を書いてください", sla.ID, "", "not_needed", "", "admin"},
		{"どれかです", sla.ID, "", "later", "", "admin"},
		{"未対応の行にありません", "00000000", "", "hold", "", "admin"},
		{"行がありません: rt-a09", sla.ID, "rt-a09", "hold", "", "admin"},
		{"判定はありません", sla.ID, "", "none", "", ""},
	} {
		if _, err := s.Judge("old", c.group, c.devices, c.verdict, c.reason, c.by); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("judge %+v: err = %v", c, err)
		}
	}
	if exists(filepath.Join(dir, "state", "judgments.yaml")) {
		t.Error("a refused judgment was written")
	}

	v, err = s.Judge("old", sla.ID, "rt-a02", "hold", "", " admin ")
	if err != nil {
		t.Fatal(err)
	}
	noNull(t, "the judgment", v)
	if v.Message != "かたまり "+sla.ID+" に「保留」を付けました（機器 1 台、2 行）" || v.Left != 12 || v.By != "admin" {
		t.Errorf("after a judgment for a device: %q, %d, %q", v.Message, v.Left, v.By)
	}
	if g := judgeGroup(t, v, "ip sla <n>"); g.Verdict != "未判定 1 台、保留 1 台" || !g.Devices[1].Own || g.Devices[1].Verdict != "hold" || g.Devices[1].By != "admin" || g.Devices[1].At == "" {
		t.Errorf("group = %+v", g)
	}
	v, err = s.Judge("old", sla.ID, "*", "manual", "", "admin")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(v.Message, "「手動で対応」を付けました（機器 1 台、5 行）。機器を指定した判定が 1 台に残っています") || v.Left != 7 {
		t.Errorf("after a judgment for all: %q, %d", v.Message, v.Left)
	}
	if len(v.Notes) != 2 || !strings.Contains(v.Notes[0], "パラメータブックがまだ無い") {
		t.Errorf("notes = %q", v.Notes)
	}
	// The name is kept for the next time.
	again := New("1.2.3", s.settings)
	if again.name != "admin" {
		t.Errorf("the name was not kept: %q", again.name)
	}

	// The import shows the judgments: in the counts, in the files, in the lines.
	// With the new side the book tells the new host names.
	for _, name := range []string{"rt-a01.txt", "rt-a02.txt"} {
		if err := os.WriteFile(filepath.Join(dir, "input", "new", name), []byte(judgeFile(t, name)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	iv, err := s.Import("old")
	if err != nil {
		t.Fatal(err)
	}
	a, b := iv.Files[0], iv.Files[1]
	if a.Unknown != 8 || a.Open != 3 || a.Mark != "unknown" || b.Unknown != 4 || b.Open != 4 {
		t.Errorf("files = %+v, %+v", a, b)
	}
	lv, err := s.Lines("old", "input/old/rt-a01.txt")
	if err != nil {
		t.Fatal(err)
	}
	noNull(t, "the lines", lv)
	byNo := map[int]LineView{}
	for _, l := range lv.Lines {
		byNo[l.No] = l
	}
	if l := byNo[14]; l.Judged != "手動で対応" || l.Open || l.Group != sla.ID {
		t.Errorf("line 14 = %+v", l)
	}
	if l := byNo[15]; l.Judged != "手動で対応（上の行に従う）" || l.Open || l.Group != sla.ID {
		t.Errorf("line 15 = %+v", l)
	}
	if l := byNo[8]; l.Judged != "未判定" || !l.Open || l.Group != policy.ID {
		t.Errorf("line 8 = %+v", l)
	}
	if l := byNo[3]; l.Judged != "" || l.Open || l.Group != "" {
		t.Errorf("a line that was taken in = %+v", l)
	}

	// A judgment after the import shows in the lines at once; the report and
	// the counts of the import stay until the next one.
	if v, err = s.Judge("old", policy.ID, "", "not_needed", "sample reason", "admin"); err != nil {
		t.Fatal(err)
	}
	if v.Left != 4 || len(v.Notes) != 2 || !strings.Contains(v.Notes[0], "M-402: rt-a01 に「手動で対応」の行が 5 行") {
		t.Errorf("after the import: %d, %q", v.Left, v.Notes)
	}
	if lv, err = s.Lines("old", "input/old/rt-a01.txt"); err != nil {
		t.Fatal(err)
	}
	if l := lv.Lines[7]; l.No != 8 || l.Judged != "対応不要" || l.Open {
		t.Errorf("line 8 after its judgment = %+v", l)
	}

	// Taking it back.
	if v, err = s.Judge("old", policy.ID, "", "none", "", ""); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(v.Message, "の判定を取り消しました（機器 2 台、3 行）") || v.Left != 7 || v.By != "admin" {
		t.Errorf("after taking a judgment back: %q, %d, %q", v.Message, v.Left, v.By)
	}

	s.Close()
	if _, err := s.Unhandled("old"); err == nil || !strings.Contains(err.Error(), "案件を開いていません") {
		t.Errorf("err = %v", err)
	}
	if _, err := s.Judge("old", sla.ID, "", "hold", "", "admin"); err == nil || !strings.Contains(err.Error(), "案件を開いていません") {
		t.Errorf("err = %v", err)
	}
}
