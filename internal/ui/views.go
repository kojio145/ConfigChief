package ui

import (
	"slices"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/kojio145/configchief/internal/importer"
	"github.com/kojio145/configchief/internal/ops"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/report"
	"github.com/kojio145/configchief/internal/textdiff"
)

// Pair is a label with its value.
type Pair struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

func pairs(list []report.Pair) []Pair {
	out := make([]Pair, 0, len(list))
	for _, p := range list {
		out = append(out, Pair{p.Label, p.Value})
	}
	return out
}

// Table is a list of a screen: the same as a sheet of a report.
type Table struct {
	Name   string     `json:"name"`
	Lead   string     `json:"lead"`
	Header []string   `json:"header"`
	Rows   [][]string `json:"rows"`
	Marks  []string   `json:"marks"` // one per row: "" | "ignored" | "unknown" | "warn"
	Code   []int      `json:"code"`  // the columns that hold configuration text
}

func table(s report.Sheet) Table {
	t := Table{Name: s.Name, Lead: s.Lead, Header: strs(s.Header), Rows: s.Rows, Marks: make([]string, len(s.Rows)), Code: s.Code}
	copy(t.Marks, s.Marks)
	if t.Rows == nil {
		t.Rows = [][]string{}
	}
	if t.Code == nil {
		t.Code = []int{}
	}
	return t
}

// Files is a report written in both forms, as paths from the case folder.
type Files struct {
	XLSX string `json:"xlsx"`
	HTML string `json:"html"`
}

// LogsView is the page of the logs (spec 14.5): the outcome of the check of
// the work ("load") or of what is on the devices ("verify").
type LogsView struct {
	What   string        `json:"what"`
	Title  string        `json:"title"`
	Counts []Pair        `json:"counts"`
	Notes  []string      `json:"notes"`
	Tables []Table       `json:"tables"`
	Report Files         `json:"report"`
	Items  []FindingView `json:"items"`
	By     string        `json:"by"`
}

// ExportView is what the home screen shows after the delivery sheets or the
// table for the other side were written (spec 14.4).
type ExportView struct {
	What     string   `json:"what"`     // "sheet" | "peer"
	Summary  string   `json:"summary"`  // one sentence on what was made
	Files    []string `json:"files"`    // what can be opened, with the folder
	Dir      string   `json:"dir"`      // the folder of the files ("" = nothing was written)
	Problems []string `json:"problems"` // what is wrong in the templates
	Notes    []string `json:"notes"`
}

// Candidate is a pack with the points the text of a file gave it.
type Candidate struct {
	Pack  string `json:"pack"`
	Score int    `json:"score"`
}

// FileView is one file of an import and what became of it.
type FileView struct {
	Path     string `json:"path"`
	Imported bool   `json:"imported"`
	Err      string `json:"err"` // why the file was not imported
	Device   string `json:"device"`
	Pack     string `json:"pack"`    // the ID of the pack that read it
	Version  string `json:"version"` // the version of that pack
	Chosen   bool   `json:"chosen"`  // a person chose the pack
	// NeedsPack: the tool cannot tell the vendor; a person chooses.
	NeedsPack bool `json:"needsPack"`
	// Same: the files of the folder with the same host name, this one
	// among them (empty if the host name is its own). NeedsFile: none of
	// them is chosen yet. Left: a person chose another one; this file is
	// left out.
	Same       []string    `json:"same"`
	NeedsFile  bool        `json:"needsFile"`
	Left       string      `json:"left"`
	Candidates []Candidate `json:"candidates"`
	Total      int         `json:"total"`
	Taken      int         `json:"taken"`
	Ignored    int         `json:"ignored"`
	Blank      int         `json:"blank"` // comments, blank lines, lines outside the configuration
	Unknown    int         `json:"unknown"`
	Open       int         `json:"open"` // the unhandled lines not judged or on hold
	Ratio      string      `json:"ratio"`
	Warnings   []string    `json:"warnings"`
	Notes      []string    `json:"notes"`
	NewHost    string      `json:"newHost"` // old side: the new host name of the book
	Mark       string      `json:"mark"`    // "" | "warn" | "unknown"
}

// ImportView is the outcome of an import (spec 14.1 取り込み).
type ImportView struct {
	Source   string     `json:"source"`
	Title    string     `json:"title"`
	Counts   []Pair     `json:"counts"`
	Notes    []string   `json:"notes"`
	Messages []string   `json:"messages"` // what became of the parameter book
	Files    []FileView `json:"files"`
	Tables   []Table    `json:"tables"` // unhandled lines, open parts of the mapping, notes
	Packs    []PackInfo `json:"packs"`
	Report   Files      `json:"report"`
	Findings int        `json:"findings"`
}

func checkSource(source string) error {
	if source != "old" && source != "new" {
		return fmt.Errorf("データ源「%s」は取り込めません（old か new）", source)
	}
	return nil
}

// Import imports input/<source>, as the command "import" does: the report is
// written, and for the old side the parameter book gets the proposals of the
// mapping.
func (s *Session) Import(source string) (*ImportView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := checkSource(source); err != nil {
		return nil, err
	}
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	return s.runImport(e, source)
}

func (s *Session) runImport(e *ops.Env, source string) (*ImportView, error) {
	now := s.now()
	o, err := e.RunImport(source, "", now)
	if err != nil {
		return nil, err
	}
	s.imports[source] = o
	return importView(e, o, e.ImportDoc(o, now)), nil
}

// ChoosePack records the pack a person chose for a file whose vendor the
// tool cannot tell, and imports the source again.
func (s *Session) ChoosePack(source, path, pack string) (*ImportView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := checkSource(source); err != nil {
		return nil, err
	}
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	if err := e.ChoosePack(path, pack, s.now()); err != nil {
		return nil, err
	}
	return s.runImport(e, source)
}

// ChooseFile records which file is taken for a host name that two or more
// files of the source have, and imports the source again.
func (s *Session) ChooseFile(source, path string) (*ImportView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := checkSource(source); err != nil {
		return nil, err
	}
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	if err := e.ChooseFile(source, path, s.now()); err != nil {
		return nil, err
	}
	return s.runImport(e, source)
}

func importView(e *ops.Env, o *ops.ImportOutcome, d *report.Doc) *ImportView {
	v := &ImportView{
		Source:   o.Source.Name,
		Title:    d.Title,
		Counts:   pairs(d.Counts),
		Notes:    strs(d.Notes),
		Messages: []string{},
		Files:    []FileView{},
		Tables:   []Table{},
		Packs:    []PackInfo{},
		Report:   Files{o.Report.XLSX, o.Report.HTML},
		Findings: o.Findings(),
	}
	for _, id := range e.PackIDs {
		v.Packs = append(v.Packs, packInfo(e, id))
	}
	// The lists of the report but the first and the last one: the files are
	// shown from Files, and the lines of a file are asked for one file at a
	// time (Lines).
	for i, sh := range d.Sheets {
		if i > 0 && i < len(d.Sheets)-1 {
			v.Tables = append(v.Tables, table(sh))
		}
	}
	for _, f := range o.Source.Files {
		fv := FileView{Path: f.Path, Err: f.Err, Pack: f.Pack, Chosen: f.Chosen, Candidates: []Candidate{}, Warnings: []string{}, Notes: []string{}, Mark: report.MarkWarn}
		if p := e.Packs[f.Pack]; p != nil {
			fv.Version = p.Version
		}
		for _, c := range f.Candidates {
			fv.Candidates = append(fv.Candidates, Candidate{c.Pack, c.Score})
		}
		fv.NeedsPack = f.Pack == "" && len(f.Candidates) > 0
		fv.Same, fv.Left, fv.Device = strs(f.Same), f.Left, f.Device
		fv.NeedsFile = len(f.Same) > 1 && f.Err != ""
		if f.Left != "" {
			fv.Mark = report.MarkIgnored
		}
		if r := f.Result; r != nil {
			n := r.Count()
			fv.Imported = true
			fv.Device = r.Device
			fv.Total, fv.Taken, fv.Ignored, fv.Blank, fv.Unknown = n.Total, n.Taken, n.Ignored, n.Separators, n.Unknown
			fv.Ratio = fmt.Sprintf("%.1f%%", n.Covered()*100)
			fv.Warnings, fv.Notes = strs(r.Warnings), strs(r.Notes)
			fv.Open = n.Unknown
			if o.Judging != nil {
				fv.Open = o.Judging.OpenLines(f.Path)
			}
			switch {
			case fv.Open > 0:
				fv.Mark = report.MarkUnknown
			case n.Unknown == 0 && len(r.Warnings) == 0:
				fv.Mark = ""
			}
			if o.Applied != nil {
				fv.NewHost = o.Applied.Hosts[r.Device]
			}
		}
		v.Files = append(v.Files, fv)
	}
	if b := o.Book; b != nil {
		name := e.Proj.Config.Book
		switch {
		case b.Created:
			v.Messages = append(v.Messages, fmt.Sprintf("パラメータブック %s を作りました（新旧対照表（機器） %d 行、新旧対照表（ポート） %d 行）", name, b.DeviceRows, b.IfRows))
		case b.Changed():
			v.Messages = append(v.Messages, fmt.Sprintf("パラメータブック %s に足しました（新旧対照表（機器） %d 行、新旧対照表（ポート） %d 行）。控え: %s", name, b.DeviceRows, b.IfRows, o.Backup))
		default:
			v.Messages = append(v.Messages, fmt.Sprintf("パラメータブック %s は変えていません（足す行がありません）", name))
		}
	}
	return v
}

// Made is a value a line gave to the design data.
type Made struct {
	Where string `json:"where"`
	Value string `json:"value"`
}

// LineView is a line of an imported file.
type LineView struct {
	No     int    `json:"no"`
	Text   string `json:"text"`  // the secrets masked
	State  string `json:"state"` // "taken" | "ignored" | "unknown"
	Word   string `json:"word"`  // the state in words
	Blank  bool   `json:"blank"`
	Rule   string `json:"rule"`
	Reason string `json:"reason"`
	Made   []Made `json:"made"`
	// An unhandled line: the judgment in words, whether it is still to be
	// decided, and the ID of its group.
	Judged string `json:"judged"`
	Open   bool   `json:"open"`
	Group  string `json:"group"`
}

// LinesView is the lines of one file (spec 14.1: 色分けして表示).
type LinesView struct {
	Path   string     `json:"path"`
	Device string     `json:"device"`
	Lines  []LineView `json:"lines"`
}

// Lines gives the lines of a file of the last import of a source.
func (s *Session) Lines(source, path string) (*LinesView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	o := s.imports[source]
	if s.env == nil || o == nil {
		return nil, errors.New("先に取り込んでください")
	}
	for _, f := range o.Source.Files {
		if f.Path != path {
			continue
		}
		if f.Result == nil {
			return nil, fmt.Errorf("%s は取り込めていません: %s", path, f.Err)
		}
		v := &LinesView{Path: path, Device: f.Result.Device, Lines: []LineView{}}
		for _, l := range s.env.Lines(f) {
			lv := LineView{No: l.No, Text: l.Text, State: l.State, Word: l.Word, Blank: l.Blank, Rule: l.Rule, Reason: l.Reason, Made: []Made{}}
			if l.Blank && l.State == importer.Ignored {
				lv.Word = "無視（コメント・空行など）"
			}
			for _, m := range l.Made {
				lv.Made = append(lv.Made, Made{m.Where, m.Value})
			}
			if l.State == importer.Unknown {
				lv.Open = true
				if j, ok := o.Judging.Line(path, l.No); ok {
					lv.Judged, lv.Open, lv.Group = j.Word(), j.Open(), j.Group()
				}
			}
			v.Lines = append(v.Lines, lv)
		}
		return v, nil
	}
	return nil, fmt.Errorf("%s は、取り込んだファイルにありません。取り込み直してください", path)
}

// MappingView is the mapping of the book (spec 14.1 対応). The page only
// shows it: the mapping is edited in Excel.
type MappingView struct {
	Book    string `json:"book"`
	HasBook bool   `json:"hasBook"`
	Counts  []Pair `json:"counts"`
	Devices Table  `json:"devices"`
	Ifs     Table  `json:"ifs"`
	Open    Table  `json:"open"` // what is not decided yet
	// Findings is the findings of the mapping (M-406 to M-408): a row of Open
	// is tied to one by the fingerprint in its last column.
	Findings []FindingView `json:"findings"`
	By       string        `json:"by"`       // the name the last approval was given with
	Problems []string      `json:"problems"` // what the book says
	Notes    []string      `json:"notes"`    // files of the old side that were not read
	// Hosts are the new host names of 新旧対照表（機器）, for making the proposals of
	// 新旧対照表（ポート） again (spec 14.11.4).
	Hosts []string `json:"hosts"`
}

// Mapping reads the old side and the book and tells what of the mapping is
// open. It writes nothing.
func (s *Session) Mapping() (*MappingView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	m, err := e.Mapping()
	if err != nil {
		return nil, err
	}
	v := &MappingView{Book: e.Proj.Config.Book, Counts: []Pair{}, Problems: []string{}, Notes: []string{}, Findings: migFindings(e, m.Mig), By: s.name, Hosts: []string{}}
	s.mig = m.Mig
	v.Devices = Table{Name: "新旧対照表（機器）", Header: []string{"ブックの行", "旧ホスト名", "旧メーカー", "新ホスト名", "新メーカー", "新機種", "備考"}}
	v.Ifs = Table{Name: "新旧対照表（ポート）", Header: []string{"ブックの行", "新ホスト名", "旧ホスト名", "旧 IF 名", "旧 IF の設定", "新 IF 名", "案の根拠"}}
	for _, f := range m.Source.Files {
		if f.Err != "" {
			v.Notes = append(v.Notes, f.Path+" は取り込めません: "+f.Err)
		}
	}
	if m.Book == nil {
		v.Devices, v.Ifs, v.Open = table(v.Devices.sheet()), table(v.Ifs.sheet()), table(report.Sheet{Name: "新旧対照表で決まっていないところ"})
		return v, nil
	}
	v.HasBook = true
	// The rows that are open come first, then in the order of the book.
	type ranked struct {
		rank int
		row  []string
		mark string
	}
	put := func(t *Table, list []ranked) {
		sort.SliceStable(list, func(i, j int) bool { return list[i].rank < list[j].rank })
		s := t.sheet()
		for _, r := range list {
			s.Rows = append(s.Rows, r.row)
			s.Marks = append(s.Marks, r.mark)
		}
		*t = table(s)
	}
	var devs, ifs []ranked
	openDevs, outDevs, openIfs, outIfs := 0, 0, 0, 0
	for _, d := range m.Book.Devices {
		if d.NewHost != "" && !slices.Contains(v.Hosts, d.NewHost) {
			v.Hosts = append(v.Hosts, d.NewHost)
		}
		r := ranked{rank: 2, row: []string{fmt.Sprint(d.Row), d.OldHost, d.OldVendor, d.NewHost, d.NewVendor, d.NewModel, d.Note}}
		switch {
		case d.NewHost == "" && d.Note == "" && !d.Out:
			r.rank, r.mark = 0, report.MarkUnknown
			openDevs++
		case d.NewHost == "":
			r.rank, r.mark = 1, report.MarkIgnored
			outDevs++
		}
		devs = append(devs, r)
	}
	for _, f := range m.Book.Ifs {
		r := ranked{rank: 2, row: []string{fmt.Sprint(f.Row), f.NewHost, f.OldHost, f.OldIf, f.Now, f.NewIf, f.Basis}}
		switch f.NewIf {
		case "":
			r.rank, r.mark = 0, report.MarkUnknown
			openIfs++
		case "-":
			r.rank, r.mark = 1, report.MarkIgnored
			outIfs++
		}
		ifs = append(ifs, r)
	}
	put(&v.Devices, devs)
	put(&v.Ifs, ifs)
	a := m.Applied
	v.Open = table(e.MappingSheet(a, m.Mig, false))
	v.Counts = []Pair{
		{"新旧対照表（機器）の行", fmt.Sprint(len(m.Book.Devices))},
		{"うち新ホスト名がまだ空", fmt.Sprint(openDevs)},
		{"うち持っていかない（新ホスト名が - か、空で備考あり）", fmt.Sprint(outDevs)},
		{"新旧対照表（ポート）の行", fmt.Sprint(len(m.Book.Ifs))},
		{"うち新 IF 名が空", fmt.Sprint(openIfs)},
		{"うち持っていかない IF（-）", fmt.Sprint(outIfs)},
		{"旧コンフィグで使っていて、新 IF 名が決まっていない IF", fmt.Sprint(len(a.Undecided))},
		{"複数の旧 IF が当たっている新 IF", fmt.Sprint(len(a.Shared))},
	}
	for _, p := range m.Book.Problems {
		v.Problems = append(v.Problems, p.String())
	}
	return v, nil
}

func (t Table) sheet() report.Sheet {
	return report.Sheet{Name: t.Name, Lead: t.Lead, Header: t.Header, Rows: t.Rows, Marks: t.Marks, Code: t.Code}
}

// BookView is what writing the parameter book did (spec 14.1).
type BookView struct {
	From     string   `json:"from"`
	Messages []string `json:"messages"`
	Open     Table    `json:"open"` // what the mapping left open or out (from the old side)
	Problems []string `json:"problems"`
	Findings int      `json:"findings"`
}

// MakeBook puts design data into the parameter book, as the command "book"
// does: from "old" with the mapping put on, from "new" as it is. replace
// empties the sheets of the tables first.
func (s *Session) MakeBook(from string, replace bool) (*BookView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	o, err := e.MakeBook(from, replace, s.now())
	if err != nil {
		return nil, err
	}
	v := &BookView{From: from, Messages: []string{}, Problems: []string{}, Open: table(report.Sheet{Name: "新旧対照表のメモ"}), Findings: o.Findings()}
	for _, f := range o.Source.Files {
		if f.Err != "" {
			v.Messages = append(v.Messages, f.Path+" は取り込めません: "+f.Err)
		}
	}
	sum, name, word := o.Summary, e.Proj.Config.Book, sourceWord(from)
	switch {
	case sum.Created:
		v.Messages = append(v.Messages, fmt.Sprintf("パラメータブック %s を作りました（%s から %d 行）", name, word, sum.Rows()))
	case sum.Changed():
		how := "足しました"
		if sum.Replaced {
			how = "設計データのシートを空にしてから書きました"
		}
		v.Messages = append(v.Messages, fmt.Sprintf("パラメータブック %s に%s（%s から %d 行）。控え: %s", name, how, word, sum.Rows(), o.Backup))
	default:
		v.Messages = append(v.Messages, fmt.Sprintf("パラメータブック %s は変えていません（足す行がありません）", name))
	}
	var names []string
	seen := map[string]bool{}
	for _, t := range e.Schema.Tables {
		if sum.Added[t.Name] > 0 || sum.Kept[t.Name] > 0 {
			names = append(names, t.Name)
			seen[t.Name] = true
		}
	}
	var rest []string
	for n := range sum.Added {
		if !seen[n] {
			rest = append(rest, n)
		}
	}
	sort.Strings(rest)
	for _, n := range append(names, rest...) {
		title := n
		if t := e.Schema.Table(n); t != nil {
			title = t.Title.In(e.Proj.Config.Language)
		}
		v.Messages = append(v.Messages, fmt.Sprintf("%s: 足した %d 行、すでにあった %d 行", title, sum.Added[n], sum.Kept[n]))
	}
	if sum.DeviceRows > 0 || sum.IfRows > 0 {
		v.Messages = append(v.Messages, fmt.Sprintf("新旧対照表（機器）に %d 行、新旧対照表（ポート）に %d 行を足しました", sum.DeviceRows, sum.IfRows))
	}
	for _, p := range o.Problems {
		v.Problems = append(v.Problems, p.String())
	}
	if o.Applied != nil {
		v.Open = table(e.MappingSheet(o.Applied, o.Mig, true))
	}
	return v, nil
}

func sourceWord(name string) string {
	switch name {
	case "old":
		return "旧コンフィグ"
	case "new":
		return "新コンフィグ"
	case "book":
		return "パラメータブック"
	}
	return name
}

// CompareView is the outcome of a comparison (spec 14.1 比較).
type CompareView struct {
	Title    string   `json:"title"`
	Counts   []Pair   `json:"counts"`
	Notes    []string `json:"notes"`
	Tables   []Table  `json:"tables"`
	Report   Files    `json:"report"`
	Findings int      `json:"findings"`
	// Items is the findings of a comparison with the old side (M-403 to
	// M-408): a row of a list is tied to one by the fingerprint in its last
	// column. Empty for the new side against the book.
	Items []FindingView `json:"items"`
	By    string        `json:"by"` // the name the last approval was given with
}

// Compare compares two sources ("old", "new", "book"), as the command
// "compare" does: the report is written.
func (s *Session) Compare(a, b string) (*CompareView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if a == b {
		return nil, errors.New("A と B に同じものを選んでいます")
	}
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	now := s.now()
	o, err := e.Compare(a, b, now)
	if err != nil {
		return nil, err
	}
	d := e.CompareDoc(o, now)
	v := &CompareView{Title: d.Title, Counts: pairs(d.Counts), Notes: strs(d.Notes), Tables: []Table{}, Report: Files{o.Report.XLSX, o.Report.HTML}, Findings: o.Findings(), Items: migFindings(e, o.Mig), By: s.name}
	s.mig = o.Mig
	for _, sh := range d.Sheets {
		v.Tables = append(v.Tables, table(sh))
	}
	return v, nil
}

// DiffLine is a line of a text comparison.
type DiffLine struct {
	Op   string `json:"op"` // "same" | "del" | "add" | "gap"
	A    int    `json:"a"`  // the line number in A (0 = not there)
	B    int    `json:"b"`
	Text string `json:"text"` // for a gap: how many lines are the same
}

// DiffView is two texts compared line by line (spec 12.1).
type DiffView struct {
	A       string     `json:"a"`
	B       string     `json:"b"`
	Equal   bool       `json:"equal"`
	Added   int        `json:"added"`
	Removed int        `json:"removed"`
	Lines   []DiffLine `json:"lines"` // the lines that differ, with the lines around them
	Side    []DiffRow  `json:"side"`  // the same lines side by side (spec 14.8.9)
}

// diffContext is how many lines that are the same are shown around a line
// that differs.
const diffContext = 3

// Diff compares two text files line by line, as the command "diff" does. It
// needs no case, and the texts are shown as they stand: the tool does not
// know which words of them are secret.
func (s *Session) Diff(a, b string) (*DiffView, error) {
	if a == "" || b == "" {
		return nil, errors.New("比べるファイルを 2 つ選んでください")
	}
	ta, err := project.ReadText(a)
	if err != nil {
		return nil, err
	}
	tb, err := project.ReadText(b)
	if err != nil {
		return nil, err
	}
	r := textdiff.Diff(ta, tb)
	v := &DiffView{A: a, B: b, Equal: r.Equal(), Added: r.Added, Removed: r.Removed, Lines: []DiffLine{}, Side: []DiffRow{}}
	if v.Equal {
		return v, nil
	}
	// near[i]: line i differs or is close to one that does.
	near := make([]bool, len(r.Lines))
	for i, l := range r.Lines {
		if l.Op == textdiff.Same {
			continue
		}
		for j := max(i-diffContext, 0); j <= min(i+diffContext, len(r.Lines)-1); j++ {
			near[j] = true
		}
	}
	ops := map[textdiff.Op]string{textdiff.Same: "same", textdiff.Del: "del", textdiff.Add: "add"}
	gap := 0
	flush := func() {
		if gap > 0 {
			v.Lines = append(v.Lines, DiffLine{Op: "gap", Text: fmt.Sprintf("同じ行 %d 行", gap)})
			gap = 0
		}
	}
	for i, l := range r.Lines {
		if !near[i] {
			gap++
			continue
		}
		flush()
		v.Lines = append(v.Lines, DiffLine{Op: ops[l.Op], A: l.ALine, B: l.BLine, Text: strings.TrimRight(l.Text, "\r\n")})
	}
	flush()
	v.Side = side(v.Lines)
	return v, nil
}

// ImportStale tells whether input/<source> changed since it was last
// imported (spec 14.19.3): the screen of the import then imports it again.
func (s *Session) ImportStale(source string) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := checkSource(source); err != nil {
		return false, err
	}
	e, err := s.begin()
	if err != nil {
		return false, err
	}
	return e.ImportStale(source)
}
