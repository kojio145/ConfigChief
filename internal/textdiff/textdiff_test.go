package textdiff

import (
	"fmt"
	"math/rand/v2"
	"reflect"
	"slices"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

// All data below is made up for these tests: sample host names and the
// documentation address ranges only.

// refLines is the tests' own reading of the line rule, written apart from
// splitLines on purpose: "\r\n", "\r" and "\n" each end a line, and the
// text after the last one is a line only when it is not empty.
func refLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '\r':
			out = append(out, s[start:i])
			if i+1 < len(s) && s[i+1] == '\n' {
				i++
			}
			start = i + 1
		case '\n':
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// lcsLen is the length of the longest common subsequence, the slow and
// obvious way: a shortest script has len(a)+len(b)-2*lcsLen edits.
func lcsLen(a, b []string) int {
	prev, cur := make([]int32, len(b)+1), make([]int32, len(b)+1)
	for i := range a {
		for j := range b {
			switch {
			case a[i] == b[j]:
				cur[j+1] = prev[j] + 1
			case prev[j+1] >= cur[j]:
				cur[j+1] = prev[j+1]
			default:
				cur[j+1] = cur[j]
			}
		}
		prev, cur = cur, prev
	}
	return int(prev[len(b)])
}

// applyUnified applies the output of Unified to the lines of a, checking
// every header number and every context and removed line on the way.
func applyUnified(t *testing.T, a []string, patch string) []string {
	t.Helper()
	if patch == "" {
		return a
	}
	if !strings.HasSuffix(patch, "\n") {
		t.Fatalf("Unified output does not end with a newline")
	}
	pl := strings.Split(strings.TrimSuffix(patch, "\n"), "\n")
	var out []string
	pos := 0 // lines of a already taken
	for i := 0; i < len(pl); {
		var as, an, bs, bn int
		if _, err := fmt.Sscanf(pl[i], "@@ -%d,%d +%d,%d @@", &as, &an, &bs, &bn); err != nil {
			t.Fatalf("patch line %d: not a hunk header: %q", i+1, pl[i])
		}
		i++
		if an == 0 && bn == 0 {
			t.Fatalf("empty hunk: %q", pl[i-1])
		}
		start := as - 1
		if an == 0 {
			start = as
		}
		if start < pos || start > len(a) {
			t.Fatalf("hunk %q starts at a line already passed or past the end", pl[i-1])
		}
		out = append(out, a[pos:start]...)
		pos = start
		wantB := len(out) + 1
		if bn == 0 {
			wantB = len(out)
		}
		if bs != wantB {
			t.Fatalf("hunk %q: b start should be %d", pl[i-1], wantB)
		}
		for ca, cb := 0, 0; ca < an || cb < bn; i++ {
			if i >= len(pl) || pl[i] == "" {
				t.Fatalf("hunk cut short at patch line %d", i+1)
			}
			text := pl[i][1:]
			switch pl[i][0] {
			case ' ', '-':
				if pos >= len(a) || a[pos] != text {
					t.Fatalf("patch line %d %q does not match line %d of a", i+1, pl[i], pos+1)
				}
				pos++
				ca++
				if pl[i][0] == ' ' {
					out = append(out, text)
					cb++
				}
			case '+':
				out = append(out, text)
				cb++
			default:
				t.Fatalf("patch line %d: unknown prefix: %q", i+1, pl[i])
			}
			if ca > an || cb > bn {
				t.Fatalf("hunk longer than its header says at patch line %d", i+1)
			}
		}
	}
	return append(out, a[pos:]...)
}

// check runs Diff and verifies what must hold for any input: replaying the
// result gives both inputs back, the line numbers count up, the totals
// match the lines, removals come before additions within a block, and the
// unified form applied to a gives b.
func check(t *testing.T, a, b string) Result {
	t.Helper()
	r := Diff(a, b)
	la, lb := refLines(a), refLines(b)
	var ga, gb []string
	an, bn, added, removed := 0, 0, 0, 0
	for i, l := range r.Lines {
		wantA, wantB := 0, 0
		switch l.Op {
		case Same:
			an, bn = an+1, bn+1
			wantA, wantB = an, bn
			ga, gb = append(ga, l.Text), append(gb, l.Text)
		case Del:
			an++
			wantA = an
			removed++
			ga = append(ga, l.Text)
			if i > 0 && r.Lines[i-1].Op == Add {
				t.Fatalf("result line %d: a removal after an addition in one block", i)
			}
		case Add:
			bn++
			wantB = bn
			added++
			gb = append(gb, l.Text)
		default:
			t.Fatalf("result line %d: unknown Op %d", i, l.Op)
		}
		if l.ALine != wantA || l.BLine != wantB {
			t.Fatalf("result line %d: ALine/BLine = %d/%d, want %d/%d", i, l.ALine, l.BLine, wantA, wantB)
		}
	}
	if !slices.Equal(ga, la) {
		t.Fatalf("Same+Del lines do not rebuild a (%d lines, want %d)", len(ga), len(la))
	}
	if !slices.Equal(gb, lb) {
		t.Fatalf("Same+Add lines do not rebuild b (%d lines, want %d)", len(gb), len(lb))
	}
	if r.Added != added || r.Removed != removed {
		t.Fatalf("Added/Removed = %d/%d, the lines say %d/%d", r.Added, r.Removed, added, removed)
	}
	if r.Equal() != (added == 0 && removed == 0) {
		t.Fatalf("Equal() = %v with %d added and %d removed", r.Equal(), added, removed)
	}
	if r.Equal() != slices.Equal(la, lb) {
		t.Fatalf("Equal() = %v, but the lines say otherwise", r.Equal())
	}
	for _, context := range []int{0, 3} {
		u := Unified(r, context)
		if (u == "") != r.Equal() {
			t.Fatalf("Unified(context %d) empty = %v, Equal() = %v", context, u == "", r.Equal())
		}
		if got := applyUnified(t, la, u); !slices.Equal(got, lb) {
			t.Fatalf("Unified(context %d) applied to a does not give b", context)
		}
	}
	return r
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	return strings.Join(lines, "\n") + "\n"
}

const sampleA = "hostname rt-a01\n" +
	"interface port1\n" +
	" ip address 192.0.2.1 255.255.255.0\n" +
	"interface port2\n" +
	" shutdown\n" +
	"end\n"

func TestEqual(t *testing.T) {
	r := check(t, sampleA, sampleA)
	if !r.Equal() || r.Added != 0 || r.Removed != 0 {
		t.Fatalf("equal texts: Equal=%v Added=%d Removed=%d", r.Equal(), r.Added, r.Removed)
	}
	if len(r.Lines) != 6 {
		t.Fatalf("equal texts: %d lines, want 6", len(r.Lines))
	}
	for i, l := range r.Lines {
		if l.Op != Same || l.ALine != i+1 || l.BLine != i+1 {
			t.Fatalf("line %d: %+v", i, l)
		}
	}
	if r.Lines[2].Text != " ip address 192.0.2.1 255.255.255.0" {
		t.Fatalf("leading blanks must be kept: %q", r.Lines[2].Text)
	}
}

func TestPureAdd(t *testing.T) {
	a := "hostname sw-b02\nend\n"
	b := "hostname sw-b02\nvlan 30\n name sample-users\nend\n"
	r := check(t, a, b)
	want := []Line{
		{Same, 1, 1, "hostname sw-b02"},
		{Add, 0, 2, "vlan 30"},
		{Add, 0, 3, " name sample-users"},
		{Same, 2, 4, "end"},
	}
	if !reflect.DeepEqual(r.Lines, want) || r.Added != 2 || r.Removed != 0 {
		t.Fatalf("got %+v (added %d, removed %d)", r.Lines, r.Added, r.Removed)
	}
	// Added at the very end and at the very start.
	if r := check(t, a, a+"banner sample\n"); r.Added != 1 || r.Removed != 0 || r.Lines[2] != (Line{Add, 0, 3, "banner sample"}) {
		t.Fatalf("add at the end: %+v", r)
	}
	if r := check(t, a, "version sample\n"+a); r.Added != 1 || r.Removed != 0 || r.Lines[0] != (Line{Add, 0, 1, "version sample"}) {
		t.Fatalf("add at the start: %+v", r)
	}
}

func TestPureDelete(t *testing.T) {
	a := "hostname sw-b02\nvlan 30\n name sample-users\nend\n"
	b := "hostname sw-b02\nend\n"
	r := check(t, a, b)
	want := []Line{
		{Same, 1, 1, "hostname sw-b02"},
		{Del, 2, 0, "vlan 30"},
		{Del, 3, 0, " name sample-users"},
		{Same, 4, 2, "end"},
	}
	if !reflect.DeepEqual(r.Lines, want) || r.Added != 0 || r.Removed != 2 {
		t.Fatalf("got %+v (added %d, removed %d)", r.Lines, r.Added, r.Removed)
	}
}

func TestChangeInMiddle(t *testing.T) {
	b := "hostname rt-a01\n" +
		"interface port1\n" +
		" ip address 192.0.2.9 255.255.255.0\n" +
		"interface port2\n" +
		" description sample spare\n" +
		" shutdown\n" +
		"end\n"
	r := check(t, sampleA, b)
	want := []Line{
		{Same, 1, 1, "hostname rt-a01"},
		{Same, 2, 2, "interface port1"},
		{Del, 3, 0, " ip address 192.0.2.1 255.255.255.0"},
		{Add, 0, 3, " ip address 192.0.2.9 255.255.255.0"},
		{Same, 4, 4, "interface port2"},
		{Add, 0, 5, " description sample spare"},
		{Same, 5, 6, " shutdown"},
		{Same, 6, 7, "end"},
	}
	if !reflect.DeepEqual(r.Lines, want) {
		t.Fatalf("got %+v", r.Lines)
	}
	if r.Added != 2 || r.Removed != 1 || r.Equal() {
		t.Fatalf("Added=%d Removed=%d Equal=%v", r.Added, r.Removed, r.Equal())
	}
}

// Several changed lines in a row: all the removals, then all the additions.
func TestBlockOrder(t *testing.T) {
	a := "start\nroute 198.51.100.0/24 via 192.0.2.1\nroute 203.0.113.0/24 via 192.0.2.1\nstop\n"
	b := "start\nroute 198.51.100.0/24 via 192.0.2.2\nroute 203.0.113.0/24 via 192.0.2.2\nstop\n"
	r := check(t, a, b)
	var ops []Op
	for _, l := range r.Lines {
		ops = append(ops, l.Op)
	}
	if want := []Op{Same, Del, Del, Add, Add, Same}; !slices.Equal(ops, want) {
		t.Fatalf("ops = %v, want %v", ops, want)
	}
}

func TestLineEnds(t *testing.T) {
	lf := sampleA
	crlf := strings.ReplaceAll(sampleA, "\n", "\r\n")
	cr := strings.ReplaceAll(sampleA, "\n", "\r")
	noEnd := strings.TrimSuffix(sampleA, "\n")
	for name, other := range map[string]string{"CRLF": crlf, "CR": cr, "no last newline": noEnd, "CRLF, no last newline": strings.TrimSuffix(crlf, "\r\n")} {
		for _, pair := range [][2]string{{lf, other}, {other, lf}} {
			r := check(t, pair[0], pair[1])
			if !r.Equal() || len(r.Lines) != 6 {
				t.Fatalf("%s: Equal=%v, %d lines", name, r.Equal(), len(r.Lines))
			}
			for _, l := range r.Lines {
				if strings.ContainsAny(l.Text, "\r\n") {
					t.Fatalf("%s: a line end left in %q", name, l.Text)
				}
			}
		}
	}
	// A real difference is still found across line-end styles.
	r := check(t, crlf, strings.Replace(lf, "port2", "port3", 1))
	if r.Added != 1 || r.Removed != 1 {
		t.Fatalf("CRLF against changed LF: Added=%d Removed=%d", r.Added, r.Removed)
	}
	// Only the newline that ends the last line is dropped: an empty line
	// before it is a line.
	r = check(t, "a\n", "a\n\n")
	if r.Added != 1 || r.Lines[1] != (Line{Add, 0, 2, ""}) {
		t.Fatalf("empty last line: %+v", r)
	}
	// An empty line in the middle is a line too.
	r = check(t, "a\n\nb\n", "a\nb\n")
	if r.Removed != 1 || r.Lines[1] != (Line{Del, 2, 0, ""}) {
		t.Fatalf("empty middle line: %+v", r)
	}
	for in, want := range map[string][]string{
		"":           nil,
		"\n":         {""},
		"\r\n":       {""},
		"a":          {"a"},
		"a\n":        {"a"},
		"a\r":        {"a"},
		"a\rb":       {"a", "b"},
		"a\r\n\r\nb": {"a", "", "b"},
		"a\n\r\nb\r": {"a", "", "b"},
		"\n\n":       {"", ""},
	} {
		if got := splitLines(in); !slices.Equal(got, want) {
			t.Errorf("splitLines(%q) = %q, want %q", in, got, want)
		}
		if got := refLines(in); !slices.Equal(got, want) {
			t.Errorf("refLines(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestEmpty(t *testing.T) {
	r := check(t, "", "")
	if !r.Equal() || len(r.Lines) != 0 {
		t.Fatalf("both empty: %+v", r)
	}
	if !(Result{}).Equal() {
		t.Fatalf("the zero Result must be Equal")
	}
	r = check(t, "", sampleA)
	if r.Added != 6 || r.Removed != 0 || r.Equal() {
		t.Fatalf("empty against text: Added=%d Removed=%d", r.Added, r.Removed)
	}
	for i, l := range r.Lines {
		if l.Op != Add || l.ALine != 0 || l.BLine != i+1 {
			t.Fatalf("line %d: %+v", i, l)
		}
	}
	r = check(t, sampleA, "")
	if r.Added != 0 || r.Removed != 6 || r.Equal() {
		t.Fatalf("text against empty: Added=%d Removed=%d", r.Added, r.Removed)
	}
	for i, l := range r.Lines {
		if l.Op != Del || l.ALine != i+1 || l.BLine != 0 {
			t.Fatalf("line %d: %+v", i, l)
		}
	}
	// One empty line is not the same as no line.
	r = check(t, "", "\n")
	if r.Added != 1 || r.Equal() {
		t.Fatalf("empty against one empty line: %+v", r)
	}
}

// sampleConfig makes n lines that look like a configuration, with many
// repeated lines (the hard case for a diff).
func sampleConfig(n int) []string {
	lines := make([]string, 0, n)
	for i := 0; len(lines) < n; i++ {
		lines = append(lines,
			fmt.Sprintf("interface port%d", i+1),
			fmt.Sprintf(" description sample link %d", i+1),
			fmt.Sprintf(" ip address 198.51.100.%d 255.255.255.255", i%254+1),
			" no shutdown",
			"!",
		)
	}
	return lines[:n]
}

func TestLarge(t *testing.T) {
	a := sampleConfig(5000)
	var b []string
	changed, deleted, inserted := 0, 0, 0
	for i, l := range a {
		switch {
		case i%97 == 42:
			b = append(b, fmt.Sprintf(" description sample changed %d", i))
			changed++
		case i%211 == 100:
			deleted++
		case i%389 == 7:
			b = append(b, l, fmt.Sprintf(" mtu %d", 1400+i%100))
			inserted++
		default:
			b = append(b, l)
		}
	}
	ta, tb := joinLines(a), joinLines(b)
	start := time.Now()
	r := Diff(ta, tb)
	elapsed := time.Since(start)
	if elapsed > 2*time.Second {
		t.Fatalf("Diff of %d lines took %v", len(a), elapsed)
	}
	t.Logf("%d against %d lines: %v, +%d -%d", len(a), len(b), elapsed, r.Added, r.Removed)
	check(t, ta, tb)
	if r.Added != changed+inserted || r.Removed != changed+deleted {
		t.Fatalf("Added/Removed = %d/%d, want %d/%d", r.Added, r.Removed, changed+inserted, changed+deleted)
	}
	if edits, want := r.Added+r.Removed, len(a)+len(b)-2*lcsLen(a, b); edits != want {
		t.Fatalf("%d edits, the shortest script has %d", edits, want)
	}
	if _, approx := myers(a, b); approx {
		t.Fatalf("a few scattered changes must not fall back to line-for-line")
	}
	// The CRLF form of b gives the same result.
	if r2 := Diff(ta, strings.ReplaceAll(tb, "\n", "\r\n")); !reflect.DeepEqual(r, r2) {
		t.Fatalf("CRLF input changed the result")
	}
}

// Two texts with nothing in common are the worst case for the search. An
// odd number of lines above 8,192 is where the edit limit starts to follow
// the size: the search must still run to the end.
func TestWhollyDifferent(t *testing.T) {
	var a, b []string
	for i := range 4097 {
		a = append(a, fmt.Sprintf("old line %d", i))
		b = append(b, fmt.Sprintf("new line %d", i))
	}
	b = b[:4096]
	start := time.Now()
	_, approx := myers(a, b)
	elapsed := time.Since(start)
	t.Logf("%d against %d unrelated lines: %v", len(a), len(b), elapsed)
	if elapsed > 2*time.Second {
		t.Fatalf("the search took %v", elapsed)
	}
	if approx {
		t.Fatalf("a pair of this size must be searched to the end")
	}
	r := check(t, joinLines(a), joinLines(b))
	if r.Added != 4096 || r.Removed != 4097 {
		t.Fatalf("Added/Removed = %d/%d", r.Added, r.Removed)
	}
}

// Many small and middle-sized random pairs over a few distinct lines: the
// result must replay and must be a shortest script.
func TestRandomShortest(t *testing.T) {
	rng := rand.New(rand.NewPCG(20, 26))
	random := func(maxLen, kinds int) []string {
		out := make([]string, rng.IntN(maxLen+1))
		for i := range out {
			out[i] = fmt.Sprintf("line %c", 'a'+rng.IntN(kinds))
		}
		return out
	}
	run := func(a, b []string) {
		t.Helper()
		r := check(t, joinLines(a), joinLines(b))
		if edits, want := r.Added+r.Removed, len(a)+len(b)-2*lcsLen(a, b); edits != want {
			t.Fatalf("%d edits, the shortest script has %d\na=%q\nb=%q", edits, want, a, b)
		}
	}
	for range 4000 {
		run(random(12, 3), random(12, 3))
	}
	for range 300 {
		run(random(300, 6), random(300, 6))
	}
	// A text against an edited copy of itself.
	for range 300 {
		a := random(200, 20)
		var b []string
		for _, l := range a {
			switch rng.IntN(10) {
			case 0:
			case 1:
				b = append(b, l, "line new")
			case 2:
				b = append(b, "line other")
			default:
				b = append(b, l)
			}
		}
		run(a, b)
	}
}

// Past maxDiffLines the middle is compared line for line: not the shortest
// script any more, but still one that rebuilds both sides.
func TestFallbackTooLarge(t *testing.T) {
	n := maxDiffLines/2 + 1
	a, b := make([]string, n), make([]string, n)
	for i := range n {
		a[i] = fmt.Sprintf("old line %d", i)
		b[i] = fmt.Sprintf("new line %d", i)
	}
	a[n/2], b[n/2] = "kept line", "kept line"
	if _, approx := myers(a, b); !approx {
		t.Fatalf("expected the line-for-line fallback")
	}
	r := check(t, joinLines(a), joinLines(b))
	if r.Added != n-1 || r.Removed != n-1 {
		t.Fatalf("Added/Removed = %d/%d, want %d/%d", r.Added, r.Removed, n-1, n-1)
	}
}

// A stretch whose search does not meet within the edit limit falls back
// the same way, and only that stretch. The real limit takes a second to
// reach, so the search runs here with a smaller one of the same shape (it
// grows with the stretch).
func TestFallbackTooManyEdits(t *testing.T) {
	small := func(lines int) int { return lines/8 + 1 }
	rng := rand.New(rand.NewPCG(3, 54))
	fellBack, exact := 0, 0
	for range 2000 {
		a := make([]string, rng.IntN(120))
		for i := range a {
			a[i] = fmt.Sprintf("line %d", rng.IntN(40))
		}
		// b: a with some stretches rewritten, so that some pairs need many
		// edits and others few.
		var b []string
		for i := 0; i < len(a); i++ {
			if rng.IntN(25) == 0 {
				for range rng.IntN(30) {
					b = append(b, fmt.Sprintf("line %d", rng.IntN(40)))
				}
				i += rng.IntN(30)
				continue
			}
			b = append(b, a[i])
		}
		ops, approx := myersLimit(a, b, small)
		var ga, gb []string
		edits := 0
		for _, op := range ops {
			switch op.kind {
			case '=':
				if a[op.i] != b[op.j] {
					t.Fatalf("an equal op for two different lines")
				}
				ga, gb = append(ga, a[op.i]), append(gb, b[op.j])
			case '-':
				ga = append(ga, a[op.i])
				edits++
			case '+':
				gb = append(gb, b[op.j])
				edits++
			}
		}
		if !slices.Equal(ga, a) || !slices.Equal(gb, b) {
			t.Fatalf("the script does not rebuild its inputs (approx=%v)\na=%q\nb=%q", approx, a, b)
		}
		shortest := len(a) + len(b) - 2*lcsLen(a, b)
		if approx {
			fellBack++
			if edits < shortest {
				t.Fatalf("%d edits, fewer than the shortest script (%d)", edits, shortest)
			}
		} else {
			exact++
			if edits != shortest {
				t.Fatalf("%d edits without a fallback, the shortest script has %d", edits, shortest)
			}
		}
	}
	t.Logf("%d fell back, %d exact", fellBack, exact)
	if fellBack < 100 || exact < 100 {
		t.Fatalf("the test data should exercise both ways: %d fell back, %d exact", fellBack, exact)
	}
}

// The real limit: a stretch of up to 20,000 lines is searched to the end
// (a middle snake is always found within half the lines), a larger one for
// 10,000 steps at most.
func TestMaxEdits(t *testing.T) {
	for lines := 2; lines <= 20_000; lines++ {
		if half := (lines + 1) / 2; maxEdits(lines) < half {
			t.Fatalf("maxEdits(%d) = %d, below %d", lines, maxEdits(lines), half)
		}
	}
	for _, lines := range []int{20_002, 100_000, 5_000_000} {
		if got := maxEdits(lines); got != 10_000 {
			t.Fatalf("maxEdits(%d) = %d, want 10000", lines, got)
		}
	}
}

const unifiedB = "hostname rt-a01\n" +
	"interface port1\n" +
	" ip address 192.0.2.9 255.255.255.0\n" +
	"interface port2\n" +
	" shutdown\n" +
	"end\n" +
	"banner sample\n"

func TestUnifiedExact(t *testing.T) {
	r := check(t, sampleA, unifiedB)
	want := "@@ -2,3 +2,3 @@\n" +
		" interface port1\n" +
		"- ip address 192.0.2.1 255.255.255.0\n" +
		"+ ip address 192.0.2.9 255.255.255.0\n" +
		" interface port2\n" +
		"@@ -6,1 +6,2 @@\n" +
		" end\n" +
		"+banner sample\n"
	if got := Unified(r, 1); got != want {
		t.Fatalf("Unified(context 1):\n%s\nwant:\n%s", got, want)
	}
	if got := Unified(check(t, sampleA, sampleA), 3); got != "" {
		t.Fatalf("equal texts: %q", got)
	}
	if got := Unified(Result{}, 3); got != "" {
		t.Fatalf("zero Result: %q", got)
	}
}

func TestUnifiedContext(t *testing.T) {
	r := check(t, sampleA, unifiedB)
	// No context: one hunk per changed block; a side with no line in the
	// hunk has count 0 and names the line the hunk comes after.
	want0 := "@@ -3,1 +3,1 @@\n" +
		"- ip address 192.0.2.1 255.255.255.0\n" +
		"+ ip address 192.0.2.9 255.255.255.0\n" +
		"@@ -6,0 +7,1 @@\n" +
		"+banner sample\n"
	if got := Unified(r, 0); got != want0 {
		t.Fatalf("Unified(context 0):\n%s\nwant:\n%s", got, want0)
	}
	if got := Unified(r, -5); got != want0 {
		t.Fatalf("a negative context must count as 0:\n%s", got)
	}
	// Context 2: the context lines of the two blocks overlap, one hunk.
	// The context stops at the ends of the text.
	want2 := "@@ -1,6 +1,7 @@\n" +
		" hostname rt-a01\n" +
		" interface port1\n" +
		"- ip address 192.0.2.1 255.255.255.0\n" +
		"+ ip address 192.0.2.9 255.255.255.0\n" +
		" interface port2\n" +
		"  shutdown\n" +
		" end\n" +
		"+banner sample\n"
	if got := Unified(r, 2); got != want2 {
		t.Fatalf("Unified(context 2):\n%s\nwant:\n%s", got, want2)
	}
	if got := Unified(r, 100); got != want2 {
		t.Fatalf("a context larger than the text:\n%s", got)
	}

	// Two changes with g unchanged lines between them are one hunk exactly
	// when g <= 2*context.
	for gap := range 8 {
		a := []string{"first old"}
		b := []string{"first new"}
		for i := range gap {
			a = append(a, fmt.Sprintf("kept %d", i))
			b = append(b, fmt.Sprintf("kept %d", i))
		}
		a, b = append(a, "second old"), append(b, "second new")
		r := check(t, joinLines(a), joinLines(b))
		for context := range 5 {
			hunks := strings.Count(Unified(r, context), "@@ -")
			want := 2
			if gap <= 2*context {
				want = 1
			}
			if hunks != want {
				t.Errorf("gap %d, context %d: %d hunks, want %d", gap, context, hunks, want)
			}
		}
	}
}

func TestUnifiedEmptySide(t *testing.T) {
	want := "@@ -0,0 +1,2 @@\n+hostname sw-b02\n+end\n"
	if got := Unified(check(t, "", "hostname sw-b02\nend\n"), 3); got != want {
		t.Fatalf("empty a: %q", got)
	}
	want = "@@ -1,2 +0,0 @@\n-hostname sw-b02\n-end\n"
	if got := Unified(check(t, "hostname sw-b02\nend\n", ""), 3); got != want {
		t.Fatalf("empty b: %q", got)
	}
	// A deletion in the middle, no context: b's start is the line before.
	want = "@@ -2,1 +1,0 @@\n-two\n"
	if got := Unified(check(t, "one\ntwo\nthree\n", "one\nthree\n"), 0); got != want {
		t.Fatalf("deletion: %q", got)
	}
}

// cut takes the spans out of s.
func cut(t *testing.T, s string, spans [][2]int) string {
	t.Helper()
	var sb strings.Builder
	pos := 0
	for _, sp := range spans {
		if sp[0] < pos || sp[1] <= sp[0] || sp[1] > len(s) {
			t.Fatalf("spans %v of %q: empty, out of order or out of range", spans, s)
		}
		if !utf8.ValidString(s[sp[0]:sp[1]]) {
			t.Fatalf("spans %v of %q: cut inside a character", spans, s)
		}
		sb.WriteString(s[pos:sp[0]])
		pos = sp[1]
	}
	sb.WriteString(s[pos:])
	return sb.String()
}

func marked(s string, spans [][2]int) []string {
	var out []string
	for _, sp := range spans {
		out = append(out, s[sp[0]:sp[1]])
	}
	return out
}

func TestSpans(t *testing.T) {
	cases := []struct {
		a, b         string
		wantA, wantB []string
	}{
		// One value of a list: the value, not the digits that differ.
		{"vlan 100,200", "vlan 100,250", []string{"200"}, []string{"250"}},
		// The last part of an address.
		{" ip address 192.0.2.1 255.255.255.0", " ip address 192.0.2.9 255.255.255.0", []string{"1"}, []string{"9"}},
		// A whole word against a whole word.
		{"speed 100 duplex full", "speed 250 duplex full", []string{"100"}, []string{"250"}},
		// A word that only grew: the part that was added.
		{"vlan 10", "vlan 10,20", nil, []string{",20"}},
		// Words added at the end: nothing marked in a.
		{"description sample uplink to rt-a01", "description sample uplink to rt-a01 port2", nil, []string{" port2"}},
		// Words removed in the middle.
		{"permit tcp any host 203.0.113.5 eq 443 log", "permit tcp any host 203.0.113.5 log", []string{"eq 443 "}, nil},
		// Two changes on one line.
		{"route 198.51.100.0/24 via 192.0.2.1 metric 10", "route 198.51.100.0/25 via 192.0.2.1 metric 20", []string{"24", "10"}, []string{"25", "20"}},
		// Characters of more than one byte.
		{"備考 予備の回線", "備考 本番の回線", []string{"予備の回線"}, []string{"本番の回線"}},
		// Nothing in common: each line as a whole.
		{"hostname rt-a01", "banner sample", []string{"hostname rt-a01"}, []string{"banner sample"}},
		// An empty line has nothing to mark.
		{"", "end", nil, []string{"end"}},
		{"end", "", []string{"end"}, nil},
	}
	for _, c := range cases {
		sa, sb := Spans(c.a, c.b)
		if ga, gb := marked(c.a, sa), marked(c.b, sb); !slices.Equal(ga, c.wantA) || !slices.Equal(gb, c.wantB) {
			t.Errorf("Spans(%q, %q) marks %q / %q, want %q / %q", c.a, c.b, ga, gb, c.wantA, c.wantB)
		}
		if ra, rb := cut(t, c.a, sa), cut(t, c.b, sb); ra != rb {
			t.Errorf("Spans(%q, %q): what is left differs: %q / %q", c.a, c.b, ra, rb)
		}
	}
	// Exact byte ranges, once.
	sa, sb := Spans("vlan 100,200", "vlan 100,250")
	if want := [][2]int{{9, 12}}; !reflect.DeepEqual(sa, want) || !reflect.DeepEqual(sb, want) {
		t.Fatalf("spans = %v / %v, want %v", sa, sb, want)
	}
	sa, sb = Spans("備考 予備", "備考 本番です")
	if wa, wb := [][2]int{{7, 13}}, [][2]int{{7, 19}}; !reflect.DeepEqual(sa, wa) || !reflect.DeepEqual(sb, wb) {
		t.Fatalf("spans = %v / %v, want %v / %v", sa, sb, wa, wb)
	}
	if sa, sb := Spans("same line", "same line"); sa != nil || sb != nil {
		t.Fatalf("equal lines: %v / %v", sa, sb)
	}
	// Not valid UTF-8: no offsets inside the line are given.
	bad := "name \xff\xfe one"
	sa, sb = Spans(bad, "name two")
	if !reflect.DeepEqual(sa, [][2]int{{0, len(bad)}}) || !reflect.DeepEqual(sb, [][2]int{{0, 8}}) {
		t.Fatalf("invalid UTF-8: %v / %v", sa, sb)
	}
}

// For any two lines: the spans are in order and inside the line, they do
// not cut a character, and what is left of both lines is the same text.
func TestSpansRandom(t *testing.T) {
	rng := rand.New(rand.NewPCG(11, 14))
	words := []string{"interface", "port1", "port2", "vlan", "10,20,30", "10,20,40", "192.0.2.1", "192.0.2.254",
		"198.51.100.0/24", "description", "sample-uplink", "sample-spare", "予備", "本番", "mtu", "1500", "9000", "a", ""}
	gaps := []string{" ", "  ", "\t", ", ", "=", ":"}
	line := func() string {
		var sb strings.Builder
		for range rng.IntN(9) {
			sb.WriteString(words[rng.IntN(len(words))])
			sb.WriteString(gaps[rng.IntN(len(gaps))])
		}
		return sb.String()
	}
	for range 5000 {
		a := line()
		b := line()
		if rng.IntN(2) == 0 {
			// A near copy: one word replaced.
			b = strings.Replace(a, words[rng.IntN(len(words))], words[rng.IntN(len(words))], 1)
		}
		sa, sb := Spans(a, b)
		ra, rb := cut(t, a, sa), cut(t, b, sb)
		if ra != rb {
			t.Fatalf("Spans(%q, %q) = %v / %v: what is left differs: %q / %q", a, b, sa, sb, ra, rb)
		}
		if (a == b) != (sa == nil && sb == nil) {
			t.Fatalf("Spans(%q, %q) = %v / %v", a, b, sa, sb)
		}
	}
	// A long line goes the character way (more than inlineMaxTokens words).
	long := strings.Repeat("permit 192.0.2.1 ", 150)
	sa, sb := Spans(long+"log", long+"count")
	if ga, gb := marked(long+"log", sa), marked(long+"count", sb); !slices.Equal(ga, []string{"log"}) || !slices.Equal(gb, []string{"count"}) {
		t.Fatalf("long line marks %q / %q", ga, gb)
	}
}
