package ops

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/convert"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/report"
)

// What the screens show (spec 14): the same outcomes as the reports. A
// screen masks the secrets whatever project.yaml says; only what is written
// to out/ follows mask_in_reports.

// forScreen gives a copy of the environment whose texts mask the secrets.
func (e *Env) forScreen() *Env {
	c := *e
	c.screen = true
	return &c
}

// Line is a line of an imported file as a screen shows it.
type Line struct {
	No     int
	Text   string // the secrets masked
	State  string // importer.Taken | importer.Ignored | importer.Unknown
	Word   string // the state in the words of the reports
	Blank  bool   // a comment, a blank line, a line outside the configuration (not counted)
	Rule   string
	Reason string
	Made   []Made
}

// Made is a value a line gave to the design data.
type Made struct {
	Where string // the table, the row and the column, by the names the book shows
	Value string // "" for the line that opened the row
}

// Lines gives the lines of an imported file for a screen (nil for a file
// that was not imported).
func (e *Env) Lines(f *File) []Line {
	if f == nil || f.Result == nil {
		return nil
	}
	s := e.forScreen()
	m := s.newMasker(f.Result)
	out := make([]Line, 0, len(f.Result.Lines))
	for _, l := range f.Result.Lines {
		line := Line{No: l.Line, Text: m.text(l), State: l.State, Word: stateWords[l.State], Blank: l.Separator, Rule: l.RuleID, Reason: l.Reason}
		seen := map[string]bool{}
		for _, ref := range l.Refs {
			made := Made{Where: s.refText(ref, m.rows)}
			if seen[made.Where] {
				continue
			}
			seen[made.Where] = true
			if row := m.rows[ref.Table+"\x00"+ref.Key]; row != nil && ref.Column != "" {
				made.Value = s.cellText(ref.Table, ref.Column, row.Cells[ref.Column])
			}
			line.Made = append(line.Made, made)
		}
		out = append(out, line)
	}
	return out
}

// ChoosePack records the pack a person chose for a file whose vendor the
// tool cannot tell (spec 8.1 step 3). The next import reads the file with it.
func (e *Env) ChoosePack(path, id string, now time.Time) error {
	if e.Packs[id] == nil {
		return fmt.Errorf("パック %s はありません（使えるパック: %s）", id, strings.Join(e.PackIDs, ", "))
	}
	found := false
	for _, source := range []string{"old", "new"} {
		paths, err := e.Proj.Inputs(source)
		if err != nil {
			return err
		}
		for _, p := range paths {
			found = found || p == path
		}
	}
	if !found {
		return fmt.Errorf("%s は input/old にも input/new にもありません", path)
	}
	chosen, err := e.Proj.Sources()
	if err != nil {
		return err
	}
	chosen.Choose(path, id)
	if err := e.Proj.SaveSources(chosen); err != nil {
		return err
	}
	e.Proj.Log(now, "pack: %s をパック %s で読みます（人が選んだ）", path, id)
	return nil
}

// Mapping is the mapping of the book and what of it is open (spec 10.2).
type Mapping struct {
	Source  *Source          // the old side
	Book    *book.Book       // nil = there is no book yet
	Applied *convert.Applied // the mapping put on the old side; nil without a book
	Mig     *Verdicts        // the findings of the mapping (M-406 to M-408) with the approvals
}

// Mapping reads the old side and the book and puts the mapping of the book
// on. It writes nothing.
func (e *Env) Mapping() (*Mapping, error) {
	src, err := e.importSide("old", "")
	if err != nil {
		return nil, err
	}
	_, bk, err := e.readBook()
	if err != nil {
		return nil, err
	}
	m := &Mapping{Source: src, Book: bk}
	if bk != nil {
		m.Applied = convert.Apply(e.Schema, src.Data, bk.Devices, bk.Ifs)
		if m.Mig, err = e.verdicts(mappingFindings(m.Applied), bk.Data); err != nil {
			return nil, err
		}
	}
	return m, nil
}

// MappingSheet lists for a screen what the mapping left open and, with all
// set, what it left out of the data.
func (e *Env) MappingSheet(a *convert.Applied, v *Verdicts, all bool) report.Sheet {
	return e.forScreen().mappingSheet(a, v, all)
}

// ImportDoc gives the outcome of an import as a screen shows it: the numbers
// and the lists of the report, with the secrets masked.
func (e *Env) ImportDoc(o *ImportOutcome, now time.Time) *report.Doc {
	return e.forScreen().importDoc(o, now)
}

// CompareDoc gives the outcome of a comparison as a screen shows it: the
// lists of the report, with the secrets masked.
func (e *Env) CompareDoc(o *CompareOutcome, now time.Time) *report.Doc {
	return e.forScreen().compareDoc(o, now)
}

// CheckDoc gives the outcome of a generation as a screen shows it: the
// numbers and the lists of the check results, with the secrets masked.
func (e *Env) CheckDoc(o *GenerateOutcome, now time.Time) *report.Doc {
	return e.forScreen().checkDoc(o, now)
}

// ChangesDoc gives the list of changes as a screen shows it (nil if the
// generation had nothing to compare with).
func (e *Env) ChangesDoc(o *GenerateOutcome, now time.Time) *report.Doc {
	if o.Changes == nil || o.Changes.Before == "" {
		return nil
	}
	return e.forScreen().changesDoc(o, now)
}

// FindingPlace is where a finding points, by the names the book shows.
type FindingPlace struct {
	Table, Row, Column string
	Text               string // the line of the configuration, the secrets masked ("" = none)
}

// FindingPlace tells a screen where a finding of an outcome points.
func (e *Env) FindingPlace(o *GenerateOutcome, f check.Finding) FindingPlace {
	var p FindingPlace
	if f.Ref.Table != "" {
		p.Table = e.tableTitle(f.Ref.Table)
		p.Row = e.keyLabel(f.Ref.Table, f.Ref.Key)
		if f.Ref.Column != "" {
			p.Column = e.columnTitle(f.Ref.Table, f.Ref.Column)
		}
	}
	if f.Line > 0 {
		for i := range o.Devices {
			if o.Devices[i].Host == f.Device {
				p.Text = e.forScreen().lineText(o.Devices[i].Output, f.Line)
			}
		}
	}
	return p
}

// ApprovalTarget tells in words what an approval is for.
func ApprovalTarget(a project.Approval) string { return approvalTarget(a) }

// LastGeneration tells when the configurations were generated last: the time
// of the newest record under state/snapshots ("" = never).
func (e *Env) LastGeneration() string {
	entries, err := os.ReadDir(e.Proj.Path(snapshotDir))
	if err != nil {
		return ""
	}
	last := ""
	for _, ent := range entries {
		if name := ent.Name(); !ent.IsDir() && strings.HasSuffix(name, ".json") && name > last {
			last = name
		}
	}
	if last == "" {
		return ""
	}
	stamp := strings.TrimSuffix(last, ".json")
	if len(stamp) > 17 {
		stamp = stamp[:17] // without the number of a second generation within a second
	}
	t, err := time.Parse("2006-01-02_150405", stamp)
	if err != nil {
		return strings.TrimSuffix(last, ".json")
	}
	return t.Format("2006-01-02 15:04:05")
}
