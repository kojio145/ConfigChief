package ops

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/compare"
	"github.com/kojio145/configchief/internal/convert"
	"github.com/kojio145/configchief/internal/importer"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/report"
	"github.com/kojio145/configchief/internal/schema"
)

// masked stands for a secret value in a report (spec 6.7).
const masked = "***"

// The words of a report for the states of a line (spec 8.2) and for the
// judgment of an unhandled line (spec 8.4).
var stateWords = map[string]string{
	importer.Taken:   "取り込み済み",
	importer.Ignored: "無視",
	importer.Unknown: "未対応",
}

const notJudged = "未判定"

func (e *Env) mask() bool { return e.screen || e.Proj.Config.Secrets.MaskInReports }

func (e *Env) facts(now time.Time) []report.Pair {
	f := []report.Pair{
		{Label: "案件", Value: e.Proj.Config.Name},
		{Label: "実施日時", Value: now.Format("2006-01-02 15:04:05")},
	}
	if e.Version != "" {
		f = append(f, report.Pair{Label: "ツール", Value: "ConfigChief " + e.Version})
	}
	return f
}

// tableTitle and columnTitle give the names the book shows.
func (e *Env) tableTitle(table string) string {
	if t := e.Schema.Table(table); t != nil {
		return t.Title.In(e.lang())
	}
	return table
}

func (e *Env) columnTitle(table, col string) string {
	if t := e.Schema.Table(table); t != nil {
		if c := t.Column(col); c != nil {
			return c.Title.In(e.lang())
		}
	}
	return col
}

func (e *Env) secret(table, col string) bool {
	if t := e.Schema.Table(table); t != nil {
		if c := t.Column(col); c != nil {
			return c.Secret
		}
	}
	return false
}

// rowLabel names a row inside its table and device: the key columns but the
// device, as "title=value". A secret that is part of the key is masked.
func (e *Env) rowLabel(r *model.Row) string {
	devCol := ""
	if t := e.Schema.Table(r.Table); t != nil {
		devCol = t.DeviceColumn()
	}
	var parts []string
	for _, k := range r.Key {
		if k.Name == devCol || k.Value == "" {
			continue
		}
		v := k.Value
		if e.mask() && e.secret(r.Table, k.Name) {
			v = masked
		}
		parts = append(parts, e.columnTitle(r.Table, k.Name)+"="+v)
	}
	return strings.Join(parts, ", ")
}

// Place names the row and the column of a note of the mapping, by the names
// the book shows.
func (e *Env) Place(n convert.Note) (table, row, col string) {
	if n.Table == "" {
		return "", "", ""
	}
	if n.Row != nil {
		row = e.rowLabel(n.Row)
	}
	if n.Column != "" {
		col = e.columnTitle(n.Table, n.Column)
	}
	return e.tableTitle(n.Table), row, col
}

// cellText gives the value of a cell for a report.
func (e *Env) cellText(table, col string, c *model.Cell) string {
	if c == nil {
		return ""
	}
	if e.mask() && e.secret(table, col) {
		return masked
	}
	var sc *schema.Column
	if t := e.Schema.Table(table); t != nil {
		sc = t.Column(col)
	}
	return book.Show(sc, c, e.lang())
}

// rowText gives the filled cells of a row that are not part of its key.
func (e *Env) rowText(r *model.Row) string {
	key := map[string]bool{}
	for _, k := range r.Key {
		key[k.Name] = true
	}
	var parts []string
	for _, col := range r.Columns() {
		if key[col] {
			continue
		}
		if v := e.cellText(r.Table, col, r.Cells[col]); v != "" {
			parts = append(parts, e.columnTitle(r.Table, col)+"="+v)
		}
	}
	return strings.Join(parts, ", ")
}

// originText tells where values came from: the lines of a configuration, or
// the cells of the book. Long lists are cut.
func originText(origins []model.Origin) string {
	const most = 6
	var parts []string
	lines := map[string][]string{} // file → line numbers
	var files []string
	seen := map[string]bool{}
	for _, o := range origins {
		switch {
		case o.Kind == "default":
			if s := "既定値"; !seen[s] {
				seen[s] = true
				parts = append(parts, s)
			}
		case (o.Kind == "book" || o.Kind == "common") && o.Sheet != "":
			s := "ブックの「" + o.Sheet + "」" + o.Addr
			if !seen[s] {
				seen[s] = true
				parts = append(parts, s)
			}
		case o.Kind == "customer":
			s := o.File + " の「" + o.Sheet + "」" + o.Addr
			if !seen[s] {
				seen[s] = true
				parts = append(parts, s)
			}
		case o.Kind == "book":
			if s := "ブックの「機器対応」"; !seen[s] {
				seen[s] = true
				parts = append(parts, s)
			}
		case o.File != "":
			n := strconv.Itoa(o.Line)
			if k := o.File + "\x00" + n; !seen[k] {
				seen[k] = true
				if lines[o.File] == nil {
					files = append(files, o.File)
				}
				lines[o.File] = append(lines[o.File], n)
			}
		}
	}
	for _, f := range files {
		ns := lines[f]
		more := ""
		if len(ns) > most {
			ns, more = ns[:most], " ほか"
		}
		parts = append(parts, f+" の "+strings.Join(ns, ", ")+" 行目"+more)
	}
	if len(parts) > most {
		parts = append(parts[:most], "ほか")
	}
	return strings.Join(parts, "; ")
}

// rowIndex gives the rows of a dataset by table and key text.
func rowIndex(d *model.Dataset) map[string]*model.Row {
	idx := map[string]*model.Row{}
	if d == nil {
		return idx
	}
	for name, t := range d.Tables {
		for _, r := range t.Rows {
			idx[name+"\x00"+r.KeyText()] = r
		}
	}
	return idx
}

// secretValue is a secret a line points at: the cell (table, key and column),
// and what it holds.
type secretValue struct {
	cell   string
	values []string
}

// secrets gives the secrets behind a reference of a line: the cell it names,
// or, for the line that opened a row, the secret columns of the key.
func (e *Env) secrets(ref importer.Ref, rows map[string]*model.Row) []secretValue {
	r := rows[ref.Table+"\x00"+ref.Key]
	if r == nil {
		return nil
	}
	id := ref.Table + "\x00" + ref.Key + "\x00"
	var out []secretValue
	add := func(col string, values []string) {
		var filled []string
		for _, v := range values {
			if v != "" {
				filled = append(filled, v)
			}
		}
		if len(filled) > 0 {
			out = append(out, secretValue{id + col, filled})
		}
	}
	if ref.Column != "" {
		if c := r.Cells[ref.Column]; c != nil && e.secret(ref.Table, ref.Column) {
			add(ref.Column, c.Values)
		}
		return out
	}
	for _, k := range r.Key {
		if e.secret(ref.Table, k.Name) {
			add(k.Name, []string{k.Value})
		}
	}
	return out
}

// masker gives the lines of one file for a report, with the secrets masked.
type masker struct {
	e    *Env
	rows map[string]*model.Row
	// found tells for a secret cell that one of the lines pointing at it
	// holds its value as it stands.
	found map[string]bool
}

func holds(text string, values []string) bool {
	for _, v := range values {
		if strings.Contains(text, v) {
			return true
		}
	}
	return false
}

func (e *Env) newMasker(r *importer.Result) *masker {
	m := &masker{e: e, rows: rowIndex(r.Data), found: map[string]bool{}}
	if !e.mask() {
		return m
	}
	for _, l := range r.Lines {
		for _, ref := range l.Refs {
			for _, s := range e.secrets(ref, m.rows) {
				if holds(l.Text, s.values) {
					m.found[s.cell] = true
				}
			}
		}
	}
	return m
}

// text gives the text of a line. With masking on, the values of the secret
// cells the line points at are replaced. A line may point at a cell it gave
// nothing to (a rule that puts several lines together names them all), so a
// value that is not in the line is left alone as long as another line holds
// it. If no line holds it as it stands, the rule changed the value on the
// way, and every line pointing at the cell is masked after its first word.
func (m *masker) text(l importer.LineState) string {
	if !m.e.mask() {
		return l.Text
	}
	var values []string
	for _, ref := range l.Refs {
		for _, s := range m.e.secrets(ref, m.rows) {
			switch {
			case holds(l.Text, s.values):
				values = append(values, s.values...)
			case !m.found[s.cell]:
				body := strings.TrimLeft(l.Text, " \t")
				indent := l.Text[:len(l.Text)-len(body)]
				word, _, _ := strings.Cut(body, " ")
				return indent + word + " " + masked
			}
		}
	}
	// The longest first: a secret may be the start of another one.
	sort.SliceStable(values, func(i, j int) bool { return len(values[i]) > len(values[j]) })
	text := l.Text
	for _, v := range values {
		text = strings.ReplaceAll(text, v, masked)
	}
	return text
}

// refText tells where the value of a line went: the table, the row and the
// column, by the names the book shows.
func (e *Env) refText(ref importer.Ref, rows map[string]*model.Row) string {
	s := e.tableTitle(ref.Table)
	if r := rows[ref.Table+"\x00"+ref.Key]; r != nil {
		if label := e.rowLabel(r); label != "" {
			s += "（" + label + "）"
		}
	}
	if ref.Column == "" {
		return s + " の行"
	}
	return s + " の " + e.columnTitle(ref.Table, ref.Column)
}

func num(n int) string { return strconv.Itoa(n) }

// importDoc builds the report of an import: every line of every file with
// its state (spec 8.3, 13.4).
func (e *Env) importDoc(o *ImportOutcome, now time.Time) *report.Doc {
	src := o.Source
	old := src.Name == "old"
	d := &report.Doc{Title: "取り込みの結果（" + sourceLabel(src.Name) + "）", Facts: e.facts(now)}
	d.Facts = append(d.Facts, report.Pair{Label: "データ源", Value: "input/" + src.Name})

	failed := 0
	for _, f := range src.Files {
		if f.Err != "" {
			failed++
		}
	}
	c := src.Count()
	d.Counts = []report.Pair{
		{Label: "ファイル", Value: num(len(src.Files))},
		{Label: "取り込めなかったファイル", Value: num(failed)},
		{Label: "全行", Value: num(c.Total)},
		{Label: "取り込み済み", Value: num(c.Taken)},
		{Label: "無視（パックが「設定ではない」と定めた行）", Value: num(c.Ignored)},
		{Label: "無視（コメント・空行）", Value: num(c.Separators - c.Outside)},
	}
	if c.Outside > 0 {
		d.Counts = append(d.Counts, report.Pair{Label: "無視（端末ログの設定の外の行）", Value: num(c.Outside)})
	}
	label := "未対応（旧にあって取り込めなかった行）"
	if !old {
		label = "未対応（取り込めなかった行）"
	}
	d.Counts = append(d.Counts, report.Pair{Label: label, Value: num(c.Unknown)})
	jd := o.Judging
	if jd == nil {
		jd = &Judging{Source: src.Name, Open: c.Unknown}
	}
	d.Counts = append(d.Counts,
		report.Pair{Label: "うち未判定（M-401）", Value: num(jd.Open)},
		report.Pair{Label: "うち保留（M-401）", Value: num(jd.Hold)},
		report.Pair{Label: "うち手動で対応", Value: num(jd.Manual)},
		report.Pair{Label: "うち対応不要", Value: num(jd.NotNeeded)},
		report.Pair{Label: "「手動で対応」があるのに追加コンフィグが当たらない機器（M-402）", Value: num(len(jd.NoExtra))},
	)
	if a := o.Applied; a != nil {
		d.Counts = append(d.Counts,
			report.Pair{Label: "新ホスト名が空の旧機器（書き換えの対象外）", Value: num(len(a.Unmapped))},
			report.Pair{Label: "うち備考もない機器（M-407）", Value: num(len(a.NoReason))},
			report.Pair{Label: "新 IF 名が決まっていない旧 IF（M-406）", Value: num(len(a.Undecided))},
			report.Pair{Label: "複数の旧 IF が当たっている新 IF（M-408）", Value: num(len(a.Shared))},
			report.Pair{Label: "対応の未決のうち、未承認の指摘", Value: num(o.Mig.Open())},
			report.Pair{Label: "対応の未決のうち、承認済みの指摘", Value: num(o.Mig.Count().Approved)},
		)
		d.Notes = append(d.Notes, "書き換えの 5 区分のうち、いま数えられるのは「旧にあって取り込めなかった」（未対応の行）だけです。残りの区分は、コンフィグの生成ができるようになってから出ます。")
	}
	if e.mask() {
		d.Notes = append(d.Notes, "秘密の列に入った値は "+masked+" に伏せています。未対応の行にある秘密の値は、ツールが見分けられないため伏せられません。このレポートを人に渡す前に確かめてください。")
	}
	if jd.HostsUnknown {
		d.Notes = append(d.Notes, "「手動で対応」と判定した行がありますが、パラメータブックがまだ無いので、新ホスト名が分かりません。追加コンフィグがあるか（M-402）は見ていません。")
	}
	if jd.BookErr != "" {
		d.Notes = append(d.Notes, "パラメータブックを読めないため、グループ向けの追加コンフィグ（@グループ名.j2）は見ていません: "+jd.BookErr)
	}

	// By device.
	devs := report.Sheet{Name: "機器別", Header: []string{"ファイル", "機器", "パック", "パックの決め方", "全行", "取り込み済み", "無視", "コメント・空行", "未対応", "未判定・保留", "取り込めた割合", "注意"}}
	if old {
		devs.Header = append(devs.Header, "新ホスト名")
	}
	for _, f := range src.Files {
		row := []string{f.Path, "", "", "", "", "", "", "", "", "", "", f.Err}
		mark := report.MarkWarn
		if f.Left != "" {
			// Left out by a person's choice: told, and no fault.
			row[1], row[11], mark = f.Device, f.Left, report.MarkIgnored
		}
		if f.Pack != "" {
			row[2] = f.Pack + " " + e.Packs[f.Pack].Version
			row[3] = "自動"
			if f.Chosen {
				row[3] = "人が選んだ（M-410）"
			}
		}
		if r := f.Result; r != nil {
			n := r.Count()
			row[1] = r.Device
			row[4], row[5], row[6], row[7], row[8] = num(n.Total), num(n.Taken), num(n.Ignored), num(n.Separators), num(n.Unknown)
			left := jd.OpenLines(f.Path)
			if o.Judging == nil {
				left = n.Unknown
			}
			row[9] = num(left)
			row[10] = fmt.Sprintf("%.1f%%", n.Covered()*100)
			if len(r.Warnings) > 0 {
				row[11] = fmt.Sprintf("注意 %d 件", len(r.Warnings))
			}
			switch {
			case left > 0:
				mark = report.MarkUnknown
			case len(r.Warnings) == 0:
				mark = ""
			}
			if old {
				host := ""
				if o.Applied != nil {
					host = o.Applied.Hosts[r.Device]
				}
				row = append(row, host)
			}
		} else if old {
			row = append(row, "")
		}
		devs.Rows = append(devs.Rows, row)
		devs.Marks = append(devs.Marks, mark)
	}

	// The groups of unhandled lines, what is open first.
	shapes := report.Sheet{
		Name:   "未対応の形",
		Lead:   "未対応の行を、値を伏せた形でまとめたものです。判定は、いちばん上の未対応の行に付けます。その下の行は「下の行数」に数えます。",
		Header: []string{"ID", "パック", "親の行", "形", "台数", "行数", "下の行数", "判定", "判定の理由", "判定した人", "例"},
		Code:   []int{2, 3},
	}
	for _, g := range sortedGroups(jd.Groups) {
		heads, below := g.Count()
		example := fmt.Sprintf("%s の %d 行目", g.Devices[0].Device, g.Example.Line)
		if g.Devices[0].Device == "" {
			example = fmt.Sprintf("%s の %d 行目", g.Devices[0].File, g.Example.Line)
		}
		shapes.Rows = append(shapes.Rows, []string{g.ID, g.Key.Pack, g.PathText(), g.Key.Shape, num(len(g.Devices)), num(heads), num(below),
			g.VerdictText(), g.ReasonText(), g.ByText(), example})
		mark := ""
		if g.Open() {
			mark = report.MarkUnknown
		}
		shapes.Marks = append(shapes.Marks, mark)
	}

	// The lines.
	unknown := report.Sheet{
		Name:   "未対応の行",
		Lead:   "どの規則にも当たらなかった行です。親が未対応なら、その子もすべて未対応になります。",
		Header: []string{"機器", "ファイル", "行番号", "文面", "判定", "判定の理由", "かたまり", "理由"},
		Code:   []int{3},
	}
	all := report.Sheet{
		Name:   "行の一覧",
		Lead:   "取り込んだファイルのすべての行です。",
		Header: []string{"機器", "ファイル", "行番号", "文面", "状態", "行き先", "規則", "判定", "判定の理由", "かたまり", "理由"},
		Code:   []int{3},
	}
	notes := report.Sheet{Name: "注意", Header: []string{"ファイル", "機器", "内容"}}
	for _, m := range jd.NoExtra {
		who := m.Device
		if m.Host != m.Device {
			who = fmt.Sprintf("%s（新ホスト名 %s）", m.Device, m.Host)
		}
		notes.Rows = append(notes.Rows, []string{"", m.Device, fmt.Sprintf("M-402: %s に「手動で対応」と判定した行が %d 行ありますが、extra に、この機器に当たる追加コンフィグ（_all.j2・@グループ名.j2・%s.j2）がありません", who, m.Lines, m.Host)})
	}
	for _, j := range jd.Stale {
		path := ""
		if len(j.Path) > 0 {
			path = "親の行 " + strings.Join(j.Path, " > ") + "、"
		}
		notes.Rows = append(notes.Rows, []string{"state/judgments.yaml", "", fmt.Sprintf("判定「%s」（パック %s、%s形 %s、範囲 %s）は、当たる行がありません（文面が変わったか、パックが対応しました）", VerdictWord(j.Verdict), j.Pack, path, j.Shape, j.Devices)})
	}
	for _, p := range e.PackProblems {
		notes.Rows = append(notes.Rows, []string{"", "", p})
	}
	for _, f := range src.Files {
		r := f.Result
		if r == nil {
			continue
		}
		for _, w := range r.Warnings {
			notes.Rows = append(notes.Rows, []string{f.Path, r.Device, w})
		}
		for _, s := range r.Notes {
			notes.Rows = append(notes.Rows, []string{f.Path, r.Device, "（お知らせ）" + s})
		}
		m := e.newMasker(r)
		rows := m.rows
		for _, l := range r.Lines {
			text := m.text(l)
			var to []string
			for _, ref := range l.Refs {
				to = append(to, e.refText(ref, rows))
			}
			judged, why, group, mark := "", "", "", ""
			switch {
			case l.State == importer.Unknown:
				judged, mark = notJudged, report.MarkUnknown
				if v, ok := jd.Line(f.Path, l.Line); ok {
					judged, why, group = v.Word(), v.Reason, v.Group()
					if !open(v.Verdict) {
						mark = report.MarkWarn
					}
				}
				unknown.Rows = append(unknown.Rows, []string{r.Device, f.Path, num(l.Line), text, judged, why, group, l.Reason})
				unknown.Marks = append(unknown.Marks, mark)
			case l.State == importer.Ignored:
				mark = report.MarkIgnored
			}
			all.Rows = append(all.Rows, []string{r.Device, f.Path, num(l.Line), text, stateWords[l.State], strings.Join(uniq(to), "; "), l.RuleID, judged, why, group, l.Reason})
			all.Marks = append(all.Marks, mark)
		}
	}
	if o.New != nil {
		for _, f := range o.New.Files {
			if f.Err != "" {
				notes.Rows = append(notes.Rows, []string{f.Path, "", "対応の案を作るために読もうとしましたが、取り込めませんでした: " + f.Err})
			}
		}
	}
	for _, p := range o.Problems {
		notes.Rows = append(notes.Rows, []string{e.Proj.Config.Book, "", p.String()})
	}

	d.Sheets = []report.Sheet{devs, shapes, unknown}
	if o.Applied != nil {
		d.Sheets = append(d.Sheets, e.mappingSheet(o.Applied, o.Mig, false))
	}
	d.Sheets = append(d.Sheets, notes, all)
	return d
}

func uniq(list []string) []string {
	var out []string
	seen := map[string]bool{}
	for _, s := range list {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

// mappingSheet lists what the mapping left open and, with all set, what it
// left out of the data. v holds the findings (M-406 to M-408) with the
// approvals put on.
func (e *Env) mappingSheet(a *convert.Applied, v *Verdicts, all bool) report.Sheet {
	s := report.Sheet{
		Name:   "対応の未決",
		Lead:   "パラメータブックの「機器対応」「IF 対応」で、まだ決まっていないところです。直すか、理由を付けて承認します（画面では行を押します。コマンドラインでは、指紋を approve に渡します）。",
		Header: []string{"チェック", "旧機器", "表", "行", "列", "対象", "内容", "状態", "承認", "指紋", "直し方"},
	}
	if all {
		s.Name = "対応のメモ"
		s.Lead = "「機器対応」「IF 対応」を当てたときに、決まっていなかったところと、書かなかった行です。"
	}
	add := func(mark, check, dev, table, row, col, target, text string) {
		s.Rows = append(s.Rows, []string{check, dev, table, row, col, target, text, "", "", "", ""})
		s.Marks = append(s.Marks, mark)
	}
	// found adds the row of a finding: marked while it is not approved.
	found := func(f check.Finding, dev, table, row, col, target string) {
		st, mark := stateCells(v, f)
		s.Rows = append(s.Rows, []string{f.RuleID, dev, table, row, col, target, f.Message, st[1], st[2], st[3], st[4]})
		s.Marks = append(s.Marks, mark)
	}
	noReason := map[string]bool{}
	for _, h := range a.NoReason {
		noReason[h] = true
	}
	for _, h := range a.Unmapped {
		if noReason[h] {
			found(noReasonFinding(h), h, "", "", "", h)
		} else if all {
			add("", "", h, "", "", "", h, "新ホスト名が空のため、書き換えの対象外です（備考あり）")
		}
	}
	place := e.Place
	for _, n := range a.Undecided {
		t, r, c := place(n)
		found(undecidedFinding(n), n.Device, t, r, c, n.Value)
	}
	for _, n := range a.Shared {
		found(sharedFinding(n), "", "", "", "", n.Value)
	}
	for _, n := range a.Clashes {
		t, r, _ := place(n)
		found(clashFinding(n), n.Device, t, r, "", "")
	}
	if all {
		for _, n := range a.Dropped {
			t, r, c := place(n)
			add("", "", n.Device, t, r, c, n.Value, "新 IF 名が - の IF の行です。書いていません")
		}
		for _, n := range a.Cleared {
			t, r, c := place(n)
			add(report.MarkWarn, "", n.Device, t, r, c, n.Value, "新 IF 名が - の IF を指していた列です。空にしました")
		}
	}
	return s
}

func sourceLabel(name string) string {
	switch name {
	case "old":
		return "旧コンフィグ"
	case "new":
		return "新コンフィグ"
	case "book":
		return "パラメータブック"
	}
	return name
}

// compareDoc builds the report of a comparison (spec 12.3, 13.5).
func (e *Env) compareDoc(o *CompareOutcome, now time.Time) *report.Doc {
	la, lb := sourceLabel(o.A.Name), sourceLabel(o.B.Name)
	d := &report.Doc{Title: "比較の結果（" + la + " と " + lb + "）", Facts: e.facts(now)}
	d.Facts = append(d.Facts, report.Pair{Label: "A", Value: sideText(o.A)}, report.Pair{Label: "B", Value: sideText(o.B)})
	r := o.Result
	d.Counts = []report.Pair{
		{Label: "A（" + la + "）にだけある行", Value: num(len(r.OnlyA))},
		{Label: "B（" + lb + "）にだけある行", Value: num(len(r.OnlyB))},
		{Label: "値が違うセル", Value: num(len(r.Cells))},
		{Label: "両方にある行", Value: num(r.Rows)},
		{Label: "うち値がすべて同じ行", Value: num(r.Same)},
	}
	mig := o.Mig
	if mig != nil {
		c := mig.Count()
		d.Counts = append(d.Counts,
			report.Pair{Label: "未承認のエラー", Value: num(c.Errors)},
			report.Pair{Label: "未承認の警告", Value: num(c.Warnings)},
			report.Pair{Label: "承認済みの指摘", Value: num(c.Approved)},
		)
		d.Notes = append(d.Notes, "旧との違いは、移行のチェック（M-403〜M-408）の指摘です。意図した違いは、理由を付けて承認します（画面では行を押します。コマンドラインでは、各行の「指紋」を approve に渡すか、範囲で承認します）。")
	}
	oldIsA := o.A.Name == "old"
	for _, s := range []*Side{o.A, o.B} {
		switch s.Name {
		case "old":
			d.Notes = append(d.Notes, "旧コンフィグの機器名と IF 名は、パラメータブックの「機器対応」「IF 対応」を当ててから比べています。新ホスト名が空の旧機器の行は、比べる対象に入っていません。")
		case "book":
			d.Notes = append(d.Notes, "パラメータブックは、「機器」列の * と @グループ、「共通値」を機器ごとの行に広げ、空の列を機器のメーカーの既定値で埋めてから比べています（取り込んだコンフィグと同じ条件）。")
		}
	}
	if e.mask() {
		d.Notes = append(d.Notes, "秘密の列の値は "+masked+" に伏せています。")
	}

	only := func(name, lead string, list []compare.RowDiff, old bool) report.Sheet {
		s := report.Sheet{Name: name, Lead: lead, Header: []string{"表", "機器", "行", "内容", "由来"}}
		if mig != nil {
			s.Header = append(s.Header, stateHeader...)
		}
		for _, x := range list {
			row := []string{e.tableTitle(x.Table), x.Device, e.rowLabel(x.Row), e.rowText(x.Row), originText(x.Row.Origins)}
			if mig != nil {
				st, mark := stateCells(mig, e.rowFinding(x, old))
				row = append(row, st...)
				s.Marks = append(s.Marks, mark)
			}
			s.Rows = append(s.Rows, row)
		}
		return s
	}
	leadA, leadB, leadC := la+" にあって、"+lb+" にない行です。", lb+" にあって、"+la+" にない行です。", "両方にある行のうち、値が違うセルです。"
	if o.A.Name == "old" || o.B.Name == "old" {
		// The checks of a migration (appendix C): seen from the old side.
		if o.A.Name == "old" {
			leadA, leadB = leadA+"（M-403）", leadB+"（M-404）"
		} else {
			leadA, leadB = leadA+"（M-404）", leadB+"（M-403）"
		}
		leadC += "（M-405）"
	}
	cells := report.Sheet{
		Name:   "値が違うセル",
		Lead:   leadC,
		Header: []string{"表", "機器", "行", "列", "A（" + la + "）の値", "B（" + lb + "）の値", "備考", "A の由来", "B の由来"},
	}
	if mig != nil {
		cells.Header = append(cells.Header, stateHeader...)
	}
	for _, x := range r.Cells {
		note := ""
		va, vb := e.cellText(x.Table, x.Column, x.A), e.cellText(x.Table, x.Column, x.B)
		switch {
		case e.mask() && e.secret(x.Table, x.Column):
			note = "秘密の値（伏せています）"
		case x.OrderOnly:
			note = "並びだけが違います"
		case isDefault(x.A) && isDefault(x.B):
			note = "A も B も既定値（書かれていない値）"
		case isDefault(x.A):
			note = "A は既定値（書かれていない値）"
		case isDefault(x.B):
			note = "B は既定値（書かれていない値）"
		}
		row := []string{e.tableTitle(x.Table), x.Device, e.rowLabel(x.ARow), e.columnTitle(x.Table, x.Column), va, vb, note, cellOrigin(x.A), cellOrigin(x.B)}
		if mig != nil {
			st, mark := stateCells(mig, e.cellFinding(x, oldIsA))
			row = append(row, st...)
			cells.Marks = append(cells.Marks, mark)
		}
		cells.Rows = append(cells.Rows, row)
	}
	d.Sheets = []report.Sheet{only("A にだけある行", leadA, r.OnlyA, oldIsA), only("B にだけある行", leadB, r.OnlyB, !oldIsA), cells}

	notes := report.Sheet{Name: "注意", Header: []string{"ファイル", "機器", "内容"}}
	for _, s := range []*Side{o.A, o.B} {
		if s.Applied != nil {
			d.Sheets = append(d.Sheets, e.mappingSheet(s.Applied, o.Mig, true))
		}
		if s.Source != nil {
			for _, f := range s.Source.Files {
				if f.Err != "" {
					notes.Rows = append(notes.Rows, []string{f.Path, "", "取り込めませんでした: " + f.Err})
				} else if n := f.Result.Count().Unknown; n > 0 {
					notes.Rows = append(notes.Rows, []string{f.Path, f.Result.Device, fmt.Sprintf("未対応の行が %d 行あります。その行の設定は、比べる対象に入っていません", n)})
				}
			}
		}
		if s.Book != nil {
			for _, p := range s.Book.Problems {
				notes.Rows = append(notes.Rows, []string{e.Proj.Config.Book, "", p.String()})
			}
			for _, p := range s.Expand {
				notes.Rows = append(notes.Rows, []string{e.Proj.Config.Book, "", p.String()})
			}
			for _, n := range s.Notes {
				notes.Rows = append(notes.Rows, []string{e.Proj.Config.Book, "", n})
			}
		}
	}
	if mig != nil {
		d.Sheets = append(d.Sheets, approvalSheet(mig))
	}
	d.Sheets = append(d.Sheets, notes)
	return d
}

func sideText(s *Side) string {
	switch s.Name {
	case "old":
		return "旧コンフィグ（input/old。対応を当てた後）"
	case "new":
		return "新コンフィグ（input/new）"
	}
	return "パラメータブック"
}

// isDefault tells whether the value of a cell is a default of the pack: no
// line and no cell of the book wrote it.
func isDefault(c *model.Cell) bool {
	return c != nil && len(c.Origins) > 0 && c.Origins[0].Kind == "default"
}

func cellOrigin(c *model.Cell) string {
	if c == nil {
		return ""
	}
	return originText(c.Origins)
}
