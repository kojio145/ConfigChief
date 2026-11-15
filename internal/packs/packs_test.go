package packs

import (
	"testing"

	"github.com/kojio145/configchief/internal/pack"
	"github.com/kojio145/configchief/internal/packtest"
	"github.com/kojio145/configchief/internal/schemas"
)

// The share of the sample lines that must be taken in or ignored
// (spec 18, M0 and M1: nine lines in ten; separators are left out).
const minShare = 0.90

// Every bundled pack loads without a warning, fits the table definitions,
// passes its own tests and keeps the share of handled lines.
func TestBundledPacks(t *testing.T) {
	sc, err := schemas.Load()
	if err != nil {
		t.Fatalf("schemas: %v", err)
	}
	ids := IDs()
	if len(ids) != 2 || ids[0] != "cisco_ios" || ids[1] != "nec_ix" {
		t.Fatalf("ids = %v", ids)
	}
	for _, id := range ids {
		t.Run(id, func(t *testing.T) {
			fsys, err := FS(id)
			if err != nil {
				t.Fatal(err)
			}
			p, err := pack.Load(fsys)
			if err != nil {
				t.Fatalf("pack: %v", err)
			}
			if p.ID != id {
				t.Errorf("id %q in pack.yaml, folder %q", p.ID, id)
			}
			for _, w := range p.Warnings {
				t.Errorf("pack warning: %s", w)
			}
			for _, s := range pack.Lint(p, sc).Problems {
				t.Errorf("pack check: %s", s)
			}
			cases, err := packtest.RunImport(p, sc, fsys)
			if err != nil {
				t.Fatal(err)
			}
			if len(cases) == 0 {
				t.Fatal("the pack has no import test")
			}
			for _, c := range cases {
				if len(c.Failures) > 0 {
					t.Errorf("%s:\n%s", c.Name, packtest.Report(c))
				}
				lines := c.Count.Total - c.Count.Separators
				share := float64(c.Count.Taken+c.Count.Ignored) / float64(lines)
				if share < minShare {
					t.Errorf("%s: %.1f%% of %d lines handled, want %.0f%%", c.Name, share*100, lines, minShare*100)
				}
				for _, w := range c.Result.Warnings {
					t.Logf("%s: warning: %s", c.Name, w)
				}
			}
			renders, err := packtest.RunRender(p, sc, fsys)
			if err != nil {
				t.Fatal(err)
			}
			if len(p.Sections) > 0 && len(renders) == 0 {
				t.Fatal("the pack has templates and no render test")
			}
			for _, c := range renders {
				if len(c.Failures) > 0 {
					t.Errorf("render %s:\n%s\n%s", c.Name, packtest.ReportRender(c), c.Output.Text)
				}
			}
		})
	}
}

func TestUnknownPack(t *testing.T) {
	if _, err := FS("no_such_pack"); err == nil {
		t.Error("no error for a pack that is not bundled")
	}
}
