package main

import (
	"context"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/listimport"
	"github.com/kojio145/configchief/internal/ui"
)

// App is what the page can call. It holds nothing but the window: every
// call goes on to the session (internal/ui), which calls the steps
// (internal/ops).
type App struct {
	ctx    context.Context
	s      *ui.Session
	notice string // what went wrong before the page was there
}

// NewApp binds a session to the window.
func NewApp(s *ui.Session) *App { return &App{s: s} }

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
	placeMainWindow()
	a.s.Progress = func(done, total int, file string) {
		runtime.EventsEmit(ctx, "progress", map[string]any{"done": done, "total": total, "file": file})
	}
}

// beforeClose keeps where the window is for the next start (spec 14.8.5).
func (a *App) beforeClose(ctx context.Context) bool {
	saveMainPlacement()
	return false
}

// shutdown lets the open case go: its mark is emptied (spec 14.8.2).
func (a *App) shutdown(ctx context.Context) {
	defer recoverGo("shutdown")
	a.s.Close()
}

func (a *App) openAtStart(dir string) {
	if _, err := a.s.Open(dir); err != nil {
		a.notice = err.Error()
	}
}

// Start is what the page asks for first.
type Start struct {
	State  ui.State `json:"state"`
	Notice string   `json:"notice"` // "" = nothing to tell
}

// Start tells the page which case is open, and once what went wrong while
// opening the case folder given at the start.
func (a *App) Start() Start {
	defer a.guard("Start", nil)
	st := Start{State: a.s.State(), Notice: a.notice}
	a.notice = ""
	return st
}

// PickFolder lets the user choose a folder ("" = the user closed the dialog).
func (a *App) PickFolder(title string) (r0 string, err error) {
	defer a.guard("PickFolder", &err)
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: title, CanCreateDirectories: true})
}

// PickFile lets the user choose a text file, starting in the open case
// folder ("" = the user closed the dialog).
func (a *App) PickFile(title string) (r0 string, err error) {
	defer a.guard("PickFile", &err)
	opts := runtime.OpenDialogOptions{Title: title}
	if dir, err := a.s.Path("input"); err == nil {
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			opts.DefaultDirectory = dir
		}
	}
	return runtime.OpenFileDialog(a.ctx, opts)
}

// Show opens a folder, the book or a report of the case folder with the
// program the system has for it. rel is a path from the case folder.
func (a *App) Show(rel string) (err error) {
	defer a.guard("Show", &err)
	path, err := a.s.Path(rel)
	if err != nil {
		return err
	}
	return openPath(path)
}

// The rest passes the call on (see internal/ui for what each one does).

// Create makes a case named name in the place of the cases (spec 14.15.2).
func (a *App) Create(name, kind string) (r0 *ui.Home, err error) {
	defer a.guard("Create", &err)
	return a.s.NewCase(name, kind)
}

// Place tells the place of the cases and the cases in it (spec 14.15.1).
func (a *App) Place() ui.PlaceView { defer a.guard("Place", nil); return a.s.Place() }

// PickPlace lets the user choose another place of the cases (nil = the
// dialog was closed).
func (a *App) PickPlace() (r0 *ui.PlaceView, err error) {
	defer a.guard("PickPlace", &err)
	dir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "案件の置き場所（この下に、案件ごとのフォルダを作ります）", CanCreateDirectories: true})
	if err != nil || dir == "" {
		return nil, err
	}
	v, err := a.s.SetPlace(dir)
	if err != nil {
		return nil, err
	}
	return &v, nil
}

// SetDeliveryNames chooses the language of the names of the delivery set
// (spec 14.15.4).
func (a *App) SetDeliveryNames(lang string) (r0 *ui.Home, err error) {
	defer a.guard("SetDeliveryNames", &err)
	return a.s.SetDeliveryNames(lang)
}

// ShowPlace opens the place of the cases in the explorer.
func (a *App) ShowPlace() (err error) {
	defer a.guard("ShowPlace", &err)
	dir, err := a.s.MadePlace()
	if err != nil {
		return err
	}
	return openPath(dir)
}

// ResetPlace makes projects next to the exe the place again.
func (a *App) ResetPlace() (r0 ui.PlaceView, err error) {
	defer a.guard("ResetPlace", &err)
	return a.s.SetPlace("")
}
func (a *App) Open(dir string) (r0 *ui.Home, err error) {
	defer a.guard("Open", &err)
	return a.s.Open(dir)
}
func (a *App) Close() ui.State                { defer a.guard("Close", nil); return a.s.Close() }
func (a *App) Forget(dir string) ui.State     { defer a.guard("Forget", nil); return a.s.Forget(dir) }
func (a *App) Home() (r0 *ui.Home, err error) { defer a.guard("Home", &err); return a.s.Home() }
func (a *App) Import(source string) (r0 *ui.ImportView, err error) {
	defer a.guard("Import", &err)
	return a.s.Import(source)
}
func (a *App) ChooseFile(source, path string) (r0 *ui.ImportView, err error) {
	defer a.guard("ChooseFile", &err)
	return a.s.ChooseFile(source, path)
}
func (a *App) Lines(source, path string) (r0 *ui.LinesView, err error) {
	defer a.guard("Lines", &err)
	return a.s.Lines(source, path)
}
func (a *App) ChoosePack(source, path, pack string) (r0 *ui.ImportView, err error) {
	defer a.guard("ChoosePack", &err)
	return a.s.ChoosePack(source, path, pack)
}
func (a *App) Mapping() (r0 *ui.MappingView, err error) {
	defer a.guard("Mapping", &err)
	return a.s.Mapping()
}
func (a *App) MakeBook(from string, replace bool) (r0 *ui.BookView, err error) {
	defer a.guard("MakeBook", &err)
	return a.s.MakeBook(from, replace)
}
func (a *App) NewBook() (r0 []string, err error) {
	defer a.guard("NewBook", &err)
	return a.s.NewBook()
}
func (a *App) CloneDevice(from, to string) (r0 []string, err error) {
	defer a.guard("CloneDevice", &err)
	return a.s.CloneDevice(from, to)
}
func (a *App) Compare(x, y string) (r0 *ui.CompareView, err error) {
	defer a.guard("Compare", &err)
	return a.s.Compare(x, y)
}
func (a *App) Diff(x, y string) (r0 *ui.DiffView, err error) {
	defer a.guard("Diff", &err)
	return a.s.Diff(x, y)
}
func (a *App) Tree(x, y string, stamps, secrets bool) (r0 *ui.TreeView, err error) {
	defer a.guard("Tree", &err)
	return a.s.Tree(x, y, stamps, secrets)
}
func (a *App) TreeAll(x, y string, stamps, secrets bool) (r0 *ui.TreeAllView, err error) {
	defer a.guard("TreeAll", &err)
	return a.s.TreeAll(x, y, stamps, secrets)
}
func (a *App) TreeOf(i int) (r0 *ui.TreeView, err error) {
	defer a.guard("TreeOf", &err)
	return a.s.TreeOf(i)
}
func (a *App) Generate() (r0 *ui.ResultView, err error) {
	defer a.guard("Generate", &err)
	return a.s.Generate()
}
func (a *App) Check() (r0 *ui.ResultView, err error) {
	defer a.guard("Check", &err)
	return a.s.Check()
}
func (a *App) Approve(key, scope, reason, by string) (r0 *ui.ResultView, err error) {
	defer a.guard("Approve", &err)
	return a.s.Approve(key, scope, reason, by)
}
func (a *App) Unapprove(id string) (r0 *ui.ResultView, err error) {
	defer a.guard("Unapprove", &err)
	return a.s.Unapprove(id)
}
func (a *App) ApproveMigration(key, scope, reason, by string) (r0 string, err error) {
	defer a.guard("ApproveMigration", &err)
	return a.s.ApproveMigration(key, scope, reason, by)
}
func (a *App) UnapproveMigration(id string) (err error) {
	defer a.guard("UnapproveMigration", &err)
	return a.s.UnapproveMigration(id)
}
func (a *App) Unhandled(source string) (r0 *ui.JudgeView, err error) {
	defer a.guard("Unhandled", &err)
	return a.s.Unhandled(source)
}
func (a *App) CarryOut(source string) (r0 *ui.CarryView, err error) {
	defer a.guard("CarryOut", &err)
	return a.s.CarryOut(source)
}
func (a *App) Judge(source, group, devices, verdict, reason, by string) (r0 *ui.JudgeView, err error) {
	defer a.guard("Judge", &err)
	return a.s.Judge(source, group, devices, verdict, reason, by)
}
func (a *App) Cancel() {
	defer a.guard("Cancel",

		// The table over the parameter book (spec 14.6).
		nil)
	a.s.Cancel()
}

func (a *App) EditSheets() (r0 *ui.EditListView, err error) {
	defer a.guard("EditSheets", &err)
	return a.s.EditSheets()
}
func (a *App) EditSheet(id string) (r0 *book.View, err error) {
	defer a.guard("EditSheet", &err)
	return a.s.EditSheet(id)
}
func (a *App) EditDraft(p book.DraftSheet) (r0 int, err error) {
	defer a.guard("EditDraft", &err)
	return a.s.EditDraft(p)
}
func (a *App) EditDiscard(sheet string) (r0 int, err error) {
	defer a.guard("EditDiscard", &err)
	return a.s.EditDiscard(sheet)
}
func (a *App) EditCheck(sheet, column string, values []string) (r0 []string, err error) {
	defer a.guard("EditCheck", &err)
	return a.s.EditCheck(sheet, column, values)
}
func (a *App) EditSave(choice map[string]string) (r0 *ui.EditSavedView, err error) {
	defer a.guard("EditSave", &err)
	return a.s.EditSave(choice)
}
func (a *App) EditColumn(sheet, title string, headings []string) (r0 string, err error) {
	defer a.guard("EditColumn", &err)
	return a.s.EditColumn(sheet, title, headings)
}
func (a *App) EditFind(query string, opt book.FindOptions) (r0 []book.FindCount, err error) {
	defer a.guard("EditFind", &err)
	return a.s.EditFind(query, opt)
}
func (a *App) EditReplace(query, with string, opt book.FindOptions, skip string) (r0 *ui.EditReplacedView, err error) {
	defer a.guard("EditReplace", &err)
	return a.s.EditReplace(query, with, opt, skip)
}
func (a *App) EditPrefs() map[string]ui.GridPref {
	defer a.guard("EditPrefs", nil)
	return a.s.EditPrefs()
}
func (a *App) EditPref(sheet string, p ui.GridPref) {
	defer a.guard("EditPref", nil)
	a.s.EditPref(sheet, p)
}

// ClipboardGet and ClipboardSet read and write the clipboard of the system
// as text: the table copies to Excel and pastes from it.
func (a *App) ClipboardGet() (r0 string, err error) {
	defer a.guard("ClipboardGet", &err)
	return runtime.ClipboardGetText(a.ctx)
}
func (a *App) ClipboardSet(text string) (err error) {
	defer a.guard("ClipboardSet", &err)
	return runtime.ClipboardSetText(a.ctx, text)
}

// The lists of the customer (spec 14.3).

func (a *App) Customer() (r0 *ui.CustomerView, err error) {
	defer a.guard("Customer", &err)
	return a.s.Customer()
}
func (a *App) Logs(what string) (r0 *ui.LogsView, err error) {
	defer a.guard("Logs", &err)
	return a.s.Logs(what)
}
func (a *App) Sites(source string) (r0 *ui.SitesView, err error) {
	defer a.guard("Sites", &err)
	return a.s.Sites(source)
}
func (a *App) Export(what string, mask bool) (r0 *ui.ExportView, err error) {
	defer a.guard("Export", &err)
	return a.s.Export(what, mask)
}
func (a *App) Reflect(keys []string, rule string) (r0 string, err error) {
	defer a.guard("Reflect", &err)
	return a.s.Reflect(keys, rule)
}
func (a *App) Mappings() (r0 *ui.MappingsView, err error) {
	defer a.guard("Mappings", &err)
	return a.s.Mappings()
}
func (a *App) ListSample(r listimport.Rule) (r0 *ui.SampleView, err error) {
	defer a.guard("ListSample", &err)
	return a.s.ListSample(r)
}
func (a *App) ListTry(r listimport.Rule) (r0 *ui.TryView, err error) {
	defer a.guard("ListTry", &err)
	return a.s.ListTry(r)
}
func (a *App) ListSave(path string, r listimport.Rule) (r0 string, err error) {
	defer a.guard("ListSave", &err)
	return a.s.ListSave(path, r)
}
func (a *App) ListDelete(path string) (r0 string, err error) {
	defer a.guard("ListDelete", &err)
	return a.s.ListDelete(path)
}
func (a *App) ListTrace(v string, steps []listimport.Step) []string {
	defer a.guard("ListTrace", nil)
	return a.s.ListTrace(v, steps)
}

// The screen of the templates (spec 14.7).
func (a *App) Fill() (r0 *ui.FillView, err error) { defer a.guard("Fill", &err); return a.s.Fill() }
func (a *App) TemplateSave(path, name, text string) (r0 []string, err error) {
	defer a.guard("TemplateSave", &err)
	return a.s.TemplateSave(path, name, text)
}
func (a *App) TemplateDelete(path string) (r0 string, err error) {
	defer a.guard("TemplateDelete", &err)
	return a.s.TemplateDelete(path)
}
func (a *App) TemplateCheck(text string) (r0 []string, err error) {
	defer a.guard("TemplateCheck", &err)
	return a.s.TemplateCheck(text)
}
func (a *App) VarSave(path, name string, r listimport.VarRule) (r0 string, err error) {
	defer a.guard("VarSave", &err)
	return a.s.VarSave(path, name, r)
}
func (a *App) VarDelete(path string) (r0 string, err error) {
	defer a.guard("VarDelete", &err)
	return a.s.VarDelete(path)
}
func (a *App) VarTry(r listimport.VarRule) (r0 *ui.VarTryView, err error) {
	defer a.guard("VarTry", &err)
	return a.s.VarTry(r)
}
func (a *App) FillDraft(file, device string) (r0 *ui.FillDraftView, err error) {
	defer a.guard("FillDraft", &err)
	return a.s.FillDraft(file, device)
}
func (a *App) FillRun(template, diff string, devices []string) (r0 *ui.FillRunView, err error) {
	defer a.guard("FillRun", &err)
	return a.s.FillRun(template, diff, devices)
}
