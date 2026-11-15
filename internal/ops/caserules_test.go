package ops

import (
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kojio145/configchief/internal/check"
)

// The rules of the case of the tests below.
const caseRulesYAML = `rules:
  - id: P-001
    title: SSH だけを使う
    severity: error
    under: '^line vty '
    require_line: '^transport input ssh$'
  - id: P-002
    title: 閉じたインターフェースを残さない
    severity: warning
    under: '^interface '
    forbid_line: '^shutdown$'
  - id: P-003
    title: NTP サーバは全台で同じ
    severity: warning
    equal_across: {table: ntp_servers, column: server}
  - id: P-004
    title: 特権のパスワードの行
    severity: info
    devices: rt-c01
    forbid_line: '^enable secret '
  - id: P-005
    title: 別のメーカーだけのルール
    severity: error
    devices: vendor:nec_ix
    require_line: '^no such line$'
  - id: P-006
    title: どの機器にも当たらない
    severity: error
    devices: rt-z99
    require_line: '^hostname '
`

// caseOfRules makes a case of two devices that are generated and one whose
// values are wrong, with rules of its own.
func caseOfRules(t *testing.T) *Env {
	t.Helper()
	first := rendered(t, "cisco_ios")
	// The second device is the first under another name, with one NTP
	// server less.
	second := strings.Replace(first, "hostname rt-c01", "hostname rt-c02", 1)
	lines := strings.Split(second, "\n")
	kept := lines[:0]
	servers := 0
	for _, l := range lines {
		if strings.HasPrefix(l, "ntp server ") {
			if servers++; servers == 2 {
				continue
			}
		}
		kept = append(kept, l)
	}
	if servers != 2 {
		t.Fatalf("the sample has %d NTP servers", servers)
	}
	return genCase(t, map[string]string{
		"input/new/rt-c01.txt": first,
		"input/new/rt-c02.txt": strings.Join(kept, "\n"),
		"input/new/ix-c01.txt": rendered(t, "nec_ix"),
		"rules/case.yaml":      caseRulesYAML,
	})
}

func itemsOfCase(o *GenerateOutcome, id string) []Judged {
	var out []Judged
	for _, f := range o.Items {
		if f.CaseRule == id {
			out = append(out, f)
		}
	}
	return out
}

func TestGenerateCaseRules(t *testing.T) {
	e := caseOfRules(t)
	o, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	if stateOf(o, "rt-c01") != GenWritten || stateOf(o, "rt-c02") != GenWritten || stateOf(o, "ix-c01") != GenBlocked {
		t.Fatalf("devices = %+v", o.Devices)
	}
	if len(o.CaseRules) != 6 {
		t.Fatalf("%d rules read", len(o.CaseRules))
	}

	// The lines: one finding for each device, at its line.
	p2 := itemsOfCase(o, "P-002")
	if len(p2) != 2 || p2[0].Device != "rt-c01" || p2[1].Device != "rt-c02" {
		t.Fatalf("P-002 = %+v", p2)
	}
	for _, f := range p2 {
		if f.RuleID != "G-611" || f.Severity != check.Warning || f.Line == 0 || f.Evidence[1].Value != "shutdown" ||
			!strings.HasPrefix(f.Evidence[2].Value, "interface ") || !strings.HasPrefix(f.Message, "P-002 閉じたインターフェースを残さない: ") {
			t.Errorf("P-002: %+v", f.Finding)
		}
		for _, d := range o.Devices {
			if d.Host == f.Device && strings.TrimSpace(d.Output.Lines[f.Line-1].Text) != "shutdown" {
				t.Errorf("P-002 points at line %d of %s: %q", f.Line, d.Host, d.Output.Lines[f.Line-1].Text)
			}
		}
	}
	if p2[0].Key == p2[1].Key {
		t.Error("P-002: the same key on two devices")
	}
	// The rows: the device whose servers differ. The device whose values
	// are wrong is not compared.
	p3 := itemsOfCase(o, "P-003")
	if len(p3) != 1 || p3[0].RuleID != "G-612" || p3[0].Device != "rt-c02" || p3[0].Ref.Table != "ntp_servers" {
		t.Fatalf("P-003 = %+v", p3)
	}
	// A line with a secret: the finding shows it covered.
	p4 := itemsOfCase(o, "P-004")
	if len(p4) != 1 || p4[0].Device != "rt-c01" || p4[0].Severity != check.Info || !strings.Contains(p4[0].Evidence[1].Value, "＊＊＊") {
		t.Fatalf("P-004 = %+v", p4)
	}
	if n := len(itemsOfCase(o, "P-001")) + len(itemsOfCase(o, "P-005")) + len(itemsOfCase(o, "P-006")); n != 0 {
		t.Errorf("%d findings of the rules that hold", n)
	}

	// What each rule came to, after the checks of the tool.
	for id, want := range map[string]RuleStat{
		"P-001": {State: wordOK + "・" + wordPartly, Unit: "台", Total: 2, OK: 2, Under: "G-610"},
		"P-002": {State: wordFound + "・" + wordPartly, Unit: "台", Total: 2, Open: 2, Findings: 2, Under: "G-611"},
		"P-003": {State: wordFound + "・" + wordPartly, Unit: "件", Total: 2, OK: 1, Open: 1, Findings: 1, Under: "G-612"},
		"P-004": {State: wordFound, Unit: "台", Total: 1, Open: 1, Findings: 1, Under: "G-611"},
		"P-005": {State: wordNotRun, Unit: "台", Under: "G-610"},
		"P-006": {State: wordNothing, Unit: "台", Under: "G-610"},
		"G-610": {State: wordOK + "・" + wordPartly, Unit: "台", Total: 2, OK: 2},
		"G-611": {State: wordFound + "・" + wordPartly, Unit: "台", Total: 2, Open: 2, Findings: 3},
		"G-612": {State: wordFound + "・" + wordPartly, Unit: "件", Total: 2, OK: 1, Open: 1, Findings: 1},
	} {
		got := statOf(t, e, o, id)
		note := got.Note
		got.Rule, got.Severity, got.Note = check.Rule{}, "", ""
		if got != want {
			t.Errorf("%s = %+v (%s), want %+v", id, got, note, want)
		}
		// The rules for every device tell of the one that was left out.
		if id != "P-004" && id != "P-006" && !strings.Contains(note, "値の誤り") {
			t.Errorf("%s: the note does not tell of the device that was left out: %q", id, note)
		}
	}
	if notes := strings.Join(o.Notes, "\n"); !strings.Contains(notes, "案件のルール P-006（rules/case.yaml）の devices「rt-z99」は、どの機器にも当たりません") {
		t.Errorf("notes = %s", notes)
	}

	h := read(t, e, o.Report.HTML)
	for _, want := range []string{"入力 rules/case.yaml", "案件のルール（G-611）", "P-002 閉じたインターフェースを残さない", "NTP サーバは全台で同じ"} {
		if !strings.Contains(h, want) {
			t.Errorf("the check results lack %q", want)
		}
	}
	for _, s := range secretValues(e, o.data) {
		if strings.Contains(h, s) {
			t.Errorf("the secret %q is in the check results", s)
		}
	}
	// The mark tells that something is left to look at.
	if o.Unverified == "" {
		t.Error("no mark with findings of the rules of the case")
	}

	// A warning of a rule can be approved by a scope; the rule then counts
	// as settled.
	if _, err := e.Approve(ApproveRequest{Rule: "G-611", Devices: "*", Reason: "sample reason", By: "sample-user"}, at); err != nil {
		t.Fatal(err)
	}
	o, err = e.Generate(GenOptions{CheckOnly: true}, at.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if s := statOf(t, e, o, "P-002"); !strings.HasPrefix(s.State, wordSettled) || s.Approved != 2 || s.Open != 0 {
		t.Errorf("P-002 after the approval = %+v", s)
	}

	// What the case says about the check holds for its rules. One device
	// only: the lines of the others are not looked at, the rows are.
	cfg := &e.Proj.Config.Checks
	cfg.Disable = []string{"G-610"}
	cfg.Severity = map[string]string{"G-612": "error"}
	o, err = e.Generate(GenOptions{CheckOnly: true, Device: "rt-c01"}, at.Add(2*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if s := statOf(t, e, o, "P-001"); s.State != wordNotRun || !strings.Contains(s.Note, "checks.disable") {
		t.Errorf("P-001 with G-610 switched off = %+v", s)
	}
	if p3 := itemsOfCase(o, "P-003"); len(p3) != 1 || p3[0].Severity != check.Error || p3[0].Device != "rt-c02" {
		t.Errorf("P-003 = %+v", p3)
	}
	if s := statOf(t, e, o, "P-003"); s.Severity != check.Error {
		t.Errorf("P-003 = %+v", s)
	}
	if p2 := itemsOfCase(o, "P-002"); len(p2) != 1 || p2[0].Device != "rt-c01" {
		t.Errorf("P-002 = %+v", p2)
	}
	if s := statOf(t, e, o, "P-002"); s.Total != 1 || !strings.Contains(s.State, wordPartly) || !strings.Contains(s.Note, "--device") {
		t.Errorf("P-002 for one device = %+v", s)
	}
}

func TestGenerateCaseRuleFaults(t *testing.T) {
	// A rule that cannot be run is an error, whatever the case says.
	e := genCase(t, map[string]string{
		"input/new/rt-c01.txt": rendered(t, "cisco_ios"),
		"rules/case.yaml": `rules:
  - {id: P-001, title: 実行できない式, severity: info, table: interfaces, expr: 'row.no_such_column == 1'}
`,
	})
	e.Proj.Config.Checks.Severity = map[string]string{"G-612": "info"}
	o, err := e.Generate(GenOptions{CheckOnly: true}, at)
	if err != nil {
		t.Fatal(err)
	}
	if fs := itemsOfCase(o, "P-001"); len(fs) != 1 || fs[0].Severity != check.Error || !strings.Contains(fs[0].Message, "式を調べられません") {
		t.Errorf("P-001 = %+v", fs)
	}

	// A fault in a file of rules: nothing starts, and every fault is told.
	for name, text := range map[string]string{
		"rules/bad.yaml": "rules:\n  - {id: P-001, title: x, severity: error}\n  - {id: P-002, title: y, severity: fatal, require_line: '^end$'}\n",
		"rules/two.yaml": "rules:\n  - {id: P-001, title: x, severity: error, require_line: '^end$'}\n  - {id: P-001, title: y, severity: error, require_line: '^end$'}\n",
		"rules/old.yml":  "rules: []\n",
		"rules/not.yaml": "rules: [\n",
	} {
		e := genCase(t, map[string]string{"input/new/rt-c01.txt": rendered(t, "cisco_ios"), name: text})
		_, err := e.Generate(GenOptions{}, at)
		if err == nil || !strings.Contains(err.Error(), name) || !strings.Contains(err.Error(), "始めていません") {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if name == "rules/bad.yaml" && (!strings.Contains(err.Error(), "P-001: 種類") || !strings.Contains(err.Error(), "P-002: severity")) {
			t.Errorf("%s: not every fault is told: %v", name, err)
		}
		if exists(e, "out") && len(mustDir(t, e, "out")) != 0 {
			t.Errorf("%s: something was written", name)
		}
	}

	// Without a rule the checks say that they did not run, and why.
	e = genCase(t, map[string]string{"input/new/rt-c01.txt": rendered(t, "cisco_ios"), "rules/notes.txt": "not a rule\n"})
	o, err = e.Generate(GenOptions{CheckOnly: true}, at)
	if err != nil {
		t.Fatal(err)
	}
	for _, id := range caseChecks {
		if s := statOf(t, e, o, id); s.State != wordNotRun || s.Note != noCaseRules {
			t.Errorf("%s = %+v", id, s)
		}
	}
	for _, in := range o.Inputs {
		if strings.HasPrefix(in.Path, "rules/") {
			t.Errorf("input %s", in.Path)
		}
	}
}

func mustDir(t *testing.T, e *Env, rel string) []string {
	t.Helper()
	entries, err := os.ReadDir(e.Proj.Path(rel))
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, ent := range entries {
		names = append(names, fmt.Sprint(ent.Name()))
	}
	return names
}
