package ops

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/model"
)

// Files a person drops are copied into input/ and nothing else (spec 4.2,
// 14.10.1).
func TestAddInputs(t *testing.T) {
	e := newCase(t, nil)
	src := t.TempDir()
	write := func(rel, text string) string {
		t.Helper()
		p := filepath.Join(src, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	a := write("rt-a01.log", "hostname rt-a01\n")
	b := write("set/rt-a02.txt", "hostname rt-a02\n")
	write("set/.hidden", "x")
	write("set/deeper/rt-a03.txt", "hostname rt-a03\n")

	got, err := e.AddInputs("old", []string{a, filepath.Join(src, "set")}, false, at)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.Copied, ",") != "rt-a01.log,rt-a02.txt" || len(got.Clash) != 0 {
		t.Fatalf("copied %v, clash %v", got.Copied, got.Clash)
	}
	if len(got.Skipped) != 2 || !strings.Contains(strings.Join(got.Skipped, "\n"), "中のフォルダ 1 個") {
		t.Errorf("skipped = %v", got.Skipped)
	}
	if read(t, e, "input/old/rt-a02.txt") != "hostname rt-a02\n" {
		t.Error("the content was not copied")
	}
	if _, err := os.Stat(b); err != nil {
		t.Error("the file chosen was moved")
	}

	// The same content again: nothing is written.
	if got, err = e.AddInputs("old", []string{a}, false, at); err != nil || len(got.Same) != 1 || len(got.Copied) != 0 {
		t.Fatalf("again: %v, %+v", err, got)
	}
	// Another content: asked first, then written over with a copy kept.
	write("rt-a01.log", "hostname rt-a01\ninterface Loopback0\n")
	if got, err = e.AddInputs("old", []string{a}, false, at); err != nil || len(got.Clash) != 1 || len(got.Copied) != 0 {
		t.Fatalf("a clash: %v, %+v", err, got)
	}
	if read(t, e, "input/old/rt-a01.log") != "hostname rt-a01\n" {
		t.Error("a clash was written over without asking")
	}
	if got, err = e.AddInputs("old", []string{a}, true, at); err != nil || len(got.Copied) != 1 || len(got.Backups) != 1 {
		t.Fatalf("written over: %v, %+v", err, got)
	}
	if read(t, e, got.Backups[0]) != "hostname rt-a01\n" || !strings.HasPrefix(got.Backups[0], "state/backup/input_old_") {
		t.Errorf("the copy %s", got.Backups[0])
	}

	// The customer's lists take only the files of a list.
	x := write("list.xlsx", "x")
	n := write("notes.txt", "x")
	if got, err = e.AddInputs("customer", []string{x, n}, false, at); err != nil || len(got.Copied) != 1 || len(got.Skipped) != 1 {
		t.Errorf("customer: %v, %+v", err, got)
	}
	if _, err := e.AddInputs("extra", []string{a}, false, at); err == nil {
		t.Error("a folder that is not of input/ took files")
	}
}

// The proposals of IF 対応 are made without a second import, and fill the
// empty new names only (spec 10.2, 14.10.2).
func TestProposeIfs(t *testing.T) {
	e := migCase(t)
	setCell(t, e, "IF 対応", "旧 IF 名", "Loopback0", "新 IF 名", "")
	setCell(t, e, "IF 対応", "旧 IF 名", "GigabitEthernet0/0", "新 IF 名", "-")
	p, err := e.ProposeIfs(at)
	if err != nil {
		t.Fatal(err)
	}
	if p.Filled != 1 || p.Added != 0 || p.Backup == "" || !strings.Contains(p.Said(), "1 行の新 IF 名") {
		t.Fatalf("proposal = %+v (%s)", p, p.Said())
	}
	_, bk, err := e.readBook()
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range bk.Ifs {
		switch m.OldIf {
		case "Loopback0":
			if m.NewIf != "Loopback0" || m.Basis == "" {
				t.Errorf("Loopback0: %+v", m)
			}
		case "GigabitEthernet0/0":
			if m.NewIf != "-" {
				t.Errorf("a name a person wrote was changed: %+v", m)
			}
		}
	}
	// Nothing to do: nothing is written.
	if p, err = e.ProposeIfs(at); err != nil || p.Backup != "" || p.Filled+p.Added != 0 {
		t.Errorf("again: %v, %+v", err, p)
	}
}

// The one thing to do next follows the case along (spec 14.10.4).
func TestNextStep(t *testing.T) {
	e := newCase(t, nil)
	step := func(want string) {
		t.Helper()
		n, err := e.NextStep()
		if err != nil {
			t.Fatal(err)
		}
		if n.Act != want {
			t.Fatalf("next = %+v, want %q", n, want)
		}
	}
	step("add-old")
	if err := os.WriteFile(e.Proj.Path("input/old/rt-a01.txt"), []byte(sample(t, "cisco_ios")), 0o644); err != nil {
		t.Fatal(err)
	}
	step("import-run")
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	step("edit-devices")
	setCell(t, e, "機器対応", "旧ホスト名", "rt-a01", "新ホスト名", "rt-a01")
	setCell(t, e, "機器対応", "旧ホスト名", "rt-a01", "新メーカー", "cisco_ios")
	step("propose-ifs")
	if _, err := e.ProposeIfs(at); err != nil {
		t.Fatal(err)
	}
	step("edit-ifs")
	_, bk, _ := e.readBook()
	for _, m := range bk.Ifs {
		setCell(t, e, "IF 対応", "旧 IF 名", m.OldIf, "新 IF 名", m.OldIf)
	}
	step("make-book")
	if _, err := e.MakeBook("old", false, at); err != nil {
		t.Fatal(err)
	}
	step("generate")
	if _, err := e.Generate(GenOptions{}, at); err != nil {
		t.Fatal(err)
	}
	n, err := e.NextStep()
	if err != nil || n.Page != "result" {
		t.Errorf("after the generation: %v, %+v", err, n)
	}

	nb := newCase(t, nil)
	nb.Proj.Config.Kind = "new"
	if n, err := nb.NextStep(); err != nil || n.Act != "new-book" {
		t.Errorf("a new build: %v, %+v", err, n)
	}
}

// Findings picked together are approved one by one with one reason (appendix
// C 3.2, spec 14.10.8).
func TestApproveFindings(t *testing.T) {
	e := newCase(t, nil)
	mk := func(rule, sev, dev string) Judged {
		return Judged{Finding: check.New(rule, sev, dev, model.Ref{Table: "interfaces", Key: dev}, "見本")}
	}
	items := []Judged{mk("V-202", check.Warning, "rt-a01"), mk("G-602", check.Error, "rt-a02"), mk("G-601", check.Error, "rt-a03"), mk("V-202", check.Info, "rt-a04")}
	keys := []string{items[0].Key, items[1].Key, items[2].Key, items[3].Key, "nothing"}
	done, left, err := e.ApproveFindings(items, keys, "確かめた", "試験", at)
	if err != nil {
		t.Fatal(err)
	}
	if len(done) != 2 || len(left) != 3 {
		t.Fatalf("done %d, left %v", len(done), left)
	}
	list, err := e.Proj.Approvals()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 || list[0].Finding != items[0].Key || list[1].Finding != items[1].Key || list[0].Scope != nil || list[0].ID == list[1].ID {
		t.Errorf("approvals = %+v", list)
	}
	if _, _, err := e.ApproveFindings(items, keys, " ", "試験", at); err == nil {
		t.Error("no reason was taken")
	}
}

// The newest output is found again, and a check that writes nothing points
// at its files (spec 14.10.6).
func TestLastOutput(t *testing.T) {
	e := genCase(t, map[string]string{"input/new/rt-c01.txt": rendered(t, "cisco_ios")})
	if e.LastOutput() != "" {
		t.Error("an output before any generation")
	}
	if _, err := e.Generate(GenOptions{}, at); err != nil {
		t.Fatal(err)
	}
	dir := e.LastOutput()
	if dir != strings.TrimSuffix(outDir, "/") {
		t.Fatalf("last output = %q", dir)
	}
	o, err := e.Generate(GenOptions{CheckOnly: true, Dry: true}, at)
	if err != nil {
		t.Fatal(err)
	}
	configs, mark := o.Attach(e, dir)
	if !configs || mark != "" || o.Devices[0].File != outDir+"configs/rt-c01.txt" || o.Report.XLSX != outDir+"reports/check_results.xlsx" {
		t.Errorf("attached: %v %q %+v %+v", configs, mark, o.Devices, o.Report)
	}
	if Stamped(dir) != "2026-01-02 03:04:05" {
		t.Errorf("stamp = %s", Stamped(dir))
	}
}
