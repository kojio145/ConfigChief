package logscan

import (
	"regexp"
	"strings"

	"github.com/kojio145/configchief/internal/conftree"
	"github.com/kojio145/configchief/internal/termlog"
)

// The levels of a message of a device (appendix B 4.4).
const (
	Rejected = "rejected" // the device did not take the command
	Warning  = "warning"  // the device took the command and said something
	Unknown  = "unknown"  // the device said something the pack does not know
)

// Message is one way a device answers a command: a line of errors.yaml.
type Message struct {
	ID    string
	Match *regexp.Regexp
	Level string
}

// Reply is a line a device wrote after a command.
type Reply struct {
	Line  int // in the log
	Text  string
	ID    string // the message it is ("" = none: any other output)
	Level string
}

// Command is a line typed after a prompt, with what the device wrote until
// the next prompt.
type Command struct {
	Line    int // in the log
	Host    string
	Mode    string // what stands in the brackets of the prompt ("" = none)
	Text    string // what was typed, without the blanks around it
	Replies []Reply
}

// Config tells that the command was typed in a mode of the configuration:
// the prompt has brackets.
func (c *Command) Config() bool { return c.Mode != "" }

// Told gives the first message of a level among the replies, or nil.
func (c *Command) Told(level string) *Reply {
	for i := range c.Replies {
		if c.Replies[i].Level == level {
			return &c.Replies[i]
		}
	}
	return nil
}

// Log is a terminal log as a list of commands.
type Log struct {
	Lines    int
	Commands []Command
}

// Hosts lists the host names of the prompts of the configuration modes, in
// the order they first appear.
func (l *Log) Hosts() []string {
	var out []string
	seen := map[string]bool{}
	for _, c := range l.Commands {
		if c.Config() && c.Host != "" && !seen[c.Host] {
			seen[c.Host] = true
			out = append(out, c.Host)
		}
	}
	return out
}

// Scan splits a terminal log into its commands. A line that starts with the
// prompt opens a command; the lines up to the next prompt are what the
// device wrote back, and each is set against the messages, the first that
// matches giving its level. The marks of a pager and the control characters
// of the terminal are taken out first.
func Scan(s *termlog.Session, messages []Message, text string) *Log {
	lines := conftree.Split(text)
	log := &Log{Lines: len(lines)}
	if s == nil || s.Prompt == nil {
		return log
	}
	host, mode := s.Prompt.SubexpIndex("host"), s.Prompt.SubexpIndex("mode")
	var cur *Command
	for i, raw := range lines {
		line := termlog.Clean(raw)
		if m := s.Prompt.FindStringSubmatchIndex(line); m != nil {
			c := Command{Line: i + 1, Text: strings.TrimSpace(line[m[1]:])}
			if host >= 0 && m[2*host] >= 0 {
				c.Host = line[m[2*host]:m[2*host+1]]
			}
			if mode >= 0 && m[2*mode] >= 0 {
				c.Mode = line[m[2*mode]:m[2*mode+1]]
			}
			log.Commands = append(log.Commands, c)
			cur = &log.Commands[len(log.Commands)-1]
			continue
		}
		if cur == nil || strings.TrimSpace(line) == "" {
			continue
		}
		r := Reply{Line: i + 1, Text: strings.TrimRight(line, " \t")}
		for _, msg := range messages {
			if msg.Match.MatchString(strings.TrimSpace(line)) {
				r.ID, r.Level = msg.ID, msg.Level
				break
			}
		}
		cur.Replies = append(cur.Replies, r)
	}
	return log
}
