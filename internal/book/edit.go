package book

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"

	"github.com/kojio145/configchief/internal/schema"
	"github.com/kojio145/configchief/internal/sheet"
)

// The table of a screen over a sheet of the book (spec 14.6): a person
// changes cells, adds rows and deletes rows there. The changes are a draft
// until they are saved; saving writes those cells and rows and nothing else.

// EditColumn is a column of a sheet as the table of a screen shows it.
type EditColumn struct {
	Key   string `json:"key"`   // the internal name; "free:…" for a column the definition does not have
	Title string `json:"title"` // the heading as the sheet has it
	// Col is the place in the sheet, from 0; -1 = the sheet does not have
	// the column yet (saving a value adds the heading at the right end).
	Col      int      `json:"col"`
	Type     string   `json:"type"` // "" for a column without a definition
	List     bool     `json:"list"`
	Required bool     `json:"required"`
	IsKey    bool     `json:"isKey"`
	Secret   bool     `json:"secret"`
	Free     bool     `json:"free"`
	Choices  []string `json:"choices"`
	Note     string   `json:"note"`
	// What the draft does to a free column: Added = the sheet gets it with
	// the save, Was = the heading it has in the sheet (Title is the new
	// one), Dropped = the save takes it out.
	Added   bool   `json:"added"`
	Was     string `json:"was"`
	Dropped bool   `json:"dropped"`

	def *schema.Column
}

// EditRow is a row of a sheet with a value.
type EditRow struct {
	Row     int      // the row of the sheet, from 1
	Cells   []string // one per column of the sheet of the screen
	Sum     string   // a mark of everything the row holds
	Formula []int    // the columns (by index) whose cell holds a formula
}

// EditSheet is a sheet of the book that can be edited.
type EditSheet struct {
	ID      string       `json:"id"`    // the internal name of the table; "@devices", "@ifs", "@common"
	Name    string       `json:"name"`  // the name of the sheet in the file
	Title   string       `json:"title"` // the name of the table
	Columns []EditColumn `json:"columns"`
	Device  int          `json:"device"` // the column that holds the device (-1 = none)
	// Missing: the book has no such sheet yet. The table shows its headings
	// alone, and saving rows makes the sheet (spec 14.6).
	Missing bool      `json:"missing"`
	Rows    []EditRow `json:"-"`

	place *SheetInfo
}

// The columns that tell the rows of the sheets that are not tables apart.
var mgmtKeys = map[string][]string{
	"@devices": {"old_host"},
	"@ifs":     {"new_host", "old_if"},
	"@common":  {"scope", "table", "column"},
}

// The Draft is what a person changed in the tables of the screen and did not
// save yet (state/draft.yaml, spec 14.6).
type Draft struct {
	Version int          `yaml:"version" json:"version"`
	Sheets  []DraftSheet `yaml:"sheets" json:"sheets"`
}

// DraftSheet is the part of the draft of one sheet.
type DraftSheet struct {
	Sheet   string     `yaml:"sheet" json:"sheet"`
	Changed []DraftRow `yaml:"changed,omitempty" json:"changed"`
	Added   []DraftAdd `yaml:"added,omitempty" json:"added"`
	Deleted []DraftDel `yaml:"deleted,omitempty" json:"deleted"`
	Columns []DraftCol `yaml:"columns,omitempty" json:"columns"`
}

// DraftCol is what happens to a column the definition does not have (a free
// column): Op "add" makes one with the heading Title, "rename" gives the one
// with the heading Title the heading To, "drop" takes it out.
type DraftCol struct {
	Op    string `yaml:"op" json:"op"`
	Title string `yaml:"title" json:"title"`
	To    string `yaml:"to,omitempty" json:"to"`
}

// DraftRow is a row of the book with cells that were changed. Key and Old
// are what the row held when it was changed: the save tells by them whether
// the book was changed elsewhere since.
type DraftRow struct {
	Row int               `yaml:"row" json:"row"`
	Key map[string]string `yaml:"key" json:"key"`
	Old map[string]string `yaml:"old" json:"old"`
	New map[string]string `yaml:"new" json:"new"`
}

// DraftAdd is a row to add. Before and BeforeKey name the row of the book
// it goes above (its number and its key when the row was put in); without
// them the row goes to the end of the sheet.
//
// With From and FromKey it is no new row but a row of the book that goes to
// that place (a row that was moved): Cells is empty then, and what was
// changed in the row is among the changed rows, by the number it had.
type DraftAdd struct {
	Cells     map[string]string `yaml:"cells,omitempty" json:"cells"`
	Before    int               `yaml:"before,omitempty" json:"before"`
	BeforeKey map[string]string `yaml:"before_key,omitempty" json:"beforeKey"`
	From      int               `yaml:"from,omitempty" json:"from"`
	FromKey   map[string]string `yaml:"from_key,omitempty" json:"fromKey"`
}

// DraftDel is a row of the book to delete.
type DraftDel struct {
	Row int               `yaml:"row" json:"row"`
	Key map[string]string `yaml:"key" json:"key"`
	Sum string            `yaml:"sum" json:"sum"`
}

// Count is the number of changes: cells changed, rows added, rows deleted.
func (d *Draft) Count() int {
	if d == nil {
		return 0
	}
	n := 0
	for i := range d.Sheets {
		n += d.Sheets[i].Count()
	}
	return n
}

// Count is the number of changes of one sheet.
func (s *DraftSheet) Count() int {
	n := len(s.Added) + len(s.Deleted) + len(s.Columns)
	for _, r := range s.Changed {
		n += len(r.New)
	}
	return n
}

// Of gives the part of a sheet, or nil.
func (d *Draft) Of(id string) *DraftSheet {
	if d == nil {
		return nil
	}
	for i := range d.Sheets {
		if d.Sheets[i].Sheet == id {
			return &d.Sheets[i]
		}
	}
	return nil
}

// Put replaces the part of a sheet; a part without a change is taken out.
// The parts stay in the order of their names.
func (d *Draft) Put(part DraftSheet) {
	part = part.clean()
	out := d.Sheets[:0:0]
	for _, s := range d.Sheets {
		if s.Sheet != part.Sheet {
			out = append(out, s)
		}
	}
	if part.Count() > 0 {
		out = append(out, part)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Sheet < out[j].Sheet })
	d.Sheets = out
	d.Version = 1
}

// clean drops what changes nothing: a cell whose new value is the old one, a
// row to add without a value.
func (s DraftSheet) clean() DraftSheet {
	out := DraftSheet{Sheet: s.Sheet, Deleted: s.Deleted}
	for _, r := range s.Changed {
		n := DraftRow{Row: r.Row, Key: r.Key, Old: map[string]string{}, New: map[string]string{}}
		for k, v := range r.New {
			if r.Old[k] != v {
				n.Old[k], n.New[k] = r.Old[k], v
			}
		}
		if len(n.New) > 0 && n.Row > 0 {
			out.Changed = append(out.Changed, n)
		}
	}
	for _, a := range s.Added {
		cells := map[string]string{}
		for k, v := range a.Cells {
			if strings.TrimSpace(v) != "" {
				cells[k] = v
			}
		}
		n := DraftAdd{Cells: cells}
		if a.From > 0 {
			n = DraftAdd{From: a.From, FromKey: a.FromKey}
		} else if len(cells) == 0 {
			continue
		}
		if a.Before > 0 {
			n.Before, n.BeforeKey = a.Before, a.BeforeKey
		}
		out.Added = append(out.Added, n)
	}
	for _, c := range s.Columns {
		c.Title, c.To = strings.TrimSpace(c.Title), strings.TrimSpace(c.To)
		switch {
		case c.Title == "":
		case c.Op == "add" || c.Op == "drop":
			out.Columns = append(out.Columns, DraftCol{Op: c.Op, Title: c.Title})
		case c.Op == "rename" && c.To != "" && c.To != c.Title:
			out.Columns = append(out.Columns, c)
		}
	}
	return out
}

// ReadSheets gives the sheets of a book a person can edit in a table: the
// sheets of the tables, and the sheets of the mapping and of the common
// values. A sheet the book does not have yet is given with its headings
// alone (Missing). The cells are as the book has them (not put into the one
// way of writing of their type).
func ReadSheets(data []byte, file string, sc *schema.Set, lang string) ([]*EditSheet, error) {
	src, err := sheet.Read(data)
	if err != nil {
		return nil, fmt.Errorf("%s を読めません: %v", file, err)
	}
	// The places of the sheets and of their columns are all that is asked of
	// the book: making its rows into design data takes most of the time of
	// reading a large book.
	bk := fromSheets(src, file, sc, true)
	byName := map[string]*sheet.Sheet{}
	for i := range src.Sheets {
		if _, dup := byName[src.Sheets[i].Name]; !dup {
			byName[src.Sheets[i].Name] = &src.Sheets[i]
		}
	}
	formulas, err := formulaCells(data, bk)
	if err != nil {
		return nil, err
	}

	var out []*EditSheet
	add := func(es *EditSheet, keys []string) {
		var s *sheet.Sheet
		if !es.Missing {
			if s = byName[es.place.Name]; s == nil {
				return
			}
		}
		isKey := map[string]bool{}
		for _, k := range keys {
			isKey[k] = true
		}
		// The columns the sheet has, in its order; then the ones it lacks.
		sort.SliceStable(es.Columns, func(i, j int) bool {
			a, b := es.Columns[i].Col, es.Columns[j].Col
			return a >= 0 && (b < 0 || a < b)
		})
		for i := range es.Columns {
			c := &es.Columns[i]
			c.IsKey = isKey[c.Key]
			if c.Col >= 0 && s != nil && len(s.Rows) > 0 {
				c.Title = strings.TrimSpace(at(s.Rows[0], c.Col))
			}
			if c.Choices == nil {
				c.Choices = []string{}
			}
		}
		fs := formulas[es.place.Name]
		for r := 1; s != nil && r < len(s.Rows); r++ {
			line := s.Rows[r]
			if blank(line) {
				continue
			}
			row := EditRow{Row: r + 1, Cells: make([]string, len(es.Columns)), Sum: rowSum(line)}
			for i, c := range es.Columns {
				if c.Col < 0 {
					continue
				}
				row.Cells[i] = at(line, c.Col)
				if fs[addr(c.Col, r)] {
					row.Formula = append(row.Formula, i)
				}
			}
			es.Rows = append(es.Rows, row)
		}
		out = append(out, es)
	}

	for _, t := range sc.Tables {
		info := bk.Sheets[t.Name]
		missing := info == nil
		if missing {
			// The sheet as the save will make it: the columns of the definition.
			info = &SheetInfo{Name: t.Title.In(lang), Cols: map[string]int{}, Width: len(t.Columns), Last: 1}
			for i, c := range t.Columns {
				info.Cols[c.Name] = i
			}
		}
		es := &EditSheet{ID: t.Name, Name: info.Name, Title: t.Title.In(lang), Device: -1, Missing: missing, place: info}
		for _, c := range t.Columns {
			col := EditColumn{Key: c.Name, Title: heading(c, lang), Col: -1, Type: c.Type, List: c.List, Required: c.Required,
				Secret: c.Secret, Choices: choices(c, lang), Note: note(t, c, lang), def: c}
			if ci, ok := info.Cols[c.Name]; ok {
				col.Col = ci
			}
			es.Columns = append(es.Columns, col)
		}
		es.Columns = append(es.Columns, freeColumns(info)...)
		add(es, t.Key)
		for i, c := range es.Columns {
			if c.Key == t.DeviceColumn() {
				es.Device = i
			}
		}
	}
	for _, m := range mgmtSheets {
		info := bk.Sheets[m.id]
		missing := info == nil
		if missing {
			info = &SheetInfo{Name: m.title.In(lang), Cols: map[string]int{}, Width: len(m.cols), Last: 1}
			for i, c := range m.cols {
				info.Cols[c.name] = i
			}
		}
		es := &EditSheet{ID: m.id, Name: info.Name, Title: m.title.In(lang), Device: -1, Missing: missing, place: info}
		for _, c := range m.cols {
			col := EditColumn{Key: c.name, Title: c.title.In(lang), Col: -1, Type: "string", Note: c.note.In(lang)}
			if ci, ok := info.Cols[c.name]; ok {
				col.Col = ci
			}
			es.Columns = append(es.Columns, col)
		}
		es.Columns = append(es.Columns, freeColumns(info)...)
		add(es, mgmtKeys[m.id])
		for i, c := range es.Columns {
			if c.Key == "new_host" {
				es.Device = i
			}
		}
	}
	return out, nil
}

// freeColumns gives the columns of a sheet the definition does not have, as
// the sheet has them.
func freeColumns(info *SheetInfo) []EditColumn {
	var out []EditColumn
	for _, h := range info.Free {
		out = append(out, EditColumn{Key: freeCol(h), Title: h, Col: info.Cols[freeCol(h)], Free: true})
	}
	return out
}

// rowSum is a mark of everything a row of a sheet holds.
func rowSum(line []string) string {
	n := len(line)
	for n > 0 && line[n-1] == "" {
		n--
	}
	h := sha256.New()
	for _, v := range line[:n] {
		io.WriteString(h, v)
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// formulaCells gives the cells that hold a formula, by sheet: "C5" → true.
// Few books have one, so the parts of the file are looked through first.
func formulaCells(data []byte, bk *Book) (map[string]map[string]bool, error) {
	out := map[string]map[string]bool{}
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return out, nil // not an xlsx file: no formulas the tool could tell
	}
	any := false
	for _, part := range zr.File {
		if !strings.HasPrefix(part.Name, "xl/worksheets/") {
			continue
		}
		rc, err := part.Open()
		if err != nil {
			continue
		}
		text, err := io.ReadAll(rc)
		rc.Close()
		if err == nil && (bytes.Contains(text, []byte("<f>")) || bytes.Contains(text, []byte("<f "))) {
			any = true
			break
		}
	}
	if !any {
		return out, nil
	}
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return out, nil
	}
	defer f.Close()
	for _, info := range bk.Sheets {
		rows, err := f.Rows(info.Name)
		if err != nil {
			continue
		}
		cells := map[string]bool{}
		for r := 0; rows.Next(); r++ {
			cols, err := rows.Columns(excelize.Options{RawCellValue: true})
			if err != nil {
				break
			}
			for c := range cols {
				a := addr(c, r)
				if text, err := f.GetCellFormula(info.Name, a); err == nil && text != "" {
					cells[a] = true
				}
			}
		}
		rows.Close()
		out[info.Name] = cells
	}
	return out, nil
}

// index gives the place of a column in Columns, or -1.
func (es *EditSheet) index(key string) int {
	for i := range es.Columns {
		if es.Columns[i].Key == key {
			return i
		}
	}
	return -1
}

// keyOf gives the cells of a row that tell it apart from the others.
func (es *EditSheet) keyOf(cells []string) map[string]string {
	out := map[string]string{}
	for i, c := range es.Columns {
		if c.IsKey {
			out[c.Key] = strings.TrimSpace(cells[i])
		}
	}
	return out
}

func sameKey(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	for k, v := range a {
		if w, ok := b[k]; !ok || strings.TrimSpace(v) != strings.TrimSpace(w) {
			return false
		}
	}
	return true
}

// locate finds the row of the book a part of the draft is about: the row of
// its number if the key is the same, else the one row with that key (rows
// were added or deleted above it). -1 = there is no such row.
func (es *EditSheet) locate(row int, key map[string]string) int {
	want := map[string]string{}
	for i := range es.Columns {
		if es.Columns[i].IsKey {
			want[es.Columns[i].Key] = key[es.Columns[i].Key]
		}
	}
	found, n := -1, 0
	for i := range es.Rows {
		if !sameKey(es.keyOf(es.Rows[i].Cells), want) {
			continue
		}
		if es.Rows[i].Row == row {
			return i
		}
		found = i
		n++
	}
	empty := true
	for _, v := range want {
		if strings.TrimSpace(v) != "" {
			empty = false
		}
	}
	if n == 1 && !empty {
		return found
	}
	return -1
}

// Check tells what is wrong with a value for a column ("" = nothing): the
// check of the type that reading the book does (spec 6.2).
func (c *EditColumn) Check(value string) string {
	if c.def == nil || strings.TrimSpace(value) == "" {
		return ""
	}
	for _, item := range split(c.def, value) {
		if _, err := c.def.Normalize(item); err != nil {
			return err.Error()
		}
	}
	return ""
}

// CheckValue is Check for a column by its names. A sheet or a column without
// a definition takes any value.
func CheckValue(sc *schema.Set, sheetID, column, value string) string {
	c := sc.Table(sheetID).Column(column)
	if c == nil {
		return ""
	}
	return (&EditColumn{def: c}).Check(value)
}

// Conflict is a change of the draft that meets a change made to the book
// elsewhere since (spec 14.6). A person chooses which one stays.
type Conflict struct {
	ID string `json:"id"`
	// Kind: "cell" (the cell holds another value now), "gone" (the row is
	// not there any more), "deleted" (the row to delete was changed),
	// "sheet" (the book has no such sheet any more: only "theirs" can be
	// chosen).
	Kind   string `json:"kind"`
	Sheet  string `json:"sheet"`  // the name of the sheet
	Row    int    `json:"row"`    // the row of the sheet now (0 = none)
	Label  string `json:"label"`  // the key of the row
	Column string `json:"column"` // the heading
	Old    string `json:"old"`    // the value when the cell was changed
	Mine   string `json:"mine"`   // the value of the draft
	Theirs string `json:"theirs"` // the value of the book now
	Secret bool   `json:"secret"`
}

// ViewRow is a row of the table of a screen: a row of the book with the
// draft put on.
type ViewRow struct {
	Row     int               `json:"row"`     // the row of the sheet now (0 = a row to add)
	Sum     string            `json:"sum"`     // the mark of the row (of the draft, for a row to delete)
	Base    []string          `json:"base"`    // the cells before the changes of the draft
	Cells   []string          `json:"cells"`   // the cells with the draft put on
	Formula []int             `json:"formula"` // the columns whose cell holds a formula
	Theirs  map[string]string `json:"theirs"`  // column index → the value of the book now, where it is not Base
	Gone    bool              `json:"gone"`    // the book has no such row any more
	Deleted bool              `json:"deleted"` // the draft deletes the row
	Moved   bool              `json:"moved"`   // the draft moves the row: it stands where it goes
	Clash   bool              `json:"clash"`   // a row to delete that was changed in the book since
	Bad     map[string]string `json:"bad"`     // column index → what is wrong with the value
}

// View is a sheet as the table of a screen shows it.
type View struct {
	*EditSheet
	Rows  []ViewRow `json:"rows"`
	Draft int       `json:"draft"` // the changes of the draft of this sheet
}

// View puts the part of the draft of a sheet on the sheet.
func (es *EditSheet) View(part *DraftSheet) *View {
	es = es.withColumns(part)
	v := &View{EditSheet: es}
	blank := func() ViewRow {
		return ViewRow{Base: make([]string, len(es.Columns)), Cells: make([]string, len(es.Columns)), Formula: []int{},
			Theirs: map[string]string{}, Bad: map[string]string{}}
	}
	rows := make([]ViewRow, len(es.Rows))
	for i, r := range es.Rows {
		formula := r.Formula
		if formula == nil {
			formula = []int{}
		}
		rows[i] = ViewRow{Row: r.Row, Sum: r.Sum, Base: r.Cells, Cells: r.Cells, Formula: formula,
			Theirs: map[string]string{}, Bad: map[string]string{}}
	}
	// A row that goes to a place: a new one, or the row of the book of an
	// index (one that is moved).
	type placed struct {
		row  ViewRow
		from int // index of es.Rows; -1 = a new row
	}
	var gone []ViewRow
	var tail []placed
	above := map[int][]placed{} // index of es.Rows → the rows to put in above it
	moved := map[int]bool{}
	if part != nil {
		v.Draft = part.Count()
		for _, d := range part.Changed {
			i := es.locate(d.Row, d.Key)
			var row *ViewRow
			if i < 0 {
				// What is known of the row: its key and the cells that were changed.
				n := blank()
				n.Row, n.Gone = d.Row, true
				for k, val := range d.Key {
					if ci := es.index(k); ci >= 0 {
						n.Base[ci] = val
					}
				}
				copy(n.Cells, n.Base)
				gone = append(gone, n)
				row = &gone[len(gone)-1]
			} else {
				row = &rows[i]
				row.Base = append([]string{}, row.Base...)
				row.Cells = append([]string{}, row.Cells...)
			}
			for k, val := range d.New {
				ci := es.index(k)
				if ci < 0 {
					continue
				}
				now := row.Base[ci]
				row.Base[ci] = d.Old[k]
				row.Cells[ci] = val
				if !row.Gone && now != d.Old[k] && now != val {
					row.Theirs[strconv.Itoa(ci)] = now
				}
			}
		}
		for _, d := range part.Deleted {
			i := es.locate(d.Row, d.Key)
			if i < 0 {
				continue // deleted in the book as well
			}
			row := &rows[i]
			row.Deleted = true
			row.Clash = row.Sum != d.Sum
			row.Sum = d.Sum
		}
		for _, a := range part.Added {
			it := placed{row: blank(), from: -1}
			if a.From > 0 {
				// A row that is gone, deleted or moved already is not moved.
				if it.from = es.locate(a.From, a.FromKey); it.from < 0 || moved[it.from] || rows[it.from].Deleted {
					continue
				}
				moved[it.from] = true
			}
			for k, val := range a.Cells {
				if ci := es.index(k); ci >= 0 {
					it.row.Cells[ci] = val
				}
			}
			// A row whose place is not there any more goes to the end, as
			// the save puts it.
			if i := es.anchor(a); i >= 0 {
				above[i] = append(above[i], it)
			} else {
				tail = append(tail, it)
			}
		}
	}
	v.Rows = make([]ViewRow, 0, len(rows)+len(gone)+len(tail))
	put := func(list []placed) {
		for _, it := range list {
			if it.from >= 0 {
				it.row = rows[it.from]
				it.row.Moved = true
			}
			v.Rows = append(v.Rows, it.row)
		}
	}
	for i := range rows {
		put(above[i])
		if !moved[i] {
			v.Rows = append(v.Rows, rows[i])
		}
	}
	v.Rows = append(v.Rows, gone...)
	put(tail)
	for i := range v.Rows {
		for ci := range es.Columns {
			if msg := es.Columns[ci].Check(v.Rows[i].Cells[ci]); msg != "" {
				v.Rows[i].Bad[strconv.Itoa(ci)] = msg
			}
		}
	}
	return v
}

// anchor gives the index of the row of the book a row to add goes above, or
// -1 for a row that goes to the end.
func (es *EditSheet) anchor(a DraftAdd) int {
	if a.Before <= 0 {
		return -1
	}
	return es.locate(a.Before, a.BeforeKey)
}

// Part gives the draft the view shows: the opposite of View. A screen that
// loads a sheet and keeps its draft does the same; this one is for a change
// made to the sheets no screen shows (a replacement in every sheet).
func (v *View) Part() DraftSheet {
	p := DraftSheet{Sheet: v.ID}
	dropped := map[int]bool{}
	for ci, c := range v.Columns {
		switch {
		case c.Dropped && c.Added:
			dropped[ci] = true
		case c.Dropped:
			dropped[ci] = true
			title := c.Title
			if c.Was != "" {
				title = c.Was
			}
			p.Columns = append(p.Columns, DraftCol{Op: "drop", Title: title})
		case c.Added:
			p.Columns = append(p.Columns, DraftCol{Op: "add", Title: c.Title})
		case c.Was != "":
			p.Columns = append(p.Columns, DraftCol{Op: "rename", Title: c.Was, To: c.Title})
		}
	}
	first := len(p.Added)
	for _, r := range v.Rows {
		switch {
		case r.Row == 0:
			cells := map[string]string{}
			for ci, val := range r.Cells {
				if val != "" && !dropped[ci] {
					cells[v.Columns[ci].Key] = val
				}
			}
			p.Added = append(p.Added, DraftAdd{Cells: cells})
			continue
		case r.Gone:
		case r.Moved && !r.Deleted:
			// It goes where the rows to add around it go.
			p.Added = append(p.Added, DraftAdd{From: r.Row, FromKey: v.keyOf(r.Base)})
		default:
			// The rows to add since the last row of the book go above this one.
			for i := first; i < len(p.Added); i++ {
				p.Added[i].Before, p.Added[i].BeforeKey = r.Row, v.keyOf(r.Base)
			}
			first = len(p.Added)
		}
		if r.Deleted {
			p.Deleted = append(p.Deleted, DraftDel{Row: r.Row, Key: v.keyOf(r.Base), Sum: r.Sum})
			continue
		}
		d := DraftRow{Row: r.Row, Key: v.keyOf(r.Base), Old: map[string]string{}, New: map[string]string{}}
		for ci, val := range r.Cells {
			if val != r.Base[ci] && !dropped[ci] {
				d.Old[v.Columns[ci].Key], d.New[v.Columns[ci].Key] = r.Base[ci], val
			}
		}
		if len(d.New) > 0 {
			p.Changed = append(p.Changed, d)
		}
	}
	return p.clean()
}

// EditSummary tells what Apply wrote.
type EditSummary struct {
	Cells   int `json:"cells"`   // cells that were changed
	Added   int `json:"added"`   // rows that were added
	Deleted int `json:"deleted"` // rows that were taken out
	Cleared int `json:"cleared"` // rows that were emptied instead (a sheet with comments on its cells)
	Skipped int `json:"skipped"` // cells that were left alone: they hold a formula
	Sheets  int `json:"sheets"`  // sheets that were made
	Moved   int `json:"moved"`   // rows that went to the end of the sheet instead of the place they were put in
	Columns int `json:"columns"` // free columns that were added, renamed or taken out
	Shifted int `json:"shifted"` // rows of the book that were moved to another place
	Stayed  int `json:"stayed"`  // rows that were to move and stay where they are (a sheet with comments on its cells)
}

// Changed tells whether Apply wrote anything.
func (s *EditSummary) Changed() bool {
	return s.Cells+s.Added+s.Deleted+s.Cleared+s.Sheets+s.Columns+s.Shifted > 0
}

// label names a row by its key.
func label(key map[string]string, es *EditSheet) string {
	var parts []string
	for _, c := range es.Columns {
		if c.IsKey && strings.TrimSpace(key[c.Key]) != "" {
			if c.Secret {
				parts = append(parts, "***")
			} else {
				parts = append(parts, strings.TrimSpace(key[c.Key]))
			}
		}
	}
	return strings.Join(parts, " / ")
}

// plan is what Apply writes into one sheet.
type plan struct {
	es   *EditSheet
	sets []set
	adds []planAdd
	dels []int // rows of the sheet, from 1
	cols []DraftCol
	had  map[string]bool // the columns the sheet has before anything is written
}

type set struct {
	row   int // row of the sheet, from 1
	col   string
	value string
}

type planAdd struct {
	cells  map[string]string
	before int  // the row of the sheet it goes above, from 1; 0 = the end of the sheet
	placed bool // it was put in at a place (which may be gone by now)
	from   int  // the row of the sheet that moves there, from 1; 0 = a new row
}

// Apply gives the bytes of the book with the draft written in: the cells
// that were changed, the rows that were added and the rows that were deleted,
// the free columns that were added, renamed or taken out, and nothing else
// (spec 14.6). A sheet the book does not have yet is made when the draft has
// rows for it.
//
// A change that meets a change made to the book since is a conflict. choice
// tells for the ID of a conflict whether the draft ("mine") or the book
// ("theirs") stays. If a conflict has no choice, nothing is written: the
// conflicts are given, with nil bytes.
func Apply(old []byte, file string, sc *schema.Set, lang string, d *Draft, choice map[string]string) ([]byte, *EditSummary, []Conflict, error) {
	sheets, err := ReadSheets(old, file, sc, lang)
	if err != nil {
		return nil, nil, nil, err
	}
	byID := map[string]*EditSheet{}
	for _, es := range sheets {
		byID[es.ID] = es
	}
	var conflicts []Conflict
	open := false
	// mine tells whether the draft wins a conflict; it notes the conflict.
	mine := func(c Conflict) bool {
		conflicts = append(conflicts, c)
		switch choice[c.ID] {
		case "mine":
			return c.Kind != "sheet"
		case "theirs":
			return false
		}
		open = true
		return false
	}

	var plans []*plan
	for si := range d.Sheets {
		part := &d.Sheets[si]
		es := byID[part.Sheet]
		// Rows that were changed or deleted in a sheet that is not there any
		// more have no place; rows to add make the sheet anew.
		if es == nil || (es.Missing && len(part.Changed)+len(part.Deleted) > 0) {
			mine(Conflict{ID: "sheet|" + part.Sheet, Kind: "sheet", Sheet: part.Sheet})
			continue
		}
		es = es.withColumns(part)
		p := &plan{es: es, cols: part.Columns}
		plans = append(plans, p)
		for _, ch := range part.Changed {
			id := fmt.Sprintf("%s|%d", part.Sheet, ch.Row)
			keys := make([]string, 0, len(ch.New))
			for k := range ch.New {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			i := es.locate(ch.Row, ch.Key)
			if i < 0 {
				// The same change made in the book already leaves nothing to do.
				if j := rowAt(es, ch.Row); j >= 0 && holds(es, j, ch.New) {
					continue
				}
				c := Conflict{ID: "gone|" + id, Kind: "gone", Sheet: es.Name, Label: label(ch.Key, es)}
				var said []string
				for _, k := range keys {
					if ci := es.index(k); ci >= 0 {
						v := ch.New[k]
						if es.Columns[ci].Secret {
							v = "***"
						}
						said = append(said, es.Columns[ci].Title+" = "+v)
					}
				}
				c.Mine = strings.Join(said, "、")
				if mine(c) {
					cells := map[string]string{}
					for k, v := range ch.Key {
						cells[k] = v
					}
					for k, v := range ch.New {
						cells[k] = v
					}
					p.adds = append(p.adds, planAdd{cells: cells})
				}
				continue
			}
			row := es.Rows[i]
			for _, k := range keys {
				ci := es.index(k)
				if ci < 0 {
					continue
				}
				now, want := row.Cells[ci], ch.New[k]
				if now == want {
					continue
				}
				if now != ch.Old[k] {
					c := Conflict{ID: "cell|" + id + "|" + k, Kind: "cell", Sheet: es.Name, Row: row.Row, Label: label(es.keyOf(row.Cells), es),
						Column: es.Columns[ci].Title, Old: ch.Old[k], Mine: want, Theirs: now, Secret: es.Columns[ci].Secret}
					if !mine(c) {
						continue
					}
				}
				p.sets = append(p.sets, set{row.Row, k, want})
			}
		}
		for _, del := range part.Deleted {
			i := es.locate(del.Row, del.Key)
			if i < 0 {
				continue
			}
			row := es.Rows[i]
			if row.Sum != del.Sum {
				c := Conflict{ID: fmt.Sprintf("deleted|%s|%d", part.Sheet, del.Row), Kind: "deleted", Sheet: es.Name, Row: row.Row, Label: label(es.keyOf(row.Cells), es)}
				if !mine(c) {
					continue
				}
			}
			p.dels = append(p.dels, row.Row)
		}
		gone := map[int]bool{}
		for _, r := range p.dels {
			gone[r] = true
		}
		for _, a := range part.Added {
			n := planAdd{cells: a.Cells, placed: a.Before > 0}
			if a.From > 0 {
				// A row that is not there any more, or that is deleted or
				// moved by the same draft, is not moved.
				i := es.locate(a.From, a.FromKey)
				if i < 0 || gone[es.Rows[i].Row] {
					continue
				}
				n.cells, n.from = nil, es.Rows[i].Row
				gone[n.from] = true
			}
			if i := es.anchor(a); i >= 0 {
				n.before = es.Rows[i].Row
			}
			p.adds = append(p.adds, n)
		}
	}
	if open {
		return nil, nil, conflicts, nil
	}

	sum := &EditSummary{}
	f, err := excelize.OpenReader(bytes.NewReader(old))
	if err != nil {
		return nil, nil, nil, fmt.Errorf("%s を開けません（表の編集を書けるのは xlsx のブックだけです）: %v", file, err)
	}
	defer f.Close()
	st, err := newStyles(f)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, p := range plans {
		if err := p.write(f, st, sc, lang, sum); err != nil {
			return nil, nil, nil, err
		}
	}
	if !sum.Changed() {
		return old, sum, conflicts, nil
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, nil, nil, err
	}
	return buf.Bytes(), sum, conflicts, nil
}

// write puts the plan of one sheet into the file: the cells, the rows for
// the end of the sheet, the rows to put in and to take out (from the bottom,
// so that the numbers of the rows above stay), then the free columns.
func (p *plan) write(f *excelize.File, st styles, sc *schema.Set, lang string, sum *EditSummary) error {
	es, info := p.es, p.es.place
	p.had = map[string]bool{}
	for k := range info.Cols {
		p.had[k] = true
	}
	dropped := map[string]bool{}
	for _, c := range p.cols {
		if c.Op == "drop" {
			dropped[freeCol(c.Title)] = true
		}
	}
	var adds []planAdd
	for _, a := range p.adds {
		if a.from > 0 {
			adds = append(adds, a)
			continue
		}
		cells := map[string]string{}
		for k, v := range a.cells {
			if strings.TrimSpace(v) != "" && !dropped[k] && es.index(k) >= 0 {
				cells[k] = v
			}
		}
		if len(cells) > 0 {
			a.cells = cells
			adds = append(adds, a)
		}
	}
	if es.Missing {
		// Nothing to write leaves the book without the sheet.
		made := len(adds) > 0
		for _, c := range p.cols {
			made = made || c.Op == "add"
		}
		if !made {
			return nil
		}
		g, ok := frame(sc, es.ID, lang)
		if !ok {
			return nil
		}
		if _, err := f.NewSheet(g.Name); err != nil {
			return fmt.Errorf("シート「%s」を作れませんでした（%v）", g.Name, err)
		}
		if err := putGrid(f, g, st); err != nil {
			return err
		}
		sum.Sheets++
	}
	place := func(key string) (int, error) {
		ci := es.index(key)
		if ci < 0 || dropped[key] {
			return -1, nil // a column the sheet lost: the value has no place
		}
		title := es.Columns[ci].Title
		if es.Columns[ci].Was != "" {
			title = es.Columns[ci].Was // the new heading is written last
		}
		return column(f, info, st, key, title)
	}
	// line writes the cells of a new row into a row of the sheet (from 0),
	// in the order of the columns: a heading that has to be added lands at
	// the same place every time.
	line := func(at int, cells map[string]string) error {
		keys := make([]string, 0, len(cells))
		for k := range cells {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool { return es.index(keys[i]) < es.index(keys[j]) })
		for _, k := range keys {
			ci, err := place(k)
			if err != nil {
				return err
			}
			if ci < 0 {
				continue
			}
			if err := f.SetCellStr(info.Name, addr(ci, at), cells[k]); err != nil {
				return err
			}
		}
		sum.Added++
		return nil
	}

	dels := map[int]bool{}
	for _, r := range p.dels {
		dels[r] = true
	}
	for _, s := range p.sets {
		if dels[s.row] {
			continue
		}
		ci, err := place(s.col)
		if err != nil {
			return err
		}
		if ci < 0 {
			continue
		}
		cell := addr(ci, s.row-1)
		if text, _ := f.GetCellFormula(info.Name, cell); text != "" {
			sum.Skipped++
			continue
		}
		if err := f.SetCellStr(info.Name, cell, s.value); err != nil {
			return err
		}
		sum.Cells++
	}

	// Putting a row in or taking one out moves the rows below it, but not
	// the comments on their cells: a sheet with such comments keeps its
	// rows. A row to delete is emptied (an empty row is not read, spec
	// 6.1), a row to put in goes to the end.
	keep := false
	if !es.Missing {
		if comments, err := f.GetComments(info.Name); err == nil {
			for _, c := range comments {
				if _, r, err := excelize.CellNameToCoordinates(c.Cell); err == nil && r > 1 {
					keep = true
				}
			}
		}
	}
	// Every row that is put in moves the rows below it: now tells where a
	// row of the sheet as it was read stands by now.
	var puts []int // the rows (from 1) a row was put in at, in the order it was done
	now := func(row int) int {
		for _, at := range puts {
			if row >= at {
				row++
			}
		}
		return row
	}
	lists := wholeColumnLists(f, info.Name)
	var out []int // the rows that leave their place: deleted, or moved away
	for i := 0; i < len(adds); i++ {
		a := adds[i]
		if a.from > 0 && keep {
			sum.Stayed++
			continue
		}
		at := info.Last + 1 // the end of the sheet
		if a.before > 0 && !keep {
			at = now(a.before)
		} else if a.placed {
			sum.Moved++
		}
		if a.from > 0 {
			// The copy takes the cells as they are, their formats too.
			if err := f.DuplicateRowTo(info.Name, now(a.from), at); err != nil {
				return err
			}
			if at <= info.Last {
				puts = append(puts, at)
			}
			info.Last++
			out = append(out, a.from)
			sum.Shifted++
			continue
		}
		// The new rows that follow for the same place go in with it at once.
		n := 1
		for i+n < len(adds) && adds[i+n].from == 0 && adds[i+n].before == a.before && (a.before > 0 && !keep) {
			n++
		}
		inside := at <= info.Last
		if inside {
			if err := f.InsertRows(info.Name, at, n); err != nil {
				return err
			}
		}
		for k := 0; k < n; k++ {
			if err := line(at-1+k, adds[i+k].cells); err != nil {
				return err
			}
			if inside {
				puts = append(puts, at)
			}
		}
		info.Last += n
		i += n - 1
	}
	if keep {
		cols, err := f.GetCols(info.Name)
		if err != nil {
			return err
		}
		for r := range dels {
			for c := range cols {
				if err := emptyCell(f, info.Name, addr(c, r-1)); err != nil {
					return err
				}
			}
			sum.Cleared++
		}
		return p.columns(f, st, sum)
	}
	sum.Deleted += len(dels)
	for r := range dels {
		out = append(out, r)
	}
	for i := range out {
		out[i] = now(out[i])
	}
	sort.Sort(sort.Reverse(sort.IntSlice(out)))
	for _, r := range out {
		if err := f.RemoveRow(info.Name, r); err != nil {
			return err
		}
	}
	if len(puts)+len(out) > 0 {
		if err := lists.restore(f, info.Name); err != nil {
			return err
		}
	}
	return p.columns(f, st, sum)
}

// columnLists are the lists to choose from that cover a column from its
// second row to the last row of a sheet (the ones the tool gives a sheet):
// the place of each among the lists of the sheet, and what it covers.
type columnLists map[int]string

func wholeColumnLists(f *excelize.File, sheet string) columnLists {
	out := columnLists{}
	dvs, err := f.GetDataValidations(sheet)
	if err != nil {
		return out
	}
	for i, dv := range dvs {
		if a, b, ok := strings.Cut(dv.Sqref, ":"); ok && strings.HasSuffix(a, "2") && strings.HasSuffix(b, "1048576") &&
			strings.TrimSuffix(a, "2") == strings.TrimSuffix(b, "1048576") && strings.Trim(strings.TrimSuffix(a, "2"), "ABCDEFGHIJKLMNOPQRSTUVWXYZ") == "" {
			out[i] = dv.Sqref
		}
	}
	return out
}

// restore gives the lists the rows they covered before rows were put in or
// taken out: a row put in at the second row would stand above the list, and
// every row taken out makes it end one row earlier.
func (l columnLists) restore(f *excelize.File, sheet string) error {
	dvs, err := f.GetDataValidations(sheet)
	if err != nil || len(l) == 0 {
		return nil
	}
	changed := false
	for i, dv := range dvs {
		if was, ok := l[i]; ok && dv.Sqref != was {
			dv.Sqref = was
			changed = true
		}
	}
	if !changed {
		return nil
	}
	// All of them are taken off and put on again: taking one off by what it
	// covers goes through every cell of it, a million for a whole column.
	if err := f.DeleteDataValidation(sheet); err != nil {
		return err
	}
	for _, dv := range dvs {
		if err := f.AddDataValidation(sheet, dv); err != nil {
			return err
		}
	}
	return nil
}

func emptyCell(f *excelize.File, sheet, cell string) error {
	if err := f.SetCellFormula(sheet, cell, ""); err != nil {
		return err
	}
	return f.SetCellValue(sheet, cell, nil)
}

// rowAt gives the index of the row of a sheet by its number, or -1.
func rowAt(es *EditSheet, row int) int {
	for i := range es.Rows {
		if es.Rows[i].Row == row {
			return i
		}
	}
	return -1
}

// holds tells whether a row has all the values already.
func holds(es *EditSheet, i int, values map[string]string) bool {
	for k, v := range values {
		ci := es.index(k)
		if ci < 0 || es.Rows[i].Cells[ci] != v {
			return false
		}
	}
	return true
}
