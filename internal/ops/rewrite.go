package ops

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/compare"
	"github.com/kojio145/configchief/internal/convert"
	"github.com/kojio145/configchief/internal/importer"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/render"
	"github.com/kojio145/configchief/internal/report"
	"github.com/kojio145/configchief/internal/schema"
)

// The kinds of the rewriting report (spec 10.4): every line of the old
// configurations is in exactly one of them.
const (
	RwAuto    = "auto"    // 自動で書き換えた
	RwUnread  = "unread"  // 旧にあって取り込めなかった
	RwCannot  = "cannot"  // 取り込めたが新メーカーのパックが書けない
	RwModel   = "model"   // 新機種が対応していない機能
	RwChanged = "changed" // 旧と値を変えた
	RwIgnored = "ignored" // 無視
	RwUnsure  = "unsure"  // 確かめられない
)

// RewriteKinds lists the kinds in the order the reports show them.
var RewriteKinds = []string{RwAuto, RwUnread, RwCannot, RwModel, RwChanged, RwUnsure, RwIgnored}

var rewriteWords = map[string]string{
	RwAuto:    "自動で書き換えた",
	RwUnread:  "旧にあって取り込めなかった",
	RwCannot:  "新メーカーのパックが書けない",
	RwModel:   "新機種が対応していない",
	RwChanged: "旧と値を変えた",
	RwIgnored: "無視",
	RwUnsure:  "確かめられない",
}

// RewriteWord gives the words of a kind.
func RewriteWord(kind string) string { return rewriteWords[kind] }

// rank orders the kinds of the places of one line: the first one wins
// (spec 10.4).
var rank = map[string]int{RwIgnored: 0, RwUnread: 1, RwModel: 2, RwCannot: 3, RwChanged: 4, RwUnsure: 5, RwAuto: 6}

// The most new lines a row of the report shows for one old line.
const mostNewLines = 10

// noNewLine is the reason of a value the new pack read but wrote no line
// for.
const noNewLine = "新メーカーでは行を書かない値です（パックは値を読みました）"

// htmlAllLines is the most lines the HTML form of the rewriting report
// lists whole (spec 13.4).
const htmlAllLines = 20000

// RewriteLine is one line of an old configuration and where it went.
type RewriteLine struct {
	Device string // the old host name ("" for a file that was not read)
	File   string
	Line   int
	Text   string // the secrets masked as the env says
	Kind   string
	// To names where the values went: the table, the row and the column, by
	// the names of the book, after the mapping.
	To        string
	NewDevice string
	NewLines  []int
	NewText   string // the new lines, one under another
	Verdict   string // the judgment of an unhandled line, the state of the findings
	Reason    string
	Rules     []string
	// Open tells that a person still has to look at the line: an unhandled
	// line not judged or on hold, a finding not approved, a line that could
	// not be followed.
	Open bool
}

// RewriteDevice is one old device of the rewriting report.
type RewriteDevice struct {
	Device    string
	File      string
	NewDevice string
	State     string // what the generation did for the new device, in words
	Total     int
	Counts    map[string]int
	Open      int
}

// NewOnlyLine is a line of a new configuration that no old line led to.
type NewOnlyLine struct {
	Device string
	Line   int
	Text   string
	From   string
	Values string
}

// Rewrite is the rewriting report (spec 10.4, 13.4).
type Rewrite struct {
	Lines   []RewriteLine
	Devices []RewriteDevice
	NewOnly []NewOnlyLine
	Notes   []string
	// Counts gives the lines of each kind.
	Counts map[string]int
	Total  int
	// Unjudged counts the unhandled lines not judged or on hold; Unapproved
	// the lines with a finding that is not approved.
	Unjudged, Unapproved int
}

// rewriteInput is what the rewriting report is made of beside the outcome
// of the generation: the old side, the mapping put on and the comparison
// with the book.
type rewriteInput struct {
	src     *Source
	applied *convert.Applied
	mig     *Verdicts
	judging *Judging
	design  *model.Dataset // the book, filled with the defaults
}

// rewriteCase tells whether a case is one of renewal: input/old holds a file
// and 機器対応 names a new host (spec 13.4).
func rewriteCase(e *Env, bk *book.Book) bool {
	if bk == nil {
		return false
	}
	named := false
	for _, d := range bk.Devices {
		named = named || strings.TrimSpace(d.NewHost) != ""
	}
	if !named {
		return false
	}
	paths, err := e.Proj.Inputs("old")
	return err == nil && len(paths) > 0
}

// readRewrite reads the old side for the rewriting report. design is the
// book filled with the defaults, as the checks of the generation have it.
func (e *Env) readRewrite(bk *book.Book, design *model.Dataset) (*rewriteInput, error) {
	quiet := *e
	quiet.Tick = nil // the progress is that of the generation
	src, err := quiet.Import("old", "")
	if err != nil {
		return nil, err
	}
	a := convert.Apply(e.Schema, src.Data, bk.Devices, bk.Ifs)
	findings := mappingFindings(a)
	findings = append(findings, e.diffFindings(compare.Datasets(e.Schema, a.Data, design), true)...)
	mig, err := e.verdicts(findings, bk.Data)
	if err != nil {
		return nil, err
	}
	jd, err := e.judging(src, hostView{known: true, hosts: a.Hosts, book: bk.Data})
	if err != nil {
		return nil, err
	}
	return &rewriteInput{src: src, applied: a, mig: mig, judging: jd, design: design}, nil
}

// HasRewrite tells whether the generation made a rewriting report.
func (o *GenerateOutcome) HasRewrite() bool { return o.rewrite != nil }

// Rewrite gives the rewriting report of a generation for a screen (the
// secrets masked); nil if the case is not one of renewal.
func (e *Env) Rewrite(o *GenerateOutcome) *Rewrite {
	if o == nil || o.rewrite == nil {
		return nil
	}
	if o.shownRw == nil {
		o.shownRw = e.forScreen().rewriteOf(o)
	}
	return o.shownRw
}

// destination is one place the values of an old line went.
type destination struct {
	table, oldKey, column string
}

// newIndex gives the lines of the generated configuration of a device by
// the cells they took their values from.
type newIndex struct {
	out   *render.Output
	cells map[string][]int // table \x00 key \x00 column → lines
	conds map[string][]int // the same for the cells a line was written on the condition of
	rows  map[string][]int // table \x00 key → lines
	used  map[int]bool     // the lines an old line led to
	// skip marks the blank lines and the comments (the separators a pack
	// writes): they are no line an old line leads to.
	skip map[int]bool
}

func indexOutput(o *render.Output, comments []string) *newIndex {
	x := &newIndex{out: o, cells: map[string][]int{}, conds: map[string][]int{}, rows: map[string][]int{}, used: map[int]bool{}, skip: map[int]bool{}}
	for i, l := range o.Lines {
		body := strings.TrimSpace(l.Text)
		if body == "" || hasAnyPrefix(body, comments) {
			x.skip[i+1] = true
			continue
		}
		seen := map[string]bool{}
		for _, v := range l.Values {
			rk := v.Row.Table + "\x00" + v.Row.KeyText()
			ck := rk + "\x00" + v.Column
			if !seen[ck] {
				seen[ck] = true
				x.cells[ck] = append(x.cells[ck], i+1)
			}
			if !seen[rk] {
				seen[rk] = true
				x.rows[rk] = append(x.rows[rk], i+1)
			}
		}
		for _, v := range l.Conds {
			ck := v.Row.Table + "\x00" + v.Row.KeyText() + "\x00" + v.Column
			if !seen["c"+ck] {
				seen["c"+ck] = true
				x.conds[ck] = append(x.conds[ck], i+1)
			}
		}
	}
	return x
}

// cell gives the lines that took their value from a cell or, if none did,
// the lines written on its condition ("no shutdown" for the column 有効).
func (x *newIndex) cell(key string) []int {
	if lines := x.cells[key]; len(lines) > 0 {
		return lines
	}
	return x.conds[key]
}

// lineSet keeps line numbers once each, in order.
type lineSet map[int]bool

func (s lineSet) add(lines []int) {
	for _, n := range lines {
		s[n] = true
	}
}

func (s lineSet) sorted() []int {
	out := make([]int, 0, len(s))
	for n := range s {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

// verdictOf tells the state of a finding for the column 判定.
func verdictOf(j *Judged) (string, bool) {
	switch {
	case j.Severity == check.Info:
		return j.RuleID + " 情報", false
	case j.Approval != nil:
		return j.RuleID + " 承認済み（" + j.Approval.ID + "）", false
	}
	return j.RuleID + " 未承認（" + check.SeverityTitle(j.Severity) + "）", true
}

// The checks of the generation that put an old line in a kind.
var (
	modelRules  = map[string]bool{"G-607": true}
	cannotRules = map[string]bool{"G-601": true, "G-602": true, "G-603": true, "G-606": true}
)

// findingHits tells whether a finding is about a place: the same table, the
// same row (or none named) and the same column (or none named on either).
func findingHits(f *Judged, table, key, column string) bool {
	return f.Ref.Table == table && (f.Ref.Key == "" || f.Ref.Key == key) &&
		(f.Ref.Column == "" || column == "" || f.Ref.Column == column)
}

// rewriteOf builds the rewriting report from the outcome of a generation.
func (e *Env) rewriteOf(o *GenerateOutcome) *Rewrite {
	in := o.rewrite
	a := in.applied
	rw := &Rewrite{Counts: map[string]int{}}

	// The findings of the generation by new device.
	gen := map[string][]*Judged{}
	for i := range o.Items {
		f := &o.Items[i]
		if modelRules[f.RuleID] || cannotRules[f.RuleID] {
			gen[f.Device] = append(gen[f.Device], f)
		}
	}
	mig := map[string][]*Judged{}
	for i := range in.mig.Items {
		f := &in.mig.Items[i]
		if f.RuleID == "M-403" || f.RuleID == "M-405" {
			mig[f.Device] = append(mig[f.Device], f)
		}
	}
	runs := map[string]*DeviceRun{}
	for i := range o.Devices {
		runs[o.Devices[i].Host] = &o.Devices[i]
	}
	// The generated lines are indexed side by side, before the files are
	// gone through (also side by side): after this, nothing is written to
	// what the files share.
	indexes := make([]*newIndex, len(o.Devices))
	work := newSpread(len(o.Devices), func(_, i int) {
		r := &o.Devices[i]
		if r.Output == nil {
			return
		}
		var comments []string
		if p := e.Packs[r.Pack]; p != nil {
			comments = p.Options.CommentPrefixes
		}
		indexes[i] = indexOutput(r.Output, comments)
	})
	outs := map[string]*newIndex{}
	for i, r := range o.Devices {
		work.wait(i)
		if indexes[i] != nil {
			outs[r.Host] = indexes[i]
		}
	}
	work.stop()
	index := func(host string) *newIndex { return outs[host] }
	applied := rowIndex(a.Data)
	design := rowIndex(in.design)
	dropped, cleared, clashed := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, n := range a.Dropped {
		dropped[n.Table+"\x00"+n.Key] = true
	}
	for _, n := range a.Cleared {
		cleared[n.Table+"\x00"+n.Key+"\x00"+n.Column] = true
	}
	for _, n := range a.Clashes {
		clashed[n.Table+"\x00"+n.Key] = true
	}
	g607 := false
	for _, d := range o.Devices {
		if ran, _ := o.ruleRan(mustRule("G-607"), d.Host); ran {
			g607 = true
		}
	}
	if !g607 {
		rw.Notes = append(rw.Notes, "機種のチェック（G-607）は今回行っていないため、「新機種が対応していない」には何も入っていません。")
	}

	// place decides the kind of one place of a line.
	type verdict struct {
		kind, reason string
		lines        []int
		rules        []string
		states       []string
		open         bool
	}
	place := func(oldHost, newHost string, d destination) verdict {
		ok := d.table + "\x00" + d.oldKey
		switch {
		case dropped[ok]:
			return verdict{kind: RwIgnored, reason: "「IF 対応」で新 IF 名を - にした IF の行です（持っていきません）"}
		case d.column != "" && cleared[ok+"\x00"+d.column]:
			return verdict{kind: RwIgnored, reason: "新 IF 名を - にした IF を指していた値です（空にしました）"}
		case clashed[ok]:
			return verdict{kind: RwChanged, reason: "複数の旧 IF が同じ新 IF に当たり、キーが重なったため書いていません", rules: []string{"M-408"}, open: true}
		}
		newKey, carried := a.Keys[ok]
		if !carried {
			return verdict{kind: RwUnsure, reason: "設計データの行き先が見つかりません", open: true}
		}
		v := verdict{}
		row := applied[d.table+"\x00"+newKey]
		refKey := ""
		if t := e.Schema.Table(d.table); t != nil && row != nil {
			refKey = check.RefOf(t, row, "").Key
		}
		hit := func(list []*Judged, rules map[string]bool) bool {
			found := false
			for _, f := range list {
				switch {
				case rules != nil && !rules[f.RuleID]:
					continue
				case f.RuleID == "G-602":
					// G-602 names no column: the cells no template read are
					// looked up one by one below.
					continue
				case f.RuleID == "M-405" && d.column == "":
					continue
				}
				if findingHits(f, d.table, refKey, d.column) {
					found = true
					st, open := verdictOf(f)
					v.rules = append(v.rules, f.RuleID)
					v.states = append(v.states, st)
					v.open = v.open || open
				}
			}
			return found
		}
		x := index(newHost)
		if x != nil {
			// The new lines are shown whatever the kind.
			rk := d.table + "\x00" + newKey
			set := lineSet{}
			if d.column != "" {
				set.add(x.cell(rk + "\x00" + d.column))
			} else {
				// The line that opened the row: the new lines of the key of
				// the row (the device column, for a table keyed by it alone).
				if t := e.Schema.Table(d.table); t != nil {
					for _, k := range rowKeys(t) {
						set.add(x.cells[rk+"\x00"+k])
					}
				}
				if len(set) == 0 {
					set.add(x.rows[rk])
				}
			}
			v.lines = set.sorted()
		}
		if hit(gen[newHost], modelRules) {
			v.kind, v.reason = RwModel, "新機種が対応していない機能です"
			return v
		}
		// unused puts on the finding of G-602 of the row (or of the whole
		// table) the cell is told in.
		unused := func() {
			for _, f := range gen[newHost] {
				if f.RuleID == "G-602" && f.Ref.Table == d.table && (f.Ref.Key == refKey || f.Ref.Key == "") {
					st, open := verdictOf(f)
					v.states = append(v.states, st)
					v.open = v.open || open
					break
				}
			}
			v.rules = append(v.rules, "G-602")
		}
		cannot := hit(gen[newHost], cannotRules)
		if !cannot && x != nil && d.column != "" && x.out.Unused(d.table, newKey, d.column) {
			cannot = true
			unused()
		}
		if cannot {
			v.kind, v.reason = RwCannot, "新メーカーのパックが書けない値です"
			if !v.open && len(v.states) == 0 {
				v.open = true
			}
			return v
		}
		if hit(mig[newHost], nil) {
			v.kind = RwChanged
			v.reason = "パラメータブックで値を変えました"
			if design[d.table+"\x00"+newKey] == nil {
				v.reason = "パラメータブックにこの行がありません"
			}
			return v
		}
		if design[d.table+"\x00"+newKey] == nil {
			v.kind, v.reason = RwChanged, "パラメータブックにこの行がありません"
			return v
		}
		r := runs[newHost]
		if x == nil {
			v.kind, v.open = RwUnsure, true
			switch {
			case r == nil:
				v.reason = "新機器 " + newHost + " は、パラメータブックの機器のシートにありません"
			case r.Why != "":
				v.reason = "新機器を生成していないため、新の行を確かめられません（" + r.Why + "）"
			default:
				v.reason = "新機器を生成していないため、新の行を確かめられません（" + GenStateWord(r.State) + "）"
			}
			return v
		}
		switch {
		case len(v.lines) > 0:
			v.kind = RwAuto
		case x.out.Failed:
			v.kind, v.reason, v.open = RwUnsure, "テンプレートが失敗して文面がそろっていないため、新の行が見つかりません", true
		case d.column == "" && x.out.UnusedRow(d.table, newKey):
			v.kind, v.reason = RwCannot, "新メーカーのパックが書けない行です"
			unused()
		default:
			v.kind, v.reason = RwAuto, noNewLine
		}
		return v
	}

	jd := in.judging
	// fileOf goes through the lines of one file. It writes nothing shared:
	// the files are gone through side by side.
	type fileOut struct {
		skip      bool
		perDevice RewriteDevice
		lines     []RewriteLine
		notes     []string
		host      string
		used      []int // the new lines of host the file led to
	}
	fileOf := func(f *File) (fo fileOut) {
		if f.Left != "" {
			fo.skip = true
			fo.notes = append(fo.notes, f.Path+": "+f.Left)
			return
		}
		dev := &fo.perDevice
		*dev = RewriteDevice{File: f.Path, Counts: map[string]int{}}
		add := func(l RewriteLine) {
			fo.lines = append(fo.lines, l)
			dev.Total++
			dev.Counts[l.Kind]++
			if l.Open {
				dev.Open++
			}
		}
		if f.Result == nil {
			dev.Device = f.Device
			fo.notes = append(fo.notes, f.Path+" は取り込めませんでした（"+f.Err+"）。空行のほかの行を「旧にあって取り込めなかった」に入れています。")
			text, err := project.ReadText(e.Proj.Path(f.Path))
			if err != nil {
				fo.notes = append(fo.notes, f.Path+" を読めません: "+err.Error())
			}
			for i, s := range splitLines(text) {
				l := RewriteLine{Device: f.Device, File: f.Path, Line: i + 1, Text: s}
				if strings.TrimSpace(s) == "" {
					l.Kind, l.Reason = RwIgnored, "空行です"
				} else {
					l.Kind, l.Reason, l.Open = RwUnread, "ファイルを取り込めませんでした: "+f.Err, true
				}
				add(l)
			}
			return
		}
		r := f.Result
		dev.Device = r.Device
		newHost, carried := a.Hosts[r.Device]
		dev.NewDevice, fo.host = newHost, newHost
		if run := runs[newHost]; carried && run != nil {
			dev.State = GenStateWord(run.State)
		}
		m := e.newMasker(r)
		for _, l := range r.Lines {
			line := RewriteLine{Device: r.Device, File: f.Path, Line: l.Line, Text: m.text(l)}
			switch {
			case l.Separator:
				line.Kind, line.Reason = RwIgnored, "コメント・空行です"
				if l.Outside {
					line.Reason = "端末ログの、設定の外の行です"
				}
			case l.State == importer.Ignored:
				line.Kind, line.Reason = RwIgnored, "パックが「設定ではない」と定めた行です"
				if l.Reason != "" {
					line.Reason += "（" + l.Reason + "）"
				}
			case l.State == importer.Unknown:
				line.Kind, line.Verdict, line.Reason, line.Open = RwUnread, notJudged, l.Reason, true
				if v, ok := jd.Line(f.Path, l.Line); ok {
					line.Verdict, line.Open = v.Word(), v.Open()
					if v.Reason != "" {
						line.Reason = v.Reason
					}
				}
			case !carried:
				line.Kind, line.Reason = RwIgnored, "「機器対応」で新ホスト名が空の機器です（書き換えの対象外）"
			default:
				// The work tables of a pack (x_…) are a step on the way: the
				// line points at the design data through them too.
				// A line that opened a row of a table keyed by the device alone
				// (機器, 遠隔ログイン) and gave it values is followed by the
				// values: the row would bring the line of the host name.
				withCells := map[string]bool{}
				for _, ref := range l.Refs {
					if ref.Column != "" {
						withCells[ref.Table+"\x00"+ref.Key] = true
					}
				}
				var places []destination
				for _, ref := range l.Refs {
					t := e.Schema.Table(ref.Table)
					if t == nil {
						continue
					}
					if ref.Column == "" && withCells[ref.Table+"\x00"+ref.Key] && len(rowKeys(t)) == 1 && rowKeys(t)[0] == t.DeviceColumn() {
						continue
					}
					places = append(places, destination{table: ref.Table, oldKey: ref.Key, column: ref.Column})
				}
				if len(places) == 0 {
					line.Kind, line.Reason, line.Open = RwUnsure, "取り込みましたが、設計データの行き先が記録されていません", true
					break
				}
				line.NewDevice = newHost
				var all []verdict
				best := ""
				set := lineSet{}
				var to, states []string
				rules := map[string]bool{}
				for _, d := range places {
					v := place(r.Device, newHost, d)
					all = append(all, v)
					if best == "" || rank[v.kind] < rank[best] {
						best = v.kind
					}
					set.add(v.lines)
					line.Open = line.Open || v.open
					states = append(states, v.states...)
					for _, id := range v.rules {
						rules[id] = true
					}
					to = append(to, e.rewriteTo(d, a, applied))
				}
				line.Kind = best
				line.To = strings.Join(uniq(to), "; ")
				line.Verdict = strings.Join(uniq(states), "; ")
				for id := range rules {
					line.Rules = append(line.Rules, id)
				}
				sort.Strings(line.Rules)
				if x := index(newHost); x != nil {
					line.NewLines = set.sorted()
					var texts []string
					fo.used = append(fo.used, line.NewLines...)
					for i, n := range line.NewLines {
						switch {
						case i < mostNewLines:
							texts = append(texts, e.lineText(x.out, n))
						case i == mostNewLines:
							texts = append(texts, fmt.Sprintf("ほか %d 行", len(line.NewLines)-mostNewLines))
						}
					}
					line.NewText = strings.Join(texts, "\n")
				}
				// The reasons of the places that gave the kind.
				var reasons []string
				for _, v := range all {
					if v.kind == best && v.reason != "" && !(v.reason == noNewLine && len(line.NewLines) > 0) {
						reasons = append(reasons, v.reason)
					}
				}
				line.Reason = strings.Join(uniq(reasons), "; ")
			}
			add(line)
		}
		return
	}
	files := in.src.Files
	done := make([]fileOut, len(files))
	work = newSpread(len(files), func(_, i int) { done[i] = fileOf(files[i]) })
	for i := range files {
		work.wait(i)
		fo := done[i]
		rw.Notes = append(rw.Notes, fo.notes...)
		if fo.skip {
			continue
		}
		rw.Lines = append(rw.Lines, fo.lines...)
		rw.Devices = append(rw.Devices, fo.perDevice)
		if x := index(fo.host); x != nil {
			for _, n := range fo.used {
				x.used[n] = true
			}
		}
	}
	work.stop()

	// The lines of the new configurations no old line led to.
	for _, d := range o.Devices {
		x := index(d.Host)
		if x == nil {
			continue
		}
		for i, l := range x.out.Lines {
			// The lines a pack always writes (no value, no condition: the
			// separators, the switches it turns on) are left out.
			if x.used[i+1] || x.skip[i+1] || (!l.Extra && len(l.Values) == 0 && len(l.Conds) == 0) {
				continue
			}
			from := "パック（" + l.Template
			if l.Extra {
				from = "手書き（" + l.Template
			}
			if l.TLine > 0 {
				from += " の " + num(l.TLine) + " 行目"
			}
			rw.NewOnly = append(rw.NewOnly, NewOnlyLine{Device: d.Host, Line: i + 1, Text: e.lineText(x.out, i+1), From: from + "）", Values: e.cellRefText(l.Values)})
		}
	}

	for _, l := range rw.Lines {
		rw.Total++
		rw.Counts[l.Kind]++
		switch {
		case l.Kind == RwUnread && l.Open:
			rw.Unjudged++
		case l.Open && l.Kind != RwUnsure:
			rw.Unapproved++
		}
	}
	if e.mask() {
		rw.Notes = append(rw.Notes, "秘密の列に入った値は伏せています。未対応の行にある秘密の値は、ツールが見分けられないため伏せられません。このレポートを人に渡す前に確かめてください。")
	}
	return rw
}

// rowKeys gives the key columns of a table but the device column; the
// device column if it is the whole key.
func rowKeys(t *schema.Table) []string {
	var keys []string
	for _, k := range t.Key {
		if k != t.DeviceColumn() {
			keys = append(keys, k)
		}
	}
	if len(keys) == 0 {
		return t.Key
	}
	return keys
}

func hasAnyPrefix(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if p != "" && strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func mustRule(id string) check.Rule {
	r, _ := check.Lookup(id)
	return r
}

// splitLines splits a text into its lines, as the import counts them.
func splitLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.TrimSuffix(text, "\n")
	if text == "" {
		return nil
	}
	return strings.Split(text, "\n")
}

// rewriteTo names a place of the new side: the table, the row (after the
// mapping) and the column, by the names the book shows.
func (e *Env) rewriteTo(d destination, a *convert.Applied, applied map[string]*model.Row) string {
	s := e.tableTitle(d.table)
	if k, ok := a.Keys[d.table+"\x00"+d.oldKey]; ok {
		if r := applied[d.table+"\x00"+k]; r != nil {
			if label := e.rowLabel(r); label != "" {
				s += "（" + label + "）"
			}
		}
	}
	if d.column == "" {
		return s + " の行"
	}
	return s + " の " + e.columnTitle(d.table, d.column)
}

// rewriteDoc builds the rewriting report (spec 13.4).
func (e *Env) rewriteDoc(rw *Rewrite, now time.Time) *report.Doc {
	d := &report.Doc{Title: "書き換えレポート（旧コンフィグ → 新コンフィグ）", Facts: e.facts(now)}
	d.Facts = append(d.Facts, report.Pair{Label: "旧", Value: "input/old（「機器対応」「IF 対応」を当てた後）"},
		report.Pair{Label: "新", Value: "パラメータブックから生成したコンフィグ"})
	d.Counts = append(d.Counts, report.Pair{Label: "旧コンフィグの全行", Value: num(rw.Total)})
	for _, k := range RewriteKinds {
		d.Counts = append(d.Counts, report.Pair{Label: RewriteWord(k), Value: num(rw.Counts[k])})
	}
	d.Counts = append(d.Counts,
		report.Pair{Label: "うち未判定・保留の未対応の行", Value: num(rw.Unjudged)},
		report.Pair{Label: "うち未承認の指摘がある行", Value: num(rw.Unapproved)},
		report.Pair{Label: "新にだけある行", Value: num(len(rw.NewOnly))})
	d.Notes = append(d.Notes, "旧コンフィグのすべての行を、上の区分のどれか 1 つに入れています（仕様書 10.4）。旧の行 → 設計データ → 新の行 を「行の一覧」で 1 行ずつ追えます。")

	devs := report.Sheet{Name: "機器別", Header: []string{"旧機器", "ファイル", "新機器", "新機器の状態", "全行"}}
	for _, k := range RewriteKinds {
		devs.Header = append(devs.Header, RewriteWord(k))
	}
	devs.Header = append(devs.Header, "人が見る行")
	for _, dv := range rw.Devices {
		row := []string{dv.Device, dv.File, dv.NewDevice, dv.State, num(dv.Total)}
		for _, k := range RewriteKinds {
			row = append(row, num(dv.Counts[k]))
		}
		row = append(row, num(dv.Open))
		devs.Rows = append(devs.Rows, row)
		mark := ""
		if dv.Open > 0 {
			mark = report.MarkWarn
		}
		devs.Marks = append(devs.Marks, mark)
	}

	lines := report.Sheet{Name: "行の一覧", Lead: "旧コンフィグのすべての行と、行き先（設計データ）、新コンフィグの行です。",
		Header: RewriteHeader, Code: []int{3, 9}}
	for _, l := range rw.Lines {
		lines.Rows = append(lines.Rows, l.Cells())
		lines.Marks = append(lines.Marks, l.Mark())
	}
	if len(rw.Lines) > htmlAllLines {
		// A browser does not open a page of a hundred thousand rows: the
		// HTML form keeps the lines a person reads through.
		lines.HTMLRows = []int{}
		for i, l := range rw.Lines {
			if l.Kind != RwAuto && l.Kind != RwIgnored {
				lines.HTMLRows = append(lines.HTMLRows, i)
			}
		}
		lines.HTMLNote = fmt.Sprintf("行が多いため（%d 行）、HTML には「%s」「%s」の行を載せていません（%d 行を載せています）。全行は rewrite.xlsx にあります。",
			len(rw.Lines), RewriteWord(RwAuto), RewriteWord(RwIgnored), len(lines.HTMLRows))
	}

	only := report.Sheet{Name: "新にだけある行", Lead: "新コンフィグの行のうち、どの旧の行ともつながらなかった行です（追加コンフィグの行、パラメータブックで足した値の行など）。",
		Header: []string{"新機器", "行番号", "文面", "由来", "値"}, Code: []int{2}}
	for _, l := range rw.NewOnly {
		only.Rows = append(only.Rows, []string{l.Device, num(l.Line), l.Text, l.From, l.Values})
	}
	notes := report.Sheet{Name: "注意", Header: []string{"内容"}}
	for _, n := range rw.Notes {
		notes.Rows = append(notes.Rows, []string{n})
	}
	d.Sheets = []report.Sheet{devs, lines, only, notes}
	return d
}

// RewriteHeader is the heading of the list of lines of the rewriting report.
var RewriteHeader = []string{"旧機器", "ファイル", "旧の行番号", "旧の文面", "区分", "行き先（表・行・列）", "新機器", "新の行番号", "新の行の数", "新の文面", "判定", "理由", "関係する指摘"}

// Cells gives the line as a row of the list (RewriteHeader).
func (l RewriteLine) Cells() []string {
	nums := make([]string, len(l.NewLines))
	for i, n := range l.NewLines {
		nums[i] = num(n)
	}
	count := ""
	if len(l.NewLines) > 0 {
		count = num(len(l.NewLines))
	}
	return []string{l.Device, l.File, num(l.Line), l.Text, RewriteWord(l.Kind), l.To, l.NewDevice, strings.Join(nums, ", "), count, l.NewText, l.Verdict, l.Reason, strings.Join(l.Rules, ", ")}
}

// Mark gives the colour of the line in the HTML form and on a screen.
func (l RewriteLine) Mark() string {
	switch {
	case l.Kind == RwIgnored:
		return report.MarkIgnored
	case l.Open && (l.Kind == RwUnread || l.Kind == RwCannot || l.Kind == RwModel):
		return report.MarkUnknown
	case l.Open || l.Kind == RwChanged || l.Kind == RwUnsure:
		return report.MarkWarn
	}
	return ""
}

// writeRewrite writes the rewriting report of a generation.
func (e *Env) writeRewrite(o *GenerateOutcome, now time.Time) error {
	if o.rewrite == nil {
		return nil
	}
	rw := e.rewriteOf(o)
	files, err := e.writeReport(now, "rewrite", e.rewriteDoc(rw, now))
	if err != nil {
		return err
	}
	o.RewriteReport, o.written = files, rw
	return nil
}

// RewriteSummary gives the rewriting report that was written (nil = none):
// its numbers for the command line.
func (o *GenerateOutcome) RewriteSummary() *Rewrite { return o.written }
