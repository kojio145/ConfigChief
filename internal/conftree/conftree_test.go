package conftree

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

// dump writes the tree as "line:depth:text" rows.
func dump(t *Tree) string {
	var sb strings.Builder
	Walk(t.Root, func(n *Node, path []*Node) {
		mark := ""
		if n.Body {
			mark = "|"
		}
		sb.WriteString(strings.Repeat(". ", len(path)) + mark + strings.TrimSpace(n.Text) + "\n")
	})
	return sb.String()
}

var banner = Block{ID: "banner", Start: regexp.MustCompile(`^banner (?P<kind>\S+) (?P<delim>\^C|.)`), EndBy: "delim"}

func TestIndent(t *testing.T) {
	src := strings.Join([]string{
		"!",
		"hostname rt-a01",
		"!",
		"interface GigabitEthernet0/0/0",
		" description to rt-b01",
		" ip address 192.0.2.1 255.255.255.252",
		"!",
		"router bgp 64496",
		" bgp log-neighbor-changes",
		" !",
		" address-family ipv4",
		"  network 198.51.100.0",
		"   deeper",
		" exit-address-family",
		"",
		"end",
	}, "\r\n") + "\r\n"
	tr, err := Parse("indent", src, Options{CommentPrefixes: []string{"!"}})
	if err != nil {
		t.Fatal(err)
	}
	want := `hostname rt-a01
interface GigabitEthernet0/0/0
. description to rt-b01
. ip address 192.0.2.1 255.255.255.252
router bgp 64496
. bgp log-neighbor-changes
. address-family ipv4
. . network 198.51.100.0
. . . deeper
. exit-address-family
end
`
	if got := dump(tr); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	if len(tr.Lines) != 16 {
		t.Errorf("lines = %d", len(tr.Lines))
	}
	var sep []int
	for i, in := range tr.InTree {
		if !in {
			sep = append(sep, i+1)
		}
	}
	if got := fmt.Sprint(sep); got != "[1 3 7 10 15]" {
		t.Errorf("separators = %s", got)
	}
	n := tr.Root.Children[1]
	if n.Line != 4 || n.Children[1].Line != 6 || n.Children[1].Text != " ip address 192.0.2.1 255.255.255.252" {
		t.Errorf("line numbers or text are off: %+v", n.Children[1])
	}
}

// Two-space indentation, a tab, and a child that comes back to a shallower
// but still indented level.
func TestIndentWidths(t *testing.T) {
	src := "a\n    b\n        c\n  d\n\te\nf\n"
	tr, err := Parse("indent", src, Options{})
	if err != nil {
		t.Fatal(err)
	}
	want := "a\n. b\n. . c\n. d\n. . e\nf\n"
	if got := dump(tr); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestFlat(t *testing.T) {
	tr, err := Parse("flat", "set a\n  set b\n# note\nset c\n", Options{CommentPrefixes: []string{"#"}})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := dump(tr), "set a\nset b\nset c\n"; got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
}

func TestBlock(t *testing.T) {
	src := strings.Join([]string{
		"hostname rt-a01",
		"banner motd ^C",
		"  Sample banner",
		"! not a comment here",
		"",
		"interface looks like a command",
		"^C",
		"banner login #one line#",
		"interface Loopback0",
		" ip address 203.0.113.1 255.255.255.255",
	}, "\n")
	tr, err := Parse("indent", src, Options{CommentPrefixes: []string{"!"}, Blocks: []Block{banner}})
	if err != nil {
		t.Fatal(err)
	}
	want := `hostname rt-a01
banner motd ^C
. |Sample banner
. |! not a comment here
. |
. |interface looks like a command
. |^C
banner login #one line#
interface Loopback0
. ip address 203.0.113.1 255.255.255.255
`
	if got := dump(tr); got != want {
		t.Errorf("got\n%s\nwant\n%s", got, want)
	}
	if len(tr.Warnings) != 0 {
		t.Errorf("warnings: %v", tr.Warnings)
	}
	if b := tr.Root.Children[1]; b.Block != "banner" || b.Children[0].Text != "  Sample banner" {
		t.Errorf("block = %+v", b)
	}
	for i, in := range tr.InTree {
		if !in {
			t.Errorf("line %d is not in the tree", i+1)
		}
	}
	if got, want := tr.Root.Children[1].BodyText, "  Sample banner\n! not a comment here\n\ninterface looks like a command"; got != want {
		t.Errorf("body = %q", got)
	}
	if got := tr.Root.Children[2].BodyText; got != "one line" {
		t.Errorf("one-line body = %q", got)
	}
}

func TestBlockNotClosed(t *testing.T) {
	tr, err := Parse("indent", "banner motd ^C\ntext\nhostname rt-a01\n", Options{Blocks: []Block{banner}})
	if err != nil {
		t.Fatal(err)
	}
	if len(tr.Warnings) != 1 || tr.Warnings[0].Line != 1 {
		t.Errorf("warnings = %+v", tr.Warnings)
	}
	if len(tr.Root.Children) != 1 || len(tr.Root.Children[0].Children) != 2 {
		t.Errorf("tree:\n%s", dump(tr))
	}
}

func TestUnknownKind(t *testing.T) {
	if _, err := Parse("braces", "a {\n}\n", Options{}); err == nil {
		t.Error("no error")
	}
}

func TestNormalize(t *testing.T) {
	if got := Normalize("  ip   address\t192.0.2.1  255.255.255.0  "); got != "ip address 192.0.2.1 255.255.255.0" {
		t.Errorf("got %q", got)
	}
}

// A vendor that closes each part with "#" and indents the single lines of the
// device by one space: the line after a mark is a child of the root.
func TestCommentCloses(t *testing.T) {
	src := strings.Join([]string{
		"#",
		" sysname sw-a01",
		"#",
		"vlan 10",
		" name sample-office",
		"#",
		"interface GigabitEthernet1/0/1",
		" port access vlan 10",
		"#",
		" ip route-static 0.0.0.0 0 192.0.2.1",
		" ip route-static 198.51.100.0 24 192.0.2.2",
		"#",
		"return",
	}, "\n")
	want := "sysname sw-a01\nvlan 10\n. name sample-office\ninterface GigabitEthernet1/0/1\n. port access vlan 10\n" +
		"ip route-static 0.0.0.0 0 192.0.2.1\nip route-static 198.51.100.0 24 192.0.2.2\nreturn\n"
	tree, err := Parse("indent", src, Options{CommentPrefixes: []string{"#"}, CommentCloses: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := dump(tree); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	// Without the option the indented lines hang under the interface.
	tree, _ = Parse("indent", src, Options{CommentPrefixes: []string{"#"}})
	if got := dump(tree); !strings.Contains(got, ". ip route-static 0.0.0.0 0 192.0.2.1\n") {
		t.Errorf("without the option:\n%s", got)
	}
}

// An indented mark stands between the parts of one block (" #" before
// "address-family" under "bgp"): the block stays open, and what stood as
// deep as the mark is closed.
func TestCommentClosesIndented(t *testing.T) {
	src := strings.Join([]string{
		"bgp 64496",
		" peer 192.0.2.2 as-number 64497",
		" #",
		" address-family ipv4 unicast",
		"  peer 192.0.2.2 enable",
		" #",
		" address-family ipv6 unicast",
		"  network 2001:db8:: 32",
		"#",
		" ip route-static 0.0.0.0 0 192.0.2.1",
	}, "\n")
	want := "bgp 64496\n. peer 192.0.2.2 as-number 64497\n. address-family ipv4 unicast\n. . peer 192.0.2.2 enable\n" +
		". address-family ipv6 unicast\n. . network 2001:db8:: 32\nip route-static 0.0.0.0 0 192.0.2.1\n"
	tree, err := Parse("indent", src, Options{CommentPrefixes: []string{"#"}, CommentCloses: true})
	if err != nil {
		t.Fatal(err)
	}
	if got := dump(tree); got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
}
