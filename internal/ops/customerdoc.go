package ops

import (
	"time"

	"github.com/kojio145/configchief/internal/report"
)

// customerDoc builds the report of reading the customer's lists (spec 6.5,
// 13.5).
func (e *Env) customerDoc(o *CustomerOutcome, now time.Time) *report.Doc {
	d := &report.Doc{Title: "外部一覧とパラメータブックの比較", Facts: e.facts(now)}
	c := o.Verdicts.Count()
	d.Counts = []report.Pair{
		{Label: "読んだ一覧", Value: num(len(o.Lists))},
		{Label: "一覧から作った行", Value: num(o.Rows())},
		{Label: "一覧にだけある行（S-501）", Value: num(len(o.OnlyList))},
		{Label: "値が違うセル（S-503）", Value: num(len(o.Cells))},
		{Label: "ブックにだけある行（S-502）", Value: num(len(o.OnlyBook))},
		{Label: "読めなかった値（S-504）", Value: num(len(o.Problems))},
		{Label: "未承認のエラー", Value: num(c.Errors)},
		{Label: "未承認の警告", Value: num(c.Warnings)},
		{Label: "承認済みの指摘", Value: num(c.Approved)},
		{Label: "注意", Value: num(len(o.Notes))},
	}
	d.Notes = append(d.Notes,
		"外部一覧は、mappings/ の列の割り当てファイルのとおりに読んでいます。一覧が空の列と、列の割り当てが行き先にしていない列は比べていません。",
		"パラメータブックは、「機器」列の * と @グループ、「共通値」を機器ごとの行に広げ、空の列を機器のメーカーの既定値で埋めてから比べています。",
		"違いは、資料のチェック（S-501〜S-504）の指摘です。一覧の値が正しければ、ブックに反映します（画面では行を選びます。コマンドラインでは、各行の「指紋」を customer --apply に渡します）。ブックの値が正しければ、承認します（approve）。")
	if e.mask() {
		d.Notes = append(d.Notes, "秘密の列の値は "+masked+" に伏せています。")
	}

	lists := report.Sheet{Name: "一覧", Lead: "列の割り当てファイルと、それが読んだ一覧です。",
		Header: []string{"列の割り当て", "ファイル", "シート", "機器名", "行", "読まなかった行（除外）", "作った行", "読めなかった値", "状態"}}
	for _, l := range o.Lists {
		names := "新しい名前"
		if l.Rule.Old() {
			names = "今の（旧の）名前"
		}
		row := []string{l.Rule.Path, l.File, "", names, "", "", "", "", "読みました"}
		mark := ""
		if r := l.Result; r != nil {
			row[2], row[4], row[5], row[6], row[7] = r.Sheet, num(r.Lines), num(r.Excluded), num(r.Rows), num(len(r.Problems))
		}
		if l.Err != "" {
			row[8], mark = "読めませんでした: "+l.Err, report.MarkUnknown
		}
		lists.Rows = append(lists.Rows, row)
		lists.Marks = append(lists.Marks, mark)
	}

	rows := func(name, lead, where string, list []RowFinding) report.Sheet {
		s := report.Sheet{Name: name, Lead: lead, Header: append([]string{"表", "機器", "行", "内容", where}, stateHeader...)}
		for _, x := range list {
			st, mark := stateCells(o.Verdicts, x.Finding)
			s.Rows = append(s.Rows, append([]string{e.tableTitle(x.Table), x.Device, e.rowLabel(x.Row), e.rowText(x.Row), originText(x.Row.Origins)}, st...))
			s.Marks = append(s.Marks, mark)
		}
		return s
	}
	cells := report.Sheet{Name: "値が違うセル", Lead: "一覧とブックの両方にある行のうち、一覧に値があって、ブックの値と違うセルです（S-503）。",
		Header: append([]string{"表", "機器", "行", "列", "一覧の値", "ブックの値", "備考", "一覧の場所", "ブックの場所"}, stateHeader...)}
	for _, x := range o.Cells {
		note := ""
		switch {
		case e.mask() && e.secret(x.Table, x.Column):
			note = "秘密の値（伏せています）"
		case x.OrderOnly:
			note = "並びだけが違います"
		case x.B == nil:
			note = "ブックは空"
		case isDefault(x.B):
			note = "ブックは既定値（書かれていない値）"
		}
		st, mark := stateCells(o.Verdicts, x.Finding)
		cells.Rows = append(cells.Rows, append([]string{e.tableTitle(x.Table), x.Device, e.rowLabel(x.ARow), e.columnTitle(x.Table, x.Column),
			e.cellText(x.Table, x.Column, x.A), e.cellText(x.Table, x.Column, x.B), note, cellOrigin(x.A), cellOrigin(x.B)}, st...))
		cells.Marks = append(cells.Marks, mark)
	}
	probs := report.Sheet{Name: "読めなかった値", Lead: "一覧の値のうち、設計データに入れられなかったものです（S-504）。一覧か列の割り当てファイルを直します。",
		Header: append([]string{"ファイル", "シート", "セル", "機器", "内容"}, stateHeader...)}
	for _, p := range o.Problems {
		st, mark := stateCells(o.Verdicts, p.Finding)
		probs.Rows = append(probs.Rows, append([]string{p.File, p.Sheet, p.Addr, p.Device, p.Text}, st...))
		probs.Marks = append(probs.Marks, mark)
	}
	d.Sheets = []report.Sheet{
		lists,
		rows("一覧にだけある行", "外部一覧にあって、パラメータブックにない行です（S-501）。", "一覧の場所", o.OnlyList),
		cells,
		rows("ブックにだけある行", "パラメータブックにあって、外部一覧にない行です（S-502。情報）。見ているのは、一覧が行き先にしている表の、一覧に出てくる機器の行と、機器の表の全部の行です。", "ブックの場所", o.OnlyBook),
		probs,
		approvalSheet(o.Verdicts),
	}
	d.Sheets[5].Lead = "state/approvals.yaml にある資料のチェック（S-501〜S-504）の承認と、今回それぞれが当たった指摘の件数です。"

	notes := report.Sheet{Name: "注意", Header: []string{"種類", "内容"}}
	for _, n := range o.Notes {
		notes.Rows = append(notes.Rows, []string{"注意", n})
		notes.Marks = append(notes.Marks, report.MarkWarn)
	}
	if o.Book != nil {
		for _, p := range o.Book.Problems {
			notes.Rows = append(notes.Rows, []string{"ブックの注意", p.String()})
			notes.Marks = append(notes.Marks, report.MarkWarn)
		}
	}
	for _, p := range o.Expand {
		notes.Rows = append(notes.Rows, []string{"ブックの注意", p.String()})
		notes.Marks = append(notes.Marks, report.MarkWarn)
	}
	for _, f := range o.Changed {
		notes.Rows = append(notes.Rows, []string{"お知らせ（S-505）", f.Evidence[0].Value + ": " + f.Message})
	}
	for _, p := range o.Unmapped {
		notes.Rows = append(notes.Rows, []string{"お知らせ", p + " は、どの列の割り当てファイルにも当たらないので、読んでいません"})
	}
	for _, n := range o.BookNotes {
		notes.Rows = append(notes.Rows, []string{"お知らせ", n})
	}
	d.Sheets = append(d.Sheets, notes)
	return d
}
