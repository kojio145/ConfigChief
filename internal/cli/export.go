package cli

import (
	"fmt"
	"io"

	"github.com/kojio145/configchief/internal/ops"
)

// cmdExport writes the delivery sheets or the table of values for the other
// side (spec 13.2, 13.5, 15).
func cmdExport(args []string, out, errOut io.Writer) int {
	pos, val, on, err := options(args, []string{"--project", "--what", "--secrets", "--source", "--revision", "--summary", "--by"}, []string{"--allow-unverified"})
	what, secrets := val["--what"], val["--secrets"]
	if err == nil && len(pos) == 0 && what == "delivery" && val["--source"] == "" && (secrets == "" || secrets == "show" || secrets == "mask") {
		return exportDelivery(val, on["--allow-unverified"], secrets == "mask", out, errOut)
	}
	if val["--revision"] != "" || val["--summary"] != "" || val["--by"] != "" || on["--allow-unverified"] {
		what = "" // they go with "delivery" alone
	}
	if err == nil && len(pos) == 0 && what == "unhandled" && secrets == "" {
		return exportUnhandled(val["--project"], val["--source"], out, errOut)
	}
	if val["--source"] != "" {
		what = "" // --source goes with "unhandled" alone
	}
	if secrets == "" {
		secrets = "show"
	}
	if err != nil || len(pos) != 0 || (what != "sheet" && what != "peer") || (secrets != "show" && secrets != "mask") {
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
	mask := secrets == "mask"
	word := "秘密の値は、そのまま書いています"
	if mask {
		word = "秘密の値は *** に伏せています"
	}
	if what == "peer" {
		o, err := e.ExportPeer(mask, now())
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 2
		}
		for _, n := range o.Notes {
			fmt.Fprintf(out, "注意: %s\n", n)
		}
		if o.Items == 0 {
			fmt.Fprintln(out, "管理外の相手（案件の中に機器がない相手）は、ありません。相手に渡すパラメータ表は作っていません")
			return 0
		}
		fmt.Fprintf(out, "相手に渡すパラメータ表: 管理外の相手 %d、項目 %d（%s）\n", o.Peers, o.Items, word)
		fmt.Fprintf(out, "  %s\n  %s\n", o.Files.XLSX, o.Files.HTML)
		return 0
	}
	o, err := e.ExportSheets(mask, now())
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	for _, n := range o.Notes {
		fmt.Fprintf(out, "注意: %s\n", n)
	}
	if o.Standard {
		fmt.Fprintln(out, "templates/ にひな形（xlsx）が無いので、標準の形で出しました")
	}
	fmt.Fprintf(out, "納品用パラメータシート: %d ファイル（%s）\n", len(o.Files), word)
	for _, f := range o.Files {
		fmt.Fprintf(out, "  %s\n", f)
	}
	if len(o.Problems) > 0 {
		fmt.Fprintf(out, "ひな形の誤り: %d 件（誤りのあるひな形の分は、出していません）\n", len(o.Problems))
		for _, p := range o.Problems {
			fmt.Fprintf(out, "  %s\n", p)
		}
		return 1
	}
	return 0
}

// exportUnhandled writes the report of unhandled lines that may leave the
// site (spec 8.6). The words that were hidden are told on the terminal only.
func exportUnhandled(dir, source string, out, errOut io.Writer) int {
	if source != "" && source != "old" && source != "new" {
		fmt.Fprint(errOut, usage)
		return 2
	}
	e, ok := openCase(dir, out, errOut)
	if !ok {
		return 2
	}
	o, err := e.CarryOut(source, now())
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	lines := 0
	for _, s := range o.Shapes {
		lines += s.Lines
	}
	fmt.Fprintf(out, "未対応行の持ち出し用レポート: %s、形 %d、未対応の行 %d\n", o.Source, len(o.Shapes), lines)
	fmt.Fprintf(out, "  %s\n  %s\n", o.Report.XLSX, o.Report.HTML)
	for _, f := range o.Unread {
		fmt.Fprintf(out, "注意: %s は取り込めなかったので、その行は入っていません\n", f)
	}
	if len(o.Hidden) > 0 {
		fmt.Fprintf(out, "名前として伏せた語: %d 種類（下の一覧は現場の語です。レポートには書いていません。人に渡さないでください）\n", len(o.Hidden))
		const show = 40
		for i, h := range o.Hidden {
			if i == show {
				fmt.Fprintf(out, "  …ほか %d 種類\n", len(o.Hidden)-show)
				break
			}
			fmt.Fprintf(out, "  %s（%d 行）\n", h.Word, h.Lines)
		}
		fmt.Fprintln(out, "コマンドの語が伏せられていたら、project.yaml の export.keep_words に足して、出し直してください")
	}
	fmt.Fprintln(out, "人に渡す前に、レポートのすべての行を目で確かめてください")
	return 0
}

// exportDelivery makes the whole set of the deliverables (spec 13.6, 13.7).
// Findings not approved stop it unless --allow-unverified (exit code 1).
func exportDelivery(val map[string]string, allow, mask bool, out, errOut io.Writer) int {
	e, ok := openCase(val["--project"], out, errOut)
	if !ok {
		return 2
	}
	plan, err := e.PlanDelivery()
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	if !plan.Ready {
		fmt.Fprintln(errOut, plan.Why)
		return 2
	}
	if plan.Known && plan.Errors+plan.Warnings > 0 && !allow {
		fmt.Fprintf(out, "未承認の指摘が残っています（エラー %d・警告 %d）。納品一式は作っていません。直すか承認するか、--allow-unverified を付けてください\n", plan.Errors, plan.Warnings)
		return 1
	}
	o, err := e.MakeDelivery(ops.DeliveryOptions{Mask: mask, AllowUnverified: allow, Version: val["--revision"], Summary: val["--summary"], By: val["--by"]}, now())
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	for _, n := range o.Notes {
		fmt.Fprintf(out, "注意: %s\n", n)
	}
	fmt.Fprintf(out, "納品一式: %s（コンフィグ %d 台・ファイル %d 個）\n  zip: %s\n", o.Dir, o.Configs, len(o.Files), o.Zip)
	if o.Revision != nil {
		fmt.Fprintf(out, "  改版履歴: 版 %s（%s）\n", o.Revision.Version, o.Revision.Summary)
	}
	if o.Unverified {
		fmt.Fprintf(out, "未承認の指摘が残ったまま作りました（エラー %d・警告 %d）\n", o.Errors, o.Warnings)
		return 1
	}
	return 0
}
