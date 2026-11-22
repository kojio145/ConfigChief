//go:build windows

package main

import (
	"fmt"
	"os/exec"

	"golang.org/x/sys/windows"
)

// openPath opens a file or a folder with the program the system has for it
// (Explorer for a folder, Excel for a book).
func openPath(path string) error {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	verb, _ := windows.UTF16PtrFromString("open")
	if err := windows.ShellExecute(0, verb, p, nil, nil, windows.SW_SHOWNORMAL); err != nil {
		return fmt.Errorf("%s を開けません: %v", path, err)
	}
	return nil
}

// selectInExplorer shows a file selected in its folder.
func selectInExplorer(path string) error {
	cmd := exec.Command("explorer.exe", "/select,"+path)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%s を開けません: %v", path, err)
	}
	go cmd.Wait()
	return nil
}
