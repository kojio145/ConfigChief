// Package ops runs the steps of a case from start to end: importing a source,
// making the parameter book, comparing. The screens and the command line call
// the same functions.
//
// Spec: spec 3, 8, 10, 12.
package ops

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/kojio145/configchief/internal/pack"
	"github.com/kojio145/configchief/internal/packs"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/schema"
	"github.com/kojio145/configchief/internal/schemas"
)

// Env is an open case with what the steps need: the table definitions and
// the vendor packs.
type Env struct {
	Proj   *project.Project
	Schema *schema.Set
	Packs  map[string]*pack.Pack
	// PackIDs lists the packs that can be used, by ID.
	PackIDs []string
	// PackProblems tells why a pack of the case folder was not loaded.
	PackProblems []string
	// Version is the version of the tool, for the reports ("" = not shown).
	Version string
	// Tick, if set, is called before each file of an import and before each
	// device of a generation, with how many are done. An error it gives
	// stops the step; nothing is written then.
	Tick func(done, total int, file string) error
	// Edit, if set, keeps the sheets of the book between the steps of the
	// table of a screen (spec 14.6).
	Edit *EditCache

	// screen is set on the copy of the environment that makes the texts of
	// the screens: they mask the secrets whatever the case says (spec 14).
	screen bool
	// draftBook, when set, is read in the place of the book: the book with
	// the draft of the table put in (spec 14.12.3). Nothing writes it.
	draftBook []byte
}

// Open opens a case folder and loads the table definitions and the packs:
// the bundled ones, and those under packs/ of the case folder, which win over
// a bundled pack of the same ID (appendix B 1).
func Open(dir string) (*Env, error) {
	proj, err := project.Open(dir)
	if err != nil {
		return nil, err
	}
	sc, err := schemas.Load()
	if err != nil {
		return nil, fmt.Errorf("表の定義を読めません:\n%v", err)
	}
	e := &Env{Proj: proj, Schema: sc, Packs: map[string]*pack.Pack{}}
	for _, id := range packs.IDs() {
		fsys, err := packs.FS(id)
		if err != nil {
			return nil, err
		}
		p, err := pack.Load(fsys)
		if err != nil {
			return nil, fmt.Errorf("同梱のパック %s を読めません:\n%v", id, err)
		}
		e.Packs[p.ID] = p
	}
	entries, _ := os.ReadDir(proj.Path("packs"))
	for _, ent := range entries {
		if !ent.IsDir() {
			continue
		}
		rel := "packs/" + ent.Name()
		p, err := pack.Load(os.DirFS(proj.Path(rel)))
		if err != nil {
			e.PackProblems = append(e.PackProblems, fmt.Sprintf("%s は使いません:\n%v", rel, err))
			continue
		}
		// A pack with a fault is not loaded (appendix B 8).
		if lint := pack.Lint(p, sc); len(lint.Problems) > 0 {
			e.PackProblems = append(e.PackProblems, fmt.Sprintf("%s は使いません:\n  %s", rel, strings.Join(lint.Problems, "\n  ")))
			continue
		}
		e.Packs[p.ID] = p
	}
	for id := range e.Packs {
		e.PackIDs = append(e.PackIDs, id)
	}
	sort.Strings(e.PackIDs)
	return e, nil
}
