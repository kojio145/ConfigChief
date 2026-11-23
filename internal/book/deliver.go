package book

import (
	"bytes"
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/xuri/excelize/v2"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
	"github.com/kojio145/configchief/internal/tmpl"
)

// The delivery sheets (spec 13.2): the values of the design data poured
// into a template the user made, or laid out in the standard form.

// DeliverOptions are what the user chooses for one run.
type DeliverOptions struct {
	Project string // project.name
	Mask    bool   // write *** for the secret values
	Lang    string
	// Revisions is the history of the versions ({{#rows revisions}}) and
	// Revision the version made now ({{ revision.version }}; nil = none).
	// Spec 13.7.
	Revisions []Revision
	Revision  *Revision
}

// Revision is one version of the history (spec 13.7).
type Revision struct {
	Version, Date, By, Summary string
}

// revisionTable tells the name {{#rows}} takes for the history.
func revisionTable(name string) bool { return name == "revisions" || name == "改版履歴" }

func (r *Revision) vars() map[string]any {
	if r == nil {
		return map[string]any{"version": "", "date": "", "by": "", "summary": ""}
	}
	return map[string]any{"version": r.Version, "date": r.Date, "by": r.By, "summary": r.Summary}
}

// Delivered is one file of delivery sheets.
type Delivered struct {
	Name string
	Data []byte
}

// maskText stands where a secret value is not shown.
const maskText = "***"

// rowsMark opens a row that is repeated for each row of a table.
var rowsMark = regexp.MustCompile(`\{\{\s*#rows\s+([^{}]+?)\s*\}\}`)

// plainNumber is a value Excel may hold as a number without changing how it
// looks: no sign, no leading zero, at most 15 digits.
var plainNumber = regexp.MustCompile(`^(0|[1-9][0-9]{0,14})$`)

// badFileChars may not stand in a file name on Windows.
const badFileChars = `\/:*?"<>|`

// deliver holds one run over one template.
type deliver struct {
	sc       *schema.Set
	ds       *model.Dataset
	opt      DeliverOptions
	eng      *tmpl.Engine
	hosts    []string
	problems []string
	// secret is set when the expression at hand read a secret column.
	secret bool
	// missing collects the names an expression asked for in vain.
	missing []string
	// revRow is the version of the history at hand in {{#rows revisions}}.
	revRow map[string]any
}

// shown is a row as an expression sees it: every cell as the book shows it.
type shown struct {
	d *deliver
	t *schema.Table
	r *model.Row
}

func (o shown) Field(name string) (any, bool) {
	c := o.t.Column(name)
	if c == nil {
		o.d.missing = append(o.d.missing, fmt.Sprintf("表 %s に列 %s はありません", o.t.Name, name))
		return "", true
	}
	if c.Secret || c.Type == "secret" {
		o.d.secret = true
	}
	return Show(c, o.r.Cells[c.Name], o.d.opt.Lang), true
}

func (o shown) Fields() []string {
	out := make([]string, len(o.t.Columns))
	for i, c := range o.t.Columns {
		out[i] = c.Name
	}
	return out
}

// nobody stands for "device" where no device is at hand.
type nobody struct{ d *deliver }

func (o nobody) Field(name string) (any, bool) {
	o.d.missing = append(o.d.missing, "device は、機器ごとのシート・ファイルの中と、機器の列を持つ表の {{#rows}} の行の中でだけ使えます")
	return "", true
}

func (o nobody) Fields() []string { return nil }

// Deliver pours the design data into one template. name is the file name
// of the template: with an expression in it, each device gets a file of its
// own. ds is the book spread out to the devices. The problems name the
// sheet and the cell; with any, no file comes back.
func Deliver(name string, template []byte, sc *schema.Set, ds *model.Dataset, opt DeliverOptions) ([]Delivered, []string) {
	d := &deliver{sc: sc, ds: ds, opt: opt, eng: tmpl.New(tmpl.Options{}), hosts: Hosts(ds)}
	if !strings.Contains(name, "{{") {
		data := d.pour(template, "")
		if len(d.problems) > 0 {
			return nil, d.problems
		}
		return []Delivered{{Name: name, Data: data}}, nil
	}
	if len(d.hosts) == 0 {
		return nil, []string{"機器が 1 台もないので、機器ごとのファイルを作れません"}
	}
	var out []Delivered
	seen := map[string]string{}
	for _, host := range d.hosts {
		file, ok := d.text(name, host, nil, nil)
		switch {
		case !ok:
			d.problems = append(d.problems, fmt.Sprintf("ファイル名（機器 %s）: 式を読めません", host))
			continue
		case strings.ContainsAny(file, badFileChars) || strings.TrimSpace(file) == "":
			d.problems = append(d.problems, fmt.Sprintf("ファイル名「%s」（機器 %s）は、ファイルの名前に使えません", file, host))
			continue
		case seen[strings.ToLower(file)] != "":
			d.problems = append(d.problems, fmt.Sprintf("ファイル名「%s」が、機器 %s と %s で重なります", file, seen[strings.ToLower(file)], host))
			continue
		}
		seen[strings.ToLower(file)] = host
		before := len(d.problems)
		data := d.pour(template, host)
		for i := before; i < len(d.problems); i++ {
			d.problems[i] = file + " の" + d.problems[i]
		}
		out = append(out, Delivered{Name: file, Data: data})
	}
	if len(d.problems) > 0 {
		return nil, d.problems
	}
	return out, nil
}

// rowsOf gives the rows of a table for a device ("" = every device, in the
// order of the book), each device's rows in the natural order of the keys.
func (d *deliver) rowsOf(t *schema.Table, host string) []*model.Row {
	tab := d.ds.Tables[t.Name]
	if tab == nil {
		return nil
	}
	devCol := t.DeviceColumn()
	if devCol == "" {
		return tab.Rows
	}
	pick := func(h string) []*model.Row {
		var rows []*model.Row
		for _, r := range tab.Rows {
			if r.Get(devCol) == h {
				rows = append(rows, r)
			}
		}
		sort.SliceStable(rows, func(i, j int) bool {
			a, b := rows[i], rows[j]
			for k := range a.Key {
				if k < len(b.Key) && a.Key[k].Value != b.Key[k].Value {
					return tmpl.NaturalLess(a.Key[k].Value, b.Key[k].Value)
				}
			}
			return false
		})
		return rows
	}
	if host != "" {
		return pick(host)
	}
	var out []*model.Row
	for _, h := range d.hosts {
		out = append(out, pick(h)...)
	}
	return out
}

// table finds a table by its inner name or by the name of its sheet.
func (d *deliver) table(name string) *schema.Table {
	if t := d.sc.Table(name); t != nil {
		return t
	}
	for _, t := range d.sc.Tables {
		if t.Title.JA == name || t.Title.EN == name {
			return t
		}
	}
	return nil
}

// text gives what a cell (or a name) turns into. host is the device of the
// sheet or the file ("" = none); t and row are the row of {{#rows}} (nil =
// none). ok is false when the expression is wrong; the reasons are in
// d.missing or were told by the engine.
func (d *deliver) text(src, host string, t *schema.Table, row *model.Row) (string, bool) {
	d.secret, d.missing = false, nil
	vars := map[string]any{"project": map[string]any{"name": d.opt.Project}, "revision": d.opt.Revision.vars()}
	dev := host
	if d.revRow != nil {
		vars["row"] = d.revRow
	}
	if row != nil {
		vars["row"] = shown{d, t, row}
		if col := t.DeviceColumn(); col != "" && dev == "" {
			dev = row.Get(col)
		}
	}
	vars["device"] = nobody{d}
	if dt := d.sc.Table("devices"); dt != nil && dev != "" {
		if tab := d.ds.Tables["devices"]; tab != nil {
			for _, r := range tab.Rows {
				if r.Get("hostname") == dev {
					vars["device"] = shown{d, dt, r}
				}
			}
		}
	}
	out, err := d.eng.Expand(src, vars)
	if err != nil {
		d.missing = append(d.missing, err.Error())
		return "", false
	}
	if len(d.missing) > 0 {
		return "", false
	}
	if d.secret && d.opt.Mask {
		return maskText, true
	}
	return out, true
}

// pour fills one copy of the template. host is the device of the whole file
// ("" = the file is for every device).
func (d *deliver) pour(template []byte, host string) []byte {
	f, err := excelize.OpenReader(bytes.NewReader(template))
	if err != nil {
		d.problems = append(d.problems, fmt.Sprintf("ひな形を開けません（xlsx のファイルですか）: %v", err))
		return nil
	}
	defer f.Close()
	names := map[string]bool{}
	for _, s := range f.GetSheetList() {
		names[strings.ToLower(s)] = true
	}
	// rename gives the name of a sheet for a device, or tells why not.
	rename := func(sheet, dev string) (string, bool) {
		name, ok := d.text(sheet, dev, nil, nil)
		switch {
		case !ok:
			d.problems = append(d.problems, fmt.Sprintf("シート名「%s」: %s", sheet, strings.Join(d.missing, "、")))
			return "", false
		case name == "" || utf8.RuneCountInString(name) > 31 || strings.ContainsAny(name, `:\/?*[]`):
			d.problems = append(d.problems, fmt.Sprintf("シート名「%s」（機器 %s）は、Excel のシートの名前に使えません（31 字まで。: \\ / ? * [ ] は使えません）", name, dev))
			return "", false
		case names[strings.ToLower(name)]:
			d.problems = append(d.problems, fmt.Sprintf("シート名「%s」（機器 %s）が、ほかのシートと重なります", name, dev))
			return "", false
		}
		names[strings.ToLower(name)] = true
		return name, true
	}
	for _, sheet := range f.GetSheetList() {
		switch {
		case !strings.Contains(sheet, "{{"):
			d.fill(f, sheet, host)
		case host != "":
			delete(names, strings.ToLower(sheet))
			if name, ok := rename(sheet, host); ok {
				if err := f.SetSheetName(sheet, name); err != nil {
					d.problems = append(d.problems, fmt.Sprintf("シート「%s」: %v", name, err))
					continue
				}
				d.fill(f, name, host)
			}
		default:
			// A copy for each device, in the place of the template sheet.
			if len(d.hosts) == 0 {
				d.problems = append(d.problems, fmt.Sprintf("シート「%s」: 機器が 1 台もないので、機器ごとのシートを作れません", sheet))
				continue
			}
			for _, dev := range d.hosts {
				name, ok := rename(sheet, dev)
				if !ok {
					continue
				}
				to, err := f.NewSheet(name)
				if err == nil {
					var from int
					if from, err = f.GetSheetIndex(sheet); err == nil {
						err = f.CopySheet(from, to)
					}
				}
				if err == nil {
					err = f.MoveSheet(name, sheet)
				}
				if err != nil {
					d.problems = append(d.problems, fmt.Sprintf("シート「%s」を写せません: %v", name, err))
					continue
				}
				d.fill(f, name, dev)
			}
			if err := f.DeleteSheet(sheet); err != nil {
				d.problems = append(d.problems, fmt.Sprintf("シート「%s」: %v", sheet, err))
			}
		}
	}
	if len(d.problems) > 0 {
		return nil
	}
	if list := f.GetSheetList(); len(list) > 0 {
		if i, err := f.GetSheetIndex(list[0]); err == nil {
			f.SetActiveSheet(i)
		}
	}
	var buf bytes.Buffer
	if err := f.Write(&buf); err != nil {
		d.problems = append(d.problems, fmt.Sprintf("書き出せません: %v", err))
		return nil
	}
	return buf.Bytes()
}

// fill turns the expressions of one sheet into values.
func (d *deliver) fill(f *excelize.File, sheet, host string) {
	rows, err := f.GetRows(sheet)
	if err != nil {
		d.problems = append(d.problems, fmt.Sprintf("シート「%s」を読めません: %v", sheet, err))
		return
	}
	at := func(col, row int) string {
		name, _ := excelize.CoordinatesToCellName(col+1, row)
		return name
	}
	put := func(cell, src string, t *schema.Table, row *model.Row) {
		val, ok := d.text(src, host, t, row)
		if !ok {
			d.problems = append(d.problems, fmt.Sprintf("シート「%s」の %s: %s", sheet, cell, strings.Join(d.missing, "、")))
			return
		}
		if plainNumber.MatchString(val) {
			if n, err := strconv.ParseInt(val, 10, 64); err == nil {
				err = f.SetCellInt(sheet, cell, n)
				if err == nil {
					return
				}
			}
		}
		if err := f.SetCellStr(sheet, cell, val); err != nil {
			d.problems = append(d.problems, fmt.Sprintf("シート「%s」の %s: %v", sheet, cell, err))
		}
	}
	shift := 0 // how far the rows below have moved
	for i, cells := range rows {
		line := i + 1 + shift
		marks := 0
		var t *schema.Table
		history := false
		for _, c := range cells {
			if m := rowsMark.FindAllStringSubmatch(c, -1); len(m) > 0 {
				marks += len(m)
				if revisionTable(m[0][1]) {
					history = true
					continue
				}
				t = d.table(m[0][1])
				if t == nil {
					d.problems = append(d.problems, fmt.Sprintf("シート「%s」の %d 行目: {{#rows %s}} の表がありません", sheet, line, m[0][1]))
				}
			}
		}
		switch {
		case marks == 0:
			for c, src := range cells {
				if strings.Contains(src, "{{") {
					put(at(c, line), src, nil, nil)
				}
			}
			continue
		case marks > 1:
			d.problems = append(d.problems, fmt.Sprintf("シート「%s」の %d 行目: {{#rows}} が 2 つ以上あります（1 つの行に 1 つです）", sheet, line))
			continue
		case history:
			// The history of the versions (spec 13.7).
			n := len(d.opt.Revisions)
			if n == 0 {
				if err := f.RemoveRow(sheet, line); err != nil {
					d.problems = append(d.problems, fmt.Sprintf("シート「%s」の %d 行目: %v", sheet, line, err))
				}
				shift--
				continue
			}
			for k := 1; k < n; k++ {
				if err := f.DuplicateRow(sheet, line); err != nil {
					d.problems = append(d.problems, fmt.Sprintf("シート「%s」の %d 行目を増やせません: %v", sheet, line, err))
					break
				}
			}
			for k := range d.opt.Revisions {
				d.revRow = d.opt.Revisions[k].vars()
				for c, src := range cells {
					if rowsMark.MatchString(src) || strings.Contains(src, "{{") {
						put(at(c, line+k), rowsMark.ReplaceAllString(src, ""), nil, nil)
					}
				}
			}
			d.revRow = nil
			shift += n - 1
			continue
		case t == nil:
			continue
		}
		data := d.rowsOf(t, host)
		if len(data) == 0 {
			if err := f.RemoveRow(sheet, line); err != nil {
				d.problems = append(d.problems, fmt.Sprintf("シート「%s」の %d 行目: %v", sheet, line, err))
			}
			shift--
			continue
		}
		for k := 1; k < len(data); k++ {
			if err := f.DuplicateRow(sheet, line); err != nil {
				d.problems = append(d.problems, fmt.Sprintf("シート「%s」の %d 行目を増やせません: %v", sheet, line, err))
				break
			}
		}
		for k, r := range data {
			for c, src := range cells {
				if src = rowsMark.ReplaceAllString(src, ""); strings.Contains(src, "{{") {
					put(at(c, line+k), src, t, r)
				} else if rowsMark.MatchString(cells[c]) {
					put(at(c, line+k), src, t, r) // the mark alone, or with plain text
				}
			}
		}
		shift += len(data) - 1
	}
}

// Standard lays the design data out as the book does, one sheet for each
// table that has rows: the form of the delivery sheets when the case has no
// template (spec 13.2).
func Standard(sc *schema.Set, ds *model.Dataset, opt DeliverOptions) ([]byte, error) {
	d := &deliver{sc: sc, ds: ds, opt: opt, hosts: Hosts(ds)}
	var grids []Grid
	for _, t := range sc.Tables {
		rows := d.rowsOf(t, "")
		if len(rows) == 0 {
			continue
		}
		g := TableGrid(t, rows, opt.Lang)
		g.Notes, g.Choices = nil, nil
		if opt.Mask {
			for i, c := range t.Columns {
				if !c.Secret && c.Type != "secret" {
					continue
				}
				for _, line := range g.Rows {
					if line[i] != "" {
						line[i] = maskText
					}
				}
			}
		}
		grids = append(grids, g)
	}
	if len(grids) == 0 {
		return nil, fmt.Errorf("パラメータブックに、書き出す行がありません")
	}
	var buf bytes.Buffer
	if err := WriteGrids(&buf, grids); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
