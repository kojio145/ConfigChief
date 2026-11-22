package ui

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/ops"
	"github.com/kojio145/configchief/internal/project"
)

// The table over the parameter book (spec 14.6). The page holds the cells
// of the sheet it shows, the secret ones too (it shows them masked until the
// user asks for them): a table that can be edited needs its values.

// EditEntryView is a sheet that can be edited.
type EditEntryView struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Title string `json:"title"`
	Rows  int    `json:"rows"`
	Draft int    `json:"draft"`
	// Missing: the book has no such sheet yet (saving rows makes it).
	Missing bool `json:"missing"`
}

// EditListView is the sheets of the book that can be edited.
type EditListView struct {
	Book    string          `json:"book"`
	HasBook bool            `json:"hasBook"`
	Sheets  []EditEntryView `json:"sheets"`
	Draft   int             `json:"draft"` // the changes that are not saved, in all
	Lost    []string        `json:"lost"`  // sheets of the draft the book does not have
}

// EditSheets lists the sheets of the book that can be edited.
func (s *Session) EditSheets() (*EditListView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	l, err := e.EditSheets()
	if err != nil {
		return nil, err
	}
	v := &EditListView{Book: e.Proj.Config.Book, HasBook: l.HasBook, Sheets: []EditEntryView{}, Draft: l.Draft, Lost: strs(l.Lost)}
	for _, x := range l.Sheets {
		v.Sheets = append(v.Sheets, EditEntryView{x.ID, x.Name, x.Title, x.Rows, x.Draft, x.Missing})
	}
	return v, nil
}

// EditSheet gives a sheet as a table, with the draft put on. id is the
// internal name of the sheet or its name in the file.
func (s *Session) EditSheet(id string) (*book.View, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	return e.EditSheet(id)
}

// EditDraft keeps the changes of one sheet as the draft of that sheet and
// gives the number of changes of the whole draft. The book is not written.
func (s *Session) EditDraft(part book.DraftSheet) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.readOnly() {
		return 0, &project.ReadOnlyError{Newer: s.newer, Mine: s.version}
	}
	if s.env == nil {
		return 0, errors.New("案件を開いていません")
	}
	return s.env.SaveDraft(part)
}

// EditDiscard throws the draft of one sheet away ("" = of every sheet) and
// gives the number of changes that are left.
func (s *Session) EditDiscard(sheet string) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.readOnly() {
		return 0, &project.ReadOnlyError{Newer: s.newer, Mine: s.version}
	}
	if s.env == nil {
		return 0, errors.New("案件を開いていません")
	}
	return s.env.DiscardDraft(sheet)
}

// EditCheck tells what is wrong with each value for a column of a sheet (""
// = nothing).
func (s *Session) EditCheck(sheet, column string, values []string) ([]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.env == nil {
		return nil, errors.New("案件を開いていません")
	}
	return s.env.CheckValues(sheet, column, values), nil
}

// EditSavedView is what saving the draft did.
type EditSavedView struct {
	// Conflicts that a person has to choose a side for: nothing was written.
	Conflicts []book.Conflict `json:"conflicts"`
	Saved     bool            `json:"saved"`
	Message   string          `json:"message"`
	Draft     int             `json:"draft"` // the changes that are left in the draft
	// Redo are the new host names whose new model was changed: their
	// proposals of IF 対応 can be made again (spec 14.11.4).
	Redo []string `json:"redo"`
}

// EditSave writes the draft into the parameter book. choice tells for the
// ID of each conflict whether the draft ("mine") or the book ("theirs")
// stays.
func (s *Session) EditSave(choice map[string]string) (*EditSavedView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	o, err := e.SaveEdits(choice, s.now())
	if err != nil {
		return nil, err
	}
	v := &EditSavedView{Conflicts: []book.Conflict{}, Draft: e.DraftCount(), Redo: []string{}}
	if o.Summary == nil {
		for _, c := range o.Conflicts {
			if c.Secret { // a screen masks the secrets
				c.Old, c.Mine, c.Theirs = masked(c.Old), masked(c.Mine), masked(c.Theirs)
			}
			v.Conflicts = append(v.Conflicts, c)
		}
		if len(v.Conflicts) == 0 {
			v.Message = "保存する変更はありません"
		}
		return v, nil
	}
	v.Saved = true
	sum := o.Summary
	if !sum.Changed() {
		v.Message = "書く変更はありませんでした（パラメータブックは、もう同じ値になっています）。下書きは消しました"
		return v, nil
	}
	v.Message = fmt.Sprintf("パラメータブック %s に書きました（変更 %d セル・追加 %d 行・削除 %d 行）。書く前の控え: %s",
		e.Proj.Config.Book, sum.Cells, sum.Added, sum.Deleted+sum.Cleared, o.Backup)
	if sum.Sheets > 0 {
		v.Message += fmt.Sprintf("\nブックに無かったシート %d 枚を作りました", sum.Sheets)
	}
	if sum.Columns > 0 {
		v.Message += fmt.Sprintf("\n自由列 %d 列を書きました（足す・名前の変更・削除）", sum.Columns)
	}
	if sum.Shifted > 0 {
		v.Message += fmt.Sprintf("\n%d 行を動かしました", sum.Shifted)
	}
	if sum.Stayed > 0 {
		v.Message += fmt.Sprintf("\n動かす行のうち %d 行は、元の場所のままです（セルにコメントのあるシートでは、行を動かしません）", sum.Stayed)
	}
	if sum.Moved > 0 {
		v.Message += fmt.Sprintf("\n途中に入れた行のうち %d 行は、シートの末尾に足しました（入れる場所の行が無くなったか、セルにコメントのあるシートのため）", sum.Moved)
	}
	if sum.Cleared > 0 {
		v.Message += fmt.Sprintf("\n削除のうち %d 行は、行を詰めずに空にしました（セルにコメントのあるシートのため）", sum.Cleared)
	}
	if sum.Skipped > 0 {
		v.Message += fmt.Sprintf("\n式の入ったセル %d 個は書き換えていません", sum.Skipped)
	}
	// 機器対応 was changed: the proposals of IF 対応 follow without a second
	// import (spec 14.10.2).
	if o.Devices {
		if old, _ := e.Proj.Inputs("old"); len(old) > 0 {
			if p, err := e.ProposeIfs(s.now()); err != nil {
				v.Message += "\nIF 対応の案は作れませんでした: " + err.Error()
			} else if p.Added+p.Filled > 0 {
				v.Message += "\n" + p.Said()
			}
		}
	}
	if len(o.Models) > 0 {
		v.Redo = o.Models
		v.Message += fmt.Sprintf("\n新機種を変えた機器（%s）の IF 対応には、前の案が残っています。「対応」の画面の「案を作り直す」で作り直せます", strings.Join(o.Models, "、"))
	}
	return v, nil
}

// RedoIfsView is the proposals of IF 対応 made again (spec 14.11.4).
type RedoIfsView struct {
	Message string       `json:"message"`
	Rows    []ops.IfRedo `json:"rows"`
	Written bool         `json:"written"`
}

// RedoIfs makes the proposals of IF 対応 again for the new host names: with
// write false it tells what would change and writes nothing.
func (s *Session) RedoIfs(hosts []string, write bool) (*RedoIfsView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var e *ops.Env
	var err error
	if write {
		e, err = s.beginWrite()
	} else {
		e, err = s.begin()
	}
	if err != nil {
		return nil, err
	}
	r, err := e.RedoIfs(hosts, write, s.now())
	if err != nil {
		return nil, err
	}
	return &RedoIfsView{Message: r.Said(), Rows: r.Rows, Written: r.Backup != ""}, nil
}

// ProposeIfs makes the proposals of IF 対応 again from input/old (spec
// 14.10.2).
func (s *Session) ProposeIfs() (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.beginWrite()
	if err != nil {
		return "", err
	}
	p, err := e.ProposeIfs(s.now())
	if err != nil {
		return "", err
	}
	return p.Said(), nil
}

func masked(v string) string {
	if v == "" {
		return ""
	}
	return "***"
}

// EditColumn tells the name of a free column with the heading for a sheet,
// or why the sheet cannot have such a column.
func (s *Session) EditColumn(sheet, title string, headings []string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.env == nil {
		return "", errors.New("案件を開いていません")
	}
	return s.env.FreeColumn(sheet, title, headings)
}

// EditFind counts the cells that hold the text in every sheet of the book,
// with the draft put on.
func (s *Session) EditFind(query string, opt book.FindOptions) ([]book.FindCount, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	return e.FindAll(query, opt)
}

// EditReplacedView is what a replacement in the sheets the screen does not
// show did.
type EditReplacedView struct {
	Counts []book.FindCount  `json:"counts"`
	Before []book.DraftSheet `json:"before"` // the parts of the draft as they were: putting them back undoes it
	After  []book.DraftSheet `json:"after"`
	Draft  int               `json:"draft"`
}

// EditReplace puts another text in place of the text in every sheet but the
// one named skip, as changes of the draft.
func (s *Session) EditReplace(query, with string, opt book.FindOptions, skip string) (*EditReplacedView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	o, err := e.ReplaceAll(query, with, opt, skip)
	if err != nil {
		return nil, err
	}
	v := &EditReplacedView{Counts: o.Counts, Before: o.Before, After: o.After, Draft: o.Draft}
	if v.Before == nil {
		v.Before = []book.DraftSheet{}
	}
	if v.After == nil {
		v.After = []book.DraftSheet{}
	}
	return v, nil
}

// GridPref is how a person likes to see the table of one sheet: the columns
// that are hidden and the widths that were changed, by the name of the
// column. It is kept with the settings of the user, not in the case.
type GridPref struct {
	Hidden []string       `json:"hidden"`
	Widths map[string]int `json:"widths"`
}

// EditPrefs gives the way to see the table of each sheet (by its internal
// name).
func (s *Session) EditPrefs() map[string]GridPref {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := map[string]GridPref{}
	for k, p := range s.grid {
		out[k] = p.clean()
	}
	return out
}

// EditPref keeps the way to see the table of one sheet.
func (s *Session) EditPref(sheet string, p GridPref) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if sheet == "" {
		return
	}
	p = p.clean()
	if s.grid == nil {
		s.grid = map[string]GridPref{}
	}
	if len(p.Hidden) == 0 && len(p.Widths) == 0 {
		delete(s.grid, sheet)
	} else {
		s.grid[sheet] = p
	}
	s.save()
}

// clean gives the pref without what says nothing, its list in one order.
func (p GridPref) clean() GridPref {
	out := GridPref{Hidden: []string{}, Widths: map[string]int{}}
	seen := map[string]bool{}
	for _, k := range p.Hidden {
		if k != "" && !seen[k] {
			seen[k] = true
			out.Hidden = append(out.Hidden, k)
		}
	}
	sort.Strings(out.Hidden)
	for k, w := range p.Widths {
		if k != "" && w >= 30 && w <= 1000 {
			out.Widths[k] = w
		}
	}
	return out
}
