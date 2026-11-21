package ops

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/project"
)

func judgeFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile("testdata/judge/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// judgeCase is a case with two old configurations that hold lines no pack
// reads: a line below an interface, a block of three lines, a single line.
func judgeCase(t *testing.T) *Env {
	t.Helper()
	return newCase(t, map[string]string{
		"input/old/rt-a01.txt": judgeFile(t, "rt-a01.txt"),
		"input/old/rt-a02.txt": judgeFile(t, "rt-a02.txt"),
	})
}

func groupOf(t *testing.T, j *Judging, shape string) *LineGroup {
	t.Helper()
	for _, g := range j.Groups {
		if g.Key.Shape == shape {
			return g
		}
	}
	t.Fatalf("no group of the shape %q", shape)
	return nil
}

func review(t *testing.T, e *Env, source string) *Judging {
	t.Helper()
	j, err := e.Review(source)
	if err != nil {
		t.Fatal(err)
	}
	return j
}

func judgeLines(t *testing.T, e *Env, req JudgeRequest) *JudgeOutcome {
	t.Helper()
	o, err := e.JudgeLines(req, at)
	if err != nil {
		t.Fatal(err)
	}
	return o
}

func TestReview(t *testing.T) {
	e := judgeCase(t)
	j := review(t, e, "old")
	if len(j.Groups) != 3 || j.Open != 12 || j.Hold+j.Manual+j.NotNeeded != 0 || j.Findings() != 12 {
		t.Fatalf("groups %d, open %d, findings %d", len(j.Groups), j.Open, j.Findings())
	}
	// A line below a line that was read: the line above is the path.
	g := j.Groups[0]
	if g.PathText() != "interface GigabitEthernet<n>" || g.Key.Shape != "no mop enabled" || g.Key.Pack != "cisco_ios" || len(g.ID) != 8 {
		t.Errorf("group 0 = %+v", g)
	}
	if heads, below := g.Count(); len(g.Devices) != 2 || heads != 3 || below != 0 {
		t.Errorf("group 0: %d devices, %d + %d lines", len(g.Devices), heads, below)
	}
	// A block: the judgment goes on its first line, the rest follows.
	g = j.Groups[1]
	if heads, below := g.Count(); g.Key.Shape != "ip sla <n>" || len(g.Key.Path) != 0 || heads != 3 || below != 4 {
		t.Errorf("group 1 = %+v: %d + %d lines", g.Key, heads, below)
	}
	if g.Example.Line != 14 || g.Example.Text != "ip sla 10" || g.Example.Below != 2 ||
		!reflect.DeepEqual(g.ExampleBelow, []string{" icmp-echo 192.0.2.254", " frequency 30"}) {
		t.Errorf("example = %+v, %q", g.Example, g.ExampleBelow)
	}
	if d := g.Device("rt-a01"); d == nil || len(d.Heads) != 2 || d.Below != 3 || d.Lines() != 5 || d.Heads[1].Text != "ip sla 20" || d.Heads[1].Below != 1 {
		t.Errorf("rt-a01 in group 1 = %+v", d)
	}
	if v, ok := j.Line("input/old/rt-a01.txt", 15); !ok || v.Head || v.Group() != g.ID || v.Word() != "未判定（上の行に従う）" {
		t.Errorf("line 15 = %+v, %v", v, ok)
	}
	if v, ok := j.Line("input/old/rt-a01.txt", 14); !ok || !v.Head || v.Word() != "未判定" {
		t.Errorf("line 14 = %+v, %v", v, ok)
	}
	if _, ok := j.Line("input/old/rt-a01.txt", 3); ok {
		t.Error("a line that was read has a verdict")
	}
	if j.OpenLines("input/old/rt-a01.txt") != 8 || j.OpenLines("input/old/rt-a02.txt") != 4 {
		t.Errorf("open lines = %d, %d", j.OpenLines("input/old/rt-a01.txt"), j.OpenLines("input/old/rt-a02.txt"))
	}
	// Looking writes nothing.
	if exists(e, "state/judgments.yaml") || exists(e, e.Proj.Config.Book) {
		t.Error("the review wrote a file")
	}

	// A group is found by its ID, or by the beginning of it.
	if got, err := j.Group(strings.ToUpper(g.ID[:5])); err != nil || got != g {
		t.Errorf("by the beginning of the ID: %v, %v", got, err)
	}
	if _, err := j.Group("zzzz"); err == nil || !strings.Contains(err.Error(), "旧コンフィグ") {
		t.Errorf("an ID that is not there: %v", err)
	}
	if _, err := j.Group(""); err == nil {
		t.Error("no ID is taken as a group")
	}
}

func TestJudgeLines(t *testing.T) {
	e := judgeCase(t)
	sla := groupOf(t, review(t, e, "old"), "ip sla <n>")

	// What cannot be judged.
	for _, c := range []struct {
		req  JudgeRequest
		want string
	}{
		{JudgeRequest{Group: sla.ID, Verdict: "not_needed", By: "admin"}, "理由"},
		{JudgeRequest{Group: sla.ID, Verdict: "manual"}, "名前"},
		{JudgeRequest{Group: sla.ID, Verdict: "done", By: "admin"}, "manual"},
		{JudgeRequest{Group: "zzzz", Verdict: "hold", By: "admin"}, "ありません"},
		{JudgeRequest{Group: sla.ID, Verdict: "hold", By: "admin", Devices: "rt-a01,rt-z99"}, "rt-z99"},
		{JudgeRequest{Group: sla.ID, Verdict: "none"}, "判定はありません"},
		{JudgeRequest{Source: "new", Group: sla.ID, Verdict: "hold", By: "admin"}, "input/new"},
	} {
		if _, err := e.JudgeLines(c.req, at); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%+v: %v, want an error with %q", c.req, err, c.want)
		}
	}
	if exists(e, "state/judgments.yaml") {
		t.Fatal("a judgment that was refused was written")
	}

	// For every device. There is no book yet, so the new host names are not known.
	o := judgeLines(t, e, JudgeRequest{Group: sla.ID, Verdict: "manual", By: " admin "})
	if o.Devices != 2 || o.Lines != 7 || o.Own != 0 || o.Group == nil || o.Group.VerdictText() != "手動で対応" || o.Group.ByText() != "admin" {
		t.Errorf("for every device = %+v, group %+v", o, o.Group)
	}
	j := o.Judging
	if j.Manual != 7 || j.Open != 5 || !j.HostsUnknown || len(j.NoExtra) != 0 || j.Findings() != 5 {
		t.Errorf("after the judgment: manual %d, open %d, hosts unknown %v, no extra %v", j.Manual, j.Open, j.HostsUnknown, j.NoExtra)
	}
	if v, _ := j.Line("input/old/rt-a02.txt", 10); v.Word() != "手動で対応（上の行に従う）" || v.By != "admin" {
		t.Errorf("a line below = %+v", v)
	}
	text := read(t, e, "state/judgments.yaml")
	for _, want := range []string{"shape: ip sla <n>", "pack: cisco_ios", `devices: '*'`, "verdict: manual", "by: admin", "at: \"2026-01-02T03:04:05Z\""} {
		if !strings.Contains(text, want) {
			t.Errorf("state/judgments.yaml lacks %q:\n%s", want, text)
		}
	}
	// The log tells the group by its ID: the shape stays out (spec 6.7).
	log := read(t, e, "state/log.txt")
	if !strings.Contains(log, "judge: old "+sla.ID+" 手動で対応（全台、機器 2 台、7 行、判定した人 admin）") || strings.Contains(log, "ip sla") {
		t.Errorf("log = %q", log)
	}

	// A device of its own comes before the judgment for every device.
	o = judgeLines(t, e, JudgeRequest{Group: sla.ID, Verdict: "hold", By: "operator", Devices: "rt-a02"})
	if o.Devices != 1 || o.Lines != 2 || o.Group.VerdictText() != "保留 1 台、手動で対応 1 台" || o.Group.ByText() != differs {
		t.Errorf("for a device = %+v, %q", o, o.Group.VerdictText())
	}
	if d := o.Group.Device("rt-a02"); !d.Own || d.Verdict != project.VerdictHold || o.Judging.Hold != 2 || o.Judging.Manual != 5 {
		t.Errorf("rt-a02 = %+v, hold %d, manual %d", d, o.Judging.Hold, o.Judging.Manual)
	}
	// Judging every device again leaves the device of its own, and says so.
	o = judgeLines(t, e, JudgeRequest{Group: sla.ID, Verdict: "not_needed", Reason: "新機器では監視の仕組みが変わる", By: "admin", Devices: "*"})
	if o.Devices != 1 || o.Lines != 5 || o.Own != 1 || o.Group.Device("rt-a02").Verdict != project.VerdictHold || o.Group.Device("rt-a01").Reason == "" {
		t.Errorf("for every device again = %+v", o)
	}
	if list, err := e.Proj.Judgments(); err != nil || len(list) != 2 {
		t.Errorf("judgments = %+v, %v", list, err)
	}

	// Taking the judgment of a device away brings it back to the one for all.
	o = judgeLines(t, e, JudgeRequest{Group: sla.ID, Verdict: "none", Devices: "rt-a02"})
	if d := o.Group.Device("rt-a02"); d.Own || d.Verdict != project.VerdictNotNeeded || o.Devices != 1 || o.Judging.NotNeeded != 7 {
		t.Errorf("after taking the judgment of rt-a02 away = %+v, %+v", d, o)
	}
	if _, err := e.JudgeLines(JudgeRequest{Group: sla.ID, Verdict: "none", Devices: "rt-a02"}, at); err == nil || !strings.Contains(err.Error(), "全台への判定が当たっています") {
		t.Errorf("taking away what is not there: %v", err)
	}
	o = judgeLines(t, e, JudgeRequest{Group: sla.ID, Verdict: "none"})
	if o.Group.VerdictText() != "未判定" || o.Judging.Open != 12 || o.Devices != 2 {
		t.Errorf("after taking all away = %+v, %q", o, o.Group.VerdictText())
	}
	if list, err := e.Proj.Judgments(); err != nil || len(list) != 0 {
		t.Errorf("judgments = %+v, %v", list, err)
	}
	// A judgment for a device alone is not taken away by "every device".
	judgeLines(t, e, JudgeRequest{Group: sla.ID, Verdict: "hold", By: "admin", Devices: "rt-a01"})
	if _, err := e.JudgeLines(JudgeRequest{Group: sla.ID, Verdict: "none"}, at); err == nil || !strings.Contains(err.Error(), "機器を指定した判定が 1 台") {
		t.Errorf("taking away for all with a device of its own: %v", err)
	}
}

func TestJudgeNoExtra(t *testing.T) {
	e := judgeCase(t)
	sla := groupOf(t, review(t, e, "old"), "ip sla <n>")
	judgeLines(t, e, JudgeRequest{Group: sla.ID, Verdict: "manual", By: "admin"})

	// The import makes the book. No new host name yet: the devices are out of
	// the rewriting, and M-402 does not look at them.
	o, err := e.RunImport("old", "", at)
	if err != nil {
		t.Fatal(err)
	}
	if j := o.Judging; j == nil || j.HostsUnknown || len(j.NoExtra) != 0 || j.Manual != 7 || j.Open != 5 {
		t.Fatalf("judging of the import = %+v", o.Judging)
	}
	setCell(t, e, "機器対応", "旧ホスト名", "rt-a01", "新ホスト名", "rt-b01")
	j := review(t, e, "old")
	if want := []NoExtra{{Device: "rt-a01", Host: "rt-b01", Lines: 5}}; !reflect.DeepEqual(j.NoExtra, want) || j.Findings() != 6 {
		t.Errorf("no extra = %+v, findings %d", j.NoExtra, j.Findings())
	}
	// The report of the import tells it.
	if o, err = e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	h := read(t, e, o.Report.HTML)
	for _, want := range []string{
		"未対応の形", sla.ID, "ip sla &lt;n&gt;", "interface GigabitEthernet&lt;n&gt;",
		"手動で対応（上の行に従う）", "うち未判定（M-401）", "うち手動で対応",
		"M-402: rt-a01（新ホスト名 rt-b01）", "rt-b01.j2",
	} {
		if !strings.Contains(h, want) {
			t.Errorf("the report lacks %q", want)
		}
	}
	if strings.Contains(h, "まだ記録できません") {
		t.Error("the report still says judgments cannot be kept")
	}

	// A file for the device, for every device or for a group of it is enough.
	// Only that the file is there counts.
	write := func(rel string) {
		t.Helper()
		if err := os.WriteFile(e.Proj.Path(rel), []byte("!\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	for _, name := range []string{"rt-b01.j2", "_all.j2"} {
		write("extra/" + name)
		if j = review(t, e, "old"); len(j.NoExtra) != 0 {
			t.Errorf("with extra/%s: %+v", name, j.NoExtra)
		}
		os.Remove(e.Proj.Path("extra/" + name))
	}
	// Files that are no extra configuration do not count.
	write("extra/rt-b01.txt")
	write("extra/.rt-b01.j2")
	write("extra/rt-a01.j2") // the old name
	if j = review(t, e, "old"); len(j.NoExtra) != 1 {
		t.Errorf("files that are no extra configuration counted: %+v", j.NoExtra)
	}
}

func TestJudgeStaleAndFaults(t *testing.T) {
	e := judgeCase(t)
	gone := project.Judgment{Pack: "cisco_ios", Shape: "ip sla responder", Devices: "*", Verdict: "hold", By: "admin", At: "2026-01-02T03:04:05Z"}
	other := project.Judgment{Source: "new", Pack: "cisco_ios", Shape: "ip sla <n>", Devices: "*", Verdict: "hold", By: "admin", At: "2026-01-02T03:04:05Z"}
	away := project.Judgment{Pack: "cisco_ios", Shape: "ip sla <n>", Devices: "rt-z99", Verdict: "hold", By: "admin", At: "2026-01-02T03:04:05Z"}
	if err := e.Proj.SaveJudgments([]project.Judgment{gone, other, away}); err != nil {
		t.Fatal(err)
	}
	// A judgment no line takes is kept and told; one of the other side is not
	// looked at.
	j := review(t, e, "old")
	if len(j.Stale) != 2 || j.Stale[0].Shape != "ip sla responder" || j.Stale[1].Devices != "rt-z99" || j.Open != 12 {
		t.Errorf("stale = %+v, open %d", j.Stale, j.Open)
	}
	o, err := e.RunImport("old", "", at)
	if err != nil {
		t.Fatal(err)
	}
	if h := read(t, e, o.Report.HTML); !strings.Contains(h, "当たる行がありません") || !strings.Contains(h, "ip sla responder") {
		t.Error("the report does not tell the judgment no line takes")
	}
	if list, _ := e.Proj.Judgments(); len(list) != 3 {
		t.Errorf("a judgment was dropped: %+v", list)
	}

	// A file that is written wrong stops the import and the judging, and
	// tells where.
	if err := os.WriteFile(e.Proj.Path("state/judgments.yaml"), []byte("judgments:\n  - pack: cisco_ios\n    shape: ip sla <n>\n    devices: \"*\"\n    verdict: not_needed\n    by: admin\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := e.RunImport("old", "", at); err == nil || !strings.Contains(err.Error(), "state/judgments.yaml の 1 件目") || !strings.Contains(err.Error(), "reason") {
		t.Errorf("import with a wrong judgment: %v", err)
	}
	if _, err := e.Review("old"); err == nil {
		t.Error("review with a wrong judgment")
	}
}

func TestJudgeNewSide(t *testing.T) {
	e := newCase(t, map[string]string{
		"input/old/rt-a01.txt": judgeFile(t, "rt-a01.txt"),
		"input/new/rt-a02.txt": judgeFile(t, "rt-a02.txt"),
	})
	old, fresh := groupOf(t, review(t, e, "old"), "ip sla <n>"), groupOf(t, review(t, e, "new"), "ip sla <n>")
	if old.ID == fresh.ID {
		t.Fatal("the groups of the two sides have one ID")
	}
	// A judgment is of its side.
	o := judgeLines(t, e, JudgeRequest{Source: "new", Group: fresh.ID, Verdict: "manual", By: "admin"})
	if o.Devices != 1 || o.Lines != 2 {
		t.Errorf("judged = %+v", o)
	}
	// On the new side the device is the host: no book is needed for M-402.
	if j := o.Judging; j.HostsUnknown || !reflect.DeepEqual(j.NoExtra, []NoExtra{{Device: "rt-a02", Host: "rt-a02", Lines: 2}}) {
		t.Errorf("no extra = %+v", j)
	}
	if j := review(t, e, "old"); j.Manual != 0 || len(j.Stale) != 0 {
		t.Errorf("the old side took the judgment of the new side: %+v", j)
	}
	n, err := e.RunImport("new", "", at)
	if err != nil {
		t.Fatal(err)
	}
	if n.Judging.Manual != 2 || n.Findings() < n.Judging.Open+1 {
		t.Errorf("import of the new side: %+v, findings %d", n.Judging, n.Findings())
	}
}
