package tmpl

import (
	"strings"
	"testing"
)

type plain map[string]any

func (r plain) Field(name string) (any, bool) { v, ok := r[name]; return v, ok }
func (r plain) Fields() []string {
	var out []string
	for k := range r {
		out = append(out, k)
	}
	return out
}

func loaded(files map[string]string) *Engine {
	e := New(Options{Lex: func(kind, word string) (string, error) {
		if word == "known" {
			return "KNOWN", nil
		}
		return "", errString("no word")
	}})
	e.SetLoader(func(name string) (string, bool) {
		s, ok := files[name]
		return s, ok
	})
	return e
}

type errString string

func (e errString) Error() string { return string(e) }

// What the templates of the packs lean on (appendix B 5.4 and spec 9.6).
func TestRenderForms(t *testing.T) {
	rows := []any{
		plain{"name": "a", "on": true, "off": false, "empty": "", "n": int64(10), "list": []string{"x", "y"}, "none": []string{}},
		plain{"name": "b", "on": true, "off": false, "empty": "", "n": int64(20), "list": []string{}, "none": []string{}},
	}
	cases := []struct{ name, src, want string }{
		{"empty is not false", "{% if r.empty == false %}eq{% else %}ne{% endif %}", "ne"},
		{"false is false", "{% if r.off == false %}eq{% else %}ne{% endif %}", "eq"},
		{"empty is falsy", "{% if r.empty %}t{% else %}f{% endif %}", "f"},
		{"empty list is falsy", "{% if r.none %}t{% else %}f{% endif %}", "f"},
		{"list is truthy", "{% if r.list %}t{% else %}f{% endif %}", "t"},
		{"inline if", "a{{ ' b' if r.on else '' }}{{ ' c' if r.empty else '' }}", "a b"},
		{"number joins", "{{ 'x ' ~ r.n }}", "x 10"},
		{"split", "{% set p = 'JST +9' | split(' ') %}{{ p[0] }}/{{ p[1] }}/{{ p | length }}", "JST/+9/2"},
		{"startingwith", "{% if '$1$abc' is startingwith('$1$') %}y{% endif %}", "y"},
		{"index", "{{ rows[0].name }}{{ rows[1].n }}", "a20"},
		{"cidr test", "{% if '192.0.2.1/30' is cidr %}y{% endif %}{% if 'dhcp' is cidr %}n{% endif %}", "y"},
		{"in list", "{% if r.name in ['a', 'z'] %}y{% endif %}", "y"},
		{"set line leaves nothing", "{% set x = 1 %}\nline {{ x }}\n", "line 1\n"},
		{"block lines leave nothing", "{% for q in rows %}\n{% if q.on %}\n item {{ q.name }}\n{% endif %}\n{% endfor %}\nend\n", " item a\n item b\nend\n"},
		{"namespace", "{% set ns = namespace(same=true) %}\n{% for q in rows %}\n{% if q.n != rows[0].n %}\n{% set ns.same = false %}\n{% endif %}\n{% endfor %}\n{{ 'same' if ns.same else 'differ' }}\n", "differ\n"},
		{"join", "{{ r.list | join(' ') }}", "x y"},
		{"replace", "{{ '+9' | replace('+', '') }}", "9"},
	}
	for _, c := range cases {
		e := loaded(map[string]string{"t.j2": c.src})
		got, _, err := e.Render("t.j2", map[string]any{"r": rows[0], "rows": rows}, nil)
		if err != nil {
			t.Errorf("%s: %v", c.name, err)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %q, want %q", c.name, got, c.want)
		}
	}
}

func TestRenderPiecesAndWrites(t *testing.T) {
	e := loaded(map[string]string{
		"main.j2": "head {{ r.name }}\n{% include 'part.j2' %}\ntail\n",
		"part.j2": " inner {{ r.n }}\n",
	})
	var writes [][2]int
	got, pieces, err := e.Render("main.j2", map[string]any{"r": plain{"name": "a", "n": int64(7)}}, func(s, e int) {
		writes = append(writes, [2]int{s, e})
	})
	if err != nil {
		t.Fatal(err)
	}
	if want := "head a\n inner 7\ntail\n"; got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
	if len(pieces) == 0 {
		t.Fatal("no pieces")
	}
	// The pieces cover the text in order, and the writes are the same parts.
	pos := 0
	for _, p := range pieces {
		if p.Start != pos || p.End <= p.Start {
			t.Fatalf("pieces do not follow each other: %+v", pieces)
		}
		pos = p.End
	}
	if pos != len(got) {
		t.Fatalf("pieces end at %d of %d", pos, len(got))
	}
	if len(writes) != len(pieces) {
		t.Fatalf("%d writes, %d pieces", len(writes), len(pieces))
	}
	for i, p := range pieces {
		if writes[i] != [2]int{p.Start, p.End} {
			t.Fatalf("write %d is %v, piece is %+v", i, writes[i], p)
		}
	}
	at := func(sub string) Piece {
		off := strings.Index(got, sub)
		for _, p := range pieces {
			if p.Start <= off && off < p.End {
				return p
			}
		}
		t.Fatalf("no piece for %q", sub)
		return Piece{}
	}
	if p := at("a\n"); !p.Expr || p.Name != "main.j2" || p.Line != 1 {
		t.Errorf("value of main: %+v", p)
	}
	if p := at("7"); !p.Expr || p.Name != "part.j2" || p.Line != 1 {
		t.Errorf("value of part: %+v", p)
	}
	if p := at("tail"); p.Expr || p.Name != "main.j2" || p.Line != 3 {
		t.Errorf("text of main: %+v", p)
	}
}

func TestRenderErrors(t *testing.T) {
	cases := []struct{ name, src, kind, text, msg string }{
		{"fail", "one\n{{ fail('stop here') }}\ntwo\n", "", "one\n", "stop here"},
		{"required", "one\n x {{ r.empty | required('the value') }}\n", "required", "one\n x ", "the value"},
		{"lex", "{{ 'unknown' | lex('kind') }}\n", "lex", "", "no word"},
		{"undefined", "one\n{{ r.nothing }}\n", "", "one\n", ""},
	}
	for _, c := range cases {
		e := loaded(map[string]string{"t.j2": c.src})
		got, pieces, err := e.Render("t.j2", map[string]any{"r": plain{"empty": ""}}, nil)
		if err == nil {
			t.Errorf("%s: no error", c.name)
			continue
		}
		te, ok := err.(*Error)
		if !ok {
			t.Errorf("%s: error is %T", c.name, err)
			continue
		}
		if te.Kind != c.kind || got != c.text || pieces != nil || !strings.Contains(te.Msg, c.msg) || te.Line != 2 && c.name != "lex" {
			t.Errorf("%s: kind %q, text %q, line %d, msg %q", c.name, te.Kind, got, te.Line, te.Msg)
		}
	}
	e := loaded(map[string]string{})
	if _, _, err := e.Render("none.j2", nil, nil); err == nil {
		t.Error("a missing template gives no error")
	}
	if err := e.Check("bad.j2", "{% if %}"); err == nil {
		t.Error("a broken template passes Check")
	}
	if _, _, err := loaded(map[string]string{"t.j2": "{{ 'known' | lex('kind') }}"}).Render("t.j2", nil, nil); err != nil {
		t.Error(err)
	}
}
