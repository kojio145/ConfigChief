//go:build windows

package main

import (
	"os"

	"golang.org/x/sys/windows"
)

// terminal gives where a command writes. The program of the window has no
// console of its own (it is built for the window), so the one it was started
// from is taken: cmd.exe, PowerShell or a batch file. An output that was
// redirected to a file or a pipe is used as it is.
func terminal() (out, errOut *os.File) {
	out, errOut = os.Stdout, os.Stderr
	usable := func(std uint32) bool {
		h, err := windows.GetStdHandle(std)
		if err != nil || h == 0 || h == windows.InvalidHandle {
			return false
		}
		t, err := windows.GetFileType(h)
		return err == nil && t != windows.FILE_TYPE_UNKNOWN
	}
	okOut, okErr := usable(windows.STD_OUTPUT_HANDLE), usable(windows.STD_ERROR_HANDLE)
	if okOut && okErr {
		return out, errOut
	}
	attach := windows.NewLazySystemDLL("kernel32.dll").NewProc("AttachConsole")
	const parent = ^uintptr(0) // ATTACH_PARENT_PROCESS
	if r, _, _ := attach.Call(parent); r == 0 {
		return out, errOut // no terminal to write to: the exit code is the answer
	}
	// Opened for reading too: only then does the os package see that it is a
	// console and write it as text (the Japanese words would come out as
	// broken bytes on a console of another code page).
	con, err := os.OpenFile("CONOUT$", os.O_RDWR, 0)
	if err != nil {
		return out, errOut
	}
	// The prompt of the terminal is already on the line: start below it.
	con.WriteString("\n")
	if !okOut {
		out = con
	}
	if !okErr {
		errOut = con
	}
	return out, errOut
}
