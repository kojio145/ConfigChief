package tmpl

import "testing"

// A value that only names a text is copied without the engine: it has to be
// what the engine gives (the same template after a letter goes through it).
func TestExpandCopied(t *testing.T) {
	e := New(Options{})
	vars := map[string]any{
		"name": "Gi0/1", "empty": "", "odd": "<a & b> {{ x }}", "n": 7, "yes": true, "nothing": nil,
		"parent": map[string]string{"name": "acl-a", "none": ""},
		"line":   map[string]any{"number": 3, "text": "ip route"},
		"list":   []string{"a", "b"},
		"true":   "not the word",
	}
	for _, src := range []string{
		"{{ name }}", "{{name}}", "{{  name  }}", "{{ empty }}", "{{ odd }}", "{{ n }}", "{{ yes }}", "{{ nothing }}",
		"{{ parent.name }}", "{{ parent.none }}", "{{ line.number }}", "{{ line.text }}", "{{ list }}",
		"{{ true }}", "{{ name | upper }}", "{{ name }}{{ name }}", "{{ name ~ n }}", "{{ 1 }}",
		"{{ nope }}", "{{ parent.nope }}", "{{ name.x }}", "{{ a.b.c }}", "{{ .name }}", "{{ name. }}", "{{ 1a }}", "{{ }}",
	} {
		got, err := e.Expand(src, vars)
		want, werr := e.Expand("x"+src, vars)
		if (err == nil) != (werr == nil) {
			t.Errorf("%s: error %v, the engine says %v", src, err, werr)
			continue
		}
		if err == nil && "x"+got != want {
			t.Errorf("%s = %q, the engine gives %q", src, got, want[1:])
		}
	}
	if got, ok := copied("{{ name }}", vars); !ok || got != "Gi0/1" {
		t.Errorf("a plain name is not copied: %q %v", got, ok)
	}
	if got, ok := copied("{{ parent.name }}", vars); !ok || got != "acl-a" {
		t.Errorf("a text of a map is not copied: %q %v", got, ok)
	}
	for _, src := range []string{"{{ n }}", "{{ nope }}", "{{ parent.nope }}", "{{ true }}", "{{ name | upper }}", "{{ line.text }}"} {
		if _, ok := copied(src, vars); ok {
			t.Errorf("%s is copied, and is for the engine", src)
		}
	}
}

// The names of a short expression are handed over as they are asked for: a
// name that is not there is still an error, and a name wins over a function.
func TestNamesAsAsked(t *testing.T) {
	e := New(Options{})
	e.SetFunc("lookup", func([]any, map[string]any) (any, error) { return "from the function", nil })
	vars := map[string]any{"a": "1", "m": map[string]string{"k": "v"}, "none_here": nil, "l": []any{"x", "y"}}
	for expr, want := range map[string]any{
		"a":                      "1",
		"a ~ a":                  "11",
		"m.k":                    "v",
		"none_here is none":      true,
		"nope is defined":        false,
		"a is defined":           true,
		"l | length":             int64(2),
		"lookup()":               "from the function",
		"m.k if a == '1' else 0": "v",
	} {
		got, err := e.Eval(expr, vars)
		if err != nil || got != want {
			t.Errorf("%s = %v (%T), %v; want %v", expr, got, got, err, want)
		}
	}
	if _, err := e.Eval("nope ~ a", vars); err == nil {
		t.Error("no error for a name that is not there")
	}
	if got, err := e.Eval("lookup", map[string]any{"lookup": "the name"}); err != nil || got != "the name" {
		t.Errorf("a name does not win over a function: %v, %v", got, err)
	}
	if got, err := e.Expand("{% for k in [a, a] %}{{ k }}{{ a }}{% endfor %}", vars); err != nil || got != "1111" {
		t.Errorf("a loop: %q, %v", got, err)
	}
}
