package ops

import (
	"fmt"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/logscan"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/pack"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/report"
)

// The check of the logs of the work (spec 11.6, appendix C 2.2): the
// commands a device did not take, and whether what was typed is the
// configuration the book gives.

// LoadLog is one file of input/logs as the check of the work saw it.
type LoadLog struct {
	Path   string
	Pack   string
	Device string // the device whose configuration the commands fit best
	State  string // LogCompared, LogSkipped or LogFailed
	Why    string
	// Commands counts the commands typed in a configuration mode; Matched
	// those of them that are lines of the generated configuration.
	Commands, Matched int
	Hosts             []string // the host names of the prompts
}

// LoadLine is a command of a log the device answered (L-701, L-704).
type LoadLine struct {
	Device  string
	Path    string
	Line    int // in the log
	Mode    string
	Command string   // as it may be shown: secret values are hidden where they are known
	Replies []string // what the device wrote back
	// ConfigLine is the line of the generated configuration (0 = the command
	// is not in it).
	ConfigLine int
	Finding    check.Finding
}

// LoadOutcome is what one run of the check found.
type LoadOutcome struct {
	Logs     []LoadLog
	Rejected []LoadLine      // L-701
	Warned   []LoadLine      // L-704
	Gaps     []check.Finding // L-702, L-703, L-705
	Verdicts *Verdicts
	Report   Files
	Notes    []string
}

// Left counts what a person has to look at.
func (o *LoadOutcome) Left() int {
	c := o.Verdicts.Count()
	n := c.Errors + c.Warnings
	for _, l := range o.Logs {
		if l.State == LogFailed {
			n++
		}
	}
	return n
}

// noCommands tells why a log is not checked.
const noCommands = "設定モードのコマンドがありません（投入のログではありません）"

// configLines gives the lines of a generated configuration a person types:
// without the comments and the blank lines. The second result maps each to
// its line in the configuration.
func configLines(p *pack.Pack, run *DeviceRun) (texts []string, at []int) {
	for i, l := range run.Output.Lines {
		t := strings.TrimSpace(l.Text)
		if t == "" || isComment(p, t) {
			continue
		}
		texts = append(texts, t)
		at = append(at, i+1)
	}
	return texts, at
}

func isComment(p *pack.Pack, text string) bool {
	for _, c := range p.Options.CommentPrefixes {
		if strings.HasPrefix(text, c) {
			return true
		}
	}
	return false
}

// typed gives the commands of a log that were typed in a configuration mode
// and are no comment.
func typed(p *pack.Pack, log *logscan.Log) []*logscan.Command {
	var out []*logscan.Command
	for i := range log.Commands {
		c := &log.Commands[i]
		if c.Config() && c.Text != "" && !isComment(p, c.Text) {
			out = append(out, c)
		}
	}
	return out
}

// LoadCheck reads the logs of input/logs and looks at the commands typed
// into each device. It writes the report.
func (e *Env) LoadCheck(now time.Time) (*LoadOutcome, error) { return e.loadCheck(now, true) }

func (e *Env) loadCheck(now time.Time, write bool) (*LoadOutcome, error) {
	paths, err := e.Proj.Inputs("logs")
	if err != nil {
		return nil, err
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("input/logs にファイルがありません。コンフィグを流したときの端末ログを置いてください")
	}
	gen, err := e.Generate(GenOptions{Dry: true}, now)
	if err != nil {
		return nil, err
	}
	_, bk, err := e.readBook()
	if err != nil {
		return nil, err
	}
	out := &LoadOutcome{}
	var findings []check.Finding
	rule := func(id, device string, ref model.Ref, msg string, ev ...model.KV) check.Finding {
		r, _ := check.Lookup(id)
		f := check.New(id, r.Severity, device, ref, msg, ev...)
		findings = append(findings, f)
		return f
	}
	// The logs as each pack reads them: the prompts differ by vendor.
	scans := map[string]map[string]*logscan.Log{}
	for i, path := range paths {
		if e.Tick != nil {
			if err := e.Tick(i, len(paths), path); err != nil {
				return nil, err
			}
		}
		l := LoadLog{Path: path, State: LogFailed}
		text, err := project.ReadText(e.Proj.Path(path))
		if err != nil {
			l.Why = err.Error()
			out.Logs = append(out.Logs, l)
			continue
		}
		scans[path] = map[string]*logscan.Log{}
		any := false
		for _, id := range e.PackIDs {
			if p := e.Packs[id]; p.Session != nil {
				log := logscan.Scan(p.Session, p.Messages, text)
				scans[path][id] = log
				any = any || len(typed(p, log)) > 0
			}
		}
		if !any {
			l.State, l.Why = LogSkipped, noCommands
			out.Logs = append(out.Logs, l)
			continue
		}
		// The device whose configuration the commands fit best; with the same
		// number of lines, the one a prompt names.
		var best *DeviceRun
		bestHit, bestNamed := 0, false
		for d := range gen.Devices {
			run := &gen.Devices[d]
			p := e.Packs[run.Pack]
			log := scans[path][run.Pack]
			if p == nil || log == nil || run.Output == nil || run.Output.Failed {
				continue
			}
			lines, _ := configLines(p, run)
			have := map[string]bool{}
			for _, t := range lines {
				have[t] = true
			}
			hit := 0
			for _, c := range typed(p, log) {
				if have[c.Text] {
					hit++
				}
			}
			named := false
			for _, h := range log.Hosts() {
				named = named || h == run.Host
			}
			if hit > bestHit || (hit == bestHit && hit > 0 && named && !bestNamed) {
				best, bestHit, bestNamed = run, hit, named
			}
		}
		if best == nil {
			l.Why = "流したコマンドが、今のパラメータブックから生成したどの機器のコンフィグにも当たりません"
			out.Logs = append(out.Logs, l)
			continue
		}
		p := e.Packs[best.Pack]
		log := scans[path][best.Pack]
		l.Pack, l.Device, l.State, l.Hosts = best.Pack, best.Host, LogCompared, log.Hosts()
		e.loadDevice(out, rule, &l, p, best, log)
		out.Logs = append(out.Logs, l)
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
	if out.Report, err = e.writeReport(now, "load_logs", e.loadDoc(out, now)); err != nil {
		return nil, err
	}
	c := out.Verdicts.Count()
	e.Proj.Log(now, "logs: ログ %d 個、弾かれた行 %d、機器のメッセージ %d（未承認のエラー %d、警告 %d）、レポート %s",
		len(out.Logs), len(out.Rejected), len(out.Warned), c.Errors, c.Warnings, out.Report.XLSX)
	return out, nil
}

// loadDevice looks at the commands of one log against the configuration of
// its device.
func (e *Env) loadDevice(out *LoadOutcome, rule func(id, device string, ref model.Ref, msg string, ev ...model.KV) check.Finding,
	l *LoadLog, p *pack.Pack, run *DeviceRun, log *logscan.Log) {
	host := run.Host
	lines, at := configLines(p, run)
	where := map[string][]int{} // the text of a line → its places in lines
	for i, t := range lines {
		where[t] = append(where[t], i)
	}
	covered := make([]bool, len(lines))
	cursor := 0
	var late, strange []string // typed out of order; not in the configuration
	place := map[int]int{}     // the line of a command in the log → its place in lines
	for _, c := range typed(p, log) {
		l.Commands++
		found := -1
		for _, i := range where[c.Text] {
			if i >= cursor {
				found = i
				break
			}
		}
		switch {
		case found >= 0:
			cursor = found + 1
		case len(where[c.Text]) > 0:
			found = where[c.Text][0]
			late = append(late, c.Text)
		default:
			strange = append(strange, c.Text)
			continue
		}
		covered[found] = true
		place[c.Line] = found
		l.Matched++
	}

	// What the device said back.
	for i := range log.Commands {
		c := &log.Commands[i]
		for _, level := range []string{logscan.Rejected, logscan.Unknown, logscan.Warning} {
			if c.Told(level) == nil {
				continue
			}
			x := LoadLine{Device: host, Path: l.Path, Line: c.Line, Mode: c.Mode, Command: c.Text}
			var ref model.Ref
			if i, ok := place[c.Line]; ok {
				x.ConfigLine = at[i]
				x.Command = strings.TrimSpace(run.Output.Shown(x.ConfigLine))
				for _, v := range run.Output.Lines[x.ConfigLine-1].Values {
					if t := e.Schema.Table(v.Row.Table); t != nil {
						ref = check.RefOf(t, v.Row, v.Column)
						break
					}
				}
			}
			for _, r := range c.Replies {
				x.Replies = append(x.Replies, strings.TrimSpace(r.Text))
			}
			ev := []model.KV{{Name: "コマンド", Value: x.Command}, {Name: "モード", Value: c.Mode},
				{Name: "機器の返事", Value: strings.Join(x.Replies, " / ")}, {Name: "ログ", Value: fmt.Sprintf("%s の %d 行目", l.Path, c.Line)}}
			if x.ConfigLine > 0 {
				ev = append(ev, model.KV{Name: "コンフィグの行", Value: num(x.ConfigLine)})
			}
			switch level {
			case logscan.Rejected:
				x.Finding = rule("L-701", host, ref, "機器がコマンドを受け付けませんでした", ev...)
				x.Finding.Line = x.ConfigLine
				out.Rejected = append(out.Rejected, x)
			case logscan.Unknown:
				x.Finding = rule("L-701", host, ref, "機器がメッセージを返しました。コマンドが弾かれたかどうかを、メーカーパックは知りません（返事を読んで確かめてください）", ev...)
				x.Finding.Line = x.ConfigLine
				out.Rejected = append(out.Rejected, x)
			default:
				x.Finding = rule("L-704", host, ref, "機器がメッセージを返しました", ev...)
				x.Finding.Line = x.ConfigLine
				out.Warned = append(out.Warned, x)
			}
			break
		}
	}

	first := func(list []string) string {
		if len(list) > 5 {
			return strings.Join(list[:5], " / ") + " …"
		}
		return strings.Join(list, " / ")
	}
	if n := len(late) + len(strange); n > 0 {
		out.Gaps = append(out.Gaps, rule("L-702", host, model.Ref{},
			fmt.Sprintf("ログのコマンドが、今のパラメータブックから生成したコンフィグと合いません（コンフィグにない行 %d、順番が違う行 %d。別の版を流した疑い）", len(strange), len(late)),
			model.KV{Name: "ログ", Value: l.Path}, model.KV{Name: "コンフィグにない行", Value: first(strange)}, model.KV{Name: "順番が違う行", Value: first(late)}))
	}
	var missing []string
	for i, ok := range covered {
		if !ok {
			missing = append(missing, strings.TrimSpace(run.Output.Shown(at[i])))
		}
	}
	if len(missing) > 0 {
		out.Gaps = append(out.Gaps, rule("L-703", host, model.Ref{},
			fmt.Sprintf("生成したコンフィグの %d 行のうち %d 行が、ログにありません（途中で切れた疑い）", len(lines), len(missing)),
			model.KV{Name: "ログ", Value: l.Path}, model.KV{Name: "ログにない行", Value: first(missing)}))
	}
	named := false
	for _, h := range l.Hosts {
		named = named || h == host
	}
	if !named {
		out.Gaps = append(out.Gaps, rule("L-705", host, model.Ref{},
			fmt.Sprintf("ログのプロンプトのホスト名（%s）が、流したコンフィグの機器 %s と合いません（機器の取り違えの疑い）", strings.Join(l.Hosts, "、"), host),
			model.KV{Name: "ログ", Value: l.Path}, model.KV{Name: "プロンプト", Value: strings.Join(l.Hosts, "、")}))
	}
}

// loadDoc builds the report of the check (spec 11.6, 13.5).
func (e *Env) loadDoc(o *LoadOutcome, now time.Time) *report.Doc {
	d := &report.Doc{Title: "投入ログのチェック", Facts: e.facts(now)}
	c := o.Verdicts.Count()
	checked := 0
	for _, l := range o.Logs {
		if l.State == LogCompared {
			checked++
		}
	}
	d.Counts = []report.Pair{
		{Label: "ログ", Value: num(len(o.Logs))},
		{Label: "調べた機器", Value: num(checked)},
		{Label: "弾かれた行（L-701）", Value: num(len(o.Rejected))},
		{Label: "機器のメッセージ（L-704）", Value: num(len(o.Warned))},
		{Label: "ログとコンフィグの食い違い（L-702・L-703・L-705）", Value: num(len(o.Gaps))},
		{Label: "未承認のエラー", Value: num(c.Errors)},
		{Label: "未承認の警告", Value: num(c.Warnings)},
		{Label: "承認済みの指摘", Value: num(c.Approved)},
	}
	d.Notes = append(d.Notes,
		"input/logs の端末ログから、プロンプトの後ろに打たれたコマンドと、機器が返した行を取り出しています。機器のメッセージの形は、メーカーパックの errors.yaml にあります。",
		"ログがどの機器のものかは、設定モードで打たれたコマンドが、今のパラメータブックから生成したどの機器のコンフィグにいちばん多く当たるかで決めています。",
		"コマンドの文面は、生成したコンフィグの行に当たったものだけ、秘密の値を伏せています。当たらなかったコマンドは、打たれたまま出ます。")
	d.Notes = append(d.Notes, o.Notes...)

	logs := report.Sheet{Name: "ログ", Lead: "input/logs のファイルと、それぞれをどう扱ったかです。",
		Header: []string{"ファイル", "パック", "機器", "プロンプトのホスト名", "設定モードのコマンド", "うちコンフィグの行", "状態"}}
	for _, l := range o.Logs {
		state, mark := "調べました", ""
		switch l.State {
		case LogSkipped:
			state, mark = "調べていません: "+l.Why, report.MarkWarn
		case LogFailed:
			state, mark = "調べられませんでした: "+l.Why, report.MarkUnknown
		}
		logs.Rows = append(logs.Rows, []string{l.Path, l.Pack, l.Device, strings.Join(l.Hosts, "、"), num(l.Commands), num(l.Matched), state})
		logs.Marks = append(logs.Marks, mark)
	}
	d.Sheets = append(d.Sheets, logs)

	lines := func(name, lead string, list []LoadLine) {
		s := report.Sheet{Name: name, Lead: lead,
			Header: append([]string{"機器", "ログ", "ログの行", "モード", "コマンド", "機器の返事", "コンフィグの行", "設計データの場所"}, stateHeader...), Code: []int{4, 5}}
		for _, x := range list {
			st, mark := stateCells(o.Verdicts, x.Finding)
			cfg, ref := "", ""
			if x.ConfigLine > 0 {
				cfg = num(x.ConfigLine)
			}
			if r := x.Finding.Ref; r.Table != "" {
				ref = strings.TrimSpace(e.tableTitle(r.Table) + " " + r.Key + " " + r.Column)
			}
			s.Rows = append(s.Rows, append([]string{x.Device, x.Path, num(x.Line), x.Mode, x.Command, strings.Join(x.Replies, "\n"), cfg, ref}, st...))
			s.Marks = append(s.Marks, mark)
		}
		d.Sheets = append(d.Sheets, s)
	}
	lines("弾かれた行", "機器が受け付けなかったコマンドです（L-701）。「コンフィグの行」は、生成したコンフィグの行番号です。", o.Rejected)
	lines("機器のメッセージ", "機器が受け付けたうえで、メッセージを返したコマンドです（L-704）。", o.Warned)

	gaps := report.Sheet{Name: "ログとコンフィグの食い違い", Lead: "流したものが、今のパラメータブックから生成したコンフィグと違う疑いです（L-702・L-703・L-705）。",
		Header: append([]string{"機器", "内容", "根拠"}, stateHeader...)}
	for _, f := range o.Gaps {
		st, mark := stateCells(o.Verdicts, f)
		var ev []string
		for _, kv := range f.Evidence {
			if kv.Value != "" {
				ev = append(ev, kv.Name+": "+kv.Value)
			}
		}
		gaps.Rows = append(gaps.Rows, append([]string{f.Device, f.Message, strings.Join(ev, "\n")}, st...))
		gaps.Marks = append(gaps.Marks, mark)
	}
	d.Sheets = append(d.Sheets, gaps)
	return d
}
