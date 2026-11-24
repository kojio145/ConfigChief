package ops

import (
	"bytes"
	"io/fs"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/xuri/excelize/v2"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/packs"
	"github.com/kojio145/configchief/internal/project"
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

// genCase makes a case whose book was made from configurations in input/new.
func genCase(t *testing.T, files map[string]string) *Env {
	t.Helper()
	e := newCase(t, files)
	if _, err := e.MakeBook("new", false, at); err != nil {
		t.Fatal(err)
	}
	return e
}

func exists(e *Env, rel string) bool {
	_, err := os.Stat(e.Proj.Path(rel))
	return err == nil
}

func stateOf(o *GenerateOutcome, host string) string {
	for _, d := range o.Devices {
		if d.Host == host {
			return d.State
		}
	}
	return ""
}

func itemsOf(o *GenerateOutcome, rule string) []Judged {
	var out []Judged
	for _, f := range o.Items {
		if f.RuleID == rule {
			out = append(out, f)
		}
	}
	return out
}

func statOf(t *testing.T, e *Env, o *GenerateOutcome, rule string) RuleStat {
	t.Helper()
	for _, s := range e.RuleStats(o) {
		if s.Rule.ID == rule {
			return s
		}
	}
	t.Fatalf("no line for %s among the checks", rule)
	return RuleStat{}
}

const outDir = "out/2026-01-02_030405/"

func TestGenerate(t *testing.T) {
	want := rendered(t, "cisco_ios")
	e := genCase(t, map[string]string{"input/new/rt-c01.txt": want})
	e.Version = "1.2.3"
	o, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	if len(o.Devices) != 1 || o.Devices[0].State != GenWritten || o.Devices[0].File != outDir+"configs/rt-c01.txt" {
		t.Fatalf("devices = %+v", o.Devices)
	}
	// The book came from this text: the text comes back.
	got := strings.ReplaceAll(read(t, e, o.Devices[0].File), "\r\n", "\n")
	if got != want {
		t.Errorf("the configuration is not the one the book came from:\n%s", got)
	}
	if o.Findings() != 0 || o.Unverified != "" || exists(e, outDir+"UNVERIFIED.txt") {
		t.Errorf("findings = %d, mark %q: %+v", o.Findings(), o.Unverified, o.Items)
	}
	if want := (Files{reports + "check_results.xlsx", reports + "check_results.html"}); o.Report != want {
		t.Errorf("report = %+v", o.Report)
	}
	// The first generation has nothing to compare with.
	ch := o.Changes
	if ch == nil || ch.Before != "" || ch.Snapshot != "state/snapshots/2026-01-02_030405.json" || ch.Report != (Files{}) {
		t.Errorf("changes = %+v", ch)
	}
	if exists(e, reports+"changes.xlsx") {
		t.Error("a list of changes without a generation before")
	}

	h := read(t, e, o.Report.HTML)
	for _, want := range []string{"チェック結果表", "ConfigChief 1.2.3", "rt-c01", "書き出し済み", "cisco_ios", "入力 parameters.xlsx",
		"G-610", "未実施", "パラメータ突合", "hostname rt-c01"} {
		if !strings.Contains(h, want) {
			t.Errorf("the check results lack %q", want)
		}
	}
	// The secrets are in the configuration and nowhere else.
	snap := read(t, e, ch.Snapshot)
	secrets := secretValues(e, o.data)
	if len(secrets) == 0 {
		t.Fatal("the sample has no secret")
	}
	for _, s := range secrets {
		if !strings.Contains(got, s) {
			t.Errorf("the configuration lacks the secret %q", s)
		}
		if strings.Contains(h, s) || strings.Contains(snap, s) {
			t.Errorf("the secret %q is in the check results or in the snapshot", s)
		}
	}

	f, err := excelize.OpenFile(e.Proj.Path(o.Report.XLSX))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if got, want := f.GetSheetList(), []string{"表紙", "機器別", "チェック項目", "指摘", "承認", "パラメータ突合"}; !reflect.DeepEqual(got, want) {
		t.Errorf("sheets = %v", got)
	}
	for _, name := range f.GetSheetList() {
		rows, err := f.GetRows(name)
		if err != nil {
			t.Fatal(err)
		}
		for _, row := range rows {
			for _, cell := range row {
				for _, s := range secrets {
					if strings.Contains(cell, s) {
						t.Errorf("the secret %q is in the sheet %s", s, name)
					}
				}
			}
		}
	}

	// Every check has its line, and one that did not run is not called fine.
	stats := e.RuleStats(o)
	if len(stats) != len(check.Rules()) {
		t.Errorf("%d lines for %d checks", len(stats), len(check.Rules()))
	}
	for _, id := range []string{"V-111", "G-607", "G-610"} {
		if s := statOf(t, e, o, id); s.State != wordNotRun || s.Note == "" {
			t.Errorf("%s = %+v", id, s)
		}
	}
	if s := statOf(t, e, o, "G-604"); s.State != wordOK || s.Total != 1 || s.OK != 1 {
		t.Errorf("G-604 = %+v", s)
	}
	if log := read(t, e, "state/log.txt"); !strings.Contains(log, "generate:") {
		t.Errorf("log = %s", log)
	}
}

func TestGenerateAgain(t *testing.T) {
	e := genCase(t, map[string]string{"input/new/rt-c01.txt": rendered(t, "cisco_ios")})
	if _, err := e.Generate(GenOptions{}, at); err != nil {
		t.Fatal(err)
	}
	const secret = "dummy-backup-secret-sample-two"
	setCell(t, e, "インタフェース", "IF 名", "Loopback0", "説明", "sample loopback two")
	setCell(t, e, "ユーザ", "ユーザ名", "backup", "パスワード", secret)
	later := at.Add(time.Minute)
	o, err := e.Generate(GenOptions{}, later)
	if err != nil {
		t.Fatal(err)
	}
	ch := o.Changes
	if ch == nil || ch.Before != "2026-01-02_030405" || len(ch.Cells) != 2 || len(ch.Lines) != 4 {
		t.Fatalf("changes = %+v", ch)
	}
	for _, c := range ch.Cells {
		switch c.Column {
		case "description":
			if c.Device != "rt-c01" || c.Table != "interfaces" || c.Before != "sample loopback" || c.After != "sample loopback two" {
				t.Errorf("cell = %+v", c)
			}
		case "secret":
			// Only that the value changed is kept.
			if c.Before == c.After || strings.Contains(c.Before+c.After, "dummy") {
				t.Errorf("cell = %+v", c)
			}
		default:
			t.Errorf("cell = %+v", c)
		}
	}
	added := 0
	for _, l := range ch.Lines {
		if l.Added {
			added++
		}
		if strings.Contains(l.Text, "dummy") {
			t.Errorf("a secret in the line %+v", l)
		}
	}
	if added != 2 {
		t.Errorf("%d lines added: %+v", added, ch.Lines)
	}
	const dir = "out/2026-01-02_030505/"
	if want := (Files{dir + "reports/changes.xlsx", dir + "reports/changes.html"}); ch.Report != want {
		t.Errorf("report = %+v", ch.Report)
	}
	h := read(t, e, ch.Report.HTML)
	for _, want := range []string{"変更一覧", "変わったセル", "変わった行", "sample loopback two", "2026-01-02_030405"} {
		if !strings.Contains(h, want) {
			t.Errorf("the list of changes lacks %q", want)
		}
	}
	if strings.Contains(h, secret) || strings.Contains(read(t, e, ch.Snapshot), secret) {
		t.Error("the secret is in the list of changes or in the snapshot")
	}
	if !strings.Contains(read(t, e, dir+"configs/rt-c01.txt"), secret) {
		t.Error("the configuration lacks the new secret")
	}

	// Only the checks: no configuration and no snapshot.
	c, err := e.Generate(GenOptions{CheckOnly: true}, later.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	const checked = "out/2026-01-02_030605/"
	if c.Changes != nil || stateOf(c, "rt-c01") != GenChecked || exists(e, checked+"configs") || !exists(e, checked+"reports/check_results.html") {
		t.Errorf("check only: changes %+v, state %s", c.Changes, stateOf(c, "rt-c01"))
	}
	// A dry run writes nothing at all.
	d, err := e.Generate(GenOptions{Dry: true}, later.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if exists(e, "out/2026-01-02_030705") || d.Report != (Files{}) || len(d.Items) != len(c.Items) {
		t.Errorf("dry run: report %+v, %d findings for %d", d.Report, len(d.Items), len(c.Items))
	}
	snaps, err := os.ReadDir(e.Proj.Path(snapshotDir))
	if err != nil || len(snaps) != 2 {
		t.Errorf("%d snapshots, %v", len(snaps), err)
	}
}

func TestGenerateBlocked(t *testing.T) {
	// The NEC sample names access lists it does not define (V-106), and both
	// samples use the same addresses (V-202).
	e := genCase(t, map[string]string{
		"input/new/rt-c01.txt": rendered(t, "cisco_ios"),
		"input/new/ix-c01.txt": rendered(t, "nec_ix"),
	})
	o, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	if stateOf(o, "ix-c01") != GenBlocked || stateOf(o, "rt-c01") != GenWritten {
		t.Fatalf("devices = %+v", o.Devices)
	}
	if exists(e, outDir+"configs/ix-c01.txt") || exists(e, outDir+"configs/_failed") || !exists(e, outDir+"configs/rt-c01.txt") {
		t.Error("the files are not those of one device written and one stopped")
	}
	v106, v202 := itemsOf(o, "V-106"), itemsOf(o, "V-202")
	if c := o.Count(); len(v106) == 0 || len(v202) == 0 || c.Errors != len(v106) || c.Warnings != len(v202) {
		t.Fatalf("count = %+v, V-106 %d, V-202 %d", o.Count(), len(v106), len(v202))
	}
	// The checks of what was generated did not run for the device.
	if s := statOf(t, e, o, "G-604"); s.State != wordOK+"・"+wordPartly || s.Total != 1 || !strings.Contains(s.Note, "V-106") {
		t.Errorf("G-604 = %+v", s)
	}
	mark, err := os.ReadFile(e.Proj.Path(outDir + "UNVERIFIED.txt"))
	if err != nil || o.Unverified != outDir+"UNVERIFIED.txt" {
		t.Fatalf("mark %q: %v", o.Unverified, err)
	}
	if !bytes.HasPrefix(mark, []byte{0xEF, 0xBB, 0xBF}) || !bytes.Contains(mark, []byte("ix-c01")) || !bytes.Contains(mark, []byte("\r\n")) {
		t.Errorf("mark = %q", mark)
	}

	ask := func(r ApproveRequest) (*Approved, error) {
		r.Reason, r.By = "sample reason", "sample-user"
		return e.Approve(r, at)
	}
	// What stops a device cannot be approved, and an error not by a scope.
	for _, r := range []ApproveRequest{
		{Finding: v106[0].Key},
		{Rule: "V-106", Devices: "*"},
		{Finding: "0000000000000000"},
		{Finding: v202[0].Key, Rule: "V-202"},
		{},
		{Rule: "V-999", Devices: "*"},
		{Rule: "V-202"},
		{Rule: "V-202", Devices: "*", Table: "no such table"},
		{Rule: "V-202", Devices: "*", Column: "IPv4 アドレス"},
		{Rule: "V-202", Devices: "ix-c01"}, // the findings are of the other device
	} {
		if a, err := ask(r); err == nil {
			t.Errorf("approved %+v: %+v", r, a)
		}
	}
	if _, err := e.Approve(ApproveRequest{Finding: v202[0].Key, By: "sample-user"}, at); err == nil {
		t.Error("approved without a reason")
	}
	if list, _ := e.Proj.Approvals(); len(list) != 0 {
		t.Fatalf("approvals = %+v", list)
	}

	// One finding by the beginning of its fingerprint, the rest by a scope.
	a, err := ask(ApproveRequest{Finding: v202[0].Key[:8]})
	if err != nil || a.Hits != 1 || a.Approval.ID != "a-0001" || a.Approval.Finding != v202[0].Key || a.Approval.Rule != "V-202" {
		t.Fatalf("approved %+v, %v", a, err)
	}
	if _, err := ask(ApproveRequest{Finding: v202[0].Key}); err == nil {
		t.Error("approved twice")
	}
	b, err := ask(ApproveRequest{Rule: "V-202", Devices: "rt-c01", Table: "インタフェース"})
	if err != nil || b.Hits != len(v202) || b.Approval.ID != "a-0002" || b.Approval.Scope == nil || b.Approval.Scope.Table != "interfaces" {
		t.Fatalf("approved %+v, %v", b, err)
	}
	// Somebody wrote an approval of an error into the file by hand.
	list, err := e.Proj.Approvals()
	if err != nil {
		t.Fatal(err)
	}
	list = append(list, project.Approval{ID: "a-0003", Finding: v106[0].Key, Rule: "V-106", Reason: "sample reason", By: "sample-user", At: "2026-01-02T03:04:05Z"})
	if err := e.Proj.SaveApprovals(list); err != nil {
		t.Fatal(err)
	}

	o, err = e.Generate(GenOptions{CheckOnly: true}, at.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if c := o.Count(); c.Warnings != 0 || c.Approved != len(v202) || c.Errors != len(v106) {
		t.Errorf("count = %+v", c)
	}
	if o.Hits["a-0001"] != 1 || o.Hits["a-0002"] != len(v202) || o.Refused["a-0003"] == "" || stateOf(o, "ix-c01") != GenBlocked {
		t.Errorf("hits = %v, refused = %v", o.Hits, o.Refused)
	}
	h := read(t, e, o.Report.HTML)
	for _, want := range []string{"sample reason", "sample-user", "a-0002", wordSettled, o.Refused["a-0003"]} {
		if !strings.Contains(h, want) {
			t.Errorf("the check results lack %q", want)
		}
	}

	// Taking an approval back.
	if err := e.Unapprove("A-0001", at); err != nil {
		t.Error(err)
	}
	if err := e.Unapprove("a-0001", at); err == nil {
		t.Error("took the same approval back twice")
	}
	if list, _ := e.Proj.Approvals(); len(list) != 2 {
		t.Errorf("approvals = %+v", list)
	}
}

func TestGenerateFailed(t *testing.T) {
	e := genCase(t, map[string]string{"input/new/rt-c01.txt": rendered(t, "cisco_ios")})
	// The pack writes the name of a route as one word: two words fail.
	setCell(t, e, "スタティック経路", "宛先", "192.0.2.64/26", "説明", "sample backup")
	o, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	d := o.Devices[0]
	if d.State != GenFailed || d.File != outDir+"configs/_failed/rt-c01.txt" || d.Why == "" {
		t.Fatalf("device = %+v", d)
	}
	if exists(e, outDir+"configs/rt-c01.txt") || !exists(e, d.File) || !exists(e, outDir+"UNVERIFIED.txt") {
		t.Error("the text of a failed device is not where it belongs")
	}
	g601 := itemsOf(o, "G-601")
	if len(g601) != 1 || g601[0].Device != "rt-c01" || o.Count().Errors == 0 {
		t.Fatalf("G-601 = %+v, count %+v", g601, o.Count())
	}
	// The values that were not used and the round trip cannot be told.
	for _, id := range []string{"G-602", "G-604", "G-605"} {
		if s := statOf(t, e, o, id); s.State != wordNotRun || s.Note == "" {
			t.Errorf("%s = %+v", id, s)
		}
	}
	if _, err := e.Approve(ApproveRequest{Finding: g601[0].Key, Reason: "sample reason", By: "sample-user"}, at); err == nil {
		t.Error("approved a template that failed")
	}
	// The lines of a failed device are kept too: the next generation tells
	// what came back.
	if o.Changes == nil || !exists(e, o.Changes.Snapshot) {
		t.Errorf("changes = %+v", o.Changes)
	}
}

func TestGenerateOptions(t *testing.T) {
	e := genCase(t, map[string]string{
		"input/new/rt-c01.txt": rendered(t, "cisco_ios"),
		"input/new/ix-c01.txt": rendered(t, "nec_ix"),
	})
	if _, err := e.Generate(GenOptions{Device: "rt-z99"}, at); err == nil {
		t.Error("generated a device that is not in the book")
	}
	o, err := e.Generate(GenOptions{Device: "rt-c01"}, at)
	if err != nil {
		t.Fatal(err)
	}
	// The values of every device are still looked at.
	if stateOf(o, "rt-c01") != GenWritten || stateOf(o, "ix-c01") != GenSkipped || len(itemsOf(o, "V-106")) == 0 {
		t.Errorf("devices = %+v", o.Devices)
	}
	v208 := len(itemsOf(o, "V-208"))
	if v208 == 0 || o.Count().Info < v208 {
		t.Fatalf("V-208 = %d, count %+v", v208, o.Count())
	}

	cfg := &e.Proj.Config.Checks
	cfg.Disable = []string{"V-202", "V-106", "V-999"}
	cfg.Severity = map[string]string{"V-208": "warning", "V-998": "info"}
	o, err = e.Generate(GenOptions{CheckOnly: true}, at.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if !o.Disabled["V-202"] || o.Disabled["V-106"] || len(itemsOf(o, "V-202")) != 0 || len(itemsOf(o, "V-106")) == 0 {
		t.Errorf("disabled = %v", o.Disabled)
	}
	if c := o.Count(); c.Warnings != v208 {
		t.Errorf("count = %+v, V-208 %d", c, v208)
	}
	notes := strings.Join(o.Notes, "\n")
	for _, want := range []string{"V-106 は止められません", "checks.disable の V-999", "checks.severity の V-998"} {
		if !strings.Contains(notes, want) {
			t.Errorf("the notes lack %q:\n%s", want, notes)
		}
	}
	if s := statOf(t, e, o, "V-202"); s.State != wordNotRun || !strings.Contains(s.Note, "checks.disable") {
		t.Errorf("V-202 = %+v", s)
	}
}

func TestGenerateExtra(t *testing.T) {
	e := genCase(t, map[string]string{"input/new/rt-c01.txt": rendered(t, "cisco_ios")})
	if err := writeTestFile(e.Proj.Path("extra/rt-c01.j2"), []byte("service sequence-numbers\n!\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// A file for a device that is not there is told about, not dropped in silence.
	if err := writeTestFile(e.Proj.Path("extra/rt-z99.j2"), []byte("service sequence-numbers\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	o, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	text := read(t, e, outDir+"configs/rt-c01.txt")
	if stateOf(o, "rt-c01") != GenWritten || !strings.Contains(text, "service sequence-numbers") {
		t.Errorf("state %s:\n%s", stateOf(o, "rt-c01"), text)
	}
	if notes := strings.Join(o.Notes, "\n"); !strings.Contains(notes, "extra/rt-z99.j2") {
		t.Errorf("notes = %s", notes)
	}
	h := read(t, e, o.Report.HTML)
	for _, want := range []string{"入力 extra/rt-c01.j2", "手書き"} {
		if !strings.Contains(h, want) {
			t.Errorf("the check results lack %q", want)
		}
	}
}

func TestGenerateWithoutBook(t *testing.T) {
	e := newCase(t, nil)
	if _, err := e.Generate(GenOptions{}, at); err == nil {
		t.Error("generated without a parameter book")
	}
}
