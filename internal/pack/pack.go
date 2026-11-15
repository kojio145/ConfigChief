package pack

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/kojio145/configchief/internal/conftree"
	"github.com/kojio145/configchief/internal/logscan"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/termlog"
)

// Pack is one vendor pack, read and checked. M0 form: the parts that the
// import needs (pack.yaml, ignore.yaml, import/*.yaml, lexicon.yaml,
// defaults.yaml), the templates and errors.yaml. models.yaml joins later.
type Pack struct {
	ID       string
	Title    map[string]string
	Version  string
	Schema   int
	Syntax   string
	Options  conftree.Options
	EndMark  *regexp.Regexp
	Detect   []Detect
	Hostname *regexp.Regexp
	IfTypes  []IfType
	Subif    *regexp.Regexp
	Session  *termlog.Session // nil: the pack does not read terminal logs

	Ignore  []*Rule // ignore.yaml: tried first at the top level
	Rules   []*Rule // import/*.yaml, by file name, then as written
	Derives []*Derive
	// Adapts rewrite the design data of a device into what this vendor can
	// write, before the templates run (appendix B 5.7).
	Adapts   []*Derive
	Lexicon  map[string][]Word
	Defaults []Default
	// Messages is errors.yaml: how the device answers a command it does not
	// take, or takes with a word of its own (appendix B 4.4). The first that
	// matches a line counts.
	Messages []logscan.Message

	Output    Output
	Sections  []Section         // render.yaml; empty: the pack writes no configurations
	Templates map[string]string // templates/*.j2 by file name
	// Words are the runs of letters of the files of the pack and of
	// words.txt, in lower case (words.go).
	Words map[string]bool

	Warnings []string // what the checks of appendix B 8 found but allow
}

// Detect is one line of the scoring that picks the pack for a text.
type Detect struct {
	Match *regexp.Regexp
	Score int
}

// IfType is one kind of interface name.
type IfType struct {
	Name  string
	Short []string
	Kind  string
}

// Rule is one import rule (appendix B 3.2).
type Rule struct {
	ID             string
	File           string
	Match          *regexp.Regexp
	When           string
	Row            *RowSpec
	Set            []model.KV
	Append         []model.KV
	Emit           []Emit
	Children       []*Rule
	From           string  // children_from: the id of the rule whose children are used here too
	Borrowed       []*Rule // the children of From, tried after Children
	Ignore         string
	IgnoreChildren bool

	// Scope holds the captured names in sight where the rule is written: its
	// own and those of the rules above it. Where the rule is borrowed, a name
	// that is not in sight there reads as "".
	Scope  []string
	hasRow bool // a row is open where the rule is written
}

// RowSpec names the row that a rule opens.
type RowSpec struct {
	Table string
	Key   []model.KV
}

// Emit makes a row in another table.
type Emit struct {
	Table  string
	Key    []model.KV
	Set    []model.KV
	Append []model.KV
}

// Derive joins rows of work tables into design data (appendix B 3.6).
type Derive struct {
	ID     string
	File   string
	For    string
	As     string
	Let    []model.KV
	When   string
	Row    *RowSpec
	Set    []model.KV
	Append []model.KV
	// Remove takes the row of For out of the design data where When holds:
	// what it said stands in another table by then. Row is nil.
	Remove bool
	// Clear and Note are for the steps of adapt.yaml (appendix B 5.7): the
	// columns of the row of For to empty, and what to tell the user.
	Clear []string
	Note  string
}

// Word is one entry of the lexicon: the neutral word and the vendor's.
type Word struct {
	Neutral string
	Out     string
	Aliases []string
}

// Default fills an empty column (appendix B 4.3).
type Default struct {
	Table  string
	Column string
	Value  string
	When   string
}

// kvs reads a YAML mapping as ordered pairs; any scalar is kept as its text
// (false → "false", 10 → "10").
type kvs []model.KV

func (k *kvs) UnmarshalYAML(n *yaml.Node) error {
	if n.Kind != yaml.MappingNode {
		return fmt.Errorf("%d 行目: 「名前: 値」の並びで書いてください", n.Line)
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		v := n.Content[i+1]
		if v.Kind != yaml.ScalarNode {
			return fmt.Errorf("%d 行目: %s の値は 1 つの文字列で書いてください", v.Line, n.Content[i].Value)
		}
		val := v.Value
		if v.Tag == "!!null" {
			val = ""
		}
		*k = append(*k, model.KV{Name: n.Content[i].Value, Value: val})
	}
	return nil
}

type rowY struct {
	Table string `yaml:"table"`
	Key   kvs    `yaml:"key"`
}

type emitY struct {
	Table  string `yaml:"table"`
	Key    kvs    `yaml:"key"`
	Set    kvs    `yaml:"set"`
	Append kvs    `yaml:"append"`
}

type ruleY struct {
	ID             string  `yaml:"id"`
	Match          string  `yaml:"match"`
	When           string  `yaml:"when"`
	Row            *rowY   `yaml:"row"`
	Set            kvs     `yaml:"set"`
	Append         kvs     `yaml:"append"`
	Emit           []emitY `yaml:"emit"`
	Children       []ruleY `yaml:"children"`
	ChildrenFrom   string  `yaml:"children_from"`
	Ignore         string  `yaml:"ignore"`
	IgnoreChildren bool    `yaml:"ignore_children"`
}

type deriveY struct {
	ID     string `yaml:"id"`
	For    string `yaml:"for"`
	As     string `yaml:"as"`
	Let    kvs    `yaml:"let"`
	When   string `yaml:"when"`
	Row    *rowY  `yaml:"row"`
	Set    kvs    `yaml:"set"`
	Append kvs    `yaml:"append"`
	Remove bool   `yaml:"remove"`
	// Only in adapt.yaml.
	Clear []string `yaml:"clear"`
	Note  string   `yaml:"note"`
}

type adaptY struct {
	Adapt []deriveY `yaml:"adapt"`
}

type importY struct {
	Rules  []ruleY   `yaml:"rules"`
	Derive []deriveY `yaml:"derive"`
}

type packY struct {
	ID              string            `yaml:"id"`
	Title           map[string]string `yaml:"title"`
	Version         string            `yaml:"version"`
	Schema          int               `yaml:"schema"`
	Syntax          string            `yaml:"syntax"`
	CommentPrefixes []string          `yaml:"comment_prefixes"`
	Output          struct {
		Encoding string `yaml:"encoding"`
		Newline  string `yaml:"newline"`
		Indent   string `yaml:"indent"`
	} `yaml:"output"`
	EndMarker string `yaml:"end_marker"`
	Detect    []struct {
		Match string `yaml:"match"`
		Score int    `yaml:"score"`
	} `yaml:"detect"`
	Blocks []struct {
		ID    string `yaml:"id"`
		Start string `yaml:"start"`
		EndBy string `yaml:"end_by"`
	} `yaml:"blocks"`
	Hostname   string `yaml:"hostname"`
	Interfaces struct {
		Types []struct {
			Name  string   `yaml:"name"`
			Short []string `yaml:"short"`
			Kind  string   `yaml:"kind"`
		} `yaml:"types"`
		Subif string `yaml:"subif"`
	} `yaml:"interfaces"`
	Session struct {
		Prompt     string   `yaml:"prompt"`
		ShowConfig []string `yaml:"show_config"`
		Pager      []string `yaml:"pager"`
	} `yaml:"session"`
}

// reserved are the names a rule's values use for themselves (appendix B 3.4).
var reserved = map[string]bool{"parent": true, "row": true, "device": true, "line": true, "body": true}

type loader struct {
	fsys  fs.FS
	p     *Pack
	errs  []string
	ids   map[string]string
	rules map[string]*Rule // every rule by id, for children_from
	wrote map[string]bool  // work tables some rule fills
}

func (l *loader) errf(format string, a ...any) {
	l.errs = append(l.errs, fmt.Sprintf(format, a...))
}

func (l *loader) warnf(format string, a ...any) {
	l.p.Warnings = append(l.p.Warnings, fmt.Sprintf(format, a...))
}

// read decodes one YAML file strictly: a key the format does not know is an
// error, so a typo like `childern:` cannot go unnoticed.
func (l *loader) read(name string, into any, must bool) bool {
	data, err := fs.ReadFile(l.fsys, name)
	if err != nil {
		if must || !errors.Is(err, fs.ErrNotExist) {
			l.errf("%s: 読めません（%v）", name, err)
		}
		return false
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(into); err != nil && !errors.Is(err, io.EOF) {
		l.errf("%s: %v", name, err)
		return false
	}
	return true
}

func (l *loader) re(where, src string) *regexp.Regexp {
	re, err := regexp.Compile(src)
	if err != nil {
		l.errf("%s: 正規表現「%s」が読めません（%v）", where, src, err)
		return nil
	}
	return re
}

// Load reads a pack from its folder and checks it. All problems are reported
// together in the error.
func Load(fsys fs.FS) (*Pack, error) {
	l := &loader{fsys: fsys, p: &Pack{Lexicon: map[string][]Word{}}, ids: map[string]string{}, rules: map[string]*Rule{}, wrote: map[string]bool{}}
	p := l.p

	var py packY
	if !l.read("pack.yaml", &py, true) {
		return nil, errors.New(strings.Join(l.errs, "\n"))
	}
	p.ID, p.Title, p.Version, p.Schema, p.Syntax = py.ID, py.Title, py.Version, py.Schema, py.Syntax
	if p.ID == "" {
		l.errf("pack.yaml: id がありません")
	}
	if p.Syntax != "indent" && p.Syntax != "flat" {
		l.errf("pack.yaml: syntax「%s」には対応していません", p.Syntax)
	}
	p.Options.CommentPrefixes = py.CommentPrefixes
	if py.EndMarker != "" {
		p.EndMark = l.re("pack.yaml end_marker", py.EndMarker)
	}
	for _, d := range py.Detect {
		if re := l.re("pack.yaml detect", d.Match); re != nil {
			p.Detect = append(p.Detect, Detect{re, d.Score})
		}
	}
	for _, b := range py.Blocks {
		re := l.re("pack.yaml blocks "+b.ID, b.Start)
		if re == nil {
			continue
		}
		if b.EndBy != "delim" || re.SubexpIndex("delim") < 0 {
			l.errf("pack.yaml blocks %s: end_by は delim だけが使えます（start に (?P<delim>…) が要ります）", b.ID)
			continue
		}
		p.Options.Blocks = append(p.Options.Blocks, conftree.Block{ID: b.ID, Start: re, EndBy: b.EndBy})
	}
	if py.Hostname != "" {
		if re := l.re("pack.yaml hostname", py.Hostname); re != nil {
			if re.SubexpIndex("name") < 0 {
				l.errf("pack.yaml hostname: (?P<name>…) が要ります")
			}
			p.Hostname = re
		}
	}
	for i, t := range py.Interfaces.Types {
		// YAML reads a bare Null, on, yes, ... as something other than text.
		if t.Name == "" {
			l.errf("pack.yaml interfaces.types の %d 件目: name が空です（Null・on・yes などの語は引用符で囲みます）", i+1)
			continue
		}
		p.IfTypes = append(p.IfTypes, IfType{t.Name, t.Short, t.Kind})
	}
	if py.Interfaces.Subif != "" {
		p.Subif = l.re("pack.yaml interfaces.subif", py.Interfaces.Subif)
	}
	if s := py.Session; s.Prompt != "" || len(s.ShowConfig) > 0 || len(s.Pager) > 0 {
		ses := &termlog.Session{}
		if s.Prompt == "" {
			l.errf("pack.yaml session: prompt がありません")
		} else {
			ses.Prompt = l.re("pack.yaml session.prompt", s.Prompt)
		}
		if len(s.ShowConfig) == 0 {
			l.errf("pack.yaml session: show_config がありません")
		}
		for _, src := range s.ShowConfig {
			if re := l.re("pack.yaml session.show_config", src); re != nil {
				ses.ShowConfig = append(ses.ShowConfig, re)
			}
		}
		for _, src := range s.Pager {
			re := l.re("pack.yaml session.pager", src)
			if re == nil {
				continue
			}
			if re.MatchString("") {
				l.errf("pack.yaml session.pager: 「%s」は空の文字に当たります（跡そのものを書きます）", src)
				continue
			}
			ses.Pager = append(ses.Pager, re)
		}
		if ses.Prompt != nil {
			p.Session = ses
		}
	}

	// ignore.yaml
	var ig struct {
		Ignore []struct {
			Match  string `yaml:"match"`
			Reason string `yaml:"reason"`
		} `yaml:"ignore"`
	}
	if l.read("ignore.yaml", &ig, false) {
		for i, x := range ig.Ignore {
			id := fmt.Sprintf("ignore.%d", i+1)
			if x.Reason == "" {
				l.errf("ignore.yaml %d 件目: reason がありません", i+1)
			}
			if re := l.re("ignore.yaml "+id, x.Match); re != nil {
				p.Ignore = append(p.Ignore, &Rule{ID: id, File: "ignore.yaml", Match: re, Ignore: x.Reason})
			}
		}
	}

	// import/*.yaml
	files, _ := fs.Glob(fsys, "import/*.yaml")
	sort.Strings(files)
	for _, f := range files {
		var iy importY
		if !l.read(f, &iy, true) {
			continue
		}
		for i := range iy.Rules {
			if r := l.rule(f, &iy.Rules[i], false, nil); r != nil {
				p.Rules = append(p.Rules, r)
			}
		}
		for i := range iy.Derive {
			l.derive(f, &iy.Derive[i])
		}
	}
	l.borrow(p.Rules)
	if _, err := fs.Stat(l.fsys, "adapt.yaml"); err == nil {
		var ay adaptY
		if l.read("adapt.yaml", &ay, true) {
			for i := range ay.Adapt {
				l.adapt("adapt.yaml", &ay.Adapt[i])
			}
		}
	}
	// A work table is used if a derive goes over it, or if any expression
	// names it: lookup('x_…'), first('x_…'), rows('x_…'), covering('x_…', …).
	used := map[string]bool{}
	for _, d := range p.Derives {
		used[d.For] = true
	}
	for _, e := range p.exprs() {
		for _, m := range reWorkRef.FindAllStringSubmatch(e.text, -1) {
			used[m[1]] = true
		}
	}
	var unusedTables []string
	for t := range l.wrote {
		if !used[t] {
			unusedTables = append(unusedTables, t)
		}
	}
	sort.Strings(unusedTables)
	for _, t := range unusedTables {
		l.warnf("作業用の表 %s に入れる規則があるのに、使う derive がありません", t)
	}

	l.lexicon()
	l.defaults()
	l.messages()
	p.Output = Output{py.Output.Encoding, py.Output.Newline, py.Output.Indent}
	switch p.Output.Newline {
	case "", "lf", "crlf":
	default:
		l.errf("pack.yaml output.newline「%s」は使えません（crlf か lf）", p.Output.Newline)
	}
	l.render()

	if len(l.errs) > 0 {
		return nil, errors.New(strings.Join(l.errs, "\n"))
	}
	words, err := LoadWords(fsys)
	if err != nil {
		return nil, fmt.Errorf("パックのファイルを読めません: %v", err)
	}
	p.Words = words
	return p, nil
}

var reWorkRef = regexp.MustCompile(`['"](x_\w+)['"]`)

func toKV(k kvs) []model.KV { return []model.KV(k) }

func (l *loader) rule(file string, y *ruleY, hasRow bool, scope []string) *Rule {
	where := file + " " + y.ID
	if y.ID == "" {
		l.errf("%s: id のない規則があります（match: %s）", file, y.Match)
		return nil
	}
	if prev, dup := l.ids[y.ID]; dup {
		l.errf("%s: 規則の id「%s」が %s と重なっています", file, y.ID, prev)
	}
	l.ids[y.ID] = file
	if y.Match == "" {
		l.errf("%s: match がありません", where)
		return nil
	}
	re := l.re(where, y.Match)
	if re == nil {
		return nil
	}
	if !strings.HasPrefix(y.Match, "^") || !strings.HasSuffix(y.Match, "$") {
		l.warnf("%s: match が ^ で始まり $ で終わっていません（行の一部だけに当たるおそれ）", where)
	}
	for _, n := range re.SubexpNames() {
		if reserved[n] {
			l.errf("%s: 取り出しの名前「%s」は使えません（parent・row・device・line・body は決まった値の名前）", where, n)
		}
	}
	r := &Rule{ID: y.ID, File: file, Match: re, When: y.When, Set: toKV(y.Set), Append: toKV(y.Append), From: y.ChildrenFrom, Ignore: y.Ignore, IgnoreChildren: y.IgnoreChildren}
	r.Scope = append([]string{}, scope...)
	for _, n := range re.SubexpNames() {
		if n != "" && !contains(r.Scope, n) {
			r.Scope = append(r.Scope, n)
		}
	}
	l.rules[y.ID] = r
	if y.Row != nil {
		if y.Row.Table == "" {
			l.errf("%s: row に table がありません", where)
		}
		r.Row = &RowSpec{y.Row.Table, toKV(y.Row.Key)}
		l.table(y.Row.Table)
		hasRow = true
	}
	if (len(r.Set) > 0 || len(r.Append) > 0) && !hasRow {
		l.errf("%s: set・append を使うには、この規則か親の規則に row が要ります", where)
	}
	if r.Ignore != "" && (r.Row != nil || len(r.Set) > 0 || len(r.Append) > 0 || len(y.Emit) > 0) {
		l.errf("%s: ignore と、値を入れる項目（row・set・append・emit）は一緒に書けません", where)
	}
	r.hasRow = hasRow
	if r.IgnoreChildren && (len(y.Children) > 0 || y.ChildrenFrom != "") {
		l.errf("%s: ignore_children と children・children_from は一緒に書けません", where)
	}
	for _, e := range y.Emit {
		if e.Table == "" {
			l.errf("%s: emit に table がありません", where)
		}
		r.Emit = append(r.Emit, Emit{e.Table, toKV(e.Key), toKV(e.Set), toKV(e.Append)})
		l.table(e.Table)
	}
	for i := range y.Children {
		if c := l.rule(file, &y.Children[i], hasRow, r.Scope); c != nil {
			r.Children = append(r.Children, c)
		}
	}
	return r
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// borrow carries out children_from: a rule gets, after its own children, the
// children written under the rule it names (appendix B 3.2).
func (l *loader) borrow(rs []*Rule) {
	for _, r := range rs {
		l.borrow(r.Children)
		if r.From == "" {
			continue
		}
		where := r.File + " " + r.ID
		from := l.rules[r.From]
		switch {
		case r.From == r.ID:
			l.errf("%s: children_from に自分自身は書けません", where)
			continue
		case from == nil:
			l.errf("%s: children_from の規則「%s」がありません", where, r.From)
			continue
		case len(from.Children) == 0:
			l.errf("%s: children_from の規則「%s」に children がありません", where, r.From)
			continue
		}
		r.Borrowed = from.Children
		if !r.hasRow {
			for _, c := range from.Children {
				if c.Row == nil && (len(c.Set) > 0 || len(c.Append) > 0) {
					l.errf("%s: 借りた規則 %s が set・append を使います。この規則か親の規則に row が要ります", where, c.ID)
				}
			}
		}
	}
}

func (l *loader) table(name string) {
	if strings.HasPrefix(name, "x_") {
		l.wrote[name] = true
	}
}

// adapt reads one step of adapt.yaml: a derive over design data that also
// tells what it changed.
func (l *loader) adapt(file string, y *deriveY) {
	where := file + " " + y.ID
	if y.Note == "" {
		l.errf("%s: note がありません（置き換えたことを必ず知らせます）", where)
	}
	if strings.HasPrefix(y.For, "x_") || (y.Row != nil && strings.HasPrefix(y.Row.Table, "x_")) {
		l.errf("%s: adapt は設計データの表だけを扱います", where)
	}
	n := len(l.p.Derives)
	if y.Row == nil && !y.Remove {
		// Only empties columns, or only tells.
		if y.ID == "" || y.For == "" || y.As == "" || y.When == "" {
			l.errf("%s: id・for・as・when が要ります", where)
			return
		}
		if prev, dup := l.ids[y.ID]; dup {
			l.errf("%s: id「%s」が %s と重なっています", file, y.ID, prev)
		}
		l.ids[y.ID] = file
		l.p.Adapts = append(l.p.Adapts, &Derive{ID: y.ID, File: file, For: y.For, As: y.As, Let: toKV(y.Let), When: y.When, Clear: y.Clear, Note: y.Note})
		return
	}
	l.derive(file, y)
	if len(l.p.Derives) > n {
		d := l.p.Derives[len(l.p.Derives)-1]
		l.p.Derives = l.p.Derives[:n]
		d.Clear, d.Note = y.Clear, y.Note
		l.p.Adapts = append(l.p.Adapts, d)
	}
}

func (l *loader) derive(file string, y *deriveY) {
	where := file + " " + y.ID
	if y.ID == "" {
		l.errf("%s: id のない derive があります", file)
		return
	}
	if prev, dup := l.ids[y.ID]; dup {
		l.errf("%s: id「%s」が %s と重なっています", file, y.ID, prev)
	}
	l.ids[y.ID] = file
	if y.For == "" {
		l.errf("%s: for がありません", where)
	}
	if y.As == "" {
		l.errf("%s: as がありません", where)
	}
	if y.Remove {
		if y.Row != nil || len(y.Set) > 0 || len(y.Append) > 0 {
			l.errf("%s: remove の derive に row・set・append は書けません", where)
		}
		if strings.HasPrefix(y.For, "x_") {
			l.errf("%s: remove で外せるのは設計データの表の行だけです", where)
		}
		if y.When == "" {
			l.errf("%s: remove の derive には when が要ります", where)
		}
		l.p.Derives = append(l.p.Derives, &Derive{ID: y.ID, File: file, For: y.For, As: y.As, Let: toKV(y.Let), When: y.When, Remove: true})
		return
	}
	if y.Row == nil || y.Row.Table == "" {
		l.errf("%s: row がありません", where)
		return
	}
	if strings.HasPrefix(y.Row.Table, "x_") {
		l.errf("%s: derive で作業用の表に行を作ることはできません", where)
	}
	l.p.Derives = append(l.p.Derives, &Derive{ID: y.ID, File: file, For: y.For, As: y.As, Let: toKV(y.Let), When: y.When,
		Row: &RowSpec{y.Row.Table, toKV(y.Row.Key)}, Set: toKV(y.Set), Append: toKV(y.Append)})
}

func (l *loader) lexicon() {
	data, err := fs.ReadFile(l.fsys, "lexicon.yaml")
	if err != nil {
		return
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		l.errf("lexicon.yaml: %v", err)
		return
	}
	if len(doc.Content) == 0 {
		return
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		l.errf("lexicon.yaml: 「種類: {中立の語: メーカーの語}」の形で書いてください")
		return
	}
	for i := 0; i+1 < len(root.Content); i += 2 {
		kind, body := root.Content[i].Value, root.Content[i+1]
		if body.Kind != yaml.MappingNode {
			l.errf("lexicon.yaml %s: 「中立の語: メーカーの語」の並びで書いてください", kind)
			continue
		}
		owner := map[string]string{}
		for j := 0; j+1 < len(body.Content); j += 2 {
			w := Word{Neutral: body.Content[j].Value}
			v := body.Content[j+1]
			switch v.Kind {
			case yaml.ScalarNode:
				w.Out = v.Value
			case yaml.MappingNode:
				var full struct {
					Out     string   `yaml:"out"`
					Aliases []string `yaml:"aliases"`
				}
				if err := v.Decode(&full); err != nil {
					l.errf("lexicon.yaml %s.%s: %v", kind, w.Neutral, err)
					continue
				}
				w.Out, w.Aliases = full.Out, full.Aliases
			default:
				l.errf("lexicon.yaml %s.%s: 値が読めません", kind, w.Neutral)
				continue
			}
			for _, vendor := range append([]string{w.Out}, w.Aliases...) {
				if prev, dup := owner[vendor]; dup && prev != w.Neutral {
					l.errf("lexicon.yaml %s: メーカーの語「%s」に中立の語が 2 つ当たります（%s と %s）", kind, vendor, prev, w.Neutral)
				}
				owner[vendor] = w.Neutral
			}
			l.p.Lexicon[kind] = append(l.p.Lexicon[kind], w)
		}
	}
}

func (l *loader) defaults() {
	var dy struct {
		Defaults []struct {
			Table  string    `yaml:"table"`
			Column string    `yaml:"column"`
			Value  yaml.Node `yaml:"value"`
			When   string    `yaml:"when"`
		} `yaml:"defaults"`
	}
	if !l.read("defaults.yaml", &dy, false) {
		return
	}
	for i, d := range dy.Defaults {
		if d.Table == "" || d.Column == "" || d.Value.Kind != yaml.ScalarNode {
			l.errf("defaults.yaml %d 件目: table・column・value（1 つの値）が要ります", i+1)
			continue
		}
		l.p.Defaults = append(l.p.Defaults, Default{d.Table, d.Column, d.Value.Value, d.When})
	}
}

// messages reads errors.yaml: the errors first, then the warnings, then the
// answers the pack cannot tell.
func (l *loader) messages() {
	type msg struct {
		ID    string `yaml:"id"`
		Match string `yaml:"match"`
		Level string `yaml:"level"`
	}
	var ey struct {
		Errors   []msg `yaml:"errors"`
		Warnings []msg `yaml:"warnings"`
		Others   []msg `yaml:"others"`
	}
	if !l.read("errors.yaml", &ey, false) {
		return
	}
	seen := map[string]bool{}
	for _, m := range append(append(append([]msg{}, ey.Errors...), ey.Warnings...), ey.Others...) {
		where := "errors.yaml " + m.ID
		switch {
		case m.ID == "" || m.Match == "":
			l.errf("errors.yaml: id と match が要ります")
			continue
		case seen[m.ID]:
			l.errf("%s: id が重なっています", where)
			continue
		case m.Level != logscan.Rejected && m.Level != logscan.Warning && m.Level != logscan.Unknown:
			l.errf("%s: level は rejected・warning・unknown のどれかです（「%s」）", where, m.Level)
			continue
		}
		seen[m.ID] = true
		re := l.re(where, m.Match)
		if re == nil {
			continue
		}
		if re.MatchString("") {
			l.errf("%s: match が空の文字に当たります", where)
			continue
		}
		l.p.Messages = append(l.p.Messages, logscan.Message{ID: m.ID, Match: re, Level: m.Level})
	}
}

// IfName turns a short interface name into the full one: Gi0/0/0 →
// GigabitEthernet0/0/0. Upper and lower case do not matter. A name the pack
// does not know comes back unchanged.
func (p *Pack) IfName(s string) (string, error) {
	s = strings.TrimSpace(s)
	head, rest := splitIf(s)
	for _, t := range p.IfTypes {
		if strings.EqualFold(head, t.Name) {
			return t.Name + rest, nil
		}
		for _, sh := range t.Short {
			if strings.EqualFold(head, sh) {
				return t.Name + rest, nil
			}
		}
	}
	return s, nil
}

// IfShort turns an interface name into its short form.
func (p *Pack) IfShort(s string) (string, error) {
	full, _ := p.IfName(s)
	head, rest := splitIf(full)
	for _, t := range p.IfTypes {
		if head == t.Name && len(t.Short) > 0 {
			return t.Short[0] + rest, nil
		}
	}
	return full, nil
}

// IfKind gives the kind of an interface from its name ("" if not known).
func (p *Pack) IfKind(s string) string {
	full, _ := p.IfName(s)
	if p.Subif != nil && p.Subif.MatchString(full) {
		return "subif"
	}
	head, _ := splitIf(full)
	for _, t := range p.IfTypes {
		if head == t.Name {
			return t.Kind
		}
	}
	return ""
}

// splitIf cuts an interface name before its first digit, dropping the space
// some outputs put there ("GigabitEthernet 0/0").
func splitIf(s string) (head, rest string) {
	i := strings.IndexAny(s, "0123456789")
	if i < 0 {
		return s, ""
	}
	return strings.TrimSpace(s[:i]), s[i:]
}

// Lex turns a neutral word into the vendor's.
func (p *Pack) Lex(kind, word string) (string, error) {
	words, ok := p.Lexicon[kind]
	if !ok {
		return "", fmt.Errorf("語の種類「%s」が lexicon.yaml にありません", kind)
	}
	for _, w := range words {
		if w.Neutral == word {
			return w.Out, nil
		}
	}
	return "", fmt.Errorf("%s の語「%s」に当たるメーカーの語がありません", kind, word)
}

// Unlex turns the vendor's word into the neutral one.
func (p *Pack) Unlex(kind, word string) (string, error) {
	words, ok := p.Lexicon[kind]
	if !ok {
		return "", fmt.Errorf("語の種類「%s」が lexicon.yaml にありません", kind)
	}
	for _, w := range words {
		if w.Out == word {
			return w.Neutral, nil
		}
		for _, a := range w.Aliases {
			if a == word {
				return w.Neutral, nil
			}
		}
	}
	return "", fmt.Errorf("%s の語「%s」に当たる中立の語がありません", kind, word)
}

// Score adds up the detect lines that hit in the first 500 lines of a text.
// In a terminal log these are the first lines of the configuration.
func (p *Pack) Score(text string) int {
	lines := p.Session.Cut(text).Section()
	if len(lines) > 500 {
		lines = lines[:500]
	}
	total := 0
	for _, d := range p.Detect {
		for _, line := range lines {
			if d.Match.MatchString(conftree.Normalize(line)) {
				total += d.Score
				break
			}
		}
	}
	return total
}

// Writes lists every table.column the rules, derives and defaults fill, with
// the id of the rule. A column whose name is made by a template
// ("acl_{{ dir }}") is listed as written.
func (p *Pack) Writes() []Write {
	var out []Write
	add := func(id, table string, lists ...[]model.KV) {
		for _, l := range lists {
			for _, kv := range l {
				out = append(out, Write{id, table, kv.Name})
			}
		}
	}
	// A borrowed rule is walked once more for each table it is used under.
	type visit struct {
		r     *Rule
		table string
	}
	seen := map[visit]bool{}
	var walk func(rs []*Rule, table string)
	walk = func(rs []*Rule, table string) {
		for _, r := range rs {
			if seen[visit{r, table}] {
				continue
			}
			seen[visit{r, table}] = true
			t := table
			if r.Row != nil {
				t = r.Row.Table
				add(r.ID, t, r.Row.Key)
			}
			add(r.ID, t, r.Set, r.Append)
			for _, e := range r.Emit {
				add(r.ID, e.Table, e.Key, e.Set, e.Append)
			}
			walk(r.Children, t)
			walk(r.Borrowed, t)
		}
	}
	walk(p.Rules, "")
	for _, d := range p.Derives {
		if d.Remove {
			continue
		}
		add(d.ID, d.Row.Table, d.Row.Key, d.Set, d.Append)
	}
	for _, d := range p.Defaults {
		out = append(out, Write{"defaults", d.Table, d.Column})
	}
	return out
}

// Write is one place where a pack fills a column.
type Write struct {
	RuleID string
	Table  string
	Column string
}
