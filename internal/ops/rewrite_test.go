package ops

import (
	"os"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/model"
)

// crossCase takes the Cisco sample to the NEC IX pack as TestCrossVendor
// does, up to the point where the values in a form the new vendor does not
// have are mended: what is left are the cells the new pack cannot write
// (G-602) and the values changed on the way (M-405). The old configuration
// has a line no rule takes.
func crossCase(t *testing.T) *Env {
	t.Helper()
	old := rendered(t, "cisco_ios") + "sample-unknown-command 1\n"
	e := newCase(t, map[string]string{"input/old/rt-c01.txt": old})
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	setCell(t, e, "新旧対照表（機器）", "旧ホスト名", "rt-c01", "新ホスト名", "ix-c01")
	setCell(t, e, "新旧対照表（機器）", "旧ホスト名", "rt-c01", "新メーカー", "nec_ix")
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	for old, name := range map[string]string{
		"GigabitEthernet0/0":    "GigaEthernet0.0",
		"GigabitEthernet0/1":    "GigaEthernet1.0",
		"GigabitEthernet0/2":    "-",
		"GigabitEthernet0/2.30": "GigaEthernet1.1",
		"GigabitEthernet0/3":    "GigaEthernet2.0",
		"Loopback0":             "Loopback0.0",
	} {
		setCell(t, e, "新旧対照表（ポート）", "旧 IF 名", old, "新 IF 名", name)
	}
	if _, err := e.MakeBook("old", false, at); err != nil {
		t.Fatal(err)
	}
	setCell(t, e, "機器", "ホスト名", "ix-c01", "時刻帯", "+9")
	setCell(t, e, "インタフェース", "IF 名", "GigaEthernet1.0", "代理 ARP", "")
	for _, name := range []string{"admin", "monitor"} {
		setCell(t, e, "ユーザ", "ユーザ名", name, "パスワード", "dummy-"+name+"-secret-sample")
		setCell(t, e, "ユーザ", "ユーザ名", name, "パスワードの形", "plain")
	}
	setCell(t, e, "ユーザ", "ユーザ名", "monitor", "権限", "monitor")
	return e
}

// countKinds counts the lines of a report by kind.
func countKinds(rw *Rewrite) map[string]int {
	n := map[string]int{}
	for _, l := range rw.Lines {
		n[l.Kind]++
	}
	return n
}

// Every line of the old configuration is in one kind of the rewriting
// report, and each kind has lines (spec 10.4, 13.4).
func TestRewriteReport(t *testing.T) {
	e := crossCase(t)
	o, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	if o.Devices[0].State != GenWritten {
		t.Fatalf("the device: %+v", o.Devices[0])
	}
	if o.RewriteReport.XLSX != reports+"rewrite.xlsx" || !exists(e, reports+"rewrite.xlsx") || !exists(e, reports+"rewrite.html") {
		t.Fatalf("the report: %+v", o.RewriteReport)
	}
	if !o.HasRewrite() {
		t.Fatal("no rewriting report")
	}

	// G-607 is made by another part of the work and may not run yet: one
	// finding on the NTP servers stands for it.
	t607 := e.Schema.Table("ntp_servers")
	var ntp *model.Row
	for _, r := range o.data.Tables["ntp_servers"].Rows {
		ntp = r
		break
	}
	if ntp == nil {
		t.Fatal("no NTP server in the book")
	}
	o.Items = append(o.Items, Judged{Finding: check.New("G-607", check.Error, "ix-c01", check.RefOf(t607, ntp, ""), "sample: the model has no such function")})

	rw := e.rewriteOf(o)
	src := o.rewrite.src
	lines := len(src.Files[0].Result.Lines)
	if len(rw.Lines) != lines || rw.Total != lines {
		t.Fatalf("%d lines in the report, %d counted, the old configuration has %d", len(rw.Lines), rw.Total, lines)
	}
	seen := map[int]bool{}
	sum := 0
	for _, l := range rw.Lines {
		if seen[l.Line] {
			t.Errorf("line %d is in the report twice", l.Line)
		}
		seen[l.Line] = true
		if rewriteWords[l.Kind] == "" {
			t.Errorf("line %d has no kind: %+v", l.Line, l)
		}
	}
	for _, k := range RewriteKinds {
		sum += rw.Counts[k]
	}
	if sum != lines {
		t.Errorf("the kinds add up to %d, want %d", sum, lines)
	}
	n := countKinds(rw)
	for _, k := range []string{RwAuto, RwUnread, RwCannot, RwModel, RwChanged, RwIgnored} {
		if n[k] == 0 {
			t.Errorf("no line is %s", RewriteWord(k))
		}
	}
	if os.Getenv("REWRITE_DUMP") != "" {
		for _, l := range rw.Lines {
			t.Logf("%d %s | %s | %s | %v %q | %s | %s", l.Line, l.Text, RewriteWord(l.Kind), l.To, l.NewLines, l.NewText, l.Verdict, l.Reason)
		}
		for _, l := range rw.NewOnly {
			t.Logf("new only %d %s", l.Line, l.Text)
		}
	}

	find := func(text string) RewriteLine {
		t.Helper()
		for _, l := range rw.Lines {
			if strings.TrimSpace(l.Text) == text {
				return l
			}
		}
		t.Fatalf("no line %q", text)
		return RewriteLine{}
	}
	if l := find("sample-unknown-command 1"); l.Kind != RwUnread || l.Verdict != notJudged || !l.Open {
		t.Errorf("the unknown line: %+v", l)
	}
	if l := find("hostname rt-c01"); l.Kind != RwAuto || l.NewDevice != "ix-c01" || !strings.Contains(l.NewText, "ix-c01") {
		t.Errorf("the host name: %+v", l)
	}
	for _, l := range rw.Lines {
		switch {
		case strings.HasPrefix(strings.TrimSpace(l.Text), "clock timezone"):
			if l.Kind != RwChanged || !strings.Contains(l.Verdict, "M-405") {
				t.Errorf("the time zone: %+v", l)
			}
		case strings.HasPrefix(strings.TrimSpace(l.Text), "ip domain name"):
			if l.Kind != RwCannot || !l.Open {
				t.Errorf("the domain name: %+v", l)
			}
		case strings.TrimSpace(l.Text) == "ntp server 192.0.2.123 prefer":
			if l.Kind != RwModel {
				t.Errorf("the NTP server: %+v", l)
			}
		case strings.HasPrefix(strings.TrimSpace(l.Text), "interface GigabitEthernet0/2") && !strings.Contains(l.Text, "."):
			if l.Kind != RwIgnored || !strings.Contains(l.Reason, "- にした") {
				t.Errorf("the port left behind: %+v", l)
			}
		}
	}

	// The report says the same.
	doc := e.rewriteDoc(rw, at)
	if doc.Sheets[1].Name != "行の一覧" || len(doc.Sheets[1].Rows) != lines {
		t.Errorf("the list of lines: %s, %d rows", doc.Sheets[1].Name, len(doc.Sheets[1].Rows))
	}

	// A screen masks the secrets whatever the case says.
	shown := e.Rewrite(o)
	for _, l := range shown.Lines {
		if strings.Contains(l.NewText, "dummy-admin-secret-sample") {
			t.Errorf("a secret on a screen: %+v", l)
		}
	}
}

// A device that is not generated leaves its lines to be looked at, and a
// case with no old configuration has no rewriting report.
func TestRewriteNotGenerated(t *testing.T) {
	e := crossCase(t)
	// The time zone in the form of the old vendor stops the templates.
	setCell(t, e, "機器", "ホスト名", "ix-c01", "時刻帯", "JST 9")
	o, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	rw := e.rewriteOf(o)
	n := countKinds(rw)
	if o.Devices[0].State == GenWritten || n[RwUnsure] == 0 {
		t.Errorf("state %s, kinds %v", o.Devices[0].State, n)
	}

	plain := genCase(t, map[string]string{"input/new/rt-c01.txt": rendered(t, "cisco_ios")})
	o, err = plain.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	if o.HasRewrite() || o.RewriteReport.XLSX != "" || exists(plain, reports+"rewrite.xlsx") {
		t.Errorf("a case of a new build has a rewriting report: %+v", o.RewriteReport)
	}
}

// The lines of a device left out of the renewal are ignored, and those of a
// file that could not be read are lines not taken: no line is dropped.
func TestRewriteEveryFile(t *testing.T) {
	e := crossCase(t)
	other := strings.Replace(rendered(t, "cisco_ios"), "hostname rt-c01", "hostname rt-c09", 1)
	files := map[string]string{
		"input/old/rt-c09.txt": other,
		"input/old/notes.txt":  "sample text that no pack can tell\n\nsecond sample line\n",
	}
	for name, text := range files {
		if err := writeTestFile(e.Proj.Path(name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	o, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	rw := e.rewriteOf(o)
	per := map[string]map[string]int{}
	for _, l := range rw.Lines {
		if per[l.File] == nil {
			per[l.File] = map[string]int{}
		}
		per[l.File][l.Kind]++
	}
	if n := per["input/old/rt-c09.txt"]; n[RwIgnored] != len(strings.Split(strings.TrimSuffix(other, "\n"), "\n")) || len(n) != 1 {
		t.Errorf("the device left out: %v", n)
	}
	if n := per["input/old/notes.txt"]; n[RwUnread] != 2 || n[RwIgnored] != 1 {
		t.Errorf("the file not read: %v", n)
	}
	if len(rw.Devices) != 3 {
		t.Errorf("%d devices", len(rw.Devices))
	}
}
