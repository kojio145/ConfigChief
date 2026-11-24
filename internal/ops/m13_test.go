package ops

import (
	"archive/zip"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/project"
	"github.com/xuri/excelize/v2"
)

// stepOf finds a step of the case by its ID.
func stepOf(t *testing.T, p *Progress, id string) Step {
	t.Helper()
	for _, s := range p.Steps {
		if s.ID == id {
			return s
		}
	}
	t.Fatalf("no step %s in %+v", id, p.Steps)
	return Step{}
}

// migrated makes a case from the sample of the Cisco pack up to a book with
// its design data.
func migrated(t *testing.T) *Env {
	t.Helper()
	e := newCase(t, map[string]string{"input/old/rt-a01.txt": sample(t, "cisco_ios")})
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	setCell(t, e, "新旧対照表（機器）", "旧ホスト名", "rt-a01", "新ホスト名", "rt-a01")
	setCell(t, e, "新旧対照表（機器）", "旧ホスト名", "rt-a01", "新メーカー", "cisco_ios")
	if _, err := e.ProposeIfs(at); err != nil {
		t.Fatal(err)
	}
	_, bk, _ := e.readBook()
	for _, m := range bk.Ifs {
		setCell(t, e, "新旧対照表（ポート）", "旧 IF 名", m.OldIf, "新 IF 名", m.OldIf)
	}
	if _, err := e.MakeBook("old", false, at); err != nil {
		t.Fatal(err)
	}
	return e
}

// later waits until the clock of the files moves on, so that a file written
// next is newer than the ones before.
func later() { time.Sleep(20 * time.Millisecond) }

// The steps of a case go on to the deliverables and the check after the
// configurations went in (spec 14.12.1).
func TestProgressSteps(t *testing.T) {
	e := newCase(t, nil)
	p, err := e.Progress(OpenCount{Known: true})
	if err != nil {
		t.Fatal(err)
	}
	if stepOf(t, p, "old").State != StepTodo || p.Next.Act != "add-old" {
		t.Fatalf("an empty case: %+v", p)
	}

	e = migrated(t)
	// The count of the unhandled lines not known yet: the next step waits.
	if p, err = e.Progress(OpenCount{}); err != nil || stepOf(t, p, "judge").State != StepCounting {
		t.Fatalf("not counted: %v, %+v", err, p)
	}
	// Lines left to judge come before the generation.
	if p, err = e.Progress(OpenCount{Lines: 3, Known: true}); err != nil || p.Next.Page != "judge" || stepOf(t, p, "judge").Note != "3 行" {
		t.Fatalf("lines to judge: %v, %+v", err, p.Next)
	}
	p, _ = e.Progress(OpenCount{Known: true})
	if p.Next.Act != "generate" || stepOf(t, p, "findings").State != StepWait {
		t.Fatalf("before the generation: %+v", p.Next)
	}
	later()
	if _, err := e.Generate(GenOptions{}, at); err != nil {
		t.Fatal(err)
	}
	p, _ = e.Progress(OpenCount{Known: true})
	if stepOf(t, p, "generate").State != StepDone {
		t.Fatalf("after the generation: %+v", p.Steps)
	}
	snap, _, _ := e.lastSnapshot()
	if snap == nil || !snap.Counted {
		t.Fatal("the generation did not keep the count of the findings")
	}
	fin := stepOf(t, p, "findings")
	if snap.Errors+snap.Warnings > 0 {
		if fin.State != StepTodo || p.Next.Page != "result" {
			t.Fatalf("findings left: %+v, next %+v", fin, p.Next)
		}
	} else if fin.State != StepDone || p.Next.Act != "make-delivery" {
		t.Fatalf("no finding left: %+v, next %+v", fin, p.Next)
	}
	if stepOf(t, p, "logs").State != StepOptional || stepOf(t, p, "deliver").State != StepTodo {
		t.Fatalf("the steps after: %+v", p.Steps)
	}

	// A log put in is a step to do.
	later()
	if err := writeTestFile(e.Proj.Path("input/logs/rt-a01.log"), []byte("rt-a01#show clock\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if p, _ = e.Progress(OpenCount{Known: true}); stepOf(t, p, "logs").State != StepTodo {
		t.Fatalf("a log to check: %+v", stepOf(t, p, "logs"))
	}

	// The whole set made: the deliverables are done.
	later()
	if _, err := e.MakeDelivery(DeliveryOptions{AllowUnverified: true}, at.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	if p, _ = e.Progress(OpenCount{Known: true}); stepOf(t, p, "deliver").State != StepDone {
		t.Fatalf("after the whole set: %+v", stepOf(t, p, "deliver"))
	}

	// The book changed after the generation: generate again.
	later()
	setCell(t, e, "新旧対照表（機器）", "旧ホスト名", "rt-a01", "備考", "sample note")
	if p, _ = e.Progress(OpenCount{Known: true}); stepOf(t, p, "generate").State != StepTodo || !strings.Contains(p.Next.Text, "パラメータブック") {
		t.Fatalf("the book changed: %+v", p.Next)
	}

	// A new build has its own steps.
	nb := newCase(t, nil)
	nb.Proj.Config.Kind = "new"
	if p, _ = nb.Progress(OpenCount{Known: true}); p.Next.Act != "new-book" || p.Steps[0].ID != "newbook" {
		t.Fatalf("a new build: %+v", p.Next)
	}
}

// The whole set gathers the newest generation and the sheets made now; the
// history of the versions goes into the sheets (spec 13.6, 13.7).
func TestDeliveryAndRevisions(t *testing.T) {
	e := migrated(t)
	if p, err := e.PlanDelivery(); err != nil || p.Ready {
		t.Fatalf("before a generation: %v, %+v", err, p)
	}
	// A template that shows the history.
	f := excelize.NewFile()
	f.SetCellStr("Sheet1", "A1", "{{ revision.version }}")
	f.SetCellStr("Sheet1", "A3", "{{#rows revisions}}{{ row.version }}")
	f.SetCellStr("Sheet1", "B3", "{{ row.summary }}")
	f.SetCellStr("Sheet1", "A4", "end")
	os.MkdirAll(e.Proj.Path("templates"), 0o755)
	if err := f.SaveAs(e.Proj.Path("templates/history.xlsx")); err != nil {
		t.Fatal(err)
	}
	later()
	if _, err := e.Generate(GenOptions{}, at); err != nil {
		t.Fatal(err)
	}
	p, err := e.PlanDelivery()
	if err != nil || !p.Ready || p.NextVersion != "1.0" || p.Summary != "初版" || p.Configs != 1 {
		t.Fatalf("plan: %v, %+v", err, p)
	}
	unapproved := p.Errors+p.Warnings > 0
	if unapproved {
		if _, err := e.MakeDelivery(DeliveryOptions{}, at.Add(time.Minute)); err == nil || !strings.Contains(err.Error(), "未承認") {
			t.Fatalf("findings left were not told: %v", err)
		}
	}
	o, err := e.MakeDelivery(DeliveryOptions{AllowUnverified: true, Version: "1.0", Summary: "初版", By: "sample-user"}, at.Add(time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if o.Configs != 1 || o.Revision == nil || o.Unverified != unapproved {
		t.Fatalf("outcome: %+v", o)
	}
	for _, want := range []string{"/configs/rt-a01.txt", "/チェック結果表.xlsx", "/history.xlsx", "/" + deliveryWords["ja"].done} {
		found := false
		for _, f := range o.Files {
			if strings.HasSuffix(f, want) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s is not in the set: %v", want, o.Files)
		}
	}
	z, err := zip.OpenReader(e.Proj.Path(o.Zip))
	if err != nil {
		t.Fatal(err)
	}
	if len(z.File) != len(o.Files) || !strings.HasPrefix(z.File[0].Name, deliveryWords["ja"].prefix+"sample/") {
		t.Errorf("zip: %d files (want %d), first %s", len(z.File), len(o.Files), z.File[0].Name)
	}
	z.Close()
	sheet := func(path string) [][]string {
		t.Helper()
		x, err := excelize.OpenFile(e.Proj.Path(path))
		if err != nil {
			t.Fatal(err)
		}
		defer x.Close()
		rows, _ := x.GetRows("Sheet1")
		return rows
	}
	rows := sheet(o.Dir + "/history.xlsx")
	if rows[0][0] != "1.0" || rows[2][0] != "1.0" || rows[2][1] != "初版" || rows[3][0] != "end" {
		t.Errorf("the history in the sheet: %v", rows)
	}
	// The same version twice is refused.
	if _, err := e.MakeDelivery(DeliveryOptions{AllowUnverified: true, Version: "1.0"}, at.Add(2*time.Minute)); err == nil {
		t.Error("the same version was taken twice")
	}

	// A change and a second version: what changed is proposed.
	later()
	setCell(t, e, "機器", "ホスト名", "rt-a01", "機種", "sample-model")
	later()
	if _, err := e.Generate(GenOptions{}, at.Add(3*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if p, err = e.PlanDelivery(); err != nil || p.NextVersion != "1.1" || !strings.Contains(p.Summary, "機器 1 セル") {
		t.Fatalf("second plan: %v, %+v", err, p)
	}
	o, err = e.MakeDelivery(DeliveryOptions{AllowUnverified: true, Version: p.NextVersion, Summary: p.Summary}, at.Add(4*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	rows = sheet(o.Dir + "/history.xlsx")
	if rows[0][0] != "1.1" || rows[2][0] != "1.0" || rows[3][0] != "1.1" || rows[4][0] != "end" {
		t.Errorf("two versions in the sheet: %v", rows)
	}
	revs, err := e.Revisions()
	if err != nil || len(revs) != 2 || revs[0].By != "sample-user" {
		t.Errorf("history: %v, %+v", err, revs)
	}
	// An ordinary export of the sheets shows the history too.
	s, err := e.ExportSheets(false, at.Add(5*time.Minute))
	if err != nil || len(s.Files) != 1 {
		t.Fatalf("sheets: %v, %+v", err, s)
	}
	if rows = sheet(s.Files[0]); rows[0][0] != "1.1" || rows[3][0] != "1.1" {
		t.Errorf("the history in an ordinary export: %v", rows)
	}

	// The list of the outputs tells what made each folder.
	list := e.Outputs()
	kinds := map[string]bool{}
	keep := 0
	for _, o := range list {
		for _, k := range o.Kinds {
			kinds[k] = true
		}
		if o.Keep {
			keep++
		}
	}
	for _, k := range []string{"生成とチェック", "納品一式", "納品用シート"} {
		if !kinds[k] {
			t.Errorf("no output of %s in %+v", k, list)
		}
	}
	if keep != 2 {
		t.Errorf("kept %d folders, want 2: %+v", keep, list)
	}
	if _, err := e.DropOutputs([]string{"out/../input"}, at); err == nil {
		t.Error("a folder outside out/ was taken away")
	}
	if n, err := e.DropOutputs([]string{list[len(list)-1].Dir}, at); err != nil || n != 1 || len(e.Outputs()) != len(list)-1 {
		t.Errorf("drop: %v, %d", err, n)
	}

	// The names of the set in English (spec 14.15.4): project.yaml keeps
	// its other lines and comments, and the set is still known as one.
	if _, err := e.SetDeliveryNames("fr", at); err == nil {
		t.Error("a language that is not there was taken")
	}
	if bak, err := e.SetDeliveryNames("en", at); err != nil || bak == "" {
		t.Fatalf("set en: %v, %q", err, bak)
	}
	if bak, err := e.SetDeliveryNames("en", at); err != nil || bak != "" {
		t.Errorf("the same language twice wrote again: %v, %q", err, bak)
	}
	yml, _ := os.ReadFile(e.Proj.Path("project.yaml"))
	if !strings.Contains(string(yml), "delivery_names: en") || !strings.Contains(string(yml), "# 止めるチェックの ID") {
		t.Errorf("project.yaml: %s", yml)
	}
	again, err := Open(e.Proj.Dir)
	if err != nil || again.Proj.Config.Export.DeliveryNames != "en" {
		t.Fatalf("reopen: %v", err)
	}
	o, err = e.MakeDelivery(DeliveryOptions{AllowUnverified: true}, at.Add(6*time.Minute))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(o.Dir, "/delivery_sample") || !strings.Contains(o.Zip, "/delivery_sample_") {
		t.Errorf("english set: %s, %s", o.Dir, o.Zip)
	}
	for _, want := range []string{"/check_results.xlsx", "/contents.txt", "/configs/rt-a01.txt"} {
		found := false
		for _, f := range o.Files {
			found = found || strings.HasSuffix(f, want)
		}
		if !found {
			t.Errorf("%s is not in the english set: %v", want, o.Files)
		}
	}
	got := false
	for _, x := range e.Outputs() {
		for _, k := range x.Kinds {
			got = got || k == "納品一式"
		}
	}
	if !got {
		t.Error("the english set is not in the list of the outputs")
	}
}

// The sample case can be imported, and has unhandled lines to judge (spec
// 14.12.7).
func TestSampleCase(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "sample")
	n, err := MakeSampleCase(dir)
	if err != nil || n != 2 {
		t.Fatalf("sample: %v, %d", err, n)
	}
	e, err := Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if e.Proj.Config.Name != SampleName || e.Proj.Config.Kind != "migration" {
		t.Errorf("config: %+v", e.Proj.Config)
	}
	imp, err := e.RunImport("old", "", at)
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range imp.Source.Files {
		if f.Pack != "cisco_ios" || f.Err != "" {
			t.Errorf("file %s: pack %s, %s", f.Path, f.Pack, f.Err)
		}
	}
	if c := e.CountOpen(); !c.Known || c.Lines == 0 {
		t.Errorf("open lines = %+v, want some to judge", c)
	}
	if _, err := MakeSampleCase(dir); err == nil {
		t.Error("a second sample in the same folder")
	}
}

// The settings of a case before go into a new one; the inputs and the book
// do not (spec 14.12.8).
func TestCopyCaseSettings(t *testing.T) {
	from := newCase(t, map[string]string{
		"input/old/rt-a01.txt": "hostname rt-a01\n",
		"extra/_all.j2":        "! all\n",
		"extra/@branch.j2":     "! branch\n",
		"extra/rt-a01.j2":      "! one\n",
		"rules/case.yaml":      "rules: []\n",
		"templates/sheet.xlsx": "x",
		"fill/lan.txt":         "hostname {{devices.hostname}}\n",
	})
	if _, err := from.SaveCheckSettings(map[string]string{"V-208": "warning"}, at); err != nil {
		t.Fatal(err)
	}
	cfg := read(t, from, "project.yaml")
	cfg = strings.Replace(cfg, "keep_words: []", "keep_words: [sample-word]", 1)
	if err := writeTestFile(from.Proj.Path("project.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
	to := filepath.Join(t.TempDir(), "next")
	if _, err := project.Init(to, "next", "migration"); err != nil {
		t.Fatal(err)
	}
	var parts []string
	for _, p := range CaseParts {
		if p.Default {
			parts = append(parts, p.ID)
		}
	}
	c, err := CopyCaseSettings(from.Proj.Dir, to, parts)
	if err != nil {
		t.Fatal(err)
	}
	has := func(rel string) bool {
		_, err := os.Stat(filepath.Join(to, filepath.FromSlash(rel)))
		return err == nil
	}
	for _, rel := range []string{"extra/_all.j2", "extra/@branch.j2", "rules/case.yaml", "templates/sheet.xlsx", "fill/lan.txt"} {
		if !has(rel) {
			t.Errorf("%s was not taken", rel)
		}
	}
	for _, rel := range []string{"extra/rt-a01.j2", "input/old/rt-a01.txt"} {
		if has(rel) {
			t.Errorf("%s was taken", rel)
		}
	}
	p, err := project.Open(to)
	if err != nil {
		t.Fatal(err)
	}
	if p.Config.Name != "next" || p.Config.Checks.Severity["V-208"] != "warning" || len(p.Config.Export.KeepWords) != 1 {
		t.Errorf("project.yaml: %+v", p.Config)
	}
	if c.Files != 5 || len(c.Parts) != 6 {
		t.Errorf("copied = %+v", c)
	}
	if _, err := CopyCaseSettings(to, to, parts); err == nil {
		t.Error("a case took its own settings")
	}
}

// A check over the draft reads the book with the draft put in and writes
// nothing (spec 14.12.3).
func TestDraftCheck(t *testing.T) {
	e := editCase(t)
	if _, err := e.DraftCheck(at); err == nil {
		t.Error("a check without a draft")
	}
	before := read(t, e, e.Proj.Config.Book)
	draft(t, e, book.DraftSheet{Sheet: "devices", Added: []book.DraftAdd{{Cells: map[string]string{"hostname": "rt-a02", "vendor": "no_such_pack"}}}})
	d, err := e.DraftCheck(at)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range d.Outcome.Items {
		if f.RuleID == "V-109" && f.Device == "rt-a02" {
			found = true
		}
	}
	if !found || !d.Moved || d.Draft != 1 {
		t.Errorf("draft check: found %v, moved %v, draft %d", found, d.Moved, d.Draft)
	}
	if read(t, e, e.Proj.Config.Book) != before || e.DraftCount() != 1 {
		t.Error("the check over the draft wrote the book or dropped the draft")
	}
	if entries, _ := os.ReadDir(e.Proj.Path("out")); len(entries) != 0 {
		t.Errorf("the check over the draft wrote %d outputs", len(entries))
	}
}

// TestDeliveryNeedsTheNewestGeneration: the whole set is not made from an
// older output when the folder of the newest generation is gone.
func TestDeliveryNeedsTheNewestGeneration(t *testing.T) {
	e := migrated(t)
	later()
	if _, err := e.Generate(GenOptions{}, at); err != nil {
		t.Fatal(err)
	}
	later()
	second := at.Add(time.Hour)
	o, err := e.Generate(GenOptions{}, second)
	if err != nil {
		t.Fatal(err)
	}
	if p, err := e.PlanDelivery(); err != nil || !p.Ready {
		t.Fatalf("plan: %v, %+v", err, p)
	}
	if err := os.RemoveAll(e.Proj.Path(o.Dir)); err != nil {
		t.Fatal(err)
	}
	p, err := e.PlanDelivery()
	if err != nil || p.Ready || !strings.Contains(p.Why, "出力フォルダが見つかりません") {
		t.Fatalf("plan without the newest folder: %v, %+v", err, p)
	}
}

// TestRunsInOneSecond: two runs within the same second write into two
// folders; one run keeps its folder.
func TestRunsInOneSecond(t *testing.T) {
	e := migrated(t)
	first, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	second, err := e.Generate(GenOptions{}, at.Add(300*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	if first.Dir == second.Dir || second.Dir != first.Dir+"_2" {
		t.Fatalf("folders: %s, %s", first.Dir, second.Dir)
	}
	if e.outDir(at) != first.Dir {
		t.Errorf("the first run moved: %s", e.outDir(at))
	}
	if _, err := os.Stat(e.Proj.Path(second.Dir + "/reports/check_results.xlsx")); err != nil {
		t.Errorf("the report of the second run is not in its folder: %v", err)
	}
}
