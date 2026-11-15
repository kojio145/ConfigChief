package logscan

import (
	"regexp"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/termlog"
)

// A log is split into its commands; what the device wrote back goes to the
// command before it, and the first message that matches gives the level.
func TestScan(t *testing.T) {
	s := &termlog.Session{Prompt: regexp.MustCompile(`^(?P<host>[\w.\-]+)(\((?P<mode>[^)]+)\))?[>#]`)}
	messages := []Message{
		{ID: "refused", Match: regexp.MustCompile(`^% Sample refusal`), Level: Rejected},
		{ID: "any", Match: regexp.MustCompile(`^% `), Level: Warning},
	}
	text := strings.Join([]string{
		"rt-a01>enable",
		"rt-a01#configure terminal",
		"Enter configuration commands, one per line.",
		"rt-a01(config)#hostname rt-b02",
		"rt-b02(config)#interface GigabitEthernet0/0",
		"rt-b02(config-if)# ip address 192.0.2.1 255.255.255.0 ",
		"rt-b02(config-if)#ip sample 1",
		"                     ^",
		"% Sample refusal at the marker.",
		"rt-b02(config-if)#no shutdown",
		"% Sample notice: the link comes up",
		"rt-b02(config-if)#!",
		"rt-b02(config-if)#end",
		"rt-b02#",
	}, "\r\n")
	log := Scan(s, messages, text)
	if log.Lines != 14 || len(log.Commands) != 10 {
		t.Fatalf("%d lines, %d commands", log.Lines, len(log.Commands))
	}
	if got := strings.Join(log.Hosts(), ","); got != "rt-a01,rt-b02" {
		t.Errorf("hosts: %s", got)
	}
	c := log.Commands[5]
	if c.Line != 7 || c.Host != "rt-b02" || c.Mode != "config-if" || c.Text != "ip sample 1" || !c.Config() || len(c.Replies) != 2 {
		t.Fatalf("the refused command: %+v", c)
	}
	if r := c.Told(Rejected); r == nil || r.ID != "refused" || r.Line != 9 || c.Replies[0].Level != "" || c.Told(Warning) != nil {
		t.Errorf("the replies: %+v", c.Replies)
	}
	if r := log.Commands[6].Told(Warning); r == nil || r.ID != "any" {
		t.Errorf("the notice: %+v", log.Commands[6])
	}
	// The blanks around what was typed are gone; a command in the EXEC mode
	// is no command of the configuration.
	if log.Commands[4].Text != "ip address 192.0.2.1 255.255.255.0" || log.Commands[1].Config() || len(log.Commands[1].Replies) != 1 {
		t.Errorf("commands: %+v / %+v", log.Commands[4], log.Commands[1])
	}
	// Without a session there is nothing to split.
	if l := Scan(nil, messages, text); len(l.Commands) != 0 {
		t.Errorf("no session: %d commands", len(l.Commands))
	}
}
