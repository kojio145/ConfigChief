package project

import (
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// The mark is held the way PalaTerm holds data\palaterm.lock
// (datalock_windows.go, v1.10.8): opened for writing with no write sharing
// for as long as the case is open. The same file however it is reached (a
// drive letter, a UNC path, another PC): the file server refuses the second
// open. Windows drops the hold when the process ends, however it ends, so a
// crash leaves nothing stale behind.

type lockHandle = syscall.Handle

const noLock = syscall.InvalidHandle

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	pCreateFileW     = kernel32.NewProc("CreateFileW")
	pOpenProcess     = kernel32.NewProc("OpenProcess")
	pGetExitCodeProc = kernel32.NewProc("GetExitCodeProcess")
)

// A sharing violation is tried again for about a second before it counts:
// an indexer or a backup tool opens the file for an instant.
const (
	lockRetries   = 5
	lockRetryWait = 200 * time.Millisecond
)

func acquireLock(path string) (lockHandle, string, error) {
	p, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return noLock, "", err
	}
	const (
		genericRead, genericWrite = 0x80000000, 0x40000000
		fileShareRead             = 0x1
		openAlways                = 4
		attrNormal                = 0x80
	)
	var h uintptr
	for attempt := 0; ; attempt++ {
		var callErr error
		h, _, callErr = pCreateFileW.Call(uintptr(unsafe.Pointer(p)), genericRead|genericWrite, fileShareRead, 0, openAlways, attrNormal, 0)
		if syscall.Handle(h) != syscall.InvalidHandle {
			break
		}
		if errno, ok := callErr.(syscall.Errno); !ok || (errno != 32 && errno != 33) { // sharing / lock violation
			return noLock, "", callErr
		}
		if attempt < lockRetries {
			time.Sleep(lockRetryWait)
			continue
		}
		holder := readHolder(path)
		if holderGoneOn(holder, hostName(), processAlive) {
			// Whoever holds the file is not the ConfigChief that wrote the
			// line (an indexer, a sync tool): no reason to refuse.
			return noLock, "", fmt.Errorf("lock file held by another program")
		}
		return noLock, holder, ErrInUse
	}
	hd := syscall.Handle(h)
	line := fmt.Sprintf("%s|%s|%d|%s", hostName(), os.Getenv("USERNAME"), os.Getpid(), time.Now().Format("2006-01-02 15:04"))
	_ = syscall.SetEndOfFile(hd)
	var n uint32
	_ = syscall.WriteFile(hd, []byte(line), &n, nil)
	_ = syscall.FlushFileBuffers(hd)
	return hd, line, nil
}

func releaseLock(h lockHandle) {
	if h == noLock || h == 0 {
		return
	}
	if _, err := syscall.Seek(h, 0, 0); err == nil {
		_ = syscall.SetEndOfFile(h)
		_ = syscall.FlushFileBuffers(h)
	}
	_ = syscall.CloseHandle(h)
}

func readHolder(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func hostName() string {
	h, _ := os.Hostname()
	return h
}

func processAlive(pid uint32) bool {
	const queryLimited, stillActive = 0x1000, 259
	h, _, _ := pOpenProcess.Call(queryLimited, 0, uintptr(pid))
	if h == 0 {
		return false
	}
	defer syscall.CloseHandle(syscall.Handle(h))
	var code uint32
	if ok, _, _ := pGetExitCodeProc.Call(h, uintptr(unsafe.Pointer(&code))); ok == 0 {
		return true
	}
	return code == stillActive
}
