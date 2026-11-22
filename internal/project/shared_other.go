//go:build !windows

package project

func remoteDrive(abs string) bool { return false }
