package ops

import (
	"os"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/project"
)

// The settings of the checks are written into project.yaml, nothing else of
// it changing (spec 4.2, 14.11.5).
func TestSaveCheckSettings(t *testing.T) {
	e := newCase(t, nil)
	path := e.Proj.Path(project.ConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// A setting of an ID this version does not know stays.
	text := strings.Replace(string(data), "severity: {}", "severity: {Z-999: warning}", 1)
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	if e.Proj, err = project.Open(e.Proj.Dir); err != nil {
		t.Fatal(err)
	}
	list := e.CheckSettings()
	if len(list.Checks) < 100 || strings.Join(list.Unknown, ",") != "Z-999" {
		t.Fatalf("%d checks, unknown %v", len(list.Checks), list.Unknown)
	}
	if _, err := e.SaveCheckSettings(map[string]string{"V-101": "off"}, at); err == nil {
		t.Error("V-101 was stopped")
	}
	if c, err := e.SaveCheckSettings(map[string]string{}, at); err != nil || c != "" {
		t.Errorf("nothing changed: %q, %v", c, err)
	}
	copyTo, err := e.SaveCheckSettings(map[string]string{"V-208": "warning", "V-210": "off", "V-101": "error"}, at)
	if err != nil || copyTo == "" {
		t.Fatalf("save: %q, %v", copyTo, err)
	}
	if read(t, e, copyTo) != text {
		t.Error("the copy is not project.yaml as it was")
	}
	after := read(t, e, project.ConfigFile)
	for _, want := range []string{"# 止めるチェックの ID", "# 重大度の上書き", "# 出力物で秘密の値を伏せる", "name:", "Z-999: warning", "V-208: warning", "[V-210]"} {
		if !strings.Contains(after, want) {
			t.Errorf("project.yaml lacks %q:\n%s", want, after)
		}
	}
	p, err := project.Open(e.Proj.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if p.Config.Checks.Severity["V-208"] != "warning" || strings.Join(p.Config.Checks.Disable, ",") != "V-210" || p.Config.Checks.Severity["V-101"] != "" {
		t.Errorf("read back: %+v", p.Config.Checks)
	}
	for _, s := range e.CheckSettings().Checks {
		if (s.ID == "V-208" && s.Now != "warning") || (s.ID == "V-210" && s.Now != "off") || (s.ID == "V-101" && (s.Now != "" || !s.Stays)) {
			t.Errorf("%+v", s)
		}
	}
	// Back to the defaults.
	if _, err := e.SaveCheckSettings(map[string]string{"V-208": "", "V-210": ""}, at); err != nil {
		t.Fatal(err)
	}
	if p, _ = project.Open(e.Proj.Dir); len(p.Config.Checks.Disable) != 0 || len(p.Config.Checks.Severity) != 1 {
		t.Errorf("after the defaults: %+v", p.Config.Checks)
	}
}

// A sample template goes into templates/ and the delivery sheets come out
// of it (spec 14.11.2).
func TestMakeSample(t *testing.T) {
	e := migCase(t)
	if _, err := e.MakeBook("old", false, at); err != nil {
		t.Fatal(err)
	}
	a, err := e.MakeSample("sheets", at)
	if err != nil || a != "templates/見本_機器ごとのシート.xlsx" {
		t.Fatalf("%q, %v", a, err)
	}
	b, err := e.MakeSample("sheets", at)
	if err != nil || b != "templates/見本_機器ごとのシート_2.xlsx" {
		t.Fatalf("again: %q, %v", b, err)
	}
	if err := os.Remove(e.Proj.Path(b)); err != nil {
		t.Fatal(err)
	}
	out, err := e.ExportSheets(false, at)
	if err != nil {
		t.Fatal(err)
	}
	if len(out.Problems) != 0 || len(out.Files) == 0 || out.Standard {
		t.Errorf("export: %+v", out)
	}
}
