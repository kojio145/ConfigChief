package ops

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/render"
	"github.com/kojio145/configchief/internal/report"
)

// The words of the check results for what a check came to.
const (
	wordOK      = "OK"
	wordFound   = "指摘あり"
	wordSettled = "承認済み"
	wordNothing = "対象なし"
	wordNotRun  = "未実施"
	wordPartly  = "一部未実施"
)

var genStateWords = map[string]string{
	GenWritten: "書き出し済み",
	GenFailed:  "失敗（文面がそろっていません）",
	GenBlocked: "生成していません",
	GenChecked: "チェックのみ",
	GenSkipped: "今回は対象外",
}

// GenStateWord is the word for what the generation did for a device.
func GenStateWord(state string) string { return genStateWords[state] }

// RuleStat is what one check came to (appendix C 5).
type RuleStat struct {
	Rule     check.Rule
	Severity string // as the case set it
	State    string
	// Unit is what Total, OK, Open and Approved count: devices for a check
	// that looks at a device as a whole, cells and rows for the others.
	Unit                      string
	Total, OK, Open, Approved int
	Findings                  int // the findings, approved or not
	Note                      string
	// Under is, for a rule of the case, the check its findings go under
	// (G-610 to G-612); "" for a check of the tool.
	Under string
}

// ruleRan tells whether a check looked at a device: "" if it did, or why not.
func (o *GenerateOutcome) ruleRan(r check.Rule, host string) (ran bool, why string) {
	return o.ran(r, o.Disabled[r.ID], host)
}

func (o *GenerateOutcome) ran(r check.Rule, disabled bool, host string) (ran bool, why string) {
	if disabled {
		return false, "設定（checks.disable）で止めています"
	}
	looked, skipped := o.Tally.Looked(r.ID), o.Tally.Skipped(r.ID)
	if why, ok := skipped[host]; ok && looked[host] == 0 {
		return false, why
	}
	if len(looked) == 0 {
		if why, ok := skipped[""]; ok {
			return false, why
		}
		return false, "この版では行っていません"
	}
	if r.PerDevice && looked[host] == 0 {
		return false, "この機器は見ていません"
	}
	return true, ""
}

// RuleStats gives what each check came to, in the order of appendix C, and
// after them what each rule of the case came to. Every check is there: one
// that did not run says so, with the reason.
func (e *Env) RuleStats(o *GenerateOutcome) []RuleStat {
	type count struct{ open, approved, all int }
	byRule := map[string]*count{}
	byDevice := map[string]map[string]*count{}
	for _, f := range o.Items {
		for _, id := range []string{f.RuleID, f.CaseRule} {
			if id == "" {
				continue
			}
			if byRule[id] == nil {
				byRule[id] = &count{}
				byDevice[id] = map[string]*count{}
			}
			if byDevice[id][f.Device] == nil {
				byDevice[id][f.Device] = &count{}
			}
			for _, c := range []*count{byRule[id], byDevice[id][f.Device]} {
				c.all++
				if f.Approval != nil {
					c.approved++
				} else {
					c.open++
				}
			}
		}
	}

	// A rule of the case is counted as a check of its own; what the case
	// says about the check it goes under (G-610 to G-612) holds for it.
	type entry struct {
		r     check.Rule
		under string
	}
	var entries []entry
	for _, r := range check.Rules() {
		entries = append(entries, entry{r: r})
	}
	for _, cr := range o.CaseRules {
		entries = append(entries, entry{check.Rule{ID: cr.ID, Severity: cr.Severity, Title: cr.Title, PerDevice: cr.OfLines()}, cr.CheckID()})
	}

	var stats []RuleStat
	for _, en := range entries {
		r, said := en.r, en.r.ID
		if en.under != "" {
			said = en.under
		}
		disabled := o.Disabled[said]
		s := RuleStat{Rule: r, Severity: r.Severity, Unit: "件", Under: en.under}
		if sev, ok := e.Proj.Config.Checks.Severity[said]; ok {
			s.Severity = sev
		}
		looked, skipped := o.Tally.Looked(r.ID), o.Tally.Skipped(r.ID)
		c := byRule[r.ID]
		if c == nil {
			c = &count{}
		}
		s.Findings = c.all

		// The devices the check did not look at, by reason.
		reasons := map[string]int{}
		for host, why := range skipped {
			if host != "" && looked[host] == 0 {
				reasons[why]++
			}
		}
		if r.PerDevice {
			s.Unit = "台"
			for host, n := range looked {
				if host == "" || n == 0 {
					continue
				}
				s.Total++
				switch d := byDevice[r.ID][host]; {
				case d == nil:
					s.OK++
				case d.open > 0:
					s.Open++
				default:
					s.Approved++
				}
			}
		} else {
			for _, n := range looked {
				s.Total += n
			}
			s.Open, s.Approved = c.open, c.approved
			if s.OK = s.Total - c.all; s.OK < 0 {
				s.OK = 0
			}
		}

		switch _, why := o.ran(r, disabled, ""); {
		case disabled:
			s.State, s.Note = wordNotRun, why
			s.Total, s.OK, s.Open, s.Approved, s.Findings = 0, 0, 0, 0, 0
		case len(looked) == 0:
			s.State, s.Note = wordNotRun, why
		case c.open > 0:
			s.State = wordFound
		case c.approved > 0:
			s.State = wordSettled
		case s.Total == 0:
			s.State = wordNothing
		default:
			s.State = wordOK
		}
		_, global := skipped[""]
		if len(reasons) > 0 && !disabled && !(len(looked) == 0 && global) {
			if len(looked) == 0 || s.Total == 0 && c.all == 0 {
				s.State, s.Note = wordNotRun, ""
			} else {
				s.State += "・" + wordPartly
			}
			var whys []string
			for why := range reasons {
				whys = append(whys, why)
			}
			sort.Strings(whys)
			for _, why := range whys {
				s.Note = join(s.Note, fmt.Sprintf("%d 台は未実施: %s", reasons[why], why))
			}
		}
		stats = append(stats, s)
	}
	return stats
}

func join(a, b string) string {
	if a == "" {
		return b
	}
	return a + " / " + b
}

// Where tells the place a finding is about, for one line of text: the
// table and the row, and the line of the configuration if there is one.
func (e *Env) Where(f check.Finding) string {
	var parts []string
	if f.Ref.Table != "" {
		s := e.tableTitle(f.Ref.Table)
		if k := e.keyLabel(f.Ref.Table, f.Ref.Key); k != "" {
			s += " " + k
		}
		parts = append(parts, s)
	}
	if f.Line > 0 {
		parts = append(parts, fmt.Sprintf("%d 行目", f.Line))
	}
	return strings.Join(parts, "、")
}

// keyLabel shows the key of a finding ("name=Gi0/1,vrf=") by the names the
// book shows.
func (e *Env) keyLabel(table, key string) string {
	t := e.Schema.Table(table)
	if t == nil || key == "" {
		return key
	}
	var parts []string
	for _, p := range strings.Split(key, ",") {
		name, val, ok := strings.Cut(p, "=")
		if c := t.Column(name); ok && c != nil {
			if val != "" {
				parts = append(parts, c.Title.In(e.lang())+"="+val)
			} else {
				parts = append(parts, "")
			}
			continue
		}
		// A comma inside a value.
		if n := len(parts); n > 0 {
			parts[n-1] += "," + p
		} else {
			parts = append(parts, p)
		}
	}
	var out []string
	for _, p := range parts {
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, ", ")
}

// lineText gives a line of a generated configuration for a report.
func (e *Env) lineText(o *render.Output, line int) string {
	if o == nil || line < 1 || line > len(o.Lines) {
		return ""
	}
	if e.mask() {
		return o.Shown(line)
	}
	return o.Lines[line-1].Text
}

// cellRefText names the cells a line was made from, with their values.
func (e *Env) cellRefText(refs []render.CellRef) string {
	var parts []string
	seen := map[string]bool{}
	for _, v := range refs {
		cell := v.Row.Cells[v.Column]
		s := e.tableTitle(v.Row.Table)
		if l := e.rowLabel(v.Row); l != "" {
			s += "「" + l + "」"
		}
		s += " の " + e.columnTitle(v.Row.Table, v.Column) + " = " + e.cellText(v.Row.Table, v.Column, cell)
		switch at := check.CellAt(v.Row, v.Column); {
		case at != "":
			s += "（" + at + "）"
		case cell != nil && len(cell.Origins) > 0 && cell.Origins[0].Kind == "default":
			s += "（既定値）"
		}
		if !seen[s] {
			seen[s] = true
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, "; ")
}

// MigrationApprovalNote is what the results of the generation say about an
// approval of a check of a migration.
const MigrationApprovalNote = "移行のチェックの承認です。当たった件数は、比較のレポートの「承認」に出ます"

// SourceApprovalNote is what the results of the generation say about an
// approval of a check of the customer's lists.
const SourceApprovalNote = "資料のチェックの承認です。当たった件数は、お客様の一覧のレポートの「承認」に出ます"

// SourceApproval tells whether an approval is for a check of the customer's
// lists.
func SourceApproval(a project.Approval) bool { return sourceApproval(a) }

// MigrationApproval tells whether an approval is for a check of a migration.
func MigrationApproval(a project.Approval) bool { return migrationApproval(a) }

func approvalTarget(a project.Approval) string {
	if a.Scope == nil {
		return "指摘 " + a.Finding
	}
	s := "範囲: " + a.Scope.Rule
	if a.Scope.Table != "" {
		s += " / " + a.Scope.Table
	}
	if a.Scope.Column != "" {
		s += "." + a.Scope.Column
	}
	return s + " / 機器 " + a.Scope.Devices
}

// checkDoc builds the check results (spec 13.3, appendix C 5).
func (e *Env) checkDoc(o *GenerateOutcome, now time.Time) *report.Doc {
	d := &report.Doc{Title: "チェック結果表", Cover: "表紙", Facts: e.facts(now)}
	what := "生成とチェック"
	if o.Options.CheckOnly {
		what = "チェックのみ（コンフィグは書き出していません）"
	}
	d.Facts = append(d.Facts, report.Pair{Label: "実施内容", Value: what})
	if o.Options.Device != "" {
		d.Facts = append(d.Facts, report.Pair{Label: "対象の機器", Value: o.Options.Device + "（値のチェックは全台を見ています）"})
	}
	d.Facts = append(d.Facts, report.Pair{Label: "出力フォルダ", Value: o.Dir})
	used := map[string]bool{}
	for _, dev := range o.Devices {
		if p := e.Packs[dev.Pack]; p != nil && !used[dev.Pack] {
			used[dev.Pack] = true
		}
	}
	var packs []string
	for id := range used {
		packs = append(packs, id+" "+e.Packs[id].Version)
	}
	sort.Strings(packs)
	d.Facts = append(d.Facts, report.Pair{Label: "メーカーパック", Value: strings.Join(packs, ", ")})
	for _, in := range o.Inputs {
		d.Facts = append(d.Facts, report.Pair{Label: "入力 " + in.Path, Value: "SHA-256 " + in.SHA256})
	}

	stats := e.RuleStats(o)
	c := o.Count()
	states := map[string]int{}
	runs := map[string]*DeviceRun{}
	for i := range o.Devices {
		states[o.Devices[i].State]++
		runs[o.Devices[i].Host] = &o.Devices[i]
	}
	ran, notRun := 0, 0
	for _, s := range stats {
		switch {
		case s.Under != "": // a rule of the case: counted under its check
		case s.State == wordNotRun:
			notRun++
		default:
			ran++
		}
	}
	d.Counts = []report.Pair{{Label: "機器", Value: num(len(o.Devices))}}
	if !o.Options.CheckOnly {
		d.Counts = append(d.Counts, report.Pair{Label: "コンフィグを書き出した機器", Value: num(states[GenWritten])})
	} else {
		d.Counts = append(d.Counts, report.Pair{Label: "チェックだけ行った機器", Value: num(states[GenChecked])})
	}
	d.Counts = append(d.Counts,
		report.Pair{Label: "文面がそろわなかった機器", Value: num(states[GenFailed])},
		report.Pair{Label: "値の誤りで生成していない機器", Value: num(states[GenBlocked])})
	if states[GenSkipped] > 0 {
		d.Counts = append(d.Counts, report.Pair{Label: "今回は対象外の機器", Value: num(states[GenSkipped])})
	}
	d.Counts = append(d.Counts,
		report.Pair{Label: "未承認のエラー", Value: num(c.Errors)},
		report.Pair{Label: "未承認の警告", Value: num(c.Warnings)},
		report.Pair{Label: "承認済みの指摘", Value: num(c.Approved)},
		report.Pair{Label: "情報", Value: num(c.Info)},
		report.Pair{Label: "パラメータブックの注意", Value: num(len(o.BookNotes))},
		report.Pair{Label: "実施したチェック", Value: num(ran)},
		report.Pair{Label: "未実施のチェック", Value: num(notRun)})

	if o.Unverified != "" {
		d.Notes = append(d.Notes, "未承認の指摘が残っているため、出力フォルダに UNVERIFIED.txt を置きました。")
	}
	if notRun > 0 {
		d.Notes = append(d.Notes, fmt.Sprintf("実施していないチェックが %d 項目あります。「チェック項目」のシートに、理由つきで未実施と書いています（OK ではありません）。", notRun))
	}
	for _, n := range o.BookNotes {
		d.Notes = append(d.Notes, "パラメータブック: "+n)
	}
	d.Notes = append(d.Notes, o.Notes...)
	if e.mask() {
		d.Notes = append(d.Notes, "秘密の列の値は伏せています（指摘の根拠は「伏せ字 #…」、コンフィグの文面は "+render.Masked+"）。書き出したコンフィグのファイルには、値がそのまま入っています。")
	} else {
		d.Notes = append(d.Notes, "指摘の根拠にある秘密の値は「伏せ字 #…」で示します。「パラメータ突合」の文面と値には、秘密の値がそのまま入っています（secrets.mask_in_reports: false）。")
	}

	// By device.
	perDev := report.Sheet{Name: "機器別", Lead: "機器ごとの結果です。チェックの数は、チェック項目（ID）の数で数えています。",
		Header: []string{"機器", "メーカーパック", "状態", "ファイル", "未承認のエラー", "未承認の警告", "承認済み", "情報",
			"値のチェック: OK", "値のチェック: 指摘あり", "値のチェック: 未実施", "生成物のチェック: OK", "生成物のチェック: 指摘あり", "生成物のチェック: 未実施", "備考"}}
	type devCount struct {
		c     Count
		rules map[string]bool
	}
	devs := map[string]*devCount{}
	for _, f := range o.Items {
		dc := devs[f.Device]
		if dc == nil {
			dc = &devCount{rules: map[string]bool{}}
			devs[f.Device] = dc
		}
		dc.rules[f.RuleID] = true
		switch {
		case f.Severity == check.Info:
			dc.c.Info++
		case f.Approval != nil:
			dc.c.Approved++
		case f.Severity == check.Error:
			dc.c.Errors++
		default:
			dc.c.Warnings++
		}
	}
	devRow := func(host, label, pack, state, file, why string) {
		dc := devs[host]
		if dc == nil {
			dc = &devCount{rules: map[string]bool{}}
		}
		n := map[string]*[3]int{"V": {}, "G": {}}
		for _, r := range check.Rules() {
			k := n[check.Kind(r.ID)]
			if k == nil {
				continue
			}
			switch ran, _ := o.ruleRan(r, host); {
			case host == "" && r.PerDevice:
			case !ran:
				k[2]++
			case dc.rules[r.ID]:
				k[1]++
			default:
				k[0]++
			}
		}
		perDev.Rows = append(perDev.Rows, []string{label, pack, state, file,
			num(dc.c.Errors), num(dc.c.Warnings), num(dc.c.Approved), num(dc.c.Info),
			num(n["V"][0]), num(n["V"][1]), num(n["V"][2]), num(n["G"][0]), num(n["G"][1]), num(n["G"][2]), why})
		mark := ""
		switch {
		case dc.c.Errors > 0 || state == genStateWords[GenFailed] || state == genStateWords[GenBlocked]:
			mark = report.MarkUnknown
		case dc.c.Warnings > 0:
			mark = report.MarkWarn
		}
		perDev.Marks = append(perDev.Marks, mark)
	}
	if devs[""] != nil {
		devRow("", "（案件全体）", "", "", "", "機器を 1 台に決められない指摘です")
	}
	for _, dev := range o.Devices {
		devRow(dev.Host, dev.Host, dev.Pack, genStateWords[dev.State], dev.File, dev.Why)
	}

	// By check.
	items := report.Sheet{Name: "チェック項目", Lead: "この版が持つチェックのすべてと、その後に案件のルール（rules/）です。実施していないものは「未実施」と理由を書いています。",
		Header: []string{"ID", "種類", "重大度", "内容", "状態", "単位", "対象", "OK", "指摘", "承認済み", "指摘の件数", "備考"}}
	for _, s := range stats {
		sev := check.SeverityTitle(s.Severity)
		if s.Severity == "" {
			sev = "ルールごと"
		}
		kind := check.KindTitle(check.Kind(s.Rule.ID))
		if s.Under != "" {
			kind = "案件のルール（" + s.Under + "）"
		}
		row := []string{s.Rule.ID, kind, sev, s.Rule.Title, s.State, s.Unit,
			num(s.Total), num(s.OK), num(s.Open), num(s.Approved), num(s.Findings), s.Note}
		mark := ""
		switch {
		case s.State == wordNotRun:
			row[6], row[7], row[8], row[9], row[10] = "", "", "", "", ""
			mark = report.MarkIgnored
		case s.Open > 0 && s.Severity == check.Error:
			mark = report.MarkUnknown
		case s.Open > 0 && s.Severity == check.Warning:
			mark = report.MarkWarn
		}
		items.Rows = append(items.Rows, row)
		items.Marks = append(items.Marks, mark)
	}

	// The findings.
	found := report.Sheet{Name: "指摘", Lead: "指摘の一覧です。承認は、指紋に結び付きます（根拠の値が変わると外れます）。",
		Header: []string{"ID", "重大度", "機器", "表", "行", "列", "ブックのセル", "コンフィグの行", "文面", "内容", "根拠", "承認", "指紋"}, Code: []int{8}}
	for _, f := range o.Items {
		var ev []string
		for _, kv := range f.Evidence {
			ev = append(ev, kv.Name+"="+kv.Value)
		}
		line, text := "", ""
		if f.Line > 0 {
			line = num(f.Line)
			if run := runs[f.Device]; run != nil {
				text = e.lineText(run.Output, f.Line)
			}
		}
		col := ""
		if f.Ref.Column != "" {
			col = e.columnTitle(f.Ref.Table, f.Ref.Column)
		}
		table := ""
		if f.Ref.Table != "" {
			table = e.tableTitle(f.Ref.Table)
		}
		state, mark := "未承認", ""
		switch {
		case f.Approval != nil:
			state = "承認済み（" + f.Approval.ID + " " + f.Approval.By + "）"
		case f.Severity == check.Info:
			state = "—"
		case !check.Approvable(f.RuleID):
			state, mark = "未承認（承認できません。値や文面を直します）", report.MarkUnknown
		case f.Severity == check.Error:
			mark = report.MarkUnknown
		default:
			mark = report.MarkWarn
		}
		found.Rows = append(found.Rows, []string{f.RuleID, check.SeverityTitle(f.Severity), f.Device, table,
			e.keyLabel(f.Ref.Table, f.Ref.Key), col, f.At, line, text, f.Message, strings.Join(ev, "; "), state, f.Key})
		found.Marks = append(found.Marks, mark)
	}

	// The approvals.
	appr := report.Sheet{Name: "承認", Lead: "state/approvals.yaml の承認と、今回それぞれが当たった指摘の件数です。",
		Header: []string{"ID", "対象", "チェック", "機器", "理由", "承認者", "日時", "当たった件数", "備考"}}
	for _, a := range o.Approvals {
		rule, dev := a.Rule, a.Device
		if a.Scope != nil {
			rule, dev = a.Scope.Rule, a.Scope.Devices
		}
		note, mark, hits := o.Refused[a.ID], "", num(o.Hits[a.ID])
		switch {
		case note == "" && o.Hits[a.ID] == 0 && migrationApproval(a):
			// A check of a migration: its findings come from the import and
			// the comparison, not from the generation (appendix C 3.2).
			note, hits = MigrationApprovalNote, "—"
		case note == "" && o.Hits[a.ID] == 0 && sourceApproval(a):
			note, hits = SourceApprovalNote, "—"
		case note == "" && o.Hits[a.ID] == 0 && appliedApproval(a):
			note, hits = AppliedApprovalNote, "—"
		case note == "" && o.Hits[a.ID] == 0:
			note, mark = "今回、当たる指摘はありません（指摘がなくなったか、根拠の値が変わりました）", report.MarkWarn
		case note != "":
			mark = report.MarkWarn
		}
		appr.Rows = append(appr.Rows, []string{a.ID, approvalTarget(a), rule, dev, a.Reason, a.By, a.At, hits, note})
		appr.Marks = append(appr.Marks, mark)
	}

	// Line by line: what each line was made from.
	trace := report.Sheet{Name: "パラメータ突合", Lead: "生成したコンフィグの 1 行ごとに、元になった値を示します。「手書き」は追加コンフィグ（extra/）の行です。",
		Header: []string{"機器", "行番号", "文面", "由来", "値", "条件"}, Code: []int{2}}
	for _, dev := range o.Devices {
		if dev.Output == nil {
			continue
		}
		for i, l := range dev.Output.Lines {
			if strings.TrimSpace(l.Text) == "" {
				continue
			}
			from := "パック（" + l.Template
			if l.Extra {
				from = "手書き（" + l.Template
			}
			if l.TLine > 0 {
				from += " の " + num(l.TLine) + " 行目"
			}
			from += "）"
			mark := ""
			if l.Extra {
				mark = report.MarkWarn
			}
			trace.Rows = append(trace.Rows, []string{dev.Host, num(i + 1), e.lineText(dev.Output, i+1), from, e.cellRefText(l.Values), e.cellRefText(l.Conds)})
			trace.Marks = append(trace.Marks, mark)
		}
	}

	d.Sheets = []report.Sheet{perDev, items, found, appr, trace}
	return d
}
