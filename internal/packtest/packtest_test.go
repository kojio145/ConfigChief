package packtest

import (
	"os"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/kojio145/configchief/internal/pack"
)

const mini = "../importer/testdata/mini"

func TestMiniPasses(t *testing.T) {
	fsys := os.DirFS(mini)
	p, err := pack.Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	cases, err := RunImport(p, defs(t), fsys)
	if err != nil {
		t.Fatal(err)
	}
	if len(cases) != 1 || cases[0].Name != "basic" {
		t.Fatalf("cases = %+v", cases)
	}
	if len(cases[0].Failures) > 0 {
		t.Errorf("failures:\n%s", Report(cases[0]))
	}
	rep := Report(cases[0])
	for _, want := range []string{"未対応 9", "合格", "mystery knob 5"} {
		if !strings.Contains(rep, want) {
			t.Errorf("the report does not have %q:\n%s", want, rep)
		}
	}
}

// A wrong expectation must fail, and say what differs.
func TestFailuresAreReported(t *testing.T) {
	read := func(name string) []byte {
		b, err := os.ReadFile(mini + "/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return b
	}
	fsys := fstest.MapFS{
		"pack.yaml":                    {Data: read("pack.yaml")},
		"ignore.yaml":                  {Data: read("ignore.yaml")},
		"import/10_rules.yaml":         {Data: read("import/10_rules.yaml")},
		"tests/import/basic/input.txt": {Data: read("tests/import/basic/input.txt")},
		"tests/import/basic/expect.yaml": {Data: []byte(`tables:
  interfaces:
    - {name: GigabitEthernet0/2, enabled: false}
  vrfs:
    - {name: none}
unknown:
  - "mystery knob 5"
  - "a line that is not in the sample"
ignored_min: 99
`)},
	}
	p, err := pack.Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	cases, err := RunImport(p, defs(t), fsys)
	if err != nil {
		t.Fatal(err)
	}
	all := strings.Join(cases[0].Failures, "\n")
	for _, want := range []string{
		"interfaces の 1 件目", // no defaults.yaml here, so enabled stays empty
		"vrfs の 1 件目: 表に行がありません",
		"「未対応」になりました: deeper mystery", // unknown but not listed
		"そうなっていません: a line that is not in the sample",
		"下限の 99 行",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("failures do not have %q:\n%s", want, all)
		}
	}
}
