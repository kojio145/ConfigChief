package netx

import (
	"strconv"
	"testing"
)

func TestOne(t *testing.T) {
	cases := []struct {
		name string
		f    func(string) (string, error)
		in   string
		want string // "!" = error
	}{
		{"ip", IP, "192.0.2.1/30", "192.0.2.1"},
		{"ip", IP, "192.0.2.1", "192.0.2.1"},
		{"ip", IP, "2001:db8::1/64", "2001:db8::1"},
		{"ip", IP, "", "!"},
		{"ip", IP, "192.0.2.300/24", "!"},
		{"mask", Mask, "192.0.2.1/30", "255.255.255.252"},
		{"mask", Mask, "192.0.2.1/0", "0.0.0.0"},
		{"mask", Mask, "192.0.2.1", "255.255.255.255"},
		{"mask", Mask, "2001:db8::1/64", "!"},
		{"wildcard", Wildcard, "192.0.2.1/30", "0.0.0.3"},
		{"wildcard", Wildcard, "198.51.100.0/24", "0.0.0.255"},
		{"wildcard", Wildcard, "192.0.2.1/32", "0.0.0.0"},
		{"network", Network, "192.0.2.1/30", "192.0.2.0/30"},
		{"network", Network, "192.0.2.130/25", "192.0.2.128/25"},
		{"network", Network, "2001:db8::1/64", "2001:db8::/64"},
		{"network", Network, "192.0.2.1/33", "!"},
		{"broadcast", Broadcast, "192.0.2.1/30", "192.0.2.3"},
		{"broadcast", Broadcast, "198.51.100.7/24", "198.51.100.255"},
		{"prefixlen", func(s string) (string, error) {
			n, err := PrefixLen(s)
			return strconv.Itoa(n), err
		}, "192.0.2.1/30", "30"},
	}
	for _, c := range cases {
		got, err := c.f(c.in)
		if c.want == "!" {
			if err == nil {
				t.Errorf("%s(%q) = %q, want an error", c.name, c.in, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("%s(%q) = %q, %v; want %q", c.name, c.in, got, err, c.want)
		}
	}
}

func TestTwo(t *testing.T) {
	cases := []struct {
		name string
		f    func(string, string) (string, error)
		a, b string
		want string
	}{
		{"with_mask", WithMask, "192.0.2.1", "255.255.255.252", "192.0.2.1/30"},
		{"with_mask", WithMask, "0.0.0.0", "0.0.0.0", "0.0.0.0/0"},
		{"with_mask", WithMask, "203.0.113.1", "255.255.255.255", "203.0.113.1/32"},
		{"with_mask", WithMask, "192.0.2.1", "255.0.255.0", "!"},
		{"with_mask", WithMask, "192.0.2.1", "0.0.0.255", "!"},
		{"with_mask", WithMask, "rt-a01", "255.255.255.0", "!"},
		{"with_wildcard", WithWildcard, "192.0.2.0", "0.0.0.255", "192.0.2.0/24"},
		{"with_wildcard", WithWildcard, "192.0.2.1", "0.0.0.0", "192.0.2.1/32"},
		{"with_wildcard", WithWildcard, "0.0.0.0", "255.255.255.255", "0.0.0.0/0"},
		{"with_wildcard", WithWildcard, "192.0.2.0", "0.0.255.0", "!"},
	}
	for _, c := range cases {
		got, err := c.f(c.a, c.b)
		if c.want == "!" {
			if err == nil {
				t.Errorf("%s(%q, %q) = %q, want an error", c.name, c.a, c.b, got)
			}
			continue
		}
		if err != nil || got != c.want {
			t.Errorf("%s(%q, %q) = %q, %v; want %q", c.name, c.a, c.b, got, err, c.want)
		}
	}
}

func TestHost(t *testing.T) {
	if got, err := Host("192.0.2.0/30", 2); err != nil || got != "192.0.2.2" {
		t.Errorf("got %q, %v", got, err)
	}
	if got, err := Host("192.0.2.9/30", 1); err != nil || got != "192.0.2.9" {
		t.Errorf("got %q, %v", got, err)
	}
	if _, err := Host("192.0.2.0/30", 4); err == nil {
		t.Error("no error past the end of the network")
	}
}

func TestInNet(t *testing.T) {
	for _, c := range []struct {
		a, n string
		want bool
	}{
		{"192.0.2.1", "192.0.2.0/24", true},
		{"192.0.2.1/30", "192.0.2.0/24", true},
		{"198.51.100.1", "192.0.2.0/24", false},
		{"2001:db8::1", "2001:db8::/32", true},
		{"192.0.2.1", "2001:db8::/32", false},
	} {
		got, err := InNet(c.a, c.n)
		if err != nil || got != c.want {
			t.Errorf("InNet(%q, %q) = %v, %v", c.a, c.n, got, err)
		}
	}
}
