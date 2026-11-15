//go:build !windows

package main

import "fmt"

// openPath opens a file or a folder with the program of the system. The
// window is built for Windows only (spec 2); elsewhere the path is told.
func openPath(path string) error {
	return fmt.Errorf("この OS では開けません。場所: %s", path)
}
