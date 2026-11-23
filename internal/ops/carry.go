package ops

import (
	"sort"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/judge"
	"github.com/kojio145/configchief/internal/pack"
	"github.com/kojio145/configchief/internal/report"
)

// The report of unhandled lines that may leave the site (spec 8.6): the
// shapes of the lines with the values and the names hidden, and how many
// devices and lines have each. It names no case, no device and no file.

// CarryShape is the unhandled lines of one shape.
type CarryShape struct {
	Pack    string
	Path    []string // the shapes of the lines above, the outermost first
	Shape   string
	Devices int
	Lines   int // the top-most unhandled lines; the lines below them are not counted
}

// HiddenWord is a word that was hidden as a name. It is a word of the site:
// shown to the person at the screen, never written to a file.
type HiddenWord struct {
	Word  string
	Lines int
}

// CarryOutcome is one report of unhandled lines to carry out.
type CarryOutcome struct {
	Source string
	Shapes []CarryShape
	Hidden []HiddenWord
	Unread []string // the files that were not imported: their lines are not in the report
	Report Files
}

// CarryOut writes the report of the unhandled lines of a source ("old" or
// "new") with everything hidden that is not known to be a word of the
// commands.
func (e *Env) CarryOut(source string, now time.Time) (*CarryOutcome, error) {
	if source == "" {
		source = "old"
	}
	j, err := e.Review(source)
	if err != nil {
		return nil, err
	}
	keep := map[string]bool{}
	for _, w := range e.Proj.Config.Export.KeepWords {
		for _, l := range pack.Letters(w) {
			keep[l] = true
		}
	}
	o := &CarryOutcome{Source: source, Unread: j.Unread}
	type sum struct {
		shape   CarryShape
		devices map[string]bool
	}
	sums := map[string]*sum{}
	hidden := map[string]int{}
	for _, g := range j.Groups {
		p := e.Packs[g.Key.Pack]
		known := func(w string) bool { return keep[w] || (p != nil && p.Words[w]) }
		lines := 0
		for _, d := range g.Devices {
			lines += len(d.Heads)
		}
		var words []string
		cs := CarryShape{Pack: g.Key.Pack}
		for _, up := range g.Key.Path {
			s, h := judge.CarryShape(up, known)
			cs.Path = append(cs.Path, s)
			words = append(words, h...)
		}
		var h []string
		cs.Shape, h = judge.CarryShape(g.Key.Shape, known)
		words = append(words, h...)
		for _, w := range words {
			hidden[w] += lines
		}
		id := cs.Pack + "\x00" + strings.Join(cs.Path, "\x01") + "\x00" + cs.Shape
		s := sums[id]
		if s == nil {
			s = &sum{shape: cs, devices: map[string]bool{}}
			sums[id] = s
		}
		s.shape.Lines += lines
		for _, d := range g.Devices {
			s.devices[d.Device] = true
		}
	}
	for _, s := range sums {
		s.shape.Devices = len(s.devices)
		o.Shapes = append(o.Shapes, s.shape)
	}
	sort.Slice(o.Shapes, func(a, b int) bool {
		x, y := o.Shapes[a], o.Shapes[b]
		switch {
		case x.Pack != y.Pack:
			return x.Pack < y.Pack
		case x.Lines != y.Lines:
			return x.Lines > y.Lines
		case strings.Join(x.Path, "\x01") != strings.Join(y.Path, "\x01"):
			return strings.Join(x.Path, "\x01") < strings.Join(y.Path, "\x01")
		}
		return x.Shape < y.Shape
	})
	for w, n := range hidden {
		o.Hidden = append(o.Hidden, HiddenWord{w, n})
	}
	sort.Slice(o.Hidden, func(a, b int) bool {
		if o.Hidden[a].Lines != o.Hidden[b].Lines {
			return o.Hidden[a].Lines > o.Hidden[b].Lines
		}
		return o.Hidden[a].Word < o.Hidden[b].Word
	})
	files, err := e.writeReport(now, "unhandled_shapes", e.carryDoc(o, now))
	if err != nil {
		return nil, err
	}
	o.Report = files
	e.Proj.Log(now, "読めなかった行の持ち出し用レポート: %s、形 %d、伏せた語 %d 種類", source, len(o.Shapes), len(o.Hidden))
	return o, nil
}

// carryDoc makes the report. Nothing of the case goes into it but the
// shapes: not its name, no device, no file.
func (e *Env) carryDoc(o *CarryOutcome, now time.Time) *report.Doc {
	d := &report.Doc{Title: "読めなかった行（持ち出し用）"}
	d.Facts = append(d.Facts, report.Pair{Label: "作った日", Value: now.Format("2006-01-02")})
	if e.Version != "" {
		d.Facts = append(d.Facts, report.Pair{Label: "ツール", Value: "ConfigChief " + e.Version})
	}
	seen := map[string]bool{}
	lines := 0
	for _, s := range o.Shapes {
		lines += s.Lines
		if p := e.Packs[s.Pack]; p != nil && !seen[s.Pack] {
			seen[s.Pack] = true
			d.Facts = append(d.Facts, report.Pair{Label: "パック", Value: p.ID + " " + p.Version})
		}
	}
	d.Counts = []report.Pair{
		{Label: "形", Value: num(len(o.Shapes))},
		{Label: "読めなかった行", Value: num(lines)},
	}
	d.Notes = append(d.Notes,
		"メーカーパックが読めなかった行を、値と名前を伏せた形にまとめたものです。案件の名前・機器の名前・ファイルの名前は入れていません。",
		"伏せ方: 数字・アドレス・引用符の中は記号（<n>・<ip> など）。秘密が続く語の後ろは "+judge.MarkSecret+"、人が書く文が続く語の後ろは "+judge.MarkText+"。そのほかの語は、パックが知っている語だけを残し、残りは "+judge.MarkName+"。",
		"伏せ方は機械的です。名前や秘密が、コマンドの語と同じ綴りなら残ります。人に渡す前に、すべての行を目で確かめてください。")
	sh := report.Sheet{Name: "形", Lead: "行数は、いちばん上の読めなかった行の数です（その下の行は数えていません）。",
		Header: []string{"パック", "親の行", "行の形", "台数", "行数"}, Code: []int{1, 2}}
	for _, s := range o.Shapes {
		sh.Rows = append(sh.Rows, []string{s.Pack, strings.Join(s.Path, " > "), s.Shape, num(s.Devices), num(s.Lines)})
	}
	d.Sheets = append(d.Sheets, sh)
	return d
}
