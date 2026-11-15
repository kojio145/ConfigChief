package cli

import (
	"strings"
	"testing"
)

// What the program of the window takes for a command, and what for a case
// folder to open.
func TestIsCommand(t *testing.T) {
	for _, args := range [][]string{{"version"}, {"--version"}, {"help"}, {"init", "dir"}, {"diff", "--tree", "a", "b"}, {"pack", "test", "dir"}, {"sites"}} {
		if !IsCommand(args) {
			t.Errorf("%v is not taken for a command", args)
		}
	}
	for _, args := range [][]string{nil, {}, {`C:\cases\sample`}, {`.\import`}, {"./generate"}, {"Import"}, {"--project", "dir"}} {
		if IsCommand(args) {
			t.Errorf("%v is taken for a command", args)
		}
	}
	// Every command the usage names is one of Commands.
	for _, line := range strings.Split(usage, "\n") {
		f := strings.Fields(line)
		if len(f) < 2 || f[0] != "ConfigChief.exe" || strings.HasPrefix(f[1], "[") {
			continue
		}
		if !IsCommand(f[1:]) {
			t.Errorf("the usage names %q, which is not taken for a command", f[1])
		}
	}
	if code, out, _ := do("help"); code != 0 || !strings.Contains(out, "usage:") {
		t.Errorf("help = %d: %s", code, out)
	}
}
