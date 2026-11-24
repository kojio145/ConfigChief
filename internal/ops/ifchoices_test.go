package ops

import (
	"slices"
	"testing"

	"github.com/kojio145/configchief/internal/book"
)

// The new interface names are offered by new device: the ports of its new
// model, the names its rows hold, and "-" (spec 14.19.6).
func TestIfChoices(t *testing.T) {
	e := newCase(t, map[string]string{"input/old/rt-a01.txt": sample(t, "cisco_ios")})
	p := e.Packs["nec_ix"]
	if p == nil || len(p.Models) == 0 {
		t.Skip("no model in the pack")
	}
	md := p.Models[0]
	devs := &book.EditSheet{ID: "@devices", Columns: []book.EditColumn{{Key: "old_host"}, {Key: "new_host"}, {Key: "new_vendor"}, {Key: "new_model"}},
		Rows: []book.EditRow{{Cells: []string{"rt-a01", "rt-b01", "nec_ix", md.ID}}, {Cells: []string{"rt-a02", "rt-b02", "", ""}}}}
	ifs := &book.EditSheet{ID: "@ifs", Columns: []book.EditColumn{{Key: "new_host"}, {Key: "old_if"}, {Key: "new_if"}},
		Rows: []book.EditRow{{Cells: []string{"rt-b01", "Loopback0", "Loopback0.0"}}, {Cells: []string{"rt-b02", "Gi0/0", "GigaEthernet9.0"}}, {Cells: []string{"rt-b02", "Gi0/1", "-"}}}}
	all, notes, by, sets := ifChoices(e, []*book.EditSheet{devs, ifs}, ifs)
	if by != "new_host" || notes["-"] == "" {
		t.Errorf("by %q, notes %v", by, notes)
	}
	one := sets["rt-b01"]
	if len(one) != len(md.Ports)+2 || one[0] != md.Ports[0].Name || !slices.Contains(one, "Loopback0.0") || one[len(one)-1] != "-" {
		t.Errorf("rt-b01: %v", one)
	}
	if two := sets["rt-b02"]; !slices.Equal(two, []string{"GigaEthernet9.0", "-"}) {
		t.Errorf("rt-b02 without a model: %v", two)
	}
	if !slices.Contains(all, "GigaEthernet9.0") || !slices.Contains(all, md.Ports[0].Name) {
		t.Errorf("all: %v", all)
	}
}
