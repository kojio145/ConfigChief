package termlog

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/conftree"
)

// The logs live in testdata and in the tests of the bundled packs (not in
// this file) so that the check of public files sees their values.
const (
	ciscoLog = "../packs/cisco_ios/tests/import/log/input.txt"
	necLog   = "../packs/nec_ix/tests/import/log/input.txt"
)

func read(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func session(pager string) *Session {
	return &Session{
		Prompt: regexp.MustCompile(`^(?P<host>[\w.\-]+)(\([^)]+\))?[>#]`),
		ShowConfig: []*regexp.Regexp{
			regexp.MustCompile(`^show running-config$`),
			regexp.MustCompile(`^show startup-config$`),
		},
		Pager: []*regexp.Regexp{regexp.MustCompile(pager)},
	}
}

func TestClean(t *testing.T) {
	for _, c := range []struct{ in, want string }{
		{"abc", "abc"},
		{"  two blanks kept  ", "  two blanks kept  "}, // no control character: as it is
		{"a\tb", "a\tb"},
		{"abcd\b\bxy", "abxy"},
		{"\b\babc", "abc"},
		{"mark\b\b\b\b    \b\b\b\bline of text", "line of text"},
		{"mark\b\b\b\b    \b\b\b\b!", "!"},
		{"abc\x1b[K", "abc"},
		{"abcdef\b\b\b\x1b[K", "abc"},
		{"abcdef\b\b\b\x1b[0Kx", "abcx"},
		{"abcdef\b\b\b\x1b[1Kx", "   xef"}, // up to the cursor, the cursor's place too
		{"mark\x1b[2K\x1b[Gline", "line"},
		{"mark\x1b[2K\x1b[1Gli", "li"},
		{"abcdef\x1b[4Dxy", "abxyef"},
		{"abcdef\x1b[99Dxy", "xycdef"},
		{"ab\x1b[3Cx", "ab   x"},
		{"abcdef\x1b[3Gx", "abxdef"},
		{"ab\x1b[5000Cx", "abx"}, // a move beyond any screen is not followed
		{"\x1b[1mbold\x1b[0m", "bold"},
		{"\x1b[?25lhidden", "hidden"},
		{"\x1b7saved\x1b8", "saved"},
		{"bell\x07 null\x00 del\x7f", "bell null del"},
		{"cut \x1b", "cut"},
		{"cut \x1b[12", "cut"},
	} {
		if got := Clean(c.in); got != c.want {
			t.Errorf("Clean(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// The configuration of the sample logs: the same in every one of them.
func config(t *testing.T) []string {
	t.Helper()
	return conftree.Split(read(t, "testdata/plain.txt"))
}

// section gives the lines that were cut out, without the blank line the
// device sent before the next prompt.
func section(t *testing.T, c *Cut) []string {
	t.Helper()
	sec := c.Section()
	if n := len(sec); n > 0 && sec[n-1] == "" {
		sec = sec[:n-1]
	}
	return sec
}

func TestCutPlainConfig(t *testing.T) {
	text := read(t, "testdata/plain.txt")
	c := session(`^\s*--More--`).Cut(text)
	if c.Log || c.Prompts != 0 || len(c.Commands) != 0 || c.Command() != 0 {
		t.Errorf("a plain configuration was taken for a log: %+v", c)
	}
	if !reflect.DeepEqual(c.Config, config(t)) || !reflect.DeepEqual(c.Section(), config(t)) {
		t.Error("the lines of a plain configuration changed")
	}
	for i, s := range c.Skip {
		if s != "" {
			t.Errorf("line %d is left out: %s", i+1, s)
		}
	}
}

func TestCutNoSession(t *testing.T) {
	text := read(t, ciscoLog)
	var s *Session
	c := s.Cut(text)
	if c.Log || !reflect.DeepEqual(c.Config, conftree.Split(text)) {
		t.Error("without a session the text must stay as it is")
	}
	if c = (&Session{}).Cut(text); c.Log {
		t.Error("a session without a prompt cut the text")
	}
}

// The marks of the pager and the blanks that wiped them are in the lines (a
// terminal that does not record control characters).
func TestCutMarksInLines(t *testing.T) {
	c := session(`^\s*--More--`).Cut(read(t, ciscoLog))
	if !c.Log || c.Command() != 6 || c.First != 7 || c.Last != 47 {
		t.Fatalf("log %v, command %d, lines %d-%d", c.Log, c.Command(), c.First, c.Last)
	}
	if got, want := section(t, c), config(t); !reflect.DeepEqual(got, want) {
		t.Errorf("the configuration is not the one that was shown:\n%s\nwant:\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	if want := []int{25, 34, 41}; !reflect.DeepEqual(c.Loose, want) {
		t.Errorf("loose = %v, want %v", c.Loose, want)
	}
	// What a person sees is the line without the mark.
	if got := c.Shown[24]; got != " ip address 203.0.113.5 255.255.255.252" {
		t.Errorf("shown = %q", got)
	}
	for i, s := range c.Skip {
		want := ""
		if i+1 < c.First || i+1 > c.Last {
			want = ReasonOutside
		}
		if s != want {
			t.Errorf("line %d: skip %q, want %q", i+1, s, want)
		}
		if s != "" && c.Config[i] != "" {
			t.Errorf("line %d is left out but has text to parse: %q", i+1, c.Config[i])
		}
	}
	if c.Shown[5] != "rt-b02#show running-config" {
		t.Errorf("the line of the command is shown as %q", c.Shown[5])
	}
}

// The log has the control characters: what the screen showed is rebuilt, and
// no line needs a guess. Of two commands the last one counts.
func TestCutBackspaces(t *testing.T) {
	c := session(`^\s*--More--`).Cut(read(t, "testdata/backspace.txt"))
	if !c.Log {
		t.Fatal("not taken for a log")
	}
	if want := []int{6, 16}; !reflect.DeepEqual(c.Commands, want) || c.Command() != 16 {
		t.Errorf("commands = %v (used: %d), want %v", c.Commands, c.Command(), want)
	}
	if got, want := section(t, c), config(t); !reflect.DeepEqual(got, want) {
		t.Errorf("the configuration is not the one that was shown:\n%s", strings.Join(got, "\n"))
	}
	if len(c.Loose) != 0 {
		t.Errorf("loose = %v, want none", c.Loose)
	}
	// The command was typed with a slip and two backspaces.
	if got := c.Shown[15]; got != "rt-b02#show running-config" {
		t.Errorf("the line of the command is shown as %q", got)
	}
	// The output of the first command is outside.
	if c.Skip[8] != ReasonOutside || c.Config[8] != "" {
		t.Errorf("line 9: skip %q, config %q", c.Skip[8], c.Config[8])
	}
}

func TestCutEraseSequence(t *testing.T) {
	c := session(`^\s*--More--`).Cut(read(t, "testdata/erase.txt"))
	if !c.Log || c.Command() != 6 {
		t.Fatalf("log %v, command %d", c.Log, c.Command())
	}
	if got, want := section(t, c), config(t); !reflect.DeepEqual(got, want) {
		t.Errorf("the configuration is not the one that was shown:\n%s", strings.Join(got, "\n"))
	}
	if len(c.Loose) != 0 {
		t.Errorf("loose = %v, want none", c.Loose)
	}
}

// The marks stand on lines of their own: these lines are left out, the others
// are not touched.
func TestCutMarksAlone(t *testing.T) {
	text := read(t, necLog)
	c := session(`(?i)^\s*--more--`).Cut(text)
	if !c.Log || c.Command() != 2 {
		t.Fatalf("log %v, command %d", c.Log, c.Command())
	}
	lines := conftree.Split(text)
	marks := 0
	for i := c.First - 1; i < c.Last; i++ {
		if strings.TrimSpace(lines[i]) == "--More--" {
			marks++
			if c.Skip[i] != ReasonPager || c.Config[i] != "" {
				t.Errorf("line %d: skip %q, config %q", i+1, c.Skip[i], c.Config[i])
			}
			continue
		}
		if c.Skip[i] != "" || c.Config[i] != lines[i] {
			t.Errorf("line %d: skip %q, config %q, want the line as it is", i+1, c.Skip[i], c.Config[i])
		}
	}
	if marks != 2 {
		t.Errorf("the sample has %d marks, want 2", marks)
	}
	if len(c.Loose) != 0 {
		t.Errorf("loose = %v, want none", c.Loose)
	}
	// The prompt of the configuration mode ends the output.
	if c.Last+1 > len(lines) || !strings.HasPrefix(lines[c.Last], "ix-b02(config)# exit") {
		t.Errorf("the output ends at line %d", c.Last)
	}
}

// A line of the output that looks like the prompt of another device does not
// end the configuration.
func TestCutOtherHost(t *testing.T) {
	text := read(t, "testdata/otherhost.txt")
	c := session(`^\s*--More--`).Cut(text)
	lines := conftree.Split(text)
	if !c.Log || c.Last != len(lines)-1 {
		t.Fatalf("log %v, lines %d-%d of %d", c.Log, c.First, c.Last, len(lines))
	}
	found := false
	for _, l := range c.Section() {
		found = found || strings.HasPrefix(l, "sample-notice#")
	}
	if !found {
		t.Error("the line that looks like a prompt is not in the configuration")
	}
}

func TestCutNoCommand(t *testing.T) {
	text := read(t, "testdata/nocommand.txt")
	c := session(`^\s*--More--`).Cut(text)
	if c.Log || c.Prompts < 2 {
		t.Errorf("log %v, prompts %d", c.Log, c.Prompts)
	}
	if !reflect.DeepEqual(c.Config, conftree.Split(text)) {
		t.Error("the lines changed")
	}
}

// The log ends with the command.
func TestCutNothingAfterCommand(t *testing.T) {
	c := session(`^\s*--More--`).Cut(read(t, "testdata/cutoff.txt"))
	if !c.Log || c.Command() != 6 || c.First <= c.Last || c.Section() != nil {
		t.Errorf("log %v, command %d, lines %d-%d, section %v", c.Log, c.Command(), c.First, c.Last, c.Section())
	}
	for i, s := range c.Skip {
		if s != ReasonOutside {
			t.Errorf("line %d: skip %q", i+1, s)
		}
	}
}

func TestUnmark(t *testing.T) {
	s := session(`--More--`)
	for _, c := range []struct {
		in, before, after string
		hit               bool
	}{
		{"no mark here", "", "no mark here", false},
		{" --More-- ", " ", " ", true},
		{" --More--     text", " ", "     text", true},
		{"--More----More--  text", "", "  text", true},
	} {
		before, after, hit := s.unmark(c.in)
		if before != c.before || after != c.after || hit != c.hit {
			t.Errorf("unmark(%q) = %q, %q, %v", c.in, before, after, hit)
		}
	}
}
