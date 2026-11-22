package main

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"github.com/kojio145/configchief/internal/update"
)

// The downloaded exe takes the place of the running one, and the tool
// starts again (spec 14.11.1). Nothing here talks to the network.

// runningExe is the path of the running exe.
func runningExe() (string, error) {
	p, err := os.Executable()
	if err != nil {
		return "", err
	}
	if r, err := filepath.EvalSymlinks(p); err == nil {
		p = r
	}
	return p, nil
}

// CanSwap tells whether the exe downloaded in this run is in the folder of
// the running one, so that it can take its place.
func (a *App) CanSwap() bool {
	defer a.guard("CanSwap", nil)
	downloaded.Lock()
	p := downloaded.path
	downloaded.Unlock()
	run, err := runningExe()
	if p == "" || err != nil {
		return false
	}
	if _, err := os.Stat(p); err != nil {
		return false
	}
	return filepath.Dir(p) == filepath.Dir(run)
}

// SwapAndRestart puts the downloaded exe in the place of the running one,
// starts it with the open case, and closes this window. When anything goes
// wrong the files are put back and the case opened again.
func (a *App) SwapAndRestart() (err error) {
	defer a.guard("SwapAndRestart", &err)
	downloaded.Lock()
	p, sum := downloaded.path, downloaded.sha
	downloaded.Unlock()
	run, err := runningExe()
	if err != nil {
		return errors.New("動いている ConfigChief.exe の場所が分かりません")
	}
	caseDir, _ := a.s.Path("")
	a.s.Close()
	reopen := func() {
		if caseDir != "" {
			_, _ = a.s.Open(caseDir)
		}
	}
	sw, err := update.Swap(run, p, sum, version)
	if err != nil {
		reopen()
		return err
	}
	cmd := exec.Command(run)
	if caseDir != "" {
		cmd.Args = append(cmd.Args, caseDir)
	}
	cmd.Dir = filepath.Dir(run)
	if err := cmd.Start(); err != nil {
		if back := sw.Undo(); back != nil {
			reopen()
			return errors.New("新しい exe を起動できませんでした（" + err.Error() + "）。元に戻すこともできませんでした: " + back.Error())
		}
		reopen()
		return errors.New("新しい exe を起動できませんでした。元の exe のまま続けます（" + err.Error() + "）")
	}
	_ = cmd.Process.Release()
	saveMainPlacement()
	go func() {
		defer recoverGo("SwapAndRestart")
		time.Sleep(300 * time.Millisecond) // the page gets the answer first
		runtime.Quit(a.ctx)
	}()
	return nil
}
