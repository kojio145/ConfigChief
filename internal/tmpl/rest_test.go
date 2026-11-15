package tmpl

import (
	"strings"
	"testing"
)

// The places where a template stops are told one after another (spec 9.6).
func TestRest(t *testing.T) {
	src := "line 1 {{ r.name | required('name') }}\n" +
		"{% if r.zone != '+9' %}{{ fail('zone ' ~ r.zone) }}{% endif %}\n" +
		"line 3 {{ r.word | lex('kind') }}\n" +
		"line 4 {{ r.empty | required('empty') }}\n" +
		"line 5 {{ r.known | lex('kind') }}\n" +
		"{{ fail('last') }}\n"
	e := loaded(map[string]string{"t.j2": src})
	vars := map[string]any{"r": plain{"name": "a", "zone": "JST +9", "word": "other", "empty": "", "known": "known"}}

	text, _, err := e.Render("t.j2", vars, nil)
	first, ok := err.(*Error)
	if !ok || !first.Stopped || first.Msg != "zone JST +9" || first.Line != 2 || first.Kind != "" {
		t.Fatalf("the first failure = %+v", err)
	}
	if text != "line 1 a\n" {
		t.Errorf("the text before the first failure = %q", text)
	}

	rest, more := e.Rest("t.j2", vars, 10)
	if more || len(rest) != 3 {
		t.Fatalf("rest = %v, more %v", rest, more)
	}
	for i, want := range []struct {
		line int
		kind string
		msg  string
	}{
		{3, "lex", "no word"},
		{4, "required", "empty が空です"},
		{6, "", "last"},
	} {
		got := rest[i]
		if got.Line != want.line || got.Kind != want.kind || !strings.Contains(got.Msg, want.msg) || got.Name != "t.j2" || !got.Stopped {
			t.Errorf("rest[%d] = %+v, want line %d, kind %q, %q", i, got, want.line, want.kind, want.msg)
		}
	}

	// No more than asked for, and it is told that there are more.
	rest, more = e.Rest("t.j2", vars, 2)
	if !more || len(rest) != 2 || rest[1].Line != 4 {
		t.Errorf("rest = %v, more %v", rest, more)
	}
	if rest, more = e.Rest("t.j2", vars, 3); more || len(rest) != 3 {
		t.Errorf("as many as asked for: rest = %v, more %v", rest, more)
	}

	// The next run of the template stops at the first place again.
	if _, _, err := e.Render("t.j2", vars, nil); err == nil || err.Error() != "zone JST +9" {
		t.Errorf("after Rest: err = %v", err)
	}
	// A template that does not stop has nothing more.
	vars["r"] = plain{"name": "a", "zone": "+9", "word": "known", "empty": "x", "known": "known"}
	e = loaded(map[string]string{"t.j2": strings.Replace(src, "{{ fail('last') }}\n", "", 1)})
	if _, _, err := e.Render("t.j2", vars, nil); err != nil {
		t.Fatal(err)
	}
	if rest, more = e.Rest("t.j2", vars, 10); more || len(rest) != 0 {
		t.Errorf("a whole template: rest = %v, more %v", rest, more)
	}
}

// What the engine itself refuses after a place that was passed over is not
// told: the empty value put there may be its only cause.
func TestRestEngineError(t *testing.T) {
	e := loaded(map[string]string{
		"t.j2": "{{ fail('first') }}\n{{ fail('second') }}\n{{ r.name | nosuchfilter }}\n{{ fail('never') }}\n",
	})
	vars := map[string]any{"r": plain{"name": "a"}}
	if _, _, err := e.Render("t.j2", vars, nil); err == nil || err.Error() != "first" {
		t.Fatalf("err = %v", err)
	}
	rest, more := e.Rest("t.j2", vars, 10)
	if more || len(rest) != 1 || rest[0].Msg != "second" {
		t.Errorf("rest = %v, more %v", rest, more)
	}
	// A failure of the engine comes first: there is nothing to look past.
	e = loaded(map[string]string{"t.j2": "{{ r.name | nosuchfilter }}\n{{ fail('after') }}\n"})
	_, _, err := e.Render("t.j2", vars, nil)
	if te, ok := err.(*Error); !ok || te.Stopped {
		t.Fatalf("err = %+v", err)
	}
	if rest, more = e.Rest("t.j2", vars, 10); more || len(rest) != 0 {
		t.Errorf("rest = %v, more %v", rest, more)
	}
	if rest, more = e.Rest("none.j2", vars, 10); more || len(rest) != 0 {
		t.Errorf("a template that is not there: rest = %v, more %v", rest, more)
	}
}
