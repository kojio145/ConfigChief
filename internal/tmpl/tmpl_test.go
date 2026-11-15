package tmpl

import (
	"errors"
	"fmt"
	"reflect"
	"sort"
	"strings"
	"testing"
)

type rec struct {
	cells map[string]any
	read  map[string]bool
}

func (r *rec) Field(name string) (any, bool) {
	v, ok := r.cells[name]
	if ok {
		r.read[name] = true
	}
	return v, ok
}

func (r *rec) Fields() []string {
	var ks []string
	for k := range r.cells {
		ks = append(ks, k)
	}
	sort.Strings(ks)
	return ks
}

func engine() *Engine {
	return New(Options{
		IfName: func(s string) (string, error) {
			if rest, ok := strings.CutPrefix(s, "Gi"); ok && !strings.HasPrefix(s, "Gigabit") {
				return "GigabitEthernet" + rest, nil
			}
			return s, nil
		},
		Unlex: func(kind, word string) (string, error) {
			if kind == "hash" && word == "sha" {
				return "sha1", nil
			}
			return "", fmt.Errorf("%s の語「%s」に当たる語がありません", kind, word)
		},
	})
}

func TestExpand(t *testing.T) {
	e := engine()
	vars := map[string]any{"ip": "192.0.2.1", "mask": "255.255.255.252", "name": "Gi0/0/0", "dist": "", "dir": "in", "n": 3}
	for _, c := range []struct{ src, want string }{
		{"plain text", "plain text"},
		{"{{ ip | with_mask(mask) }}", "192.0.2.1/30"},
		{"{{ ip | with_mask(mask) | network }}", "192.0.2.0/30"},
		{"{{ '192.0.2.1/30' | mask }} {{ '192.0.2.1/30' | wildcard }} {{ '192.0.2.1/30' | prefixlen }}", "255.255.255.252 0.0.0.3 30"},
		{"{{ '192.0.2.1/30' | broadcast }} {{ '192.0.2.0/30' | host(2) }} {{ '192.0.2.0' | with_wildcard('0.0.0.255') }}", "192.0.2.3 192.0.2.2 192.0.2.0/24"},
		{"{{ name | ifname }}", "GigabitEthernet0/0/0"},
		{"acl_{{ dir }}", "acl_in"},
		{"{{ dist }}", ""},
		{"{{ '0.0.0.0/0' if name == 'default' else name }}", "Gi0/0/0"},
		{"{{ 'sha' | unlex('hash') }}", "sha1"},
		{"{{ '192.0.2.9' | in_net('192.0.2.0/24') }}", "True"},
		{"{{ '16.10' | version_ge('16.9') }} {{ '15.2' | version_ge('16.9') }}", "True False"},
		{"{{ ['Gi0/0/10', 'Gi0/0/2', 'Gi0/0/1'] | natural | join(' ') }}", "Gi0/0/1 Gi0/0/2 Gi0/0/10"},
		{"{{ n + 1 }}", "4"},
	} {
		got, err := e.Expand(c.src, vars)
		if err != nil || got != c.want {
			t.Errorf("%s\n got  %q, %v\n want %q", c.src, got, err, c.want)
		}
	}
}

func TestExpandErrors(t *testing.T) {
	e := engine()
	for _, c := range []struct{ src, want string }{
		{"{{ nosuch }}", "undefined"}, // strict: a typo in a rule is an error
		{"{{ '192.0.2.1' | with_mask('255.0.255.0') }}", "連続したマスク"}, // the filter's own words
		{"{{ 'aes' | unlex('hash') }}", "当たる語がありません"},
		{"{{ 'x' | lex('hash') }}", "使えません"},
		{"{{ '' | required('外側 IF') }}", "外側 IF が空です"},
		{"{{ fail('止めます') }}", "止めます"},
		{"{{ 1 +", ""},
	} {
		_, err := e.Expand(c.src, map[string]any{})
		var te *Error
		if err == nil || !errors.As(err, &te) || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want one with %q", c.src, err, c.want)
		}
	}
	_, err := e.Expand("a\nb\n{{ '' | required('x') }}\n", nil)
	var te *Error
	if !errors.As(err, &te) || te.Line != 3 {
		t.Errorf("line = %+v", err)
	}
}

func TestEval(t *testing.T) {
	e := engine()
	row := &rec{cells: map[string]any{"name": "ISAKMP-SAMPLE", "priority": "10", "unused": "x"}, read: map[string]bool{}}
	e.SetFunc("lookup", func(args []any, kwargs map[string]any) (any, error) {
		if len(args) == 1 && args[0] == "x_profiles" && kwargs["name"] == "ISAKMP-SAMPLE" {
			return row, nil
		}
		return nil, nil
	})
	vars := map[string]any{"t": row, "v": "10", "empty": "", "list": []string{"a", "b"}}
	for _, c := range []struct {
		expr string
		want any
	}{
		{"v == '10'", true},
		{"empty", ""},
		{"v | int + 1", int64(11)},
		{"1.5", 1.5},
		{"none", nil},
		{"list", []any{"a", "b"}},
		{"{'a': 1}", map[string]any{"a": int64(1)}},
		{"t.name", "ISAKMP-SAMPLE"},
		{"lookup('x_profiles', name=t.name)", Record(row)},
		{"lookup('x_profiles', name='other')", nil},
		{"lookup('x_profiles', name=t.name) and lookup('x_profiles', name='other')", nil},
		{"t.priority in ['10', '20']", true},
	} {
		got, err := e.Eval(c.expr, vars)
		if err != nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s = %#v, %v; want %#v", c.expr, got, err, c.want)
		}
	}
	if !row.read["name"] || !row.read["priority"] || row.read["unused"] {
		t.Errorf("read = %v", row.read)
	}
	if _, err := e.Eval("t.nosuch == 1", vars); err == nil {
		t.Error("no error for a field that does not exist")
	}
	for v, want := range map[any]bool{nil: false, "": false, "a": true, false: false, int64(0): false, int64(2): true} {
		if Truth(v) != want {
			t.Errorf("Truth(%#v) != %v", v, want)
		}
	}
	if !Truth(Record(row)) || Truth([]any{}) {
		t.Error("Truth of a record or an empty list")
	}
}
