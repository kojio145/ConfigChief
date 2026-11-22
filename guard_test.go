package main

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/errlog"
)

// Every method the page calls starts with the guard (spec 14.8.4): a method
// added without it would let an internal error end the window.
func TestEveryBoundMethodIsGuarded(t *testing.T) {
	fset := token.NewFileSet()
	files, _ := filepath.Glob("*.go")
	n := 0
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Recv == nil || !fn.Name.IsExported() {
				continue
			}
			if star, ok := fn.Recv.List[0].Type.(*ast.StarExpr); !ok || star.X.(*ast.Ident).Name != "App" {
				continue
			}
			n++
			ok = false
			if len(fn.Body.List) > 0 {
				if ds, isDefer := fn.Body.List[0].(*ast.DeferStmt); isDefer {
					if sel, isSel := ds.Call.Fun.(*ast.SelectorExpr); isSel && sel.Sel.Name == "guard" {
						ok = true
					}
				}
			}
			if !ok {
				t.Errorf("%s: %s does not start with defer a.guard", path, fn.Name.Name)
			}
		}
	}
	if n < 60 {
		t.Fatalf("only %d methods were looked at", n)
	}
}

func TestGuardTurnsAPanicIntoAnError(t *testing.T) {
	dir := t.TempDir()
	was := errlog.Path
	errlog.Path = func() string { return filepath.Join(dir, "error.log") }
	defer func() { errlog.Path = was }()
	a := &App{}
	f := func() (err error) {
		defer a.guard("Sample", &err)
		var m map[string]int
		m["x"] = 1 // a panic
		return nil
	}
	err := f()
	if err == nil || !strings.Contains(err.Error(), "内部エラーが発生しました（Sample:") {
		t.Fatalf("err = %v", err)
	}
	if data, _ := os.ReadFile(errlog.Path()); !strings.Contains(string(data), "panic in Sample") {
		t.Errorf("the log: %q", data)
	}
	g := func() { defer a.guard("NoError", nil); panic(errors.New("x")) }
	g() // no ctx: written down only, and the call returns
}

func TestWindowArithmetic(t *testing.T) {
	work := rect{0, 0, 1920, 1040}
	if r := clampInto(rect{1800, 900, 3080, 1720}, work); r != (rect{640, 220, 1920, 1040}) {
		t.Errorf("clampInto moves a window back on screen: %+v", r)
	}
	if r := clampInto(rect{-100, 0, 2500, 1500}, work); r != work {
		t.Errorf("clampInto shrinks a window larger than the screen: %+v", r)
	}
	if r := firstPlace(work, 1280, 820); r != (rect{320, 110, 1600, 930}) {
		t.Errorf("firstPlace: %+v", r)
	}
	if r := firstPlace(rect{0, 0, 1280, 672}, 1600, 1025); r != (rect{0, 0, 1280, 672}) {
		t.Errorf("firstPlace on a small screen: %+v", r)
	}
	if r := expandBy(work, rect{-7, 0, 1287, 827}, rect{0, 0, 1280, 820}); r != (rect{-7, 0, 1927, 1047}) {
		t.Errorf("expandBy: %+v", r)
	}
	if r := workspaceToScreen(rect{0, 0, 100, 100}, monitorInfo{Monitor: rect{0, 0, 1920, 1080}, Work: rect{48, 0, 1920, 1080}}); r != (rect{48, 0, 148, 100}) {
		t.Errorf("workspaceToScreen: %+v", r)
	}
}
