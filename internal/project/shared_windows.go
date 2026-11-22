package project

import (
	"path/filepath"
	"syscall"
	"unsafe"
)

var pGetDriveTypeW = kernel32.NewProc("GetDriveTypeW")

// remoteDrive tells a path on a network drive (a drive letter mapped to a
// share).
func remoteDrive(abs string) bool {
	vol := filepath.VolumeName(abs)
	if len(vol) != 2 || vol[1] != ':' {
		return false
	}
	p, err := syscall.UTF16PtrFromString(vol + `\`)
	if err != nil {
		return false
	}
	const driveRemote = 4
	t, _, _ := pGetDriveTypeW.Call(uintptr(unsafe.Pointer(p)))
	return t == driveRemote
}
