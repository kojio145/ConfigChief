package main

// The window opens where it was last closed, at the size it had, maximised
// if it was (spec 14.8.5), inside the work area of a monitor that is still
// there (the screen minus the taskbar). Copied from PalaTerm (mainplace.go,
// winplace.go, v1.10.8), the main window only.
//
// Everything here is in screen coordinates (physical pixels).
// WINDOWPLACEMENT's rcNormalPosition is in workspace coordinates (offset by
// the taskbar when it sits on the left or the top), so it is only read for a
// maximised window, converted, and never written back.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32                  = syscall.NewLazyDLL("user32.dll")
	kernel32                = syscall.NewLazyDLL("kernel32.dll")
	dwmapi                  = syscall.NewLazyDLL("dwmapi.dll")
	pGetWindowRect          = user32.NewProc("GetWindowRect")
	pSetWindowPos           = user32.NewProc("SetWindowPos")
	pMonitorFromRect        = user32.NewProc("MonitorFromRect")
	pMonitorFromWindow      = user32.NewProc("MonitorFromWindow")
	pGetMonitorInfo         = user32.NewProc("GetMonitorInfoW")
	pGetWindowPlacement     = user32.NewProc("GetWindowPlacement")
	pShowWindow             = user32.NewProc("ShowWindow")
	pEnumWindows            = user32.NewProc("EnumWindows")
	pGetWindowThreadProcess = user32.NewProc("GetWindowThreadProcessId")
	pIsWindowVisible        = user32.NewProc("IsWindowVisible")
	pGetWindow              = user32.NewProc("GetWindow")
	pGetDpiForWindow        = user32.NewProc("GetDpiForWindow")
	pGetCurrentProcessId    = kernel32.NewProc("GetCurrentProcessId")
	pDwmGetWindowAttribute  = dwmapi.NewProc("DwmGetWindowAttribute")
)

type rect struct{ L, T, R, B int32 }

type monitorInfo struct {
	CbSize  uint32
	Monitor rect
	Work    rect
	Flags   uint32
}

type point struct{ X, Y int32 }

type windowPlacement struct {
	Length  uint32
	Flags   uint32
	ShowCmd uint32
	MinPos  point
	MaxPos  point
	Normal  rect // workspace coordinates
}

const (
	swShowMinimized     = 2
	swShowMaximized     = 3
	wpfRestoreToMaximum = 2
)

// winState is the last placement: the restored window's rectangle and
// whether it was maximised.
type winState struct {
	L, T, R, B int32
	Max        bool `json:"max,omitempty"`
}

// winStateFile is %APPDATA%\ConfigChief\window.json, next to the settings.
func winStateFile() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "ConfigChief", "window.json")
}

// The search for this process's window.
var (
	enumMu    sync.Mutex
	enumPID   uintptr
	enumFound uintptr
	enumTopCB = syscall.NewCallback(enumTopProc)
)

func enumTopProc(h uintptr, _ uintptr) uintptr {
	const gwOwner = 4
	var wpid uint32
	pGetWindowThreadProcess.Call(h, uintptr(unsafe.Pointer(&wpid)))
	if uintptr(wpid) != enumPID {
		return 1
	}
	if v, _, _ := pIsWindowVisible.Call(h); v == 0 {
		return 1
	}
	if owner, _, _ := pGetWindow.Call(h, gwOwner); owner != 0 {
		return 1 // a tooltip or a dialog owned by the frame
	}
	enumFound = h
	return 0
}

// ownWindow is this process's visible, unowned top-level window (0 = none
// yet).
func ownWindow() uintptr {
	enumMu.Lock()
	defer enumMu.Unlock()
	enumPID, _, _ = pGetCurrentProcessId.Call()
	enumFound = 0
	pEnumWindows.Call(enumTopCB, 0)
	return enumFound
}

func waitOwnWindow() uintptr {
	for i := 0; i < 250; i++ {
		if h := ownWindow(); h != 0 {
			return h
		}
		time.Sleep(40 * time.Millisecond)
	}
	return 0
}

func monitorWork(mon uintptr) (monitorInfo, bool) {
	mi := monitorInfo{CbSize: uint32(unsafe.Sizeof(monitorInfo{}))}
	if mon == 0 {
		return mi, false
	}
	if ok, _, _ := pGetMonitorInfo.Call(mon, uintptr(unsafe.Pointer(&mi))); ok == 0 {
		return mi, false
	}
	return mi, mi.Work.R > mi.Work.L && mi.Work.B > mi.Work.T
}

func windowMonitor(hwnd uintptr) (monitorInfo, bool) {
	const monitorDefaultToNearest = 2
	mon, _, _ := pMonitorFromWindow.Call(hwnd, monitorDefaultToNearest)
	return monitorWork(mon)
}

func windowDPI(hwnd uintptr) int32 {
	if pGetDpiForWindow.Find() != nil {
		return 96
	}
	d, _, _ := pGetDpiForWindow.Call(hwnd)
	if d == 0 {
		return 96
	}
	return int32(d)
}

// frameRects is the window's rectangle and the visible part of its frame
// (Windows 10/11 frames carry an invisible resize border).
func frameRects(hwnd uintptr) (win, ext rect) {
	pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&win)))
	const dwmaExtendedFrameBounds = 9
	if r, _, _ := pDwmGetWindowAttribute.Call(hwnd, dwmaExtendedFrameBounds, uintptr(unsafe.Pointer(&ext)), unsafe.Sizeof(ext)); r != 0 {
		ext = rect{}
	}
	return win, ext
}

// placeWindow moves the window to target, and again after 150 ms: landing on
// a monitor with other scaling resizes the frame.
func placeWindow(hwnd uintptr, target func(win, ext rect, dpi int32) rect) {
	const swpNoZOrder, swpNoActivate = 0x0004, 0x0010
	move := func() {
		win, ext := frameRects(hwnd)
		r := target(win, ext, windowDPI(hwnd))
		if r.R > r.L && r.B > r.T {
			pSetWindowPos.Call(hwnd, 0, uintptr(r.L), uintptr(r.T), uintptr(r.R-r.L), uintptr(r.B-r.T), swpNoZOrder|swpNoActivate)
		}
	}
	move()
	time.Sleep(150 * time.Millisecond)
	move()
}

// placeMainWindow puts the window where it belongs. Runs in the background
// from startup: the frame may not be on screen yet.
func placeMainWindow() {
	go func() {
		defer recoverGo("placeMainWindow")
		hwnd := waitOwnWindow()
		if hwnd == 0 {
			return
		}
		if b, err := os.ReadFile(winStateFile()); err == nil {
			var st winState
			_ = json.Unmarshal(b, &st)
			if r := (rect{st.L, st.T, st.R, st.B}); r.R-r.L >= 200 && r.B-r.T >= 150 {
				const monitorDefaultToNull = 0
				mon, _, _ := pMonitorFromRect.Call(uintptr(unsafe.Pointer(&r)), monitorDefaultToNull)
				if mi, ok := monitorWork(mon); ok {
					placeWindow(hwnd, func(win, ext rect, _ int32) rect { return clampInto(r, expandBy(mi.Work, win, ext)) })
					if st.Max {
						pShowWindow.Call(hwnd, swShowMaximized)
					}
					return
				}
			}
		}
		mi, ok := windowMonitor(hwnd)
		if !ok {
			return
		}
		placeWindow(hwnd, func(win, ext rect, dpi int32) rect {
			return frameFit(firstPlace(mi.Work, int32(winWidth)*dpi/96, int32(winHeight)*dpi/96), win, ext)
		})
	}()
}

// saveMainPlacement records where the window is now, for the next start.
func saveMainPlacement() {
	defer recoverGo("saveMainPlacement")
	hwnd := ownWindow()
	if hwnd == 0 || winStateFile() == "" {
		return
	}
	wp := windowPlacement{Length: uint32(unsafe.Sizeof(windowPlacement{}))}
	if ok, _, _ := pGetWindowPlacement.Call(hwnd, uintptr(unsafe.Pointer(&wp))); ok == 0 {
		return
	}
	max := wp.ShowCmd == swShowMaximized || (wp.ShowCmd == swShowMinimized && wp.Flags&wpfRestoreToMaximum != 0)
	var r rect
	if wp.ShowCmd == swShowMaximized || wp.ShowCmd == swShowMinimized {
		mi, ok := windowMonitor(hwnd)
		if !ok {
			return
		}
		r = workspaceToScreen(wp.Normal, mi)
	} else if ok, _, _ := pGetWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&r))); ok == 0 {
		return
	}
	if r.R-r.L < 200 || r.B-r.T < 150 {
		return
	}
	b, _ := json.Marshal(winState{L: r.L, T: r.T, R: r.R, B: r.B, Max: max})
	_ = os.MkdirAll(filepath.Dir(winStateFile()), 0o755)
	_ = os.WriteFile(winStateFile(), b, 0o644)
}
