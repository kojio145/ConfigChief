package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/textdiff"
)

var treeOpt = textdiff.TreeOptions{Stamps: true, Secrets: true}

func treeDevice(t *testing.T, o *TreeOutcome, name string) *TreeDevice {
	t.Helper()
	for i := range o.Devices {
		if o.Devices[i].Device == name {
			return &o.Devices[i]
		}
	}
	var have []string
	for _, d := range o.Devices {
		have = append(have, d.Device)
	}
	t.Fatalf("no device %s in %v", name, have)
	return nil
}

// Two files, one of them a terminal log: only the configuration the log
// shows is compared, the comment lines are left out, and the numbers are
// those of the files.
func TestTreeFiles(t *testing.T) {
	e := exportCase(t)
	gen, err := e.Generate(GenOptions{Dry: true}, at)
	if err != nil {
		t.Fatal(err)
	}
	text := gen.Devices[0].Output.Text
	host := gen.Devices[0].Host
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.txt"), filepath.Join(dir, "b.txt")
	log := host + "#show clock\n10:00:00.000 JST Mon Nov 2 2026\n" + host + "#show running-config\nBuilding configuration...\n\nCurrent configuration : 2000 bytes\n" +
		strings.Replace(text, "hostname "+host+"\n", "hostname "+host+"\nservice sequence-numbers\n", 1) + host + "#exit\n"
	for p, s := range map[string]string{a: text, b: log} {
		if err := writeTestFile(p, []byte(s), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ps, err := Bundled()
	if err != nil {
		t.Fatal(err)
	}
	p, err := TreeFiles(ps, a, b, treeOpt)
	if err != nil {
		t.Fatal(err)
	}
	if p.A.Pack == "" || p.A.Log || !p.B.Log || p.B.Pack != p.A.Pack {
		t.Fatalf("A %+v, B %+v", p.A, p.B)
	}
	r := p.Result
	if r.Added != 1 || r.Removed+r.Changed+r.Moved != 0 {
		t.Fatalf("added %d removed %d changed %d moved %d", r.Added, r.Removed, r.Changed, r.Moved)
	}
	if r.Stamps != 2 || r.Comments == 0 {
		t.Fatalf("stamps %d, comments %d", r.Stamps, r.Comments)
	}
	for _, row := range r.Rows {
		if strings.Contains(row.B, "show clock") || strings.Contains(row.B, "JST") {
			t.Fatalf("a line of the log outside the configuration: %q", row.B)
		}
		if row.Op == textdiff.Add && !strings.Contains(conftreeLine(log, row.BLine), "service sequence-numbers") {
			t.Fatalf("line %d of the log is not the added line", row.BLine)
		}
	}

	// A text no pack knows is compared as it stands.
	c := filepath.Join(dir, "c.txt")
	if err := writeTestFile(c, []byte("alpha\n beta 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := filepath.Join(dir, "d.txt")
	if err := writeTestFile(d, []byte("alpha\n beta 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err = TreeFiles(ps, c, d, treeOpt)
	if err != nil {
		t.Fatal(err)
	}
	if p.A.Pack != "" || len(p.A.Notes) != 1 || p.Result.Changed != 1 {
		t.Fatalf("%+v %+v", p.A, p.Result)
	}
}

func conftreeLine(text string, n int) string {
	lines := strings.Split(text, "\n")
	if n < 1 || n > len(lines) {
		return ""
	}
	return lines[n-1]
}

// Every device of two sides: what the book gives beside a folder of logs
// taken after the work.
func TestTreeDiffSides(t *testing.T) {
	e := exportCase(t)
	gen, err := e.Generate(GenOptions{Dry: true}, at)
	if err != nil {
		t.Fatal(err)
	}
	if len(gen.Devices) < 2 {
		t.Fatalf("%d devices", len(gen.Devices))
	}
	if err := os.MkdirAll(e.Proj.Path("input/logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := e.TreeDiff("generated", "logs", treeOpt, at); err == nil || !strings.Contains(err.Error(), "input/logs にファイルがありません") {
		t.Fatalf("no log: %v", err)
	}
	first, second := gen.Devices[0], gen.Devices[1]
	// The first device shows another description under an interface and its
	// log has another name; the second device has no log; a third log is of
	// a device the book does not have.
	changed := first.Output.Text
	i := strings.Index(changed, "\ninterface ")
	if i < 0 {
		t.Fatal("no interface in the generated text")
	}
	j := i + 1 + strings.Index(changed[i+1:], "\n")
	head := strings.TrimSpace(changed[i+1 : j])
	changed = changed[:j+1] + " load-interval 30\n" + changed[j+1:]
	files := map[string]string{
		"input/logs/after-1.txt": first.Host + "#show running-config\n" + changed + first.Host + "#\n",
		"input/logs/other.txt":   strings.Replace(second.Output.Text, "hostname "+second.Host+"\n", "hostname rt-zz9\n", 1),
		"input/logs/empty.txt":   "",
	}
	for rel, text := range files {
		if err := writeTestFile(e.Proj.Path(rel), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	o, err := e.TreeDiff("generated", "logs", treeOpt, at)
	if err != nil {
		t.Fatal(err)
	}
	d := treeDevice(t, o, first.Host)
	if d.Result == nil || d.Result.Added != 1 || d.Result.Removed+d.Result.Changed+d.Result.Moved != 0 {
		t.Fatalf("%s: %+v", first.Host, d.Result)
	}
	d.Result.Walk(func(_ int, row textdiff.Row, parents []textdiff.Row) {
		if row.Op == textdiff.Add && TreeTrail(parents) != head {
			t.Errorf("the added line stands below %q, want %q", TreeTrail(parents), head)
		}
	})
	if d := treeDevice(t, o, second.Host); d.Result != nil || d.B != nil {
		t.Fatalf("%s has a partner", second.Host)
	}
	if d := treeDevice(t, o, "rt-zz9"); d.Result != nil || d.A != nil {
		t.Fatal("rt-zz9 has a partner")
	}
	if len(o.Failed) != 1 || !strings.Contains(o.Failed[0].Path, "empty.txt") {
		t.Fatalf("failed: %+v", o.Failed)
	}
	if want := len(gen.Devices) + 2; o.Left() != want {
		t.Fatalf("left %d, want %d", o.Left(), want)
	}
	for _, f := range []string{o.Report.XLSX, o.Report.HTML} {
		if _, err := os.Stat(e.Proj.Path(f)); err != nil {
			t.Fatal(err)
		}
	}
	html, _ := os.ReadFile(e.Proj.Path(o.Report.HTML))
	for _, want := range []string{"load-interval 30", head, "B だけ", "A にだけあります", "B にだけあります", "中身が空です"} {
		if !strings.Contains(string(html), want) {
			t.Errorf("the report does not hold %q", want)
		}
	}

	// A folder outside the case is a side too, and the same texts do not
	// differ.
	dir := t.TempDir()
	for _, dev := range gen.Devices {
		if err := writeTestFile(filepath.Join(dir, dev.Host+".cfg"), []byte(dev.Output.Text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	o, err = e.TreeDiff(dir, "generated", treeOpt, at)
	if err != nil {
		t.Fatal(err)
	}
	if o.Left() != 0 || len(o.Devices) != len(gen.Devices) {
		t.Fatalf("left %d, devices %d", o.Left(), len(o.Devices))
	}
	if _, err := e.TreeDiff("old", "old", treeOpt, at); err == nil {
		t.Fatal("the same side twice")
	}
	if _, err := e.TreeDiff(filepath.Join(dir, "none"), "generated", treeOpt, at); err == nil {
		t.Fatal("a folder that is not there")
	}
}

// The old devices are paired with the new ones by the names the book gives
// them.
func TestTreeDiffRenamed(t *testing.T) {
	old := siteFile(t, "site", "rt-s01.txt")
	e := newCase(t, map[string]string{"input/old/rt-s01.txt": old})
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	setCell(t, e, "新旧対照表（機器）", "旧ホスト名", "rt-s01", "新ホスト名", "rt-n01")
	if err := writeTestFile(e.Proj.Path("input/new/n.txt"), []byte(strings.Replace(old, "hostname rt-s01\n", "hostname rt-n01\n", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, sides := range [][2]string{{"old", "new"}, {"new", "old"}} {
		o, err := e.TreeDiff(sides[0], sides[1], treeOpt, at)
		if err != nil {
			t.Fatal(err)
		}
		d := treeDevice(t, o, "rt-s01 → rt-n01")
		if d.Result == nil || d.Result.Changed != 1 || d.Result.Added+d.Result.Removed != 0 {
			t.Fatalf("%v: %+v", sides, d.Result)
		}
	}
}
