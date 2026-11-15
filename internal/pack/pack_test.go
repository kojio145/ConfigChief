package pack

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"
)

const joins = "../importer/testdata/joins"

func read(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(joins + "/" + name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestLoad(t *testing.T) {
	p, err := Load(os.DirFS(joins))
	if err != nil {
		t.Fatal(err)
	}
	if p.ID != "joins" || len(p.Warnings) != 0 {
		t.Errorf("id %q, warnings %v", p.ID, p.Warnings)
	}
	if got, _ := p.IfName("gi0/1"); got != "GigabitEthernet0/1" {
		t.Errorf("IfName = %q", got)
	}
	if got := p.IfKind("Loopback0"); got != "loopback" {
		t.Errorf("IfKind = %q", got)
	}
}

// YAML reads a bare Null as "no value": a type without a name is an error,
// not a type that silently matches nothing.
func TestEmptyInterfaceTypeName(t *testing.T) {
	y := strings.Replace(read(t, "pack.yaml"),
		"- {name: Loopback, short: [Lo], kind: loopback}",
		"- {name: Loopback, short: [Lo], kind: loopback}\n    - {name: Null, short: [], kind: other}", 1)
	if !strings.Contains(y, "name: Null") {
		t.Fatal("the sample pack changed: the line to replace is gone")
	}
	_, err := Load(fstest.MapFS{
		"pack.yaml":            {Data: []byte(y)},
		"import/10_rules.yaml": {Data: []byte(read(t, "import/10_rules.yaml"))},
	})
	if err == nil || !strings.Contains(err.Error(), "name が空") {
		t.Errorf("err = %v, want an error about the empty name", err)
	}
}

// A derive rule may go over a table of the design data, and a rule and a
// derive rule may not share an id.
func TestDeriveIDs(t *testing.T) {
	rules := strings.Replace(read(t, "import/10_rules.yaml"), "- id: relay.each", "- id: relay", 1)
	_, err := Load(fstest.MapFS{
		"pack.yaml":            {Data: []byte(read(t, "pack.yaml"))},
		"import/10_rules.yaml": {Data: []byte(rules)},
	})
	if err == nil || !strings.Contains(err.Error(), "relay") {
		t.Errorf("err = %v, want an error about the id used twice", err)
	}
}

const borrow = "../importer/testdata/borrow"

// children_from must name another rule that has children, and what it borrows
// must find a row to write to.
func TestChildrenFrom(t *testing.T) {
	p, err := Load(os.DirFS(borrow))
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Warnings) != 0 {
		t.Errorf("warnings %v", p.Warnings)
	}
	// The writes of a borrowed rule count for the table it is borrowed under.
	found := false
	for _, w := range p.Writes() {
		if w.RuleID == "note" && w.Table == "vrfs" && w.Column == "description" {
			found = true
		}
	}
	if !found {
		t.Errorf("Writes: no note -> vrfs.description")
	}

	rd := func(name string) string {
		b, err := os.ReadFile(borrow + "/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return string(b)
	}
	rules := rd("import/10_rules.yaml")
	from := "    children_from: zone" + "\n" + "    children:"
	cases := []struct{ name, old, new, want string }{
		{"no such rule", from, strings.Replace(from, "zone", "nowhere", 1), "nowhere"},
		{"itself", from, strings.Replace(from, "zone", "global", 1), "自分自身"},
		{"no children", from, strings.Replace(from, "zone", "note", 1), "children がありません"},
		{"no row", "    row: {table: vrfs, key: {name: \"global\"}}" + "\n", "", "row が要ります"},
	}
	for _, c := range cases {
		y := strings.Replace(rules, c.old, c.new, 1)
		if y == rules {
			t.Fatalf("%s: the sample pack changed: the text to replace is gone", c.name)
		}
		_, err := Load(fstest.MapFS{
			"pack.yaml":            {Data: []byte(rd("pack.yaml"))},
			"import/10_rules.yaml": {Data: []byte(y)},
		})
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%s: err = %v, want an error with %q", c.name, err, c.want)
		}
	}
}

// A session needs its prompt and a command, and a pattern of the pager must
// not hit the empty text.
func TestSession(t *testing.T) {
	load := func(session string) (*Pack, error) {
		return Load(fstest.MapFS{
			"pack.yaml":            {Data: []byte(read(t, "pack.yaml") + session)},
			"import/10_rules.yaml": {Data: []byte(read(t, "import/10_rules.yaml"))},
		})
	}
	const (
		prompt = "\nsession:\n  prompt: '^(?P<host>\\S+)#'\n"
		show   = "  show_config: ['^show running-config$']\n"
	)
	p, err := load("")
	if err != nil || p.Session != nil {
		t.Fatalf("no session: err %v, session %v", err, p.Session)
	}
	p, err = load(prompt + show + "  pager: ['--More--']\n")
	if err != nil {
		t.Fatal(err)
	}
	if s := p.Session; s == nil || s.Prompt == nil || len(s.ShowConfig) != 1 || len(s.Pager) != 1 {
		t.Errorf("session = %+v", p.Session)
	}
	// The pager is optional.
	if p, err = load(prompt + show); err != nil || p.Session == nil {
		t.Errorf("without pager: err %v", err)
	}
	for _, c := range []struct{ session, want string }{
		{"\nsession:\n" + show, "prompt がありません"},
		{prompt, "show_config がありません"},
		{prompt + show + "  pager: ['^\\s*']\n", "空の文字に当たります"},
		{"\nsession:\n  prompt: '^(\\S+#'\n" + show, "session.prompt"},
	} {
		if _, err := load(c.session); err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("err = %v, want an error with %q", err, c.want)
		}
	}
}
