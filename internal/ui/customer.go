package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kojio145/configchief/internal/listimport"
	"github.com/kojio145/configchief/internal/ops"
	"github.com/kojio145/configchief/internal/report"
)

// The screen of the lists of the customer (spec 6.5, 14.3): the comparison
// with the book, and the editor of the column mappings.

// ColumnInfo is a column of a table of the design data, for choosing where
// a column of a list goes.
type ColumnInfo struct {
	Name   string   `json:"name"`
	Title  string   `json:"title"`
	Key    bool     `json:"key"`
	Secret bool     `json:"secret"`
	Type   string   `json:"type"`
	Values []string `json:"values"` // the choices of an enum
	Note   string   `json:"note"`
}

// TableInfo is a table of the design data.
type TableInfo struct {
	Name    string       `json:"name"`
	Title   string       `json:"title"`
	Columns []ColumnInfo `json:"columns"`
}

// MappingInfo is one column mapping file. Rule is nil for a file that could
// not be read as one.
type MappingInfo struct {
	Path     string           `json:"path"`
	Rule     *listimport.Rule `json:"rule"`
	Problems []string         `json:"problems"`
}

// MappingsView is what the editor of the column mappings starts from.
type MappingsView struct {
	Mappings []MappingInfo `json:"mappings"`
	Files    []string      `json:"files"` // the lists of input/customer, by name
	Tables   []TableInfo   `json:"tables"`
	Steps    []string      `json:"steps"`
	Ops      []string      `json:"ops"`
}

// fullRule gives a rule whose lists are never nil: the page gets [].
func fullRule(r *listimport.Rule) *listimport.Rule {
	if r == nil {
		return nil
	}
	c := *r
	if c.Exclude == nil {
		c.Exclude = []listimport.Cond{}
	}
	c.Rows = append([]listimport.Target{}, c.Rows...)
	for i := range c.Rows {
		c.Rows[i].Columns = append([]listimport.Field{}, c.Rows[i].Columns...)
		for j := range c.Rows[i].Columns {
			f := &c.Rows[i].Columns[j]
			f.Fallback = strs(f.Fallback)
			f.Steps = append([]listimport.Step{}, f.Steps...)
			for k := range f.Steps {
				if f.Steps[k].Rows == nil {
					f.Steps[k].Rows = []listimport.MapRow{}
				}
			}
		}
	}
	return &c
}

// Mappings lists the column mappings of the case and what a new one can be
// made of.
func (s *Session) Mappings() (*MappingsView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	files, lists, err := e.MappingFiles()
	if err != nil {
		return nil, err
	}
	v := &MappingsView{Mappings: []MappingInfo{}, Files: strs(lists), Tables: []TableInfo{}, Steps: listimport.StepTypes, Ops: listimport.CondOps}
	for _, f := range files {
		v.Mappings = append(v.Mappings, MappingInfo{Path: f.Path, Rule: fullRule(f.Rule), Problems: strs(f.Problems)})
	}
	for _, t := range e.Schema.Tables {
		ti := TableInfo{Name: t.Name, Title: t.Title.JA}
		for _, c := range t.Columns {
			typ := c.Type
			if c.List {
				typ = "list<" + typ + ">"
			}
			ti.Columns = append(ti.Columns, ColumnInfo{Name: c.Name, Title: c.Title.JA, Key: t.IsKey(c.Name), Secret: c.Secret, Type: typ, Values: strs(c.Values), Note: c.Note.JA})
		}
		v.Tables = append(v.Tables, ti)
	}
	return v, nil
}

// SampleView is a look at the list a rule names (spec 14.3).
type SampleView struct {
	File      string              `json:"file"`
	Sheets    []string            `json:"sheets"`
	Sheet     string              `json:"sheet"`
	HeaderRow int                 `json:"headerRow"`
	Suggest   []listimport.Field  `json:"suggest"`
	Headers   []listimport.Header `json:"headers"`
	Grid      [][]string          `json:"grid"`
	Rows      int                 `json:"rows"`
	Warnings  []string            `json:"warnings"`
	Err       string              `json:"err"`
}

// ListSample shows the top of the list a rule names. The rule need not be
// whole yet.
func (s *Session) ListSample(r listimport.Rule) (*SampleView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	o, err := e.SampleList(&r)
	if err != nil {
		return nil, err
	}
	v := &SampleView{File: o.File, Sheets: strs(o.Sheets), Sheet: o.Sheet, HeaderRow: o.HeaderRow, Suggest: []listimport.Field{},
		Headers: []listimport.Header{}, Grid: [][]string{}, Rows: o.Rows, Warnings: strs(o.Warnings), Err: o.Err}
	for _, f := range o.Suggest {
		f.Fallback, f.Steps = strs(f.Fallback), append([]listimport.Step{}, f.Steps...)
		for k := range f.Steps {
			if f.Steps[k].Rows == nil {
				f.Steps[k].Rows = []listimport.MapRow{}
			}
		}
		v.Suggest = append(v.Suggest, f)
	}
	for _, h := range o.Headers {
		if h.Name != "" {
			v.Headers = append(v.Headers, h)
		}
	}
	for _, row := range o.Grid {
		v.Grid = append(v.Grid, strs(row))
	}
	return v, nil
}

// TryView is a reading of one list by a rule that wrote nothing.
type TryView struct {
	Faults   []string `json:"faults"`
	File     string   `json:"file"`
	Sheet    string   `json:"sheet"`
	Lines    int      `json:"lines"`
	Excluded int      `json:"excluded"`
	Rows     int      `json:"rows"`
	Missing  []string `json:"missing"`
	Warnings []string `json:"warnings"`
	Problems Table    `json:"problems"`
	Tables   []Table  `json:"tables"`
}

// ListTry reads the list a rule names and shows what the rule makes of it.
func (s *Session) ListTry(r listimport.Rule) (*TryView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	o, err := e.TryList(&r)
	if err != nil {
		return nil, err
	}
	v := &TryView{Faults: strs(o.Faults), File: o.File, Missing: []string{}, Warnings: []string{}, Tables: []Table{},
		Problems: table(report.Sheet{Name: "読めなかった値", Header: []string{"セル", "機器", "内容"}})}
	if o.Result == nil {
		return v, nil
	}
	res := o.Result
	v.Sheet, v.Lines, v.Excluded, v.Rows, v.Missing, v.Warnings = res.Sheet, res.Lines, res.Excluded, res.Rows, strs(res.Missing), strs(res.Warnings)
	for _, p := range res.Problems {
		v.Problems.Rows = append(v.Problems.Rows, []string{p.Addr, p.Device, p.Text})
		v.Problems.Marks = append(v.Problems.Marks, report.MarkWarn)
	}
	for _, tg := range o.Targets {
		name := fmt.Sprintf("%s: %d 行", tg.Title, tg.Total)
		if tg.Total > len(tg.Rows) {
			name += fmt.Sprintf("（先頭の %d 行）", len(tg.Rows))
		}
		v.Tables = append(v.Tables, table(report.Sheet{Name: name, Header: tg.Header, Rows: tg.Rows}))
	}
	return v, nil
}

// ListSave writes a column mapping: over the file at path, or as a new file
// if path is "". It gives the path of the file.
func (s *Session) ListSave(path string, r listimport.Rule) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return "", err
	}
	return e.SaveMapping(path, &r)
}

// CustomerView is the outcome of reading the lists of the customer and
// comparing them with the book. A row of a list is tied to its finding
// (S-501 to S-505) by the fingerprint in its last column.
type CustomerView struct {
	Title    string        `json:"title"`
	Counts   []Pair        `json:"counts"`
	Notes    []string      `json:"notes"`
	Tables   []Table       `json:"tables"`
	Report   Files         `json:"report"`
	Findings int           `json:"findings"`
	Items    []FindingView `json:"items"`
	By       string        `json:"by"`
}

// Export writes the delivery sheets ("sheet") or the table of values for the
// other side ("peer"), as the command "export" does.
func (s *Session) Export(what string, mask bool) (*ExportView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	secrets := "秘密の値は、そのまま書いています"
	if mask {
		secrets = "秘密の値は *** に伏せています"
	}
	v := &ExportView{What: what, Files: []string{}, Problems: []string{}, Notes: []string{}}
	switch what {
	case "sheet":
		o, err := e.ExportSheets(mask, s.now())
		if err != nil {
			return nil, err
		}
		v.Files, v.Problems, v.Notes = strs(o.Files), strs(o.Problems), strs(o.Notes)
		if len(o.Files) > 0 {
			v.Dir = o.Dir
		}
		v.Summary = fmt.Sprintf("納品用パラメータシートを %d ファイル出しました（%s）。", len(o.Files), secrets)
		if o.Standard {
			v.Summary += "templates にひな形（xlsx）が無いので、標準の形です。"
		}
		if len(o.Problems) > 0 {
			v.Summary += fmt.Sprintf("ひな形の誤りが %d 件あります（誤りのあるひな形の分は、出していません）。", len(o.Problems))
		}
	case "peer":
		o, err := e.ExportPeer(mask, s.now())
		if err != nil {
			return nil, err
		}
		v.Notes = strs(o.Notes)
		if o.Items == 0 {
			v.Summary = "管理外の相手（案件の中に機器がない相手）は、ありません。相手に渡すパラメータ表は作っていません。"
			break
		}
		v.Files = []string{o.Files.XLSX, o.Files.HTML}
		v.Dir = o.Files.XLSX[:strings.LastIndex(o.Files.XLSX, "/")]
		v.Summary = fmt.Sprintf("相手に渡すパラメータ表を出しました: 管理外の相手 %d、項目 %d（%s）。", o.Peers, o.Items, secrets)
	default:
		return nil, fmt.Errorf("出すものが分かりません: %s", what)
	}
	return v, nil
}

// Logs runs a check of the logs of input/logs, as the commands "logs"
// (what = "load") and "verify" (what = "verify") do: the report is written.
func (s *Session) Logs(what string) (*LogsView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	now := s.now()
	var d *report.Doc
	var files ops.Files
	var verdicts *ops.Verdicts
	switch what {
	case "load":
		o, err := e.LoadCheck(now)
		if err != nil {
			return nil, err
		}
		d, files, verdicts = e.LoadDoc(o, now), o.Report, o.Verdicts
	case "verify":
		o, err := e.Verify(now)
		if err != nil {
			return nil, err
		}
		d, files, verdicts = e.VerifyDoc(o, now), o.Report, o.Verdicts
	default:
		return nil, fmt.Errorf("調べるものが分かりません: %s", what)
	}
	v := &LogsView{What: what, Title: d.Title, Counts: pairs(d.Counts), Notes: strs(d.Notes), Tables: []Table{}, Report: Files{files.XLSX, files.HTML},
		Items: migFindings(e, verdicts), By: s.name}
	s.mig = verdicts
	for _, sh := range d.Sheets {
		v.Tables = append(v.Tables, table(sh))
	}
	return v, nil
}

// Customer reads the lists of the customer and compares them with the book,
// as the command "customer" does: the report is written.
func (s *Session) Customer() (*CustomerView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	now := s.now()
	o, err := e.Customer(true, now)
	if err != nil {
		return nil, err
	}
	d := e.CustomerDoc(o, now)
	v := &CustomerView{Title: d.Title, Counts: pairs(d.Counts), Notes: strs(d.Notes), Tables: []Table{}, Report: Files{o.Report.XLSX, o.Report.HTML},
		Findings: o.Findings(), Items: migFindings(e, o.Verdicts), By: s.name}
	s.mig = o.Verdicts
	for _, sh := range d.Sheets {
		v.Tables = append(v.Tables, table(sh))
	}
	return v, nil
}

// Reflect writes what the lists say into the parameter book: for the
// findings with the given fingerprints, or, with a check ("S-501", "S-503"),
// for all its findings that are not approved. It tells what was done; the
// screen then reads and compares again.
func (s *Session) Reflect(keys []string, rule string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.env == nil {
		return "", errors.New("案件を開いていません")
	}
	e, err := s.begin()
	if err != nil {
		return "", err
	}
	o, err := e.Reflect(ops.ReflectRequest{Findings: keys, Rule: rule}, s.now())
	if err != nil {
		return "", err
	}
	msg := fmt.Sprintf("パラメータブックに反映しました（足した行 %d、書き換えたセル %d）", o.Rows, o.Cells)
	if o.Backup != "" {
		msg += "。書き換える前の控え: " + o.Backup
	}
	var skipped []string
	for _, it := range o.Items {
		if it.Skipped != "" {
			skipped = append(skipped, fmt.Sprintf("%s %s: %s", it.Finding.RuleID, strings.TrimSpace(it.Finding.Device), it.Skipped))
		}
	}
	if len(skipped) > 0 {
		msg += fmt.Sprintf("\n反映しなかった指摘が %d 件あります:\n  %s", len(skipped), strings.Join(skipped, "\n  "))
	}
	return msg, nil
}
