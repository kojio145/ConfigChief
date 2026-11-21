package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/ops"
)

// cmdCustomer reads the customer's lists and compares them with the book,
// writes a first column mapping for a list, or puts the values of chosen
// findings into the book (spec 6.5, 15).
func cmdCustomer(args []string, out, errOut io.Writer) int {
	pos, val, _, err := options(args, []string{"--project", "--draft", "--sheet", "--apply", "--apply-rule", "--devices"}, nil)
	modes := 0
	for _, k := range []string{"--draft", "--apply", "--apply-rule"} {
		if val[k] != "" {
			modes++
		}
	}
	if err != nil || len(pos) != 0 || modes > 1 || (val["--sheet"] != "" && val["--draft"] == "") || (val["--devices"] != "" && val["--apply-rule"] == "") {
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
	switch {
	case val["--draft"] != "":
		path, err := e.DraftMapping(val["--draft"], val["--sheet"])
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 2
		}
		fmt.Fprintf(out, "列対応ファイルの下書きを作りました: %s\n見出しの行と行き先を確かめて直してから、customer を実行してください\n", path)
		return 0
	case val["--apply"] != "" || val["--apply-rule"] != "":
		req := ops.ReflectRequest{Rule: val["--apply-rule"], Devices: val["--devices"]}
		for _, f := range strings.Split(val["--apply"], ",") {
			if f = strings.TrimSpace(f); f != "" {
				req.Findings = append(req.Findings, f)
			}
		}
		o, err := e.Reflect(req, now())
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 2
		}
		fmt.Fprintf(out, "ブックに反映しました: 足した行 %d、書き換えたセル %d\n", o.Rows, o.Cells)
		if o.Backup != "" {
			fmt.Fprintf(out, "  書き換える前のブックの控え: %s\n", o.Backup)
		}
		for _, it := range o.Items {
			if it.Skipped != "" {
				fmt.Fprintf(out, "  反映していません（%s %s）: %s\n", it.Finding.RuleID, it.Finding.Key, it.Skipped)
			}
		}
		if o.Skipped() > 0 {
			return 1
		}
		return 0
	}
	o, err := e.Customer(true, now())
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	fmt.Fprintln(out, "外部一覧とパラメータブックの比較")
	for _, l := range o.Lists {
		switch {
		case l.Err != "":
			fmt.Fprintf(out, "  %s: 読めません: %s\n", l.Rule.Path, l.Err)
		default:
			r := l.Result
			fmt.Fprintf(out, "  %s → %s（シート「%s」）: %d 行（除外 %d）から %d 行を作りました\n", l.Rule.Path, l.File, r.Sheet, r.Lines, r.Excluded, r.Rows)
		}
	}
	for _, p := range o.Unmapped {
		fmt.Fprintf(out, "  お知らせ: %s は、どの列対応ファイルにも当たらないので、読んでいません\n", p)
	}
	for _, f := range o.Changed {
		fmt.Fprintf(out, "  お知らせ（S-505）: %s: %s\n", f.Evidence[0].Value, f.Message)
	}
	for _, n := range o.Notes {
		fmt.Fprintf(out, "  注意: %s\n", n)
	}
	if o.Book != nil {
		if n := len(o.Book.Problems) + len(o.Expand); n > 0 {
			fmt.Fprintf(out, "  ブックの注意: %d 件（レポートの「注意」に一覧があります）\n", n)
		}
	}
	c := o.Verdicts.Count()
	fmt.Fprintf(out, "  一覧にだけある行（S-501）: %d\n  値が違うセル（S-503）: %d\n  ブックにだけある行（S-502）: %d\n  読めなかった値（S-504）: %d\n",
		len(o.OnlyList), len(o.Cells), len(o.OnlyBook), len(o.Problems))
	fmt.Fprintf(out, "  未承認のエラー %d 件、未承認の警告 %d 件、承認済み %d 件\n", c.Errors, c.Warnings, c.Approved)
	// The findings that are not approved, up to 20: the rest is in the report.
	shown := 0
	for _, f := range o.Verdicts.Items {
		if f.Approval != nil || f.Severity == check.Info {
			continue
		}
		if shown == 20 {
			fmt.Fprintln(out, "  …（残りはレポートにあります）")
			break
		}
		shown++
		where := f.Device
		if at := e.Where(f.Finding); at != "" {
			where = strings.TrimSpace(where + " " + at)
		}
		fmt.Fprintf(out, "  %s %s %s: %s", check.SeverityTitle(f.Severity), f.RuleID, where, f.Message)
		for _, kv := range f.Evidence {
			fmt.Fprintf(out, " [%s=%s]", kv.Name, kv.Value)
		}
		fmt.Fprintf(out, "（指紋 %s）\n", f.Key)
	}
	fmt.Fprintf(out, "レポート: %s\n          %s\n", o.Report.XLSX, o.Report.HTML)
	if o.Findings() > 0 {
		return 1
	}
	return 0
}
