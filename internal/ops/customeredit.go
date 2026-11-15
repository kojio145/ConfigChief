package ops

import (
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/listimport"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/report"
	"github.com/kojio145/configchief/internal/sheet"
)

// What the editor of the column mappings needs (spec 14.3): the files, a
// look at a list, a reading that writes nothing, and saving.

// MappingFiles lists the column mappings of the case, each with its faults,
// and the lists of input/customer by name.
func (e *Env) MappingFiles() (files []listimport.File, lists []string, err error) {
	if files, err = listimport.Files(os.DirFS(e.Proj.Path("mappings")), "mappings", e.Schema); err != nil {
		return nil, nil, err
	}
	inputs, err := e.Proj.Inputs("customer")
	if err != nil {
		return nil, nil, err
	}
	for _, p := range inputs {
		lists = append(lists, path.Base(p))
	}
	return files, lists, nil
}

// The part of a sheet the editor shows.
const (
	sampleRows = 30
	sampleCols = 40
	tryRows    = 100
)

// ListSample is a look at the list a rule names.
type ListSample struct {
	File   string // the list, from the case folder
	Sheets []string
	Sheet  string
	// HeaderRow is the header row of the rule, or a guess if it has none
	// yet; Suggest then holds columns for the table of the devices.
	HeaderRow int
	Suggest   []listimport.Field
	Headers   []listimport.Header
	// Grid is the top of the sheet, merged areas filled as the rule reads
	// them. The values under a header that goes to a secret column are
	// masked.
	Grid     [][]string
	Rows     int // the rows of the sheet
	Warnings []string
	// Err tells why the table could not be cut out (the grid is still
	// shown: the header row is likely wrong).
	Err string
}

// readList opens the list a rule names and picks its sheet.
func (e *Env) readList(r *listimport.Rule) (file string, bk *sheet.Book, s *sheet.Sheet, err error) {
	inputs, err := e.Proj.Inputs("customer")
	if err != nil {
		return "", nil, nil, err
	}
	if file, err = listFile(r, inputs); err != nil {
		return "", nil, nil, err
	}
	if bk, err = sheet.ReadFile(e.Proj.Path(file)); err != nil {
		return "", nil, nil, fmt.Errorf("%s: %v", file, err)
	}
	if s, err = listimport.PickSheet(bk, r); err != nil {
		return "", nil, nil, fmt.Errorf("%s: %v", file, err)
	}
	return file, bk, s, nil
}

// secretHeaders gives the headers of a list that go to a secret column.
func (e *Env) secretHeaders(r *listimport.Rule) map[string]bool {
	out := map[string]bool{}
	for _, tg := range r.Rows {
		for i := range tg.Columns {
			f := &tg.Columns[i]
			if e.secret(tg.Table, f.To) {
				for _, h := range f.Sources() {
					out[strings.ToLower(strings.Join(strings.Fields(h), " "))] = true
				}
			}
		}
	}
	return out
}

// SampleList shows the list a rule names. The rule need not be whole: it is
// asked while a person writes it.
func (e *Env) SampleList(r *listimport.Rule) (*ListSample, error) {
	rule := *r
	rule.Validate(e.Schema) // the names of the book become the internal ones
	file, bk, s, err := e.readList(&rule)
	if err != nil {
		return nil, err
	}
	out := &ListSample{File: file, Sheet: s.Name, HeaderRow: rule.HeaderRow, Warnings: append([]string{}, bk.Warnings...), Rows: len(s.Rows)}
	for i := range bk.Sheets {
		out.Sheets = append(out.Sheets, bk.Sheets[i].Name)
	}
	var t *listimport.Table
	if rule.HeaderRow < 1 {
		out.HeaderRow, out.Suggest, t = listimport.Suggest(s, rule)
		rule.HeaderRow, rule.DataRow = out.HeaderRow, 0
	}
	if t == nil {
		if t, err = listimport.ReadTable(s, &rule); err != nil {
			out.Err = err.Error()
		}
	}
	secret := e.secretHeaders(&rule)
	hide := map[int]bool{}
	if t != nil {
		out.Headers = t.Headers
		out.Warnings = append(out.Warnings, t.Warnings...)
		for _, h := range t.Headers {
			if secret[strings.ToLower(h.Name)] {
				hide[h.Col] = true
			}
		}
	}
	for i, row := range listimport.FilledRows(s, &rule) {
		if i == sampleRows {
			break
		}
		if len(row) > sampleCols {
			row = row[:sampleCols]
		}
		line := append([]string{}, row...)
		for c := range line {
			if hide[c+1] && i+1 > rule.HeaderRow && line[c] != "" {
				line[c] = masked
			}
		}
		out.Grid = append(out.Grid, line)
	}
	return out, nil
}

// ListTry is a reading of one list by a rule that wrote nothing.
type ListTry struct {
	// Faults are what is wrong with the rule itself: nothing was read then.
	Faults []string
	File   string
	Result *listimport.Result
	// Targets holds, per row of the rule, the rows it gave (the first ones).
	Targets []TryTarget
}

// TryTarget is what one row of a rule gave.
type TryTarget struct {
	Title  string
	Header []string
	Rows   [][]string // the line of the list first, then the cells
	Total  int
}

// TryList reads the list a rule names and gives what the rule makes of it.
// Nothing is written and nothing is compared.
func (e *Env) TryList(r *listimport.Rule) (*ListTry, error) {
	rule := *r
	out := &ListTry{}
	if out.Faults = rule.Validate(e.Schema); len(out.Faults) > 0 {
		return out, nil
	}
	file, _, s, err := e.readList(&rule)
	if err != nil {
		return nil, err
	}
	t, err := listimport.ReadTable(s, &rule)
	if err != nil {
		return nil, fmt.Errorf("%s: %v", file, err)
	}
	sc := e.forScreen()
	out.File = file
	out.Result = listimport.Convert(t, &rule, e.Schema, file, model.NewDataset("customer"))
	for i, tg := range rule.Rows {
		tt := TryTarget{Title: sc.tableTitle(tg.Table), Header: []string{"一覧の行"}}
		if tg.Label != "" {
			tt.Title += "（" + tg.Label + "）"
		}
		for _, f := range tg.Columns {
			tt.Header = append(tt.Header, sc.columnTitle(tg.Table, f.To))
		}
		for _, m := range out.Result.Made {
			if m.Target != i {
				continue
			}
			if tt.Total++; tt.Total > tryRows {
				continue
			}
			row := []string{fmt.Sprintf("%d", m.Line)}
			for _, f := range tg.Columns {
				row = append(row, sc.cellText(tg.Table, f.To, m.Row.Cells[f.To]))
			}
			tt.Rows = append(tt.Rows, row)
		}
		out.Targets = append(out.Targets, tt)
	}
	return out, nil
}

// SaveMapping writes a column mapping (spec 4.2, 14.3): over the file it was
// read from, or for a new one as mappings/<name of the list>.yaml, with a
// number after the name if that file is there. A rule with a fault is not
// written. It gives the path.
func (e *Env) SaveMapping(rel string, r *listimport.Rule) (string, error) {
	rule := *r
	if faults := rule.Validate(e.Schema); len(faults) > 0 {
		return "", fmt.Errorf("この列対応は保存できません:\n  %s", strings.Join(faults, "\n  "))
	}
	if rel != "" {
		// Only a mapping file of the case that is there is written over.
		ok := false
		files, _, err := e.MappingFiles()
		if err != nil {
			return "", err
		}
		for _, f := range files {
			ok = ok || (f.Path == rel && strings.HasSuffix(strings.ToLower(rel), ".yaml"))
		}
		if !ok {
			return "", fmt.Errorf("%s は、この案件の列対応ファイルではありません", rel)
		}
	} else {
		name := strings.TrimSuffix(rule.File, path.Ext(rule.File))
		name = strings.Map(func(c rune) rune {
			if strings.ContainsRune(`\/:*?"<>|`, c) || c < ' ' {
				return '_'
			}
			return c
		}, strings.TrimSpace(name))
		if strings.Trim(name, "_. ") == "" {
			name = "list"
		}
		rel = "mappings/" + name + ".yaml"
		for n := 2; ; n++ {
			if _, err := os.Stat(e.Proj.Path(rel)); err != nil {
				break
			}
			rel = fmt.Sprintf("mappings/%s_%d.yaml", name, n)
		}
	}
	data, err := listimport.Marshal(&rule)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(e.Proj.Path("mappings"), 0o755); err != nil {
		return "", fmt.Errorf("mappings を作れません: %v", err)
	}
	head := "# お客様の一覧の列対応（仕様書 6.5）。画面の「お客様の一覧」で保存したものです。\n"
	if err := os.WriteFile(e.Proj.Path(rel), append([]byte(head), data...), 0o644); err != nil {
		return "", fmt.Errorf("%s を書けません: %v", rel, err)
	}
	return rel, nil
}

// CustomerDoc gives the outcome of reading the lists as a screen shows it:
// the lists of the report, with the secrets masked.
// LoadDoc and VerifyDoc are the reports of the checks of the logs as the
// screen shows them: the secret values are hidden whatever the case says.
func (e *Env) LoadDoc(o *LoadOutcome, now time.Time) *report.Doc { return e.forScreen().loadDoc(o, now) }

func (e *Env) VerifyDoc(o *VerifyOutcome, now time.Time) *report.Doc {
	return e.forScreen().verifyDoc(o, now)
}

func (e *Env) CustomerDoc(o *CustomerOutcome, now time.Time) *report.Doc {
	return e.forScreen().customerDoc(o, now)
}
