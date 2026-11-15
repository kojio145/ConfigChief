package packs

import (
	"io/fs"
	"strings"
	"testing"

	"github.com/kojio145/configchief/internal/importer"
	"github.com/kojio145/configchief/internal/pack"
	"github.com/kojio145/configchief/internal/schemas"
)

// An access list that only IPsec names is taken out of the access lists once
// its lines stand on the sheet of the selectors (appendix B 3.6, "remove"):
// the lines stay read, and a list that something else names stays whole.
func TestIPsecTakesItsAccessList(t *testing.T) {
	sc, err := schemas.Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range []struct{ id, gone, kept string }{
		{"cisco_ios", "ACL-SAMPLE-VPN", "ACL-SAMPLE-NAT"},
		{"nec_ix", "acl-sample-vpn", "acl-sample-nat"},
	} {
		fsys, err := FS(c.id)
		if err != nil {
			t.Fatal(err)
		}
		p, err := pack.Load(fsys)
		if err != nil {
			t.Fatal(err)
		}
		text, err := fs.ReadFile(fsys, "tests/import/basic/input.txt")
		if err != nil {
			t.Fatal(err)
		}
		res, err := importer.Import(p, sc, "sample.txt", string(text))
		if err != nil {
			t.Fatal(err)
		}
		names := map[string]bool{}
		if tb := res.Data.Tables["acls"]; tb != nil {
			for _, r := range tb.Rows {
				names[r.Get("name")] = true
			}
		}
		if names[c.gone] {
			t.Errorf("%s: the list %s of IPsec is still on the sheet of the access lists", c.id, c.gone)
		}
		if !names[c.kept] {
			t.Errorf("%s: the list %s, which NAT names, is gone", c.id, c.kept)
		}
		if tb := res.Data.Tables["acl_entries"]; tb != nil {
			for _, r := range tb.Rows {
				if r.Get("acl") == c.gone {
					t.Errorf("%s: an entry of %s is still there: %s", c.id, c.gone, r.KeyText())
				}
			}
		}
		for _, l := range res.Lines {
			if strings.Contains(l.Text, c.gone) && l.State != importer.Taken {
				t.Errorf("%s: line %d (%s) is not read any more", c.id, l.Line, l.Text)
			}
			for _, ref := range l.Refs {
				if ref.Table == "acls" && strings.Contains(ref.Key, c.gone) {
					t.Errorf("%s: line %d still points at the row that is gone", c.id, l.Line)
				}
			}
		}
	}
}
