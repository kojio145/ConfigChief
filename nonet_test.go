package main

import (
	"go/ast"
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
// network, except internal/update, which asks GitHub for the newest release
// when the user presses its buttons (spec 14.8.1) — and there only the fixed
// addresses of ConfigChief's releases are written. What the window is built on (Wails) serves the page to the
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
			if netImport(p) && !isUpdate(path) {
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

// isUpdate tells the files of internal/update, the one place that may talk
// to the network.
func isUpdate(path string) bool {
	return strings.HasPrefix(filepath.ToSlash(path), "internal/update/")
}

// internal/update names no address but those of ConfigChief's releases on
// GitHub, and only the window (the main package) uses it.
func TestUpdateAddresses(t *testing.T) {
	allowed := []string{
		"https://api.github.com/repos/kojio145/ConfigChief/releases",
		"https://github.com/kojio145/ConfigChief",
		"https://kojio145.github.io/ConfigChief/",
	}
	fset := token.NewFileSet()
	files, err := filepath.Glob("internal/update/*.go")
	if err != nil || len(files) == 0 {
		t.Fatalf("internal/update: %v", err)
	}
	seen := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(f, func(n ast.Node) bool {
			lit, ok := n.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			v, _ := strconv.Unquote(lit.Value)
			if strings.Contains(v, "://") {
				seen++
				ok := false
				for _, a := range allowed {
					ok = ok || strings.HasPrefix(v, a)
				}
				// "://" and "http://" alone are the scheme put before a proxy
				// written without one (proxy_windows.go).
				if !ok && v != "http://" && v != "://" {
					t.Errorf("%s names %q", filepath.ToSlash(path), v)
				}
			}
			return true
		})
	}
	if seen < 5 {
		t.Fatalf("only %d addresses were seen: the test does not see the code", seen)
	}
	err = filepath.WalkDir("internal", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || isUpdate(path) {
			return err
		}
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); p == "github.com/kojio145/configchief/internal/update" {
				t.Errorf("%s uses internal/update: only the window may", filepath.ToSlash(path))
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
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
