//go:build !windows

package main

// Off Windows the window opens where the system puts it.

type monitorInfo struct {
	CbSize  uint32
	Monitor rect
	Work    rect
	Flags   uint32
}

type rect struct{ L, T, R, B int32 }

func placeMainWindow()   {}
func saveMainPlacement() {}
