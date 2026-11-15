// Package caserule reads the rules a case has of its own (rules/*.yaml) and
// runs them: lines a generated configuration must or must not have, and
// conditions on the rows of the design data.
//
// Spec: appendix C 4.
package caserule

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
	"github.com/kojio145/configchief/internal/tmpl"
)

// The kinds of rule.
const (
	Require = "require_line"
	Forbid  = "forbid_line"
	Expr    = "expr"
	Unique  = "unique"
	Equal   = "equal_across"
)

// Rule is one rule of a case.
type Rule struct {
	ID       string
	Title    string
	Severity string
	File     string // the file it was read from, for the messages
	Kind     string
	// Devices is what the rule looks at: "*", "@group", a host name or
	// "vendor:<pack>", any of them.
	Devices []string

	// Pattern and Under are the expressions of a rule of lines, as written.
	Pattern, Under string
	line, under    *regexp.Regexp

	// Table, Column, Where and Expr are those of a rule of rows.
	Table, Column, Where, Expr string
	// InDevice makes a rule of unique values look inside each device.
	InDevice bool
}

// CheckID gives the check of the tool the findings of the rule go under.
func (r *Rule) CheckID() string {
	switch r.Kind {
	case Require:
		return "G-610"
	case Forbid:
		return "G-611"
	}
	return "G-612"
}

// OfLines tells whether the rule looks at the generated text (and not at the
// design data).
func (r *Rule) OfLines() bool { return r.Kind == Require || r.Kind == Forbid }

// KindTitle names the kind of a rule.
func (r *Rule) KindTitle() string {
	switch r.Kind {
	case Require:
		return "含むべき行"
	case Forbid:
		return "含んではいけない行"
	case Expr:
		return "行ごとの式"
	case Unique:
		return "値の重複"
	case Equal:
		return "機器の間でそろえる値"
	}
	return r.Kind
}

func (r *Rule) prefix() string { return r.ID + " " + r.Title + ": " }

type columnY struct {
	Table  string `yaml:"table"`
	Column string `yaml:"column"`
	Where  string `yaml:"where"`
	Scope  string `yaml:"scope"`
}

type ruleY struct {
	ID       string   `yaml:"id"`
	Title    string   `yaml:"title"`
	Severity string   `yaml:"severity"`
	Devices  string   `yaml:"devices"`
	Require  *string  `yaml:"require_line"`
	Forbid   *string  `yaml:"forbid_line"`
	Under    *string  `yaml:"under"`
	Table    string   `yaml:"table"`
	Where    string   `yaml:"where"`
	Expr     *string  `yaml:"expr"`
	Unique   *columnY `yaml:"unique"`
	Equal    *columnY `yaml:"equal_across"`
}

type fileY struct {
	Rules []ruleY `yaml:"rules"`
}

var reToolID = regexp.MustCompile(`^[VXMSGLR]-[0-9]+$`)

// Parse reads one file of rules. It gives the rules that are well written
// and, for the others, what is wrong with them: a case with a fault in its
// rules is not checked at all (appendix C 4.1).
func Parse(file string, data []byte, sc *schema.Set, eng *tmpl.Engine) ([]*Rule, []string) {
	var y fileY
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&y); err != nil && !errors.Is(err, io.EOF) {
		return nil, []string{fmt.Sprintf("%s: %v", file, err)}
	}
	var rules []*Rule
	var problems []string
	for i, ry := range y.Rules {
		name := fmt.Sprintf("%s の %d 件目", file, i+1)
		if ry.ID != "" {
			name = fmt.Sprintf("%s の %s", file, ry.ID)
		}
		bad := func(format string, a ...any) {
			problems = append(problems, name+": "+fmt.Sprintf(format, a...))
		}
		n := len(problems)
		r := &Rule{ID: strings.TrimSpace(ry.ID), Title: strings.TrimSpace(ry.Title), Severity: ry.Severity, File: file}
		switch {
		case r.ID == "":
			bad("id が要ります")
		case reToolID.MatchString(r.ID):
			bad("id「%s」は、ツールのチェックと同じ形なので使えません（P-001 のように書きます）", r.ID)
		}
		if r.Title == "" {
			bad("title が要ります")
		}
		switch r.Severity {
		case check.Error, check.Warning, check.Info:
		case "":
			bad("severity が要ります（error、warning、info）")
		default:
			bad("severity「%s」は使えません（error、warning、info）", r.Severity)
		}
		for _, d := range strings.Split(ry.Devices, ",") {
			if d = strings.TrimSpace(d); d != "" {
				r.Devices = append(r.Devices, d)
			}
		}
		if len(r.Devices) == 0 {
			r.Devices = []string{"*"}
		}

		var kinds []string
		if ry.Require != nil {
			kinds = append(kinds, Require)
		}
		if ry.Forbid != nil {
			kinds = append(kinds, Forbid)
		}
		if ry.Expr != nil {
			kinds = append(kinds, Expr)
		}
		if ry.Unique != nil {
			kinds = append(kinds, Unique)
		}
		if ry.Equal != nil {
			kinds = append(kinds, Equal)
		}
		if len(kinds) != 1 {
			bad("種類（require_line、forbid_line、expr、unique、equal_across）を 1 つだけ書きます（今は %d 個）", len(kinds))
			continue
		}
		r.Kind = kinds[0]
		expr := func(what, src string) {
			if strings.TrimSpace(src) == "" {
				return
			}
			if err := eng.Check("", "{{ capture("+src+") }}"); err != nil {
				bad("%s の式を読めません: %v", what, err)
			}
		}
		column := func(table, col string) {
			t := sc.Table(table)
			switch {
			case table == "":
				bad("table が要ります")
			case t == nil:
				bad("表「%s」は、表の定義にありません（内部名で書きます。例: interfaces）", table)
			case col != "" && t.Column(col) == nil:
				bad("表「%s」に、列「%s」はありません（内部名で書きます）", table, col)
			}
		}
		switch r.Kind {
		case Require, Forbid:
			if r.Kind == Require {
				r.Pattern = *ry.Require
			} else {
				r.Pattern = *ry.Forbid
			}
			if ry.Table != "" || ry.Where != "" {
				bad("%s には table と where を書けません", r.Kind)
			}
			if strings.TrimSpace(r.Pattern) == "" {
				bad("%s の正規表現が空です", r.Kind)
			} else if re, err := regexp.Compile(r.Pattern); err != nil {
				bad("%s の正規表現を読めません: %v", r.Kind, err)
			} else {
				r.line = re
			}
			if ry.Under != nil {
				r.Under = *ry.Under
				if strings.TrimSpace(r.Under) == "" {
					bad("under の正規表現が空です")
				} else if re, err := regexp.Compile(r.Under); err != nil {
					bad("under の正規表現を読めません: %v", err)
				} else {
					r.under = re
				}
			}
		case Expr:
			r.Table, r.Where, r.Expr = ry.Table, ry.Where, *ry.Expr
			if ry.Under != nil {
				bad("expr には under を書けません")
			}
			column(r.Table, "")
			if strings.TrimSpace(r.Expr) == "" {
				bad("expr の式が空です")
			}
			expr("expr", r.Expr)
			expr("where", r.Where)
		case Unique, Equal:
			c := ry.Unique
			if r.Kind == Equal {
				c = ry.Equal
			}
			r.Table, r.Column, r.Where = c.Table, c.Column, c.Where
			if ry.Table != "" || ry.Where != "" || ry.Under != nil {
				bad("%s では、table と where は %s の中に書きます（under は書けません）", r.Kind, r.Kind)
			}
			column(r.Table, r.Column)
			if r.Column == "" {
				bad("column が要ります")
			}
			switch {
			case c.Scope == "":
			case c.Scope == "device" && r.Kind == Unique:
				r.InDevice = true
			case r.Kind == Unique:
				bad("scope「%s」は使えません（device だけ。書かなければ案件の中）", c.Scope)
			default:
				bad("equal_across には scope を書けません")
			}
			expr("where", r.Where)
		}
		if len(problems) == n {
			rules = append(rules, r)
		}
	}
	return rules, problems
}

// Doubled tells the IDs that more than one rule has.
func Doubled(rules []*Rule) []string {
	first := map[string]*Rule{}
	var problems []string
	for _, r := range rules {
		if f := first[r.ID]; f != nil {
			problems = append(problems, fmt.Sprintf("%s の %s: 同じ id が %s にもあります", r.File, r.ID, f.File))
			continue
		}
		first[r.ID] = r
	}
	return problems
}

// Targets gives the devices a rule looks at, in the order of the book.
// vendor gives the pack of a device.
func (r *Rule) Targets(ds *model.Dataset, vendor func(host string) string) []string {
	hit := map[string]bool{}
	for _, d := range r.Devices {
		if id, ok := strings.CutPrefix(d, "vendor:"); ok {
			for _, h := range book.Hosts(ds) {
				if vendor(h) == strings.TrimSpace(id) {
					hit[h] = true
				}
			}
			continue
		}
		for _, h := range book.Targets(ds, d) {
			hit[h] = true
		}
	}
	var out []string
	for _, h := range book.Hosts(ds) {
		if hit[h] {
			out = append(out, h)
		}
	}
	return out
}

// Text is the generated configuration of a device, as a rule of lines sees
// it.
type Text struct {
	Lines []string
	// Shown gives a line (from 1) as a report may show it: without the
	// secret values.
	Shown func(line int) string
	// Comment tells whether a line, without its indentation, is a comment.
	Comment func(body string) bool
}

type textLine struct {
	body   string
	indent int
	skip   bool // an empty line or a comment: not looked at
	parent int  // the line it hangs under; -1 = none
}

func layout(t Text) []textLine {
	out := make([]textLine, len(t.Lines))
	var stack []int
	for i, l := range t.Lines {
		body := strings.TrimLeft(l, " \t")
		tl := textLine{body: strings.TrimRight(body, " \t\r"), indent: len(l) - len(body), parent: -1}
		if tl.body == "" {
			tl.skip = true
			out[i] = tl
			continue
		}
		for len(stack) > 0 && out[stack[len(stack)-1]].indent >= tl.indent {
			stack = stack[:len(stack)-1]
		}
		if len(stack) > 0 {
			tl.parent = stack[len(stack)-1]
		}
		tl.skip = t.Comment != nil && t.Comment(tl.body)
		out[i] = tl
		stack = append(stack, i)
	}
	return out
}

// Lines runs a rule of lines on the generated configuration of a device
// (appendix C 4.2).
func (r *Rule) Lines(device string, t Text) []check.Finding {
	lines := layout(t)
	shown := func(i int) string { return strings.TrimSpace(t.Shown(i + 1)) }
	// under[i] is the nearest line above i, i itself included, that the
	// expression of `under` hits; -1 = none.
	under := make([]int, len(lines))
	for i, l := range lines {
		under[i] = -1
		if l.parent >= 0 {
			under[i] = under[l.parent]
		}
		if r.under != nil && !l.skip && r.under.MatchString(l.body) {
			under[i] = i
		}
	}
	inScope := func(i int) bool {
		if r.under == nil {
			return true
		}
		return lines[i].parent >= 0 && under[lines[i].parent] >= 0
	}

	var findings []check.Finding
	seen := map[string]bool{}
	add := func(line int, msg string, ev ...model.KV) {
		ev = append([]model.KV{{Name: "ルール", Value: r.ID}}, ev...)
		f := check.New(r.CheckID(), r.Severity, device, model.Ref{}, r.prefix()+msg, ev...)
		if seen[f.Key] {
			return
		}
		seen[f.Key] = true
		f.Line, f.CaseRule = line, r.ID
		findings = append(findings, f)
	}

	switch r.Kind {
	case Forbid:
		for i, l := range lines {
			if l.skip || !inScope(i) || !r.line.MatchString(l.body) {
				continue
			}
			ev := []model.KV{{Name: "行", Value: shown(i)}}
			if l.parent >= 0 {
				ev = append(ev, model.KV{Name: "上の行", Value: shown(l.parent)})
			}
			add(i+1, "含んではいけない行があります", ev...)
		}
	case Require:
		if r.under == nil {
			for _, l := range lines {
				if !l.skip && r.line.MatchString(l.body) {
					return nil
				}
			}
			add(0, "含むべき行がありません", model.KV{Name: "正規表現", Value: r.Pattern})
			return findings
		}
		// Each line the expression of `under` hits has to have the line
		// somewhere below it.
		has := map[int]bool{}
		for _, l := range lines {
			if l.skip || !r.line.MatchString(l.body) {
				continue
			}
			for p := l.parent; p >= 0; p = lines[p].parent {
				has[p] = true
			}
		}
		for i, l := range lines {
			if l.skip || !r.under.MatchString(l.body) || has[i] {
				continue
			}
			add(i+1, "この行の下に、含むべき行がありません",
				model.KV{Name: "正規表現", Value: r.Pattern}, model.KV{Name: "上の行", Value: shown(i)})
		}
	}
	return findings
}

// Data is the design data a rule of rows looks at.
type Data struct {
	Schema *schema.Set
	// All is the expanded design data of every device.
	All *model.Dataset
	// Engine gives the engine of the pack of a device.
	Engine func(host string) *tmpl.Engine
	// Project is what the expressions see as `project`.
	Project map[string]any

	all   map[string]any
	heads map[string]map[string]bool
	byDev map[string]map[string]any
}

// rowRec hands a row to an expression, each value in the form of its column.
type rowRec struct {
	t     *schema.Table
	row   *model.Row
	heads map[string]bool
}

func (o rowRec) Field(name string) (any, bool) {
	c := o.t.Column(name)
	if c == nil {
		if name == "extra" {
			return extraRec{o.row, o.heads}, true
		}
		return nil, false
	}
	if cell := o.row.Cells[name]; cell != nil {
		return c.Value(cell.Values), true
	}
	return c.Value(nil), true
}

func (o rowRec) Fields() []string {
	out := make([]string, len(o.t.Columns))
	for i, c := range o.t.Columns {
		out[i] = c.Name
	}
	sort.Strings(out)
	return out
}

// extraRec is the free columns of a row.
type extraRec struct {
	row   *model.Row
	heads map[string]bool
}

func (o extraRec) Field(name string) (any, bool) {
	for _, kv := range o.row.Extra {
		if kv.Name == name {
			return kv.Value, true
		}
	}
	return "", o.heads[name]
}

func (o extraRec) Fields() []string {
	var out []string
	for h := range o.heads {
		out = append(out, h)
	}
	sort.Strings(out)
	return out
}

func (d *Data) prepare() {
	if d.all != nil {
		return
	}
	d.all, d.heads, d.byDev = map[string]any{}, map[string]map[string]bool{}, map[string]map[string]any{}
	for _, t := range d.Schema.Tables {
		h := map[string]bool{}
		rows := []any{}
		if tab := d.All.Tables[t.Name]; tab != nil {
			for _, row := range tab.Rows {
				for _, kv := range row.Extra {
					h[kv.Name] = true
				}
			}
			for _, row := range tab.Rows {
				rows = append(rows, rowRec{t, row, h})
			}
		}
		d.heads[t.Name] = h
		d.all[t.Name] = rows
	}
}

// vars gives what an expression sees for a row of a device.
func (d *Data) vars(host string) map[string]any {
	d.prepare()
	if v := d.byDev[host]; v != nil {
		return v
	}
	mine := map[string]any{}
	var dev any = map[string]any{}
	for _, t := range d.Schema.Tables {
		col := t.DeviceColumn()
		rows := []*model.Row{}
		if tab := d.All.Tables[t.Name]; tab != nil && col != "" {
			for _, row := range tab.Rows {
				if row.Get(col) == host {
					rows = append(rows, row)
				}
			}
		}
		sort.SliceStable(rows, func(i, j int) bool { return keyLess(rows[i], rows[j]) })
		recs := make([]any, len(rows))
		for i, row := range rows {
			recs[i] = rowRec{t, row, d.heads[t.Name]}
		}
		mine[t.Name] = recs
		if t.Name == "devices" && len(recs) > 0 {
			dev = recs[0]
		}
	}
	project := d.Project
	if project == nil {
		project = map[string]any{}
	}
	v := map[string]any{"device": dev, "t": mine, "all": d.all, "project": project}
	d.byDev[host] = v
	return v
}

func keyLess(a, b *model.Row) bool {
	for i := range a.Key {
		if i >= len(b.Key) {
			return false
		}
		if a.Key[i].Value != b.Key[i].Value {
			return tmpl.NaturalLess(a.Key[i].Value, b.Key[i].Value)
		}
	}
	return false
}

// eval gives the value of an expression for a row.
func (d *Data) eval(src, host string, t *schema.Table, row *model.Row) (any, error) {
	base := d.vars(host)
	vars := make(map[string]any, len(base)+1)
	for k, v := range base {
		vars[k] = v
	}
	vars["row"] = rowRec{t, row, d.heads[t.Name]}
	return d.Engine(host).Eval(src, vars)
}

// label names a row by its device and its key, as a finding may tell it.
func label(t *schema.Table, row *model.Row) string {
	var parts []string
	for _, k := range row.Key {
		v := k.Value
		if v == "" {
			continue
		}
		if c := t.Column(k.Name); c != nil && c.Secret {
			v = check.Seal(v)
		}
		parts = append(parts, v)
	}
	return strings.Join(parts, " ")
}

func shownValue(c *schema.Column, v string) string {
	if c.Secret {
		return check.Seal(v)
	}
	return v
}

// Rows runs a rule of rows on the design data of the devices given (those
// the rule is for, without the ones whose values are wrong). look is told
// what the rule looked at, by device (appendix C 4.3).
func (r *Rule) Rows(d *Data, hosts []string, look func(host string, n int)) []check.Finding {
	t := d.Schema.Table(r.Table)
	tab := d.All.Tables[r.Table]
	devCol := t.DeviceColumn()
	in := map[string]bool{}
	for _, h := range hosts {
		in[h] = true
	}
	var findings []check.Finding
	add := func(sev, host string, ref model.Ref, at, msg string, ev ...model.KV) {
		ev = append([]model.KV{{Name: "ルール", Value: r.ID}}, ev...)
		f := check.New("G-612", sev, host, ref, r.prefix()+msg, ev...)
		f.At, f.CaseRule = at, r.ID
		findings = append(findings, f)
	}
	// failed tells that an expression could not be run: an error whatever
	// the rule says, and the rule looks no further.
	failed := func(what, src, host string, row *model.Row, err error) []check.Finding {
		add(check.Error, host, check.RefOf(t, row, ""), check.CellAt(row, ""),
			fmt.Sprintf("%s の式を調べられません: %v", what, err), model.KV{Name: "式", Value: src})
		findings[len(findings)-1].Firm = true
		return findings
	}
	// rows gives the rows the rule looks at, after `where`.
	type picked struct {
		host string
		row  *model.Row
	}
	var rows []picked
	if tab != nil && devCol != "" {
		for _, row := range tab.Rows {
			host := row.Get(devCol)
			if !in[host] {
				continue
			}
			if strings.TrimSpace(r.Where) != "" {
				v, err := d.eval(r.Where, host, t, row)
				if err != nil {
					return failed("where", r.Where, host, row, err)
				}
				if !tmpl.Truth(v) {
					continue
				}
			}
			rows = append(rows, picked{host, row})
		}
	}

	switch r.Kind {
	case Expr:
		for _, p := range rows {
			look(p.host, 1)
			v, err := d.eval(r.Expr, p.host, t, p.row)
			if err != nil {
				return failed("expr", r.Expr, p.host, p.row, err)
			}
			if !tmpl.Truth(v) {
				add(r.Severity, p.host, check.RefOf(t, p.row, ""), check.CellAt(p.row, ""),
					"式が偽です", model.KV{Name: "式", Value: r.Expr})
			}
		}
	case Unique:
		c := t.Column(r.Column)
		first := map[string]picked{}
		for _, p := range rows {
			cell := p.row.Cells[r.Column]
			if cell == nil {
				continue
			}
			counted := false
			mine := map[string]bool{}
			for _, v := range cell.Values {
				if v == "" || mine[v] {
					continue
				}
				mine[v] = true
				if !counted {
					counted = true
					look(p.host, 1)
				}
				key := v
				if r.InDevice {
					key = p.host + string(rune(0)) + v
				}
				f, ok := first[key]
				if !ok {
					first[key] = p
					continue
				}
				other := label(t, f.row)
				add(r.Severity, p.host, check.RefOf(t, p.row, r.Column), check.CellAt(p.row, r.Column),
					fmt.Sprintf("「%s」の値が、%s にもあります", c.Title.JA, other),
					model.KV{Name: "値", Value: shownValue(c, v)}, model.KV{Name: "先の行", Value: other})
			}
		}
	case Equal:
		c := t.Column(r.Column)
		sets := map[string][]string{}
		for _, p := range rows {
			if cell := p.row.Cells[r.Column]; cell != nil {
				for _, v := range cell.Values {
					if v != "" {
						sets[p.host] = append(sets[p.host], shownValue(c, v))
					}
				}
			}
		}
		if len(hosts) < 2 {
			return nil
		}
		text := map[string]string{}
		count := map[string]int{}
		for _, h := range hosts {
			look(h, 1)
			vals := sets[h]
			sort.Slice(vals, func(i, j int) bool { return tmpl.NaturalLess(vals[i], vals[j]) })
			var uniq []string
			for i, v := range vals {
				if i == 0 || v != vals[i-1] {
					uniq = append(uniq, v)
				}
			}
			text[h] = strings.Join(uniq, ", ")
			count[text[h]]++
		}
		// The set most devices have; of two as many, the one of the device
		// that comes first in the book.
		most := text[hosts[0]]
		for _, h := range hosts {
			if count[text[h]] > count[most] {
				most = text[h]
			}
		}
		none := func(s string) string {
			if s == "" {
				return "（なし）"
			}
			return s
		}
		for _, h := range hosts {
			if text[h] == most {
				continue
			}
			add(r.Severity, h, model.Ref{Table: r.Table, Column: r.Column}, "",
				fmt.Sprintf("「%s」の値の集まりが、ほかの %d 台と違います", c.Title.JA, count[most]),
				model.KV{Name: "この機器", Value: none(text[h])}, model.KV{Name: "多い方", Value: none(most)})
		}
	}
	return findings
}
