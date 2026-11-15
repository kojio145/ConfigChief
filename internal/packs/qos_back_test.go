package packs

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/importer"
	"github.com/kojio145/configchief/internal/pack"
	"github.com/kojio145/configchief/internal/render"
	"github.com/kojio145/configchief/internal/schemas"
)

// The QoS samples of the import tests are written back by the same pack and
// by the other one: what was read is written with nothing lost, and the
// text reads back as the tables say. The findings that are expected are
// listed; anything else is a fault of a pack.
func TestQoSSamplesWriteBack(t *testing.T) {
	sc, err := schemas.Load()
	if err != nil {
		t.Fatal(err)
	}
	load := func(id string) *pack.Pack {
		fsys, err := FS(id)
		if err != nil {
			t.Fatal(err)
		}
		p, err := pack.Load(fsys)
		if err != nil {
			t.Fatal(err)
		}
		return p
	}
	for _, c := range []struct {
		from, sample, to string
		want             []string // the findings that are told, by what they begin with
	}{
		{"cisco_ios", "qos", "cisco_ios", nil},
		{"cisco_ios", "qos_old", "cisco_ios", []string{"G-614"}},
		{"nec_ix", "qos", "nec_ix", nil},
	} {
		t.Run(c.from+"/"+c.sample+" to "+c.to, func(t *testing.T) {
			fsys, _ := FS(c.from)
			text, err := fs.ReadFile(fsys, "tests/import/"+c.sample+"/input.txt")
			if err != nil {
				t.Fatal(err)
			}
			res, err := importer.Import(load(c.from), sc, "sample.txt", string(text))
			if err != nil {
				t.Fatal(err)
			}
			if c.to != c.from {
				for _, r := range res.Data.Tables["devices"].Rows {
					r.Cells["vendor"].Values = []string{c.to}
				}
			}
			r := render.New(load(c.to), sc, nil)
			out, findings := r.Render(res.Data, res.Device, nil)
			findings = append(findings, r.RoundTrip(out)...)
			var got []string
			for _, f := range findings {
				if f.Severity == "info" {
					continue
				}
				line := f.RuleID + " " + f.Ref.Table + " " + f.Ref.Key + " " + f.Ref.Column + ": " + f.Message
				ok := false
				for _, w := range c.want {
					if strings.HasPrefix(line, w) {
						ok = true
					}
				}
				if !ok {
					got = append(got, line)
				}
			}
			t.Logf("%d lines written, %d findings told, failed=%v", strings.Count(out.Text, "\n"), len(findings), out.Failed)
			if out.Failed {
				t.Errorf("the text is not whole")
			}
			if len(got) > 0 {
				t.Errorf("findings:\n%s\n\n%s", strings.Join(got, "\n"), out.Text)
			}
		})
	}
}
