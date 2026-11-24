package ops

import (
	"os"
	"sort"
	"strings"
	"testing"
)

// kinds lists the findings of a check run as "ID device table column".
func verifyKinds(o *VerifyOutcome) []string {
	var out []string
	for _, f := range o.Verdicts.Items {
		out = append(out, strings.TrimSpace(f.RuleID+" "+f.Device+" "+f.Ref.Table+" "+f.Ref.Column))
	}
	sort.Strings(out)
	return out
}

// What a device shows after the work is set beside the book: a row that is
// missing, one too many, a value that differs, a line of an extra
// configuration that is not there, a line nothing explains; a log without a
// configuration and a log of a device that is not of the case.
func TestVerify(t *testing.T) {
	e := exportCase(t)
	if err := writeTestFile(e.Proj.Path("extra/rt-s01.j2"), []byte("ip multicast-routing\nservice sequence-numbers\n!\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := e.Verify(at); err == nil || !strings.Contains(err.Error(), "input/logs にファイルがありません") {
		t.Fatalf("no log: %v", err)
	}
	gen, err := e.Generate(GenOptions{Dry: true}, at)
	if err != nil {
		t.Fatal(err)
	}
	texts := map[string]string{}
	for _, d := range gen.Devices {
		if d.Output == nil || d.Output.Failed {
			t.Fatalf("%s was not generated: %s", d.Host, d.Why)
		}
		texts[d.Host] = d.Output.Text
	}
	logOf := func(host, config string) string {
		return host + ">enable\n" + host + "#show running-config\nBuilding configuration...\n\n" + config + host + "#exit\n"
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

	// The device shows what was generated: nothing to tell.
	put("rt-s01.txt", logOf("rt-s01", texts["rt-s01"]))
	put("rt-w01.txt", texts["rt-w01"]) // the configuration alone, no log around it
	o, err := e.Verify(at)
	if err != nil {
		t.Fatal(err)
	}
	if got := verifyKinds(o); len(got) != 0 || o.Left() != 0 || len(o.Logs) != 2 || o.Logs[0].State != LogCompared || o.Logs[1].State != LogCompared {
		t.Fatalf("the same configuration: %v, logs %+v", got, o.Logs)
	}

	// The device differs from the book.
	changed := texts["rt-s01"]
	for _, r := range [][2]string{
		{" vrrp 1 priority 110\n", " vrrp 1 priority 120\n"},                                    // R-803
		{"ip route 0.0.0.0 0.0.0.0 203.0.113.50 track 1\n", ""},                                 // R-801
		{"service sequence-numbers\n", ""},                                                      // R-804
		{"hostname rt-s01\n", "hostname rt-s01\nservice tcp-keepalives-in\n"},                   // R-805
		{"router ospf 1\n", "ip route 192.0.2.0 255.255.255.128 203.0.113.50\nrouter ospf 1\n"}, // R-802
	} {
		if !strings.Contains(changed, r[0]) {
			t.Fatalf("the generated text has no %q:\n%s", r[0], changed)
		}
		changed = strings.Replace(changed, r[0], r[1], 1)
	}
	put("rt-s01.txt", logOf("rt-s01", changed))
	// A log that shows no configuration, and a device that is not of the case.
	put("rt-w01.txt", "rt-w01#show ip route\nGateway of last resort is not set\nrt-w01#exit\n")
	put("rt-z09.txt", "version 15.2\nhostname rt-z09\n!\ninterface GigabitEthernet0/0\n ip address 192.0.2.9 255.255.255.0\n!\nend\n")
	later := at.Add(1e9)
	if o, err = e.Verify(later); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"R-801 rt-s01 static_routes",
		"R-802 rt-s01 static_routes",
		"R-803 rt-s01 fhrp priority",
		"R-804 rt-s01",
		"R-805 rt-s01",
		"R-806 rt-z09",
	}
	if got := verifyKinds(o); strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("findings:\n  %s\nwant:\n  %s", strings.Join(got, "\n  "), strings.Join(want, "\n  "))
	}
	states := map[string]string{}
	for _, l := range o.Logs {
		states[l.Path] = l.State + " " + l.Why
	}
	if !strings.HasPrefix(states["input/logs/rt-w01.txt"], LogSkipped+" 設定全体の表示") || !strings.HasPrefix(states["input/logs/rt-z09.txt"], LogFailed) || !strings.HasPrefix(states["input/logs/rt-s01.txt"], LogCompared) {
		t.Errorf("logs: %v", states)
	}
	// Four errors and one warning, and the log that fits no device.
	if c := o.Verdicts.Count(); c.Errors != 4 || c.Warnings != 1 || c.Info != 1 || o.Left() != 6 {
		t.Errorf("count %+v, left %d", c, o.Left())
	}
	if len(o.Untold) != 1 || o.Untold[0] != "rt-w01" {
		t.Errorf("untold: %v", o.Untold)
	}
	html := read(t, e, o.Report.HTML)
	for _, s := range []string{"反映確認", "service sequence-numbers", "service tcp-keepalives-in", "設定全体の表示"} {
		if !strings.Contains(html, s) {
			t.Errorf("the report lacks %q", s)
		}
	}

	// A finding is approved by its fingerprint, and counts no more.
	var key string
	for _, f := range o.Verdicts.Items {
		if f.RuleID == "R-802" {
			key = f.Key
		}
	}
	if _, err := e.Approve(ApproveRequest{Finding: key, Reason: "a route of the test bed", By: "sample-user"}, later); err != nil {
		t.Fatal(err)
	}
	if o, err = e.Verify(later.Add(1e9)); err != nil {
		t.Fatal(err)
	}
	if c := o.Verdicts.Count(); c.Warnings != 0 || c.Approved != 1 || c.Errors != 4 {
		t.Errorf("after the approval: %+v", c)
	}
}
