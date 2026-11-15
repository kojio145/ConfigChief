package packs

import (
	"io/fs"
	"testing"

	"github.com/kojio145/configchief/internal/importer"
	"github.com/kojio145/configchief/internal/pack"
	"github.com/kojio145/configchief/internal/schemas"
)

// A list that only picks a DSCP or a precedence for a QoS class becomes the
// condition of the class (appendix A 3.18) and leaves the access lists; a
// list with addresses, and a list the filter also uses, stay.
func TestQoSTakesItsAccessList(t *testing.T) {
	sc, err := schemas.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct {
		id         string
		gone, kept []string
	}{
		{"nec_ix", []string{"al-q-ef", "al-q-af41", "al-q-prec"}, []string{"al-q-biz", "al-q-peer", "al-q-flt"}},
	} {
		fsys, err := FS(c.id)
		if err != nil {
			t.Fatal(err)
		}
		p, err := pack.Load(fsys)
		if err != nil {
			t.Fatal(err)
		}
		text, err := fs.ReadFile(fsys, "tests/import/qos/input.txt")
		if err != nil {
			t.Fatal(err)
		}
		res, err := importer.Import(p, sc, "sample.txt", string(text))
		if err != nil {
			t.Fatal(err)
		}
		names := map[string]bool{}
		for _, r := range res.Data.Tables["acls"].Rows {
			names[r.Get("name")] = true
		}
		entries := map[string]bool{}
		for _, r := range res.Data.Tables["acl_entries"].Rows {
			entries[r.Get("acl")] = true
		}
		for _, n := range c.gone {
			if names[n] || entries[n] {
				t.Errorf("%s: the list %s is still among the access lists", c.id, n)
			}
		}
		for _, n := range c.kept {
			if !names[n] || !entries[n] {
				t.Errorf("%s: the list %s is gone", c.id, n)
			}
		}
		for _, l := range res.Lines {
			if l.State == importer.Unknown {
				t.Errorf("%s: line %d is unknown: %s", c.id, l.Line, l.Text)
			}
		}
	}
}

// A Cisco parent policy that only shapes the whole interface is folded into
// the binding (appendix A 3.18): its row leaves the policies, its lines stay
// read.
func TestQoSFoldsTheParentPolicy(t *testing.T) {
	sc, err := schemas.Load()
	if err != nil {
		t.Fatal(err)
	}
	fsys, err := FS("cisco_ios")
	if err != nil {
		t.Fatal(err)
	}
	p, err := pack.Load(fsys)
	if err != nil {
		t.Fatal(err)
	}
	text, err := fs.ReadFile(fsys, "tests/import/qos/input.txt")
	if err != nil {
		t.Fatal(err)
	}
	res, err := importer.Import(p, sc, "sample.txt", string(text))
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range res.Data.Tables["qos_policies"].Rows {
		if r.Get("policy") == "pm-q-out" || r.Get("policy") == "pm-q-shape" {
			t.Errorf("the parent policy is still there: %s", r.KeyText())
		}
	}
	for _, l := range res.Lines {
		if l.State == importer.Unknown && l.Text != " qos pre-classify" {
			t.Errorf("line %d is unknown: %s", l.Line, l.Text)
		}
	}
}
