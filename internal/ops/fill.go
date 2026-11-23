package ops

import (
	"fmt"
	"os"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/fill"
	"github.com/kojio145/configchief/internal/listimport"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/render"
	"github.com/kojio145/configchief/internal/report"
	"github.com/kojio145/configchief/internal/schema"
)

// Filling a template with variables (spec 9.7): the templates of fill/, the
// variable rules of vars/, the lists of input/customer and the book.

const (
	fillDir = "fill"
	varsDir = "vars"
)

// FillSides are the configurations a change is made against.
var FillSides = []string{"old", "new", "logs"}

// FillTemplate is one template of the case.
type FillTemplate struct {
	Path string // fill/<name>.txt
	Name string
	Text string
	// Err is the fault of its form ("" if it reads).
	Err string
}

// FillTemplates lists the templates of the case, by name.
func (e *Env) FillTemplates() ([]FillTemplate, error) {
	entries, err := os.ReadDir(e.Proj.Path(fillDir))
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("%s を読めません: %v", fillDir, err)
	}
	var out []FillTemplate
	for _, ent := range entries {
		n := ent.Name()
		if ent.IsDir() || strings.HasPrefix(n, ".") || strings.HasPrefix(n, "~$") || !strings.EqualFold(path.Ext(n), ".txt") {
			continue
		}
		t := FillTemplate{Path: fillDir + "/" + n, Name: strings.TrimSuffix(n, path.Ext(n))}
		text, err := project.ReadText(e.Proj.Path(t.Path))
		if err != nil {
			t.Err = err.Error()
		} else {
			t.Text = text
			if _, err := fill.Parse(t.Name, text); err != nil {
				t.Err = err.Error()
			}
		}
		out = append(out, t)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// VarFiles lists the variable rules of the case, each with its faults.
func (e *Env) VarFiles() ([]listimport.VarFile, error) {
	return listimport.VarFiles(os.DirFS(e.Proj.Path(varsDir)), varsDir)
}

func (e *Env) varRules() ([]*listimport.VarRule, error) {
	files, err := e.VarFiles()
	if err != nil {
		return nil, err
	}
	var rules []*listimport.VarRule
	var problems []string
	for _, f := range files {
		for _, p := range f.Problems {
			problems = append(problems, f.Path+": "+p)
		}
		if f.Rule != nil {
			rules = append(rules, f.Rule)
		}
	}
	if len(problems) > 0 {
		return nil, fmt.Errorf("変数ルール（vars/）に誤りがあります。直してから、もう一度実行してください:\n  %s", strings.Join(problems, "\n  "))
	}
	return rules, nil
}

// fillList reads the list of one variable rule. The list has the names of
// the rule whatever happens; err tells why its rows could not be read. The
// device names of the list take the names of the known devices the rule's
// match finds (spec 9.7.2).
func (e *Env) fillList(r *listimport.VarRule, known []string) (l *fill.List, file string, res *listimport.VarResult, err error) {
	l = &fill.List{Name: r.Name, Rows: map[string][]listimport.VarRow{}}
	for _, v := range r.Vars {
		l.Vars = append(l.Vars, v.Name)
	}
	loc := r.Location()
	file, _, s, err := e.readList(loc)
	if err != nil {
		return l, file, nil, err
	}
	t, err := listimport.ReadTable(s, loc)
	if err != nil {
		return l, file, nil, fmt.Errorf("%s: %v", file, err)
	}
	res = listimport.ReadVars(t, r)
	res.Align(known, r.Match)
	l.Rows, l.Devices = res.Rows, res.Devices
	return l, file, res, nil
}

// fillBook is the book as a template reads it.
type fillBook struct {
	sc    *schema.Set
	of    func(string) *model.Dataset
	hosts []string
	keys  map[string]string // folded name → host name
}

func (e *Env) fillBook() (*fillBook, []string, error) {
	_, bk, err := e.readBook()
	if err != nil || bk == nil {
		return nil, nil, err
	}
	ds, problems := bk.Expand(e.Schema)
	var notes []string
	for _, p := range problems {
		notes = append(notes, "ブックの注意: "+p.String())
	}
	b := &fillBook{sc: e.Schema, of: render.ByDevice(e.Schema, ds), hosts: book.Hosts(ds), keys: map[string]string{}}
	for _, h := range b.hosts {
		b.keys[listimport.DeviceKey(h)] = h
	}
	return b, notes, nil
}

func (b *fillBook) table(name string) *schema.Table {
	name = strings.TrimSpace(name)
	for _, t := range b.sc.Tables {
		if strings.EqualFold(t.Name, name) || t.Title.JA == name || strings.EqualFold(t.Title.EN, name) {
			return t
		}
	}
	return nil
}

func (b *fillBook) Table(name string) (string, bool) {
	if t := b.table(name); t != nil {
		return t.Name, true
	}
	return "", false
}

func (b *fillBook) Column(table, name string) (string, bool) {
	t := b.table(table)
	if t == nil {
		return "", false
	}
	name = strings.TrimSpace(name)
	for _, c := range t.Columns {
		if strings.EqualFold(c.Name, name) || c.Title.JA == name || strings.EqualFold(c.Title.EN, name) {
			return c.Name, true
		}
	}
	return "", false
}

func (b *fillBook) Has(device string) bool { _, ok := b.keys[device]; return ok }

func (b *fillBook) Rows(device, table string) []map[string]string {
	h, ok := b.keys[device]
	if !ok {
		return nil
	}
	tab := b.of(h).Tables[table]
	if tab == nil {
		return nil
	}
	var out []map[string]string
	for _, r := range tab.Rows {
		m := map[string]string{}
		for _, c := range r.Columns() {
			m[c] = r.Get(c)
		}
		out = append(out, m)
	}
	return out
}

func (b *fillBook) Devices() []string { return b.hosts }

func (b *fillBook) Secret(table, column string) bool {
	if t := b.table(table); t != nil {
		for _, c := range t.Columns {
			if c.Name == column {
				return c.Secret
			}
		}
	}
	return false
}

// FillOptions say what to make.
type FillOptions struct {
	Template string   // the name of a template of fill/
	Diff     string   // "" = the whole text; old | new | logs = the change against it
	Devices  []string // "" = every device the template is for
}

// FillDevice is the outcome of one device.
type FillDevice struct {
	Name      string
	Made      bool
	Unchanged bool   // a change with nothing to change
	Why       string // why it was not made
	File      string // the file written, from the case folder
	Lines     int
	Text      string
	Pack      string // the pack of the configuration now (a change)
	Values    []string
	Added     []fill.Item
	Removed   []fill.Item
	Kept      []fill.Item
	Left      []string
}

// FillOutcome is the outcome of filling one template.
type FillOutcome struct {
	Template string
	Diff     string
	Names    []fill.NameUse
	Devices  []*FillDevice
	Notes    []string
	Dir      string // out/<time>/fill/<template>
	Report   Files
}

// NotMade counts the devices not made.
func (o *FillOutcome) NotMade() int {
	n := 0
	for _, d := range o.Devices {
		if !d.Made && !d.Unchanged {
			n++
		}
	}
	return n
}

var secretName = regexp.MustCompile(`(?i)pass|secret|key|community`)

// fillSources reads what a template's names are looked up in. The device
// names of the lists are matched with the book's and with more (the hosts
// of the configurations a change is made against).
func (e *Env) fillSources(more ...string) (*fill.Sources, map[string]error, []string, error) {
	rules, err := e.varRules()
	if err != nil {
		return nil, nil, nil, err
	}
	src := &fill.Sources{}
	bk, bnotes, err := e.fillBook()
	if err != nil {
		return nil, nil, nil, err
	}
	known := append([]string{}, more...)
	if bk != nil {
		src.Book = bk
		known = append(known, bk.hosts...)
	}
	failed := map[string]error{}
	var notes []string
	for _, r := range rules {
		l, file, res, err := e.fillList(r, known)
		src.Lists = append(src.Lists, l)
		if err != nil {
			failed[l.Name] = err
			continue
		}
		if res.Excluded > 0 {
			notes = append(notes, fmt.Sprintf("変数ルール「%s」（%s）: 除外の条件に当たった %d 行は読みませんでした", r.Name, file, res.Excluded))
		}
		if len(res.Missing) > 0 {
			notes = append(notes, fmt.Sprintf("変数ルール「%s」（%s）: 一覧に見出し「%s」がありません。空として読みました", r.Name, file, strings.Join(res.Missing, "」「")))
		}
		for _, w := range res.Warnings {
			notes = append(notes, fmt.Sprintf("変数ルール「%s」（%s）: %s", r.Name, file, w))
		}
	}
	return src, failed, append(notes, bnotes...), nil
}

// FillSourcesForCheck gives what a template's names are looked up in, for
// checking a template as it is being written: a list that cannot be read
// still has its names.
func (e *Env) FillSourcesForCheck() (*fill.Sources, error) {
	src, _, _, err := e.fillSources()
	return src, err
}

// readTemplate finds a template by its name or its path.
func (e *Env) readTemplate(name string) (*fill.Template, string, error) {
	name = strings.TrimSpace(strings.TrimPrefix(strings.ReplaceAll(name, `\`, "/"), fillDir+"/"))
	name = strings.TrimSuffix(name, ".txt")
	ts, err := e.FillTemplates()
	if err != nil {
		return nil, "", err
	}
	for _, t := range ts {
		if strings.EqualFold(t.Name, name) {
			if t.Err != "" {
				return nil, t.Path, fmt.Errorf("ひな形 %s が読めません:\n  %s", t.Path, t.Err)
			}
			tp, err := fill.Parse(t.Name, t.Text)
			return tp, t.Path, err
		}
	}
	var names []string
	for _, t := range ts {
		names = append(names, t.Name)
	}
	if len(names) == 0 {
		return nil, "", fmt.Errorf("fill/ にひな形（.txt）がありません")
	}
	return nil, "", fmt.Errorf("ひな形「%s」がありません（fill/ にあるもの: %s）", name, strings.Join(names, "、"))
}

// Fill fills a template for its devices and writes the files and the report
// (spec 9.7.3, 9.7.4).
func (e *Env) Fill(opt FillOptions, now time.Time) (*FillOutcome, error) {
	if opt.Diff != "" && opt.Diff != "old" && opt.Diff != "new" && opt.Diff != "logs" {
		return nil, fmt.Errorf("今のコンフィグ「%s」は使えません（old・new・logs）", opt.Diff)
	}
	tp, _, err := e.readTemplate(opt.Template)
	if err != nil {
		return nil, err
	}
	// The configurations now, by folded host name.
	type now_ struct{ text, pack, file string }
	current := map[string]now_{}
	var hosts, readNotes []string
	if opt.Diff != "" {
		files, fails, err := e.treeSide(opt.Diff, now, nil)
		if err != nil {
			return nil, err
		}
		for _, f := range fails {
			readNotes = append(readNotes, "読めなかったファイル: "+f.Path+": "+f.Why)
		}
		for _, f := range files {
			var lines []string
			for i, l := range f.Listing.Source.Lines {
				if f.Listing.Source.Outside == nil || !f.Listing.Source.Outside[i] {
					lines = append(lines, l)
				}
			}
			current[listimport.DeviceKey(f.Host)] = now_{strings.Join(lines, "\n") + "\n", f.Listing.Pack, f.Listing.Path}
			hosts = append(hosts, f.Host)
		}
	}
	src, failed, notes, err := e.fillSources(hosts...)
	if err != nil {
		return nil, err
	}
	notes = append(readNotes, notes...)
	b, errs := fill.Bind(tp, src)
	if len(errs) > 0 {
		var msgs []string
		for _, e := range errs {
			msgs = append(msgs, e.Error())
		}
		return nil, fmt.Errorf("ひな形「%s」の変数を読めません:\n  %s", tp.Name, strings.Join(msgs, "\n  "))
	}
	for _, l := range b.Lists {
		if err := failed[l.Name]; err != nil {
			return nil, fmt.Errorf("変数ルール「%s」の一覧を読めません: %v", l.Name, err)
		}
	}
	if b.UsesBook && src.Book == nil {
		return nil, fmt.Errorf("%s がありません（ひな形がブックの値を使っています）", e.Proj.Config.Book)
	}
	o := &FillOutcome{Template: tp.Name, Diff: opt.Diff, Names: b.Names, Notes: notes}
	devices := b.Devices()
	if len(opt.Devices) > 0 {
		want := map[string]bool{}
		for _, d := range opt.Devices {
			if k := listimport.DeviceKey(d); k != "" {
				want[k] = true
			}
		}
		var keep []fill.Device
		for _, d := range devices {
			if want[d.Key] {
				keep = append(keep, d)
				delete(want, d.Key)
			}
		}
		if len(want) > 0 {
			var miss []string
			for k := range want {
				miss = append(miss, k)
			}
			sort.Strings(miss)
			return nil, fmt.Errorf("ひな形「%s」の対象に無い機器です: %s", tp.Name, strings.Join(miss, "、"))
		}
		devices = keep
	}
	if len(devices) == 0 {
		return nil, fmt.Errorf("ひな形「%s」の対象の機器がありません（変数ルールの一覧に機器の行が無いか、ブックに機器がありません）", tp.Name)
	}
	mask := e.mask()
	o.Dir = e.outDir(now) + "/" + fillDir + "/" + safeName(tp.Name)
	for _, dv := range devices {
		d := &FillDevice{Name: dv.Name, Values: e.fillValues(b, src, dv.Key, mask)}
		o.Devices = append(o.Devices, d)
		out, err := b.Fill(dv.Name)
		if err != nil {
			d.Why = err.Error()
			continue
		}
		d.Left = out.Left
		text := out.Text(tp.CRLF)
		if opt.Diff != "" {
			cur, ok := current[dv.Key]
			switch {
			case !ok:
				d.Why = fmt.Sprintf("今のコンフィグ（%s）に、この機器のファイルがありません", sideTitle(opt.Diff))
				continue
			case cur.pack == "" || e.Packs[cur.pack] == nil:
				d.Why = fmt.Sprintf("今のコンフィグ（%s）のメーカーを判定できません", cur.file)
				continue
			}
			d.Pack = cur.pack
			delta, err := fill.MakeDelta(out, cur.text, e.Packs[cur.pack])
			if err != nil {
				d.Why = err.Error()
				continue
			}
			d.Added, d.Removed, d.Kept = delta.Added, delta.Removed, delta.Kept
			if delta.Empty() {
				d.Unchanged = true
				continue
			}
			nl := "\n"
			if tp.CRLF {
				nl = "\r\n"
			}
			text = strings.Join(delta.Commands, nl) + nl
		}
		d.Made, d.Text = true, text
		d.Lines = strings.Count(text, "\n")
		d.File = o.Dir + "/" + safeName(dv.Name) + ".txt"
	}
	made := 0
	for _, d := range o.Devices {
		if d.Made {
			made++
		}
	}
	if made > 0 {
		if err := os.MkdirAll(e.Proj.Path(o.Dir), 0o755); err != nil {
			return nil, fmt.Errorf("%s を作れません: %v", o.Dir, err)
		}
		for _, d := range o.Devices {
			if d.Made {
				if err := os.WriteFile(e.Proj.Path(d.File), []byte(d.Text), 0o644); err != nil {
					return nil, fmt.Errorf("%s を書けません: %v", d.File, err)
				}
			}
		}
	} else {
		o.Dir = ""
	}
	files, err := e.writeReport(now, "fill", e.fillDoc(o, now))
	if err != nil {
		return nil, err
	}
	o.Report = files
	kind := "全文"
	if opt.Diff != "" {
		kind = "今（" + opt.Diff + "）との差"
	}
	e.Proj.Log(now, "ひな形から作る: %s（%s）、機器 %d、作った %d、作らなかった %d", tp.Name, kind, len(o.Devices), made, o.NotMade())
	return o, nil
}

// fillValues are a device's values of the names a template uses, for the
// report: "name = value", secrets hidden.
func (e *Env) fillValues(b *fill.Bound, src *fill.Sources, key string, mask bool) []string {
	var out []string
	for _, u := range b.Names {
		v := ""
		if u.List != "" {
			for _, l := range src.Lists {
				if l.Name == u.List {
					if rows := l.Rows[key]; len(rows) > 0 {
						v = rows[0].Values[u.Var]
						if len(rows) > 1 {
							v += fmt.Sprintf("（ほか %d 行）", len(rows)-1)
						}
					}
				}
			}
			if v != "" && mask && secretName.MatchString(u.Var) {
				v = masked
			}
		} else if src.Book != nil {
			if rows := src.Book.Rows(key, u.Table); len(rows) > 0 {
				v = rows[0][u.Column]
			}
			if v != "" && mask && src.Book.Secret(u.Table, u.Column) {
				v = masked
			}
		}
		out = append(out, u.Name+" = "+v)
	}
	return out
}

func itemText(it fill.Item) string {
	if len(it.Path) == 0 {
		return it.Text
	}
	return strings.Join(it.Path, " > ") + " > " + it.Text
}

// fillDoc is the report of filling a template.
func (e *Env) fillDoc(o *FillOutcome, now time.Time) *report.Doc {
	d := &report.Doc{Title: "ひな形から作る", Facts: e.facts(now)}
	kind := "全文"
	if o.Diff != "" {
		kind = "今のコンフィグ（" + sideTitle(o.Diff) + "）との差から投入"
	}
	d.Facts = append(d.Facts, report.Pair{Label: "ひな形", Value: fillDir + "/" + o.Template + ".txt"}, report.Pair{Label: "作ったもの", Value: kind})
	if o.Dir != "" {
		d.Facts = append(d.Facts, report.Pair{Label: "出力", Value: o.Dir})
	}
	made, same := 0, 0
	for _, dv := range o.Devices {
		if dv.Made {
			made++
		}
		if dv.Unchanged {
			same++
		}
	}
	d.Counts = []report.Pair{{Label: "機器", Value: num(len(o.Devices))}, {Label: "作った", Value: num(made)}}
	if o.Diff != "" {
		d.Counts = append(d.Counts, report.Pair{Label: "変更なし", Value: num(same)})
	}
	d.Counts = append(d.Counts, report.Pair{Label: "作らなかった", Value: num(o.NotMade())})
	d.Notes = append(d.Notes, o.Notes...)
	if o.Diff != "" {
		d.Notes = append(d.Notes, "消す行（no …）は、ひな形の {{#sync}} の印の所だけです。「残る行」は、ひな形が書くブロックにあってひな形に無い今の行で、消していません。")
	}
	devs := report.Sheet{Name: "機器", Header: []string{"機器", "結果", "理由", "行数", "ファイル"}}
	vals := report.Sheet{Name: "値", Lead: "機器ごとの変数の値です（同じ機器の行が複数あるときは最初の行）。", Header: []string{"機器"}}
	for _, u := range o.Names {
		vals.Header = append(vals.Header, u.Name)
	}
	for _, dv := range o.Devices {
		res := "作った"
		switch {
		case dv.Unchanged:
			res = "変更なし"
		case !dv.Made:
			res = "作らなかった"
		}
		why := dv.Why
		if why == "" && len(dv.Left) > 0 {
			why = strings.Join(dv.Left, "\n")
		}
		lines := ""
		if dv.Made {
			lines = num(dv.Lines)
		}
		devs.Rows = append(devs.Rows, []string{dv.Name, res, why, lines, dv.File})
		row := []string{dv.Name}
		for _, v := range dv.Values {
			if i := strings.Index(v, " = "); i >= 0 {
				v = v[i+3:]
			}
			row = append(row, v)
		}
		vals.Rows = append(vals.Rows, row)
	}
	devs.Marks = make([]string, len(devs.Rows))
	for i, dv := range o.Devices {
		if !dv.Made && !dv.Unchanged {
			devs.Marks[i] = "unknown"
		}
	}
	d.Sheets = append(d.Sheets, devs, vals)
	if o.Diff != "" {
		df := report.Sheet{Name: "差", Lead: "+ 足す行、- 消す行（右は消すコマンド）、= 残る行（ひな形に無いが消していない）。", Header: []string{"機器", "印", "行", "消すコマンド"}}
		for _, dv := range o.Devices {
			for _, it := range dv.Added {
				df.Rows = append(df.Rows, []string{dv.Name, "+", itemText(it), ""})
			}
			for _, it := range dv.Removed {
				df.Rows = append(df.Rows, []string{dv.Name, "-", itemText(it), it.Cmd})
			}
			for _, it := range dv.Kept {
				df.Rows = append(df.Rows, []string{dv.Name, "=", itemText(it), ""})
			}
		}
		d.Sheets = append(d.Sheets, df)
	}
	return d
}
