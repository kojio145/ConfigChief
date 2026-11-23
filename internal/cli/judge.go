package cli

import (
	"fmt"
	"io"
	"strings"

	"github.com/kojio145/configchief/internal/ops"
)

// cmdJudge lists the groups of unhandled lines of a side, or puts a judgment
// on one of them (spec 8.4, 8.5, 15).
func cmdJudge(args []string, out, errOut io.Writer) int {
	pos, val, on, err := options(args,
		[]string{"--project", "--source", "--group", "--verdict", "--devices", "--reason", "--by"}, []string{"--all"})
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
	source := val["--source"]
	if source == "" {
		source = "old"
	}
	if val["--group"] == "" && val["--verdict"] == "" {
		if val["--devices"]+val["--reason"]+val["--by"] != "" {
			fmt.Fprintln(errOut, "判定を付けるには、--group <ID> と --verdict <判定> を指定してください")
			return 2
		}
		j, err := e.Review(source)
		if err != nil {
			fmt.Fprintln(errOut, err)
			return 2
		}
		printJudging(out, j, on["--all"])
		if j.Findings() > 0 {
			return 1
		}
		return 0
	}
	if val["--group"] == "" || val["--verdict"] == "" || on["--all"] {
		fmt.Fprintln(errOut, "判定を付けるには、--group <ID> と --verdict <判定> の両方を指定してください（--all は一覧のときだけ）")
		return 2
	}
	o, err := e.JudgeLines(ops.JudgeRequest{
		Source: source, Group: val["--group"], Devices: val["--devices"],
		Verdict: val["--verdict"], Reason: val["--reason"], By: val["--by"],
	}, now())
	if err != nil {
		fmt.Fprintln(errOut, err)
		return 2
	}
	id := strings.ToLower(strings.TrimSpace(val["--group"]))
	if o.Group != nil {
		id = o.Group.ID
	}
	if o.Verdict == ops.VerdictNone {
		fmt.Fprintf(out, "かたまり %s の判定を取り消しました（機器 %d 台、%d 行）\n", id, o.Devices, o.Lines)
	} else {
		fmt.Fprintf(out, "かたまり %s に「%s」を付けました（機器 %d 台、%d 行）\n", id, ops.VerdictWord(o.Verdict), o.Devices, o.Lines)
	}
	if o.Own > 0 {
		fmt.Fprintf(out, "機器を指定した判定が %d 台に残っています（その機器は変わりません）\n", o.Own)
	}
	if o.Group != nil {
		fmt.Fprintf(out, "このかたまりの判定: %s\n", o.Group.VerdictText())
	}
	j := o.Judging
	fmt.Fprintf(out, "残り（%s）: 未判定 %d 行、保留 %d 行\n", sourceWord(source), j.Open, j.Hold)
	printNoExtra(out, j)
	fmt.Fprintln(out, "取り込みのレポートには、次の取り込み（import）から出ます")
	return 0
}

// printJudging lists the groups: the ones that are open, or all of them.
func printJudging(out io.Writer, j *ops.Judging, all bool) {
	fmt.Fprintf(out, "読めなかった行（%s）: 未判定 %d 行、保留 %d 行、追加コンフィグで書く %d 行、要らない %d 行。かたまり %d 個\n",
		sourceWord(j.Source), j.Open, j.Hold, j.Manual, j.NotNeeded, len(j.Groups))
	shown := 0
	for _, g := range j.Groups {
		if !all && !g.Open() {
			continue
		}
		shown++
		heads, below := g.Count()
		lines := fmt.Sprintf("%d 行", heads)
		if below > 0 {
			lines += fmt.Sprintf("（下に %d 行）", below)
		}
		shape := g.Key.Shape
		if p := g.PathText(); p != "" {
			shape = p + " > " + shape
		}
		fmt.Fprintf(out, "  %s  %s  %d 台  %s  %s  %s\n", g.ID, g.VerdictText(), len(g.Devices), lines, g.Key.Pack, shape)
	}
	switch {
	case len(j.Groups) == 0:
		fmt.Fprintln(out, "  読めなかった行はありません")
	case shown == 0:
		fmt.Fprintln(out, "  未判定・保留のかたまりはありません（--all ですべて出ます）")
	case !all && shown < len(j.Groups):
		fmt.Fprintf(out, "  ほかに、判定の済んだかたまりが %d 個あります（--all ですべて出ます）\n", len(j.Groups)-shown)
	}
	printNoExtra(out, j)
	for _, s := range j.Stale {
		fmt.Fprintf(out, "注意: 判定「%s」（パック %s、形 %s、範囲 %s）は、当たる行がありません\n", ops.VerdictWord(s.Verdict), s.Pack, s.Shape, s.Devices)
	}
	for _, f := range j.Unread {
		fmt.Fprintf(out, "注意: %s は取り込めなかったので、ここに入っていません（import で理由が出ます）\n", f)
	}
}

func printNoExtra(out io.Writer, j *ops.Judging) {
	for _, m := range j.NoExtra {
		who := m.Device
		if m.Host != m.Device {
			who = fmt.Sprintf("%s（新ホスト名 %s）", m.Device, m.Host)
		}
		fmt.Fprintf(out, "M-402: %s に「追加コンフィグで書く」の行が %d 行ありますが、extra に、この機器に当たる追加コンフィグがありません\n", who, m.Lines)
	}
	if j.HostsUnknown {
		fmt.Fprintln(out, "注意: パラメータブックがまだ無いので、「追加コンフィグで書く」の機器に追加コンフィグがあるか（M-402）は見ていません")
	}
	if j.BookErr != "" {
		fmt.Fprintf(out, "注意: パラメータブックを読めないため、グループ向けの追加コンフィグは見ていません: %s\n", j.BookErr)
	}
}
