package ops

import (
	"fmt"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/importer"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/report"
	"github.com/kojio145/configchief/internal/schema"
)

// The list of many sites (spec 12.5): one table with the devices side by
// side, a row for each item (a row of the table and a column), and only the
// values that are not the same on every device.

// What stands in the list for a value that is not there.
const (
	siteEmpty  = "（空）"
	siteNoRow  = "—"
	siteSplit  = "（ばらばら）"
	siteRow    = "（行があるか）"
	siteOther  = masked + "（別の値）"
	siteHas    = "あり"
	siteHasNot = "なし"
	// Why a log is not in the list.
	siteNoConfig = "設定全体の表示（show running-config など）がないので、並べていません"
)

// SiteItem is one item of a table that is not the same on every device.
type SiteItem struct {
	Row, Column string // by the names the book shows; Column is siteRow for the row itself
	// Common is the value most devices have, CommonN how many have it, Of
	// how many devices have the row. Split: no value is held by more than
	// half of them.
	Common      string
	CommonN, Of int
	Split       bool
	// Values has one entry per device: "" for the common value, siteNoRow
	// for a device without the row.
	Values []string
	Differ int
}

// SiteTable is the items of one table.
type SiteTable struct {
	Name, Title string
	Items       []SiteItem
	Same        int // the items that are the same on every device
}

// SiteOutlier is a device that holds another value than most.
type SiteOutlier struct {
	Table, Row, Column, Device, Value, Common string
	CommonN, Of                               int
}

// SitesOutcome is the devices of one source side by side.
type SitesOutcome struct {
	Source   string
	Devices  []string
	Tables   []SiteTable
	Outliers []SiteOutlier
	Notes    []string
	Failed   []string // the files that were not read, with why
	Report   Files

	data *model.Dataset // what the tables were made of
}

// SiteSources are the sources the list can be made of.
var SiteSources = []string{"book", "old", "new", "logs"}

func siteSourceTitle(source string) string {
	switch source {
	case "book":
		return "パラメータブック"
	case "old":
		return "旧コンフィグ（input/old）"
	case "new":
		return "新コンフィグ（input/new）"
	}
	return "投入後のログ（input/logs）"
}

// importLogs reads the configurations the logs of input/logs show into
// design data. A log that shows none is told and left out.
func (e *Env) importLogs() (*Source, []string, error) {
	paths, err := e.Proj.Inputs("logs")
	if err != nil {
		return nil, nil, err
	}
	if len(paths) == 0 {
		return nil, nil, fmt.Errorf("input/logs にファイルがありません")
	}
	chosen, err := e.Proj.Sources()
	if err != nil {
		return nil, nil, err
	}
	src := &Source{Name: "logs", Data: model.NewDataset("logs")}
	var notes []string
	seen := map[string]string{}
	for i, path := range paths {
		if e.Tick != nil {
			if err := e.Tick(i, len(paths), path); err != nil {
				return nil, nil, err
			}
		}
		f := &File{Path: path}
		src.Files = append(src.Files, f)
		text, err := project.ReadText(e.Proj.Path(path))
		if err != nil {
			f.Err = err.Error()
			continue
		}
		if strings.TrimSpace(text) == "" {
			f.Err = "中身が空です"
			continue
		}
		var clear string
		f.Candidates, clear = e.detect(text)
		switch {
		case clear != "":
			f.Pack = clear
		case chosen.Pack(path) != "" && e.Packs[chosen.Pack(path)] != nil:
			f.Pack, f.Chosen = chosen.Pack(path), true
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
				notes = append(notes, path+": "+siteNoConfig)
				src.Files = src.Files[:len(src.Files)-1]
			} else {
				f.Err = "メーカーを判定できません（" + candidateText(f.Candidates) + "）。state/sources.yaml にパックを書いてください"
			}
			continue
		}
		p := e.Packs[f.Pack]
		if cut := p.Session.Cut(text); !cut.Log && cut.Prompts >= 1 {
			notes = append(notes, path+": "+siteNoConfig)
			src.Files = src.Files[:len(src.Files)-1]
			continue
		}
		res, err := importer.Import(p, e.Schema, path, text)
		if err != nil {
			f.Err = err.Error()
			continue
		}
		if first, dup := seen[res.Device]; dup {
			f.Err = fmt.Sprintf("ホスト名 %s が %s と同じです。このファイルは取り込みません", res.Device, first)
			continue
		}
		seen[res.Device] = path
		f.Result = res
		for _, name := range res.Data.TableNames() {
			for _, r := range res.Data.Tables[name].Rows {
				src.Data.Adopt(r)
			}
		}
	}
	return src, notes, nil
}

// siteData gives the design data of a source and its devices in order.
func (e *Env) siteData(source string) (ds *model.Dataset, devices, notes, failed []string, err error) {
	var src *Source
	switch source {
	case "book":
		_, bk, err := e.readBook()
		if err != nil {
			return nil, nil, nil, nil, err
		}
		if bk == nil {
			return nil, nil, nil, nil, fmt.Errorf("%s がありません。先にパラメータブックを作ってください", e.Proj.Config.Book)
		}
		ds, problems, fill := e.bookData(bk)
		for _, p := range problems {
			notes = append(notes, "ブックの注意: "+p.String())
		}
		for _, p := range bk.Problems {
			notes = append(notes, "ブックの注意: "+p.String())
		}
		notes = append(notes, fill...)
		return ds, book.Hosts(ds), notes, nil, nil
	case "old", "new":
		if src, err = e.importSide(source, ""); err != nil {
			return nil, nil, nil, nil, err
		}
	case "logs":
		if src, notes, err = e.importLogs(); err != nil {
			return nil, nil, nil, nil, err
		}
	default:
		return nil, nil, nil, nil, fmt.Errorf("元「%s」は並べられません（%s）", source, strings.Join(SiteSources, "・"))
	}
	for _, f := range src.Files {
		if f.Err != "" {
			failed = append(failed, f.Path+": "+f.Err)
		}
	}
	return src.Data, src.Devices(), notes, failed, nil
}

// Sites sets the devices of one source side by side and writes the report
// (spec 12.5).
func (e *Env) Sites(source string, now time.Time) (*SitesOutcome, error) {
	ds, devices, notes, failed, err := e.siteData(source)
	if err != nil {
		return nil, err
	}
	if len(devices) < 2 {
		return nil, fmt.Errorf("%s の機器が %d 台です。並べて比べるには 2 台以上が要ります", siteSourceTitle(source), len(devices))
	}
	o := &SitesOutcome{Source: source, Devices: devices, Notes: notes, Failed: failed, data: ds}
	o.Tables, o.Outliers = e.siteTables(ds, devices)
	files, err := e.writeReport(now, "sites", e.sitesDoc(o, now))
	if err != nil {
		return nil, err
	}
	o.Report = files
	e.Proj.Log(now, "多拠点の一覧: %s、機器 %d、外れ値 %d", source, len(devices), len(o.Outliers))
	return o, nil
}

// siteTables sets the rows of every table side by side and lists the
// devices that hold another value than most.
func (e *Env) siteTables(ds *model.Dataset, devices []string) (tables []SiteTable, outliers []SiteOutlier) {
	at := map[string]int{}
	for i, d := range devices {
		at[d] = i
	}
	for _, t := range e.Schema.Tables {
		devCol := t.DeviceColumn()
		tab := ds.Tables[t.Name]
		if devCol == "" || tab == nil {
			continue
		}
		st := e.siteTable(t, tab, devCol, at, len(devices))
		if len(st.Items)+st.Same == 0 {
			continue
		}
		for _, it := range st.Items {
			if it.Split {
				continue
			}
			for i, v := range it.Values {
				if v != "" && v != siteNoRow {
					outliers = append(outliers, SiteOutlier{st.Title, it.Row, it.Column, devices[i], v, it.Common, it.CommonN, it.Of})
				}
			}
		}
		tables = append(tables, st)
	}
	return tables, outliers
}

// siteTable sets the rows of one table side by side.
func (e *Env) siteTable(t *schema.Table, tab *model.Table, devCol string, at map[string]int, n int) SiteTable {
	st := SiteTable{Name: t.Name, Title: e.tableTitle(t.Name)}
	// The rows by what tells them apart inside a device, in the order they
	// come first.
	type line struct {
		label string
		rows  []*model.Row // one per device; nil = the device has no such row
	}
	var lines []*line
	byKey := map[string]*line{}
	for _, r := range tab.Rows {
		i, ok := at[r.Get(devCol)]
		if !ok {
			continue
		}
		var key []string
		for _, k := range r.Key {
			if k.Name != devCol {
				key = append(key, k.Value)
			}
		}
		id := strings.Join(key, "\x00")
		l := byKey[id]
		if l == nil {
			l = &line{label: e.rowLabel(r), rows: make([]*model.Row, n)}
			byKey[id] = l
			lines = append(lines, l)
		}
		if l.rows[i] == nil {
			l.rows[i] = r
		}
	}
	for _, l := range lines {
		have := 0
		for _, r := range l.rows {
			if r != nil {
				have++
			}
		}
		if have < n {
			// The row itself is not on every device.
			raw, shown := make([]string, n), make([]string, n)
			for i, r := range l.rows {
				raw[i], shown[i] = siteHasNot, siteHasNot
				if r != nil {
					raw[i], shown[i] = siteHas, siteHas
				}
			}
			st.Items = append(st.Items, siteItem(l.label, siteRow, raw, shown, nil, false))
		}
		if have < 2 {
			continue
		}
		for _, c := range t.Columns {
			if c.Name == devCol || t.IsKey(c.Name) {
				continue
			}
			raw, shown := make([]string, n), make([]string, n)
			all := true
			first := ""
			started := false
			for i, r := range l.rows {
				if r == nil {
					continue
				}
				raw[i] = r.Get(c.Name)
				shown[i] = e.cellText(t.Name, c.Name, r.Cells[c.Name])
				if raw[i] == "" {
					shown[i] = siteEmpty
				}
				if !started {
					first, started = raw[i], true
				} else if raw[i] != first {
					all = false
				}
			}
			if all {
				st.Same++
				continue
			}
			st.Items = append(st.Items, siteItem(l.label, e.columnTitle(t.Name, c.Name), raw, shown, l.rows, e.mask() && c.Secret))
		}
	}
	return st
}

// siteItem makes the item of one row and column. rows tells which devices
// have the row (nil = all); raw is what is compared, shown what is written.
func siteItem(row, column string, raw, shown []string, rows []*model.Row, secret bool) SiteItem {
	it := SiteItem{Row: row, Column: column, Values: make([]string, len(raw))}
	count := map[string]int{}
	var order []string
	for i, v := range raw {
		if rows != nil && rows[i] == nil {
			continue
		}
		it.Of++
		if count[v] == 0 {
			order = append(order, v)
		}
		count[v]++
	}
	common := ""
	for _, v := range order {
		if count[v] > it.CommonN {
			common, it.CommonN = v, count[v]
		}
	}
	it.Split = it.CommonN*2 <= it.Of
	for i, v := range raw {
		switch {
		case rows != nil && rows[i] == nil:
			it.Values[i] = siteNoRow
		case v == common && !it.Split:
			if it.Common == "" {
				it.Common = shown[i]
			}
		default:
			it.Values[i] = shown[i]
			if secret && !it.Split && v != "" {
				it.Values[i] = siteOther
			}
			if !it.Split {
				it.Differ++
			}
		}
	}
	if it.Split {
		it.Common, it.CommonN = siteSplit, 0
	}
	return it
}

// SitesDoc is the list of many sites as a screen shows it: the secrets are
// masked whatever the case says.
func (e *Env) SitesDoc(o *SitesOutcome, now time.Time) *report.Doc {
	s := e.forScreen()
	c := *o
	c.Tables, c.Outliers = s.siteTables(o.data, o.Devices)
	return s.sitesDoc(&c, now)
}

// sitesDoc is the report of the list of many sites.
func (e *Env) sitesDoc(o *SitesOutcome, now time.Time) *report.Doc {
	d := &report.Doc{Title: "多拠点の一覧", Facts: e.facts(now)}
	d.Facts = append(d.Facts, report.Pair{Label: "元", Value: siteSourceTitle(o.Source)})
	same, differ, split := 0, 0, 0
	for _, t := range o.Tables {
		same += t.Same
		for _, it := range t.Items {
			if it.Split {
				split++
			} else {
				differ++
			}
		}
	}
	d.Counts = []report.Pair{
		{Label: "機器", Value: num(len(o.Devices))},
		{Label: "全台で同じ項目", Value: num(same)},
		{Label: "違う機器がある項目", Value: num(differ)},
		{Label: "ばらばらの項目", Value: num(split)},
		{Label: "外れ値", Value: num(len(o.Outliers))},
		{Label: "読めなかったファイル", Value: num(len(o.Failed))},
	}
	d.Notes = append(d.Notes,
		"項目は「行（機器を除くキー）＋列」です。全台で同じ値の項目は出していません。",
		"機器の欄は、いちばん多い値と違う機器にだけ値を書いています（空欄 = いちばん多い値と同じ、"+siteNoRow+" = その行がない、"+siteEmpty+" = 値が空）。",
		"「"+siteSplit+"」は、いちばん多い値を持つ機器が半分以下の項目です（アドレスのように、もともと機器ごとに違う項目）。全機器の値を書いています。「外れ値」には入れていません。")
	if e.mask() {
		d.Notes = append(d.Notes, "秘密の列の値は "+masked+" に伏せています（いちばん多い値と違う値は「"+siteOther+"」）。")
	}
	for _, f := range o.Failed {
		d.Notes = append(d.Notes, "読めなかったファイル: "+f)
	}
	d.Notes = append(d.Notes, o.Notes...)

	sum := report.Sheet{Name: "表", Lead: "表ごとの項目の数です。", Header: []string{"表", "全台で同じ項目", "違う機器がある項目", "ばらばらの項目"}}
	for _, t := range o.Tables {
		df, sp := 0, 0
		for _, it := range t.Items {
			if it.Split {
				sp++
			} else {
				df++
			}
		}
		sum.Rows = append(sum.Rows, []string{t.Title, num(t.Same), num(df), num(sp)})
	}
	out := report.Sheet{Name: "外れ値", Lead: "いちばん多い値と違う値を持つ機器です（ばらばらの項目を除く）。",
		Header: []string{"表", "行", "項目", "機器", "値", "いちばん多い値", "その台数", "行がある台数"}}
	for _, x := range o.Outliers {
		out.Rows = append(out.Rows, []string{x.Table, x.Row, x.Column, x.Device, x.Value, x.Common, num(x.CommonN), num(x.Of)})
	}
	d.Sheets = append(d.Sheets, sum, out)
	for _, t := range o.Tables {
		if len(t.Items) == 0 {
			continue
		}
		sh := report.Sheet{Name: t.Title, Lead: fmt.Sprintf("全台で同じ項目 %d 個は出していません。", t.Same),
			Header: append([]string{"行", "項目", "いちばん多い値", "その台数", "違う台数"}, o.Devices...)}
		for _, it := range t.Items {
			n, df := num(it.CommonN), num(it.Differ)
			if it.Split {
				n, df = "", ""
			}
			sh.Rows = append(sh.Rows, append([]string{it.Row, it.Column, it.Common, n, df}, it.Values...))
		}
		d.Sheets = append(d.Sheets, sh)
	}
	return d
}
