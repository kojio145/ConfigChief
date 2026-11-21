package ops

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"sync"
	"time"

	"github.com/kojio145/configchief/internal/book"
)

// The table of a screen over the parameter book (spec 14.6): the sheets as
// tables, the draft of the changes (state/draft.yaml), and the save that
// writes the draft into the book.

// EditEntry is a sheet of the book that can be edited.
type EditEntry struct {
	ID, Name, Title string
	Rows            int  // rows with a value
	Draft           int  // changes of the draft
	Missing         bool // the book has no such sheet yet: saving rows makes it
}

// EditList is the sheets of the book that can be edited.
type EditList struct {
	HasBook bool
	Sheets  []EditEntry
	Draft   int // the changes of the draft in all
	// Lost: sheets the draft has changes for that have no place: the tool
	// does not know the sheet, or rows of the book were changed or deleted
	// and the book does not have the sheet any more.
	Lost []string
}

func (e *Env) draft() (*book.Draft, error) {
	d := &book.Draft{}
	if err := e.Proj.ReadDraft(d); err != nil {
		return nil, err
	}
	return d, nil
}

// DraftCount is the number of changes that are not saved (0 if the draft
// cannot be read: the screen of the table tells why).
func (e *Env) DraftCount() int {
	d, err := e.draft()
	if err != nil {
		return 0
	}
	return d.Count()
}

// EditCache keeps the sheets of a book as the table of a screen reads them,
// for as long as the file is the same one (its time and its size): reading a
// large book takes seconds, and a screen asks for it with every sheet it
// opens. A window gives the same cache to every step (Env.Edit); the steps
// of the command line have none. What it keeps is never changed.
type EditCache struct {
	mu     sync.Mutex
	path   string
	lang   string
	mod    time.Time
	size   int64
	data   []byte
	sheets []*book.EditSheet
}

func (c *EditCache) get(path, lang string, st os.FileInfo) ([]byte, []*book.EditSheet) {
	if c == nil {
		return nil, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.sheets == nil || c.path != path || c.lang != lang || !c.mod.Equal(st.ModTime()) || c.size != st.Size() {
		return nil, nil
	}
	return c.data, c.sheets
}

func (c *EditCache) put(path, lang string, st os.FileInfo, data []byte, sheets []*book.EditSheet) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.path, c.lang, c.mod, c.size, c.data, c.sheets = path, lang, st.ModTime(), st.Size(), data, sheets
}

// Drop forgets what the cache keeps (the tool wrote the book).
func (c *EditCache) Drop() {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data, c.sheets = nil, nil
}

// bookBytes reads the file of the book as it is (nil = there is none).
func (e *Env) bookBytes() ([]byte, os.FileInfo, error) {
	path := e.Proj.BookPath()
	st, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, nil
	}
	var data []byte
	if err == nil {
		data, err = os.ReadFile(path)
	}
	if err != nil {
		return nil, nil, fmt.Errorf("%s を読めません: %v", e.Proj.Config.Book, err)
	}
	return data, st, nil
}

// editSheets gives the bytes of the book and its sheets as tables (nil =
// there is no book). The sheets may be the ones of the cache: they are not
// to be changed.
func (e *Env) editSheets() ([]byte, []*book.EditSheet, error) {
	path, lang := e.Proj.BookPath(), e.lang()
	var old []byte
	var sheets []*book.EditSheet
	if st, err := os.Stat(path); err == nil {
		old, sheets = e.Edit.get(path, lang, st)
	}
	if sheets == nil {
		data, st, err := e.bookBytes()
		if err != nil || data == nil {
			return nil, nil, err
		}
		if sheets, err = book.ReadSheets(data, e.Proj.Config.Book, e.Schema, lang); err != nil {
			return nil, nil, err
		}
		old = data
		e.Edit.put(path, lang, st, old, sheets)
	}
	out := make([]*book.EditSheet, len(sheets))
	copy(out, sheets)
	for i, es := range out {
		if es.ID != "@devices" {
			continue
		}
		// The new vendor is the ID of a pack that can be used.
		cp := *es
		cp.Columns = append([]book.EditColumn{}, es.Columns...)
		for k := range cp.Columns {
			if cp.Columns[k].Key == "new_vendor" {
				cp.Columns[k].Choices = append([]string{}, e.PackIDs...)
			}
		}
		out[i] = &cp
	}
	return old, out, nil
}

// EditSheets lists the sheets of the book that can be edited. It writes
// nothing.
func (e *Env) EditSheets() (*EditList, error) {
	d, err := e.draft()
	if err != nil {
		return nil, err
	}
	out := &EditList{Draft: d.Count()}
	old, sheets, err := e.editSheets()
	if err != nil || old == nil {
		return out, err
	}
	out.HasBook = true
	have := map[string]bool{}
	for _, es := range sheets {
		have[es.ID] = true
		entry := EditEntry{ID: es.ID, Name: es.Name, Title: es.Title, Rows: len(es.Rows), Missing: es.Missing}
		if part := d.Of(es.ID); part != nil {
			entry.Draft = part.Count()
			if es.Missing && len(part.Changed)+len(part.Deleted) > 0 {
				have[es.ID] = false
			}
		}
		out.Sheets = append(out.Sheets, entry)
	}
	for _, part := range d.Sheets {
		if !have[part.Sheet] {
			out.Lost = append(out.Lost, part.Sheet)
		}
	}
	return out, nil
}

// EditSheet gives one sheet as a table, with the draft put on. id is the
// internal name of the sheet or its name in the file. It writes nothing.
func (e *Env) EditSheet(id string) (*book.View, error) {
	old, sheets, err := e.editSheets()
	if err != nil {
		return nil, err
	}
	if old == nil {
		return nil, fmt.Errorf("パラメータブック %s がありません", e.Proj.Config.Book)
	}
	d, err := e.draft()
	if err != nil {
		return nil, err
	}
	for _, es := range sheets {
		if es.ID == id || es.Name == id {
			return es.View(d.Of(es.ID)), nil
		}
	}
	return nil, fmt.Errorf("パラメータブック %s に、表で開けるシート「%s」がありません", e.Proj.Config.Book, id)
}

// SaveDraft keeps the changes of one sheet in the draft, in place of the
// ones it had for the sheet, and gives the number of changes of the whole
// draft. The book is not written.
func (e *Env) SaveDraft(part book.DraftSheet) (int, error) {
	if part.Sheet == "" {
		return 0, fmt.Errorf("下書きのシートの名前がありません")
	}
	d, err := e.draft()
	if err != nil {
		return 0, err
	}
	d.Put(part)
	return d.Count(), e.writeDraft(d)
}

func (e *Env) writeDraft(d *book.Draft) error {
	if d.Count() == 0 {
		return e.Proj.RemoveDraft()
	}
	return e.Proj.WriteDraft(d)
}

// DiscardDraft throws the changes of one sheet away ("" = of every sheet)
// and gives the number of changes that are left.
func (e *Env) DiscardDraft(sheet string) (int, error) {
	if sheet == "" {
		return 0, e.Proj.RemoveDraft()
	}
	d, err := e.draft()
	if err != nil {
		return 0, err
	}
	d.Put(book.DraftSheet{Sheet: sheet})
	return d.Count(), e.writeDraft(d)
}

// EditSaved tells what SaveEdits did.
type EditSaved struct {
	// Conflicts that have no choice yet: nothing was written, the draft is
	// as it was.
	Conflicts []book.Conflict
	Summary   *book.EditSummary // nil = nothing was written
	Backup    string
}

// SaveEdits writes the draft into the parameter book: the cells that were
// changed, the rows that were added and the rows that were deleted (spec
// 14.6). choice tells for each conflict which side stays; with a conflict
// that has no choice nothing is written. After the book is written the draft
// is gone.
func (e *Env) SaveEdits(choice map[string]string, now time.Time) (*EditSaved, error) {
	d, err := e.draft()
	if err != nil {
		return nil, err
	}
	out := &EditSaved{}
	if d.Count() == 0 {
		return out, nil
	}
	old, _, err := e.bookBytes()
	if err != nil {
		return nil, err
	}
	file := e.Proj.Config.Book
	if old == nil {
		return nil, fmt.Errorf("パラメータブック %s がありません（下書きは残っています）", file)
	}
	// Before the copy is taken: a book that cannot be written leaves nothing behind.
	if err := e.Proj.BookWritable(); err != nil {
		return nil, fmt.Errorf("%v（下書きは残っています）", err)
	}
	next, sum, conflicts, err := book.Apply(old, file, e.Schema, e.lang(), d, choice)
	if err != nil {
		return nil, err
	}
	if next == nil {
		out.Conflicts = conflicts
		return out, nil
	}
	out.Summary = sum
	if sum.Changed() {
		if out.Backup, err = e.saveBook(now, next); err != nil {
			return nil, fmt.Errorf("%v（下書きは残っています）", err)
		}
		e.Proj.Log(now, "edit: 表の編集をパラメータブックに書きました（セル %d・追加 %d 行・削除 %d 行・空にした行 %d 行・作ったシート %d 枚・自由列 %d 列）", sum.Cells, sum.Added, sum.Deleted, sum.Cleared, sum.Sheets, sum.Columns)
	}
	return out, e.Proj.RemoveDraft()
}

// CheckValues tells what is wrong with each value for a column of a sheet
// ("" = nothing): the check of the type that reading the book does.
func (e *Env) CheckValues(sheet, column string, values []string) []string {
	out := make([]string, len(values))
	for i, v := range values {
		out[i] = book.CheckValue(e.Schema, sheet, column, v)
	}
	return out
}

// FreeColumn tells the name of a free column with the heading for a sheet,
// or why the sheet cannot have such a column. headings are the headings the
// table of the screen has.
func (e *Env) FreeColumn(sheet, title string, headings []string) (string, error) {
	return book.FreeColumn(e.Schema, sheet, title, headings)
}

// FindAll counts the cells that hold the text in every sheet, with the draft
// put on. It writes nothing.
func (e *Env) FindAll(query string, opt book.FindOptions) ([]book.FindCount, error) {
	old, sheets, err := e.editSheets()
	if err != nil || old == nil {
		return []book.FindCount{}, err
	}
	d, err := e.draft()
	if err != nil {
		return nil, err
	}
	return book.Find(sheets, d, query, opt), nil
}

// EditReplaced tells what ReplaceAll did.
type EditReplaced struct {
	Counts []book.FindCount
	// Before and After are the parts of the draft of the sheets that were
	// changed, as they were and as they are: putting Before back undoes the
	// replacement.
	Before, After []book.DraftSheet
	Draft         int // the changes of the whole draft now
}

// ReplaceAll puts another text in place of the text in every sheet but the
// one named skip, as changes of the draft. The book is not written.
func (e *Env) ReplaceAll(query, with string, opt book.FindOptions, skip string) (*EditReplaced, error) {
	out := &EditReplaced{Counts: []book.FindCount{}}
	old, sheets, err := e.editSheets()
	if err != nil {
		return nil, err
	}
	d, err := e.draft()
	if err != nil {
		return nil, err
	}
	out.Draft = d.Count()
	if old == nil {
		return out, nil
	}
	out.After, out.Counts = book.Replace(sheets, d, query, with, opt, skip)
	if len(out.After) == 0 {
		return out, nil
	}
	for _, part := range out.After {
		before := book.DraftSheet{Sheet: part.Sheet}
		if was := d.Of(part.Sheet); was != nil {
			before = *was
		}
		out.Before = append(out.Before, before)
	}
	for _, part := range out.After {
		d.Put(part)
	}
	out.Draft = d.Count()
	return out, e.writeDraft(d)
}
