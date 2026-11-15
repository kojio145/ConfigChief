package ops

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/compare"
	"github.com/kojio145/configchief/internal/convert"
	"github.com/kojio145/configchief/internal/listimport"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/schema"
	"github.com/kojio145/configchief/internal/sheet"
)

// CustomerList is one column mapping with the list it read.
type CustomerList struct {
	Rule *listimport.Rule
	// File is the list, from the case folder ("" = none was found), SHA256
	// the mark of what it holds, and Err why it could not be read.
	File, SHA256, Err string
	Result            *listimport.Result
}

// CustomerOutcome is what Customer did (spec 6.5).
type CustomerOutcome struct {
	Lists []CustomerList
	// Unmapped are the files of input/customer no mapping reads (a notice).
	Unmapped []string
	// Notes are what kept the reading from being whole: a mapping without
	// its file, a header that is not there, a device whose new name is not
	// decided. A person has to look at them.
	Notes []string
	// Data is what the lists say, by the new names.
	Data *model.Dataset
	// Book is the parameter book (nil = there is none: nothing was
	// compared), Design its data as a comparison takes it.
	Book      *book.Book
	Design    *model.Dataset
	Expand    []book.Problem
	BookNotes []string
	// The differences, each with its finding: rows only the lists have
	// (S-501), rows only the book has (S-502), cells that differ (S-503).
	OnlyList []RowFinding
	OnlyBook []RowFinding
	Cells    []CellFinding
	// Problems are the values that could not be read (S-504) and Changed the
	// lists that changed since they were last read (S-505).
	Problems []ProblemFinding
	Changed  []check.Finding
	// Verdicts is the findings with the approvals put on.
	Verdicts *Verdicts
	Report   Files
}

// RowFinding is a row one side lacks, with its finding.
type RowFinding struct {
	compare.RowDiff
	Finding check.Finding
}

// CellFinding is a cell that differs, with its finding. A is the list.
type CellFinding struct {
	compare.CellDiff
	Finding check.Finding
}

// ProblemFinding is a value that could not be read, with its finding.
type ProblemFinding struct {
	listimport.Problem
	Finding check.Finding
}

// Findings counts what a person has to look at.
func (o *CustomerOutcome) Findings() int {
	n := o.Verdicts.Open() + len(o.Notes) + len(o.Expand)
	if o.Book != nil {
		n += len(o.Book.Problems)
	}
	return n
}

func srcFinding(rule, device string, ref model.Ref, message string, evidence ...model.KV) check.Finding {
	r, _ := check.Lookup(rule)
	return check.New(rule, r.Severity, device, ref, message, evidence...)
}

// mappings reads the column mappings of the case. With a fault in one of
// them nothing is read (spec 6.5.1).
func (e *Env) mappings() ([]*listimport.Rule, error) {
	rules, problems := listimport.Load(os.DirFS(e.Proj.Path("mappings")), "mappings", e.Schema)
	if len(problems) > 0 {
		return nil, fmt.Errorf("列対応ファイル（mappings/）に誤りがあります。直してから、もう一度実行してください:\n  %s", strings.Join(problems, "\n  "))
	}
	return rules, nil
}

// listFile finds the file of input/customer a mapping names.
func listFile(r *listimport.Rule, inputs []string) (string, error) {
	var found []string
	for _, p := range inputs {
		name := path.Base(p)
		if ok, _ := path.Match(strings.ToLower(r.File), strings.ToLower(name)); ok || strings.EqualFold(r.File, name) {
			found = append(found, p)
		}
	}
	switch len(found) {
	case 0:
		return "", fmt.Errorf("input/customer に「%s」に当たるファイルがありません", r.File)
	case 1:
		return found[0], nil
	}
	names := make([]string, len(found))
	for i, p := range found {
		names[i] = path.Base(p)
	}
	return "", fmt.Errorf("「%s」に当たるファイルが %d 個あります（%s）。1 つだけにしてください", r.File, len(found), strings.Join(names, "、"))
}

// merge puts the rows of one dataset into another, cell by cell: a cell that
// is there stays, and a different value for it is told.
func merge(sc *schema.Set, into, from *model.Dataset, clash func(r *model.Row, col string)) {
	for _, t := range sc.Tables {
		tab := from.Tables[t.Name]
		if tab == nil {
			continue
		}
		for _, r := range tab.Rows {
			if into.Adopt(r) {
				continue
			}
			have := into.Tables[t.Name].Find(r.Key)
			have.Origins = append(have.Origins, r.Origins...)
			for col, c := range r.Cells {
				if c.Text() == "" {
					continue
				}
				if old := have.Cells[col]; old != nil && old.Text() != "" {
					if old.Text() != c.Text() {
						clash(r, col)
					}
					continue
				}
				have.Cells[col] = c
			}
		}
	}
}

// Customer reads the customer's lists by the column mappings of the case and
// compares what they say with the parameter book (spec 6.5). With write it
// writes the report, and keeps the marks of the files for the next time.
func (e *Env) Customer(write bool, now time.Time) (*CustomerOutcome, error) {
	rules, err := e.mappings()
	if err != nil {
		return nil, err
	}
	if len(rules) == 0 {
		return nil, fmt.Errorf("mappings/ に列対応ファイル（.yaml）がありません。画面の「お客様の一覧」か customer --draft で作ってください")
	}
	inputs, err := e.Proj.Inputs("customer")
	if err != nil {
		return nil, err
	}
	out := &CustomerOutcome{Data: model.NewDataset("customer")}
	note := func(format string, a ...any) { out.Notes = append(out.Notes, fmt.Sprintf(format, a...)) }
	used := map[string]bool{}
	byNew, byOld := model.NewDataset("customer"), model.NewDataset("customer")
	var problems []listimport.Problem
	for _, r := range rules {
		l := CustomerList{Rule: r}
		func() {
			file, err := listFile(r, inputs)
			if err != nil {
				l.Err = err.Error()
				return
			}
			l.File, used[file] = file, true
			data, err := os.ReadFile(e.Proj.Path(file))
			if err != nil {
				l.Err = fmt.Sprintf("読めません: %v", err)
				return
			}
			sum := sha256.Sum256(data)
			l.SHA256 = hex.EncodeToString(sum[:])
			bk, err := sheet.Read(data)
			if err != nil {
				l.Err = err.Error()
				return
			}
			for _, w := range bk.Warnings {
				note("%s: %s", file, w)
			}
			s, err := listimport.PickSheet(bk, r)
			if err != nil {
				l.Err = err.Error()
				return
			}
			t, err := listimport.ReadTable(s, r)
			if err != nil {
				l.Err = err.Error()
				return
			}
			into := byNew
			if r.Old() {
				into = byOld
			}
			l.Result = listimport.Convert(t, r, e.Schema, file, into)
			problems = append(problems, l.Result.Problems...)
			for _, w := range l.Result.Warnings {
				note("%s: %s", file, w)
			}
			if len(l.Result.Missing) > 0 {
				note("%s: 一覧のシート「%s」に、見出し「%s」がありません。その列は空として読みました（%s を直すか、一覧の見出しを確かめてください）",
					file, l.Result.Sheet, strings.Join(l.Result.Missing, "」「"), r.Path)
			}
		}()
		if l.Err != "" {
			note("%s: %s", r.Path, l.Err)
		}
		out.Lists = append(out.Lists, l)
	}
	for _, p := range inputs {
		if !used[p] {
			out.Unmapped = append(out.Unmapped, p)
		}
	}

	_, bk, err := e.readBook()
	if err != nil {
		return nil, err
	}
	out.Book = bk
	clash := func(r *model.Row, col string) {
		p := listimport.Problem{Table: r.Table, Column: col, Text: fmt.Sprintf("%s の列「%s」: 同じ行に、先に別の値があります。先の値を使います", e.tableTitle(r.Table), e.columnTitle(r.Table, col))}
		if t := e.Schema.Table(r.Table); t != nil {
			p.Device = r.Get(t.DeviceColumn())
		}
		if c := r.Cells[col]; c != nil && len(c.Origins) > 0 {
			o := c.Origins[0]
			p.File, p.Sheet, p.Addr, p.Line = o.File, o.Sheet, o.Addr, o.Line
		}
		problems = append(problems, p)
	}
	merge(e.Schema, out.Data, byNew, clash)
	// The lists that hold the old names: the mapping of the book is put on
	// (spec 6.5.2).
	if len(byOld.Tables) > 0 {
		if bk == nil {
			note("%s がないので、今の（旧の）名前で書かれた一覧（names: old）を新しい名前に直せません。その一覧は比べていません", e.Proj.Config.Book)
		} else {
			a := convert.ApplyList(e.Schema, byOld, bk.Devices, bk.Ifs)
			for _, h := range a.Unmapped {
				note("一覧の機器 %s は、「機器対応」で新ホスト名が決まっていないので、比べていません", h)
			}
			type key struct{ dev, name string }
			seen := map[key]bool{}
			for _, n := range a.Undecided {
				if k := (key{n.Device, n.Value}); !seen[k] {
					seen[k] = true
					note("一覧の機器 %s の IF %s は、「IF 対応」で新 IF 名が決まっていないので、今の名前のまま比べています", n.Device, n.Value)
				}
			}
			for _, n := range a.Dropped {
				note("一覧の機器 %s の IF %s は、「IF 対応」で持っていかない（-）ことになっているので、その行は比べていません", n.Device, n.Value)
			}
			for _, n := range a.Clashes {
				note("一覧の機器 %s: 名前を直すと、先にある行とキーが同じになる行がありました（%s）。その行は比べていません", n.Device, e.tableTitle(n.Table))
			}
			merge(e.Schema, out.Data, a.Data, clash)
		}
	}

	var findings []check.Finding
	if bk == nil {
		note("%s がないので、一覧と比べていません（読めた行と、読めなかった値だけを出しています）", e.Proj.Config.Book)
	} else {
		out.Design, out.Expand, out.BookNotes = e.bookData(bk)
		res := compare.Datasets(e.Schema, out.Data, out.Design)
		targeted := listimport.Targeted(rules)
		// The devices the lists speak of.
		listed := map[string]bool{}
		for _, t := range e.Schema.Tables {
			if tab := out.Data.Tables[t.Name]; tab != nil && t.DeviceColumn() != "" {
				for _, r := range tab.Rows {
					listed[r.Get(t.DeviceColumn())] = true
				}
			}
		}
		for _, x := range res.OnlyA {
			ref, ev := e.rowRef(x.Row)
			f := srcFinding("S-501", x.Device, ref, "お客様の一覧にあって、パラメータブックにない行です", ev...)
			out.OnlyList = append(out.OnlyList, RowFinding{x, f})
			findings = append(findings, f)
		}
		for _, x := range res.Cells {
			if x.A == nil {
				continue // the list says nothing of the column
			}
			ref := model.Ref{Table: x.Table, Key: x.ARow.KeyText(), Column: x.Column}
			var col *schema.Column
			if t := e.Schema.Table(x.Table); t != nil {
				ref, col = check.RefOf(t, x.ARow, x.Column), t.Column(x.Column)
			}
			f := srcFinding("S-503", x.Device, ref, "お客様の一覧とパラメータブックで値が違います",
				model.KV{Name: "一覧", Value: check.Shown(col, x.A)}, model.KV{Name: "ブック", Value: check.Shown(col, x.B)})
			f.At = check.CellAt(x.BRow, x.Column)
			out.Cells = append(out.Cells, CellFinding{x, f})
			findings = append(findings, f)
		}
		for _, x := range res.OnlyB {
			if targeted[x.Table] == nil || (x.Table != "devices" && !listed[x.Device]) {
				continue
			}
			ref, ev := e.rowRef(x.Row)
			f := srcFinding("S-502", x.Device, ref, "パラメータブックにあって、お客様の一覧にない行です", ev...)
			f.At = check.CellAt(x.Row, "")
			out.OnlyBook = append(out.OnlyBook, RowFinding{x, f})
			findings = append(findings, f)
		}
	}
	for _, p := range problems {
		f := srcFinding("S-504", p.Device, model.Ref{Table: p.Table, Column: p.Column}, p.Text,
			model.KV{Name: "ファイル", Value: p.File}, model.KV{Name: "シート", Value: p.Sheet}, model.KV{Name: "セル", Value: p.Addr})
		out.Problems = append(out.Problems, ProblemFinding{p, f})
		findings = append(findings, f)
	}

	// The lists that changed since they were last read (S-505).
	last, err := e.Proj.ListFiles()
	if err != nil {
		return nil, err
	}
	marks := map[string]project.ListFile{}
	for _, f := range last {
		marks[f.File] = f
	}
	seen := map[string]bool{}
	for _, l := range out.Lists {
		if l.SHA256 == "" || seen[l.File] {
			continue
		}
		seen[l.File] = true
		if old, ok := marks[l.File]; ok && old.SHA256 != l.SHA256 {
			f := srcFinding("S-505", "", model.Ref{}, "お客様の一覧のファイルが、前回の読み込み（"+old.At+"）から変わりました", model.KV{Name: "ファイル", Value: l.File})
			out.Changed = append(out.Changed, f)
			findings = append(findings, f)
		}
		marks[l.File] = project.ListFile{File: l.File, SHA256: l.SHA256, At: now.Format("2006-01-02 15:04:05")}
	}

	var design *model.Dataset
	if bk != nil {
		design = bk.Data
	}
	if out.Verdicts, err = e.verdicts(findings, design); err != nil {
		return nil, err
	}
	out.Verdicts.source = true
	if !write {
		return out, nil
	}
	if out.Report, err = e.writeReport(now, "customer", e.customerDoc(out, now)); err != nil {
		return nil, err
	}
	var keep []project.ListFile
	for _, f := range marks {
		keep = append(keep, f)
	}
	sort.Slice(keep, func(i, j int) bool { return keep[i].File < keep[j].File })
	if err := e.Proj.SaveListFiles(keep); err != nil {
		return nil, err
	}
	c := out.Verdicts.Count()
	e.Proj.Log(now, "customer: 一覧 %d 個、一覧にだけある行 %d、ブックにだけある行 %d、値が違うセル %d、読めなかった値 %d（未承認のエラー %d、警告 %d）、レポート %s",
		len(out.Lists), len(out.OnlyList), len(out.OnlyBook), len(out.Cells), len(out.Problems), c.Errors, c.Warnings, out.Report.XLSX)
	return out, nil
}

// Rows counts the rows of the design data the lists gave.
func (o *CustomerOutcome) Rows() int {
	n := 0
	for _, t := range o.Data.Tables {
		n += len(t.Rows)
	}
	return n
}
