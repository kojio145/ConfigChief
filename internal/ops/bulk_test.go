package ops

import (
	"testing"
	"time"

	"github.com/kojio145/configchief/internal/bulk"
	"github.com/kojio145/configchief/internal/importer"
	"github.com/kojio145/configchief/internal/project"
)

// What spec 16 asks of a case of 300 devices of 500 lines each.
const (
	bulkDevices  = 300
	bulkLines    = 500
	importLimit  = 10 * time.Second
	generateTime = 15 * time.Second
)

// bulkCase makes a case folder with the made-up configurations of n
// devices, as the old and as the new ones.
func bulkCase(t *testing.T, n, lines int) *Env {
	t.Helper()
	files, err := bulk.Configs(n, lines)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	p, err := project.Init(dir, "sample", "")
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		for _, side := range []string{"input/old/", "input/new/"} {
			if err := writeTestFile(p.Path(side+f.Name), []byte(f.Text), 0o644); err != nil {
				t.Fatal(err)
			}
		}
	}
	e, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

// bulkTimes is how long the steps of a made-up case took.
type bulkTimes struct {
	read  time.Duration // the import of the new configurations, with its report
	first time.Duration // the first generation with its checks
	again time.Duration // the next one, which also tells what changed
}

// bulkRun takes a made-up case through the steps: import, book, generate,
// generate again. Nothing of the case may be left for a person to look at:
// the time of a run that stops early says nothing.
func bulkRun(t *testing.T, e *Env, n int) bulkTimes {
	t.Helper()
	var took bulkTimes

	start := time.Now()
	in, err := e.RunImport("new", "", at)
	if err != nil {
		t.Fatal(err)
	}
	took.read = time.Since(start)
	if len(in.Source.Files) != n {
		t.Fatalf("%d files were read, want %d", len(in.Source.Files), n)
	}
	bad := 0
	for _, f := range in.Source.Files {
		if f.Result == nil {
			t.Fatalf("%s was not imported: %s", f.Path, f.Err)
		}
		for _, l := range f.Result.Lines {
			if l.State == importer.Unknown && bad < 20 {
				bad++
				t.Errorf("%s: a line was not read: %s", f.Path, l.Text)
			}
		}
	}
	if bad > 0 {
		t.FailNow()
	}

	b, err := e.MakeBook("new", false, at)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Problems) != 0 {
		t.Fatalf("the book: %v", b.Problems)
	}

	took.first = bulkGenerate(t, e, n, at)
	took.again = bulkGenerate(t, e, n, at.Add(time.Hour))
	return took
}

// bulkGenerate generates every device and tells how long it took.
func bulkGenerate(t *testing.T, e *Env, n int, now time.Time) time.Duration {
	t.Helper()
	start := time.Now()
	o, err := e.Generate(GenOptions{}, now)
	if err != nil {
		t.Fatal(err)
	}
	took := time.Since(start)
	if left := unsettled(o); len(left) != 0 {
		total := len(left)
		if total > 20 {
			left = left[:20]
		}
		t.Fatalf("%d findings are left, the first:\n%v", total, left)
	}
	written := 0
	for _, d := range o.Devices {
		if d.State == GenWritten {
			written++
		}
	}
	if written != n {
		t.Fatalf("%d configurations were written, want %d", written, n)
	}
	return took
}

// A small made-up case goes through the steps with nothing left to look at
// (what the large one is made of holds), and the pairs of its devices are
// found: the checks of the large case have something to do.
func TestBulkSmall(t *testing.T) {
	e := bulkCase(t, 4, bulkLines)
	bulkRun(t, e, 4)
	o, err := e.Generate(GenOptions{Dry: true}, at)
	if err != nil {
		t.Fatal(err)
	}
	looked := map[string]int{}
	for _, id := range []string{"X-341", "X-331", "X-300"} {
		looked[id] = len(o.Tally.Looked(id))
	}
	if looked["X-341"] == 0 || looked["X-331"] == 0 {
		t.Errorf("the pairs of the devices were not looked at: %v", looked)
	}
}

// The case of spec 16: 300 devices of 500 lines each are imported within 10
// seconds and generated and checked within 15 (spec 19 "大量").
//
// The time depends on what else the machine does, and the packages are
// tested side by side: a step that took too long is done once more, and
// only a step that is too slow twice fails. `go test -short` leaves the
// test out.
func TestBulkTime(t *testing.T) {
	if testing.Short() {
		t.Skip("300 devices take half a minute")
	}
	e := bulkCase(t, bulkDevices, bulkLines)
	took := bulkRun(t, e, bulkDevices)
	t.Logf("%d devices of %d lines: import %.1fs, generate %.1fs, generate again %.1fs",
		bulkDevices, bulkLines, took.read.Seconds(), took.first.Seconds(), took.again.Seconds())

	if took.read > importLimit {
		start := time.Now()
		if _, err := e.RunImport("new", "", at.Add(2*time.Hour)); err != nil {
			t.Fatal(err)
		}
		if again := time.Since(start); again > importLimit {
			t.Errorf("the import took %.1fs and %.1fs, the limit is %v", took.read.Seconds(), again.Seconds(), importLimit)
		}
	}
	if took.first > generateTime || took.again > generateTime {
		if once := bulkGenerate(t, e, bulkDevices, at.Add(3*time.Hour)); once > generateTime {
			t.Errorf("the generation took %.1fs, %.1fs and %.1fs, the limit is %v",
				took.first.Seconds(), took.again.Seconds(), once.Seconds(), generateTime)
		}
	}
}
