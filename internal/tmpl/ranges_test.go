package tmpl

import (
	"fmt"
	"testing"
)

// A list of numbers and ranges becomes the numbers, in order and once each.
func TestRanges(t *testing.T) {
	for _, c := range []struct {
		in   []string
		want string
	}{
		{[]string{"10", "30-32", "10"}, "[10 30 31 32]"},
		{[]string{"20,10 5-6"}, "[5 6 10 20]"},
		{[]string{"all"}, "[]"},
		{nil, "[]"},
	} {
		got, err := Ranges(c.in)
		if err != nil || fmt.Sprint(got) != c.want {
			t.Errorf("%v: %v, %v; want %s", c.in, got, err, c.want)
		}
	}
	for _, bad := range []string{"10-", "32-30", "a1", "1-9999"} {
		if _, err := Ranges([]string{bad}); err == nil {
			t.Errorf("%q: no error", bad)
		}
	}
	e := New(Options{})
	if got, err := e.Expand("{{ ['10', '30-31'] | ranges | join(' ') }}", nil); err != nil || got != "10 30 31" {
		t.Errorf("the filter: %q, %v", got, err)
	}
}
