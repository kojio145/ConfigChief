package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/kojio145/configchief/internal/ops"
	"github.com/kojio145/configchief/internal/textdiff"
)

// cmdTreeDiff compares configurations block by block (spec 12.2, 15): two
// files, or every device of two sides of a case.
func cmdTreeDiff(args []string, out, errOut io.Writer) int {
	pos, val, on, err := options(args, []string{"--project", "--a", "--b"}, []string{"--tree", "--keep-stamps", "--keep-secrets"})
	if err != nil {
		fmt.Fprintln(errOut, err)
		fmt.Fprint(errOut, usage)
		return 2
	}
	opt := textdiff.TreeOptions{Stamps: !on["--keep-stamps"], Secrets: !on["--keep-secrets"]}
	if val["--project"] != "" || val["--a"] != "" || val["--b"] != "" {
		if len(pos) != 0 {
			fmt.Fprint(errOut, usage)
			return 2
		}
		return treeSides(val["--project"], val["--a"], val["--b"], opt, out, errOut)
	}
	if len(pos) != 2 {
		fmt.Fprint(errOut, usage)
		return 2
	}
	ps, err := ops.Bundled()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	p, err := ops.TreeFiles(ps, pos[0], pos[1], opt)
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	for i, l := range []*ops.Listing{p.A, p.B} {
		for _, n := range l.Notes {
			fmt.Fprintf(out, "注意: %s（%s）: %s\n", "AB"[i:i+1], l.Path, n)
		}
	}
	r := p.Result
	if !r.Differs() {
		fmt.Fprintf(out, "違いはありません（%s）\n", leftOut(r))
		return 0
	}
	fmt.Fprintf(out, "--- %s\n+++ %s\n", pos[0], pos[1])
	printTree(out, r)
	fmt.Fprintf(out, "B だけの行 %d、A だけの行 %d、変更 %d、順序 %d（%s）\n", r.Added, r.Removed, r.Changed, r.Moved, leftOut(r))
	if r.Approx {
		fmt.Fprintln(out, "注意: 違いが多すぎるので、行ごとに比べた所があります（件数は多めに出ています）")
	}
	return 1
}

func leftOut(r textdiff.TreeResult) string {
	return fmt.Sprintf("比べなかった行: 時刻の行 %d、証明書・暗号化値 %d、コメントの行 %d", r.Stamps, r.Secrets, r.Comments)
}

// printTree writes the rows that differ, each run of them below the
// headings it stands in ("@@ interface … > …"). The marks: "-" only in A,
// "+" only in B, "~" changed (the line of A, then the line of B), ">" at
// another place.
func printTree(out io.Writer, r textdiff.TreeResult) {
	last := "\x00"
	r.Walk(func(_ int, row textdiff.Row, parents []textdiff.Row) {
		kind := ops.TreeKind(row)
		if kind == "" {
			return
		}
		if trail := ops.TreeTrail(parents); trail != last {
			last = trail
			if trail == "" {
				trail = "（いちばん上）"
			}
			fmt.Fprintf(out, "@@ %s\n", trail)
		}
		a, b := strings.TrimRight(row.A, " \t"), strings.TrimRight(row.B, " \t")
		switch {
		case row.Op == textdiff.Del:
			fmt.Fprintf(out, "-%5d: %s\n", row.ALine, a)
		case row.Op == textdiff.Add:
			fmt.Fprintf(out, "+%5d: %s\n", row.BLine, b)
		case row.Op == textdiff.Chg:
			mark := "~"
			if row.Moved {
				mark = "~>"
			}
			fmt.Fprintf(out, "%s%5d: %s\n %5d: %s\n", mark, row.ALine, a, row.BLine, b)
		default:
			fmt.Fprintf(out, ">%5d: %s\n       （B では %d 行目）\n", row.ALine, a, row.BLine)
		}
	})
}

// treeSides compares every device of two sides.
func treeSides(dir, a, b string, opt textdiff.TreeOptions, out, errOut io.Writer) int {
	e, ok := openCase(dir, out, errOut)
	if !ok {
		return 2
	}
	o, err := e.TreeDiff(a, b, opt, now())
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	fmt.Fprintf(out, "階層を見た比較: A = %s、B = %s（%s）\n", a, b, ops.TreeOptionText(opt))
	for _, d := range o.Devices {
		switch r := d.Result; {
		case r == nil && d.A != nil:
			fmt.Fprintf(out, "  %s: A にだけあります（%s）\n", d.Device, d.A.Path)
		case r == nil:
			fmt.Fprintf(out, "  %s: B にだけあります（%s）\n", d.Device, d.B.Path)
		case !r.Differs():
			fmt.Fprintf(out, "  %s: 違いはありません\n", d.Device)
		default:
			fmt.Fprintf(out, "  %s: B だけの行 %d、A だけの行 %d、変更 %d、順序 %d\n", d.Device, r.Added, r.Removed, r.Changed, r.Moved)
		}
		for i, f := range []*ops.TreeFile{d.A, d.B} {
			if f == nil {
				continue
			}
			for _, n := range f.Notes {
				fmt.Fprintf(out, "      注意: %s（%s）: %s\n", "AB"[i:i+1], f.Path, n)
			}
		}
	}
	for _, f := range o.Failed {
		fmt.Fprintf(out, "  比べなかったファイル %s: %s\n", f.Path, f.Why)
	}
	t := o.Total()
	fmt.Fprintf(out, "機器 %d、B だけの行 %d、A だけの行 %d、変更 %d、順序 %d、比べなかったファイル %d\n", len(o.Devices), t.Added, t.Removed, t.Changed, t.Moved, len(o.Failed))
	fmt.Fprintf(out, "レポート: %s\n          %s\n", o.Report.XLSX, o.Report.HTML)
	if o.Left() > 0 {
		return 1
	}
	return 0
}
