package importer

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"unicode"

	"github.com/kojio145/configchief/internal/conftree"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/netx"
	"github.com/kojio145/configchief/internal/pack"
	"github.com/kojio145/configchief/internal/schema"
	"github.com/kojio145/configchief/internal/termlog"
	"github.com/kojio145/configchief/internal/tmpl"
)

// The states of a line (spec 8.2).
const (
	Taken   = "taken"   // a rule hit and the values went into the design data
	Ignored = "ignored" // the pack says the line carries no setting
	Unknown = "unknown" // no rule could take the line
)

// Ref names a cell that a line fed.
type Ref struct {
	Table  string
	Key    string
	Column string // "" = the line opened the row
}

// LineState is what became of one line of the configuration.
type LineState struct {
	File      string
	Line      int
	Parent    int // the line this one hangs below (0 = the top level)
	Text      string
	State     string
	RuleID    string
	Reason    string // why ignored, or why unknown
	Refs      []Ref
	Separator bool // a comment, a blank line, or a line outside the configuration
	Outside   bool // in a terminal log: not part of the configuration (Separator is set too)
}

// Result is one imported configuration.
type Result struct {
	Device   string
	Data     *model.Dataset
	Work     *model.Dataset // the work tables (x_…), for looking into a pack
	Lines    []LineState    // every line of the text; Lines[i] is line i+1
	Warnings []string
	Notes    []string // good to know, nothing a person has to look at
}

// Count is the number of lines by state. Outside is the part of Separators
// that a terminal log has around its configuration.
type Count struct {
	Total, Separators, Taken, Ignored, Unknown int
	Outside                                    int
}

// Count counts the lines. Separators (comments, blank lines, the lines of a
// terminal log that are not configuration) are counted apart from the lines
// a rule ignored.
func (r *Result) Count() Count {
	c := Count{Total: len(r.Lines)}
	for _, l := range r.Lines {
		if l.Outside {
			c.Outside++
		}
		switch {
		case l.Separator:
			c.Separators++
		case l.State == Taken:
			c.Taken++
		case l.State == Ignored:
			c.Ignored++
		default:
			c.Unknown++
		}
	}
	return c
}

// Covered is the share of the lines that are taken or ignored, separators
// left out (0 to 1).
func (c Count) Covered() float64 {
	n := c.Total - c.Separators
	if n == 0 {
		return 1
	}
	return float64(c.Taken+c.Ignored) / float64(n)
}

type run struct {
	// whenHeld tells deriveItem that the "when" of the step was asked and
	// held already (deriveOne).
	whenHeld bool
	p      *pack.Pack
	sc     *schema.Set // nil: no table definitions, nothing is checked
	file   string
	eng    *tmpl.Engine
	ds     *model.Dataset
	res    *Result
	device string

	// only, if set, limits defaults() to the columns it allows.
	only func(t *schema.Table, c *schema.Column) bool

	// While a derive runs: the work rows and cells it has looked at so far.
	touched map[*model.Row]map[string]bool
	// After the derives: what was used by a derive that made a row.
	used map[*model.Row]map[string]bool
	// Why a derive that looked at a work row made no row.
	failed map[*model.Row]string
	// What the steps of adapt.yaml told.
	notes []Note
	// adapting: the steps of adapt.yaml run; a value may replace another.
	adapting bool
}

type scope struct {
	caps map[string]string // every captured name in sight
	own  map[string]string // the names of the nearest rule above (`parent`)
	row  *model.Row        // the open row
}

// Import reads one configuration text with a pack. With table definitions,
// every value is brought to the one way of writing of its column as it goes
// in, and a line with a value that does not fit stays "unknown" (appendix B
// 3.3). Without them (sc == nil) nothing is checked: for looking into a pack
// that fills tables of its own.
//
// A terminal log is read as the pack's session tells: only the output of the
// command that shows the configuration is parsed, and the other lines keep
// their numbers and are listed as ignored (appendix B 2.4).
func Import(p *pack.Pack, sc *schema.Set, file, text string) (*Result, error) {
	cut := p.Session.Cut(text)
	tree, err := conftree.Parse(p.Syntax, strings.Join(cut.Config, "\n")+"\n", p.Options)
	if err != nil {
		return nil, err
	}
	if len(tree.Lines) != len(cut.Config) {
		return nil, fmt.Errorf("行の数が合いません（%d 行と %d 行）", len(tree.Lines), len(cut.Config))
	}
	x := &run{p: p, sc: sc, file: file, ds: model.NewDataset("old"),
		used: map[*model.Row]map[string]bool{}, failed: map[*model.Row]string{}}
	x.res = &Result{Data: x.ds, Lines: make([]LineState, len(tree.Lines))}
	x.eng = tmpl.New(tmpl.Options{IfName: p.IfName, IfShort: p.IfShort, Lex: p.Lex, Unlex: p.Unlex})
	x.eng.SetFunc("lookup", x.fnLookup)
	x.eng.SetFunc("first", x.fnFirst)
	x.eng.SetFunc("rows", x.fnRows)
	x.eng.SetFunc("covering", x.fnCovering)

	for i := range tree.Lines {
		l := LineState{File: file, Line: i + 1, Text: cut.Shown[i], State: Unknown}
		switch {
		case cut.Skip[i] != "":
			l.State, l.Separator, l.Outside, l.Reason = Ignored, true, true, cut.Skip[i]
		case !tree.InTree[i]:
			l.State, l.Separator, l.Reason = Ignored, true, "区切り（コメント・空行）"
		}
		x.res.Lines[i] = l
	}
	conftree.Walk(tree.Root, func(n *conftree.Node, path []*conftree.Node) {
		if len(path) > 0 {
			x.res.Lines[n.Line-1].Parent = path[len(path)-1].Line
		}
	})
	x.session(cut)
	for _, w := range tree.Warnings {
		x.warnf("%d 行目: %s", w.Line, w.Msg)
	}

	// The host name: the pack's hostname line, or else the file name.
	x.device = strings.TrimSuffix(filepath.Base(file), filepath.Ext(file))
	ended := p.EndMark == nil
	for _, n := range tree.Root.Children {
		norm := conftree.Normalize(n.Text)
		if p.Hostname != nil {
			if m := p.Hostname.FindStringSubmatch(norm); m != nil {
				x.device = m[p.Hostname.SubexpIndex("name")]
			}
		}
		if p.EndMark != nil && p.EndMark.MatchString(norm) {
			ended = true
		}
	}
	x.res.Device = x.device
	if !ended {
		x.warnf("終わりの印の行がありません。文面が途中で切れている疑いがあります")
	}

	root := append(append([]*pack.Rule{}, p.Ignore...), p.Rules...)
	x.apply(tree.Root.Children, root, scope{caps: map[string]string{}, own: map[string]string{}})
	x.derive()
	x.vendor()
	x.kinds()
	x.defaults()

	x.res.Work = model.NewDataset("work")
	for _, name := range x.ds.TableNames() {
		if strings.HasPrefix(name, "x_") {
			x.res.Work.Tables[name] = x.ds.Tables[name]
			x.ds.Drop(name)
		}
	}
	return x.res, nil
}

// session tells what reading the text as a terminal log did.
func (x *run) session(cut *termlog.Cut) {
	if !cut.Log {
		if cut.Prompts >= 2 {
			x.warnf("端末ログのようですが、設定を表示するコマンドの行が見つかりません。全体をコンフィグとして読みました")
		}
		return
	}
	cmd := cut.Command()
	typed := strings.TrimSpace(cut.Shown[cmd-1])
	if cut.First > cut.Last {
		x.warnf("端末ログの %d 行目（%s）の後ろに、設定の行がありません", cmd, typed)
		return
	}
	x.res.Notes = append(x.res.Notes, fmt.Sprintf("端末ログとして読みました。%d 行目（%s）の出力（%d〜%d 行目）を設定として取り込み、ほかの行は「設定の外」にしました",
		cmd, typed, cut.First, cut.Last))
	if n := len(cut.Commands); n > 1 {
		x.warnf("設定を表示するコマンドが %d 回あります（%s）。最後のもの（%d 行目）の出力を使いました", n, lineList(cut.Commands), cmd)
	}
	if len(cut.Loose) > 0 {
		x.warnf("ページ送りの跡と設定が同じ行にありました（%s）。跡を取り除き、字下げは同じログのほかの行に合わせました。元と違うおそれがあります（ページ送りを止めて取り直すと確実です）",
			lineList(cut.Loose))
	}
}

// lineList writes line numbers for a message: the first ten, then how many
// more there are.
func lineList(lines []int) string {
	const most = 10
	parts := make([]string, 0, most)
	for i, n := range lines {
		if i == most {
			break
		}
		parts = append(parts, strconv.Itoa(n))
	}
	s := strings.Join(parts, "・") + " 行目"
	if len(lines) > most {
		s += fmt.Sprintf(" ほか %d 行", len(lines)-most)
	}
	return s
}

// later tells which value a cell keeps after a second one came. The value of
// a secret column is not shown: warnings go into reports.
func (x *run) later(table, col, val string) string {
	if x.sc != nil {
		if t := x.sc.Table(table); t != nil {
			if c := t.Column(col); c != nil && c.Secret {
				return "後の値を使います"
			}
		}
	}
	return "後の値「" + val + "」を使います"
}

func (x *run) warnf(format string, a ...any) {
	x.res.Warnings = append(x.res.Warnings, fmt.Sprintf(format, a...))
}

func (x *run) mark(n *conftree.Node, state, ruleID, reason string, refs []Ref) {
	l := &x.res.Lines[n.Line-1]
	l.State, l.RuleID, l.Reason, l.Refs = state, ruleID, reason, refs
}

func (x *run) markTree(nodes []*conftree.Node, state, ruleID, reason string) {
	for _, n := range nodes {
		x.mark(n, state, ruleID, reason, nil)
		x.markTree(n.Children, state, ruleID, reason)
	}
}

// unknown marks a line and everything below it as not taken.
func (x *run) unknown(n *conftree.Node, ruleID, reason string) {
	x.mark(n, Unknown, ruleID, reason, nil)
	x.markTree(n.Children, Unknown, "", "親の行が未対応")
}

// apply tries the rules on each node; the first rule that hits is used.
func (x *run) apply(nodes []*conftree.Node, rules []*pack.Rule, sc scope) {
	for _, n := range nodes {
		norm := conftree.Normalize(n.Text)
		hit := false
		for _, r := range rules {
			m := r.Match.FindStringSubmatch(norm)
			if m == nil {
				continue
			}
			own := map[string]string{}
			caps := make(map[string]string, len(sc.caps)+len(m))
			for k, v := range sc.caps {
				caps[k] = v
			}
			for i, name := range r.Match.SubexpNames() {
				if name != "" {
					own[name] = m[i]
					caps[name] = m[i]
				}
			}
			// A borrowed rule (children_from) may name a capture that is in sight
			// only where the rule is written: it reads as "" here.
			for _, name := range r.Scope {
				if _, ok := caps[name]; !ok {
					caps[name] = ""
				}
			}
			vars := x.vars(n, caps, sc.own, sc.row)
			if r.When != "" {
				v, err := x.eng.Eval(r.When, vars)
				if err != nil {
					x.unknown(n, r.ID, fmt.Sprintf("規則 %s の when（%s）: %v", r.ID, r.When, err))
					hit = true
					break
				}
				if !tmpl.Truth(v) {
					continue
				}
			}
			hit = true
			x.fire(n, r, sc, caps, own, vars)
			break
		}
		if !hit {
			x.unknown(n, "", "")
		}
	}
}

func (x *run) vars(n *conftree.Node, caps, parent map[string]string, row *model.Row) map[string]any {
	vars := make(map[string]any, len(caps)+5)
	for k, v := range caps {
		vars[k] = v
	}
	vars["parent"] = parent
	vars["device"] = x.device
	vars["line"] = map[string]any{"number": n.Line, "text": n.Text}
	vars["body"] = n.BodyText
	vars["row"] = rowRec{row}
	return vars
}

// rowRec lets expressions read a row: row.name. An empty column reads as "".
type rowRec struct{ r *model.Row }

func (o rowRec) Field(name string) (any, bool) {
	if o.r == nil {
		return "", true
	}
	return cellValue(o.r.Cells[name]), true
}

func (o rowRec) Fields() []string {
	if o.r == nil {
		return nil
	}
	return o.r.Columns()
}

func cellValue(c *model.Cell) any {
	if c == nil {
		return ""
	}
	if c.List {
		return append([]string{}, c.Values...)
	}
	return c.Text()
}

// write is one value waiting to go into a row.
type write struct {
	emit   int // index into the emitted rows, -1 if not one
	col    string
	vals   []string // one value, or the items of an `append`
	append bool
	// replace drops what the list held before the items are added (the
	// "replace" of adapt.yaml).
	replace bool
}

// fire carries out a rule that hit a line. Every value is worked out before
// anything is written, so a line that fails leaves no half-made row behind.
func (x *run) fire(n *conftree.Node, r *pack.Rule, sc scope, caps, own map[string]string, vars map[string]any) {
	child := scope{caps: caps, own: own, row: sc.row}
	if r.Ignore != "" {
		x.mark(n, Ignored, r.ID, r.Ignore, nil)
		x.below(n, r, child)
		return
	}
	fail := func(err error) {
		x.unknown(n, r.ID, fmt.Sprintf("規則 %s: %v", r.ID, err))
	}
	o := model.Origin{Kind: "config", File: x.file, Line: n.Line}

	// The row this rule opens. It is not put into the data yet.
	var key []model.KV
	target := sc.row
	if r.Row != nil {
		var err error
		if key, err = x.key(r.Row.Table, r.Row.Key, vars); err != nil {
			fail(err)
			return
		}
		target = x.peek(r.Row.Table, key)
		vars["row"] = rowRec{target}
	}

	var writes []write
	collect := func(emit int, into *model.Row, set, app []model.KV) error {
		for i, kv := range append(append([]model.KV{}, set...), app...) {
			col, err := x.eng.Expand(kv.Name, vars)
			if err != nil {
				return fmt.Errorf("列の名前（%s）: %w", kv.Name, err)
			}
			val, err := x.eng.Expand(kv.Value, vars)
			if err != nil {
				return fmt.Errorf("%s（%s）: %w", col, kv.Value, err)
			}
			if val == "" {
				continue // an empty value is not written
			}
			vals, err := x.values(into, col, val, i >= len(set))
			if err != nil {
				return err
			}
			writes = append(writes, write{emit: emit, col: col, vals: vals, append: i >= len(set)})
		}
		return nil
	}
	if len(r.Set)+len(r.Append) > 0 && target == nil {
		fail(fmt.Errorf("値を入れる行がありません（row がありません）"))
		return
	}
	if err := collect(-1, target, r.Set, r.Append); err != nil {
		fail(err)
		return
	}
	emitKeys := make([][]model.KV, len(r.Emit))
	for i, e := range r.Emit {
		k, err := x.key(e.Table, e.Key, vars)
		if err != nil {
			fail(err)
			return
		}
		emitKeys[i] = k
		if err := collect(i, x.peek(e.Table, k), e.Set, e.Append); err != nil {
			fail(err)
			return
		}
	}

	// Nothing failed: write.
	var refs []Ref
	if r.Row != nil {
		target = x.ds.Open(r.Row.Table, key, o)
		refs = append(refs, Ref{target.Table, target.KeyText(), ""})
	}
	emitted := make([]*model.Row, len(r.Emit))
	for i, e := range r.Emit {
		emitted[i] = x.ds.Open(e.Table, emitKeys[i], o)
		refs = append(refs, Ref{e.Table, emitted[i].KeyText(), ""})
	}
	for _, w := range writes {
		row := target
		if w.emit >= 0 {
			row = emitted[w.emit]
		}
		if w.append {
			for _, item := range w.vals {
				row.Append(w.col, item, o)
			}
		} else if row.Set(w.col, w.vals[0], o) {
			x.warnf("%d 行目: %s の %s に 2 回目の値が入りました（%s）", n.Line, row.Table, w.col, x.later(row.Table, w.col, w.vals[0]))
		}
		refs = append(refs, Ref{row.Table, row.KeyText(), w.col})
	}
	x.mark(n, Taken, r.ID, "", refs)
	child.row = target
	x.below(n, r, child)
}

// below deals with the lines under a line that a rule hit.
func (x *run) below(n *conftree.Node, r *pack.Rule, sc scope) {
	self := x.res.Lines[n.Line-1]
	switch {
	case len(n.Children) == 0:
	case n.Block != "":
		// The body of a block goes with its first line (the rule reads `body`).
		for _, c := range n.Children {
			x.mark(c, self.State, r.ID, self.Reason, self.Refs)
		}
	case r.IgnoreChildren:
		reason := r.Ignore
		if reason == "" {
			reason = "規則 " + r.ID + " が子の行を無視"
		}
		x.markTree(n.Children, Ignored, r.ID, reason)
	case len(r.Children)+len(r.Borrowed) > 0:
		x.apply(n.Children, append(append([]*pack.Rule{}, r.Children...), r.Borrowed...), sc)
	default:
		for _, c := range n.Children {
			x.unknown(c, "", "親の行の規則 "+r.ID+" に、子の行の規則がありません")
		}
	}
}

// items cuts the value of an `append` into the items of the list: one line
// can carry several ("dns-server 192.0.2.53 192.0.2.54").
func items(val string) []string {
	return strings.FieldsFunc(val, func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
}

// key works out the key of a row; the device comes first by itself. For a
// table of the design data the key is made whole: the key columns of the
// definition in their order, "" for one the rule does not name.
func (x *run) key(table string, spec []model.KV, vars map[string]any) ([]model.KV, error) {
	devcol := "device"
	if table == "devices" {
		devcol = "hostname"
	}
	key := []model.KV{{Name: devcol, Value: x.device}}
	for _, kv := range spec {
		if kv.Name == devcol {
			continue
		}
		v, err := x.eng.Expand(kv.Value, vars)
		if err != nil {
			return nil, fmt.Errorf("キー %s（%s）: %w", kv.Name, kv.Value, err)
		}
		key = append(key, model.KV{Name: kv.Name, Value: v})
	}
	t, err := x.table(table)
	if t == nil {
		return key, err
	}
	given := map[string]string{}
	for _, kv := range key {
		if !t.IsKey(kv.Name) {
			return nil, fmt.Errorf("%s は表 %s のキーの列ではありません（キーは %s）", kv.Name, table, strings.Join(t.Key, ", "))
		}
		given[kv.Name] = kv.Value
	}
	whole := make([]model.KV, len(t.Key))
	for i, name := range t.Key {
		v, err := x.one(t, t.Column(name), given[name])
		if err != nil {
			return nil, err
		}
		whole[i] = model.KV{Name: name, Value: v}
	}
	return whole, nil
}

// table gives the definition of a table of the design data. A work table
// (x_…) has none; so has every table when no definitions were given.
func (x *run) table(name string) (*schema.Table, error) {
	if x.sc == nil || strings.HasPrefix(name, "x_") {
		return nil, nil
	}
	t := x.sc.Table(name)
	if t == nil {
		return nil, fmt.Errorf("表 %s は設計データにありません", name)
	}
	return t, nil
}

// one brings one value to the way of writing of its column.
func (x *run) one(t *schema.Table, c *schema.Column, val string) (string, error) {
	if c.Type == "ifname" && val != "" {
		name, err := x.p.IfName(val)
		if err != nil {
			return "", fmt.Errorf("%s の %s: %w", t.Name, c.Name, err)
		}
		val = name
	}
	v, err := c.Normalize(val)
	if err != nil {
		return "", fmt.Errorf("%s の %s: %w", t.Name, c.Name, err)
	}
	return v, nil
}

// values checks a value that a rule is about to put into a row, and gives it
// in the way of writing of the column: one value for `set`, the items for
// `append`.
func (x *run) values(row *model.Row, col, val string, app bool) ([]string, error) {
	vals := []string{val}
	if app {
		vals = items(val)
	}
	t, err := x.table(row.Table)
	if t == nil {
		return vals, err
	}
	c := t.Column(col)
	switch {
	case c == nil:
		return nil, fmt.Errorf("表 %s に列 %s はありません", t.Name, col)
	case c.List && !app:
		return nil, fmt.Errorf("%s の %s は値を並べる列です。規則では append で入れます", t.Name, col)
	case !c.List && app:
		return nil, fmt.Errorf("%s の %s は値が 1 つの列です。規則では set で入れます", t.Name, col)
	}
	for i, v := range vals {
		if vals[i], err = x.one(t, c, v); err != nil {
			return nil, err
		}
	}
	if t.IsKey(col) && vals[0] != row.Get(col) {
		return nil, fmt.Errorf("%s の %s はキーの列です。値は row の key に書きます", t.Name, col)
	}
	return vals, nil
}

// peek gives the row with the key if it is there, or a stand-in that is not
// in the data, so that expressions can read `row` before the line is taken.
func (x *run) peek(table string, key []model.KV) *model.Row {
	if r := x.ds.Tables[table].Find(key); r != nil {
		return r
	}
	r := &model.Row{Table: table, Key: key, Cells: map[string]*model.Cell{}}
	for _, k := range key {
		r.Cells[k.Name] = &model.Cell{Values: []string{k.Value}}
	}
	return r
}

// ---- derive (appendix B 3.6) ----

// workRec lets a derive read a work row and remembers what it read.
type workRec struct {
	r *model.Row
	x *run
}

func (o workRec) Field(name string) (any, bool) {
	o.x.touch(o.r, name)
	return cellValue(o.r.Cells[name]), true
}

func (o workRec) Fields() []string { return o.r.Columns() }

func (x *run) touch(r *model.Row, col string) {
	if x.touched == nil {
		return
	}
	if x.touched[r] == nil {
		x.touched[r] = map[string]bool{}
	}
	if col != "" {
		x.touched[r][col] = true
	}
}

func (x *run) match(args []any, kwargs map[string]any) ([]*model.Row, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("表の名前を 1 つ書いてください")
	}
	name, _ := args[0].(string)
	t := x.ds.Tables[name]
	if t == nil {
		return nil, nil
	}
	cols := make([]string, 0, len(kwargs))
	for c := range kwargs {
		if c != "sort" {
			cols = append(cols, c)
		}
	}
	sort.Strings(cols)
	// The rows of the design data hold their values in the way of writing of
	// the column: what is looked for is brought to it too.
	want := map[string]string{}
	isList := map[string]bool{}
	def, _ := x.table(name)
	for _, c := range cols {
		if kwargs[c] == nil {
			continue
		}
		want[c] = fmt.Sprint(kwargs[c])
		if col := def.Column(c); col != nil {
			isList[c] = col.List
			if v, err := x.one(def, col, want[c]); err == nil {
				want[c] = v
			}
		}
	}
	var out []*model.Row
	for _, r := range t.Rows {
		ok := true
		for _, c := range cols {
			if want[c] == "" {
				ok = false
				break
			}
			// A column that holds a list is hit by a row that has the value
			// among its items (appendix B 3.6).
			if isList[c] {
				has := false
				if cell := r.Cells[c]; cell != nil {
					for _, v := range cell.Values {
						has = has || v == want[c]
					}
				}
				if !has {
					ok = false
					break
				}
				continue
			}
			if r.Get(c) != want[c] {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, r)
		}
	}
	if by, _ := kwargs["sort"].(string); by != "" {
		sort.SliceStable(out, func(i, j int) bool { return lessNum(out[i].Get(by), out[j].Get(by)) })
	}
	for _, r := range out {
		for _, c := range cols {
			x.touch(r, c)
		}
	}
	return out, nil
}

func lessNum(a, b string) bool {
	x, e1 := strconv.Atoi(a)
	y, e2 := strconv.Atoi(b)
	if e1 == nil && e2 == nil {
		return x < y
	}
	return a < b
}

func (x *run) fnLookup(args []any, kwargs map[string]any) (any, error) {
	rows, err := x.match(args, kwargs)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	if len(rows) > 1 {
		x.warnf("lookup(%v): %d 行が当たりました。最初の 1 行を使います", args[0], len(rows))
	}
	x.touch(rows[0], "")
	return workRec{rows[0], x}, nil
}

func (x *run) fnFirst(args []any, kwargs map[string]any) (any, error) {
	rows, err := x.match(args, kwargs)
	if err != nil || len(rows) == 0 {
		return nil, err
	}
	x.touch(rows[0], "")
	if by, _ := kwargs["sort"].(string); by != "" {
		x.touch(rows[0], by)
	}
	return workRec{rows[0], x}, nil
}

func (x *run) fnRows(args []any, kwargs map[string]any) (any, error) {
	rows, err := x.match(args, kwargs)
	if err != nil {
		return nil, err
	}
	out := make([]any, len(rows))
	for i, r := range rows {
		x.touch(r, "")
		out[i] = workRec{r, x}
	}
	return out, nil
}

// fnCovering gives the row whose column holds the most specific network that
// has the address inside: covering('x_ospf_networks', 'prefix', i.ipv4).
func (x *run) fnCovering(args []any, _ map[string]any) (any, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("covering には、表・列・アドレスの 3 つを書いてください")
	}
	name, _ := args[0].(string)
	col, _ := args[1].(string)
	addr := fmt.Sprint(args[2])
	t := x.ds.Tables[name]
	if t == nil || args[2] == nil || addr == "" {
		return nil, nil
	}
	if _, err := netx.IP(addr); err != nil {
		return nil, nil // "dhcp", "negotiated" …: no address to compare
	}
	var best *model.Row
	bestLen := -1
	for _, r := range t.Rows {
		net := r.Get(col)
		if ok, err := netx.InNet(addr, net); err != nil || !ok {
			continue
		}
		if n, _ := netx.PrefixLen(net); n > bestLen {
			best, bestLen = r, n
		}
	}
	if best == nil {
		return nil, nil
	}
	x.touch(best, col)
	return workRec{best, x}, nil
}

func (x *run) derive() {
	for _, d := range x.p.Derives {
		t := x.ds.Tables[d.For]
		if t == nil || !x.onlyIf(d) {
			continue
		}
		for _, src := range append([]*model.Row{}, t.Rows...) {
			x.touched = map[*model.Row]map[string]bool{}
			x.touch(src, "")
			err := x.deriveOne(d, src)
			if err != nil {
				x.warnf("derive %s（%s の %s）: %v", d.ID, d.For, src.KeyText(), err)
				for r := range x.touched {
					x.failed[r] = fmt.Sprintf("まとめる規則 %s が行を作れませんでした: %v", d.ID, err)
				}
			}
			x.touched = nil
		}
	}

	// A work row, or a value of it, that no derive used goes back to "unknown":
	// being parked in a work table is not being taken.
	for _, name := range x.ds.TableNames() {
		if !strings.HasPrefix(name, "x_") {
			continue
		}
		for _, r := range x.ds.Tables[name].Rows {
			cols, ok := x.used[r]
			if !ok {
				lines := originLines(r.Origins)
				for _, c := range r.Cells {
					lines = append(lines, originLines(c.Origins)...)
				}
				reason := x.failed[r]
				if reason == "" {
					reason = fmt.Sprintf("作業用の表 %s に入りましたが、まとめる規則（derive）に使われませんでした", name)
				}
				x.back(lines, reason)
				continue
			}
			isKey := map[string]bool{}
			for _, k := range r.Key {
				isKey[k.Name] = true
			}
			for _, col := range r.Columns() {
				if !isKey[col] && !cols[col] {
					x.back(originLines(r.Cells[col].Origins), fmt.Sprintf("作業用の表 %s の %s に入りましたが、まとめる規則（derive）がこの値を使っていません", name, col))
				}
			}
		}
	}
}

func originLines(os []model.Origin) []int {
	var out []int
	for _, o := range os {
		if o.Line > 0 {
			out = append(out, o.Line)
		}
	}
	return out
}

// ref notes on a line a cell that a derive rule filled from it, so that the
// line points at the design data and not only at the work table in between.
func (x *run) ref(line int, row *model.Row, col string) {
	if line < 1 || line > len(x.res.Lines) || strings.HasPrefix(row.Table, "x_") {
		return
	}
	l := &x.res.Lines[line-1]
	r := Ref{row.Table, row.KeyText(), col}
	for _, have := range l.Refs {
		if have == r {
			return
		}
	}
	l.Refs = append(l.Refs, r)
}

// remove takes out of the design data a row whose settings stand in another
// table (appendix B 3.6). Its lines stay read; they point at the row no more.
func (x *run) remove(r *model.Row) {
	x.ds.Remove(r)
	gone := Ref{Table: r.Table, Key: r.KeyText()}
	for i := range x.res.Lines {
		l := &x.res.Lines[i]
		kept := l.Refs[:0]
		for _, have := range l.Refs {
			if have.Table != gone.Table || have.Key != gone.Key {
				kept = append(kept, have)
			}
		}
		l.Refs = kept
	}
}

// back turns lines back to "unknown".
func (x *run) back(lines []int, reason string) {
	for _, n := range lines {
		l := &x.res.Lines[n-1]
		if l.State == Taken {
			l.State, l.Reason, l.Refs = Unknown, reason, nil
		}
	}
}

func (x *run) deriveOne(d *pack.Derive, src *model.Row) error {
	vars := map[string]any{d.As: workRec{src, x}, "device": x.device}
	let := func(kv model.KV) error {
		v, err := x.eng.Eval(kv.Value, vars)
		if err != nil {
			return fmt.Errorf("let %s（%s）: %w", kv.Name, kv.Value, err)
		}
		if v == nil {
			// So that `prof.transform_set` of a missing row is empty, not an error.
			v = rowRec{}
		}
		vars[kv.Name] = v
		return nil
	}
	// Most rows of a table are not what a step is for: its "when" says no.
	// So the values of "let" that "when" reads are worked out first, and the
	// others only for a row the step takes. What a value of "let" looked at
	// counts only when the step makes a row (touched), so a row the step
	// leaves is the same either way. (With "each" the "when" is asked for
	// every item, after all of "let".)
	if d.When != "" && d.Each == "" && len(d.Let) > 0 {
		need := whenNeeds(d)
		for i, kv := range d.Let {
			if need[i] {
				if err := let(kv); err != nil {
					return err
				}
			}
		}
		v, err := x.eng.Eval(d.When, vars)
		if err != nil {
			return fmt.Errorf("when（%s）: %w", d.When, err)
		}
		if !tmpl.Truth(v) {
			return nil
		}
		for i, kv := range d.Let {
			if !need[i] {
				if err := let(kv); err != nil {
					return err
				}
			}
		}
		x.whenHeld = true
		defer func() { x.whenHeld = false }()
		return x.deriveItem(d, src, vars, false)
	}
	for _, kv := range d.Let {
		if err := let(kv); err != nil {
			return err
		}
	}
	return x.deriveItem(d, src, vars, false)
}

// onlyIf tells whether a step runs for the device at all: its "only_if",
// asked once before the rows (appendix B 3.6). What the expression looks at
// is not counted as used (no row is being made). An expression that cannot
// be worked out lets the step run: its own "when" decides then.
func (x *run) onlyIf(d *pack.Derive) bool {
	if d.OnlyIf == "" {
		return true
	}
	v, err := x.eng.Eval(d.OnlyIf, map[string]any{"device": x.device})
	if err != nil {
		x.warnf("%s: only_if（%s）: %v", d.ID, d.OnlyIf, err)
		return true
	}
	return tmpl.Truth(v)
}

// whenNeeds tells which values of "let" the "when" of a step reads, by
// itself or through another value of "let" (by the names in the text of the
// expressions: a name that only looks like one costs a value worked out too
// early, nothing else). A value that needs one that stands after it is
// taken as it comes, in the order of the step.
func whenNeeds(d *pack.Derive) []bool {
	if v, ok := whenNeed.Load(d); ok {
		return v.([]bool)
	}
	need := make([]bool, len(d.Let))
	names := func(text string, mark func(int)) {
		for i, kv := range d.Let {
			if !need[i] && hasName(text, kv.Name) {
				mark(i)
			}
		}
	}
	var mark func(int)
	mark = func(i int) {
		need[i] = true
		names(d.Let[i].Value, mark)
	}
	names(d.When, mark)
	// A value "when" needs may read one that is not needed and stands
	// before it only through a name: the walk above took those too. What is
	// left is worked out after "when", in the order of the step; one of
	// them never stands before a needed one that reads it.
	whenNeed.Store(d, need)
	return need
}

var whenNeed sync.Map // *pack.Derive → []bool

// hasName tells whether the text holds the name as a word of its own.
func hasName(text, name string) bool {
	for from := 0; ; {
		i := strings.Index(text[from:], name)
		if i < 0 {
			return false
		}
		i += from
		j := i + len(name)
		before := i == 0 || !isWord(text[i-1])
		after := j == len(text) || !isWord(text[j])
		if before && after {
			return true
		}
		from = i + 1
	}
}

func isWord(c byte) bool {
	return c == '_' || c >= '0' && c <= '9' || c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z'
}

// deriveItem is the part of a derive after its "let": once for the row, or,
// with "each", once for every item of the list (each tells that vars holds
// an item).
func (x *run) deriveItem(d *pack.Derive, src *model.Row, vars map[string]any, each bool) error {
	if d.Each != "" && !each {
		v, err := x.eng.Eval(d.Each, vars)
		if err != nil {
			return fmt.Errorf("each %s（%s）: %w", d.EachAs, d.Each, err)
		}
		list, _ := v.([]any)
		for i, item := range list {
			vars[d.EachAs], vars[d.EachAs+"_n"] = item, i+1
			if err := x.deriveItem(d, src, vars, true); err != nil {
				return err
			}
		}
		return nil
	}
	if d.When != "" && !x.whenHeld {
		v, err := x.eng.Eval(d.When, vars)
		if err != nil {
			return fmt.Errorf("when（%s）: %w", d.When, err)
		}
		if !tmpl.Truth(v) {
			return nil
		}
	}
	if d.Note != "" {
		text, err := x.eng.Expand(d.Note, vars)
		if err != nil {
			return fmt.Errorf("note（%s）: %w", d.Note, err)
		}
		// A step that goes over many rows may tell once: an empty text is
		// not told.
		if text = strings.TrimSpace(text); text != "" {
			x.notes = append(x.notes, Note{ID: d.ID, Table: src.Table, Row: src, Text: text})
		}
	}
	if d.Remove {
		x.remove(src)
		return nil
	}
	if d.Row == nil {
		for _, c := range d.Clear {
			delete(src.Cells, c)
		}
		if d.Drop != nil {
			key, err := x.key(d.Drop.Table, d.Drop.Key, vars)
			if err != nil {
				return err
			}
			if t := x.ds.Tables[d.Drop.Table]; t != nil {
				if r := t.Find(key); r != nil {
					x.ds.Remove(r)
				}
			}
		}
		return nil
	}
	key, err := x.key(d.Row.Table, d.Row.Key, vars)
	if err != nil {
		return err
	}
	into := x.peek(d.Row.Table, key)
	vars["row"] = rowRec{into}
	var writes []write
	for i, s := range append(append([]model.KV{}, d.Set...), d.Append...) {
		col, err := x.eng.Expand(s.Name, vars)
		if err != nil {
			return fmt.Errorf("列の名前（%s）: %w", s.Name, err)
		}
		val, err := x.eng.Expand(s.Value, vars)
		if err != nil {
			return fmt.Errorf("%s（%s）: %w", col, s.Value, err)
		}
		if val == "" {
			continue
		}
		vals, err := x.values(into, col, val, i >= len(d.Set))
		if err != nil {
			return err
		}
		writes = append(writes, write{col: col, vals: vals, append: i >= len(d.Set)})
	}
	// A list that is put anew (adapt.yaml): its items are told like those of
	// "append", and an empty value leaves the column empty.
	for _, s := range d.Replace {
		col, err := x.eng.Expand(s.Name, vars)
		if err != nil {
			return fmt.Errorf("列の名前（%s）: %w", s.Name, err)
		}
		val, err := x.eng.Expand(s.Value, vars)
		if err != nil {
			return fmt.Errorf("%s（%s）: %w", col, s.Value, err)
		}
		w := write{col: col, append: true, replace: true}
		if strings.TrimSpace(val) != "" {
			if w.vals, err = x.values(into, col, val, true); err != nil {
				return err
			}
		} else if t, _ := x.table(into.Table); t != nil && (t.Column(col) == nil || !t.Column(col).List) {
			return fmt.Errorf("replace は値を並べる列に書きます（%s の %s）", into.Table, col)
		}
		writes = append(writes, w)
	}

	// The origin of the new values: the line that opened each row the derive
	// looked at, and the lines behind the values it read.
	seen := map[int]bool{}
	var origins []model.Origin
	for r, cols := range x.touched {
		var ls []int
		if len(r.Origins) > 0 {
			ls = originLines(r.Origins[:1])
		}
		for c := range cols {
			if cell := r.Cells[c]; cell != nil {
				ls = append(ls, originLines(cell.Origins)...)
			}
		}
		for _, l := range ls {
			if !seen[l] {
				seen[l] = true
				origins = append(origins, model.Origin{Kind: "derived", File: x.file, Line: l})
			}
		}
	}
	sort.Slice(origins, func(i, j int) bool { return origins[i].Line < origins[j].Line })
	first := model.Origin{Kind: "derived", File: x.file}
	if len(origins) > 0 {
		first = origins[0]
	}

	// An item added to a list keeps one origin: the line that opened the row
	// the derive goes over.
	item := first
	if ls := originLines(src.Origins); len(ls) > 0 {
		item.Line = ls[0]
	}

	row := x.ds.Open(d.Row.Table, key, first)
	for _, w := range writes {
		if w.replace {
			delete(row.Cells, w.col)
		}
		if w.append {
			for _, v := range w.vals {
				row.Append(w.col, v, item)
			}
			x.ref(item.Line, row, w.col)
			continue
		}
		if row.Set(w.col, w.vals[0], first) && !x.adapting {
			x.warnf("derive %s: %s の %s に 2 回目の値が入りました（%s）", d.ID, row.Table, w.col, x.later(row.Table, w.col, w.vals[0]))
		}
		row.Cells[w.col].Origins = origins
		for _, o := range origins {
			x.ref(o.Line, row, w.col)
		}
	}
	for r, cols := range x.touched {
		if x.used[r] == nil {
			x.used[r] = map[string]bool{}
		}
		for c := range cols {
			x.used[r][c] = true
		}
	}
	for _, c := range d.Clear {
		delete(src.Cells, c)
	}
	return nil
}

// Note is what a step of adapt.yaml tells: the pack wrote a setting in another
// way than the design data has it.
type Note struct {
	ID    string
	Table string
	Row   *model.Row // the row the step went over
	Text  string
}

// Adapt rewrites the design data of one device into what the pack can write
// (appendix B 5.7): the steps of adapt.yaml, in their order. ds holds the rows
// of that device and is changed in place. What is wrong in the pack comes
// back as warnings.
func Adapt(p *pack.Pack, sc *schema.Set, ds *model.Dataset, device string) ([]Note, []string) {
	if len(p.Adapts) == 0 {
		return nil, nil
	}
	x := &run{p: p, sc: sc, ds: ds, device: device, res: &Result{}, adapting: true,
		used: map[*model.Row]map[string]bool{}, failed: map[*model.Row]string{}}
	x.eng = tmpl.New(tmpl.Options{IfName: p.IfName, IfShort: p.IfShort, Lex: p.Lex, Unlex: p.Unlex})
	x.eng.SetFunc("lookup", x.fnLookup)
	x.eng.SetFunc("first", x.fnFirst)
	x.eng.SetFunc("rows", x.fnRows)
	x.eng.SetFunc("covering", x.fnCovering)
	for _, d := range p.Adapts {
		t := ds.Tables[d.For]
		if t == nil || !x.onlyIf(d) {
			continue
		}
		for _, src := range append([]*model.Row{}, t.Rows...) {
			if err := x.deriveOne(d, src); err != nil {
				x.warnf("adapt %s（%s の %s）: %v", d.ID, d.For, src.KeyText(), err)
			}
		}
	}
	return x.notes, x.res.Warnings
}

// ---- defaults (appendix B 3.7) ----

// vendor notes the pack that read the text on the row of the device. The row
// is made if no line opened it: every text is one device.
func (x *run) vendor() {
	o := model.Origin{Kind: "default"}
	key := []model.KV{{Name: "hostname", Value: x.device}}
	row := x.ds.Tables["devices"].Find(key)
	if row == nil {
		row = x.ds.Open("devices", key, o)
	}
	row.Set("vendor", x.p.ID, o)
}

// FillDefaults puts into design data what Import puts into the empty columns
// of an imported configuration: the kind of each interface, from its name,
// and the defaults of the pack. With explicitOnly, only the columns the
// definitions mark as explicit get their default (spec 9.6). ds holds the
// rows of one device. What is wrong in the pack comes back as text.
func FillDefaults(p *pack.Pack, sc *schema.Set, ds *model.Dataset, device string, explicitOnly bool) []string {
	x := &run{p: p, sc: sc, ds: ds, device: device, res: &Result{}}
	x.eng = p.Engine()
	if explicitOnly {
		x.only = func(_ *schema.Table, c *schema.Column) bool { return c.Explicit }
	}
	x.kinds()
	x.defaults()
	return x.res.Warnings
}

// fill puts a value that no line gave into an empty cell, in the way of
// writing of the column. A value that does not fit is a fault of the pack: it
// is left out and reported.
func (x *run) fill(r *model.Row, col, val, what string) {
	vals := []string{val}
	if t, _ := x.table(r.Table); t != nil {
		c := t.Column(col)
		if c == nil {
			x.warnf("%s: 表 %s に列 %s はありません", what, r.Table, col)
			return
		}
		if c.List {
			vals = items(val)
		}
		for i, v := range vals {
			n, err := x.one(t, c, v)
			if err != nil {
				x.warnf("%s: %v", what, err)
				return
			}
			vals[i] = n
		}
		if c.List {
			for _, v := range vals {
				r.Append(col, v, model.Origin{Kind: "default"})
			}
			return
		}
	}
	r.Set(col, vals[0], model.Origin{Kind: "default"})
}

// kinds fills the kind of an interface from its name, where no rule set it
// (appendix B 2.3).
func (x *run) kinds() {
	t := x.ds.Tables["interfaces"]
	if t == nil {
		return
	}
	for _, r := range t.Rows {
		if r.Get("kind") != "" {
			continue
		}
		if k := x.p.IfKind(r.Get("name")); k != "" {
			x.fill(r, "kind", k, "IF の種類（pack.yaml の interface_types）")
		}
	}
}

func (x *run) defaults() {
	for _, d := range x.p.Defaults {
		t := x.ds.Tables[d.Table]
		if t == nil {
			continue
		}
		for _, r := range t.Rows {
			if r.Get(d.Column) != "" {
				continue
			}
			if x.only != nil {
				st := x.sc.Table(d.Table)
				if c := st.Column(d.Column); c == nil || !x.only(st, c) {
					continue
				}
			}
			if d.When != "" {
				v, err := x.eng.Eval(d.When, map[string]any{"row": rowRec{r}, "device": x.device})
				if err != nil {
					x.warnf("defaults %s.%s の when（%s）: %v", d.Table, d.Column, d.When, err)
					continue
				}
				if !tmpl.Truth(v) {
					continue
				}
			}
			x.fill(r, d.Column, d.Value, fmt.Sprintf("defaults %s.%s", d.Table, d.Column))
		}
	}
}
