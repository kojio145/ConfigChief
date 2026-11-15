package termlog

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/kojio145/configchief/internal/conftree"
)

// Why a line of a terminal log is not read as configuration.
const (
	ReasonOutside = "設定の外（端末ログ）"
	ReasonPager   = "ページ送りの跡"
)

// Session is how a pack tells the lines of a terminal log apart.
type Session struct {
	Prompt     *regexp.Regexp   // the prompt at the start of a line; may have a group "host"
	ShowConfig []*regexp.Regexp // tried on what was typed after the prompt
	Pager      []*regexp.Regexp // the mark a pager leaves in a line
}

// Cut is a text split into its lines, and, if it is a terminal log, the part
// that is the configuration. Every slice has one entry per line of the text:
// entry i is line i+1.
type Cut struct {
	Log    bool     // a command that shows the configuration was found
	Config []string // the lines to parse; a line that is not configuration is ""
	Shown  []string // the lines to show to a person
	Skip   []string // why a line is not configuration ("" for the others)

	Commands []int // the lines of the commands that show the configuration
	// The lines of the configuration (First > Last: there are none). Only
	// in a log.
	First, Last int
	// The lines where configuration followed a pager mark: their
	// indentation was set from the other lines of the log.
	Loose   []int
	Prompts int // the lines that start with the prompt
}

// Command gives the line of the command whose output was used (0 if the
// text is not a log).
func (c *Cut) Command() int {
	if !c.Log {
		return 0
	}
	return c.Commands[len(c.Commands)-1]
}

// Section gives the lines of the configuration: in a log the range that was
// cut out, or else every line.
func (c *Cut) Section() []string {
	if !c.Log {
		return c.Config
	}
	if c.First > c.Last {
		return nil
	}
	return c.Config[c.First-1 : c.Last]
}

// Cut splits a text and looks for the configuration in it. Without a session
// (s == nil), or in a text without a command that shows the configuration,
// every line is configuration and nothing is changed.
func (s *Session) Cut(text string) *Cut {
	lines := conftree.Split(text)
	c := &Cut{Config: lines, Shown: lines, Skip: make([]string, len(lines))}
	if s == nil || s.Prompt == nil {
		return c
	}

	clean := make([]string, len(lines))
	prompt := make([]bool, len(lines))
	host := make([]string, len(lines))
	hostAt := s.Prompt.SubexpIndex("host")
	for i, l := range lines {
		clean[i] = Clean(l)
		m := s.Prompt.FindStringSubmatchIndex(clean[i])
		if m == nil || m[0] != 0 {
			continue
		}
		prompt[i] = true
		c.Prompts++
		if hostAt >= 0 && m[2*hostAt] >= 0 {
			host[i] = clean[i][m[2*hostAt]:m[2*hostAt+1]]
		}
		typed := strings.TrimSpace(clean[i][m[1]:])
		for _, re := range s.ShowConfig {
			if re.MatchString(typed) {
				c.Commands = append(c.Commands, i+1)
				break
			}
		}
	}
	if len(c.Commands) == 0 {
		return c
	}

	// The output of the last command, up to the next prompt of the device.
	c.Log = true
	start := c.Commands[len(c.Commands)-1] - 1
	end := len(lines)
	for i := start + 1; i < len(lines); i++ {
		if prompt[i] && (host[i] == host[start] || host[i] == "" || host[start] == "") {
			end = i
			break
		}
	}
	c.First, c.Last = start+2, end
	c.Config = make([]string, len(lines))
	c.Shown = clean
	for i := range lines {
		if i <= start || i >= end {
			c.Skip[i] = ReasonOutside
		}
	}

	// Pager marks. Where configuration follows a mark, the blanks between
	// them are what the device sent to wipe the mark plus the indentation
	// of the line: the shortest run of the log is taken to be the wipe.
	type marked struct {
		line   int
		before string
		after  string // what follows the mark, blanks first
	}
	var loose []marked
	wipe := -1
	for i := start + 1; i < end; i++ {
		before, after, hit := s.unmark(clean[i])
		if !hit {
			c.Config[i] = clean[i]
			continue
		}
		rest := strings.TrimLeft(after, " \t")
		if strings.TrimSpace(before) == "" && rest == "" {
			c.Skip[i] = ReasonPager
			continue
		}
		loose = append(loose, marked{i, before, after})
		if rest != "" {
			if n := len(after) - len(rest); wipe < 0 || n < wipe {
				wipe = n
			}
		}
	}
	for _, m := range loose {
		after := m.after
		if rest := strings.TrimLeft(after, " \t"); rest == "" {
			after = ""
		} else if wipe > 0 {
			after = after[wipe:]
		}
		c.Config[m.line] = m.before + after
		c.Shown[m.line] = c.Config[m.line]
		c.Loose = append(c.Loose, m.line+1)
	}
	return c
}

// unmark takes the pager marks out of a line. It gives what stands before the
// first mark and what follows the last one.
func (s *Session) unmark(line string) (before, after string, hit bool) {
	after = line
	// A line holds one mark as a rule; the bound keeps a pattern that hits
	// its own leftovers from going on forever.
	for range 8 {
		found := false
		for _, re := range s.Pager {
			loc := re.FindStringIndex(after)
			if loc == nil || loc[0] == loc[1] {
				continue
			}
			if !hit {
				before = after[:loc[0]]
			}
			after = after[loc[1]:]
			hit, found = true, true
			break
		}
		if !found {
			break
		}
	}
	return before, after, hit
}

// Clean gives a line of a terminal log as the screen showed it: a backspace
// steps back and what follows writes over the old characters, "ESC [ K"
// erases, other control characters (the tab is kept) and escape sequences are
// dropped. A line without control characters comes back as it is.
func Clean(line string) string {
	if !strings.ContainsFunc(line, func(r rune) bool { return isControl(r) }) {
		return line
	}
	rs := []rune(line)
	var buf []rune
	pos := 0
	for i := 0; i < len(rs); i++ {
		r := rs[i]
		switch {
		case r == '\b':
			if pos > 0 {
				pos--
			}
		case r == 0x1b:
			if i+1 >= len(rs) {
				break
			}
			if rs[i+1] != '[' {
				i++ // an escape sequence of two characters
				break
			}
			// A control sequence: parameters, intermediates, one final.
			j := i + 2
			for j < len(rs) && rs[j] >= 0x30 && rs[j] <= 0x3f {
				j++
			}
			param := string(rs[i+2 : j])
			for j < len(rs) && rs[j] >= 0x20 && rs[j] <= 0x2f {
				j++
			}
			if j < len(rs) {
				buf, pos = control(buf, pos, param, rs[j])
			}
			i = j
		case isControl(r):
			// dropped
		default:
			if pos < len(buf) {
				buf[pos] = r
			} else {
				buf = append(buf, r)
			}
			pos++
		}
	}
	return strings.TrimRight(string(buf), " ")
}

// control does the control sequences that change what a line shows: erase in
// line (K) and the moves of the cursor within the line (C forward, D back, G
// to a column). The others do nothing.
func control(buf []rune, pos int, param string, final rune) ([]rune, int) {
	n := 1
	if v, err := strconv.Atoi(param); err == nil && v > 0 {
		n = v
	}
	switch final {
	case 'K':
		buf = erase(buf, pos, param)
	case 'C':
		pos += n
	case 'D':
		pos = max(pos-n, 0)
	case 'G':
		pos = n - 1
	}
	// A cursor beyond the end of the line: what is written next stands
	// after blanks.
	for final != 'K' && len(buf) < pos && pos <= maxColumn {
		buf = append(buf, ' ')
	}
	return buf, min(pos, len(buf))
}

// maxColumn bounds how far a control sequence may move the cursor to the
// right.
const maxColumn = 1000

// erase does "erase in line": to the end of the line (no parameter or 0), up
// to the cursor (1), or all of it (2).
func erase(buf []rune, pos int, param string) []rune {
	switch param {
	case "", "0":
		if pos < len(buf) {
			buf = buf[:pos]
		}
	case "1":
		for k := 0; k <= pos && k < len(buf); k++ {
			buf[k] = ' '
		}
	case "2":
		for k := range buf {
			buf[k] = ' '
		}
	}
	return buf
}

func isControl(r rune) bool {
	return (r < 0x20 && r != '\t') || r == 0x7f
}
