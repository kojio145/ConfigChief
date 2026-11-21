// Package fill puts values into a template with variables (spec 9.7): the
// values of a list file (variable rules) and of the parameter book, one
// text per device. It knows no vendor; the change commands made from the
// difference with a device's configuration take the vendor's ways from its
// pack (delta.go).
package fill

import (
	"fmt"
	"regexp"
	"strings"
)

// The forms a template holds (spec 9.7.1):
//
//	{{name}}  {{name?}}  {{table.column}}         a value
//	{{#each}} {{#each name}} … {{/each}}          repeat per row
//	{{#if cond}} … {{#else}} … {{/if}}            a condition
//	{{#sync}} {{#sync words}} … {{/sync}}         what a change may remove
//	{{! note}}                                   a note, not put out
var (
	// valueRe is a value: a name, "." between a table and its column.
	valueRe = regexp.MustCompile(`\{\{\s*([\p{L}_][\p{L}\p{N}_\-]*(?:\.[\p{L}_][\p{L}\p{N}_\-]*)?)\s*(\?)?\s*\}\}`)
	// directiveRe is a line of its own that starts or ends a part.
	directiveRe = regexp.MustCompile(`^\{\{\s*([#/!])\s*(.*?)\s*\}\}$`)
	nameRe      = regexp.MustCompile(`^[\p{L}_][\p{L}\p{N}_\-]*(?:\.[\p{L}_][\p{L}\p{N}_\-]*)?$`)
	condRe      = regexp.MustCompile(`^(!)?\s*([\p{L}_][\p{L}\p{N}_\-]*(?:\.[\p{L}_][\p{L}\p{N}_\-]*)?)\s*(?:(!=|=)\s*(.*))?$`)
)

const (
	maxTemplateLines = 20000
	maxDepth         = 20
)

// Kinds of a node.
const (
	kText = iota
	kEach
	kIf
	kSync
)

// node is one part of a template.
type node struct {
	kind int
	line int // 1-based line of the template
	// kText: the line; kEach: the source named ("" = from the names
	// inside); kIf: the condition; kSync: the words ("" = the whole blocks).
	text   string
	indent string // kSync: the indentation of the marker line
	body   []*node
	els    []*node // kIf: the lines after {{#else}}
	cond   cond
}

// cond is a parsed condition.
type cond struct {
	not   bool
	name  string
	op    string // "", "=", "!="
	value string
}

// Template is a parsed template.
type Template struct {
	Name  string
	nodes []*node
	CRLF  bool // the file ends its lines with CR LF
}

// Error is a fault of a template, with its line.
type Error struct {
	Line int
	Msg  string
}

func (e *Error) Error() string {
	if e.Line > 0 {
		return fmt.Sprintf("%d 行目: %s", e.Line, e.Msg)
	}
	return e.Msg
}

func errAt(line int, format string, a ...any) *Error {
	return &Error{Line: line, Msg: fmt.Sprintf(format, a...)}
}

// Parse reads a template. Every fault is one of the form: the first one stops
// the reading, with its line.
func Parse(name, text string) (*Template, error) {
	t := &Template{Name: name, CRLF: strings.Contains(text, "\r\n")}
	text = strings.TrimPrefix(text, string(rune(0xFEFF)))
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	if len(lines) > maxTemplateLines {
		return nil, errAt(0, "ひな形が長すぎます（%d 行まで）", maxTemplateLines)
	}
	type frame struct {
		n    *node
		into *[]*node
	}
	root := []*node{}
	stack := []frame{{nil, &root}}
	top := func() *frame { return &stack[len(stack)-1] }
	inEach, inSync := 0, 0
	for i, raw := range lines {
		ln := i + 1
		trim := strings.TrimSpace(raw)
		if m := directiveRe.FindStringSubmatch(trim); m != nil && strings.Count(trim, "{{") == 1 {
			mark, body := m[1], m[2]
			if mark == "!" {
				continue
			}
			word, arg := body, ""
			if j := strings.IndexAny(body, " \t"); j >= 0 {
				word, arg = body[:j], strings.TrimSpace(body[j+1:])
			}
			word = strings.ToLower(word)
			switch {
			case mark == "#" && word == "each":
				if inEach > 0 {
					return nil, errAt(ln, "{{#each}} の中に {{#each}} は書けません")
				}
				if arg != "" && !nameRe.MatchString(arg) {
					return nil, errAt(ln, "{{#each %s}}: 名前の形が読めません", arg)
				}
				n := &node{kind: kEach, line: ln, text: arg}
				*top().into = append(*top().into, n)
				stack = append(stack, frame{n, &n.body})
				inEach++
			case mark == "#" && word == "if":
				c, ok := parseCond(arg)
				if !ok {
					return nil, errAt(ln, "{{#if %s}}: 条件が読めません（名前・!名前・名前=値・名前!=値）", arg)
				}
				n := &node{kind: kIf, line: ln, text: arg, cond: c}
				*top().into = append(*top().into, n)
				stack = append(stack, frame{n, &n.body})
			case mark == "#" && word == "else":
				f := top()
				if f.n == nil || f.n.kind != kIf || f.into == &f.n.els {
					return nil, errAt(ln, "{{#else}} に当たる {{#if}} がありません")
				}
				f.into = &f.n.els
			case mark == "#" && word == "sync":
				if inSync > 0 {
					return nil, errAt(ln, "{{#sync}} の中に {{#sync}} は書けません")
				}
				n := &node{kind: kSync, line: ln, text: strings.Join(strings.Fields(arg), " "), indent: raw[:len(raw)-len(strings.TrimLeft(raw, " \t"))]}
				*top().into = append(*top().into, n)
				stack = append(stack, frame{n, &n.body})
				inSync++
			case mark == "/" && (word == "each" || word == "if" || word == "sync") && arg == "":
				f := top()
				want := map[string]int{"each": kEach, "if": kIf, "sync": kSync}[word]
				if f.n == nil || f.n.kind != want {
					return nil, errAt(ln, "{{/%s}} に当たる {{#%s}} がありません", word, word)
				}
				stack = stack[:len(stack)-1]
				switch want {
				case kEach:
					inEach--
				case kSync:
					inSync--
				}
			default:
				return nil, errAt(ln, "%s は読めません（{{#each}}・{{#if}}・{{#else}}・{{#sync}} と、その終わりの {{/…}}・{{! メモ}} だけ書けます）", trim)
			}
			if len(stack) > maxDepth {
				return nil, errAt(ln, "入れ子が深すぎます（%d 段まで）", maxDepth)
			}
			continue
		}
		// A text line: its values, and nothing else in braces.
		rest := valueRe.ReplaceAllString(raw, "")
		if j := strings.Index(rest, "{{"); j >= 0 {
			return nil, errAt(ln, "「%s」が読めません（{{名前}}・{{名前?}}・{{表.列}} の形で書きます。{{#…}}・{{/…}} は 1 行に単独で書きます）", snippet(rest[j:]))
		}
		*top().into = append(*top().into, &node{kind: kText, line: ln, text: raw})
	}
	if len(stack) > 1 {
		f := top()
		word := map[int]string{kEach: "each", kIf: "if", kSync: "sync"}[f.n.kind]
		return nil, errAt(f.n.line, "{{#%s}} に当たる {{/%s}} がありません", word, word)
	}
	t.nodes = root
	return t, nil
}

func snippet(s string) string {
	if j := strings.Index(s, "}}"); j >= 0 && j < 60 {
		return s[:j+2]
	}
	r := []rune(s)
	if len(r) > 30 {
		return string(r[:30]) + "…"
	}
	return s
}

func parseCond(s string) (cond, bool) {
	m := condRe.FindStringSubmatch(strings.TrimSpace(s))
	if m == nil {
		return cond{}, false
	}
	c := cond{not: m[1] != "", name: m[2], op: m[3], value: strings.TrimSpace(m[4])}
	if c.not && c.op != "" {
		return cond{}, false
	}
	return c, true
}

// Ref is a name a template uses, as it is written.
type Ref struct {
	Name     string
	Line     int
	Optional bool
	// Each is the source of the {{#each}} the name stands in ("" outside;
	// "*" for a bare {{#each}}).
	Each string
	// EachLine is the line of that {{#each}}.
	EachLine int
}

// Refs lists the names a template uses, values and conditions, in the order
// they come; and the {{#each}} parts with the names inside each.
func (t *Template) Refs() (refs []Ref, eaches []*EachPart) {
	var walk func(ns []*node, each *EachPart)
	walk = func(ns []*node, each *EachPart) {
		for _, n := range ns {
			src, line := "", 0
			if each != nil {
				src, line = each.Source, each.Line
				if src == "" {
					src = "*"
				}
			}
			switch n.kind {
			case kText:
				for _, m := range valueRe.FindAllStringSubmatch(n.text, -1) {
					r := Ref{Name: m[1], Line: n.line, Optional: m[2] != "", Each: src, EachLine: line}
					refs = append(refs, r)
					if each != nil {
						each.Names = append(each.Names, r.Name)
					}
				}
			case kIf:
				r := Ref{Name: n.cond.name, Line: n.line, Optional: true, Each: src, EachLine: line}
				refs = append(refs, r)
				if each != nil {
					each.Names = append(each.Names, r.Name)
				}
				walk(n.body, each)
				walk(n.els, each)
			case kSync:
				walk(n.body, each)
			case kEach:
				e := &EachPart{Source: n.text, Line: n.line, node: n}
				eaches = append(eaches, e)
				walk(n.body, e)
			}
		}
	}
	walk(t.nodes, nil)
	return refs, eaches
}

// EachPart is one {{#each}} of a template.
type EachPart struct {
	Source string // as written ("" = from the names inside)
	Line   int
	Names  []string
	node   *node
}

// HasSync tells whether a template marks anything a change may remove.
func (t *Template) HasSync() bool {
	var has func(ns []*node) bool
	has = func(ns []*node) bool {
		for _, n := range ns {
			if n.kind == kSync || has(n.body) || has(n.els) {
				return true
			}
		}
		return false
	}
	return has(t.nodes)
}
