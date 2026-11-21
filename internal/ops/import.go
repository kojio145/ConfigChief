package ops

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kojio145/configchief/internal/importer"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/project"
)

// Candidate is a pack with the points the text of a file gave it.
type Candidate struct {
	Pack  string
	Score int
}

// File is one file of a source and what became of it.
type File struct {
	Path       string      // from the case folder, written with "/"
	Pack       string      // the pack that read it ("" if none)
	Chosen     bool        // a person chose the pack (M-410)
	Candidates []Candidate // highest first
	Err        string      // why the file was not imported
	Result     *importer.Result
	// Device is the host name the file gave, also when it was not imported
	// because another file has the same one.
	Device string
	// Same lists the files of the folder with the same host name, this one
	// among them (nil if the host name is its own).
	Same []string
	// Left tells that a person chose another file for the host name: the
	// file is left out, and that is no fault ("" if not).
	Left string
}

// Source is the outcome of importing one folder of input/.
type Source struct {
	Name  string // "old" | "new"
	Files []*File
	Data  *model.Dataset // the rows of all devices
}

// File gives the file a device was read from, or nil.
func (s *Source) File(device string) *File {
	for _, f := range s.Files {
		if f.Result != nil && f.Result.Device == device {
			return f
		}
	}
	return nil
}

// Devices lists the host names that were imported, in the order of the
// files.
func (s *Source) Devices() []string {
	var out []string
	for _, f := range s.Files {
		if f.Result != nil {
			out = append(out, f.Result.Device)
		}
	}
	return out
}

// Count adds up the lines of all files by state.
func (s *Source) Count() importer.Count {
	var c importer.Count
	for _, f := range s.Files {
		if f.Result == nil {
			continue
		}
		n := f.Result.Count()
		c.Total += n.Total
		c.Separators += n.Separators
		c.Outside += n.Outside
		c.Taken += n.Taken
		c.Ignored += n.Ignored
		c.Unknown += n.Unknown
	}
	return c
}

// The pack is clear when it has at least minScore points and leads the next
// one by at least minLead (appendix B 2.1).
const (
	minScore = 3
	minLead  = 2
)

// detect gives the packs by points and the pack that is clearly the one
// ("" if none is).
func (e *Env) detect(text string) (cands []Candidate, clear string) {
	for _, id := range e.PackIDs {
		cands = append(cands, Candidate{id, e.Packs[id].Score(text)})
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].Score > cands[j].Score })
	if len(cands) == 0 || cands[0].Score < minScore {
		return cands, ""
	}
	if len(cands) > 1 && cands[0].Score-cands[1].Score < minLead {
		return cands, ""
	}
	return cands, cands[0].Pack
}

// Import reads every file of input/<source> into design data (spec 8.1).
//
// choose is the pack for the files whose vendor is not clear ("" = none);
// what it was used for is kept in state/sources.yaml. A file that cannot be
// read or told does not stop the others: it is listed with its reason.
func (e *Env) Import(source, choose string) (*Source, error) {
	if source != "old" && source != "new" {
		return nil, fmt.Errorf("データ源「%s」は取り込めません（old か new）", source)
	}
	if choose != "" && e.Packs[choose] == nil {
		return nil, fmt.Errorf("パック %s はありません（使えるパック: %s）", choose, strings.Join(e.PackIDs, ", "))
	}
	paths, err := e.Proj.Inputs(source)
	if err != nil {
		return nil, err
	}
	chosen, err := e.Proj.Sources()
	if err != nil {
		return nil, err
	}
	src := &Source{Name: source, Data: model.NewDataset(source)}
	changed := false
	// The files are read side by side (none needs another), and taken in
	// their order: the host names that come twice, and what is put together.
	type read struct {
		f     *File
		fresh bool // the pack is the one chosen in this call
		res   *importer.Result
	}
	reads := make([]read, len(paths))
	work := newSpread(len(paths), func(_, i int) {
		path := paths[i]
		f := &File{Path: path}
		reads[i].f = f
		text, err := project.ReadText(e.Proj.Path(path))
		if err != nil {
			f.Err = err.Error()
			return
		}
		if strings.TrimSpace(text) == "" {
			f.Err = "中身が空です"
			return
		}
		var clear string
		f.Candidates, clear = e.detect(text)
		switch {
		case clear != "":
			f.Pack = clear
		case chosen.Pack(path) != "" && e.Packs[chosen.Pack(path)] != nil:
			f.Pack, f.Chosen = chosen.Pack(path), true
		case choose != "":
			f.Pack, f.Chosen = choose, true
			reads[i].fresh = true
		default:
			f.Err = "メーカーを判定できません（" + candidateText(f.Candidates) + "）。パックを選んでください"
			return
		}
		res, err := importer.Import(e.Packs[f.Pack], e.Schema, path, text)
		if err != nil {
			f.Err = err.Error()
			return
		}
		reads[i].res = res
	})
	defer work.stop()
	var fresh []string
	same := map[string][]string{} // host name → the files that have it
	for i, path := range paths {
		if e.Tick != nil {
			if err := e.Tick(i, len(paths), path); err != nil {
				return nil, err
			}
		}
		work.wait(i)
		f, res := reads[i].f, reads[i].res
		src.Files = append(src.Files, f)
		if reads[i].fresh {
			fresh = append(fresh, path)
		}
		if res != nil {
			f.Device = res.Device
			same[res.Device] = append(same[res.Device], path)
		}
	}
	// A host name that two or more files have: the tool does not pick one
	// by itself. The file a person chose is taken and the others are left
	// out; until one is chosen, none is taken (spec 8.1 step 4).
	for i := range paths {
		f, res := reads[i].f, reads[i].res
		if res == nil {
			continue
		}
		if list := same[res.Device]; len(list) > 1 {
			f.Same = list
			use := chosen.Use(source, res.Device)
			found := false
			for _, p := range list {
				found = found || p == use
			}
			switch {
			case !found:
				f.Err = fmt.Sprintf("ホスト名 %s のファイルが %d つあります（%s）。どれを取り込むか選んでください", res.Device, len(list), strings.Join(list, "、"))
				continue
			case use != f.Path:
				f.Left = fmt.Sprintf("ホスト名 %s は %s を取り込みます（人が選んだ）。このファイルは使いません", res.Device, use)
				continue
			}
		}
		f.Result = res
		for _, name := range res.Data.TableNames() {
			for _, r := range res.Data.Tables[name].Rows {
				src.Data.Adopt(r)
			}
		}
	}
	// Written down only now: while the files are read, what was chosen
	// before is only looked at.
	for _, path := range fresh {
		chosen.Choose(path, choose)
		changed = true
	}
	if changed {
		if err := e.Proj.SaveSources(chosen); err != nil {
			return nil, err
		}
	}
	return src, nil
}

func candidateText(cands []Candidate) string {
	if len(cands) == 0 {
		return "使えるパックがありません"
	}
	parts := make([]string, len(cands))
	for i, c := range cands {
		parts[i] = fmt.Sprintf("%s %d 点", c.Pack, c.Score)
	}
	return "候補: " + strings.Join(parts, "、")
}
