package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kojio145/configchief/internal/ops"
	"github.com/kojio145/configchief/internal/textdiff"
)

// The structured comparison of configurations (spec 12.2).

// Piece is a stretch of a line; Hit says that it differs from the other
// side.
type Piece struct {
	Text string `json:"t"`
	Hit  bool   `json:"h,omitempty"`
}

// TreeRow is one row of a structured comparison, or a run of rows that are
// the same (Op "gap").
type TreeRow struct {
	Op    string  `json:"op"` // "same" | "del" | "add" | "chg" | "gap"
	A     int     `json:"a"`  // line of A (0 = none)
	B     int     `json:"b"`
	TA    []Piece `json:"ta"`
	TB    []Piece `json:"tb"`
	Level int     `json:"level"`
	Head  bool    `json:"head"`
	Moved bool    `json:"moved"`
	Noise bool    `json:"noise"`
	Text  string  `json:"text"` // of a gap
}

// TreeView is two configurations compared block by block.
type TreeView struct {
	Title   string    `json:"title"` // the device ("" = two files)
	A       string    `json:"a"`
	B       string    `json:"b"`
	Notes   []string  `json:"notes"`
	Equal   bool      `json:"equal"`
	Added   int       `json:"added"`
	Removed int       `json:"removed"`
	Changed int       `json:"changed"`
	Moved   int       `json:"moved"`
	LeftOut string    `json:"leftOut"`
	Rows    []TreeRow `json:"rows"`
}

// treeContext is how many rows that are the same are shown around a row
// that differs. The headings a row stands below are always shown.
const treeContext = 2

func pieces(text string, spans [][2]int) []Piece {
	text = strings.TrimRight(text, " \t\r\n")
	out := []Piece{}
	pos := 0
	for _, sp := range spans {
		s, e := min(sp[0], len(text)), min(sp[1], len(text))
		if s > pos {
			out = append(out, Piece{Text: text[pos:s]})
		}
		if e > s {
			out = append(out, Piece{Text: text[s:e], Hit: true})
		}
		pos = max(pos, e)
	}
	if pos < len(text) {
		out = append(out, Piece{Text: text[pos:]})
	}
	return out
}

func treeView(title string, a, b *ops.Listing, r *textdiff.TreeResult) *TreeView {
	v := &TreeView{Title: title, A: a.Path, B: b.Path, Notes: []string{}, Equal: !r.Differs(),
		Added: r.Added, Removed: r.Removed, Changed: r.Changed, Moved: r.Moved, Rows: []TreeRow{},
		LeftOut: fmt.Sprintf("比べなかった行（A と B の合計）: 時刻の行 %d、証明書・暗号化値 %d、コメントの行 %d", r.Stamps, r.Secrets, r.Comments)}
	for i, l := range []*ops.Listing{a, b} {
		for _, n := range l.Notes {
			v.Notes = append(v.Notes, "AB"[i:i+1]+": "+n)
		}
	}
	if r.Approx {
		v.Notes = append(v.Notes, "違いが多すぎるので、行ごとに比べた所があります（件数は多めに出ています）")
	}
	if v.Equal {
		return v
	}
	// show[i]: row i differs, is close to one that does, or is a heading
	// over one.
	show := make([]bool, len(r.Rows))
	var stack []int
	for i, row := range r.Rows {
		for len(stack) > row.Level {
			stack = stack[:len(stack)-1]
		}
		if ops.TreeKind(row) != "" {
			for j := max(i-treeContext, 0); j <= min(i+treeContext, len(r.Rows)-1); j++ {
				show[j] = true
			}
			for _, j := range stack {
				show[j] = true
			}
		}
		if row.Head && !row.Noise {
			stack = append(stack, i)
		}
	}
	names := map[textdiff.Op]string{textdiff.Same: "same", textdiff.Del: "del", textdiff.Add: "add", textdiff.Chg: "chg"}
	gap := 0
	flush := func() {
		if gap > 0 {
			v.Rows = append(v.Rows, TreeRow{Op: "gap", Text: fmt.Sprintf("同じ行 %d 行", gap), TA: []Piece{}, TB: []Piece{}})
			gap = 0
		}
	}
	for i, row := range r.Rows {
		if !show[i] {
			gap++
			continue
		}
		flush()
		v.Rows = append(v.Rows, TreeRow{Op: names[row.Op], A: row.ALine, B: row.BLine, TA: pieces(row.A, row.ASpans), TB: pieces(row.B, row.BSpans),
			Level: row.Level, Head: row.Head, Moved: row.Moved, Noise: row.Noise})
	}
	flush()
	return v
}

func treeOptions(stamps, secrets bool) textdiff.TreeOptions {
	return textdiff.TreeOptions{Stamps: stamps, Secrets: secrets}
}

// Tree compares two files block by block, as the command "diff --tree"
// does. It needs no case; the texts are shown as they stand.
func (s *Session) Tree(a, b string, stamps, secrets bool) (*TreeView, error) {
	if a == "" || b == "" {
		return nil, errors.New("比べるファイルを 2 つ選んでください")
	}
	s.mu.Lock()
	env := s.env
	s.mu.Unlock()
	var ps *ops.Packset
	if env != nil {
		ps = env.Packset()
	} else {
		var err error
		if ps, err = ops.Bundled(); err != nil {
			return nil, err
		}
	}
	p, err := ops.TreeFiles(ps, a, b, treeOptions(stamps, secrets))
	if err != nil {
		return nil, err
	}
	return treeView("", p.A, p.B, &p.Result), nil
}

// TreeDevice is one device of a comparison of every device.
type TreeDevice struct {
	Device  string `json:"device"`
	A       string `json:"a"` // the file ("" = the side has no such device)
	B       string `json:"b"`
	Both    bool   `json:"both"`
	Differs bool   `json:"differs"`
	Added   int    `json:"added"`
	Removed int    `json:"removed"`
	Changed int    `json:"changed"`
	Moved   int    `json:"moved"`
	Notes   string `json:"notes"`
}

// TreeAllView is every device of two sides compared block by block.
type TreeAllView struct {
	Title   string       `json:"title"`
	Counts  []Pair       `json:"counts"`
	Notes   []string     `json:"notes"`
	Devices []TreeDevice `json:"devices"`
	Failed  Table        `json:"failed"`
	Report  Files        `json:"report"`
}

// TreeAll compares every device of two sides, as the command "diff --tree
// --project" does: the report is written. A side is "old", "new", "logs",
// "generated" or a folder.
func (s *Session) TreeAll(a, b string, stamps, secrets bool) (*TreeAllView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	s.tree = nil
	now := s.now()
	o, err := e.TreeDiff(a, b, treeOptions(stamps, secrets), now)
	if err != nil {
		return nil, err
	}
	s.tree = o
	d := e.TreeDoc(o, now)
	v := &TreeAllView{Title: d.Title, Counts: pairs(d.Counts), Notes: strs(d.Notes), Devices: []TreeDevice{},
		Failed: Table{Header: []string{}, Rows: [][]string{}, Marks: []string{}, Code: []int{}}, Report: Files{o.Report.XLSX, o.Report.HTML}}
	for _, sh := range d.Sheets {
		if sh.Name == "比べなかったファイル" {
			v.Failed = table(sh)
		}
	}
	for i := range o.Devices {
		x := &o.Devices[i]
		dev := TreeDevice{Device: x.Device, Both: x.Result != nil, Differs: x.Differs()}
		var notes []string
		for j, f := range []*ops.TreeFile{x.A, x.B} {
			if f == nil {
				continue
			}
			if j == 0 {
				dev.A = f.Path
			} else {
				dev.B = f.Path
			}
			for _, n := range f.Notes {
				notes = append(notes, "AB"[j:j+1]+": "+n)
			}
		}
		if r := x.Result; r != nil {
			dev.Added, dev.Removed, dev.Changed, dev.Moved = r.Added, r.Removed, r.Changed, r.Moved
		}
		dev.Notes = strings.Join(notes, " / ")
		v.Devices = append(v.Devices, dev)
	}
	return v, nil
}

// TreeOf gives the rows of one device of the comparison TreeAll made last.
func (s *Session) TreeOf(i int) (*TreeView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.tree == nil || i < 0 || i >= len(s.tree.Devices) {
		return nil, errors.New("比べ直してください（前の結果がありません）")
	}
	d := &s.tree.Devices[i]
	if d.Result == nil {
		return nil, fmt.Errorf("機器 %s は片方にしかありません", d.Device)
	}
	return treeView(d.Device, d.A.Listing, d.B.Listing, d.Result), nil
}
