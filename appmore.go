package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/kojio145/configchief/internal/errlog"
	"github.com/kojio145/configchief/internal/ui"
	"github.com/kojio145/configchief/internal/update"
)

// guard is deferred first in every method the page calls (spec 14.8.4): an
// internal error (a panic) is written to error.log and becomes the error the
// page shows, instead of ending the window. err is the method's error
// result (nil when it has none: the page is then told by an event).
func (a *App) guard(where string, err *error) {
	r := recover()
	if r == nil {
		return
	}
	msg := errlog.Panic(where, r)
	if err != nil {
		*err = errors.New(msg)
		return
	}
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "internal-error", msg)
	}
}

// recoverGo is deferred first in a goroutine of the window: the panic is
// written down and the goroutine ends, the window lives on.
func recoverGo(where string) {
	if r := recover(); r != nil {
		errlog.Panic(where, r)
	}
}

// exeDir is the folder of the running exe ("" if the system cannot say).
func exeDir() string {
	p, err := os.Executable()
	if err != nil {
		return ""
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	return filepath.Dir(p)
}

// ---- バージョン情報 (spec 14.8.1) ----

// downloaded is the exe this run downloaded last (開く shows it selected).
var downloaded struct {
	sync.Mutex
	path string
}

// CheckLatest asks GitHub for the newest public release: the only place,
// with DownloadLatest, where the tool talks to the network, and only when
// the user presses the button.
func (a *App) CheckLatest() (r0 *update.Info, err error) {
	defer a.guard("CheckLatest", &err)
	return update.Check(a.ctx, version, exeDir())
}

// DownloadLatest saves the newest release's exe next to the running one
// (never replacing it) and gives its path.
func (a *App) DownloadLatest() (r0 string, err error) {
	defer a.guard("DownloadLatest", &err)
	p, err := update.Download(a.ctx, version, exeDir())
	if err == nil {
		downloaded.Lock()
		downloaded.path = p
		downloaded.Unlock()
	}
	return p, err
}

// ShowDownloaded shows the downloaded exe selected in Explorer (its folder
// when the file is gone).
func (a *App) ShowDownloaded() (err error) {
	defer a.guard("ShowDownloaded", &err)
	downloaded.Lock()
	p := downloaded.path
	downloaded.Unlock()
	if p != "" {
		if st, err := os.Stat(p); err == nil && !st.IsDir() {
			return selectInExplorer(p)
		}
	}
	return openPath(update.SaveDir(exeDir()))
}

// ShowAppFolder opens the folder of the running exe.
func (a *App) ShowAppFolder() (err error) {
	defer a.guard("ShowAppFolder", &err)
	d := exeDir()
	if d == "" {
		return errors.New("ConfigChief.exe のフォルダが分かりません")
	}
	return openPath(d)
}

// OpenWeb opens one of the fixed pages in the browser: "github", "manual",
// "release" or "notes:<version>". Only these addresses; the tool itself
// fetches nothing.
func (a *App) OpenWeb(which string) {
	defer a.guard("OpenWeb", nil)
	switch {
	case which == "github":
		runtime.BrowserOpenURL(a.ctx, update.RepoPage)
	case which == "manual":
		runtime.BrowserOpenURL(a.ctx, update.ManualPage)
	case which == "release":
		runtime.BrowserOpenURL(a.ctx, update.ReleasePage)
	case strings.HasPrefix(which, "notes:"):
		runtime.BrowserOpenURL(a.ctx, update.NotesPage(strings.TrimPrefix(which, "notes:")))
	}
}

// ---- the zoom, the search and the comparison side by side ----

func (a *App) SetZoom(z int) int { defer a.guard("SetZoom", nil); return a.s.SetZoom(z) }

func (a *App) Search(o ui.SearchOptions) (r0 *ui.SearchView, err error) {
	defer a.guard("Search", &err)
	return a.s.Search(o)
}
func (a *App) ReadFile(rel string) (r0 *ui.TextView, err error) {
	defer a.guard("ReadFile", &err)
	return a.s.ReadFile(rel)
}

// SearchSave writes the hits of a search as text where the user chooses ("" =
// the dialog was closed).
func (a *App) SearchSave(text string) (r0 string, err error) {
	defer a.guard("SearchSave", &err)
	p, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "検索の結果を保存", DefaultFilename: "search.txt",
		Filters: []runtime.FileFilter{{DisplayName: "テキスト (*.txt)", Pattern: "*.txt"}},
	})
	if err != nil || p == "" {
		return "", err
	}
	return p, os.WriteFile(p, []byte(strings.ReplaceAll(text, "\n", "\r\n")), 0o644)
}

// DiffSave writes the comparison of two files side by side as one HTML file
// where the user chooses ("" = the dialog was closed). Spec 14.8.9.
func (a *App) DiffSave(x, y string) (r0 string, err error) {
	defer a.guard("DiffSave", &err)
	v, err := a.s.Diff(x, y)
	if err != nil {
		return "", err
	}
	page, err := a.s.DiffHTML(v)
	if err != nil {
		return "", err
	}
	name := "diff_" + strings.TrimSuffix(filepath.Base(x), filepath.Ext(x)) + "_" + strings.TrimSuffix(filepath.Base(y), filepath.Ext(y)) + ".html"
	p, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "左右の差分を HTML に保存", DefaultFilename: name,
		Filters: []runtime.FileFilter{{DisplayName: "HTML (*.html)", Pattern: "*.html"}},
	})
	if err != nil || p == "" {
		return "", err
	}
	return p, os.WriteFile(p, page, 0o644)
}

// ---- the case written out and read back (spec 14.9) ----

// CaseExport writes the open case out to a file the user chooses ("" path =
// the dialog was closed).
func (a *App) CaseExport(noSecrets bool, password string) (r0 *ui.CaseExportView, err error) {
	defer a.guard("CaseExport", &err)
	v, data, err := a.s.CaseExport(noSecrets, password)
	if err != nil {
		return nil, err
	}
	p, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title: "案件を書き出す", DefaultFilename: v.Name,
		Filters: []runtime.FileFilter{{DisplayName: "ConfigChief の案件 (*.ccase)", Pattern: "*.ccase"}},
	})
	if err != nil || p == "" {
		return v, err
	}
	if !strings.EqualFold(filepath.Ext(p), ".ccase") {
		p += ".ccase"
	}
	if err := os.WriteFile(p, data, 0o600); err != nil {
		return nil, err
	}
	v.Path = p
	return v, nil
}

// CasePick lets the user choose a case file to read (nil = the dialog was
// closed).
func (a *App) CasePick() (r0 *ui.CaseFileView, err error) {
	defer a.guard("CasePick", &err)
	p, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title:   "読み込む案件のファイル",
		Filters: []runtime.FileFilter{{DisplayName: "ConfigChief の案件 (*.ccase)", Pattern: "*.ccase"}},
	})
	if err != nil || p == "" {
		return nil, err
	}
	return ui.CaseFile(p)
}

// CaseImport checks the password, lets the user choose the folder the case
// goes under, spreads it there and opens it (nil = the dialog was closed).
func (a *App) CaseImport(path, password string) (r0 *ui.CaseImportView, err error) {
	defer a.guard("CaseImport", &err)
	if err := ui.CaseCheck(path, password); err != nil {
		return nil, err
	}
	parent, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{Title: "案件を広げる場所（この下に案件のフォルダを作ります）", CanCreateDirectories: true})
	if err != nil || parent == "" {
		return nil, err
	}
	return a.s.CaseImport(path, password, parent)
}
