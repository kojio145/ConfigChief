package ops

import (
	"fmt"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/compare"
	"github.com/kojio145/configchief/internal/importer"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/report"
)

// The check of what is on the devices (spec 11.7, appendix C 2.3): the
// configuration a device shows after the work is read from input/logs and
// set beside the configuration the book gives now.

// The states of a log of input/logs.
const (
	LogCompared = "compared" // set beside the generated configuration
	LogSkipped  = "skipped"  // nothing to compare: Why tells
	LogFailed   = "failed"   // could not be read: Why tells
)

// VerifyLog is one file of input/logs.
type VerifyLog struct {
	Path   string
	Pack   string
	Device string // the host name the configuration in the log gives
	State  string
	Why    string
	// Lines counts the lines of the configuration in the log; Unknown those
	// of them no rule of the pack read.
	Lines, Unknown int
	Notes          []string
}

// ExtraLine is a line of an extra configuration that the device does not
// show (R-804), or a line of the device that nothing explains (R-805).
type ExtraLine struct {
	Device  string
	Path    string // the log, or the template of the extra configuration
	Line    int
	Parents []string
	Text    string
	Finding check.Finding
}

// VerifyOutcome is what one run of the check found.
type VerifyOutcome struct {
	Logs     []VerifyLog
	Missing  []RowFinding  // R-801: in the book, not on the device
	Surplus  []RowFinding  // R-802: on the device, not in the book
	Cells    []CellFinding // R-803 and R-807
	Extras   []ExtraLine   // R-804
	Unknown  []ExtraLine   // R-805
	Strays   []check.Finding
	Verdicts *Verdicts
	Report   Files
	Notes    []string
	// Untold lists the devices of the book no log spoke of.
	Untold []string
}

// Left counts what a person has to look at: the findings that are errors or
// warnings and not approved, and the logs that could not be read.
func (o *VerifyOutcome) Left() int {
	c := o.Verdicts.Count()
	n := c.Errors + c.Warnings
	for _, l := range o.Logs {
		if l.State == LogFailed {
			n++
		}
	}
	return n
}

// lineTrail gives the texts of the lines a line hangs below, the outermost
// first, and its own text.
func lineTrail(lines []importer.LineState, i int) ([]string, string) {
	var parents []string
	for p := lines[i].Parent; p > 0 && p <= len(lines); p = lines[p-1].Parent {
		parents = append([]string{strings.TrimSpace(lines[p-1].Text)}, parents...)
	}
	return parents, strings.TrimSpace(lines[i].Text)
}

func pathKey(parents []string, text string) string {
	return strings.Join(append(append([]string{}, parents...), text), "\x00")
}

// noConfigShown tells why a log is not compared.
const noConfigShown = "設定全体の表示（show running-config など）がありません。反映確認には使いません"

// AppliedApprovalNote stands in the list of the approvals of the generation
// for an approval of a check of what is on the devices.
const AppliedApprovalNote = "投入ログ・反映確認のチェックの承認（当たった件数は、そのレポートにあります）"

// Verify reads the logs of input/logs and compares what each device shows
// with the configuration the book gives. It writes the report.
func (e *Env) Verify(now time.Time) (*VerifyOutcome, error) { return e.verify(now, true) }

// verify is Verify; without write, no report and no line of the log.
func (e *Env) verify(now time.Time, write bool) (*VerifyOutcome, error) {
	paths, err := e.Proj.Inputs("logs")
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("input/logs にファイルがありません。投入後に取ったログ（設定全体の表示が入ったもの）を置いてください")
	}
	gen, err := e.Generate(GenOptions{Dry: true}, now)
	if err != nil {
		return nil, err
	}
	_, bk, err := e.readBook()
	if err != nil {
		return nil, err
	}
	chosen, err := e.Proj.Sources()
	if err != nil {
		return nil, err
	}
	runs := map[string]*DeviceRun{}
	for i := range gen.Devices {
		runs[gen.Devices[i].Host] = &gen.Devices[i]
	}
	out := &VerifyOutcome{}
	var findings []check.Finding
	rule := func(id, device string, ref model.Ref, msg string, ev ...model.KV) check.Finding {
		r, _ := check.Lookup(id)
		f := check.New(id, r.Severity, device, ref, msg, ev...)
		findings = append(findings, f)
		return f
	}
	seen := map[string]string{}
	for i, path := range paths {
		if e.Tick != nil {
			if err := e.Tick(i, len(paths), path); err != nil {
				return nil, err
			}
		}
		l := VerifyLog{Path: path, State: LogFailed}
		text, err := project.ReadText(e.Proj.Path(path))
		switch {
		case err != nil:
			l.Why = err.Error()
		case strings.TrimSpace(text) == "":
			l.Why = "中身が空です"
		}
		if l.Why != "" {
			out.Logs = append(out.Logs, l)
			continue
		}
		cands, clear := e.detect(text)
		switch {
		case clear != "":
			l.Pack = clear
		case chosen.Pack(path) != "" && e.Packs[chosen.Pack(path)] != nil:
			l.Pack = chosen.Pack(path)
		default:
			// A log that shows no configuration gives no pack either: told
			// as what it is, by the prompts any pack sees in it.
			shows, prompts := false, false
			for _, id := range e.PackIDs {
				cut := e.Packs[id].Session.Cut(text)
				shows = shows || cut.Log
				prompts = prompts || cut.Prompts >= 1
			}
			if prompts && !shows {
				l.State, l.Why = LogSkipped, noConfigShown
			} else {
				l.Why = "メーカーを判定できません（" + candidateText(cands) + "）。state/sources.yaml にパックを書いてください"
			}
			out.Logs = append(out.Logs, l)
			continue
		}
		p := e.Packs[l.Pack]
		// A log that shows no configuration has nothing to compare.
		if cut := p.Session.Cut(text); !cut.Log && cut.Prompts >= 1 {
			l.State, l.Why = LogSkipped, noConfigShown
			out.Logs = append(out.Logs, l)
			continue
		}
		res, err := importer.Import(p, e.Schema, path, text)
		if err != nil {
			l.Why = err.Error()
			out.Logs = append(out.Logs, l)
			continue
		}
		l.Device, l.Notes = res.Device, res.Warnings
		c := res.Count()
		l.Lines, l.Unknown = c.Total-c.Separators, c.Unknown
		run := runs[res.Device]
		switch {
		case seen[res.Device] != "":
			l.Why = fmt.Sprintf("ホスト名 %s が %s と同じです。このファイルは比べません", res.Device, seen[res.Device])
		case run == nil:
			l.Why = "ホスト名が、パラメータブックのどの機器にも当たりません"
			out.Strays = append(out.Strays, rule("R-806", res.Device, model.Ref{}, fmt.Sprintf("ログ %s のホスト名 %s が、案件のどの機器にも当たりません", path, res.Device),
				model.KV{Name: "ログ", Value: path}, model.KV{Name: "ホスト名", Value: res.Device}))
		case run.Pack != l.Pack:
			l.Why = fmt.Sprintf("ログは %s の形ですが、機器のシートのメーカーは %s です", l.Pack, run.Pack)
			out.Strays = append(out.Strays, rule("R-806", res.Device, model.Ref{}, fmt.Sprintf("ログ %s は %s の形ですが、機器 %s のメーカーは %s です（別の機器のログの疑い）", path, l.Pack, res.Device, run.Pack),
				model.KV{Name: "ログ", Value: path}, model.KV{Name: "ログのメーカー", Value: l.Pack}, model.KV{Name: "機器のメーカー", Value: run.Pack}))
		case run.Output == nil || run.Output.Failed:
			l.State, l.Why = LogSkipped, "この機器は、いまコンフィグを生成できないので比べられません（check で理由を確かめてください）"
			if run.Why != "" {
				l.Why += ": " + run.Why
			}
		}
		if l.Why != "" {
			out.Logs = append(out.Logs, l)
			continue
		}
		seen[res.Device] = path
		want, err := importer.Import(p, e.Schema, res.Device+".txt", run.Output.Text)
		if err != nil {
			l.Why = "生成したコンフィグを取り込み直せません: " + err.Error()
			out.Logs = append(out.Logs, l)
			continue
		}
		l.State = LogCompared
		out.Logs = append(out.Logs, l)
		e.verifyDevice(out, rule, res.Device, path, run, want, res)
	}
	for _, d := range gen.Devices {
		if seen[d.Host] == "" {
			out.Untold = append(out.Untold, d.Host)
		}
	}
	if len(out.Untold) > 0 {
		out.Notes = append(out.Notes, fmt.Sprintf("ログが無い機器が %d 台あります（比べていません）: %s", len(out.Untold), strings.Join(out.Untold, ", ")))
	}
	var bookData *model.Dataset
	if bk != nil {
		bookData = bk.Data
	}
	if out.Verdicts, err = e.verdicts(findings, bookData); err != nil {
		return nil, err
	}
	if !write {
		return out, nil
	}
	if out.Report, err = e.writeReport(now, "verify", e.verifyDoc(out, now)); err != nil {
		return nil, err
	}
	c := out.Verdicts.Count()
	e.Proj.Log(now, "verify: ログ %d 個、機器の設定にない行 %d、ブックにない行 %d、値が違うセル %d、見つからない追加コンフィグの行 %d（未承認のエラー %d、警告 %d）、レポート %s",
		len(out.Logs), len(out.Missing), len(out.Surplus), len(out.Cells), len(out.Extras), c.Errors, c.Warnings, out.Report.XLSX)
	return out, nil
}

// verifyDevice compares one device: want is the generated configuration
// read back, got the configuration of the log.
func (e *Env) verifyDevice(out *VerifyOutcome, rule func(id, device string, ref model.Ref, msg string, ev ...model.KV) check.Finding,
	host, path string, run *DeviceRun, want, got *importer.Result) {
	diff := compare.Datasets(e.Schema, want.Data, got.Data)
	for _, d := range diff.OnlyA {
		t := e.Schema.Table(d.Table)
		f := rule("R-801", host, check.RefOf(t, d.Row, ""), fmt.Sprintf("%s の行が、機器の設定にありません", t.Title.JA), model.KV{Name: "行", Value: e.rowText(d.Row)})
		out.Missing = append(out.Missing, RowFinding{RowDiff: d, Finding: f})
	}
	for _, d := range diff.OnlyB {
		t := e.Schema.Table(d.Table)
		f := rule("R-802", host, check.RefOf(t, d.Row, ""), fmt.Sprintf("機器の設定にある %s の行が、パラメータブックにありません", t.Title.JA), model.KV{Name: "行", Value: e.rowText(d.Row)})
		if len(d.Row.Origins) > 0 {
			f.Line = d.Row.Origins[0].Line
		}
		out.Surplus = append(out.Surplus, RowFinding{RowDiff: d, Finding: f})
	}
	for _, d := range diff.Cells {
		t := e.Schema.Table(d.Table)
		c := t.Column(d.Column)
		if c == nil {
			continue
		}
		// A secret the device keeps as a hash cannot be set beside the value
		// of the book (appendix C 2.3, R-807).
		if t.Column("secret_form") != nil && d.BRow.Get("secret_form") == "hashed" && d.ARow.Get("secret_form") != "hashed" {
			if d.Column == "secret_form" {
				continue
			}
			if c.Secret {
				f := rule("R-807", host, check.RefOf(t, d.ARow, d.Column), fmt.Sprintf("「%s」は、機器がハッシュで持っているので比べていません", c.Title.JA))
				out.Cells = append(out.Cells, CellFinding{CellDiff: d, Finding: f})
				continue
			}
		}
		f := rule("R-803", host, check.RefOf(t, d.ARow, d.Column), fmt.Sprintf("「%s」が、パラメータブックと機器の設定で違います", c.Title.JA),
			model.KV{Name: "パラメータブック", Value: check.Shown(c, d.A)}, model.KV{Name: "機器の設定", Value: check.Shown(c, d.B)})
		if d.B != nil && len(d.B.Origins) > 0 {
			f.Line = d.B.Origins[0].Line
		}
		out.Cells = append(out.Cells, CellFinding{CellDiff: d, Finding: f})
	}

	// The lines of the extra configurations the pack cannot read: looked for
	// by their text, below the same lines.
	onDevice := map[string]bool{}
	for i, l := range got.Lines {
		if !l.Separator {
			parents, text := lineTrail(got.Lines, i)
			onDevice[pathKey(parents, text)] = true
		}
	}
	extra := map[string]bool{}
	for i, l := range want.Lines {
		if l.Separator || l.State != importer.Unknown {
			continue
		}
		parents, text := lineTrail(want.Lines, i)
		key := pathKey(parents, text)
		extra[key] = true
		if onDevice[key] {
			continue
		}
		from := ""
		if i < len(run.Output.Lines) {
			from = run.Output.Lines[i].Template
		}
		f := rule("R-804", host, model.Ref{}, "追加コンフィグの行が、機器の設定に見つかりません",
			model.KV{Name: "行", Value: text}, model.KV{Name: "親の行", Value: strings.Join(parents, " > ")}, model.KV{Name: "書いた所", Value: from})
		out.Extras = append(out.Extras, ExtraLine{Device: host, Path: from, Line: i + 1, Parents: parents, Text: text, Finding: f})
	}
	for i, l := range got.Lines {
		if l.Separator || l.State != importer.Unknown {
			continue
		}
		parents, text := lineTrail(got.Lines, i)
		if extra[pathKey(parents, text)] {
			continue
		}
		f := rule("R-805", host, model.Ref{}, "機器の設定に、パックが読めず、追加コンフィグにも無い行があります",
			model.KV{Name: "行", Value: text}, model.KV{Name: "親の行", Value: strings.Join(parents, " > ")}, model.KV{Name: "ログ", Value: fmt.Sprintf("%s の %d 行目", path, l.Line)})
		f.Line = l.Line
		out.Unknown = append(out.Unknown, ExtraLine{Device: host, Path: path, Line: l.Line, Parents: parents, Text: text, Finding: f})
	}
}

// verifyDoc builds the report of the check (spec 11.7, 13.5).
func (e *Env) verifyDoc(o *VerifyOutcome, now time.Time) *report.Doc {
	d := &report.Doc{Title: "反映確認（投入後の設定とパラメータブック）", Facts: e.facts(now)}
	c := o.Verdicts.Count()
	compared := 0
	for _, l := range o.Logs {
		if l.State == LogCompared {
			compared++
		}
	}
	d.Counts = []report.Pair{
		{Label: "ログ", Value: num(len(o.Logs))},
		{Label: "比べた機器", Value: num(compared)},
		{Label: "機器の設定にない行（R-801）", Value: num(len(o.Missing))},
		{Label: "ブックにない行（R-802）", Value: num(len(o.Surplus))},
		{Label: "値が違うセル（R-803）", Value: num(len(o.Cells))},
		{Label: "見つからない追加コンフィグの行（R-804）", Value: num(len(o.Extras))},
		{Label: "機器にだけある読めなかった行（R-805）", Value: num(len(o.Unknown))},
		{Label: "未承認のエラー", Value: num(c.Errors)},
		{Label: "未承認の警告", Value: num(c.Warnings)},
		{Label: "承認済みの指摘", Value: num(c.Approved)},
	}
	d.Notes = append(d.Notes,
		"input/logs のログから設定全体の表示を切り出して取り込み、パラメータブックから今生成したコンフィグ（同じメーカーパックで取り込み直したもの）と、値で比べています。並び・省略形・機器が自動で足す行の違いは、指摘になりません。",
		"パックが書けない値（生成のチェックの G-602）は、ここでは比べていません。チェック結果表を見てください。")
	d.Notes = append(d.Notes, o.Notes...)
	if e.mask() {
		d.Notes = append(d.Notes, "秘密の列の値は "+masked+" に伏せています。")
	}

	logs := report.Sheet{Name: "ログ", Lead: "input/logs のファイルと、それぞれをどう扱ったかです。",
		Header: []string{"ファイル", "パック", "ホスト名", "設定の行", "読めなかった行", "状態", "注意"}}
	for _, l := range o.Logs {
		state, mark := "比べました", ""
		switch l.State {
		case LogSkipped:
			state, mark = "比べていません: "+l.Why, report.MarkWarn
		case LogFailed:
			state, mark = "読めませんでした: "+l.Why, report.MarkUnknown
		}
		logs.Rows = append(logs.Rows, []string{l.Path, l.Pack, l.Device, num(l.Lines), num(l.Unknown), state, strings.Join(l.Notes, " / ")})
		logs.Marks = append(logs.Marks, mark)
	}
	d.Sheets = append(d.Sheets, logs)

	rows := func(name, lead string, list []RowFinding, where bool) {
		head := []string{"表", "機器", "行", "内容"}
		if where {
			head = append(head, "ログの場所")
		}
		s := report.Sheet{Name: name, Lead: lead, Header: append(head, stateHeader...)}
		for _, x := range list {
			st, mark := stateCells(o.Verdicts, x.Finding)
			line := []string{e.tableTitle(x.Table), x.Device, e.rowLabel(x.Row), e.rowText(x.Row)}
			if where {
				line = append(line, originText(x.Row.Origins))
			}
			s.Rows = append(s.Rows, append(line, st...))
			s.Marks = append(s.Marks, mark)
		}
		d.Sheets = append(d.Sheets, s)
	}
	rows("機器の設定にない行", "パラメータブックから生成したコンフィグにはあって、機器の設定に無い行です（R-801）。", o.Missing, false)
	rows("ブックにない行", "機器の設定にあって、パラメータブックに無い行です（R-802）。", o.Surplus, true)

	cells := report.Sheet{Name: "値が違うセル", Lead: "両方にある行のうち、値が違うセルです（R-803）。機器がハッシュで持つ秘密は、比べていません（R-807）。",
		Header: append([]string{"表", "機器", "行", "列", "パラメータブック", "機器の設定", "備考", "ログの場所"}, stateHeader...)}
	for _, x := range o.Cells {
		t := e.Schema.Table(x.Table)
		col := t.Column(x.Column)
		a, b, note := check.Shown(col, x.A), check.Shown(col, x.B), ""
		if !e.mask() && col != nil && !col.Secret {
			a, b = cellText(x.A), cellText(x.B)
		}
		if x.Finding.RuleID == "R-807" {
			a, b, note = masked, masked, "機器がハッシュで持つ値（比べていません）"
		} else if x.OrderOnly {
			note = "並びだけが違います"
		}
		where := ""
		if x.B != nil {
			where = originText(x.B.Origins)
		}
		st, mark := stateCells(o.Verdicts, x.Finding)
		cells.Rows = append(cells.Rows, append([]string{e.tableTitle(x.Table), x.Device, e.rowLabel(x.ARow), col.Title.JA, a, b, note, where}, st...))
		cells.Marks = append(cells.Marks, mark)
	}
	d.Sheets = append(d.Sheets, cells)

	lines := func(name, lead, from string, list []ExtraLine) {
		s := report.Sheet{Name: name, Lead: lead, Header: append([]string{"機器", from, "行番号", "親の行", "行"}, stateHeader...), Code: []int{3, 4}}
		for _, x := range list {
			st, mark := stateCells(o.Verdicts, x.Finding)
			s.Rows = append(s.Rows, append([]string{x.Device, x.Path, num(x.Line), strings.Join(x.Parents, " > "), x.Text}, st...))
			s.Marks = append(s.Marks, mark)
		}
		d.Sheets = append(d.Sheets, s)
	}
	lines("追加コンフィグ", "追加コンフィグに書いた行のうち、機器の設定に見つからない行です（R-804）。行番号は、生成したコンフィグの行です。", "書いた所", o.Extras)
	lines("機器にだけある読めなかった行", "機器の設定のうち、パックが読めず、追加コンフィグにも無い行です（R-805）。行番号は、ログの行です。", "ログ", o.Unknown)

	if len(o.Strays) > 0 {
		s := report.Sheet{Name: "機器に当たらないログ", Lead: "ホスト名が案件の機器に当たらない・メーカーが違う、というログです（R-806）。", Header: append([]string{"機器", "内容"}, stateHeader...)}
		for _, f := range o.Strays {
			st, mark := stateCells(o.Verdicts, f)
			s.Rows = append(s.Rows, append([]string{f.Device, f.Message}, st...))
			s.Marks = append(s.Marks, mark)
		}
		d.Sheets = append(d.Sheets, s)
	}
	return d
}

// cellText gives the values of a cell as they are.
func cellText(c *model.Cell) string {
	if c == nil {
		return ""
	}
	return strings.Join(c.Values, ", ")
}
