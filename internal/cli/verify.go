package cli

import (
	"fmt"
	"io"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/ops"
)

// cmdVerify compares what the devices show after the work (the logs of
// input/logs) with the configuration the book gives (spec 11.7, 15).
func cmdVerify(args []string, out, errOut io.Writer) int {
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
	o, err := e.Verify(now())
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	fmt.Fprintln(out, "反映確認（投入後の設定とパラメータブック）")
	for _, l := range o.Logs {
		switch l.State {
		case ops.LogCompared:
			fmt.Fprintf(out, "  %s: %s（%s）を比べました\n", l.Path, l.Device, l.Pack)
		case ops.LogSkipped:
			fmt.Fprintf(out, "  %s: 比べていません: %s\n", l.Path, l.Why)
		default:
			fmt.Fprintf(out, "  %s: 読めませんでした: %s\n", l.Path, l.Why)
		}
	}
	for _, n := range o.Notes {
		fmt.Fprintf(out, "注意: %s\n", n)
	}
	c := o.Verdicts.Count()
	fmt.Fprintf(out, "機器の設定にない行 %d、ブックにない行 %d、値が違うセル %d、見つからない追加コンフィグの行 %d、機器にだけある未対応の行 %d\n",
		len(o.Missing), len(o.Surplus), len(o.Cells), len(o.Extras), len(o.Unknown))
	fmt.Fprintf(out, "未承認のエラー %d、未承認の警告 %d、承認済み %d、情報 %d\n", c.Errors, c.Warnings, c.Approved, c.Info)
	shown := 0
	for _, f := range o.Verdicts.Items {
		if f.Approval != nil || f.Severity == check.Info {
			continue
		}
		if shown++; shown > 20 {
			fmt.Fprintln(out, "  …（残りはレポートにあります）")
			break
		}
		fmt.Fprintf(out, "  %s %s %s %s %s [%s]\n", check.SeverityTitle(f.Severity), f.RuleID, f.Device, refText(f.Finding), f.Message, f.Key)
	}
	fmt.Fprintf(out, "レポート: %s\n          %s\n", o.Report.XLSX, o.Report.HTML)
	if o.Left() > 0 {
		return 1
	}
	return 0
}

// refText gives the place of a finding as "table key column".
func refText(f check.Finding) string {
	s := f.Ref.Table
	if f.Ref.Key != "" {
		s += " " + f.Ref.Key
	}
	if f.Ref.Column != "" {
		s += " " + f.Ref.Column
	}
	return s
}
