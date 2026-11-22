package ops

import (
	"fmt"
	"sort"
	"time"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/convert"
)

// IfRedo is one row of IF 対応 whose proposal is made again (spec 14.11.4):
// what the row holds and what goes in.
type IfRedo struct {
	NewHost  string `json:"newHost"`
	OldIf    string `json:"oldIf"`
	Was      string `json:"was"`
	WasBasis string `json:"wasBasis"`
	Now      string `json:"now"`
	Basis    string `json:"basis"`
}

// IfRedone is what RedoIfs found or did.
type IfRedone struct {
	Rows   []IfRedo `json:"rows"`
	Kept   int      `json:"kept"`   // rows a person wrote (no basis), left as they are
	Backup string   `json:"backup"` // "" = nothing was written
}

// RedoIfs makes the proposals of IF 対応 again for the devices of the new
// host names (spec 14.11.4). The rows that hold a proposal of the tool (a
// basis) and the rows whose new name is empty get the proposal of now; a
// row without a proposal now is emptied. A name a person wrote (no basis)
// stays. With write false nothing is written: the rows tell what would
// change.
func (e *Env) RedoIfs(hosts []string, write bool, now time.Time) (*IfRedone, error) {
	old, bk, err := e.readBook()
	if err != nil {
		return nil, err
	}
	if bk == nil {
		return nil, fmt.Errorf("%s がありません。先に旧コンフィグを取り込んでください", e.Proj.Config.Book)
	}
	if len(hosts) == 0 {
		return nil, fmt.Errorf("案を作り直す機器（新ホスト名）を選んでください")
	}
	want := map[string]bool{}
	for _, h := range hosts {
		want[h] = true
	}
	src, err := e.importSide("old", "")
	if err != nil {
		return nil, err
	}
	nw, err := e.newSide()
	if err != nil {
		return nil, err
	}
	_, prop := convert.ProposeWith(src.Data, data(nw), bk.Devices, e.shapes)
	by := map[string]book.IfMap{}
	for _, p := range prop {
		by[p.NewHost+"\x00"+p.OldIf] = p
	}
	out := &IfRedone{Rows: []IfRedo{}}
	var set []book.IfMap
	for _, m := range bk.Ifs {
		if m.Row == 0 || !want[m.NewHost] {
			continue
		}
		if m.Basis == "" && m.NewIf != "" {
			out.Kept++
			continue
		}
		p := by[m.NewHost+"\x00"+m.OldIf]
		if p.NewIf == "" {
			p.Basis = ""
		}
		if p.NewIf == m.NewIf && p.Basis == m.Basis {
			continue
		}
		out.Rows = append(out.Rows, IfRedo{NewHost: m.NewHost, OldIf: m.OldIf, Was: m.NewIf, WasBasis: m.Basis, Now: p.NewIf, Basis: p.Basis})
		set = append(set, book.IfMap{NewHost: m.NewHost, OldIf: m.OldIf, NewIf: p.NewIf, Basis: p.Basis})
	}
	sort.SliceStable(out.Rows, func(i, j int) bool { return out.Rows[i].NewHost < out.Rows[j].NewHost })
	if !write || len(set) == 0 {
		return out, nil
	}
	next, _, err := book.Update(old, e.Proj.Config.Book, e.Schema, e.lang(), book.Change{IfSet: set})
	if err != nil {
		return nil, err
	}
	if err := e.Proj.BookWritable(); err != nil {
		return nil, err
	}
	if out.Backup, err = e.saveBook(now, next); err != nil {
		return nil, err
	}
	e.Proj.Log(now, "IF 対応の案を作り直しました: %d 行（機器 %d 台、人が書いた %d 行はそのまま。控え %s）", len(set), len(hosts), out.Kept, out.Backup)
	return out, nil
}

// Said tells what RedoIfs did, in a sentence.
func (r *IfRedone) Said() string {
	kept := ""
	if r.Kept > 0 {
		kept = fmt.Sprintf("。人が書いた %d 行（根拠が空）は変えていません", r.Kept)
	}
	if len(r.Rows) == 0 {
		return "作り直して変わる行はありませんでした" + kept
	}
	if r.Backup == "" {
		return fmt.Sprintf("%d 行が変わります%s", len(r.Rows), kept)
	}
	return fmt.Sprintf("IF 対応の %d 行の案を作り直しました（控え %s）%s", len(r.Rows), r.Backup, kept)
}
