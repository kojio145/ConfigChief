// Package textdiff compares two texts line by line.
//
// Diff gives the aligned lines of both sides (a shortest edit script, see
// myers.go), Unified turns that into the hunks a terminal shows, and Spans
// tells which part of one changed line differs from its counterpart.
// In these three nothing knows what the text is: no line is skipped or
// normalised except for its line end. Tree (tree.go) compares two
// configurations block by block.
package textdiff

import (
	"strconv"
	"strings"
)

// Op says which side a Line belongs to.
type Op int

const (
	Same Op = iota // on both sides
	Del            // only in a
	Add            // only in b
	Chg            // in both, with another value (Tree only)
)

// Line is one line of the result. ALine/BLine are 1-based; 0 = the side has no such line.
type Line struct {
	Op           Op
	ALine, BLine int
	Text         string
}

// Result is the whole comparison: every line of a and b, in order, plus the
// number of added and removed lines.
type Result struct {
	Lines          []Line
	Added, Removed int
}

// Equal reports whether the two texts hold the same lines (no Add and no
// Del).
func (r Result) Equal() bool {
	return r.Added == 0 && r.Removed == 0
}

// Diff compares a and b line by line. "\r\n" and a bare "\r" end a line
// just as "\n" does, and the newline that ends the last line does not make
// an extra empty line, so a text with and without it compare equal.
//
// The Same and Del lines, in order, are the lines of a; the Same and Add
// lines are the lines of b. Between two Same lines, the Del lines come
// before the Add lines.
func Diff(a, b string) Result {
	la, lb := splitLines(a), splitLines(b)
	ops, _ := myers(la, lb)
	var out Result
	if len(ops) > 0 {
		out.Lines = make([]Line, 0, len(ops))
	}
	for _, op := range ops {
		switch op.kind {
		case '=':
			out.Lines = append(out.Lines, Line{Op: Same, ALine: op.i + 1, BLine: op.j + 1, Text: la[op.i]})
		case '-':
			out.Lines = append(out.Lines, Line{Op: Del, ALine: op.i + 1, Text: la[op.i]})
			out.Removed++
		case '+':
			out.Lines = append(out.Lines, Line{Op: Add, BLine: op.j + 1, Text: lb[op.j]})
			out.Added++
		}
	}
	return out
}

// splitLines breaks a text into lines. An empty text has no lines; a text
// that is only a newline has one empty line.
func splitLines(s string) []string {
	if s == "" {
		return nil
	}
	if strings.Contains(s, "\r") {
		s = strings.ReplaceAll(s, "\r\n", "\n")
		s = strings.ReplaceAll(s, "\r", "\n")
	}
	s = strings.TrimSuffix(s, "\n")
	return strings.Split(s, "\n")
}

// Unified renders r as unified-diff hunks for a terminal: each hunk starts
// with "@@ -a,n +b,m @@" (n lines of a from line a, m lines of b from line
// b) and holds the changed lines with up to context unchanged lines around
// them, prefixed " ", "-" and "+". Hunks whose context lines touch or
// overlap are joined into one. When a hunk holds no line of one side, that
// side's count is 0 and its start is the line the hunk comes after (0 at
// the top), as diff -u writes it. Every line ends with "\n". The result is
// "" when r has no changed line. There are no "---" / "+++" file headers.
func Unified(r Result, context int) string {
	if context < 0 {
		context = 0
	}
	lines := r.Lines
	// The hunks as index ranges [start, end) of lines.
	var hunks [][2]int
	for i := 0; i < len(lines); {
		if lines[i].Op == Same {
			i++
			continue
		}
		j := i
		for j < len(lines) && lines[j].Op != Same {
			j++
		}
		start, end := max(i-context, 0), min(j+context, len(lines))
		if n := len(hunks); n > 0 && start <= hunks[n-1][1] {
			hunks[n-1][1] = end
		} else {
			hunks = append(hunks, [2]int{start, end})
		}
		i = j
	}
	var sb strings.Builder
	aPos, bPos, at := 0, 0, 0 // lines of a and of b before lines[at]
	for _, h := range hunks {
		for ; at < h[0]; at++ {
			if lines[at].Op != Add {
				aPos++
			}
			if lines[at].Op != Del {
				bPos++
			}
		}
		an, bn := 0, 0
		for _, l := range lines[h[0]:h[1]] {
			if l.Op != Add {
				an++
			}
			if l.Op != Del {
				bn++
			}
		}
		aStart, bStart := aPos+1, bPos+1
		if an == 0 {
			aStart = aPos
		}
		if bn == 0 {
			bStart = bPos
		}
		sb.WriteString("@@ -")
		sb.WriteString(strconv.Itoa(aStart))
		sb.WriteByte(',')
		sb.WriteString(strconv.Itoa(an))
		sb.WriteString(" +")
		sb.WriteString(strconv.Itoa(bStart))
		sb.WriteByte(',')
		sb.WriteString(strconv.Itoa(bn))
		sb.WriteString(" @@\n")
		for _, l := range lines[h[0]:h[1]] {
			switch l.Op {
			case Del:
				sb.WriteByte('-')
			case Add:
				sb.WriteByte('+')
			default:
				sb.WriteByte(' ')
			}
			sb.WriteString(l.Text)
			sb.WriteByte('\n')
		}
		aPos, bPos, at = aPos+an, bPos+bn, h[1]
	}
	return sb.String()
}
