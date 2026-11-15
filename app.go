package main

import (
	"context"
	"os"

	"github.com/wailsapp/wails/v2/pkg/runtime"

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
	a.s.Progress = func(done, total int, file string) {
		runtime.EventsEmit(ctx, "progress", map[string]any{"done": done, "total": total, "file": file})
	}
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
	st := Start{State: a.s.State(), Notice: a.notice}
	a.notice = ""
	return st
}

// PickFolder lets the user choose a folder ("" = the user closed the dialog).
func (a *App) PickFolder(title string) (string, error) {
	return runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: title, CanCreateDirectories: true})
}

// PickFile lets the user choose a text file, starting in the open case
// folder ("" = the user closed the dialog).
func (a *App) PickFile(title string) (string, error) {
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
func (a *App) Show(rel string) error {
	path, err := a.s.Path(rel)
	if err != nil {
		return err
	}
	return openPath(path)
}

// The rest passes the call on (see internal/ui for what each one does).

func (a *App) Create(dir, name, kind string) (*ui.Home, error) { return a.s.Create(dir, name, kind) }
func (a *App) Open(dir string) (*ui.Home, error)               { return a.s.Open(dir) }
func (a *App) Close() ui.State                                 { return a.s.Close() }
func (a *App) Forget(dir string) ui.State                      { return a.s.Forget(dir) }
func (a *App) Home() (*ui.Home, error)                         { return a.s.Home() }
func (a *App) Import(source string) (*ui.ImportView, error)    { return a.s.Import(source) }
func (a *App) Lines(source, path string) (*ui.LinesView, error) {
	return a.s.Lines(source, path)
}
func (a *App) ChoosePack(source, path, pack string) (*ui.ImportView, error) {
	return a.s.ChoosePack(source, path, pack)
}
func (a *App) Mapping() (*ui.MappingView, error) { return a.s.Mapping() }
func (a *App) MakeBook(from string, replace bool) (*ui.BookView, error) {
	return a.s.MakeBook(from, replace)
}
func (a *App) NewBook() ([]string, error) { return a.s.NewBook() }
func (a *App) CloneDevice(from, to string) ([]string, error) {
	return a.s.CloneDevice(from, to)
}
func (a *App) Compare(x, y string) (*ui.CompareView, error) { return a.s.Compare(x, y) }
func (a *App) Diff(x, y string) (*ui.DiffView, error)       { return a.s.Diff(x, y) }
func (a *App) Tree(x, y string, stamps, secrets bool) (*ui.TreeView, error) {
	return a.s.Tree(x, y, stamps, secrets)
}
func (a *App) TreeAll(x, y string, stamps, secrets bool) (*ui.TreeAllView, error) {
	return a.s.TreeAll(x, y, stamps, secrets)
}
func (a *App) TreeOf(i int) (*ui.TreeView, error) { return a.s.TreeOf(i) }
func (a *App) Generate() (*ui.ResultView, error)  { return a.s.Generate() }
func (a *App) Check() (*ui.ResultView, error)     { return a.s.Check() }
func (a *App) Approve(key, scope, reason, by string) (*ui.ResultView, error) {
	return a.s.Approve(key, scope, reason, by)
}
func (a *App) Unapprove(id string) (*ui.ResultView, error) { return a.s.Unapprove(id) }
func (a *App) ApproveMigration(key, scope, reason, by string) (string, error) {
	return a.s.ApproveMigration(key, scope, reason, by)
}
func (a *App) UnapproveMigration(id string) error             { return a.s.UnapproveMigration(id) }
func (a *App) Unhandled(source string) (*ui.JudgeView, error) { return a.s.Unhandled(source) }
func (a *App) CarryOut(source string) (*ui.CarryView, error)  { return a.s.CarryOut(source) }
func (a *App) Judge(source, group, devices, verdict, reason, by string) (*ui.JudgeView, error) {
	return a.s.Judge(source, group, devices, verdict, reason, by)
}
func (a *App) Cancel() { a.s.Cancel() }

// The lists of the customer (spec 14.3).

func (a *App) Customer() (*ui.CustomerView, error)    { return a.s.Customer() }
func (a *App) Logs(what string) (*ui.LogsView, error) { return a.s.Logs(what) }
func (a *App) Sites(source string) (*ui.SitesView, error) {
	return a.s.Sites(source)
}
func (a *App) Export(what string, mask bool) (*ui.ExportView, error) {
	return a.s.Export(what, mask)
}
func (a *App) Reflect(keys []string, rule string) (string, error) {
	return a.s.Reflect(keys, rule)
}
func (a *App) Mappings() (*ui.MappingsView, error) { return a.s.Mappings() }
func (a *App) ListSample(r listimport.Rule) (*ui.SampleView, error) {
	return a.s.ListSample(r)
}
func (a *App) ListTry(r listimport.Rule) (*ui.TryView, error) { return a.s.ListTry(r) }
func (a *App) ListSave(path string, r listimport.Rule) (string, error) {
	return a.s.ListSave(path, r)
}
