package textdiff

// The shortest edit script (Myers 1986, linear space). Used for lines by
// Diff and for words and characters by Spans.

type editOp struct {
	kind byte // '=', '-', '+'
	i, j int  // index in a and in b; only i counts for '-', only j for '+'
}

// maxDiffLines caps how many lines (both sides together, once the common
// head and tail are set aside) the shortest edit script is computed for;
// beyond it the middle is compared line for line. Memory is linear either
// way; this bounds the time.
const maxDiffLines = 100_000

// maxEdits is how many steps one middle-snake search (see middleSnake)
// takes from each end before giving up on that stretch, which lets it find
// scripts of about twice as many edits: two versions of one text differ in
// a few lines, so a stretch that needs tens of thousands of edits is a
// different text altogether, and a line-for-line comparison serves it as
// well as an exact script that would take seconds to find. Scaled with the
// input so that a large pair with many scattered changes still gets the
// exact answer: a stretch of up to 20,000 lines (both sides together) is
// always searched to the end (the two searches meet within half the lines,
// rounded up).
func maxEdits(lines int) int {
	limit := (lines + 1) / 2
	if limit < 4096 {
		limit = 4096
	}
	if limit > 10000 {
		limit = 10000
	}
	return limit
}

// myers computes a shortest edit script between a and b (Myers 1986, the
// linear-space divide-and-conquer form of section 4b): two frontiers of
// n+m ints and a recursion on the middle snake, so the memory is the
// script itself. The second result says whether a stretch was compared
// line for line instead (see maxDiffLines, maxEdits); the script is then
// still a correct one, only not the shortest.
func myers(a, b []string) ([]editOp, bool) {
	return myersLimit(a, b, maxEdits)
}

// myersLimit is myers with the edit limit given by the caller (maxEdits
// everywhere but in the tests, which cannot wait for the real one).
func myersLimit(a, b []string, limit func(lines int) int) ([]editOp, bool) {
	n, m := len(a), len(b)
	if n+m == 0 {
		return nil, false
	}
	d := &differ{a: a, b: b, limit: limit, vf: make([]int, 2*(n+m)+4), vb: make([]int, 2*(n+m)+4)}
	d.diff(0, n, 0, m)
	orderEdits(d.ops)
	return d.ops, d.approx
}

// orderEdits puts, within each run of edits between two equal lines, the
// removed lines before the added ones. A reader takes "removed, then
// added" as one changed block; which side of a split the middle snake
// lands on could otherwise put an addition first. The ops of one run are
// independent (removals index a, additions b), so their order does not
// change what they rebuild.
func orderEdits(ops []editOp) {
	for i := 0; i < len(ops); {
		if ops[i].kind == '=' {
			i++
			continue
		}
		j := i
		for j < len(ops) && ops[j].kind != '=' {
			j++
		}
		run := ops[i:j]
		var minus, plus []editOp
		for _, op := range run {
			if op.kind == '-' {
				minus = append(minus, op)
			} else {
				plus = append(plus, op)
			}
		}
		copy(run, minus)
		copy(run[len(minus):], plus)
		i = j
	}
}

type differ struct {
	a, b   []string
	ops    []editOp
	approx bool
	limit  func(lines int) int // maxEdits
	vf, vb []int               // the forward and backward frontiers, sized for the whole
}

// diff appends the edit script of a[i0:i1] against b[j0:j1], in order.
func (d *differ) diff(i0, i1, j0, j1 int) {
	// The common head and tail cost nothing to find and keep the search to
	// the part that differs.
	for i0 < i1 && j0 < j1 && d.a[i0] == d.b[j0] {
		d.ops = append(d.ops, editOp{'=', i0, j0})
		i0++
		j0++
	}
	tail := 0
	for i0 < i1-tail && j0 < j1-tail && d.a[i1-1-tail] == d.b[j1-1-tail] {
		tail++
	}
	ti, tj := i1-tail, j1-tail
	i1, j1 = ti, tj
	switch {
	case i0 == i1:
		for j := j0; j < j1; j++ {
			d.ops = append(d.ops, editOp{'+', i0, j})
		}
	case j0 == j1:
		for i := i0; i < i1; i++ {
			d.ops = append(d.ops, editOp{'-', i, j0})
		}
	case (i1-i0)+(j1-j0) > maxDiffLines:
		d.approx = true
		d.pairwise(i0, i1, j0, j1)
	default:
		x, y, u, v, ok := d.middleSnake(i0, i1, j0, j1)
		if !ok {
			d.approx = true
			d.pairwise(i0, i1, j0, j1)
			break
		}
		d.diff(i0, x, j0, y)
		for ; x < u; x, y = x+1, y+1 {
			d.ops = append(d.ops, editOp{'=', x, y})
		}
		d.diff(u, i1, v, j1)
	}
	for k := 0; k < tail; k++ {
		d.ops = append(d.ops, editOp{'=', ti + k, tj + k})
	}
}

// pairwise compares a[i0:i1] and b[j0:j1] line for line — the fallback for
// a stretch too large or too different for the shortest script. Equal
// lines at the same offset are kept as such; what is left over on the
// longer side is added or removed.
func (d *differ) pairwise(i0, i1, j0, j1 int) {
	for i0 < i1 && j0 < j1 {
		if d.a[i0] == d.b[j0] {
			d.ops = append(d.ops, editOp{'=', i0, j0})
		} else {
			d.ops = append(d.ops, editOp{'-', i0, j0}, editOp{'+', i0, j0})
		}
		i0++
		j0++
	}
	for ; i0 < i1; i0++ {
		d.ops = append(d.ops, editOp{'-', i0, j0})
	}
	for ; j0 < j1; j0++ {
		d.ops = append(d.ops, editOp{'+', i0, j0})
	}
}

// middleSnake finds the middle snake of a shortest edit script between
// a[i0:i1] and b[j0:j1] — both non-empty, with no common head or tail — by
// running the forward search from the start and the backward search from
// the end until they meet. It returns the snake's ends (x,y)-(u,v) in whole-
// input coordinates, and false when the two searches have not met within
// the edit limit (the caller then compares the stretch line for line).
func (d *differ) middleSnake(i0, i1, j0, j1 int) (x, y, u, v int, ok bool) {
	a, b := d.a[i0:i1], d.b[j0:j1]
	n, m := len(a), len(b)
	delta := n - m
	odd := delta&1 != 0
	half := (n + m + 1) / 2
	limit := min(d.limit(n+m), half)
	// vf[off+k] is the furthest x reached forward on diagonal k = x-y;
	// vb[offb+k] the furthest-back x on diagonal k from the end. Both index
	// ranges fit the scratch arrays sized for the whole input.
	vf, vb := d.vf, d.vb
	off := half + 2
	offb := half + 2 - delta
	vf[off+1] = 0
	vb[offb+delta-1] = n
	for dd := 0; dd <= limit; dd++ {
		for k := -dd; k <= dd; k += 2 {
			var x0 int
			if k == -dd || (k != dd && vf[off+k-1] < vf[off+k+1]) {
				x0 = vf[off+k+1]
			} else {
				x0 = vf[off+k-1] + 1
			}
			y0 := x0 - k
			xx, yy := x0, y0
			for xx < n && yy < m && a[xx] == b[yy] {
				xx++
				yy++
			}
			vf[off+k] = xx
			if odd && k-delta >= -(dd-1) && k-delta <= dd-1 && vf[off+k] >= vb[offb+k] {
				return i0 + x0, j0 + y0, i0 + xx, j0 + yy, true
			}
		}
		for k := delta - dd; k <= delta+dd; k += 2 {
			var x0 int
			if k == delta+dd || (k != delta-dd && vb[offb+k-1] < vb[offb+k+1]) {
				x0 = vb[offb+k-1]
			} else {
				x0 = vb[offb+k+1] - 1
			}
			y0 := x0 - k
			xx, yy := x0, y0
			for xx > 0 && yy > 0 && a[xx-1] == b[yy-1] {
				xx--
				yy--
			}
			vb[offb+k] = xx
			if !odd && k >= -dd && k <= dd && vb[offb+k] <= vf[off+k] {
				return i0 + xx, j0 + yy, i0 + x0, j0 + y0, true
			}
		}
	}
	return 0, 0, 0, 0, false
}
