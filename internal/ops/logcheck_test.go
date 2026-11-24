package ops

import (
	"os"
	"sort"
	"strings"
	"testing"
)

// pasted makes the terminal log of a configuration typed into a device: the
// prompt before each line, and what the device answers to some lines.
func pasted(host, config string, answers map[string]string) string {
	var sb strings.Builder
	sb.WriteString("Router>enable\nRouter#configure terminal\nEnter configuration commands, one per line.  End with CNTL/Z.\n")
	name, mode := "Router", "config"
	for _, line := range strings.Split(strings.TrimRight(config, "\n"), "\n") {
		text := strings.TrimSpace(line)
		// A line without indentation leaves the mode of a block.
		if !strings.HasPrefix(line, " ") && text != "" && text != "!" {
			mode = "config"
		}
		sb.WriteString(name + "(" + mode + ")#" + line + "\n")
		if a, ok := answers[text]; ok {
			sb.WriteString(a)
		}
		switch {
		case strings.HasPrefix(text, "hostname "):
			name = strings.TrimPrefix(text, "hostname ")
		case strings.HasPrefix(text, "interface "):
			mode = "config-if"
		case strings.HasPrefix(text, "router "):
			mode = "config-router"
		case text == "end":
			sb.WriteString(name + "#\n")
		}
	}
	return sb.String()
}

func loadKinds(o *LoadOutcome) []string {
	var out []string
	for _, f := range o.Verdicts.Items {
		out = append(out, strings.TrimSpace(f.RuleID+" "+f.Device+" "+f.Ref.Table+" "+f.Ref.Column))
	}
	sort.Strings(out)
	return out
}

// The log of the work: what the device did not take, what it only remarked
// on, an answer the pack does not know; a log that is another version of the
// configuration, one that stops half way, one typed into another device.
func TestLoadCheck(t *testing.T) {
	e := exportCase(t)
	if _, err := e.LoadCheck(at); err == nil || !strings.Contains(err.Error(), "input/logs にファイルがありません") {
		t.Fatalf("no log: %v", err)
	}
	gen, err := e.Generate(GenOptions{Dry: true}, at)
	if err != nil {
		t.Fatal(err)
	}
	texts := map[string]string{}
	for _, d := range gen.Devices {
		texts[d.Host] = d.Output.Text
	}
	put := func(name, text string) {
		t.Helper()
		if err := os.MkdirAll(e.Proj.Path("input/logs"), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := writeTestFile(e.Proj.Path("input/logs/"+name), []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Every line was taken: nothing to tell.
	put("rt-s01.txt", pasted("rt-s01", texts["rt-s01"], nil))
	o, err := e.LoadCheck(at)
	if err != nil {
		t.Fatal(err)
	}
	if got := loadKinds(o); len(got) != 0 || len(o.Logs) != 1 || o.Logs[0].State != LogCompared || o.Logs[0].Device != "rt-s01" || o.Logs[0].Commands == 0 || o.Logs[0].Commands != o.Logs[0].Matched {
		t.Fatalf("the clean log: %v, %+v", got, o.Logs)
	}

	// The device answers three lines.
	put("rt-s01.txt", pasted("rt-s01", texts["rt-s01"], map[string]string{
		"vrrp 1 priority 110":                     "                      ^\n% Invalid input detected at '^' marker.\n\n",
		"ip nat inside":                           "% Sample answer the pack has never seen\n",
		"ip address 203.0.113.49 255.255.255.252": "% Warning: use /31 mask on non point-to-point interface cautiously\n",
		"no shutdown":                             "*Mar  1 00:00:46: %LINK-3-UPDOWN: Interface GigabitEthernet0/0, changed state to up\n",
	}))
	// Another version (a line that is not in the configuration, the routes
	// before the interfaces would be out of order) that stops half way, and
	// whose prompt never takes the name of the device.
	w := texts["rt-w01"]
	half := w[:strings.Index(w, "interface Tunnel1")]
	half = strings.Replace(half, "hostname rt-w01\n", "ip multicast-routing\n", 1)
	put("rt-w01.txt", pasted("rt-w01", half, nil))
	// No command of a configuration mode.
	put("status.txt", "rt-s01#show ip route\nGateway of last resort is not set\nrt-s01#exit\n")
	later := at.Add(1e9)
	if o, err = e.LoadCheck(later); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"L-701 rt-s01 fhrp priority",
		"L-701 rt-s01 interfaces nat_role",
		"L-702 rt-w01",
		"L-703 rt-w01",
		"L-704 rt-s01 interfaces ipv4",
		"L-705 rt-w01",
	}
	if got := loadKinds(o); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("findings:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
	if len(o.Rejected) != 2 || len(o.Warned) != 1 {
		t.Fatalf("rejected %d, warned %d", len(o.Rejected), len(o.Warned))
	}
	var caret, unknown *LoadLine
	for i := range o.Rejected {
		if strings.Contains(o.Rejected[i].Finding.Message, "受け付けませんでした") {
			caret = &o.Rejected[i]
		} else {
			unknown = &o.Rejected[i]
		}
	}
	if caret == nil || unknown == nil {
		t.Fatalf("the two kinds of L-701: %+v", o.Rejected)
	}
	// The command, the mode, the answer with the caret line, and the line of
	// the generated configuration.
	if caret.Command != "vrrp 1 priority 110" || caret.Mode != "config-if" || len(caret.Replies) != 2 || caret.Replies[0] != "^" || caret.ConfigLine == 0 ||
		strings.TrimSpace(strings.Split(texts["rt-s01"], "\n")[caret.ConfigLine-1]) != "vrrp 1 priority 110" {
		t.Errorf("the refused line: %+v", caret)
	}
	if !strings.Contains(unknown.Finding.Message, "知りません") || unknown.Command != "ip nat inside" {
		t.Errorf("the unknown answer: %+v", unknown)
	}
	states := map[string]string{}
	for _, l := range o.Logs {
		states[l.Path] = l.State + " " + l.Device + " " + l.Why
	}
	if !strings.HasPrefix(states["input/logs/status.txt"], LogSkipped+"  設定モードのコマンドがありません") ||
		!strings.HasPrefix(states["input/logs/rt-w01.txt"], LogCompared+" rt-w01") || !strings.HasPrefix(states["input/logs/rt-s01.txt"], LogCompared+" rt-s01") {
		t.Errorf("logs: %v", states)
	}
	// Three errors (two refused lines, the other device) and two warnings.
	if c := o.Verdicts.Count(); c.Errors != 3 || c.Warnings != 2 || c.Info != 1 || o.Left() != 5 {
		t.Errorf("count %+v, left %d", c, o.Left())
	}
	html := read(t, e, o.Report.HTML)
	for _, s := range []string{"投入ログのチェック", "Invalid input detected", "ip multicast-routing", "途中で切れた疑い", "機器の取り違えの疑い"} {
		if !strings.Contains(html, s) {
			t.Errorf("the report lacks %q", s)
		}
	}

	// The answer nobody knew is looked at and approved.
	if _, err := e.Approve(ApproveRequest{Finding: unknown.Finding.Key, Reason: "the command was taken", By: "sample-user"}, later); err != nil {
		t.Fatal(err)
	}
	if o, err = e.LoadCheck(later.Add(1e9)); err != nil {
		t.Fatal(err)
	}
	if c := o.Verdicts.Count(); c.Errors != 2 || c.Approved != 1 {
		t.Errorf("after the approval: %+v", c)
	}
}
