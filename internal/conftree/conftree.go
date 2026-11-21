package conftree

import (
	"fmt"
	"regexp"
	"strings"
)

// Node is one line of the configuration and the lines below it.
type Node struct {
	Text     string // the line as written, without the line ending
	Line     int    // 1-based
	Children []*Node
	Block    string // id of the block this line starts (see Options.Blocks)
	Body     bool   // a line inside a block: kept verbatim, never matched on its own
	BodyText string // on the first line of a block: its text without the delimiters
}

// Tree is a parsed configuration text.
type Tree struct {
	Root     *Node    // Line 0, no text; its children are the top-level lines
	Lines    []string // every line of the text; Lines[i] is line i+1
	InTree   []bool   // InTree[i] is false for separators (comments, blank lines)
	Warnings []Warning
}

// Warning is a problem in the text that does not stop the parsing.
type Warning struct {
	Line int
	Msg  string
}

// Block describes lines that belong together whatever their indentation, like
// a banner. Start must have a named group "delim" when EndBy is "delim".
type Block struct {
	ID    string
	Start *regexp.Regexp
	EndBy string // "delim": up to the next line that holds the delimiter
}

// Options are the pack's settings for reading a text.
type Options struct {
	CommentPrefixes []string
	Blocks          []Block
	// CommentCloses: a comment line ends every open line, so the line after
	// it is a child of the root whatever its indentation. For a vendor that
	// closes each part of the configuration with a mark ("#") and indents
	// the single lines of the device as a whole by one space.
	CommentCloses bool
}

// Split gives the lines of a text without their line ends. The empty piece
// after the last line end is not a line.
func Split(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	lines := strings.Split(text, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	return lines
}

// Parse turns a configuration text into a tree. kind is the pack's syntax:
// "indent" (a line indented deeper than the line before is its child) or
// "flat" (every line is a child of the root).
func Parse(kind, text string, opt Options) (*Tree, error) {
	if kind != "indent" && kind != "flat" {
		return nil, fmt.Errorf("文面の形「%s」には対応していません", kind)
	}
	lines := Split(text)
	t := &Tree{Root: &Node{}, Lines: lines, InTree: make([]bool, len(lines))}

	type level struct {
		indent int
		node   *Node
	}
	stack := []level{{-1, t.Root}}

	for i := 0; i < len(lines); i++ {
		raw := lines[i]
		body := strings.TrimLeft(raw, " \t")
		if strings.TrimSpace(body) == "" {
			continue
		}
		if isComment(body, opt.CommentPrefixes) {
			if opt.CommentCloses {
				// A mark at the left edge ends everything. An indented one
				// (" #" between the parts of a block, as under "bgp") ends
				// what stands as deep as the mark or deeper, and the block
				// itself stays open.
				ind := 0
				if kind == "indent" {
					ind = width(raw[:len(raw)-len(body)])
				}
				if ind == 0 {
					stack = stack[:1]
				}
				for len(stack) > 1 && stack[len(stack)-1].indent >= ind {
					stack = stack[:len(stack)-1]
				}
			}
			continue
		}
		indent := 0
		if kind == "indent" {
			indent = width(raw[:len(raw)-len(body)])
		}
		for len(stack) > 1 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		parent := stack[len(stack)-1].node
		n := &Node{Text: raw, Line: i + 1}
		parent.Children = append(parent.Children, n)
		t.InTree[i] = true
		stack = append(stack, level{indent, n})

		for _, b := range opt.Blocks {
			m := b.Start.FindStringSubmatchIndex(body)
			if m == nil {
				continue
			}
			n.Block = b.ID
			di := b.Start.SubexpIndex("delim")
			if b.EndBy != "delim" || di < 0 || m[2*di] < 0 {
				return nil, fmt.Errorf("かたまり「%s」の終わり方（end_by）が読めません", b.ID)
			}
			delim := body[m[2*di]:m[2*di+1]]
			rest := body[m[2*di+1]:]
			if j := strings.Index(rest, delim); j >= 0 {
				n.BodyText = rest[:j]
				break // opened and closed on the same line
			}
			var text []string
			if strings.TrimSpace(rest) != "" {
				text = append(text, rest)
			}
			closed := false
			for i++; i < len(lines); i++ {
				n.Children = append(n.Children, &Node{Text: lines[i], Line: i + 1, Body: true})
				t.InTree[i] = true
				if j := strings.Index(lines[i], delim); j >= 0 {
					if j > 0 {
						text = append(text, lines[i][:j])
					}
					closed = true
					break
				}
				text = append(text, lines[i])
			}
			n.BodyText = strings.Join(text, "\n")
			if !closed {
				t.Warnings = append(t.Warnings, Warning{n.Line, fmt.Sprintf("「%s」で始まるかたまりが、終わりの印「%s」のないまま文面の終わりまで続いています", strings.TrimSpace(n.Text), delim)})
			}
			break
		}
	}
	return t, nil
}

func isComment(s string, prefixes []string) bool {
	for _, p := range prefixes {
		if p != "" && strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// width counts the columns of leading white space; a tab goes to the next
// multiple of 8.
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

// Normalize gives the text that rules match against: no indentation, no
// trailing white space, runs of white space made one space.
func Normalize(s string) string {
	return strings.Join(strings.Fields(s), " ")
}

// Walk calls f for every node below n, parents first.
func Walk(n *Node, f func(n *Node, path []*Node)) {
	var rec func(n *Node, path []*Node)
	rec = func(n *Node, path []*Node) {
		for _, c := range n.Children {
			f(c, path)
			rec(c, append(path[:len(path):len(path)], c))
		}
	}
	rec(n, nil)
}
