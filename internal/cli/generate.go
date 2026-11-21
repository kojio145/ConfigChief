package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/ops"
)

// shownFindings is how many findings of one kind a command prints; the rest
// are in the check results.
const shownFindings = 20

func cmdGenerate(args []string, out, errOut io.Writer) int {
	return generate(args, false, out, errOut)
}

func cmdCheck(args []string, out, errOut io.Writer) int {
	return generate(args, true, out, errOut)
}

// generate runs the checks and, unless only the checks were asked for,
// writes the configurations (spec 9, 11).
func generate(args []string, checkOnly bool, out, errOut io.Writer) int {
	values := []string{"--project"}
	if !checkOnly {
		values = append(values, "--device")
	}
	pos, val, _, err := options(args, values, nil)
	if err != nil || len(pos) != 0 {
		if err != nil {
			fmt.Fprintln(errOut, err)
		}
		fmt.Fprint(errOut, usage)
		return 2
	}
	e, ok := openCase(val["--project"], out, errOut)
	if !ok {
		return 2
	}
	o, err := e.Generate(ops.GenOptions{Device: val["--device"], CheckOnly: checkOnly}, now())
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	printGenerate(out, e, o, checkOnly)
	if o.Findings() > 0 {
		return 1
	}
	for _, d := range o.Devices {
		if d.State == ops.GenFailed || d.State == ops.GenBlocked {
			return 1
		}
	}
	return 0
}

func printGenerate(out io.Writer, e *ops.Env, o *ops.GenerateOutcome, checkOnly bool) {
	states := map[string]int{}
	for _, d := range o.Devices {
		states[d.State]++
	}
	if checkOnly {
		fmt.Fprintf(out, "チェック: %d 台（確かめた %d、値の誤りで止めた %d、生成に失敗 %d）\n",
			len(o.Devices), states[ops.GenChecked], states[ops.GenBlocked], states[ops.GenFailed])
	} else {
		fmt.Fprintf(out, "生成: %d 台（書いた %d、値の誤りで止めた %d、生成に失敗 %d、今回の対象外 %d）\n",
			len(o.Devices), states[ops.GenWritten], states[ops.GenBlocked], states[ops.GenFailed], states[ops.GenSkipped])
	}
	for _, d := range o.Devices {
		switch d.State {
		case ops.GenWritten:
			fmt.Fprintf(out, "  %s: %s\n", d.Host, d.File)
		case ops.GenChecked, ops.GenSkipped:
		default:
			line := fmt.Sprintf("  %s: %s", d.Host, ops.GenStateWord(d.State))
			if d.Why != "" {
				line += "（" + d.Why + "）"
			}
			if d.File != "" {
				line += " " + d.File
			}
			fmt.Fprintln(out, line)
		}
	}
	c := o.Count()
	fmt.Fprintf(out, "指摘: 未承認のエラー %d、未承認の警告 %d、承認済み %d、情報 %d\n", c.Errors, c.Warnings, c.Approved, c.Info)
	for _, sev := range []string{check.Error, check.Warning} {
		n := 0
		for _, f := range o.Items {
			if f.Severity != sev || f.Approval != nil {
				continue
			}
			if n++; n > shownFindings {
				continue
			}
			where := f.Device
			if where == "" {
				where = "案件全体"
			}
			if at := e.Where(f.Finding); at != "" {
				where += " " + at
			}
			fmt.Fprintf(out, "  %s %s %s: %s [%s]\n", check.SeverityTitle(sev), f.RuleID, where, f.Message, f.Key)
		}
		if n > shownFindings {
			fmt.Fprintf(out, "  …ほか %d 件（チェック結果表の「指摘」に一覧があります）\n", n-shownFindings)
		}
	}
	for _, s := range o.BookNotes {
		fmt.Fprintf(out, "  ブックの注意: %s\n", s)
	}
	for _, s := range o.Notes {
		fmt.Fprintf(out, "  お知らせ: %s\n", s)
	}
	if len(o.Refused) > 0 {
		fmt.Fprintf(out, "  効いていない承認: %d 件（チェック結果表の「承認」に理由があります）\n", len(o.Refused))
	}
	if o.Unverified != "" {
		fmt.Fprintf(out, "未承認の指摘が残っています。印を置きました: %s\n", o.Unverified)
	}
	if o.Report.XLSX != "" {
		fmt.Fprintf(out, "チェック結果表: %s\n                %s\n", o.Report.XLSX, o.Report.HTML)
	}
	if ch := o.Changes; ch != nil && ch.Report.XLSX != "" {
		added, removed := 0, 0
		for _, l := range ch.Lines {
			if l.Added {
				added++
			} else {
				removed++
			}
		}
		fmt.Fprintf(out, "前回の生成（%s）との違い: 変わったセル %d、増えた行 %d、減った行 %d\n", ch.Before, len(ch.Cells), added, removed)
		fmt.Fprintf(out, "変更一覧: %s\n          %s\n", ch.Report.XLSX, ch.Report.HTML)
	}
}

// cmdApprove adds an approval of a finding or of a scope, or takes one out
// (appendix C 3).
func cmdApprove(args []string, out, errOut io.Writer) int {
	pos, val, _, err := options(args,
		[]string{"--project", "--finding", "--rule", "--devices", "--table", "--column", "--reason", "--by", "--remove"}, nil)
	if err != nil || len(pos) != 0 {
		if err != nil {
			fmt.Fprintln(errOut, err)
		}
		fmt.Fprint(errOut, usage)
		return 2
	}
	e, ok := openCase(val["--project"], out, errOut)
	if !ok {
		return 2
	}
	if id := val["--remove"]; id != "" {
		if len(val) != 2 {
			fmt.Fprintln(errOut, "--remove は、ほかの指定と一緒に使えません")
			return 2
		}
		if err := e.Unapprove(id, now()); err != nil {
			fmt.Fprintln(errOut, err)
			return 2
		}
		fmt.Fprintf(out, "承認 %s を取り消しました\n", id)
		return 0
	}
	if val["--finding"] != "" && val["--devices"]+val["--table"]+val["--column"] != "" {
		fmt.Fprintln(errOut, "--finding と、範囲の指定（--devices・--table・--column）は一緒に使えません")
		return 2
	}
	a, err := e.Approve(ops.ApproveRequest{
		Finding: strings.TrimSpace(val["--finding"]),
		Rule:    strings.TrimSpace(val["--rule"]), Devices: val["--devices"], Table: val["--table"], Column: val["--column"],
		Reason: val["--reason"], By: val["--by"],
	}, now())
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	next := "次の生成・チェックから効きます"
	if a.Migration {
		next = "次の取り込み・比較から効きます"
	}
	if a.Source {
		next = "次に外部一覧を読んだとき（customer）から効きます"
	}
	fmt.Fprintf(out, "承認 %s を足しました（当たった指摘 %d 件）。%s\n", a.Approval.ID, a.Hits, next)
	return 0
}
