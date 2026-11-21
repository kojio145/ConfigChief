package packs

import (
	"testing"

	"github.com/kojio145/configchief/internal/logscan"
	"github.com/kojio145/configchief/internal/pack"
)

// A log of commands fed to a QX-type switch (made-up values): the lines
// typed after a prompt in square brackets are commands of the
// configuration, and the answers of the manual's table of errors are
// rejections. A log line of the device is no answer.
func TestQXLoadLog(t *testing.T) {
	fsys, err := FS("nec_qx")
	if err != nil {
		t.Fatal(err)
	}
	p, err := pack.Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	text := "<sw-q05>system-view\n" +
		"System View: return to User View with Ctrl+Z.\n" +
		"[sw-q05]vlan 10\n" +
		"[sw-q05-vlan10]name sample-office\n" +
		"[sw-q05-vlan10]quit\n" +
		"[sw-q05]interface GigabitEthernet1/0/1\n" +
		"[sw-q05-GigabitEthernet1/0/1]port acess vlan 10\n" +
		"                              ^\n" +
		" % Unrecognized command found at '^' position.\n" +
		"[sw-q05-GigabitEthernet1/0/1]port access vlan\n" +
		" % Incomplete command found at '^' position.\n" +
		"[sw-q05-GigabitEthernet1/0/1]port access vlan 10 20\n" +
		" % Too many parameters.\n" +
		"[sw-q05-GigabitEthernet1/0/1]port access vlan 5000\n" +
		" % Wrong parameter found at '^' position.\n" +
		"[sw-q05-GigabitEthernet1/0/1]shutdown\n" +
		"%Jan  5 12:34:56:789 2026 sw-q05 IFNET/3/PHY_UPDOWN: sample log line\n" +
		"[sw-q05-GigabitEthernet1/0/1]undo shutdown\n" +
		" % Sample answer the pack does not know.\n" +
		"[sw-q05-GigabitEthernet1/0/1]return\n" +
		"<sw-q05>display vlan brief\n"
	log := logscan.Scan(p.Session, p.Messages, text)
	type want struct {
		text, id string
		config   bool
	}
	wants := []want{
		{"system-view", "", false},
		{"vlan 10", "", true},
		{"name sample-office", "", true},
		{"quit", "", true},
		{"interface GigabitEthernet1/0/1", "", true},
		{"port acess vlan 10", "unrecognized", true},
		{"port access vlan", "incomplete", true},
		{"port access vlan 10 20", "too_many", true},
		{"port access vlan 5000", "wrong", true},
		{"shutdown", "", true},
		{"undo shutdown", "other", true},
		{"return", "", true},
		{"display vlan brief", "", false},
	}
	if len(log.Commands) != len(wants) {
		t.Fatalf("%d commands, want %d: %+v", len(log.Commands), len(wants), log.Commands)
	}
	for i, w := range wants {
		c := log.Commands[i]
		id := ""
		for _, r := range c.Replies {
			if r.ID != "" {
				id = r.ID
			}
		}
		if c.Text != w.text || id != w.id || c.Config() != w.config {
			t.Errorf("command %d: %q, message %q, config %v; want %+v", i, c.Text, id, c.Config(), w)
		}
	}
}
