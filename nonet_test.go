package main

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The tool talks to nobody (spec 16): its own code calls nothing of the
// network. What the window is built on (Wails) serves the page to the
// WebView2 inside the process and is not covered here; it opens no port in
// the build the users get.

// netImport tells whether an import path is one of the network.
func netImport(path string) bool {
	switch {
	case path == "net/netip": // addresses as values: no communication
		return false
	case path == "golang.org/x/net/html", strings.HasPrefix(path, "golang.org/x/net/html/"): // reads HTML text
		return false
	case path == "net", strings.HasPrefix(path, "net/"):
		return true
	case path == "crypto/tls", strings.HasPrefix(path, "golang.org/x/net"):
		return true
	}
	return false
}

func TestNoNetworkInGoCode(t *testing.T) {
	fset := token.NewFileSet()
	files := 0
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if name := d.Name(); name == "build" || (strings.HasPrefix(name, ".") && name != ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		files++
		for _, imp := range f.Imports {
			p, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			if netImport(p) {
				t.Errorf("%s imports %s", filepath.ToSlash(path), p)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if files < 20 {
		t.Fatalf("only %d Go files were looked at: the test does not see the code", files)
	}
}

// The page asks the tool over the bridge of the window and nothing else: it
// names no address and uses none of the ways a page has to send something.
func TestNoNetworkInPage(t *testing.T) {
	words := []string{"http://", "https://", "//cdn", "fetch(", "XMLHttpRequest", "WebSocket", "EventSource", "sendBeacon", "import("}
	entries, err := os.ReadDir("frontend/dist")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("frontend/dist is empty")
	}
	for _, e := range entries {
		data, err := os.ReadFile(filepath.Join("frontend/dist", e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		for _, w := range words {
			if strings.Contains(string(data), w) {
				t.Errorf("frontend/dist/%s contains %q", e.Name(), w)
			}
		}
	}
}
