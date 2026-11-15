//go:build !windows

package main

import "os"

// terminal gives where a command writes.
func terminal() (out, errOut *os.File) { return os.Stdout, os.Stderr }
