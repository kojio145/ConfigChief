package pack

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
)

// LintResult is what the check of a pack against the table definitions found
// (appendix B 8).
type LintResult struct {
	// Problems are faults of the pack: a line that meets one of them cannot be
	// taken in.
	Problems []string
	// Valueless lists the rules that take a line without putting a value
	// anywhere ("id（file）"). They are allowed in two cases only (appendix B
	// 3.2), so a person compares the list with the README of the pack.
	Valueless []string
}

var (
	reTemplate = regexp.MustCompile(`\{\{.*?\}\}|\{%.*?%\}`)
	reLexKind  = regexp.MustCompile(`\b(?:un)?lex\(\s*['"]([^'"]*)['"]`)
)

// Lint checks the tables, columns, keys and fixed values of a pack against
// the table definitions, and the kinds of words against the lexicon.
func Lint(p *Pack, sc *schema.Set) *LintResult {
	res := &LintResult{}
	seen := map[string]bool{}
	problem := func(format string, a ...any) {
		if s := fmt.Sprintf(format, a...); !seen[s] {
			seen[s] = true
			res.Problems = append(res.Problems, s)
		}
	}

	// table gives the definition of a table; nil for a work table and for a
	// table that is not there (reported).
	table := func(where, name string) *schema.Table {
		if strings.HasPrefix(name, "x_") {
			return nil
		}
		t := sc.Table(name)
		if t == nil {
			problem("%s: 表 %s は設計データにありません", where, name)
		}
		return t
	}
	key := func(where string, t *schema.Table, kvs []model.KV) {
		if t == nil {
			return
		}
		for _, kv := range kvs {
			if kv.Name == t.DeviceColumn() || kv.Name == "device" {
				continue // filled by the tool
			}
			if !t.IsKey(kv.Name) {
				problem("%s: %s は表 %s のキーの列ではありません（キーは %s）", where, kv.Name, t.Name, strings.Join(t.Key, ", "))
				continue
			}
			fixed(problem, where, t, t.Column(kv.Name), kv.Value, false)
		}
	}
	values := func(where string, t *schema.Table, kvs []model.KV, app bool) {
		if t == nil {
			return
		}
		for _, kv := range kvs {
			if reTemplate.MatchString(kv.Name) {
				// A name made by a template ("acl_{{ dir }}"): some column must fit.
				if !anyColumn(t, kv.Name) {
					problem("%s: 表 %s に、%s に当たる列がありません", where, t.Name, kv.Name)
				}
				continue
			}
			c := t.Column(kv.Name)
			switch {
			case c == nil:
				problem("%s: 表 %s に列 %s はありません", where, t.Name, kv.Name)
			case t.IsKey(kv.Name):
				problem("%s: %s の %s はキーの列です。値は row の key に書きます", where, t.Name, kv.Name)
			case c.List && !app:
				problem("%s: %s の %s は値を並べる列です。append で入れます", where, t.Name, kv.Name)
			case !c.List && app:
				problem("%s: %s の %s は値が 1 つの列です。set で入れます", where, t.Name, kv.Name)
			default:
				fixed(problem, where, t, c, kv.Value, app)
			}
		}
	}

	// A borrowed rule is checked once more for each table it is used under.
	type visit struct {
		r     *Rule
		table string
	}
	visited := map[visit]bool{}
	valueless := map[string]bool{}
	var walk func(rs []*Rule, open string)
	walk = func(rs []*Rule, open string) {
		for _, r := range rs {
			if visited[visit{r, open}] {
				continue
			}
			visited[visit{r, open}] = true
			where := "規則 " + r.ID
			now := open
			if r.Row != nil {
				now = r.Row.Table
				key(where, table(where, now), r.Row.Key)
			}
			if len(r.Set)+len(r.Append) > 0 && now != "" {
				t := table(where, now)
				values(where, t, r.Set, false)
				values(where, t, r.Append, true)
			}
			for _, e := range r.Emit {
				t := table(where, e.Table)
				key(where, t, e.Key)
				values(where, t, e.Set, false)
				values(where, t, e.Append, true)
			}
			if r.Ignore == "" && r.Row == nil && len(r.Set)+len(r.Append)+len(r.Emit) == 0 {
				valueless[r.ID+"（"+r.File+"）"] = true
			}
			walk(r.Children, now)
			walk(r.Borrowed, now)
		}
	}
	walk(p.Rules, "")

	for _, d := range p.Derives {
		where := "derive " + d.ID
		if !strings.HasPrefix(d.For, "x_") && sc.Table(d.For) == nil {
			problem("%s: for の表 %s は設計データにありません", where, d.For)
		}
		if d.Remove {
			continue
		}
		t := table(where, d.Row.Table)
		key(where, t, d.Row.Key)
		values(where, t, d.Set, false)
		values(where, t, d.Append, true)
	}

	// The steps of adapt.yaml work on design data only.
	for _, d := range p.Adapts {
		where := "adapt " + d.ID
		ft := sc.Table(d.For)
		if ft == nil {
			problem("%s: for の表 %s は設計データにありません", where, d.For)
			continue
		}
		for _, c := range d.Clear {
			switch {
			case ft.Column(c) == nil:
				problem("%s: clear の列 %s は、表 %s にありません", where, c, d.For)
			case ft.IsKey(c):
				problem("%s: clear でキーの列 %s は空にできません", where, c)
			}
		}
		if d.Drop != nil {
			key(where, table(where, d.Drop.Table), d.Drop.Key)
		}
		if d.Row == nil {
			continue
		}
		t := table(where, d.Row.Table)
		key(where, t, d.Row.Key)
		values(where, t, d.Set, false)
		values(where, t, d.Append, true)
		values(where, t, d.Replace, true)
	}

	for _, d := range p.Defaults {
		where := fmt.Sprintf("defaults %s.%s", d.Table, d.Column)
		t := table(where, d.Table)
		if t == nil {
			continue
		}
		c := t.Column(d.Column)
		if c == nil {
			problem("%s: 表 %s に列 %s はありません", where, d.Table, d.Column)
			continue
		}
		fixed(problem, where, t, c, d.Value, c.List)
	}

	for _, e := range p.exprs() {
		for _, m := range reLexKind.FindAllStringSubmatch(e.text, -1) {
			if _, ok := p.Lexicon[m[1]]; !ok {
				problem("%s: 語の種類「%s」が lexicon.yaml にありません", e.where, m[1])
			}
		}
	}

	for id := range valueless {
		res.Valueless = append(res.Valueless, id)
	}
	sort.Strings(res.Valueless)
	return res
}

// fixed checks a value that is written out in the pack (no template in it)
// against the type of its column.
func fixed(problem func(string, ...any), where string, t *schema.Table, c *schema.Column, val string, list bool) {
	if c == nil || val == "" || reTemplate.MatchString(val) {
		return
	}
	vals := []string{val}
	if list {
		vals = strings.FieldsFunc(val, func(r rune) bool { return r == ',' || unicode.IsSpace(r) })
	}
	for _, v := range vals {
		if _, err := c.Normalize(v); err != nil {
			problem("%s: %s の %s: %v", where, t.Name, c.Name, err)
		}
	}
}

// anyColumn tells whether a column name made by a template can be the name
// of a column: the template parts stand for any letters.
func anyColumn(t *schema.Table, name string) bool {
	var sb strings.Builder
	sb.WriteString("^")
	last := 0
	for _, loc := range reTemplate.FindAllStringIndex(name, -1) {
		sb.WriteString(regexp.QuoteMeta(name[last:loc[0]]) + `\w*`)
		last = loc[1]
	}
	sb.WriteString(regexp.QuoteMeta(name[last:]) + "$")
	re := regexp.MustCompile(sb.String())
	for _, c := range t.Columns {
		if re.MatchString(c.Name) {
			return true
		}
	}
	return false
}

// expr is one expression or template of a pack, with the place it stands in.
type expr struct{ where, text string }

// exprs lists every expression and template of the rules, the derives and
// the defaults.
func (p *Pack) exprs() []expr {
	var out []expr
	add := func(where string, texts ...string) {
		for _, t := range texts {
			if t != "" {
				out = append(out, expr{where, t})
			}
		}
	}
	pairs := func(where string, lists ...[]model.KV) {
		for _, l := range lists {
			for _, kv := range l {
				add(where, kv.Name, kv.Value)
			}
		}
	}
	seen := map[*Rule]bool{}
	var walk func(rs []*Rule)
	walk = func(rs []*Rule) {
		for _, r := range rs {
			if seen[r] {
				continue
			}
			seen[r] = true
			where := "規則 " + r.ID
			add(where, r.When)
			if r.Row != nil {
				pairs(where, r.Row.Key)
			}
			pairs(where, r.Set, r.Append)
			for _, e := range r.Emit {
				pairs(where, e.Key, e.Set, e.Append)
			}
			walk(r.Children)
		}
	}
	walk(p.Rules)
	for _, d := range p.Derives {
		where := "derive " + d.ID
		add(where, d.When)
		add(where, d.OnlyIf)
		if d.Remove {
			pairs(where, d.Let)
			continue
		}
		pairs(where, d.Let, d.Row.Key, d.Set, d.Append)
	}
	for _, d := range p.Defaults {
		add(fmt.Sprintf("defaults %s.%s", d.Table, d.Column), d.When)
	}
	return out
}
