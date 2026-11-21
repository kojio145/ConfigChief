package fill

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kojio145/configchief/internal/listimport"
)

// List is a variable rule read from its list file.
type List struct {
	Name string   // the rule's name (vars/<name>.yaml)
	Vars []string // the variable names as the rule writes them
	// Rows are each device's rows by its folded name (listimport.DeviceKey);
	// Devices the names as the list writes them, in its order.
	Rows    map[string][]listimport.VarRow
	Devices []string
}

func (l *List) varName(name string) (string, bool) {
	for _, v := range l.Vars {
		if strings.EqualFold(v, name) {
			return v, true
		}
	}
	return "", false
}

// Book is the parameter book as a template reads it: the values of each
// device after "*", "@group" and the common values are spread (spec 12.3).
type Book interface {
	// Table gives the internal name of a table named by its internal name or
	// its sheet name.
	Table(name string) (string, bool)
	// Column gives the internal name of a column of a table, named by its
	// internal name or its header.
	Column(table, name string) (string, bool)
	// Has tells whether the book has the device (folded name).
	Has(device string) bool
	// Rows gives a device's rows of a table: column → value.
	Rows(device, table string) []map[string]string
	// Devices are the devices of the book in its order.
	Devices() []string
	// Secret tells a column whose values are not shown in reports.
	Secret(table, column string) bool
}

// Sources are where a template's names are looked up.
type Sources struct {
	Lists []*List
	Book  Book // nil: the case has no book
}

func (s *Sources) list(name string) *List {
	for _, l := range s.Lists {
		if strings.EqualFold(l.Name, name) {
			return l
		}
	}
	return nil
}

// binding is what a name stands for.
type binding struct {
	list    *List
	varName string
	table   string
	column  string
}

func (b binding) book() bool { return b.list == nil }

// Bound is a template with its names looked up.
type Bound struct {
	T   *Template
	Src *Sources
	// each is the source of each {{#each}}: a list, or a table of the book.
	each map[*node]binding
	// refs is what each name stands for, by its {{#each}} node and its
	// folded name.
	refs map[*node]map[string]binding
	// Lists are the variable rules the template uses; UsesBook whether it
	// uses the book.
	Lists    []*List
	UsesBook bool
	// Names are the names the template uses, as written the first time,
	// with what they stand for (for the report of the values).
	Names []NameUse
}

// NameUse is a name the template uses.
type NameUse struct {
	Name   string
	List   string // the rule, or "" for the book
	Var    string
	Table  string
	Column string
}

// Bind looks up every name of a template. All faults are given, each with its
// line.
func Bind(t *Template, src *Sources) (*Bound, []error) {
	b := &Bound{T: t, Src: src, each: map[*node]binding{}, refs: map[*node]map[string]binding{}}
	var errs []error
	usedList := map[*List]bool{}
	known := func() string {
		var names []string
		for _, l := range src.Lists {
			names = append(names, l.Vars...)
		}
		sort.Strings(names)
		if len(names) == 0 {
			return "変数ルールがありません"
		}
		if len(names) > 30 {
			names = append(names[:30], "…")
		}
		return "読み込んだ変数: " + strings.Join(names, "、")
	}
	// resolve finds what a name stands for inside an {{#each}} of source in.
	resolve := func(name string, line int, in *binding) (binding, bool) {
		if i := strings.Index(name, "."); i >= 0 {
			if src.Book == nil {
				errs = append(errs, errAt(line, "「%s」: パラメータブックがありません（{{表.列}} はブックの値です）", name))
				return binding{}, false
			}
			tb, ok := src.Book.Table(name[:i])
			if !ok {
				errs = append(errs, errAt(line, "「%s」: ブックに「%s」という表はありません", name, name[:i]))
				return binding{}, false
			}
			col, ok := src.Book.Column(tb, name[i+1:])
			if !ok {
				errs = append(errs, errAt(line, "「%s」: 表「%s」に「%s」という列はありません", name, name[:i], name[i+1:]))
				return binding{}, false
			}
			return binding{table: tb, column: col}, true
		}
		if in != nil && in.book() && src.Book != nil {
			if col, ok := src.Book.Column(in.table, name); ok {
				return binding{table: in.table, column: col}, true
			}
		}
		var found []binding
		for _, l := range src.Lists {
			if v, ok := l.varName(name); ok {
				found = append(found, binding{list: l, varName: v})
			}
		}
		switch len(found) {
		case 0:
			errs = append(errs, errAt(line, "「%s」という変数はありません（%s）", name, known()))
			return binding{}, false
		case 1:
			return found[0], true
		}
		var rules []string
		for _, f := range found {
			rules = append(rules, f.list.Name)
		}
		errs = append(errs, errAt(line, "変数「%s」が 2 つ以上の変数ルールにあります（%s）。どちらの値か決められません", name, strings.Join(rules, "、")))
		return binding{}, false
	}

	refs, eaches := t.Refs()
	// The source of each {{#each}} first: the names inside depend on it.
	for _, e := range eaches {
		if e.Source != "" {
			if l := src.list(e.Source); l != nil {
				b.each[e.node] = binding{list: l}
				usedList[l] = true
				continue
			}
			if src.Book != nil {
				if tb, ok := src.Book.Table(e.Source); ok {
					b.each[e.node] = binding{table: tb}
					b.UsesBook = true
					continue
				}
			}
			errs = append(errs, errAt(e.Line, "{{#each %s}}: 「%s」という変数ルールもブックの表もありません", e.Source, e.Source))
			continue
		}
		lists := map[*List]bool{}
		var order []*List
		for _, n := range e.Names {
			if strings.Contains(n, ".") {
				continue
			}
			for _, l := range src.Lists {
				if _, ok := l.varName(n); ok && !lists[l] {
					lists[l] = true
					order = append(order, l)
				}
			}
		}
		switch len(order) {
		case 0:
			errs = append(errs, errAt(e.Line, "{{#each}} の間に変数ルールの変数がありません（ブックの表を回すときは {{#each 表}} と書きます）"))
		case 1:
			b.each[e.node] = binding{list: order[0]}
			usedList[order[0]] = true
		default:
			var names []string
			for _, l := range order {
				names = append(names, l.Name)
			}
			errs = append(errs, errAt(e.Line, "{{#each}} の間に 2 つ以上の変数ルールの変数があります（%s）。どの一覧の行を回すか、{{#each 名前}} と書きます", strings.Join(names, "、")))
		}
	}
	nodeOf := map[int]*node{}
	for _, e := range eaches {
		nodeOf[e.Line] = e.node
	}
	seen := map[string]bool{}
	for _, r := range refs {
		var in *binding
		var key *node
		if r.EachLine > 0 {
			key = nodeOf[r.EachLine]
			if eb, ok := b.each[key]; ok {
				in = &eb
			}
		}
		bd, ok := resolve(r.Name, r.Line, in)
		if !ok {
			continue
		}
		if b.refs[key] == nil {
			b.refs[key] = map[string]binding{}
		}
		b.refs[key][strings.ToLower(r.Name)] = bd
		if bd.book() {
			b.UsesBook = true
		} else {
			usedList[bd.list] = true
		}
		if !seen[strings.ToLower(r.Name)] {
			seen[strings.ToLower(r.Name)] = true
			u := NameUse{Name: r.Name, Table: bd.table, Column: bd.column}
			if !bd.book() {
				u.List, u.Var = bd.list.Name, bd.varName
			}
			b.Names = append(b.Names, u)
		}
	}
	for _, l := range src.Lists {
		if usedList[l] {
			b.Lists = append(b.Lists, l)
		}
	}
	if len(errs) == 0 && len(b.Lists) == 0 && !b.UsesBook {
		errs = append(errs, errAt(0, "ひな形に変数もブックの値もありません。どの機器の分を作るか決められません（{{変数}} か {{表.列}} を書きます）"))
	}
	return b, errs
}

// Device is one device a template is filled for.
type Device struct {
	Key  string // folded name
	Name string // as the list or the book writes it
}

// Devices are the devices a template is filled for (spec 9.7.3): those the
// variable rules it uses have rows for, in the order of the rules and their
// lists; or, when it uses only the book, the devices of the book.
func (b *Bound) Devices() []Device {
	var out []Device
	seen := map[string]bool{}
	add := func(name string) {
		k := listimport.DeviceKey(name)
		if k != "" && !seen[k] {
			seen[k] = true
			out = append(out, Device{Key: k, Name: name})
		}
	}
	if len(b.Lists) > 0 {
		for _, l := range b.Lists {
			for _, d := range l.Devices {
				add(d)
			}
		}
		return out
	}
	if b.Src.Book != nil {
		for _, d := range b.Src.Book.Devices() {
			add(d)
		}
	}
	return out
}

// Line is one line of a filled template.
type Line struct {
	Text  string
	TLine int // the line of the template
	// Sync is 1 + the index of the {{#sync}} part the line stands in (0 =
	// none).
	Sync int
}

// Sync is one {{#sync}} part as it was filled.
type Sync struct {
	Words  string // "" = the whole blocks inside
	Indent int    // the columns of the marker's indentation
	Start  int    // the index of the first line inside (= the lines before)
	TLine  int
}

// Output is a template filled for one device.
type Output struct {
	Lines []Line
	Syncs []Sync
	// Left are notes on lines left out because a value was empty.
	Left []string
}

// Text joins the lines.
func (o *Output) Text(crlf bool) string {
	nl := "\n"
	if crlf {
		nl = "\r\n"
	}
	var sb strings.Builder
	for _, l := range o.Lines {
		sb.WriteString(l.Text)
		sb.WriteString(nl)
	}
	return sb.String()
}

// Stop tells why a device is not made.
type Stop struct {
	Line int
	Msg  string
}

func (s *Stop) Error() string {
	if s.Line > 0 {
		return fmt.Sprintf("ひな形の %d 行目: %s", s.Line, s.Msg)
	}
	return s.Msg
}

const maxOutLines = 200000

// ctx is where a filling stands.
type ctx struct {
	devKey  string
	eachKey *node
	list    *List
	row     *listimport.VarRow
	table   string
	bookRow map[string]string
	sync    int
}

// Fill puts one device's values into the template.
func (b *Bound) Fill(device string) (*Output, error) {
	out := &Output{}
	c := ctx{devKey: listimport.DeviceKey(device)}
	if err := b.fill(b.T.nodes, c, out); err != nil {
		return nil, err
	}
	return out, nil
}

func (b *Bound) value(name string, line int, c ctx) (val string, leave bool, err error) {
	bd, ok := b.refs[c.eachKey][strings.ToLower(name)]
	if !ok {
		return "", false, &Stop{line, fmt.Sprintf("「%s」が読めません", name)}
	}
	if bd.book() {
		if b.Src.Book == nil || !b.Src.Book.Has(c.devKey) {
			return "", false, &Stop{line, "パラメータブックにこの機器がありません"}
		}
		if c.table == bd.table && c.bookRow != nil {
			return strings.TrimSpace(c.bookRow[bd.column]), false, nil
		}
		rows := b.Src.Book.Rows(c.devKey, bd.table)
		if len(rows) == 0 {
			return "", false, nil
		}
		return strings.TrimSpace(rows[0][bd.column]), false, nil
	}
	row := c.row
	if c.list != bd.list || row == nil {
		rows := bd.list.Rows[c.devKey]
		if len(rows) == 0 {
			return "", false, &Stop{line, fmt.Sprintf("変数ルール「%s」の一覧に、この機器の行がありません", bd.list.Name)}
		}
		row = &rows[0]
	}
	v := row.Values[bd.varName]
	if v == "" && row.Empty[bd.varName] {
		return "", false, nil
	}
	return v, row.Lines[bd.varName], nil
}

func (b *Bound) fill(ns []*node, c ctx, out *Output) error {
	for _, n := range ns {
		switch n.kind {
		case kText:
			var stop error
			leave := false
			text := valueRe.ReplaceAllStringFunc(n.text, func(s string) string {
				m := valueRe.FindStringSubmatch(s)
				v, lv, err := b.value(m[1], n.line, c)
				if err != nil {
					if stop == nil {
						stop = err
					}
					return ""
				}
				if strings.ContainsAny(v, "\r\n") {
					if stop == nil {
						stop = &Stop{n.line, fmt.Sprintf("「%s」の値に改行が入っています", m[1])}
					}
					return ""
				}
				if v == "" {
					switch {
					case m[2] != "" || lv:
						leave = true
					case stop == nil:
						stop = &Stop{n.line, fmt.Sprintf("「%s」の値が空です%s", m[1], b.where(m[1], c))}
					}
				}
				return v
			})
			if stop != nil {
				return stop
			}
			if leave {
				out.Left = append(out.Left, fmt.Sprintf("ひな形の %d 行目: 値が空なので出しませんでした", n.line))
				continue
			}
			if len(out.Lines) >= maxOutLines {
				return &Stop{n.line, fmt.Sprintf("出す行が多すぎます（%d 行まで）", maxOutLines)}
			}
			out.Lines = append(out.Lines, Line{Text: text, TLine: n.line, Sync: c.sync})
		case kIf:
			ok, err := b.test(n, c)
			if err != nil {
				return err
			}
			body := n.body
			if !ok {
				body = n.els
			}
			if err := b.fill(body, c, out); err != nil {
				return err
			}
		case kSync:
			out.Syncs = append(out.Syncs, Sync{Words: n.text, Indent: width(n.indent), Start: len(out.Lines), TLine: n.line})
			cc := c
			cc.sync = len(out.Syncs)
			if err := b.fill(n.body, cc, out); err != nil {
				return err
			}
		case kEach:
			src := b.each[n]
			cc := c
			cc.eachKey = n
			if src.book() {
				if b.Src.Book == nil || !b.Src.Book.Has(c.devKey) {
					return &Stop{n.line, "パラメータブックにこの機器がありません"}
				}
				cc.table, cc.list, cc.row = src.table, nil, nil
				for _, r := range b.Src.Book.Rows(c.devKey, src.table) {
					cc.bookRow = r
					if err := b.fill(n.body, cc, out); err != nil {
						return err
					}
				}
				continue
			}
			rows := src.list.Rows[c.devKey]
			if len(rows) == 0 {
				return &Stop{n.line, fmt.Sprintf("変数ルール「%s」の一覧に、この機器の行がありません", src.list.Name)}
			}
			cc.list, cc.table, cc.bookRow = src.list, "", nil
			for i := range rows {
				cc.row = &rows[i]
				if err := b.fill(n.body, cc, out); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// where says which line of the list a value came from.
func (b *Bound) where(name string, c ctx) string {
	bd := b.refs[c.eachKey][strings.ToLower(name)]
	if bd.book() {
		return "（パラメータブック）"
	}
	row := c.row
	if c.list != bd.list || row == nil {
		if rows := bd.list.Rows[c.devKey]; len(rows) > 0 {
			row = &rows[0]
		}
	}
	if row == nil {
		return ""
	}
	return fmt.Sprintf("（変数ルール「%s」の一覧の %d 行目）", bd.list.Name, row.Line)
}

// test judges a {{#if}}.
func (b *Bound) test(n *node, c ctx) (bool, error) {
	v, _, err := b.value(n.cond.name, n.line, c)
	if err != nil {
		return false, err
	}
	v = strings.TrimSpace(v)
	switch n.cond.op {
	case "=":
		return strings.EqualFold(v, n.cond.value), nil
	case "!=":
		return !strings.EqualFold(v, n.cond.value), nil
	}
	on := v != ""
	switch strings.ToLower(v) {
	case "false", "no", "0", "無", "なし", "off", "いいえ", "×":
		on = false
	}
	if n.cond.not {
		on = !on
	}
	return on, nil
}

// width counts the columns of leading white space (a tab to the next
// multiple of 8), as conftree does.
func width(ws string) int {
	n := 0
	for _, r := range ws {
		if r == '\t' {
			n += 8 - n%8
		} else {
			n++
		}
	}
	return n
}
