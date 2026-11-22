package ui

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/ops"
	"github.com/kojio145/configchief/internal/report"
)

// DeviceResult is what the generation did for one device.
type DeviceResult struct {
	Host     string `json:"host"` // "" = the findings that belong to no one device
	Label    string `json:"label"`
	Pack     string `json:"pack"`
	State    string `json:"state"` // ops.GenWritten and the like ("" for the row of no device)
	Word     string `json:"word"`
	File     string `json:"file"` // the configuration, from the case folder ("" = none)
	Why      string `json:"why"`
	Errors   int    `json:"errors"`   // not approved
	Warnings int    `json:"warnings"` // not approved
	Approved int    `json:"approved"`
	Info     int    `json:"info"`
	Mark     string `json:"mark"`
}

// FindingView is a finding with where it points and what the case says
// about it.
type FindingView struct {
	Key      string `json:"key"` // the fingerprint
	Rule     string `json:"rule"`
	Title    string `json:"title"` // what the check looks at
	Kind     string `json:"kind"`
	Severity string `json:"severity"` // check.Error | check.Warning | check.Info
	Word     string `json:"word"`     // the severity in words
	Device   string `json:"device"`
	Table    string `json:"table"`
	Row      string `json:"row"`
	Column   string `json:"column"`
	At       string `json:"at"`   // the cell of the book
	Line     int    `json:"line"` // the line of the configuration (0 = none)
	Text     string `json:"text"` // that line, the secrets masked
	Message  string `json:"message"`
	Fix      string `json:"fix"` // how the finding is put right
	Evidence []Pair `json:"evidence"`
	// State: "open" (a person has to look at it), "fix" (open, and it cannot
	// be approved: the value or the text is put right), "approved", "info".
	State     string `json:"state"`
	StateWord string `json:"stateWord"`
	// CanScope: the finding can be approved together with the others of its
	// check (appendix C 3.2).
	CanScope bool `json:"canScope"`
	// The approval that covers it ("" = none).
	Approval string `json:"approval"`
	Reason   string `json:"reason"`
	By       string `json:"by"`
	When     string `json:"when"`
	Mark     string `json:"mark"`
}

// ApprovalView is an approval of the case and what it covers now.
type ApprovalView struct {
	ID      string `json:"id"`
	Target  string `json:"target"`
	Rule    string `json:"rule"`
	Devices string `json:"devices"`
	Reason  string `json:"reason"`
	By      string `json:"by"`
	At      string `json:"at"`
	Hits    int    `json:"hits"`
	Note    string `json:"note"` // why it does not count, or that it covers nothing
	Mark    string `json:"mark"`
}

// ChangesView is what changed since the generation before (spec 12.4).
type ChangesView struct {
	Has    bool     `json:"has"` // false = there was nothing to compare with
	Before string   `json:"before"`
	Counts []Pair   `json:"counts"`
	Notes  []string `json:"notes"`
	Tables []Table  `json:"tables"`
	Report Files    `json:"report"`
}

// ResultView is the outcome of a generation or of the checks (spec 14.2 結果).
type ResultView struct {
	Title string `json:"title"`
	// Message tells what the last step did; Notice that the files of Dir are
	// older than what the screen shows ("" = they are not).
	Message string `json:"message"`
	Notice  string `json:"notice"`
	// Dir is the folder the last step wrote to; Configs tells that it holds
	// configurations.
	Dir        string         `json:"dir"`
	Configs    bool           `json:"configs"`
	Unverified string         `json:"unverified"` // the mark that was put ("" = none)
	Report     Files          `json:"report"`
	Counts     []Pair         `json:"counts"`
	Notes      []string       `json:"notes"`
	Devices    []DeviceResult `json:"devices"`
	Findings   []FindingView  `json:"findings"`
	Checks     Table          `json:"checks"`
	Approvals  []ApprovalView `json:"approvals"`
	Changes    ChangesView    `json:"changes"`
	Left       int            `json:"left"` // what a person has to look at
	By         string         `json:"by"`   // the name the last approval was given with
}

// written is what the last step that wrote files left in the case folder.
type written struct {
	dir        string
	configs    bool
	unverified string
	report     Files
	changes    ChangesView
}

// Generate reads the parameter book, checks it and writes the configurations,
// the check results and the list of changes, as the command "generate" does.
func (s *Session) Generate() (*ResultView, error) { return s.generate(false) }

// Check runs the checks and writes the check results but no configuration,
// as the command "check" does.
func (s *Session) Check() (*ResultView, error) { return s.generate(true) }

func (s *Session) generate(checkOnly bool) (*ResultView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	now := s.now()
	o, err := e.Generate(ops.GenOptions{CheckOnly: checkOnly}, now)
	if err != nil {
		return nil, err
	}
	s.shown = o
	s.wrote = written{dir: o.Dir, configs: !checkOnly, unverified: o.Unverified,
		report: Files{o.Report.XLSX, o.Report.HTML}, changes: changesView(e, o, now)}
	v := s.result(e, o, now)
	v.Title = "生成とチェックの結果"
	if checkOnly {
		v.Title = "チェックの結果（コンフィグは書き出していません）"
	}
	return v, nil
}

// Approve approves a finding of the result that is shown. scope: "" = this
// finding only, "device" = the findings of the same check and table on its
// device, "all" = those on every device. The checks then run again; no file
// is written by them.
func (s *Session) Approve(key, scope, reason, by string) (*ResultView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.env == nil || s.shown == nil {
		return nil, errors.New("先に「生成とチェック」か「チェックだけ」を行ってください")
	}
	var f *ops.Judged
	for i := range s.shown.Items {
		if s.shown.Items[i].Key == key {
			f = &s.shown.Items[i]
			break
		}
	}
	if f == nil {
		return nil, errors.New("その指摘は、今の結果にありません。チェックをやり直してください")
	}
	req := ops.ApproveRequest{Reason: reason, By: by}
	switch scope {
	case "":
		req.Finding = key
	case "device":
		if f.Device == "" {
			return nil, errors.New("機器を 1 台に決められない指摘です。「この指摘だけ」か「全機器」で承認してください")
		}
		req.Rule, req.Devices, req.Table = f.RuleID, f.Device, f.Ref.Table
	case "all":
		req.Rule, req.Devices, req.Table = f.RuleID, "*", f.Ref.Table
	default:
		return nil, fmt.Errorf("承認の範囲「%s」は使えません", scope)
	}
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	now := s.now()
	a, err := e.Approve(req, now)
	if err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(by); name != s.name {
		s.name = name
		s.save()
	}
	return s.recheck(e, now, fmt.Sprintf("承認しました（%s。当たった指摘 %d 件）", a.Approval.ID, a.Hits))
}

// Unapprove takes an approval back. The checks then run again; no file is
// written by them.
func (s *Session) Unapprove(id string) (*ResultView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.env == nil || s.shown == nil {
		return nil, errors.New("先に「生成とチェック」か「チェックだけ」を行ってください")
	}
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	now := s.now()
	if err := e.Unapprove(id, now); err != nil {
		return nil, err
	}
	return s.recheck(e, now, fmt.Sprintf("承認 %s を取り消しました", id))
}

// recheck runs the checks again after the approvals changed, and writes
// nothing: the files of the last step stay as they are, and the screen says
// so (spec 14.2).
func (s *Session) recheck(e *ops.Env, now time.Time, done string) (*ResultView, error) {
	o, err := e.Generate(ops.GenOptions{CheckOnly: true, Dry: true}, now)
	if err != nil {
		return nil, fmt.Errorf("%s。ただし、チェックをやり直せませんでした: %v", done, err)
	}
	s.shown = o
	v := s.result(e, o, now)
	v.Title = "チェックの結果（承認の変更を反映）"
	v.Message = done
	what := "チェック結果表"
	if s.wrote.configs {
		what = "コンフィグ・チェック結果表・UNVERIFIED.txt"
	}
	v.Notice = fmt.Sprintf("承認の変更を反映して、チェックをやり直しました（ファイルは書いていません）。出力フォルダ %s の%sは、変更の前のままです。", s.wrote.dir, what)
	if s.wrote.configs {
		v.Notice += "もう一度「生成とチェック」を押すと、新しく書き出します。"
	} else {
		v.Notice += "もう一度「チェックだけ」か「生成とチェック」を押すと、新しく書き出します。"
	}
	return v, nil
}

func changesView(e *ops.Env, o *ops.GenerateOutcome, now time.Time) ChangesView {
	v := ChangesView{Counts: []Pair{}, Notes: []string{}, Tables: []Table{}}
	d := e.ChangesDoc(o, now)
	if d == nil {
		return v
	}
	v.Has = true
	v.Before = o.Changes.Before
	v.Counts, v.Notes = pairs(d.Counts), strs(d.Notes)
	v.Report = Files{o.Changes.Report.XLSX, o.Changes.Report.HTML}
	for _, sh := range d.Sheets {
		v.Tables = append(v.Tables, table(sh))
	}
	return v
}

// result makes the view of an outcome. What is in the case folder comes
// from the last step that wrote.
func (s *Session) result(e *ops.Env, o *ops.GenerateOutcome, now time.Time) *ResultView {
	d := e.CheckDoc(o, now)
	w := s.wrote
	v := &ResultView{
		Dir: w.dir, Configs: w.configs, Unverified: w.unverified, Report: w.report, Changes: w.changes,
		Counts: pairs(d.Counts), Notes: strs(d.Notes),
		Devices: []DeviceResult{}, Findings: []FindingView{}, Approvals: []ApprovalView{},
		Checks: table(report.Sheet{Name: "チェック項目"}),
		Left:   o.Findings(), By: s.name,
	}
	for _, sh := range d.Sheets {
		if sh.Name == v.Checks.Name {
			v.Checks = table(sh)
		}
	}

	per := map[string]*DeviceResult{}
	for _, f := range o.Items {
		c := per[f.Device]
		if c == nil {
			c = &DeviceResult{}
			per[f.Device] = c
		}
		fv := findingView(e, o, f)
		switch fv.State {
		case "info":
			c.Info++
		case "approved":
			c.Approved++
		default:
			if f.Severity == check.Error {
				c.Errors++
			} else {
				c.Warnings++
			}
		}
		v.Findings = append(v.Findings, fv)
	}
	row := func(r DeviceResult) {
		if c := per[r.Host]; c != nil {
			r.Errors, r.Warnings, r.Approved, r.Info = c.Errors, c.Warnings, c.Approved, c.Info
		}
		switch {
		case r.Errors > 0 || r.State == ops.GenFailed || r.State == ops.GenBlocked:
			r.Mark = report.MarkUnknown
		case r.Warnings > 0:
			r.Mark = report.MarkWarn
		}
		v.Devices = append(v.Devices, r)
	}
	if per[""] != nil {
		row(DeviceResult{Label: "（案件全体）", Why: "機器を 1 台に決められない指摘です"})
	}
	for _, dev := range o.Devices {
		row(DeviceResult{Host: dev.Host, Label: dev.Host, Pack: dev.Pack, State: dev.State, Word: ops.GenStateWord(dev.State), File: dev.File, Why: dev.Why})
	}

	for _, a := range o.Approvals {
		av := ApprovalView{ID: a.ID, Target: ops.ApprovalTarget(a), Rule: a.Rule, Devices: a.Device,
			Reason: a.Reason, By: a.By, At: a.At, Hits: o.Hits[a.ID], Note: o.Refused[a.ID]}
		if a.Scope != nil {
			av.Rule, av.Devices = a.Scope.Rule, a.Scope.Devices
		}
		switch {
		case av.Note == "" && av.Hits == 0 && ops.MigrationApproval(a):
			av.Note = ops.MigrationApprovalNote
		case av.Note == "" && av.Hits == 0 && ops.SourceApproval(a):
			av.Note = ops.SourceApprovalNote
		case av.Note == "" && av.Hits == 0:
			av.Note, av.Mark = "今回、当たる指摘はありません（指摘がなくなったか、根拠の値が変わりました）", report.MarkWarn
		case av.Note != "":
			av.Mark = report.MarkWarn
		}
		v.Approvals = append(v.Approvals, av)
	}
	return v
}

func findingView(e *ops.Env, o *ops.GenerateOutcome, f ops.Judged) FindingView {
	p := e.FindingPlace(o, f.Finding)
	rule, _ := check.Lookup(f.RuleID)
	fv := FindingView{
		Key: f.Key, Rule: f.RuleID, Title: rule.Title, Kind: check.KindTitle(check.Kind(f.RuleID)),
		Severity: f.Severity, Word: check.SeverityTitle(f.Severity), Device: f.Device,
		Table: p.Table, Row: p.Row, Column: p.Column, At: f.At, Line: f.Line, Text: p.Text,
		Message: f.Message, Fix: rule.Fix, Evidence: []Pair{},
	}
	for _, kv := range f.Evidence {
		fv.Evidence = append(fv.Evidence, Pair{kv.Name, kv.Value})
	}
	switch {
	case f.Approval != nil:
		fv.State, fv.StateWord = "approved", "承認済み"
		fv.Approval, fv.Reason, fv.By, fv.When = f.Approval.ID, f.Approval.Reason, f.Approval.By, f.Approval.At
	case f.Severity == check.Info:
		fv.State, fv.StateWord = "info", "情報"
	case !check.Approvable(f.RuleID):
		fv.State, fv.StateWord, fv.Mark = "fix", "要修正", report.MarkUnknown
	default:
		fv.State, fv.StateWord = "open", "未承認"
		fv.CanScope = f.Severity != check.Error || check.ScopeOK(f.RuleID)
		fv.Mark = report.MarkWarn
		if f.Severity == check.Error {
			fv.Mark = report.MarkUnknown
		}
	}
	return fv
}
