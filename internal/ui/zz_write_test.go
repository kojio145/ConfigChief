package ui

import (
	"os"
	"path/filepath"
)

// writeTestFile writes a file of a test, making its folder first: a case
// has only the folders it used (spec 14.15.5).
func writeTestFile(name string, data []byte, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(name), 0o755); err != nil {
		return err
	}
	return os.WriteFile(name, data, perm)
}
