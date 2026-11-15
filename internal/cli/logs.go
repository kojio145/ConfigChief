package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/ops"
)

// cmdLogs looks at the logs of the work in input/logs: the commands a device
// did not take, and whether what was typed is the configuration the book
// gives (spec 11.6, 15).
func cmdLogs(args []string, out, errOut io.Writer) int {
	pos, val, _, err := options(args, []string{"--project"}, nil)
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
	o, err := e.LoadCheck(now())
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	fmt.Fprintln(out, "投入ログのチェック")
	for _, l := range o.Logs {
		switch l.State {
		case ops.LogCompared:
			fmt.Fprintf(out, "  %s: %s（%s）のログとして調べました。設定モードのコマンド %d、うちコンフィグの行 %d\n", l.Path, l.Device, l.Pack, l.Commands, l.Matched)
		case ops.LogSkipped:
			fmt.Fprintf(out, "  %s: 調べていません: %s\n", l.Path, l.Why)
		default:
			fmt.Fprintf(out, "  %s: 調べられませんでした: %s\n", l.Path, l.Why)
		}
	}
	for _, n := range o.Notes {
		fmt.Fprintf(out, "注意: %s\n", n)
	}
	for _, x := range o.Rejected {
		at := ""
		if x.ConfigLine > 0 {
			at = fmt.Sprintf("（コンフィグの %d 行目）", x.ConfigLine)
		}
		fmt.Fprintf(out, "  弾かれた行 %s %s の %d 行目%s: %s\n      → %s [%s]\n", x.Device, x.Path, x.Line, at, x.Command, strings.Join(x.Replies, " / "), x.Finding.Key)
	}
	for _, f := range o.Gaps {
		fmt.Fprintf(out, "  %s %s %s: %s [%s]\n", check.SeverityTitle(f.Severity), f.RuleID, f.Device, f.Message, f.Key)
	}
	c := o.Verdicts.Count()
	fmt.Fprintf(out, "弾かれた行 %d、機器のメッセージ %d、未承認のエラー %d、未承認の警告 %d、承認済み %d\n", len(o.Rejected), len(o.Warned), c.Errors, c.Warnings, c.Approved)
	fmt.Fprintf(out, "レポート: %s\n          %s\n", o.Report.XLSX, o.Report.HTML)
	if o.Left() > 0 {
		return 1
	}
	return 0
}
