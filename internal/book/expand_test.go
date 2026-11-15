package book

import (
	"reflect"
	"testing"

	"github.com/kojio145/configchief/internal/model"
)

// expandBook is a book of three devices: rt-a01 and rt-a02 are in the group
// "edge", rt-a03 is in no group.
func expandBook() *Book {
	d := model.NewDataset("book")
	put(d, "devices", kv("hostname", "rt-a01"), "vendor", "cisco_ios", "group", "edge|sample")
	put(d, "devices", kv("hostname", "rt-a02"), "vendor", "cisco_ios", "group", "edge")
	put(d, "devices", kv("hostname", "rt-a03"), "vendor", "nec_ix")
	return &Book{File: "book.xlsx", Data: d, Sheets: map[string]*SheetInfo{}}
}

func kinds(ps []Problem) []string {
	var out []string
	for _, p := range ps {
		out = append(out, p.Kind)
	}
	return out
}

// A row for "*" or "@group" becomes a row of each device it stands for, and
// of the rows with the same key the nearest one wins as a whole (spec 9.6).
func TestExpandRows(t *testing.T) {
	sc := defs(t)
	b := expandBook()
	put(b.Data, "ntp_servers", kv("device", "*", "server", "192.0.2.123"), "prefer", "true", "source_if", "Loopback0")
	put(b.Data, "ntp_servers", kv("device", "@edge", "server", "192.0.2.123"), "source_if", "Loopback1")
	put(b.Data, "ntp_servers", kv("device", "rt-a02", "server", "192.0.2.123"), "prefer", "false")
	put(b.Data, "ntp_servers", kv("device", "@sample", "server", "192.0.2.124"))
	before := flat(b.Data)

	ds, problems := b.Expand(sc)
	if len(problems) != 0 {
		t.Fatalf("problems: %v", problems)
	}
	got := flat(ds)["ntp_servers"]
	want := map[string]map[string]string{
		// The row of the group, whole: "prefer" of the "*" row is not mixed in.
		"device=rt-a01,server=192.0.2.123": {"device": "rt-a01", "server": "192.0.2.123", "source_if": "Loopback1"},
		"device=rt-a01,server=192.0.2.124": {"device": "rt-a01", "server": "192.0.2.124"},
		"device=rt-a02,server=192.0.2.123": {"device": "rt-a02", "server": "192.0.2.123", "prefer": "false"},
		"device=rt-a03,server=192.0.2.123": {"device": "rt-a03", "server": "192.0.2.123", "prefer": "true", "source_if": "Loopback0"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ntp_servers:\n got %v\nwant %v", got, want)
	}
	if !reflect.DeepEqual(flat(b.Data), before) {
		t.Error("the rows of the book were changed")
	}
	// The rows of a device stay together, in the order of the devices.
	var order []string
	for _, r := range ds.Tables["ntp_servers"].Rows {
		order = append(order, r.Get("device"))
	}
	if want := []string{"rt-a01", "rt-a01", "rt-a02", "rt-a03"}; !reflect.DeepEqual(order, want) {
		t.Errorf("order = %v, want %v", order, want)
	}
}

func TestExpandCommon(t *testing.T) {
	sc := defs(t)
	b := expandBook()
	put(b.Data, "syslog_servers", kv("device", "*", "server", "192.0.2.50"), "severity", "warnings")
	put(b.Data, "syslog_servers", kv("device", "rt-a03", "server", "192.0.2.51"), "facility", "local3")
	b.Common = []Common{
		{Scope: "*", Table: "syslog_servers", Column: "facility", Value: "local6", Row: 2},
		{Scope: "@edge", Table: "ログ転送先", Column: "facility", Value: "local5", Row: 3},
		{Scope: "*", Table: "syslog_servers", Column: "severity", Value: "errors", Row: 4},
		{Scope: "*", Table: "devices", Column: "name_servers", Value: "192.0.2.53, 192.0.2.54", Row: 5},
	}
	ds, problems := b.Expand(sc)
	if len(problems) != 0 {
		t.Fatalf("problems: %v", problems)
	}
	got := flat(ds)
	for key, want := range map[string]map[string]string{
		"device=rt-a01,server=192.0.2.50": {"facility": "local5", "severity": "warnings"}, // the group before "*"
		"device=rt-a03,server=192.0.2.50": {"facility": "local6", "severity": "warnings"},
		"device=rt-a03,server=192.0.2.51": {"facility": "local3", "severity": "errors"}, // a written cell stays
	} {
		for col, v := range want {
			if g := got["syslog_servers"][key][col]; g != v {
				t.Errorf("%s %s = %q, want %q", key, col, g, v)
			}
		}
	}
	if g := got["devices"]["hostname=rt-a03"]["name_servers"]; g != "192.0.2.53|192.0.2.54" {
		t.Errorf("name_servers = %q", g)
	}
	// The cell tells where its value came from.
	for _, r := range ds.Tables["syslog_servers"].Rows {
		if r.Get("device") == "rt-a01" {
			if o := r.Cells["facility"].Origins; len(o) != 1 || o[0].Kind != "common" || o[0].Addr != "A3" {
				t.Errorf("origin of a common value = %+v", o)
			}
		}
	}
}

func TestExpandProblems(t *testing.T) {
	sc := defs(t)
	b := expandBook()
	put(b.Data, "ntp_servers", kv("device", "rt-z99", "server", "192.0.2.123"))
	put(b.Data, "ntp_servers", kv("device", "@nobody", "server", "192.0.2.123"))
	b.Common = []Common{
		{Scope: "@nobody", Table: "ntp_servers", Column: "prefer", Value: "true", Row: 2},
		{Scope: "rt-a01", Table: "ntp_servers", Column: "prefer", Value: "true", Row: 3},
		{Scope: "*", Table: "no_such_table", Column: "prefer", Value: "true", Row: 4},
		{Scope: "*", Table: "ntp_servers", Column: "no_such_column", Value: "true", Row: 5},
		{Scope: "*", Table: "ntp_servers", Column: "server", Value: "192.0.2.124", Row: 6},
		{Scope: "*", Table: "ntp_servers", Column: "prefer", Value: "sometimes", Row: 7},
	}
	ds, problems := b.Expand(sc)
	want := []string{"V-107", "V-107", "V-107", "V-107", "V-106", "V-106", "V-106", "V-102"}
	if got := kinds(problems); !reflect.DeepEqual(got, want) {
		t.Errorf("kinds = %v, want %v\n%v", got, want, problems)
	}
	if tab := ds.Tables["ntp_servers"]; tab != nil && len(tab.Rows) != 0 {
		t.Errorf("rows for nobody: %d", len(tab.Rows))
	}
}
