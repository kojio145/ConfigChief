package ui

import (
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/kojio145/configchief/internal/fill"
	"github.com/kojio145/configchief/internal/listimport"
	"github.com/kojio145/configchief/internal/ops"
	"github.com/kojio145/configchief/internal/report"
)

// The screen "ひな形から作る" (spec 9.7, 14.7): the templates of fill/, the
// variable rules of vars/, the filling and a first template from one device.

// TemplateInfo is one template.
type TemplateInfo struct {
	Path string `json:"path"`
	Name string `json:"name"`
	Text string `json:"text"`
	Err  string `json:"err"`
}

// VarRuleInfo is one variable rule file; Rule is nil for a file that could
// not be read as one.
type VarRuleInfo struct {
	Path     string              `json:"path"`
	Name     string              `json:"name"`
	Rule     *listimport.VarRule `json:"rule"`
	Problems []string            `json:"problems"`
}

// NameGroup is a group of names a template may use: the variables of a rule,
// or the columns of a table of the book.
type NameGroup struct {
	Title string   `json:"title"`
	Each  string   `json:"each"` // what {{#each …}} takes for the group
	Names []string `json:"names"`
}

// FillView is what the screen starts from.
type FillView struct {
	Templates []TemplateInfo `json:"templates"`
	Rules     []VarRuleInfo  `json:"rules"`
	Lists     []string       `json:"lists"`   // input/customer, by name
	Configs   []string       `json:"configs"` // input/old, new and logs
	Names     []NameGroup    `json:"names"`
	Book      bool           `json:"book"`
	Steps     []string       `json:"steps"`
	Ops       []string       `json:"ops"`
}

func fullVarRule(r *listimport.VarRule) *listimport.VarRule {
	if r == nil {
		return nil
	}
	c := *r
	if c.Exclude == nil {
		c.Exclude = []listimport.VarCond{}
	}
	fix := func(f *listimport.Field) {
		f.Fallback = strs(f.Fallback)
		f.Steps = append([]listimport.Step{}, f.Steps...)
		for k := range f.Steps {
			if f.Steps[k].Rows == nil {
				f.Steps[k].Rows = []listimport.MapRow{}
			}
		}
	}
	fix(&c.Device)
	c.Vars = append([]listimport.Var{}, c.Vars...)
	for i := range c.Vars {
		fix(&c.Vars[i].Field)
	}
	c.Also = append([]listimport.AlsoDevice{}, c.Also...)
	for i := range c.Also {
		fix(&c.Also[i].Device)
		c.Also[i].Vars = append([]listimport.Var{}, c.Also[i].Vars...)
		for j := range c.Also[i].Vars {
			fix(&c.Also[i].Vars[j].Field)
		}
	}
	return &c
}

// Fill lists the templates and the variable rules of the case.
func (s *Session) Fill() (*FillView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	v := &FillView{Templates: []TemplateInfo{}, Rules: []VarRuleInfo{}, Lists: []string{}, Configs: []string{}, Names: []NameGroup{},
		Steps: listimport.StepTypes, Ops: listimport.CondOps}
	ts, err := e.FillTemplates()
	if err != nil {
		return nil, err
	}
	for _, t := range ts {
		v.Templates = append(v.Templates, TemplateInfo{Path: t.Path, Name: t.Name, Text: t.Text, Err: t.Err})
	}
	vs, err := e.VarFiles()
	if err != nil {
		return nil, err
	}
	for _, f := range vs {
		info := VarRuleInfo{Path: f.Path, Rule: fullVarRule(f.Rule), Problems: strs(f.Problems)}
		if f.Rule != nil {
			info.Name = f.Rule.Name
			g := NameGroup{Title: "変数ルール " + f.Rule.Name, Each: f.Rule.Name}
			for _, x := range f.Rule.Vars {
				g.Names = append(g.Names, x.Name)
			}
			g.Names = strs(g.Names)
			v.Names = append(v.Names, g)
		}
		v.Rules = append(v.Rules, info)
	}
	if _, lists, err := e.MappingFiles(); err == nil {
		v.Lists = strs(lists)
	}
	for _, side := range ops.FillSides {
		if ins, err := e.Proj.Inputs(side); err == nil {
			v.Configs = append(v.Configs, ins...)
		}
	}
	if _, err := os.Stat(e.Proj.BookPath()); err == nil {
		v.Book = true
		for _, t := range e.Schema.Tables {
			g := NameGroup{Title: "ブック " + t.Title.JA, Each: t.Name}
			for _, c := range t.Columns {
				g.Names = append(g.Names, t.Name+"."+c.Name)
			}
			v.Names = append(v.Names, g)
		}
	}
	return v, nil
}

// TemplateSave writes a template and gives its path and the fault of its
// form ("" if none).
func (s *Session) TemplateSave(path, name, text string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	rel, fault, err := e.SaveTemplate(path, name, text)
	if err != nil {
		return nil, err
	}
	return []string{rel, fault}, nil
}

// TemplateDelete moves a template to state/backup.
func (s *Session) TemplateDelete(path string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return "", err
	}
	return e.DeleteTemplate(path, s.now())
}

// TemplateCheck reads a template as it is in the editor and gives its
// faults, each with its line: the form, and the names the variable rules
// and the book do not have.
func (s *Session) TemplateCheck(text string) ([]string, error) {
	tp, err := fill.Parse("t", text)
	if err != nil {
		return []string{err.Error()}, nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	src, err := e.FillSourcesForCheck()
	if err != nil {
		return []string{err.Error()}, nil
	}
	_, errs := fill.Bind(tp, src)
	out := []string{}
	for _, x := range errs {
		out = append(out, x.Error())
	}
	return out, nil
}

// VarSave writes a variable rule and gives its path.
func (s *Session) VarSave(path, name string, r listimport.VarRule) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return "", err
	}
	return e.SaveVarRule(path, name, &r)
}

// VarDelete moves a variable rule to state/backup.
func (s *Session) VarDelete(path string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return "", err
	}
	return e.DeleteVarRule(path, s.now())
}

// VarTryView is a reading of a variable rule that writes nothing.
type VarTryView struct {
	Faults   []string `json:"faults"`
	File     string   `json:"file"`
	Lines    int      `json:"lines"`
	Excluded int      `json:"excluded"`
	Devices  int      `json:"devices"`
	Missing  []string `json:"missing"`
	Warnings []string `json:"warnings"`
	// Sample is the first lines as the rule reads them: the line, which
	// device of it, the device (a continuing row marked), each variable as
	// "raw → value", and why it was left out.
	Sample  Table    `json:"sample"`
	Table   Table    `json:"table"`
	Unknown []string `json:"unknown"`
	Known   int      `json:"known"`
	NoBook  bool     `json:"noBook"`
}

// VarTry reads a rule's list and shows each device's values.
func (s *Session) VarTry(r listimport.VarRule) (*VarTryView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	o, err := e.TryVarRule(&r)
	if err != nil {
		return nil, err
	}
	name := "機器ごとの値"
	if o.Total > len(o.Rows) {
		name += "（先頭の " + itoa(len(o.Rows)) + " 行）"
	}
	sample := report.Sheet{Name: "先頭の行の読み方", Header: []string{"一覧の行", "台", "機器"}}
	for _, v := range r.Vars {
		sample.Header = append(sample.Header, v.Name)
	}
	sample.Header = append(sample.Header, "除外")
	for _, l := range o.Sample {
		dev := l.Device
		if l.Cont {
			dev += "（続きの行）"
		}
		row := []string{itoa(l.Line), itoa(l.Unit) + " 台目", dev}
		for _, v := range r.Vars {
			raw, val := l.Raw[v.Name], l.Values[v.Name]
			cell := val
			if strings.TrimSpace(raw) != val {
				cell = raw + " → " + val
			}
			if val == "" {
				cell = strings.TrimSpace(raw + " →（空）")
			}
			row = append(row, cell)
		}
		row = append(row, l.Excluded)
		sample.Rows = append(sample.Rows, row)
		mark := ""
		if l.Excluded != "" {
			mark = "ignored"
		}
		sample.Marks = append(sample.Marks, mark)
	}
	return &VarTryView{Faults: strs(o.Faults), File: o.File, Lines: o.Lines, Excluded: o.Excluded, Devices: o.Devices,
		Missing: strs(o.Missing), Warnings: strs(o.Warnings), Sample: table(sample), Table: table(report.Sheet{Name: name, Header: o.Header, Rows: o.Rows}),
		Unknown: strs(o.Unknown), Known: o.Known, NoBook: o.NoBook}, nil
}

// FillDeviceView is the outcome of one device.
type FillDeviceView struct {
	Name    string   `json:"name"`
	Result  string   `json:"result"` // made | unchanged | not
	Why     string   `json:"why"`
	File    string   `json:"file"`
	Lines   int      `json:"lines"`
	Text    string   `json:"text"`
	Values  []string `json:"values"`
	Added   []string `json:"added"`
	Removed []string `json:"removed"`
	Kept    []string `json:"kept"`
	Left    []string `json:"left"`
}

// FillRunView is the outcome of filling a template.
type FillRunView struct {
	Template string           `json:"template"`
	Diff     string           `json:"diff"`
	Devices  []FillDeviceView `json:"devices"`
	Notes    []string         `json:"notes"`
	Dir      string           `json:"dir"`
	Report   string           `json:"report"`
	Made     int              `json:"made"`
	NotMade  int              `json:"notMade"`
}

func items(list []fill.Item, removal bool) []string {
	out := []string{}
	for _, it := range list {
		t := strings.Join(append(append([]string{}, it.Path...), it.Text), " > ")
		if removal && it.Cmd != "" {
			t += "　→ " + it.Cmd
		}
		out = append(out, t)
	}
	return out
}

// FillRun fills a template for its devices (all when devices is empty).
func (s *Session) FillRun(template, diff string, devices []string) (*FillRunView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	o, err := e.Fill(ops.FillOptions{Template: template, Diff: diff, Devices: devices}, s.now())
	if err != nil {
		return nil, err
	}
	v := &FillRunView{Template: o.Template, Diff: o.Diff, Devices: []FillDeviceView{}, Notes: strs(o.Notes), Dir: o.Dir, Report: o.Report.HTML, NotMade: o.NotMade()}
	for _, d := range o.Devices {
		r := "not"
		switch {
		case d.Made:
			r = "made"
			v.Made++
		case d.Unchanged:
			r = "unchanged"
		}
		v.Devices = append(v.Devices, FillDeviceView{Name: d.Name, Result: r, Why: d.Why, File: d.File, Lines: d.Lines, Text: d.Text,
			Values: strs(d.Values), Added: items(d.Added, false), Removed: items(d.Removed, true), Kept: items(d.Kept, false), Left: strs(d.Left)})
	}
	return v, nil
}

// FillDraftView is a first template made from one device.
type FillDraftView struct {
	File     string   `json:"file"`
	Device   string   `json:"device"`
	Text     string   `json:"text"`
	Replaced []string `json:"replaced"`
	Notes    []string `json:"notes"`
}

// FillDraft makes a first template from a device's configuration (nothing is
// written).
func (s *Session) FillDraft(file, device string) (*FillDraftView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	d, err := e.FillDraft(file, device)
	if err != nil {
		return nil, err
	}
	v := &FillDraftView{File: d.File, Device: d.Device, Text: d.Text, Replaced: []string{}, Notes: strs(d.Notes)}
	for n, c := range d.Replaced {
		v.Replaced = append(v.Replaced, "{{"+n+"}} "+itoa(c)+" か所")
	}
	sort.Strings(v.Replaced)
	return v, nil
}

func itoa(n int) string { return strconv.Itoa(n) }
