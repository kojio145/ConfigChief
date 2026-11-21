package tmpl

import (
	"strings"
	"testing"
)

// What fail() says inside an included template is told as it is, with the
// name and the line of that template, and Rest finds the next failures.
func TestFailInsideInclude(t *testing.T) {
	e := New(Options{})
	src := map[string]string{
		"outer.j2": "line 1\n{% include 'inner.j2' %}\nline 3\n",
		"inner.j2": "inner 1\n{{ fail('the first thing') }}\n{{ fail('the second thing') }}\n",
	}
	e.SetLoader(func(name string) (string, bool) { s, ok := src[name]; return s, ok })
	_, _, err := e.Render("outer.j2", map[string]any{}, nil)
	te, ok := err.(*Error)
	if !ok {
		t.Fatalf("err = %v", err)
	}
	if te.Msg != "the first thing" || te.Name != "inner.j2" || te.Line != 2 || !te.Stopped {
		t.Errorf("the error: %+v", te)
	}
	if strings.Contains(te.Error(), "bad include") {
		t.Errorf("the error tells of the include: %s", te.Error())
	}
	rest, more := e.Rest("outer.j2", map[string]any{}, 5)
	if len(rest) != 1 || more || rest[0].Msg != "the second thing" || rest[0].Line != 3 {
		t.Errorf("rest = %+v, more = %v", rest, more)
	}
}
