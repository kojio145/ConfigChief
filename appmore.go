package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/kojio145/configchief/internal/errlog"
	"github.com/kojio145/configchief/internal/ops"
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
	sha  string // its SHA-256 as it was saved (spec 14.11.1)
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
		sum, _ := update.FileSHA256(p)
		downloaded.Lock()
		downloaded.path, downloaded.sha = p, sum
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
	case which == "approval":
		runtime.BrowserOpenURL(a.ctx, update.ApprovalPage)
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

// The things that save a person work (spec 14.10).

// AddFiles copies files into input/<source> (spec 14.10.1).
func (a *App) AddFiles(source string, paths []string, overwrite bool) (r0 *ui.AddView, err error) {
	defer a.guard("AddFiles", &err)
	return a.s.AddFiles(source, paths, overwrite)
}

// PickFiles lets the user choose files to add (nil = the dialog was closed).
func (a *App) PickFiles(title string) (r0 []string, err error) {
	defer a.guard("PickFiles", &err)
	return runtime.OpenMultipleFilesDialog(a.ctx, runtime.OpenDialogOptions{Title: title})
}

// ProposeIfs makes the proposals of IF 対応 again (spec 14.10.2).
func (a *App) ProposeIfs() (r0 string, err error) {
	defer a.guard("ProposeIfs", &err)
	return a.s.ProposeIfs()
}

// RedoIfs makes the proposals of IF 対応 again for the new host names
// (spec 14.11.4); write false only tells what would change.
func (a *App) RedoIfs(hosts []string, write bool) (r0 *ui.RedoIfsView, err error) {
	defer a.guard("RedoIfs", &err)
	return a.s.RedoIfs(hosts, write)
}

// CheckSettings lists the checks with the settings of the case (spec
// 14.11.5).
func (a *App) CheckSettings() (r0 *ops.CheckSettings, err error) {
	defer a.guard("CheckSettings", &err)
	return a.s.CheckSettings()
}

// SaveCheckSettings writes the settings of the checks into project.yaml.
func (a *App) SaveCheckSettings(set map[string]string) (r0 string, err error) {
	defer a.guard("SaveCheckSettings", &err)
	return a.s.SaveCheckSettings(set)
}

// MakeSample writes a sample template of the delivery sheets into
// templates/ (spec 14.11.2).
func (a *App) MakeSample(kind string) (r0 string, err error) {
	defer a.guard("MakeSample", &err)
	return a.s.MakeSample(kind)
}

// ExtraPlaces lists the extra configurations (spec 14.11.3).
func (a *App) ExtraPlaces() (r0 *ops.ExtraPlaces, err error) {
	defer a.guard("ExtraPlaces", &err)
	return a.s.ExtraPlaces()
}

// ExtraName gives the file name of an extra configuration for a target.
func (a *App) ExtraName(kind, target string) (r0 string, err error) {
	defer a.guard("ExtraName", &err)
	return a.s.ExtraName(kind, target)
}

// ReadExtra gives the text of an extra configuration.
func (a *App) ReadExtra(name string) (r0 string, err error) {
	defer a.guard("ReadExtra", &err)
	return a.s.ReadExtra(name)
}

// SaveExtra writes an extra configuration; from names the unhandled lines
// it was drafted from (nil = none).
func (a *App) SaveExtra(name, text string, from *ui.ExtraFrom) (r0 *ui.ExtraSavedView, err error) {
	defer a.guard("SaveExtra", &err)
	return a.s.SaveExtra(name, text, from)
}

// DeleteExtra moves an extra configuration to state/backup.
func (a *App) DeleteExtra(name string) (r0 string, err error) {
	defer a.guard("DeleteExtra", &err)
	return a.s.DeleteExtra(name)
}

// DraftExtra gives the extra configuration of a device with the unhandled
// lines of a group added at its end.
func (a *App) DraftExtra(source, group, device string) (r0 *ops.ExtraDraft, err error) {
	defer a.guard("DraftExtra", &err)
	return a.s.DraftExtra(source, group, device)
}

// Next tells the one thing to do next (spec 14.10.4).
func (a *App) Next() *ops.Next { defer a.guard("Next", nil); return a.s.Next() }

// LastResult gives the result of the newest output (spec 14.10.6).
func (a *App) LastResult() (r0 *ui.ResultView, err error) {
	defer a.guard("LastResult", &err)
	return a.s.LastResult()
}

// ApproveMany approves the findings picked, one approval each (spec 14.10.8).
func (a *App) ApproveMany(keys []string, reason, by string) (r0 *ui.ResultView, err error) {
	defer a.guard("ApproveMany", &err)
	return a.s.ApproveMany(keys, reason, by)
}

// BookWritable tells whether Excel has let the book go (spec 14.10.7).
func (a *App) BookWritable() bool { defer a.guard("BookWritable", nil); return a.s.BookWritable() }
