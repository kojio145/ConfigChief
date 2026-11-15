package textdiff

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// In-line (within-line) differences.
//
// A changed setting is one line in a and one in b that say the same thing
// with a different value: the reader wants the value marked, not the line.
// The segments of such a pair come from inlineDiff: a word-level script
// first, the replaced words split again by character, prefixes and
// suffixes snapped to a delimiter so "100,200" against "100,250" marks
// "200" / "250" and not "00" / "50".

// Spans compares one line of a with one line of b and returns the parts of
// each that differ from the other, as byte ranges [start, end) in order
// (a[s[0]:s[1]] is a changed stretch). Taking the spans out of both lines
// leaves the same text on both sides.
//
// Equal lines have no spans. When the two lines have too little in common
// for a mark to help (see inlineMaxChanged), or one of them is not valid
// UTF-8, each non-empty line is one span from end to end.
func Spans(a, b string) (aSpans, bSpans [][2]int) {
	if a == b {
		return nil, nil
	}
	if !utf8.ValidString(a) || !utf8.ValidString(b) {
		return wholeSpan(a), wholeSpan(b)
	}
	sa, sb := inlineDiff(a, b)
	if sa == nil && sb == nil {
		return wholeSpan(a), wholeSpan(b)
	}
	return segSpans(sa), segSpans(sb)
}

func wholeSpan(s string) [][2]int {
	if s == "" {
		return nil
	}
	return [][2]int{{0, len(s)}}
}

// segSpans gives the byte ranges of the changed segments of one line.
func segSpans(segs []seg) [][2]int {
	var out [][2]int
	pos := 0
	for _, s := range segs {
		if s.kind != '=' {
			out = append(out, [2]int{pos, pos + len(s.text)})
		}
		pos += len(s.text)
	}
	return out
}

// seg is one stretch of a changed line: the same on both sides ('='), only
// in a ('-') or only in b ('+').
type seg struct {
	kind byte
	text string
}

// Inline limits.
const (
	inlineMaxTokens = 200     // token script above this many tokens on a side
	inlineMaxChars  = 200_000 // character script above this product of lengths
	// inlineMaxChanged: a line changed in this share of its characters or
	// more is taken as a whole changed line — a mark cut into slivers reads
	// worse than none.
	inlineMaxChanged = 0.8
)

// inlineDiff returns the segments of a and b, or nil, nil when the two have
// too little in common for a mark to help.
func inlineDiff(a, b string) ([]seg, []seg) {
	if a == b {
		return []seg{{'=', a}}, []seg{{'=', b}}
	}
	ta, tb := tokenize(a), tokenize(b)
	var sa, sb []seg
	if len(ta) >= 2 && len(tb) >= 2 && len(ta) <= inlineMaxTokens && len(tb) <= inlineMaxTokens {
		sa, sb = tokenDiff(ta, tb)
	} else {
		sa, sb = charDiff([]rune(a), []rune(b))
	}
	sa, sb = mergeSegs(sa), mergeSegs(sb)
	if len(sa) == 0 || len(sb) == 0 || tooChanged(sa, a) || tooChanged(sb, b) {
		return nil, nil
	}
	return sa, sb
}

// tooChanged: the changed share of the line is inlineMaxChanged or more.
func tooChanged(segs []seg, whole string) bool {
	changed := 0
	for _, s := range segs {
		if s.kind != '=' {
			changed += utf8.RuneCountInString(s.text)
		}
	}
	n := utf8.RuneCountInString(whole)
	return n > 0 && float64(changed)/float64(n) >= inlineMaxChanged
}

// tokenize splits a line into runs of non-blank and blank characters, in
// turn, so that joining the tokens gives the line back.
func tokenize(s string) []string {
	var out []string
	start := 0
	rs := []rune(s)
	for i := 1; i <= len(rs); i++ {
		if i == len(rs) || unicode.IsSpace(rs[i]) != unicode.IsSpace(rs[start]) {
			out = append(out, string(rs[start:i]))
			start = i
		}
	}
	return out
}

// tokenDiff: the token script of a and b, replaced tokens split by
// character.
func tokenDiff(ta, tb []string) ([]seg, []seg) {
	ops, _ := myers(ta, tb)
	var sa, sb []seg
	for i := 0; i < len(ops); {
		if ops[i].kind == '=' {
			sa = append(sa, seg{'=', ta[ops[i].i]})
			sb = append(sb, seg{'=', tb[ops[i].j]})
			i++
			continue
		}
		var del, ins []string
		for ; i < len(ops) && ops[i].kind != '='; i++ {
			if ops[i].kind == '-' {
				del = append(del, ta[ops[i].i])
			} else {
				ins = append(ins, tb[ops[i].j])
			}
		}
		k := 0
		for ; k < len(del) && k < len(ins); k++ {
			ca, cb := charDiff([]rune(del[k]), []rune(ins[k]))
			sa = append(sa, ca...)
			sb = append(sb, cb...)
		}
		for ; k < len(del); k++ {
			sa = append(sa, seg{'-', del[k]})
		}
		for k2 := len(del); k2 < len(ins); k2++ {
			sb = append(sb, seg{'+', ins[k2]})
		}
	}
	return sa, sb
}

// isDelim: the characters a mark should not start or end in the middle of
// a run between.
func isDelim(r rune) bool {
	return unicode.IsSpace(r) || strings.ContainsRune(",/:;-.=", r)
}

// charDiff: the character script of a and b. The common prefix and suffix
// are taken first and pulled back to a delimiter, the middle is the
// shortest script when the lengths allow it, else one replaced stretch.
func charDiff(a, b []rune) ([]seg, []seg) {
	p := 0
	for p < len(a) && p < len(b) && a[p] == b[p] {
		p++
	}
	for p > 0 && p < len(a) && p < len(b) && !isDelim(a[p-1]) {
		p--
	}
	s := 0
	for s < len(a)-p && s < len(b)-p && a[len(a)-1-s] == b[len(b)-1-s] {
		s++
	}
	for s > 0 && !isDelim(a[len(a)-s]) {
		s--
	}
	ma, mb := a[p:len(a)-s], b[p:len(b)-s]
	var sa, sb []seg
	if p > 0 {
		sa = append(sa, seg{'=', string(a[:p])})
		sb = append(sb, seg{'=', string(b[:p])})
	}
	switch {
	case len(ma) == 0 && len(mb) == 0:
	case len(ma) == 0:
		sb = append(sb, seg{'+', string(mb)})
	case len(mb) == 0:
		sa = append(sa, seg{'-', string(ma)})
	case len(ma)*len(mb) > inlineMaxChars || (!hasDelim(ma) && !hasDelim(mb)):
		// One word against one word ("10" / "11", "20" / "25"): the whole
		// word is what changed, not the characters they happen to share.
		sa = append(sa, seg{'-', string(ma)})
		sb = append(sb, seg{'+', string(mb)})
	default:
		xa, xb := make([]string, len(ma)), make([]string, len(mb))
		for i, r := range ma {
			xa[i] = string(r)
		}
		for i, r := range mb {
			xb[i] = string(r)
		}
		ops, _ := myers(xa, xb)
		for _, op := range ops {
			switch op.kind {
			case '=':
				sa = append(sa, seg{'=', xa[op.i]})
				sb = append(sb, seg{'=', xb[op.j]})
			case '-':
				sa = append(sa, seg{'-', xa[op.i]})
			case '+':
				sb = append(sb, seg{'+', xb[op.j]})
			}
		}
	}
	if s > 0 {
		sa = append(sa, seg{'=', string(a[len(a)-s:])})
		sb = append(sb, seg{'=', string(b[len(b)-s:])})
	}
	return sa, sb
}

func hasDelim(rs []rune) bool {
	for _, r := range rs {
		if isDelim(r) {
			return true
		}
	}
	return false
}

// mergeSegs joins neighbouring segments of one kind.
func mergeSegs(segs []seg) []seg {
	var out []seg
	for _, s := range segs {
		if s.text == "" {
			continue
		}
		if n := len(out); n > 0 && out[n-1].kind == s.kind {
			out[n-1].text += s.text
			continue
		}
		out = append(out, s)
	}
	return out
}
