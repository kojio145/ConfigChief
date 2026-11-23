package ops

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/compare"
	"github.com/kojio145/configchief/internal/conftree"
	"github.com/kojio145/configchief/internal/fill"
	"github.com/kojio145/configchief/internal/importer"
	"github.com/kojio145/configchief/internal/listimport"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/pack"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/render"
	"github.com/kojio145/configchief/internal/report"
)

// The change commands made from the parameter book (spec 9.8): the
// configuration generated from the book is what each device should be, and
// the difference from its configuration now is what is put in. The lines to
// remove are those the book says are gone: the lines now the pack reads into
// the tables, whose cells the book leaves empty, has no row for, or holds
// another value in.

const deltaDir = "delta"

// DeltaOptions say what to make.
type DeltaOptions struct {
	Side    string   // old | new | logs: the configurations now
	Devices []string // "" = every device of the book
}

// DeltaDevice is the outcome of one device.
type DeltaDevice struct {
	Name      string // the host name of the book
	Made      bool
	Unchanged bool   // nothing to change
	Why       string // why it was not made
	Now       string // the file of the configuration now
	NowHost   string // the host name in it (another one for input/old)
	Pack      string
	File      string // the file written, from the case folder
	Lines     int
	Text      string
	Added     []fill.Item
	Removed   []fill.Item
	Replaced  []fill.Item // Why: the line that replaces it
	Kept      []fill.Item // Why: why it stays

	// secrets are the values of the secret columns of the device, in the
	// book and in the configuration now, the longest first.
	secrets []string
}

// DeltaOutcome is the outcome of making the changes.
type DeltaOutcome struct {
	Side    string
	Devices []*DeltaDevice
	Notes   []string
	Dir     string // out/<time>/delta ("" = nothing written)
	Report  Files
}

// NotMade counts the devices not made.
func (o *DeltaOutcome) NotMade() int {
	n := 0
	for _, d := range o.Devices {
		if !d.Made && !d.Unchanged {
			n++
		}
	}
	return n
}

// Made counts the devices made.
func (o *DeltaOutcome) Made() int {
	n := 0
	for _, d := range o.Devices {
		if d.Made {
			n++
		}
	}
	return n
}

// Why a line now that the generated configuration does not have stays.
const (
	keptUnknown  = "取り込めない行（読めなかった）なので消しません"
	keptIgnored  = "パックが無視する行なので消しません"
	keptNoCell   = "取り込んだが、設計データのどのセルに入ったか分からないので消しません"
	keptSame     = "ブックに同じ値があります（書き方が違うだけかもしれないので消しません）"
	keptListed   = "ブックのこのセルに、この行の値が残っています"
	keptUnused   = "ブックに値があるのに、パックが書けない値です（G-602）。消すと設定が無くなるので消しません"
	keptRow      = "ブックに同じ行があります"
	keptPhysical = "ブックに行が無い物理ポートです。ポートは消せず、中の行を消すとポートの状態が変わる（shutdown が外れるなど）ので、触りません"
	keptInside   = "中に消さない行があるので、この行は消さずに、中の行を 1 行ずつ見ます"
)

// Delta generates the configurations of the book (writing nothing of the
// generation), compares each with the device's configuration now and
// writes the change commands and the report (spec 9.8).
func (e *Env) Delta(opt DeltaOptions, now time.Time) (*DeltaOutcome, error) {
	if opt.Side != "old" && opt.Side != "new" && opt.Side != "logs" {
		return nil, fmt.Errorf("今のコンフィグ「%s」は使えません（old・new・logs）", opt.Side)
	}
	gen, err := e.Generate(GenOptions{Dry: true}, now)
	if err != nil {
		return nil, err
	}
	files, fails, err := e.treeSide(opt.Side, now, nil)
	if err != nil {
		return nil, err
	}
	o := &DeltaOutcome{Side: opt.Side}
	for _, f := range fails {
		o.Notes = append(o.Notes, "読めなかったファイル: "+f.Path+": "+f.Why)
	}
	if c := gen.Count(); c.Errors+c.Warnings > 0 {
		o.Notes = append(o.Notes, fmt.Sprintf("生成のチェックに、未承認のエラー %d 件・警告 %d 件があります（「生成とチェック」のチェック結果表で確かめてください）", c.Errors, c.Warnings))
	}
	for _, n := range gen.BookNotes {
		o.Notes = append(o.Notes, "ブックの注意: "+n)
	}

	// The new host name of a device of input/old, and the devices whose
	// interfaces the book renames.
	newHost := map[string]string{}
	renamed := map[string]bool{}
	if opt.Side == "old" {
		_, bk, err := e.readBook()
		if err != nil {
			return nil, err
		}
		if bk != nil {
			for _, d := range bk.Devices {
				if d.OldHost != "" && d.NewHost != "" {
					newHost[d.OldHost] = d.NewHost
				}
			}
			for _, m := range bk.Ifs {
				nif := strings.TrimSpace(m.NewIf)
				if nif != "" && nif != "-" && !strings.EqualFold(conftree.Normalize(nif), conftree.Normalize(m.OldIf)) {
					renamed[listimport.DeviceKey(m.NewHost)] = true
				}
			}
		}
	}
	current := map[string]*TreeFile{}
	for _, f := range files {
		h := f.Host
		if n := newHost[h]; n != "" {
			h = n
		}
		current[listimport.DeviceKey(h)] = f
	}

	want := map[string]bool{}
	for _, d := range opt.Devices {
		if k := listimport.DeviceKey(d); k != "" {
			want[k] = true
		}
	}
	o.Dir = e.outDir(now) + "/" + deltaDir
	// filtered is kept apart from want: want empties as the devices asked
	// for are found, and the devices after the last of them are not asked.
	filtered := len(want) > 0
	for _, run := range gen.Devices {
		key := listimport.DeviceKey(run.Host)
		if filtered {
			if !want[key] {
				continue
			}
			delete(want, key)
		}
		d := &DeltaDevice{Name: run.Host}
		o.Devices = append(o.Devices, d)
		e.deltaDevice(d, run, current[key], renamed[key], opt.Side)
		if d.Made {
			d.File = o.Dir + "/" + safeName(d.Name) + ".txt"
		}
	}
	if len(want) > 0 {
		var miss []string
		for k := range want {
			miss = append(miss, k)
		}
		sort.Strings(miss)
		return nil, fmt.Errorf("パラメータブックに無い機器です: %s", strings.Join(miss, "、"))
	}
	if o.Made() > 0 {
		if err := os.MkdirAll(e.Proj.Path(o.Dir), 0o755); err != nil {
			return nil, fmt.Errorf("%s を作れません: %v", o.Dir, err)
		}
		for _, d := range o.Devices {
			if !d.Made {
				continue
			}
			p := e.Packs[d.Pack]
			enc, nl := e.Proj.Config.Output.Encoding, e.Proj.Config.Output.Newline
			if enc == "" {
				enc = p.Output.Encoding
			}
			if nl == "" {
				nl = p.Output.Newline
			}
			data, err := project.Encode(d.Text, enc, nl)
			if err != nil {
				data, _ = project.Encode(d.Text, "utf-8", nl)
			}
			if err := os.WriteFile(e.Proj.Path(d.File), data, 0o644); err != nil {
				return nil, fmt.Errorf("%s を書けません: %v", d.File, err)
			}
		}
	} else {
		o.Dir = ""
	}
	o.Report, err = e.writeReport(now, "delta", e.deltaDoc(o, now))
	if err != nil {
		return nil, err
	}
	same := 0
	for _, d := range o.Devices {
		if d.Unchanged {
			same++
		}
	}
	e.Proj.Log(now, "差分の投入（今: %s）: 機器 %d、作った %d、変更なし %d、作らなかった %d", opt.Side, len(o.Devices), o.Made(), same, o.NotMade())
	return o, nil
}

// deltaDevice makes the change of one device.
func (e *Env) deltaDevice(d *DeltaDevice, run DeviceRun, f *TreeFile, renamed bool, side string) {
	switch {
	case run.Output == nil:
		d.Why = "コンフィグを生成していません"
		if run.Why != "" {
			d.Why += "（" + run.Why + "）"
		}
		return
	case run.Output.Failed:
		d.Why = "テンプレートが失敗して、生成したコンフィグがそろっていません"
		return
	case f == nil:
		d.Why = fmt.Sprintf("今のコンフィグ（%s）に、この機器のファイルがありません", sideTitle(side))
		return
	}
	d.Now, d.NowHost, d.Pack = f.Path, f.Host, run.Pack
	d.secrets = e.secretValues(run.Output.Data)
	switch {
	case f.Pack == "" || e.Packs[f.Pack] == nil:
		d.Why = fmt.Sprintf("今のコンフィグ（%s）のメーカーを判定できません", f.Path)
		return
	case f.Pack != run.Pack:
		d.Why = fmt.Sprintf("今のコンフィグのメーカーパック（%s）と、生成するメーカーパック（%s）が違います。別のメーカー・OS へは全文で入れてください", f.Pack, run.Pack)
		return
	case renamed:
		d.Why = "新旧対照表（ポート）で IF 名が変わる機器です（入れ替え）。差分は作らないので、全文で入れてください"
		return
	}
	p := e.Packs[run.Pack]
	if p.Change == nil {
		d.Why = fmt.Sprintf("メーカーパック %s には、変更の投入の書き方（change）がありません", p.ID)
		return
	}
	// The configuration now, without the lines of a log that are not
	// configuration (as 9.7.4).
	var lines []string
	for i, l := range f.Source.Lines {
		if f.Source.Outside == nil || !f.Source.Outside[i] {
			lines = append(lines, l)
		}
	}
	curText := strings.Join(lines, "\n") + "\n"
	cur, err := importer.Import(p, e.Schema, f.Path, curText)
	if err != nil || len(cur.Lines) != len(lines) {
		d.Why = fmt.Sprintf("今のコンフィグ（%s）を取り込めません", f.Path)
		if err != nil {
			d.Why += ": " + err.Error()
		}
		return
	}
	d.secrets = e.secretValues(run.Output.Data, cur.Data)
	o := run.Output
	gres, err := importer.Import(p, e.Schema, run.Host+".txt", o.Text)
	if err != nil {
		d.Why = "生成したコンフィグを取り込み直せません: " + err.Error()
		return
	}
	target := &fill.Output{}
	for _, l := range strings.Split(strings.TrimSuffix(o.Text, "\n"), "\n") {
		target.Lines = append(target.Lines, fill.Line{Text: l})
	}
	j := e.newDeltaJudge(p, o, cur, gres, run.Host)
	delta, err := fill.MakeDeltaBy(target, curText, p, j.judge)
	if err != nil {
		d.Why = err.Error()
		return
	}
	d.Added, d.Removed, d.Replaced, d.Kept = delta.Added, delta.Removed, delta.Replaced, delta.Kept
	if delta.Empty() {
		d.Unchanged = true
		return
	}
	d.Made = true
	d.Text = strings.Join(delta.Commands, "\n") + "\n"
	d.Lines = len(delta.Commands)
}

// deltaJudge decides about the lines now that the generated configuration
// does not have (spec 9.8).
type deltaJudge struct {
	e    *Env
	p    *pack.Pack
	out  *render.Output
	cur  *importer.Result
	gen  *importer.Result
	rows map[string]*model.Row // a ref of a line now → the row (renamed to the host of the book)
	only map[*model.Row]bool   // rows now the book does not have
	diff map[*model.Row]map[string]*compare.CellDiff
	// many: a cell more than one line (now, or generated) writes: a value
	// on each line.
	many map[string]bool
}

func (e *Env) newDeltaJudge(p *pack.Pack, o *render.Output, cur, gen *importer.Result, host string) *deltaJudge {
	j := &deltaJudge{e: e, p: p, out: o, cur: cur, gen: gen, rows: map[string]*model.Row{},
		only: map[*model.Row]bool{}, diff: map[*model.Row]map[string]*compare.CellDiff{}, many: map[string]bool{}}
	// The rows now by the key the lines name, then named after the host of
	// the book (input/old: the new host name).
	data := model.NewDataset(cur.Data.Source)
	for _, name := range cur.Data.TableNames() {
		t := e.Schema.Table(name)
		for _, r := range cur.Data.Tables[name].Rows {
			j.rows[name+"\x00"+r.KeyText()] = r
			if t != nil && cur.Device != "" && cur.Device != host {
				if dc := t.DeviceColumn(); dc != "" {
					for i := range r.Key {
						if r.Key[i].Name == dc && r.Key[i].Value == cur.Device {
							r.Key[i].Value = host
						}
					}
					if c := r.Cells[dc]; c != nil {
						for i, v := range c.Values {
							if v == cur.Device {
								c.Values[i] = host
							}
						}
					}
				}
			}
			data.Adopt(r)
		}
	}
	res := compare.Datasets(e.Schema, o.Design(p), data)
	for _, d := range res.OnlyB {
		j.only[d.Row] = true
	}
	for i := range res.Cells {
		c := &res.Cells[i]
		if j.diff[c.BRow] == nil {
			j.diff[c.BRow] = map[string]*compare.CellDiff{}
		}
		j.diff[c.BRow][c.Column] = c
	}
	count := func(lines []importer.LineState, cell func(importer.Ref) string) {
		seen := map[string]int{}
		for _, l := range lines {
			if l.State != importer.Taken {
				continue
			}
			for _, r := range l.Refs {
				if r.Column != "" {
					seen[cell(r)]++
				}
			}
		}
		for k, n := range seen {
			if n > 1 {
				j.many[k] = true
			}
		}
	}
	count(cur.Lines, j.curCell)
	count(gen.Lines, genCell)
	return j
}

// curCell names the cell a ref of a line now points at, as the generated
// lines name it.
func (j *deltaJudge) curCell(r importer.Ref) string {
	key := r.Key
	if row := j.rows[r.Table+"\x00"+r.Key]; row != nil {
		key = row.KeyText()
	}
	return r.Table + "\x00" + key + "\x00" + r.Column
}

func genCell(r importer.Ref) string { return r.Table + "\x00" + r.Key + "\x00" + r.Column }

// refs are the refs of a line to the tables of the design data.
func (j *deltaJudge) refs(l importer.LineState) []importer.Ref {
	var out []importer.Ref
	for _, r := range l.Refs {
		if j.e.Schema.Table(r.Table) != nil {
			out = append(out, r)
		}
	}
	return out
}

func (j *deltaJudge) line(n *conftree.Node) importer.LineState {
	if n.Line < 1 || n.Line > len(j.cur.Lines) {
		return importer.LineState{State: importer.Unknown}
	}
	return j.cur.Lines[n.Line-1]
}

// gone tells whether the book says the setting of a line is gone; if not,
// why it stays.
func (j *deltaJudge) gone(n *conftree.Node) (bool, string) {
	l := j.line(n)
	switch {
	case l.State == importer.Ignored:
		return false, keptIgnored
	case l.State != importer.Taken:
		return false, keptUnknown
	}
	refs := j.refs(l)
	if len(refs) == 0 {
		return false, keptNoCell
	}
	text := strings.TrimSpace(n.Text)
	for _, r := range refs {
		row := j.rows[r.Table+"\x00"+r.Key]
		if row == nil {
			return false, keptNoCell
		}
		if j.only[row] {
			continue
		}
		if r.Column == "" || isKey(row, r.Column) {
			return false, keptRow
		}
		cd := j.diff[row][r.Column]
		if cd == nil {
			return false, keptSame
		}
		if cd.ARow != nil && j.out.UnusedCell(cd.ARow, r.Column) {
			return false, keptUnused
		}
		if cd.A == nil {
			continue
		}
		if c := j.e.Schema.Table(r.Table).Column(r.Column); c != nil && c.List {
			for _, v := range cd.A.Values {
				if v != "" && strings.Contains(text, v) {
					return false, keptListed
				}
			}
		}
	}
	return true, ""
}

func isKey(r *model.Row, col string) bool {
	for _, k := range r.Key {
		if k.Name == col {
			return true
		}
	}
	return false
}

// removable tells that a line and every line below it may go; if not, why.
func (j *deltaJudge) removable(n *conftree.Node) (bool, string) {
	if ok, why := j.gone(n); !ok {
		return false, why
	}
	for _, c := range n.Children {
		if c.Body {
			continue
		}
		if j.line(c).State == importer.Ignored {
			continue
		}
		if ok, _ := j.removable(c); !ok {
			return false, keptInside
		}
	}
	return true, ""
}

// physical tells that a line opens the row of a physical port.
func (j *deltaJudge) physical(l importer.LineState) bool {
	for _, r := range j.refs(l) {
		if r.Column != "" {
			continue
		}
		t := j.e.Schema.Table(r.Table)
		row := j.rows[r.Table+"\x00"+r.Key]
		if t == nil || row == nil {
			continue
		}
		for _, k := range row.Key {
			if c := t.Column(k.Name); c != nil && c.Type == "ifname" && j.p.IfKind(k.Value) == "physical" {
				return true
			}
		}
	}
	return false
}

func (j *deltaJudge) judge(n *conftree.Node, added []*conftree.Node) (int, string, *conftree.Node) {
	l := j.line(n)
	if l.State == importer.Taken {
		// An added line that writes the same cell writes over this one.
		mine := map[string]bool{}
		for _, r := range j.refs(l) {
			if k := j.curCell(r); !j.many[k] {
				mine[k] = true
			}
		}
		for _, a := range added {
			if a.Line < 1 || a.Line > len(j.gen.Lines) {
				continue
			}
			for _, r := range j.gen.Lines[a.Line-1].Refs {
				if mine[genCell(r)] {
					return fill.Replace, "", a
				}
			}
		}
	}
	block := false
	for _, c := range n.Children {
		if !c.Body {
			block = true
		}
	}
	ok, why := j.gone(n)
	if !ok {
		if block && l.State == importer.Taken {
			return fill.Enter, why, nil
		}
		return fill.Keep, why, nil
	}
	if j.physical(l) {
		return fill.Keep, keptPhysical, nil
	}
	if ok, why := j.removable(n); !ok {
		return fill.Enter, why, nil
	}
	return fill.Remove, "", nil
}

// secretValues are the values of the secret columns of datasets, the
// longest first (a secret may be the start of another one).
func (e *Env) secretValues(sets ...*model.Dataset) []string {
	var out []string
	seen := map[string]bool{}
	for _, ds := range sets {
		if ds == nil {
			continue
		}
		for _, t := range e.Schema.Tables {
			tab := ds.Tables[t.Name]
			if tab == nil {
				continue
			}
			for _, c := range t.Columns {
				if !c.Secret {
					continue
				}
				for _, r := range tab.Rows {
					if cell := r.Cells[c.Name]; cell != nil {
						for _, v := range cell.Values {
							if v != "" && !seen[v] {
								seen[v] = true
								out = append(out, v)
							}
						}
					}
				}
			}
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return len(out[i]) > len(out[j]) })
	return out
}

// Shown gives a line of the change as the report and the screen show it:
// with masking on, the secrets hidden.
func (e *Env) Shown(d *DeltaDevice, text string) string {
	if !e.mask() {
		return text
	}
	return hide(text, d.secrets)
}

func hide(text string, secrets []string) string {
	for _, v := range secrets {
		text = strings.ReplaceAll(text, v, masked)
	}
	return text
}

// deltaDoc is the report of the changes.
func (e *Env) deltaDoc(o *DeltaOutcome, now time.Time) *report.Doc {
	d := &report.Doc{Title: "差分の投入", Facts: e.facts(now)}
	d.Facts = append(d.Facts, report.Pair{Label: "あるべき姿", Value: e.Proj.Config.Book + " から生成したコンフィグ"},
		report.Pair{Label: "今のコンフィグ", Value: sideTitle(o.Side)})
	if o.Dir != "" {
		d.Facts = append(d.Facts, report.Pair{Label: "出力", Value: o.Dir})
	}
	same := 0
	for _, dv := range o.Devices {
		if dv.Unchanged {
			same++
		}
	}
	d.Counts = []report.Pair{{Label: "機器", Value: num(len(o.Devices))}, {Label: "作った", Value: num(o.Made())},
		{Label: "変更なし", Value: num(same)}, {Label: "作らなかった", Value: num(o.NotMade())}}
	d.Notes = append(d.Notes, o.Notes...)
	d.Notes = append(d.Notes, "消す行（no …）は、今のコンフィグを取り込んで表に入った行のうち、ブックでそのセルが空・行が無い・値が違うものだけです。取り込めない行は消していません（「残る行」）。",
		"「置き換わる行」は、同じセルに値を入れる足す行で上書きされるとみなして、消していません。1 行に 1 つの値を書く設定で値を入れ替えたときは古い行が残ることがあるので、確かめてください。",
		"保存のコマンドは付けていません。")
	if e.mask() {
		d.Notes = append(d.Notes, "秘密の列の値（ブックの値と、今のコンフィグから取り込んだ値）は "+masked+" に伏せています。投入のファイルにはそのまま入っています。")
	}
	devs := report.Sheet{Name: "機器", Header: []string{"機器", "結果", "理由", "今のコンフィグ", "パック", "足す", "消す", "置き換わる", "残る", "ファイル"}}
	for _, dv := range o.Devices {
		res := "作った"
		switch {
		case dv.Unchanged:
			res = "変更なし"
		case !dv.Made:
			res = "作らなかった"
		}
		n := func(items []fill.Item) string {
			if !dv.Made && !dv.Unchanged {
				return ""
			}
			return num(len(items))
		}
		devs.Rows = append(devs.Rows, []string{dv.Name, res, dv.Why, dv.Now, dv.Pack, n(dv.Added), n(dv.Removed), n(dv.Replaced), n(dv.Kept), dv.File})
		mark := ""
		if !dv.Made && !dv.Unchanged {
			mark = "unknown"
		}
		devs.Marks = append(devs.Marks, mark)
	}
	df := report.Sheet{Name: "差", Lead: "+ 足す行、- 消す行（右は消すコマンド）、~ 置き換わる行（右は置き換える行）、= 残る行（右は理由）。", Header: []string{"機器", "印", "行", "消すコマンド・置き換える行・理由"}, Code: []int{2}}
	for _, dv := range o.Devices {
		for _, it := range dv.Added {
			df.Rows = append(df.Rows, []string{dv.Name, "+", e.Shown(dv, itemText(it)), ""})
		}
		for _, it := range dv.Removed {
			df.Rows = append(df.Rows, []string{dv.Name, "-", e.Shown(dv, itemText(it)), e.Shown(dv, it.Cmd)})
		}
		for _, it := range dv.Replaced {
			df.Rows = append(df.Rows, []string{dv.Name, "~", e.Shown(dv, itemText(it)), e.Shown(dv, it.Why)})
		}
		for _, it := range dv.Kept {
			df.Rows = append(df.Rows, []string{dv.Name, "=", e.Shown(dv, itemText(it)), it.Why})
		}
	}
	d.Sheets = append(d.Sheets, devs, df)
	return d
}
