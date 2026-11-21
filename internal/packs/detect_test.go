package packs

import (
	"io/fs"
	"sort"
	"testing"

	"github.com/kojio145/configchief/internal/pack"
)

// The import samples of every bundled pack are scored by all the packs: the
// pack of the sample comes first, 2 points or more ahead of the next (the
// gap below which a person has to choose: appendix B 2.1), so that a text
// of one vendor is never taken for another's. A whole configuration also
// reaches the 3 points that choose a pack; some samples of the older packs
// are parts of a configuration and stay below that.
func TestDetectApart(t *testing.T) {
	loaded := map[string]*pack.Pack{}
	for _, id := range IDs() {
		fsys, err := FS(id)
		if err != nil {
			t.Fatal(err)
		}
		p, err := pack.Load(fsys)
		if err != nil {
			t.Fatalf("%s: %v", id, err)
		}
		loaded[id] = p
	}
	for _, id := range IDs() {
		fsys, _ := FS(id)
		inputs, _ := fs.Glob(fsys, "tests/import/*/input.txt")
		if len(inputs) == 0 {
			t.Errorf("%s has no import sample", id)
		}
		for _, in := range inputs {
			text, err := fs.ReadFile(fsys, in)
			if err != nil {
				t.Fatal(err)
			}
			type scored struct {
				id string
				n  int
			}
			var all []scored
			for _, other := range IDs() {
				all = append(all, scored{other, loaded[other].Score(string(text))})
			}
			sort.SliceStable(all, func(i, j int) bool { return all[i].n > all[j].n })
			if all[0].id != id || all[0].n-all[1].n < 2 || (id == "nec_qx" && all[0].n < 3) {
				t.Errorf("%s/%s: scores %v", id, in, all)
			}
		}
	}
}
