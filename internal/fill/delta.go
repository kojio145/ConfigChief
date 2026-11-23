package fill

import (
	"fmt"
	"strings"

	"github.com/kojio145/configchief/internal/conftree"
	"github.com/kojio145/configchief/internal/pack"
)

// The change commands made from the difference between a filled template (the
// configuration as it should be) and a device's configuration now (spec
// 9.7.4).

// Item is one line of a change, for the report.
type Item struct {
	Path []string // the lines of the blocks it stands in (without indentation)
	Text string   // the line (without indentation); for a removal the line as it is now
	Cmd  string   // for a removal: the line that removes it
	// Why tells why a line was kept; for a line replaced, the line that
	// replaces it (MakeDeltaBy).
	Why string
	// Line is the line of the configuration now (1-based; 0 for an added
	// line).
	Line int
}

// Delta is the change of one device.
type Delta struct {
	// Commands are the lines to put in, with the lines that enter and leave
	// the configuration mode; empty when nothing differs.
	Commands []string
	Added    []Item
	Removed  []Item
	// Kept are lines of a block the template writes that the template does
	// not have, left as they are (no {{#sync}} over them).
	Kept []Item
	// Replaced are lines now that an added line writes over (MakeDeltaBy).
	Replaced []Item
}

// Empty tells that nothing differs.
func (d *Delta) Empty() bool { return len(d.Commands) == 0 }

// What becomes of a line now that the target does not have (MakeDeltaBy).
const (
	Keep    = iota // left as it is (Kept)
	Remove         // removed
	Replace        // an added line writes over it (Replaced)
	Enter          // the line stays, and the lines below it are decided one by one
)

// Judge decides about a line now that the target does not have. added are
// the lines of the target at the same level that the configuration now does
// not have. It gives what becomes of the line, why (for Keep and Enter) and,
// for Replace, the added line that writes over it.
type Judge func(now *conftree.Node, added []*conftree.Node) (what int, why string, by *conftree.Node)

// MakeDelta compares a filled template with a configuration now, both read
// with the pack's syntax.
func MakeDelta(out *Output, current string, p *pack.Pack) (*Delta, error) {
	return MakeDeltaBy(out, current, p, nil)
}

// MakeDeltaBy is MakeDelta with the lines to remove decided by judge instead
// of the {{#sync}} marks (spec 9.8); judge nil = the marks.
func MakeDeltaBy(out *Output, current string, p *pack.Pack, judge Judge) (*Delta, error) {
	if p.Change == nil {
		return nil, fmt.Errorf("メーカーパック %s には、変更の投入の書き方（change）がありません", p.ID)
	}
	texts := make([]string, len(out.Lines))
	for i, l := range out.Lines {
		texts[i] = l.Text
	}
	tt, err := conftree.Parse(p.Syntax, strings.Join(texts, "\n")+"\n", p.Options)
	if err != nil {
		return nil, err
	}
	ct, err := conftree.Parse(p.Syntax, current, p.Options)
	if err != nil {
		return nil, err
	}
	d := &differ{p: p, ch: p.Change, out: out, delta: &Delta{}, syncAll: map[*conftree.Node]bool{}, prefixes: map[*conftree.Node][]string{}, judge: judge}
	// Which lines of the template stand in which {{#sync}}.
	byLine := map[int]*conftree.Node{}
	conftree.Walk(tt.Root, func(n *conftree.Node, _ []*conftree.Node) {
		byLine[n.Line] = n
		if i := n.Line - 1; i >= 0 && i < len(out.Lines) {
			if s := out.Lines[i].Sync; s > 0 && out.Syncs[s-1].Words == "" {
				d.syncAll[n] = true
			}
		}
	})
	for _, s := range out.Syncs {
		if s.Words == "" {
			continue
		}
		anchor := tt.Root
		if p.Syntax == "indent" {
			for j := s.Start - 1; j >= 0; j-- {
				n := byLine[j+1]
				if n == nil || n.Body {
					continue
				}
				if lead(n.Text) < s.Indent {
					anchor = n
					break
				}
			}
		}
		d.prefixes[anchor] = append(d.prefixes[anchor], strings.ToLower(s.Words))
	}
	lines := d.walk(tt.Root, ct.Root, false, nil)
	if len(lines) > 0 {
		d.delta.Commands = append(append(append([]string{}, p.Change.Begin...), lines...), p.Change.End...)
	}
	return d.delta, nil
}

type differ struct {
	p        *pack.Pack
	ch       *pack.Change
	out      *Output
	delta    *Delta
	syncAll  map[*conftree.Node]bool
	prefixes map[*conftree.Node][]string
	judge    Judge
}

func lead(s string) int { return width(s[:len(s)-len(strings.TrimLeft(s, " \t"))]) }

func indentOf(s string) string { return s[:len(s)-len(strings.TrimLeft(s, " \t"))] }

// key is what two lines are matched by: the text with its spaces made one,
// and for a block (a banner) the text inside.
func key(n *conftree.Node) string {
	k := conftree.Normalize(n.Text)
	if n.Block != "" {
		k += "\x00" + n.BodyText
	}
	return k
}

func kids(n *conftree.Node) []*conftree.Node {
	var out []*conftree.Node
	for _, c := range n.Children {
		if !c.Body {
			out = append(out, c)
		}
	}
	return out
}

// walk gives the commands for the lines below t (the template) and c (now),
// standing inside them. synced: t's lines not in the template are removed.
func (d *differ) walk(t, c *conftree.Node, synced bool, path []string) []string {
	var out []string
	cur := kids(c)
	used := make([]bool, len(cur))
	pair := func(tc *conftree.Node) *conftree.Node {
		k := key(tc)
		for i, cc := range cur {
			if !used[i] && key(cc) == k {
				used[i] = true
				return cc
			}
		}
		return nil
	}
	var added []*conftree.Node
	for _, tc := range kids(t) {
		cc := pair(tc)
		text := strings.TrimSpace(tc.Text)
		if cc == nil {
			out = append(out, d.subtree(tc)...)
			d.delta.Added = append(d.delta.Added, Item{Path: path, Text: text})
			added = append(added, tc)
			continue
		}
		if tc.Block != "" {
			continue
		}
		sub := d.walk(tc, cc, synced || d.syncAll[tc], append(path[:len(path):len(path)], text))
		if len(sub) > 0 {
			out = append(out, tc.Text)
			out = append(out, sub...)
			out = append(out, d.exit(tc, sub[0]))
		}
	}
	var removed []string
	words := d.prefixes[t]
	for i, cc := range cur {
		if used[i] {
			continue
		}
		text := strings.TrimSpace(cc.Text)
		if d.judge != nil {
			what, why, by := d.judge(cc, added)
			switch what {
			case Remove:
				cmd := d.remove(cc)
				removed = append(removed, cmd)
				d.delta.Removed = append(d.delta.Removed, Item{Path: path, Text: text, Cmd: strings.TrimSpace(cmd), Line: cc.Line})
			case Replace:
				w := ""
				if by != nil {
					w = strings.TrimSpace(by.Text)
				}
				d.delta.Replaced = append(d.delta.Replaced, Item{Path: path, Text: text, Why: w, Line: cc.Line})
			case Enter:
				d.delta.Kept = append(d.delta.Kept, Item{Path: path, Text: text, Why: why, Line: cc.Line})
				if cc.Block != "" {
					continue
				}
				sub := d.walk(&conftree.Node{}, cc, false, append(path[:len(path):len(path)], text))
				if len(sub) > 0 {
					removed = append(removed, cc.Text)
					removed = append(removed, sub...)
					removed = append(removed, d.exit(cc, sub[0]))
				}
			default:
				d.delta.Kept = append(d.delta.Kept, Item{Path: path, Text: text, Why: why, Line: cc.Line})
			}
			continue
		}
		if (synced && t.Line > 0) || hasWords(text, words) {
			cmd := d.remove(cc)
			removed = append(removed, cmd)
			d.delta.Removed = append(d.delta.Removed, Item{Path: path, Text: text, Cmd: strings.TrimSpace(cmd)})
		} else if t.Line > 0 {
			d.delta.Kept = append(d.delta.Kept, Item{Path: path, Text: text})
		}
	}
	return append(out, removed...)
}

// hasWords tells a line that starts with one of the words of a {{#sync 語}}.
func hasWords(text string, words []string) bool {
	low := strings.ToLower(conftree.Normalize(text))
	for _, w := range words {
		if low == w || strings.HasPrefix(low, w+" ") {
			return true
		}
	}
	return false
}

// subtree is a line of the template with everything below it, and the lines
// that leave the blocks it opens.
func (d *differ) subtree(n *conftree.Node) []string {
	out := []string{n.Text}
	var last string
	for _, c := range n.Children {
		if c.Body {
			out = append(out, c.Text)
			continue
		}
		out = append(out, d.subtree(c)...)
		if last == "" {
			last = c.Text
		}
	}
	if last != "" {
		out = append(out, d.exit(n, last))
	}
	return out
}

// exit is the line that leaves the block of n, indented as the line inside.
func (d *differ) exit(n *conftree.Node, inside string) string {
	text := strings.TrimSpace(n.Text)
	for _, x := range d.ch.Exits {
		if x.Parent.MatchString(text) {
			return indentOf(inside) + x.Exit
		}
	}
	return indentOf(inside) + d.ch.Exit
}

// remove is the line that removes a line now.
func (d *differ) remove(n *conftree.Node) string {
	text := strings.TrimSpace(n.Text)
	ind := indentOf(n.Text)
	for _, f := range d.ch.Forms {
		if m := f.Match.FindStringSubmatchIndex(text); m != nil {
			return ind + string(f.Match.ExpandString(nil, f.Remove, text, m))
		}
	}
	neg := d.ch.Negate
	if d.ch.Unnegate && strings.HasPrefix(strings.ToLower(text), strings.ToLower(neg)) {
		return ind + strings.TrimSpace(text[len(neg):])
	}
	return ind + neg + text
}
