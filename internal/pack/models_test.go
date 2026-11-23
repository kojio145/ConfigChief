package pack

import (
	"strings"
	"testing"
	"testing/fstest"

	"github.com/kojio145/configchief/internal/schemas"
)

// What models.yaml may say for the checks of the model (appendix B 6.1):
// the marks of other_names, and what a model does not have.
func TestModelsForChecks(t *testing.T) {
	load := func(models string) (*Pack, error) {
		return Load(fstest.MapFS{
			"pack.yaml":            {Data: []byte(read(t, "pack.yaml"))},
			"import/10_rules.yaml": {Data: []byte(read(t, "import/10_rules.yaml"))},
			"models.yaml":          {Data: []byte(models)},
		})
	}
	const head = "models:\n  - id: sample-a\n    ports: [{name: GigabitEthernet0/0}, {range: \"GigabitEthernet0/1/[0-3]\", role: switch}]\n"
	p, err := load(head + "    other_names: [\"Sample{n}/{n}\"]\n    unsupported:\n      - {name: スイッチ, table: switchports, when: \"row.mode != 'routed'\"}\n")
	if err != nil {
		t.Fatal(err)
	}
	m := p.Model("SAMPLE-A")
	for name, want := range map[string]bool{"GigabitEthernet0/1/3": true, "gigabitethernet0/0": true, "Sample1/2": true, "Sample1": false, "GigabitEthernet0/1/4": false, "GigabitEthernet0/0.10": true, "Loopback3": true} {
		if got := p.FitsModel(m, name); got != want {
			t.Errorf("FitsModel(%s) = %v", name, got)
		}
	}
	sc, err := schemas.Load()
	if err != nil {
		t.Fatal(err)
	}
	if res := Lint(p, sc); len(res.Problems) != 0 {
		t.Errorf("lint: %v", res.Problems)
	}

	if _, err := load(head + "    other_names: [\"Sample{vlan}\"]\n"); err == nil || !strings.Contains(err.Error(), "other_names") {
		t.Errorf("a mark other_names cannot have: %v", err)
	}
	if _, err := load(head + "    unsupported: [{name: スイッチ}]\n"); err == nil || !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("unsupported without a table: %v", err)
	}
	p, err = load(head + "    unsupported: [{name: スイッチ, table: no_such_table}, {name: 式, table: switchports, when: \"row.mode ==\"}]\n")
	if err != nil {
		t.Fatal(err)
	}
	res := Lint(p, sc)
	if txt := strings.Join(res.Problems, "\n"); !strings.Contains(txt, "no_such_table") || !strings.Contains(txt, "when") {
		t.Errorf("lint: %v", res.Problems)
	}
}
