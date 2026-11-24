package ops

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"time"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/compare"
	"github.com/kojio145/configchief/internal/convert"
	"github.com/kojio145/configchief/internal/importer"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/render"
	"github.com/kojio145/configchief/internal/report"
)

// Files is a report written in both forms, as paths from the case folder.
type Files struct {
	XLSX, HTML string
}

func (e *Env) lang() string {
	if e.Proj.Config.Language == "en" {
		return "en"
	}
	return "ja"
}

// readBook reads the parameter book. All three are nil if there is none yet.
func (e *Env) readBook() ([]byte, *book.Book, error) {
	if e.draftBook != nil {
		bk, err := book.Read(e.draftBook, e.Proj.Config.Book, e.Schema)
		if err != nil {
			return nil, nil, err
		}
		return e.draftBook, bk, nil
	}
	data, err := os.ReadFile(e.Proj.BookPath())
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("%s を読めません: %v", e.Proj.Config.Book, err)
	}
	bk, err := book.Read(data, e.Proj.Config.Book, e.Schema)
	if err != nil {
		return nil, nil, err
	}
	return data, bk, nil
}

// saveBook writes the book, after a copy of the one that is there went to
// state/backup (spec 15).
func (e *Env) saveBook(now time.Time, data []byte) (backup string, err error) {
	// A book that Excel holds open is told before the copy is taken: a step
	// that is done again when Excel lets it go leaves one copy (spec 14.10.7).
	if err := e.Proj.BookWritable(); err != nil {
		return "", err
	}
	if backup, err = e.Proj.Backup(now); err != nil {
		return "", err
	}
	e.Edit.Drop()
	return backup, e.Proj.WriteBook(data)
}

// writeReport puts a report under out/<time>/reports in both forms.
func (e *Env) writeReport(now time.Time, name string, d *report.Doc) (Files, error) {
	dir := e.outDir(now) + "/reports"
	if err := os.MkdirAll(e.Proj.Path(dir), 0o755); err != nil {
		return Files{}, fmt.Errorf("%s を作れません: %v", dir, err)
	}
	out := Files{XLSX: dir + "/" + name + ".xlsx", HTML: dir + "/" + name + ".html"}
	var buf bytes.Buffer
	if err := d.WriteXLSX(&buf); err != nil {
		return Files{}, err
	}
	if err := os.WriteFile(e.Proj.Path(out.XLSX), buf.Bytes(), 0o644); err != nil {
		return Files{}, fmt.Errorf("%s を書けません: %v", out.XLSX, err)
	}
	if err := os.WriteFile(e.Proj.Path(out.HTML), []byte(d.HTML()), 0o644); err != nil {
		return Files{}, fmt.Errorf("%s を書けません: %v", out.HTML, err)
	}
	return out, nil
}

// importSide imports a folder of input/ and tells when it holds no file.
func (e *Env) importSide(source, choose string) (*Source, error) {
	src, err := e.Import(source, choose)
	if err != nil {
		return nil, err
	}
	if len(src.Files) == 0 {
		return nil, fmt.Errorf("input/%s にファイルがありません", source)
	}
	return src, nil
}

// newSide imports input/new if it holds a file (nil if not): the proposals
// for the mapping look at it.
func (e *Env) newSide() (*Source, error) {
	paths, err := e.Proj.Inputs("new")
	if err != nil || len(paths) == 0 {
		return nil, err
	}
	return e.Import("new", "")
}

func data(s *Source) *model.Dataset {
	if s == nil {
		return nil
	}
	return s.Data
}

// ImportOutcome is what RunImport did.
type ImportOutcome struct {
	Source *Source
	// New is the new side, read for the proposals of the mapping (only when
	// the old side is imported and input/new holds a file).
	New *Source
	// Book tells what went into the book, and Backup where the copy of the
	// book before is ("" if there was no book). Only for the old side.
	Book     *book.Summary
	Backup   string
	Problems []book.Problem // what the book says after the step
	// Applied is the old data with the mapping of the book put on: it tells
	// what of the mapping is not decided yet. Only for the old side.
	Applied *convert.Applied
	// Mig is the findings of the mapping (M-406 to M-408) with the approvals
	// put on. Only for the old side.
	Mig *Verdicts
	// Judging is the judgments of the case put on the unhandled lines.
	Judging *Judging
	Report  Files
}

// Findings counts what a person has to look at: files that were not read,
// unhandled lines that are not judged or on hold (M-401), devices judged "by
// hand" with no extra configuration (M-402), warnings, and the findings of
// the mapping that are not approved (M-406 to M-408).
func (o *ImportOutcome) Findings() int {
	n := o.Source.Count().Unknown
	if o.Judging != nil {
		n = o.Judging.Findings()
	}
	for _, f := range o.Source.Files {
		if f.Err != "" {
			n++
		} else if f.Result != nil {
			n += len(f.Result.Warnings)
		}
	}
	return n + o.Mig.Open()
}

// RunImport imports input/<source> and writes the report of it (spec 8, 13.4).
// For the old side it also makes the parameter book if there is none and adds
// the proposals to 新旧対照表（機器） and 新旧対照表（ポート）; rows that are there stay (spec 10.2).
func (e *Env) RunImport(source, choose string, now time.Time) (*ImportOutcome, error) {
	if source == "logs" {
		return nil, fmt.Errorf("logs の取り込みは、まだありません（old か new）")
	}
	src, err := e.importSide(source, choose)
	if err != nil {
		return nil, err
	}
	out := &ImportOutcome{Source: src}
	view := hostView{known: true}
	if source == "old" {
		if out.New, err = e.newSide(); err != nil {
			return nil, err
		}
		old, bk, err := e.readBook()
		if err != nil {
			return nil, err
		}
		var have []book.DeviceMap
		if bk != nil {
			have = bk.Devices
		}
		dev, ifs := convert.ProposeWith(src.Data, data(out.New), have, e.shapes)
		convert.Refs(e.Schema, src.Data, append(append([]book.DeviceMap{}, have...), dev...), ifs)
		file := e.Proj.Config.Book
		next, sum, err := book.Update(old, file, e.Schema, e.lang(), book.Change{Devices: dev, Ifs: ifs, IfFill: ifs})
		if err != nil {
			return nil, err
		}
		out.Book = sum
		if sum.Changed() {
			if out.Backup, err = e.saveBook(now, next); err != nil {
				return nil, err
			}
		}
		after, err := book.Read(next, file, e.Schema)
		if err != nil {
			return nil, err
		}
		out.Problems = after.Problems
		out.Applied = convert.Apply(e.Schema, src.Data, after.Devices, after.Ifs)
		if out.Mig, err = e.verdicts(mappingFindings(out.Applied), after.Data); err != nil {
			return nil, err
		}
		view.hosts, view.book = out.Applied.Hosts, after.Data
	} else {
		view = e.sideView(src)
	}
	if out.Judging, err = e.judging(src, view); err != nil {
		return nil, err
	}
	if out.Report, err = e.writeReport(now, "import_"+source, e.importDoc(out, now)); err != nil {
		return nil, err
	}
	if sig, err := e.inputSig(source); err == nil {
		// What was taken, for ImportStale (spec 14.19.3). A mark that cannot
		// be written only makes the next visit import again.
		if os.MkdirAll(e.Proj.Path("state"), 0o755) == nil {
			_ = os.WriteFile(e.Proj.Path(importSigFile(source)), []byte(sig), 0o644)
		}
	}
	c := src.Count()
	e.Proj.Log(now, "import %s: %d ファイル、%d 行（取り込み済み %d、無視 %d、読めなかった %d、うち未判定・保留 %d）、レポート %s",
		source, len(src.Files), c.Total, c.Taken, c.Ignored+c.Separators, c.Unknown, out.Judging.Open+out.Judging.Hold, out.Report.XLSX)
	return out, nil
}

// fillIfs gives the rows of 新旧対照表（ポート） with the proposals put into the rows
// whose new interface name is empty, as book.Update does with IfFill.
func fillIfs(have, prop []book.IfMap) []book.IfMap {
	by := map[string]book.IfMap{}
	for _, p := range prop {
		if p.NewIf != "" {
			by[p.NewHost+"\x00"+p.OldIf] = p
		}
	}
	out := make([]book.IfMap, len(have))
	for i, m := range have {
		if p, ok := by[m.NewHost+"\x00"+m.OldIf]; ok && m.NewIf == "" {
			m.NewIf, m.Basis = p.NewIf, p.Basis
			delete(by, m.NewHost+"\x00"+m.OldIf)
		}
		out[i] = m
	}
	return out
}

// IfProposal is what ProposeIfs did.
type IfProposal struct {
	Added    int            // rows added to 新旧対照表（ポート）
	Filled   int            // rows of 新旧対照表（ポート） whose empty new name was filled
	Backup   string         // the copy of the book taken first ("" = nothing was written)
	Problems []book.Problem // what Read said about the book
}

// ProposeIfs makes the proposals of 新旧対照表（ポート） again from input/old, without
// the report of an import (spec 10.2, 14.10.2): the rows the book lacks are
// added, and the empty new names are filled. Nothing else changes.
func (e *Env) ProposeIfs(now time.Time) (*IfProposal, error) {
	old, bk, err := e.readBook()
	if err != nil {
		return nil, err
	}
	if bk == nil {
		return nil, fmt.Errorf("%s がありません。先に旧コンフィグを取り込んでください", e.Proj.Config.Book)
	}
	src, err := e.importSide("old", "")
	if err != nil {
		return nil, err
	}
	nw, err := e.newSide()
	if err != nil {
		return nil, err
	}
	_, ifs := convert.ProposeWith(src.Data, data(nw), bk.Devices, e.shapes)
	convert.Refs(e.Schema, src.Data, bk.Devices, ifs)
	next, sum, err := book.Update(old, e.Proj.Config.Book, e.Schema, e.lang(), book.Change{Ifs: ifs, IfFill: ifs})
	if err != nil {
		return nil, err
	}
	out := &IfProposal{Added: sum.IfRows, Filled: sum.IfFilled, Problems: sum.Problems}
	if !sum.Changed() {
		return out, nil
	}
	if err := e.Proj.BookWritable(); err != nil {
		return nil, err
	}
	if out.Backup, err = e.saveBook(now, next); err != nil {
		return nil, err
	}
	e.Proj.Log(now, "新旧対照表（ポート）の案: %d 行を足し、%d 行の新 IF 名を埋めました（控え %s）", out.Added, out.Filled, out.Backup)
	return out, nil
}

// Said tells what ProposeIfs did, in a sentence.
func (p *IfProposal) Said() string {
	if p.Added == 0 && p.Filled == 0 {
		return "新旧対照表（ポート）に足す行・埋める新 IF 名はありませんでした（新ホスト名の書かれた機器の IF は、すべて行があり、案の出せる空の新 IF 名もありません）"
	}
	return fmt.Sprintf("新旧対照表（ポート）に %d 行を足し、%d 行の新 IF 名を案で埋めました（控え %s）", p.Added, p.Filled, p.Backup)
}

// BookOutcome is what MakeBook did.
type BookOutcome struct {
	From     string
	Source   *Source
	Applied  *convert.Applied // only from the old side
	Mig      *Verdicts        // the findings of the mapping, only from the old side
	Summary  *book.Summary
	Backup   string
	Problems []book.Problem // what the book says after the step
}

// Findings counts what a person has to look at.
func (o *BookOutcome) Findings() int {
	n := len(o.Problems)
	for _, f := range o.Source.Files {
		if f.Err != "" {
			n++
		}
	}
	return n + o.Mig.Open()
}

// MakeBook puts design data into the sheets of the tables (spec 10.3, 15):
// from "old" the old data with the mapping put on, from "new" the new
// configurations as they are. Rows that are in the book stay; replace empties
// the sheets of the tables first.
func (e *Env) MakeBook(from string, replace bool, now time.Time) (*BookOutcome, error) {
	if from != "old" && from != "new" {
		return nil, fmt.Errorf("--from に「%s」は使えません（old か new）", from)
	}
	src, err := e.importSide(from, "")
	if err != nil {
		return nil, err
	}
	out := &BookOutcome{From: from, Source: src}
	old, bk, err := e.readBook()
	if err != nil {
		return nil, err
	}
	ch := book.Change{Data: src.Data, Replace: replace}
	if from == "old" {
		nw, err := e.newSide()
		if err != nil {
			return nil, err
		}
		var devs []book.DeviceMap
		var ifs []book.IfMap
		if bk != nil {
			devs, ifs = bk.Devices, bk.Ifs
		}
		// The rows the book lacks: the devices that came since, and the
		// interfaces of the devices whose new host name a person wrote.
		ch.Devices, ch.Ifs = convert.ProposeWith(src.Data, data(nw), devs, e.shapes)
		ch.IfFill = ch.Ifs
		devs = append(append([]book.DeviceMap{}, devs...), ch.Devices...)
		convert.Refs(e.Schema, src.Data, devs, ch.Ifs)
		ifs = append(fillIfs(ifs, ch.Ifs), ch.Ifs...)
		out.Applied = convert.Apply(e.Schema, src.Data, devs, ifs)
		ch.Data = out.Applied.Data
	}
	file := e.Proj.Config.Book
	next, sum, err := book.Update(old, file, e.Schema, e.lang(), ch)
	if err != nil {
		return nil, err
	}
	out.Summary = sum
	if sum.Changed() {
		if out.Backup, err = e.saveBook(now, next); err != nil {
			return nil, err
		}
	}
	after, err := book.Read(next, file, e.Schema)
	if err != nil {
		return nil, err
	}
	out.Problems = after.Problems
	if out.Applied != nil {
		if out.Mig, err = e.verdicts(mappingFindings(out.Applied), after.Data); err != nil {
			return nil, err
		}
	}
	backup := "控えなし（新しく作りました）"
	if out.Backup != "" {
		backup = "控え " + out.Backup
	}
	e.Proj.Log(now, "book --from %s: %d 行を足しました（%s）", from, sum.Rows(), backup)
	return out, nil
}

// Side is one of the two sources of a comparison.
type Side struct {
	Name    string           // "old" | "new" | "book"
	Data    *model.Dataset   // what is compared
	Source  *Source          // the import, for old and new
	Applied *convert.Applied // the mapping put on, for old
	Book    *book.Book       // for book
	// Expand is what giving the rows of the book to the devices had to tell,
	// and Notes what else kept the book from being taken as an imported
	// configuration is (a device without a pack gets no defaults).
	Expand []book.Problem
	Notes  []string
}

// bookData gives the design data of the book as a comparison takes it (spec
// 12.3): the rows written for "*" and "@group" and the common values given
// to the devices, and the empty columns filled with the defaults of the pack
// of each device. That is what an imported configuration holds.
func (e *Env) bookData(bk *book.Book) (*model.Dataset, []book.Problem, []string) {
	return e.bookDataAs(bk, true)
}

// bookDataAs is bookData; adapt false leaves out the rewriting into the way
// the vendor of the device writes a value (appendix B 5.7). The book is
// compared with the old side that way: the old values are those of the
// design, not of the new vendor (a VLAN is not its group of the new switch).
func (e *Env) bookDataAs(bk *book.Book, adapt bool) (*model.Dataset, []book.Problem, []string) {
	ds, problems := bk.Expand(e.Schema)
	out := model.NewDataset(ds.Source)
	var notes []string
	hosts := book.Hosts(ds)
	known := map[string]bool{}
	of := render.ByDevice(e.Schema, ds)
	for _, h := range hosts {
		known[h] = true
		dd := of(h)
		vendor := ""
		if t := dd.Tables["devices"]; t != nil && len(t.Rows) > 0 {
			vendor = t.Rows[0].Get("vendor")
		}
		if p := e.Packs[vendor]; p != nil {
			// The same rewriting as the generation does (appendix B 5.7), so
			// that the book and a configuration made from it compare equal.
			if adapt {
				importer.Adapt(p, e.Schema, dd, h)
			}
			for _, w := range importer.FillDefaults(p, e.Schema, dd, h, false) {
				notes = append(notes, fmt.Sprintf("機器 %s: パック %s の既定値: %s", h, vendor, w))
			}
		} else {
			notes = append(notes, fmt.Sprintf("機器 %s のメーカー「%s」に当たるパックがないので、既定値を補わずに比べています", h, vendor))
		}
		for _, t := range e.Schema.Tables {
			if tab := dd.Tables[t.Name]; tab != nil {
				for _, r := range tab.Rows {
					out.Adopt(r)
				}
			}
		}
	}
	// The rows of no device of the sheet (a table that is not per device, a
	// host name the sheet of the devices lacks) go in as they are.
	for _, t := range e.Schema.Tables {
		tab := ds.Tables[t.Name]
		if tab == nil {
			continue
		}
		devCol := t.DeviceColumn()
		for _, r := range tab.Rows {
			if devCol == "" || !known[r.Get(devCol)] {
				out.Adopt(r)
			}
		}
	}
	return out, problems, notes
}

// CompareOutcome is what Compare did.
type CompareOutcome struct {
	A, B   *Side
	Result *compare.Result
	// Mig is the findings of a comparison with the old side (M-403 to M-408)
	// with the approvals put on; nil for the new side against the book.
	Mig    *Verdicts
	Report Files
}

// Findings counts what a person has to look at: the differences (with the
// old side, the findings that are not approved), and what kept the
// comparison from being whole (files that were not read, unhandled lines,
// what the book says).
func (o *CompareOutcome) Findings() int {
	r := o.Result
	n := len(r.OnlyA) + len(r.OnlyB) + len(r.Cells)
	if o.Mig != nil {
		n = o.Mig.Open()
	}
	for _, s := range []*Side{o.A, o.B} {
		if s.Source != nil {
			n += s.Source.Count().Unknown
			for _, f := range s.Source.Files {
				if f.Err != "" {
					n++
				}
			}
		}
		if s.Book != nil {
			n += len(s.Book.Problems) + len(s.Expand)
		}
	}
	return n
}

// Compare compares two sources and writes the result (spec 12.3, 13.5). The
// old side is compared after the mapping of the book is put on.
func (e *Env) Compare(a, b string, now time.Time) (*CompareOutcome, error) {
	if a == b {
		return nil, fmt.Errorf("--a と --b が同じです（%s）", a)
	}
	var bk *book.Book
	var read bool
	theBook := func() (*book.Book, error) {
		if !read {
			var err error
			if _, bk, err = e.readBook(); err != nil {
				return nil, err
			}
			read = true
		}
		return bk, nil
	}
	side := func(name string) (*Side, error) {
		s := &Side{Name: name}
		var err error
		switch name {
		case "old":
			if s.Source, err = e.importSide("old", ""); err != nil {
				return nil, err
			}
			bk, err := theBook()
			if err != nil {
				return nil, err
			}
			if bk == nil {
				return nil, fmt.Errorf("%s がありません。旧の機器名と IF 名を新の名前に直すために要ります。先に import --source old を実行してください", e.Proj.Config.Book)
			}
			s.Applied = convert.Apply(e.Schema, s.Source.Data, bk.Devices, bk.Ifs)
			s.Data = s.Applied.Data
		case "new":
			if s.Source, err = e.importSide("new", ""); err != nil {
				return nil, err
			}
			s.Data = s.Source.Data
		case "book":
			if s.Book, err = theBook(); err != nil {
				return nil, err
			}
			if s.Book == nil {
				return nil, fmt.Errorf("%s がありません", e.Proj.Config.Book)
			}
			if a == "old" || b == "old" {
				s.Data, s.Expand, s.Notes = e.bookDataAs(s.Book, false)
			} else {
				s.Data, s.Expand, s.Notes = e.bookData(s.Book)
			}
		default:
			return nil, fmt.Errorf("データ源「%s」は比べられません（old、new、book）", name)
		}
		return s, nil
	}
	out := &CompareOutcome{}
	var err error
	if out.A, err = side(a); err != nil {
		return nil, err
	}
	if out.B, err = side(b); err != nil {
		return nil, err
	}
	out.Result = compare.Datasets(e.Schema, out.A.Data, out.B.Data)
	if a == "old" || b == "old" {
		var findings []check.Finding
		for _, s := range []*Side{out.A, out.B} {
			if s.Applied != nil {
				findings = mappingFindings(s.Applied)
			}
		}
		findings = append(findings, e.diffFindings(out.Result, a == "old")...)
		if out.Mig, err = e.verdicts(findings, bk.Data); err != nil {
			return nil, err
		}
	}
	if out.Report, err = e.writeReport(now, "compare_"+a+"_"+b, e.compareDoc(out, now)); err != nil {
		return nil, err
	}
	r := out.Result
	e.Proj.Log(now, "compare %s %s: %s にだけある行 %d、%s にだけある行 %d、値が違うセル %d、レポート %s",
		a, b, a, len(r.OnlyA), b, len(r.OnlyB), len(r.Cells), out.Report.XLSX)
	return out, nil
}

// shapes finds the model a row of 新旧対照表（機器） names in the pack of its vendor
// (appendix B 6.3).
func (e *Env) shapes(vendor, name string) *convert.Shape {
	m := e.Packs[vendor].Model(name)
	if m == nil {
		return nil
	}
	sh := &convert.Shape{ID: m.ID, Logical: m.Logical}
	for _, p := range m.Ports {
		sh.Ports = append(sh.Ports, convert.ShapePort{Name: p.Name, Role: p.Role, Sub: p.Sub, VlanIf: p.VlanIf})
	}
	return sh
}
