package project

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Sources is state/sources.yaml: the vendor pack a person chose for a file
// whose vendor the tool could not tell (spec 8.1 step 3).
//
// It also keeps which file a person chose for a host name that two or more
// files of a folder have (spec 8.1 step 4).
type Sources struct {
	byFile map[string]string
	use    map[string]string // folder + " " + host name -> file
}

type sourcesY struct {
	Sources []sourceY `yaml:"sources"`
	Use     []useY    `yaml:"use,omitempty"`
}

type useY struct {
	Source string `yaml:"source"` // "old" | "new"
	Device string `yaml:"device"`
	File   string `yaml:"file"`
}

type sourceY struct {
	File string `yaml:"file"`
	Pack string `yaml:"pack"`
}

const sourcesFile = "state/sources.yaml"

// Sources reads state/sources.yaml (empty if the file is not there).
func (p *Project) Sources() (*Sources, error) {
	s := &Sources{byFile: map[string]string{}, use: map[string]string{}}
	var y sourcesY
	if err := p.readState(sourcesFile, &y); err != nil {
		return nil, err
	}
	for _, e := range y.Sources {
		if e.File == "" || e.Pack == "" {
			return nil, fmt.Errorf("%s: file と pack が要ります", sourcesFile)
		}
		s.byFile[e.File] = e.Pack
	}
	for _, u := range y.Use {
		if u.Source == "" || u.Device == "" || u.File == "" {
			return nil, fmt.Errorf("%s: use には source・device・file が要ります", sourcesFile)
		}
		s.use[u.Source+" "+u.Device] = u.File
	}
	return s, nil
}

// Pack gives the pack chosen for a file ("" if none).
func (s *Sources) Pack(file string) string { return s.byFile[file] }

// Use gives the file chosen for a host name that several files of the
// folder have ("" if none was chosen).
func (s *Sources) Use(source, device string) string { return s.use[source+" "+device] }

// SetUse records the file chosen for a host name of a folder.
func (s *Sources) SetUse(source, device, file string) { s.use[source+" "+device] = file }

// Choose records the pack chosen for a file.
func (s *Sources) Choose(file, pack string) { s.byFile[file] = pack }

// SaveSources writes state/sources.yaml, by file name.
func (p *Project) SaveSources(s *Sources) error {
	var y sourcesY
	for f, id := range s.byFile {
		y.Sources = append(y.Sources, sourceY{File: f, Pack: id})
	}
	sort.Slice(y.Sources, func(i, j int) bool { return y.Sources[i].File < y.Sources[j].File })
	for k, f := range s.use {
		source, device, _ := strings.Cut(k, " ")
		y.Use = append(y.Use, useY{Source: source, Device: device, File: f})
	}
	sort.Slice(y.Use, func(i, j int) bool {
		if y.Use[i].Source != y.Use[j].Source {
			return y.Use[i].Source < y.Use[j].Source
		}
		return y.Use[i].Device < y.Use[j].Device
	})
	return p.writeState(sourcesFile, &y)
}

// readState reads a YAML file under the case folder; a file that is not
// there leaves into as it is.
func (p *Project) readState(rel string, into any) error {
	data, err := os.ReadFile(p.Path(rel))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("%s を読めません: %v", rel, err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(into); err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("%s: %v", rel, err)
	}
	return nil
}

func (p *Project) writeState(rel string, from any) error {
	var buf bytes.Buffer
	enc := yaml.NewEncoder(&buf)
	enc.SetIndent(2)
	if err := enc.Encode(from); err != nil {
		return err
	}
	path := p.Path(rel)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("%s を書けません: %v", rel, err)
	}
	return nil
}

// Approval is one entry of state/approvals.yaml (appendix C 3.2): either one
// finding by its fingerprint, or a scope.
type Approval struct {
	ID      string         `yaml:"id"`
	Finding string         `yaml:"finding,omitempty"`
	Rule    string         `yaml:"rule,omitempty"`
	Device  string         `yaml:"device,omitempty"`
	Scope   *ApprovalScope `yaml:"scope,omitempty"`
	Reason  string         `yaml:"reason"`
	By      string         `yaml:"by"`
	At      string         `yaml:"at"`
}

// ApprovalScope is what an approval of a scope covers. Devices is "*",
// "@group", or host names with commas between.
type ApprovalScope struct {
	Rule    string `yaml:"rule"`
	Table   string `yaml:"table,omitempty"`
	Column  string `yaml:"column,omitempty"`
	Devices string `yaml:"devices"`
}

type approvalsY struct {
	Approvals []Approval `yaml:"approvals"`
}

const approvalsFile = "state/approvals.yaml"

// Approvals reads state/approvals.yaml (empty if the file is not there).
func (p *Project) Approvals() ([]Approval, error) {
	var y approvalsY
	if err := p.readState(approvalsFile, &y); err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for i, a := range y.Approvals {
		where := fmt.Sprintf("%s の %d 件目", approvalsFile, i+1)
		switch {
		case a.ID == "":
			return nil, fmt.Errorf("%s: id が要ります", where)
		case seen[a.ID]:
			return nil, fmt.Errorf("%s: id「%s」が 2 つあります", where, a.ID)
		case (a.Finding == "") == (a.Scope == nil):
			return nil, fmt.Errorf("%s（%s）: finding か scope のどちらか 1 つを書きます", where, a.ID)
		case a.Scope != nil && (a.Scope.Rule == "" || a.Scope.Devices == ""):
			return nil, fmt.Errorf("%s（%s）: scope には rule と devices が要ります", where, a.ID)
		case a.Reason == "" || a.By == "":
			return nil, fmt.Errorf("%s（%s）: reason と by が要ります", where, a.ID)
		}
		seen[a.ID] = true
	}
	return y.Approvals, nil
}

// SaveApprovals writes state/approvals.yaml, in the order given.
func (p *Project) SaveApprovals(list []Approval) error {
	return p.writeState(approvalsFile, &approvalsY{Approvals: list})
}

// The verdicts of a judgment (spec 8.4). A line without a judgment is "not
// judged yet": that is not written down.
const (
	VerdictManual    = "manual"     // written into an extra configuration
	VerdictNotNeeded = "not_needed" // the new device does not need it
	VerdictHold      = "hold"       // to be decided later
)

// Judgment is one entry of state/judgments.yaml (appendix C 3.3): what a
// person decided about the unhandled lines of one shape.
type Judgment struct {
	Source  string   `yaml:"source,omitempty"` // "old" (also when empty) | "new"
	Pack    string   `yaml:"pack"`
	Path    []string `yaml:"path,omitempty"` // the shapes of the lines above, the outermost first
	Shape   string   `yaml:"shape"`
	Devices string   `yaml:"devices"` // "*", or host names with commas between
	Verdict string   `yaml:"verdict"`
	Reason  string   `yaml:"reason,omitempty"`
	By      string   `yaml:"by"`
	At      string   `yaml:"at"`
}

type judgmentsY struct {
	Judgments []Judgment `yaml:"judgments"`
}

const judgmentsFile = "state/judgments.yaml"

// Judgments reads state/judgments.yaml (empty if the file is not there).
func (p *Project) Judgments() ([]Judgment, error) {
	var y judgmentsY
	if err := p.readState(judgmentsFile, &y); err != nil {
		return nil, err
	}
	for i, j := range y.Judgments {
		where := fmt.Sprintf("%s の %d 件目", judgmentsFile, i+1)
		switch {
		case j.Source != "" && j.Source != "old" && j.Source != "new":
			return nil, fmt.Errorf("%s: source は old か new です（「%s」）", where, j.Source)
		case j.Pack == "" || j.Shape == "" || j.Devices == "":
			return nil, fmt.Errorf("%s: pack・shape・devices が要ります", where)
		case j.Verdict != VerdictManual && j.Verdict != VerdictNotNeeded && j.Verdict != VerdictHold:
			return nil, fmt.Errorf("%s: verdict は manual・not_needed・hold のどれかです（「%s」）", where, j.Verdict)
		case j.Verdict == VerdictNotNeeded && j.Reason == "":
			return nil, fmt.Errorf("%s: not_needed（要らない）には reason が要ります", where)
		case j.By == "":
			return nil, fmt.Errorf("%s: by が要ります", where)
		}
	}
	return y.Judgments, nil
}

// SaveJudgments writes state/judgments.yaml, in the order given.
func (p *Project) SaveJudgments(list []Judgment) error {
	return p.writeState(judgmentsFile, &judgmentsY{Judgments: list})
}

// NextApprovalID gives the ID for a new approval: one after the largest.
func NextApprovalID(list []Approval) string {
	max := 0
	for _, a := range list {
		var n int
		if _, err := fmt.Sscanf(a.ID, "a-%d", &n); err == nil && n > max {
			max = n
		}
	}
	return fmt.Sprintf("a-%04d", max+1)
}

// ListFile is what state/customer.yaml keeps of a customer's list: the mark
// of what it held when it was last read (S-505).
type ListFile struct {
	File   string `yaml:"file"`
	SHA256 string `yaml:"sha256"`
	At     string `yaml:"at"`
}

type listsY struct {
	Files []ListFile `yaml:"files"`
}

const listsFile = "state/customer.yaml"

// ListFiles reads state/customer.yaml (empty if the file is not there).
func (p *Project) ListFiles() ([]ListFile, error) {
	var y listsY
	if err := p.readState(listsFile, &y); err != nil {
		return nil, err
	}
	return y.Files, nil
}

// SaveListFiles writes state/customer.yaml, by file name.
func (p *Project) SaveListFiles(list []ListFile) error {
	y := listsY{Files: append([]ListFile{}, list...)}
	sort.Slice(y.Files, func(i, j int) bool { return y.Files[i].File < y.Files[j].File })
	return p.writeState(listsFile, &y)
}

// draftFile holds what a person changed in the tables of the screen and did
// not save into the parameter book yet (spec 14.6).
const draftFile = "state/draft.yaml"

// ReadDraft reads the draft into a value; a draft that is not there leaves
// it as it is.
func (p *Project) ReadDraft(into any) error { return p.readState(draftFile, into) }

// WriteDraft writes the draft.
func (p *Project) WriteDraft(from any) error { return p.writeState(draftFile, from) }

// RemoveDraft takes the draft away; none is no error.
func (p *Project) RemoveDraft() error {
	if err := os.Remove(p.Path(draftFile)); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("%s を消せません: %v", draftFile, err)
	}
	return nil
}

// BookWritable tells, before anything is written, whether the parameter book
// can be replaced: a book that Excel holds open cannot (spec 4.2).
func (p *Project) BookWritable() error {
	f, err := os.OpenFile(p.BookPath(), os.O_RDWR, 0)
	if err == nil {
		f.Close()
		return nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return fmt.Errorf("%s に書けません。Excel を閉じてください", p.Config.Book)
}
