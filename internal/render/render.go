package render

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/importer"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/pack"
	"github.com/kojio145/configchief/internal/schema"
	"github.com/kojio145/configchief/internal/tmpl"
)

// ExtraFile is an extra configuration of the case (spec 9.4). The name tells
// the devices it goes to: "_all.j2", "@<group>.j2" or "<hostname>.j2".
type ExtraFile struct {
	Name string
	Text string
}

// CellRef names a cell of the rows a configuration was made from.
type CellRef struct {
	Row    *model.Row
	Column string
}

// Line is one line of a generated configuration with what it was made from
// (spec 9.3).
type Line struct {
	Text     string
	Section  string
	Template string // the template that wrote the line; "extra/<file>" for an extra configuration
	TLine    int    // the line in the template; 0 if not known
	Extra    bool   // written by a person (an extra configuration)
	Values   []CellRef
	Conds    []CellRef
}

// Output is the generated configuration of one device.
type Output struct {
	Device string
	Text   string // the lines, each ended by "\n"
	Lines  []Line
	// Failed tells that a template failed: the text is what there is of it.
	Failed bool
	// Data is the rows of the device as the templates saw them.
	Data *model.Dataset

	sc     *schema.Set
	unused map[string]map[string]bool // table + "\x00" + key → columns G-602 told about
}

// Renderer generates the configurations of the devices of one pack.
type Renderer struct {
	p      *pack.Pack
	sc     *schema.Set
	eng    *tmpl.Engine
	extras []extra

	reads []CellRef
	used  map[*model.Row]map[string]bool
	// whole is what was worked out of the dataset of the last call, once for
	// all its devices.
	whole *whole
}

// whole is what the templates of every device get of one dataset: with
// hundreds of devices, working it out again for each device takes most of
// the time of a generation (spec 16).
type whole struct {
	ds    *model.Dataset
	rows  map[string]map[string][]*model.Row // table → device → its rows, in the order of the table
	heads map[string]map[string]bool         // table → the free columns of its sheet
	all   any                                // the rows of all devices, as the templates take them
}

// of gives what is worked out of a dataset. The dataset must not change
// between the calls for its devices.
func (r *Renderer) of(ds *model.Dataset) *whole {
	if r.whole != nil && r.whole.ds == ds {
		return r.whole
	}
	w := &whole{ds: ds, rows: map[string]map[string][]*model.Row{}, heads: map[string]map[string]bool{}}
	all := map[string]any{}
	for _, t := range r.sc.Tables {
		devCol := t.DeviceColumn()
		if devCol == "" {
			continue
		}
		h := map[string]bool{}
		by := map[string][]*model.Row{}
		others := []any{}
		if tab := ds.Tables[t.Name]; tab != nil {
			for _, row := range tab.Rows {
				for _, kv := range row.Extra {
					h[kv.Name] = true
				}
				dev := row.Get(devCol)
				by[dev] = append(by[dev], row)
				others = append(others, rowRec{r, t, row, false, h})
			}
		}
		w.heads[t.Name], w.rows[t.Name], all[t.Name] = h, by, others
	}
	w.all = tmpl.Prepare(all)
	r.whole = w
	return w
}

type extra struct {
	file   string
	where  string // "after" | "before" | ""
	id     string
	offset int // lines taken off the top of the file
}

var rePosition = regexp.MustCompile(`^\{#\s*position:\s*(after|before)\s+(\S+)\s*#\}\s*$`)

// New prepares the templates of a pack and the extra configurations of a
// case.
func New(p *pack.Pack, sc *schema.Set, extras []ExtraFile) *Renderer {
	r := &Renderer{p: p, sc: sc, eng: p.Engine()}
	src := map[string]string{}
	for name, text := range p.Templates {
		src[name] = text
	}
	for _, e := range extras {
		x := extra{file: e.Name}
		text := strings.ReplaceAll(e.Text, "\r\n", "\n")
		first, rest, _ := strings.Cut(text, "\n")
		if m := rePosition.FindStringSubmatch(first); m != nil {
			x.where, x.id, x.offset = m[1], m[2], 1
			text = rest
		}
		src["extra/"+e.Name] = text
		r.extras = append(r.extras, x)
	}
	sort.Slice(r.extras, func(i, j int) bool { return r.extras[i].file < r.extras[j].file })
	r.eng.SetLoader(func(name string) (string, bool) {
		s, ok := src[name]
		return s, ok
	})
	return r
}

// rowRec hands a row to a template, with each value in the form of its
// column (spec 9.6), and notes which cells were read.
type rowRec struct {
	r     *Renderer
	t     *schema.Table
	row   *model.Row
	track bool
	heads map[string]bool // the free columns of the sheet
}

func (o rowRec) Field(name string) (any, bool) {
	if name == "extra" && o.t.Column(name) == nil {
		return extraRec{o.row, o.heads}, true
	}
	c := o.t.Column(name)
	if c == nil {
		return nil, false
	}
	cell := o.row.Cells[name]
	if o.track && cell != nil && len(cell.Values) > 0 {
		o.r.reads = append(o.r.reads, CellRef{o.row, name})
		if o.r.used[o.row] == nil {
			o.r.used[o.row] = map[string]bool{}
		}
		o.r.used[o.row][name] = true
	}
	return typed(c, cell), true
}

func (o rowRec) Fields() []string {
	out := make([]string, len(o.t.Columns))
	for i, c := range o.t.Columns {
		out[i] = c.Name
	}
	return out
}

// extraRec is the free columns of a row: a heading the sheet has is "" where
// the cell is empty, and a heading it does not have is undefined.
type extraRec struct {
	row   *model.Row
	heads map[string]bool
}

func (o extraRec) Field(name string) (any, bool) {
	for _, kv := range o.row.Extra {
		if kv.Name == name {
			return kv.Value, true
		}
	}
	return "", o.heads[name]
}

func (o extraRec) Fields() []string {
	var out []string
	for h := range o.heads {
		out = append(out, h)
	}
	sort.Strings(out)
	return out
}

func typed(c *schema.Column, cell *model.Cell) any {
	if cell == nil {
		return c.Value(nil)
	}
	return c.Value(cell.Values)
}

func cloneRow(r *model.Row) *model.Row {
	c := &model.Row{Table: r.Table, Key: append([]model.KV{}, r.Key...), Cells: map[string]*model.Cell{}, Origins: r.Origins, Extra: r.Extra}
	for name, cell := range r.Cells {
		cp := *cell
		cp.Values = append([]string{}, cell.Values...)
		cp.Origins = append([]model.Origin{}, cell.Origins...)
		c.Cells[name] = &cp
	}
	return c
}

// DeviceData copies the rows of one device out of expanded design data, each
// table sorted by key.
func DeviceData(sc *schema.Set, ds *model.Dataset, device string) *model.Dataset {
	return deviceData(sc, ds, device)
}

// ByDevice gives the same as DeviceData for every device of a dataset: the
// rows are told apart once, not once for each device.
func ByDevice(sc *schema.Set, ds *model.Dataset) func(device string) *model.Dataset {
	rows := map[string]map[string][]*model.Row{}
	for _, t := range sc.Tables {
		devCol := t.DeviceColumn()
		if tab := ds.Tables[t.Name]; tab != nil && devCol != "" {
			by := map[string][]*model.Row{}
			for _, row := range tab.Rows {
				dev := row.Get(devCol)
				by[dev] = append(by[dev], row)
			}
			rows[t.Name] = by
		}
	}
	return func(device string) *model.Dataset {
		return copyRows(sc, ds, func(t *schema.Table) []*model.Row { return rows[t.Name][device] })
	}
}

// deviceData copies the rows of one device, each table sorted by key.
func deviceData(sc *schema.Set, ds *model.Dataset, device string) *model.Dataset {
	return copyRows(sc, ds, func(t *schema.Table) []*model.Row {
		src := ds.Tables[t.Name]
		devCol := t.DeviceColumn()
		if src == nil || devCol == "" {
			return nil
		}
		var rows []*model.Row
		for _, r := range src.Rows {
			if r.Get(devCol) == device {
				rows = append(rows, r)
			}
		}
		return rows
	})
}

// copyRows copies the rows of each table that of gives, sorted by key.
func copyRows(sc *schema.Set, ds *model.Dataset, of func(t *schema.Table) []*model.Row) *model.Dataset {
	out := model.NewDataset(ds.Source)
	for _, t := range sc.Tables {
		src := of(t)
		rows := make([]*model.Row, len(src))
		for i, r := range src {
			rows[i] = cloneRow(r)
		}
		sort.SliceStable(rows, func(i, j int) bool { return keyLess(rows[i], rows[j]) })
		for _, r := range rows {
			out.Adopt(r)
		}
	}
	return out
}

func keyLess(a, b *model.Row) bool {
	for i := range a.Key {
		if i >= len(b.Key) {
			return false
		}
		if a.Key[i].Value != b.Key[i].Value {
			return tmpl.NaturalLess(a.Key[i].Value, b.Key[i].Value)
		}
	}
	return false
}

func rowKey(r *model.Row) string { return r.Table + "\x00" + r.KeyText() }

// part is the text of one section, or of one extra configuration.
type part struct {
	section string
	lines   []Line
}

// Render generates the configuration of one device from the expanded design
// data. project is what the templates see as `project`.
func (r *Renderer) Render(ds *model.Dataset, device string, project map[string]any) (*Output, []check.Finding) {
	out := &Output{Device: device, sc: r.sc, unused: map[string]map[string]bool{}}
	var findings []check.Finding
	if len(r.p.Sections) == 0 {
		out.Failed = true
		return out, []check.Finding{check.New("G-601", check.Error, device, model.Ref{}, fmt.Sprintf("パック %s にはコンフィグを書くテンプレートがありません", r.p.ID))}
	}

	w := r.of(ds)
	data := copyRows(r.sc, ds, func(t *schema.Table) []*model.Row { return w.rows[t.Name][device] })
	out.Data = data
	// The pack writes some settings in another way than the book has them
	// (appendix B 5.7): told as G-614, and the text is made from the
	// rewritten rows.
	notes, warns := importer.Adapt(r.p, r.sc, data, device)
	for _, n := range notes {
		ref := model.Ref{Table: n.Table}
		if t := r.sc.Table(n.Table); t != nil {
			ref = check.RefOf(t, n.Row, "")
		}
		findings = append(findings, check.New("G-614", check.Warning, device, ref, n.Text))
	}
	for _, w := range warns {
		findings = append(findings, check.New("G-601", check.Error, device, model.Ref{}, "パック "+r.p.ID+" の置き換え（adapt.yaml）: "+w))
	}
	for _, w := range importer.FillDefaults(r.p, r.sc, data, device, true) {
		findings = append(findings, check.New("G-601", check.Error, device, model.Ref{}, "パック "+r.p.ID+" の既定値: "+w))
	}
	for _, t := range r.sc.Tables {
		tab := data.Tables[t.Name]
		if tab == nil {
			continue
		}
		for _, row := range tab.Rows {
			for _, c := range t.Columns {
				cell := row.Cells[c.Name]
				if c.Explicit && cell != nil && len(cell.Origins) > 0 && cell.Origins[0].Kind == "default" {
					findings = append(findings, check.New("V-113", check.Info, device, check.RefOf(t, row, c.Name),
						fmt.Sprintf("「%s」が空なので、%s の既定値で書き出します", c.Title.JA, r.p.ID),
						model.KV{Name: "既定値", Value: check.Shown(c, cell)}))
				}
				// A value that has to be there, and the pack has none to
				// give: the templates would take the empty cell for "no".
				if c.Explicit && c.Required && cell.Text() == "" {
					out.Failed = true
					f := check.New("G-606", check.Error, device, check.RefOf(t, row, c.Name),
						fmt.Sprintf("「%s」が空で、%s に既定値もありません。値を書いてください", c.Title.JA, r.p.ID))
					f.At = check.CellAt(row, c.Name)
					findings = append(findings, f)
				}
			}
		}
	}

	r.used = map[*model.Row]map[string]bool{}
	vars := r.vars(w, data, project)

	// The sections, then the extra configurations of the device.
	parts := map[string]*part{}
	for _, s := range r.p.Sections {
		if s.Template == "" {
			continue
		}
		lines, errs := r.run(s.Template, vars, 0)
		for i := range lines {
			lines[i].Section = s.ID
		}
		parts[s.ID] = &part{s.ID, lines}
		for _, err := range errs {
			out.Failed = true
			findings = append(findings, r.failure(device, err, false, 0))
		}
	}
	slot := r.p.ExtraSlot()
	before, after := map[string][]*part{}, map[string][]*part{}
	for _, x := range r.applying(ds, device) {
		name := "extra/" + x.file
		lines, errs := r.run(name, vars, x.offset)
		for i := range lines {
			lines[i].Extra = true
			if lines[i].Template == "" {
				lines[i].Template = name
			}
		}
		for _, err := range errs {
			out.Failed = true
			findings = append(findings, r.failure(device, err, true, x.offset))
		}
		p := &part{lines: lines}
		id, where := x.id, x.where
		if id != "" && parts[id] == nil && id != slot {
			out.Failed = true
			findings = append(findings, check.New("G-608", check.Error, device, model.Ref{},
				fmt.Sprintf("追加コンフィグ %s: 位置に書いた「%s」という節はパック %s にありません", x.file, id, r.p.ID)))
			id, where = slot, "after"
		}
		switch {
		case id == "":
			after[slot] = append(after[slot], p)
		case where == "before":
			before[id] = append(before[id], p)
		default:
			after[id] = append(after[id], p)
		}
	}
	var lines []Line
	add := func(ps []*part, section string) {
		for _, p := range ps {
			for _, l := range p.lines {
				l.Section = section
				lines = append(lines, l)
			}
		}
	}
	for _, s := range r.p.Sections {
		add(before[s.ID], s.ID)
		if p := parts[s.ID]; p != nil {
			lines = append(lines, p.lines...)
		}
		add(after[s.ID], s.ID)
	}
	out.Lines = tidy(lines)
	var sb strings.Builder
	for _, l := range out.Lines {
		sb.WriteString(l.Text)
		sb.WriteByte('\n')
	}
	out.Text = sb.String()

	if !out.Failed {
		// A section that failed did not read what it would have (spec 9.6).
		findings = append(findings, r.unusedCells(out)...)
	}
	findings = append(findings, r.doubled(out)...)
	return out, findings
}

func (r *Renderer) vars(w *whole, data *model.Dataset, project map[string]any) map[string]any {
	mine := map[string]any{}
	var dev any = map[string]any{}
	for _, t := range r.sc.Tables {
		if t.DeviceColumn() == "" {
			continue
		}
		rows := []any{}
		if tab := data.Tables[t.Name]; tab != nil {
			for _, row := range tab.Rows {
				rows = append(rows, rowRec{r, t, row, true, w.heads[t.Name]})
			}
		}
		mine[t.Name] = rows
		if t.Name == "devices" && len(rows) > 0 {
			dev = rows[0]
		}
	}
	if project == nil {
		project = map[string]any{}
	}
	// Made ready once: every template of the device takes the same values.
	return map[string]any{
		"device":  tmpl.Prepare(dev),
		"t":       tmpl.Prepare(mine),
		"all":     w.all,
		"model":   tmpl.Prepare(modelVars(r.p, data)),
		"pack":    tmpl.Prepare(map[string]any{"id": r.p.ID, "version": r.p.Version, "indent": r.p.Output.Indent}),
		"project": tmpl.Prepare(project),
	}
}

// applying gives the extra configurations of a device: the one for every
// device, those of its groups by name, then its own.
func (r *Renderer) applying(ds *model.Dataset, device string) []extra {
	var groups []string
	if t := ds.Tables["devices"]; t != nil {
		for _, row := range t.Rows {
			if row.Get("hostname") == device && row.Cells["group"] != nil {
				groups = append(groups, row.Cells["group"].Values...)
			}
		}
	}
	sort.Strings(groups)
	var out []extra
	find := func(file string) {
		for _, x := range r.extras {
			if x.file == file {
				out = append(out, x)
			}
		}
	}
	find("_all.j2")
	for _, g := range groups {
		find("@" + g + ".j2")
	}
	find(device + ".j2")
	return out
}

// write is one piece of text the engine wrote, with the cells read since
// the piece before.
type write struct {
	start, end int
	reads      []CellRef
}

// run runs one template and gives its text line by line. offset is added to
// the template lines (the lines taken off the top of an extra configuration).
// When the template fails, the errors are its first one and what else stops
// it after that (spec 9.6); the text ends before the first.
func (r *Renderer) run(name string, vars map[string]any, offset int) ([]Line, []error) {
	var writes []write
	r.reads = nil
	text, pieces, err := r.eng.Render(name, vars, func(start, end int) {
		writes = append(writes, write{start, end, r.reads})
		r.reads = nil
	})
	var errs []error
	if err != nil {
		// A line that was not written to its end is dropped (spec 9.6).
		text = text[:strings.LastIndexByte(text, '\n')+1]
		errs = append(errs, err)
		if te, ok := err.(*tmpl.Error); ok && te.Stopped {
			rest, more := r.eng.Rest(name, vars, maxFailures-1)
			for _, e := range rest {
				errs = append(errs, e)
			}
			if more {
				errs = append(errs, &tmpl.Error{Name: name, Msg: fmt.Sprintf("ほかにも誤りがあります（1 つのテンプレートで調べるのは %d 件までです。直してから、もう一度生成してください）", maxFailures)})
			}
		}
	}
	if text == "" {
		return nil, errs
	}
	if !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	// Where each line starts.
	starts := []int{0}
	for i := 0; i < len(text)-1; i++ {
		if text[i] == '\n' {
			starts = append(starts, i+1)
		}
	}
	lineOf := func(off int) int { return sort.SearchInts(starts, off+1) - 1 }
	lines := make([]Line, len(starts))
	for i, s := range starts {
		end := len(text) - 1
		if i+1 < len(starts) {
			end = starts[i+1] - 1
		}
		lines[i].Text = text[s:end]
	}
	pieceAt := func(off int) *tmpl.Piece {
		i := sort.Search(len(pieces), func(i int) bool { return pieces[i].End > off })
		if i < len(pieces) && pieces[i].Start <= off {
			return &pieces[i]
		}
		return nil
	}
	for _, p := range pieces {
		// The first piece of a line names the place in the template.
		off := p.Start
		for off < p.End && off < len(text) && text[off] == '\n' {
			off++
		}
		if off >= p.End || off >= len(text) {
			continue
		}
		if l := &lines[lineOf(off)]; l.Template == "" {
			l.Template, l.TLine = p.Name, p.Line+offset
		}
	}
	var pending []CellRef
	for _, w := range writes {
		reads := append(pending, w.reads...)
		pending = nil
		if len(reads) == 0 {
			continue
		}
		p := pieceAt(w.start)
		if p == nil {
			continue // a failed run has no pieces: the lines stay without origins
		}
		off := w.start
		if !p.Expr {
			for off < w.end && off < len(text) && text[off] == '\n' {
				off++
			}
			if off >= w.end {
				pending = reads // only line ends: the cells decide what comes next
				continue
			}
		}
		if off >= len(text) {
			continue
		}
		l := &lines[lineOf(off)]
		if p.Expr {
			l.Values = addRefs(l.Values, reads)
		} else {
			l.Conds = addRefs(l.Conds, reads)
		}
	}
	return lines, errs
}

func addRefs(list, more []CellRef) []CellRef {
	for _, m := range more {
		dup := false
		for _, x := range list {
			if x == m {
				dup = true
				break
			}
		}
		if !dup {
			list = append(list, m)
		}
	}
	return list
}

// maxFailures is how many failures of one template are told (spec 9.6).
const maxFailures = 20

// failure turns a template error into a finding.
func (r *Renderer) failure(device string, err error, isExtra bool, offset int) check.Finding {
	rule, msg := "G-601", err.Error()
	if te, ok := err.(*tmpl.Error); ok {
		switch te.Kind {
		case "required":
			rule = "G-606"
		case "lex":
			rule = "G-603"
		}
		msg = te.Msg
		switch {
		case te.Name != "" && te.Line > 0:
			msg = fmt.Sprintf("%s の %d 行目: %s", te.Name, te.Line+offset, te.Msg)
		case te.Name != "":
			msg = fmt.Sprintf("%s: %s", te.Name, te.Msg)
		}
	}
	if isExtra {
		rule = "G-608"
	}
	return check.New(rule, check.Error, device, model.Ref{}, msg)
}

// tidy takes the spaces off the line ends and makes one empty line of three
// or more (appendix B 5.4).
func tidy(lines []Line) []Line {
	out := make([]Line, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		lines[i].Text = strings.TrimRight(lines[i].Text, " \t\r")
		if lines[i].Text != "" {
			out = append(out, lines[i])
			continue
		}
		j := i
		for j < len(lines) && strings.TrimRight(lines[j].Text, " \t\r") == "" {
			j++
		}
		if j-i >= 3 {
			out = append(out, lines[i])
		} else {
			for k := i; k < j; k++ {
				lines[k].Text = ""
				out = append(out, lines[k])
			}
		}
		i = j - 1
	}
	return out
}

// unusedCells tells the values no template read (G-602): one finding a row,
// or one for the table if the pack read nothing of it (a table it has no
// template for yet).
func (r *Renderer) unusedCells(out *Output) []check.Finding {
	var findings []check.Finding
	for _, t := range r.sc.Tables {
		tab := out.Data.Tables[t.Name]
		if tab == nil {
			continue
		}
		var perRow []check.Finding
		read, rows := false, 0
		for _, row := range tab.Rows {
			if len(r.used[row]) > 0 {
				read = true
			}
			var titles []string
			var ev []model.KV
			for _, c := range t.Columns {
				cell := row.Cells[c.Name]
				if !c.Generate || c.Name == t.DeviceColumn() || cell == nil || len(cell.Values) == 0 || r.used[row][c.Name] {
					continue
				}
				titles = append(titles, c.Title.JA)
				ev = append(ev, model.KV{Name: c.Title.JA, Value: check.Shown(c, cell)})
				k := rowKey(row)
				if out.unused[k] == nil {
					out.unused[k] = map[string]bool{}
				}
				out.unused[k][c.Name] = true
			}
			if len(titles) > 0 {
				rows++
				perRow = append(perRow, check.New("G-602", check.Error, out.Device, check.RefOf(t, row, ""),
					fmt.Sprintf("値があるのに生成に使われませんでした（パック %s は書けません）: %s", r.p.ID, strings.Join(titles, "、")), ev...))
			}
		}
		if !read && rows > 0 {
			findings = append(findings, check.New("G-602", check.Error, out.Device, model.Ref{Table: t.Name},
				fmt.Sprintf("表「%s」の %d 行は生成に使われませんでした（パック %s はこの表をまだ書けません）", t.Title.JA, rows, r.p.ID)))
			continue
		}
		findings = append(findings, perRow...)
	}
	return findings
}

// path gives each line the lines above it it belongs to, by indentation.
func paths(lines []Line) []string {
	out := make([]string, len(lines))
	type level struct {
		indent int
		text   string
	}
	var stack []level
	for i, l := range lines {
		body := strings.TrimLeft(l.Text, " ")
		if body == "" {
			continue
		}
		indent := len(l.Text) - len(body)
		for len(stack) > 0 && stack[len(stack)-1].indent >= indent {
			stack = stack[:len(stack)-1]
		}
		var sb strings.Builder
		for _, s := range stack {
			sb.WriteString(s.text + "\n")
		}
		out[i] = sb.String()
		stack = append(stack, level{indent, body})
	}
	return out
}

// doubled tells the lines of the extra configurations that the pack wrote
// too, under the same lines (G-609).
func (r *Renderer) doubled(out *Output) []check.Finding {
	ps := paths(out.Lines)
	byPack := map[string]int{}
	for i, l := range out.Lines {
		if body := strings.TrimSpace(l.Text); !l.Extra && body != "" {
			if _, ok := byPack[ps[i]+body]; !ok {
				byPack[ps[i]+body] = i + 1
			}
		}
	}
	var findings []check.Finding
	for i, l := range out.Lines {
		body := strings.TrimSpace(l.Text)
		if !l.Extra || body == "" || isComment(r.p, body) {
			continue
		}
		if n, ok := byPack[ps[i]+body]; ok {
			f := check.New("G-609", check.Warning, out.Device, model.Ref{},
				fmt.Sprintf("追加コンフィグ（%s）の行が、パックが書いた %d 行目と同じです", strings.TrimPrefix(l.Template, "extra/"), n),
				model.KV{Name: "行", Value: out.Shown(i + 1)})
			f.Line = i + 1
			findings = append(findings, f)
		}
	}
	return findings
}

func isComment(p *pack.Pack, body string) bool {
	for _, c := range p.Options.CommentPrefixes {
		if strings.HasPrefix(body, c) {
			return true
		}
	}
	return false
}

// Shown gives a line for a report: the values that came from secret cells
// are covered.
func (o *Output) Shown(line int) string {
	return o.cover(line, func(string) string { return Masked })
}

// Sealed gives a line for a record that is kept (the snapshots): each secret
// value is replaced by its seal, so a change of the value still shows.
func (o *Output) Sealed(line int) string { return o.cover(line, check.Seal) }

func (o *Output) cover(line int, with func(secret string) string) string {
	if line < 1 || line > len(o.Lines) {
		return ""
	}
	l := o.Lines[line-1]
	text := l.Text
	for _, v := range l.Values {
		c := o.sc.Table(v.Row.Table).Column(v.Column)
		if c == nil || !c.Secret {
			continue
		}
		for _, val := range v.Row.Cells[v.Column].Values {
			if val != "" {
				text = strings.ReplaceAll(text, val, with(val))
			}
		}
	}
	return text
}

// Unused tells whether no template read a cell that holds a value (the
// cells G-602 tells about). key is the key text of the row as the
// templates saw it.
func (o *Output) Unused(table, key, column string) bool {
	return o.unused[table+"\x00"+key][column]
}

// UnusedRow tells whether a row has a cell with a value that no template
// read.
func (o *Output) UnusedRow(table, key string) bool {
	return len(o.unused[table+"\x00"+key]) > 0
}

// Masked stands for a secret value in a line that is shown.
const Masked = "＊＊＊"

// modelVars is what a template reads of the model of the device (appendix B
// 6.1): all empty where the device names no model the pack defines.
func modelVars(p *pack.Pack, data *model.Dataset) map[string]any {
	out := map[string]any{"id": "", "family": "", "ports": []any{}, "limits": map[string]any{}}
	tab := data.Tables["devices"]
	if tab == nil || len(tab.Rows) == 0 {
		return out
	}
	m := p.Model(tab.Rows[0].Get("model"))
	if m == nil {
		return out
	}
	ports := []any{}
	for _, pt := range m.Ports {
		ports = append(ports, pt.Name)
	}
	limits := map[string]any{}
	for k, v := range m.Limits {
		limits[k] = v
	}
	out["id"], out["family"], out["ports"], out["limits"] = m.ID, m.Family, ports, limits
	return out
}
