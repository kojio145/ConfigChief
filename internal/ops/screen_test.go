package ops

import (
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/kojio145/configchief/internal/importer"
)

// The lines for a screen mask the secrets even when the reports do not.
func TestLines(t *testing.T) {
	e := newCase(t, map[string]string{"input/old/rt-a01.txt": sample(t, "cisco_ios")})
	e.Proj.Config.Secrets.MaskInReports = false
	src, err := e.Import("old", "")
	if err != nil {
		t.Fatal(err)
	}
	f := src.Files[0]
	lines := e.Lines(f)
	if len(lines) != len(f.Result.Lines) {
		t.Fatalf("lines = %d, want %d", len(lines), len(f.Result.Lines))
	}
	secrets := secretValues(e, src.Data)
	if len(secrets) < 5 {
		t.Fatalf("secrets of the sample = %q", secrets)
	}
	var texts []string
	hostname, values, blanks := false, 0, 0
	for i, l := range lines {
		if l.No != i+1 {
			t.Fatalf("line %d has the number %d", i+1, l.No)
		}
		texts = append(texts, l.Text)
		for _, m := range l.Made {
			texts = append(texts, m.Where, m.Value)
			if m.Value != "" {
				values++
			}
		}
		if l.Blank {
			blanks++
			if l.State != importer.Ignored {
				t.Errorf("line %d: a blank line with the state %q", l.No, l.State)
			}
		}
		if l.Text == "hostname rt-a01" {
			hostname = true
			if l.State != importer.Taken || len(l.Made) == 0 {
				t.Errorf("the line of the host name: %+v", l)
			}
		}
		if l.State != importer.Taken && len(l.Made) != 0 {
			t.Errorf("line %d is not taken, but made a value: %q", l.No, l.Text)
		}
	}
	if !hostname || values == 0 || blanks == 0 {
		t.Errorf("host name line %v, values %d, blanks %d", hostname, values, blanks)
	}
	all := strings.Join(texts, "\n")
	for _, v := range secrets {
		if strings.Contains(all, v) {
			t.Errorf("the screen shows the secret %q", v)
		}
	}
	if !strings.Contains(all, "snmp-server community *** RO 10") {
		t.Error("the masked line is not as in the report")
	}
	// The environment itself still follows the case.
	if e.mask() {
		t.Error("making the lines of a screen turned masking on for the reports")
	}
	if e.Lines(nil) != nil || e.Lines(&File{Err: "x"}) != nil {
		t.Error("lines for a file that was not imported")
	}
}

func TestChoosePack(t *testing.T) {
	e := newCase(t, map[string]string{
		"input/old/a_cisco.txt": sample(t, "cisco_ios"),
		"input/old/d_short.txt": "hostname sample-short\n", // too short to tell the vendor from
	})
	if err := e.ChoosePack("input/old/d_short.txt", "sample_os", at); err == nil || !strings.Contains(err.Error(), "使えるパック") {
		t.Errorf("err = %v", err)
	}
	if err := e.ChoosePack("input/old/none.txt", "nec_ix", at); err == nil || !strings.Contains(err.Error(), "ありません") {
		t.Errorf("err = %v", err)
	}
	if _, err := os.Stat(e.Proj.Path("state/sources.yaml")); err == nil {
		t.Error("a choice that failed was written")
	}
	if err := e.ChoosePack("input/old/d_short.txt", "nec_ix", at); err != nil {
		t.Fatal(err)
	}
	src, err := e.Import("old", "")
	if err != nil {
		t.Fatal(err)
	}
	if d := src.Files[1]; d.Pack != "nec_ix" || !d.Chosen || d.Result == nil {
		t.Errorf("the file after the choice = %+v", d)
	}
	if log := read(t, e, "state/log.txt"); !strings.Contains(log, "input/old/d_short.txt をパック nec_ix で読みます") {
		t.Errorf("log = %q", log)
	}
}

func TestMapping(t *testing.T) {
	e := newCase(t, map[string]string{"input/old/rt-a01.txt": sample(t, "cisco_ios")})
	m, err := e.Mapping()
	if err != nil {
		t.Fatal(err)
	}
	if m.Book != nil || m.Applied != nil || len(m.Source.Files) != 1 {
		t.Errorf("without a book: %+v", m)
	}
	if _, err := os.Stat(e.Proj.BookPath()); err == nil {
		t.Error("looking at the mapping made a book")
	}
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	if m, err = e.Mapping(); err != nil {
		t.Fatal(err)
	}
	if m.Book == nil || len(m.Book.Devices) != 1 || m.Applied == nil || len(m.Applied.NoReason) != 1 {
		t.Fatalf("with a book: %+v", m)
	}
	s := e.MappingSheet(m.Applied, m.Mig, false)
	if len(s.Rows) != 1 || s.Rows[0][0] != "M-407" || s.Rows[0][1] != "rt-a01" {
		t.Errorf("open parts = %v", s.Rows)
	}

	empty := newCase(t, nil)
	if _, err := empty.Mapping(); err == nil || !strings.Contains(err.Error(), "input/old にファイルがありません") {
		t.Errorf("err = %v", err)
	}
}

// The comparison for a screen masks the secrets even when the report does
// not.
func TestCompareDoc(t *testing.T) {
	e := newCase(t, map[string]string{
		"input/old/rt-a01.txt": sample(t, "cisco_ios"),
		"input/new/ix-a01.txt": sample(t, "nec_ix"),
	})
	e.Proj.Config.Secrets.MaskInReports = false
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	setCell(t, e, "新旧対照表（機器）", "旧ホスト名", "rt-a01", "新ホスト名", "ix-a01")
	o, err := e.Compare("old", "new", at)
	if err != nil {
		t.Fatal(err)
	}
	d := e.CompareDoc(o, at)
	if len(d.Sheets) < 3 || d.Sheets[0].Name != "A にだけある行" || d.Sheets[2].Name != "値が違うセル" {
		t.Fatalf("sheets = %+v", d.Sheets)
	}
	var texts []string
	for _, s := range d.Sheets {
		for _, r := range s.Rows {
			texts = append(texts, r...)
		}
	}
	all := strings.Join(texts, "\n")
	secrets := append(secretValues(e, o.A.Data), secretValues(e, o.B.Data)...)
	if len(secrets) < 5 {
		t.Fatalf("secrets = %q", secrets)
	}
	for _, v := range secrets {
		if strings.Contains(all, v) {
			t.Errorf("the screen shows the secret %q", v)
		}
	}
	if !strings.Contains(all, "***") {
		t.Error("nothing is masked")
	}
	// The report holds the secrets of the rows that differ; which ones those
	// are is not the same from run to run of the list above.
	h, shown := read(t, e, o.Report.HTML), 0
	for _, v := range secrets {
		if strings.Contains(h, v) {
			shown++
		}
	}
	if shown == 0 {
		t.Error("masking is off for the reports, but the report holds no secret")
	}
}

// An import that is stopped writes nothing.
func TestTick(t *testing.T) {
	e := newCase(t, map[string]string{
		"input/old/a_cisco.txt": sample(t, "cisco_ios"),
		"input/old/b_nec.txt":   sample(t, "nec_ix"),
	})
	var seen []string
	e.Tick = func(done, total int, file string) error {
		if total != 2 || done != len(seen) {
			t.Errorf("tick %d of %d after %d files", done, total, len(seen))
		}
		seen = append(seen, file)
		return nil
	}
	if _, err := e.Import("old", ""); err != nil {
		t.Fatal(err)
	}
	if strings.Join(seen, ",") != "input/old/a_cisco.txt,input/old/b_nec.txt" {
		t.Errorf("ticks = %v", seen)
	}
	stop := errors.New("止めました")
	e.Tick = func(done, total int, file string) error {
		if done == 1 {
			return stop
		}
		return nil
	}
	if _, err := e.RunImport("old", "nec_ix", at); !errors.Is(err, stop) {
		t.Errorf("err = %v", err)
	}
	for _, rel := range []string{e.Proj.Config.Book, "state/sources.yaml"} {
		if _, err := os.Stat(e.Proj.Path(rel)); err == nil {
			t.Errorf("a stopped import wrote %s", rel)
		}
	}
	if dirs, _ := os.ReadDir(e.Proj.Path("out")); len(dirs) != 0 {
		t.Error("a stopped import wrote a report")
	}
}

// What a screen shows of a generation: the time of the last one, where a
// finding points, and the numbers of the check results.
func TestResultForScreen(t *testing.T) {
	e := genCase(t, map[string]string{
		"input/new/rt-c01.txt": rendered(t, "cisco_ios"),
		"input/new/ix-c01.txt": rendered(t, "nec_ix"),
	})
	if got := e.LastGeneration(); got != "" {
		t.Errorf("last generation before the first = %q", got)
	}
	o, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	if got := e.LastGeneration(); got != "2026-01-02 03:04:05" {
		t.Errorf("last generation = %q", got)
	}
	// The checks alone are not a generation.
	if _, err := e.Generate(GenOptions{CheckOnly: true}, at.Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	if got := e.LastGeneration(); got != "2026-01-02 03:04:05" {
		t.Errorf("last generation after the checks alone = %q", got)
	}

	var rows []string
	for _, f := range itemsOf(o, "V-106") {
		p := e.FindingPlace(o, f.Finding)
		if p.Table == "" || p.Column == "" || p.Text != "" {
			t.Errorf("place = %+v", p)
		}
		rows = append(rows, p.Table+"/"+p.Row+"/"+p.Column)
	}
	all := strings.Join(rows, "\n")
	if !strings.Contains(all, "インタフェース/IF 名=GigaEthernet0.0/入力 ACL") {
		t.Errorf("places = %s", all)
	}
	// The key of a row that is a secret is not shown, whatever the case says
	// about its reports.
	e.Proj.Config.Secrets.MaskInReports = false
	for _, f := range itemsOf(o, "V-106") {
		if p := e.FindingPlace(o, f.Finding); p.Table == "SNMP コミュニティ" && !strings.Contains(p.Row, "伏せ字") {
			t.Errorf("the row of a secret = %q", p.Row)
		}
	}
	if !strings.Contains(all, "SNMP コミュニティ/") {
		t.Errorf("places = %s", all)
	}

	d := e.CheckDoc(o, at)
	if len(d.Counts) == 0 || len(d.Sheets) == 0 {
		t.Fatalf("doc = %+v", d)
	}
	if e.ChangesDoc(o, at) != nil {
		t.Error("changes without a generation before")
	}
	again, err := e.Generate(GenOptions{}, at.Add(2*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if c := e.ChangesDoc(again, at); c == nil || len(c.Counts) == 0 {
		t.Errorf("changes of the second generation = %+v", c)
	}
}
