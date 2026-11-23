package ops

import (
	"strings"
	"testing"
)

// The checks of the model (appendix C 1.1): the interface names of a device
// whose model is known (V-111), a model that is not known (V-114, V-116),
// and a function the model does not have (G-607). The data is made up.
func TestModelChecks(t *testing.T) {
	e := newCase(t, map[string]string{"input/old/rt-m01.txt": siteFile(t, "model", "rt-m01.txt")})
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	setCell(t, e, "機器対応", "旧ホスト名", "rt-m01", "新ホスト名", "ix-m01")
	setCell(t, e, "機器対応", "旧ホスト名", "rt-m01", "新メーカー", "nec_ix")
	setCell(t, e, "機器対応", "旧ホスト名", "rt-m01", "新機種", "ix-2ge-8sw")
	if _, err := e.RunImport("old", "", at); err != nil {
		t.Fatal(err)
	}
	if _, err := e.MakeBook("old", false, at); err != nil {
		t.Fatal(err)
	}
	of := func(o *GenerateOutcome, id string) []Judged {
		var out []Judged
		for _, f := range o.Items {
			if f.RuleID == id {
				out = append(out, f)
			}
		}
		return out
	}

	// The model is known and every name fits it: the ports, the interfaces
	// of the VLAN groups, the subinterface, the loopback.
	o, err := e.Generate(GenOptions{}, at)
	if err != nil {
		t.Fatal(err)
	}
	if fs := of(o, "V-111"); len(fs) != 0 || o.Tally.Looked("V-111")["ix-m01"] == 0 {
		t.Errorf("V-111 with the model known: %d findings, looked at %d", len(fs), o.Tally.Looked("V-111")["ix-m01"])
	}
	for _, id := range []string{"V-114", "V-116"} {
		if fs := of(o, id); len(fs) != 0 || o.Tally.Looked(id)["ix-m01"] != 1 {
			t.Errorf("%s with the model known: %v", id, fs)
		}
	}
	if why := o.Tally.Skipped("G-607")["ix-m01"]; why != "機種の定義（ix-2ge-8sw）に、対応していない機能を書いていません" {
		t.Errorf("G-607 not run: %q", why)
	}
	if s := statOf(t, e, o, "G-607"); s.State != wordNotRun || !strings.Contains(s.Note, "対応していない機能を書いていません") {
		t.Errorf("G-607 = %+v", s)
	}

	// A model without the built-in switch: its ports are not there (V-111),
	// and the rows of the switch ports use what it does not have (G-607).
	setCell(t, e, "機器", "ホスト名", "ix-m01", "機種", "ix-4ge")
	if o, err = e.Generate(GenOptions{}, at); err != nil {
		t.Fatal(err)
	}
	bad := map[string]bool{}
	for _, f := range of(o, "V-111") {
		bad[f.Ref.Key] = true
		if f.Severity != "error" {
			t.Errorf("V-111 is %s", f.Severity)
		}
	}
	for _, name := range []string{"GigaEthernet2:port1", "GigaEthernet2:1.0", "GigaEthernet2:3.0"} {
		if !bad["name="+name] {
			t.Errorf("V-111 is not told for %s: %v", name, bad)
		}
	}
	for _, name := range []string{"GigaEthernet0.0", "GigaEthernet0.1", "Loopback0.0"} {
		if bad["name="+name] {
			t.Errorf("V-111 is told for %s, which the model has", name)
		}
	}
	if fs := of(o, "G-607"); len(fs) != 1 || !strings.Contains(fs[0].Message, "内蔵スイッチ") || fs[0].Ref.Table != "switchports" {
		t.Errorf("G-607 = %+v", fs)
	}

	// A mistyped ID: a warning that names the ID, and no V-114.
	setCell(t, e, "機器", "ホスト名", "ix-m01", "機種", "IX-2GE-8SV")
	if o, err = e.Generate(GenOptions{}, at); err != nil {
		t.Fatal(err)
	}
	if fs := of(o, "V-116"); len(fs) != 1 || fs[0].Severity != "warning" || !strings.Contains(fs[0].Message, "ix-2ge-8sw") {
		t.Errorf("V-116 = %+v", fs)
	}
	if fs := of(o, "V-114"); len(fs) != 0 {
		t.Errorf("V-114 with V-116: %+v", fs)
	}
	if why := o.Tally.Skipped("V-111")["ix-m01"]; !strings.Contains(why, "V-114・V-116") {
		t.Errorf("V-111 not run: %q", why)
	}

	// A product name, far from every ID, and no model: information only.
	for _, m := range []string{"sample router 100", ""} {
		setCell(t, e, "機器", "ホスト名", "ix-m01", "機種", m)
		if o, err = e.Generate(GenOptions{}, at); err != nil {
			t.Fatal(err)
		}
		fs := of(o, "V-114")
		if len(fs) != 1 || fs[0].Severity != "info" || !strings.Contains(fs[0].Message, "ix-4ge") || len(of(o, "V-116")) != 0 {
			t.Errorf("model %q: V-114 = %+v", m, fs)
		}
		if s := statOf(t, e, o, "V-111"); s.State != wordNotRun || !strings.Contains(s.Note, "機種が空か") {
			t.Errorf("model %q: V-111 = %+v", m, s)
		}
	}
}
