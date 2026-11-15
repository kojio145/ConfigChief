package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The pages in docs/ are the ones the definitions give now.
func TestPagesAreCurrent(t *testing.T) {
	docsDir = filepath.Join("..", "..", "docs")
	pages, err := Pages()
	if err != nil {
		t.Fatal(err)
	}
	for name, want := range pages {
		got, err := os.ReadFile(filepath.Join(docsDir, name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.ReplaceAll(string(got), "\r\n", "\n") != want {
			t.Errorf("docs/%s is not current: run `go run ./cmd/docgen`", name)
		}
	}
}
