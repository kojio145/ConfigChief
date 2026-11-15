package ops

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/conftree"
	"github.com/kojio145/configchief/internal/importer"
	"github.com/kojio145/configchief/internal/pack"
	"github.com/kojio145/configchief/internal/packs"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/report"
	"github.com/kojio145/configchief/internal/textdiff"
)

// The structured comparison of configurations (spec 12.2): two files, or
// every device of two sides.

// Packset is the packs a text is told by.
type Packset struct {
	Packs map[string]*pack.Pack
	IDs   []string
}

// Bundled loads the packs that come with the tool: what a comparison of two
// files uses when no case is open.
func Bundled() (*Packset, error) {
	ps := &Packset{Packs: map[string]*pack.Pack{}}
	for _, id := range packs.IDs() {
		fsys, err := packs.FS(id)
		if err != nil {
			return nil, err
		}
		p, err := pack.Load(fsys)
		if err != nil {
			return nil, fmt.Errorf("同梱のパック %s を読めません:\n%v", id, err)
		}
		ps.Packs[p.ID] = p
		ps.IDs = append(ps.IDs, p.ID)
	}
	sort.Strings(ps.IDs)
	return ps, nil
}

// Packset gives the packs of the case.
func (e *Env) Packset() *Packset { return &Packset{Packs: e.Packs, IDs: e.PackIDs} }

// clear gives the pack that is clearly the one of a text ("" if none is).
func (ps *Packset) clear(text string) string {
	e := &Env{Packs: ps.Packs, PackIDs: ps.IDs}
	_, id := e.detect(text)
	return id
}

// The notes of a listing.
const (
	treeNoPack   = "メーカーを判定できないので、ファイル全体をそのまま比べました（コメントの行も比べています）"
	treeNoConfig = "端末ログですが、設定全体の表示（show running-config など）がありません。ログ全体を比べました"
)

// Listing is the configuration of one file, ready to be compared.
type Listing struct {
	Path   string
	Pack   string // "" = the vendor is not clear: the whole text is compared
	Log    bool   // a terminal log: only the configuration it shows is compared
	Source textdiff.Source
	Notes  []string
}

// listing cuts the configuration out of a text. id is the pack to read it
// with ("" = the one that is clearly its vendor's).
func (ps *Packset) listing(path, text, id string) *Listing {
	l := &Listing{Path: path}
	if id == "" {
		id = ps.clear(text)
	}
	p := ps.Packs[id]
	if p == nil {
		l.Source = textdiff.Text(text)
		l.Notes = append(l.Notes, treeNoPack)
		return l
	}
	l.Pack = id
	cut := p.Session.Cut(text)
	if !cut.Log && cut.Prompts >= 1 {
		l.Notes = append(l.Notes, treeNoConfig)
	}
	l.Log = cut.Log
	src := textdiff.Source{Lines: cut.Config, Outside: make([]bool, len(cut.Config)), Comment: make([]bool, len(cut.Config))}
	for i, why := range cut.Skip {
		src.Outside[i] = why != ""
	}
	// The comment lines are the lines the pack's syntax keeps out of the
	// tree (the text of a banner is in it).
	if tree, err := conftree.Parse(p.Syntax, strings.Join(cut.Config, "\n")+"\n", p.Options); err == nil {
		for i := range src.Comment {
			src.Comment[i] = i < len(tree.InTree) && !tree.InTree[i] && !src.Outside[i] && strings.TrimSpace(cut.Config[i]) != ""
		}
	}
	l.Source = src
	return l
}

// TreePair is two files compared.
type TreePair struct {
	A, B   *Listing
	Result textdiff.TreeResult
}

// TreeFiles compares two files block by block (spec 12.2). It needs no
// case.
func TreeFiles(ps *Packset, a, b string, opt textdiff.TreeOptions) (*TreePair, error) {
	var ls [2]*Listing
	for i, p := range []string{a, b} {
		text, err := project.ReadText(p)
		if err != nil {
			return nil, err
		}
		ls[i] = ps.listing(p, text, "")
	}
	return &TreePair{A: ls[0], B: ls[1], Result: textdiff.Tree(ls[0].Source, ls[1].Source, opt)}, nil
}

// TreeSides lists the sides of a comparison of every device that have a
// name; anything else is a folder.
var TreeSides = []string{"old", "new", "logs", "generated"}

// TreeFile is the configuration of one device on one side.
type TreeFile struct {
	*Listing
	Host string // the host name the device is paired by
}

// TreeDevice is one device of a comparison of every device.
type TreeDevice struct {
	Device string
	A, B   *TreeFile // nil = the side has no such device
	Result *textdiff.TreeResult
}

// Differs says whether a person has something to look at for the device.
func (d *TreeDevice) Differs() bool { return d.Result == nil || d.Result.Differs() }

// TreeFail is a file that was not compared.
type TreeFail struct {
	Side, Path, Why string
}

// TreeOutcome is every device of two sides compared.
type TreeOutcome struct {
	SideA, SideB string
	Options      textdiff.TreeOptions
	Devices      []TreeDevice
	Failed       []TreeFail
	Notes        []string
	Report       Files
}

// Left counts what a person has to look at: the devices that differ or are
// on one side only, and the files that were not compared.
func (o *TreeOutcome) Left() int {
	n := len(o.Failed)
	for i := range o.Devices {
		if o.Devices[i].Differs() {
			n++
		}
	}
	return n
}

// Total adds up the counts of the devices that were compared.
func (o *TreeOutcome) Total() (t textdiff.TreeResult) {
	for _, d := range o.Devices {
		if r := d.Result; r != nil {
			t.Added += r.Added
			t.Removed += r.Removed
			t.Changed += r.Changed
			t.Moved += r.Moved
			t.Stamps += r.Stamps
			t.Secrets += r.Secrets
			t.Comments += r.Comments
		}
	}
	return t
}

// treeSide reads one side: the files of a folder of input/, the
// configurations the book gives now, or the files of any folder.
func (e *Env) treeSide(name string, now time.Time, tick func(file string) error) (files []*TreeFile, failed []TreeFail, err error) {
	if name == "generated" {
		gen, err := e.Generate(GenOptions{Dry: true}, now)
		if err != nil {
			return nil, nil, err
		}
		for _, d := range gen.Devices {
			if d.Output == nil || d.Output.Failed {
				failed = append(failed, TreeFail{name, d.Host, "コンフィグを生成できませんでした: " + d.Why})
				continue
			}
			l := e.Packset().listing(d.Host+"（生成）", d.Output.Text, d.Pack)
			files = append(files, &TreeFile{Listing: l, Host: d.Host})
		}
		return files, failed, nil
	}
	var paths []string // as shown
	read := func(p string) string { return p }
	chosen, err := e.Proj.Sources()
	if err != nil {
		return nil, nil, err
	}
	switch name {
	case "old", "new", "logs":
		if paths, err = e.Proj.Inputs(name); err != nil {
			return nil, nil, err
		}
		read = e.Proj.Path
	default:
		entries, err := os.ReadDir(name)
		if err != nil {
			return nil, nil, fmt.Errorf("%s を読めません。側は old・new・logs・generated か、フォルダです: %v", name, err)
		}
		for _, ent := range entries {
			if n := ent.Name(); !ent.IsDir() && !strings.HasPrefix(n, ".") && !strings.HasPrefix(n, "~$") {
				paths = append(paths, filepath.ToSlash(filepath.Join(name, n)))
			}
		}
		sort.Strings(paths)
	}
	if len(paths) == 0 {
		return nil, nil, fmt.Errorf("%s にファイルがありません", sideTitle(name))
	}
	seen := map[string]string{}
	for _, p := range paths {
		if tick != nil {
			if err := tick(p); err != nil {
				return nil, nil, err
			}
		}
		text, err := project.ReadText(read(p))
		switch {
		case err != nil:
			failed = append(failed, TreeFail{name, p, err.Error()})
			continue
		case strings.TrimSpace(text) == "":
			failed = append(failed, TreeFail{name, p, "中身が空です"})
			continue
		}
		id := ""
		if c := chosen.Pack(p); c != "" && e.Packs[c] != nil && e.Packset().clear(text) == "" {
			id = c
		}
		l := e.Packset().listing(p, text, id)
		host := strings.TrimSuffix(path.Base(p), path.Ext(p))
		if l.Pack != "" {
			if res, err := importer.Import(e.Packs[l.Pack], e.Schema, p, text); err == nil && res.Device != "" {
				host = res.Device
			} else {
				l.Notes = append(l.Notes, "コンフィグからホスト名を読めないので、ファイル名で組にしました")
			}
		}
		if first := seen[host]; first != "" {
			failed = append(failed, TreeFail{name, p, fmt.Sprintf("ホスト名 %s が %s と同じです。このファイルは比べません", host, first)})
			continue
		}
		seen[host] = p
		files = append(files, &TreeFile{Listing: l, Host: host})
	}
	return files, failed, nil
}

func sideTitle(name string) string {
	switch name {
	case "old", "new", "logs":
		return "input/" + name
	case "generated":
		return "パラメータブックから生成したコンフィグ"
	}
	return name
}

// TreeDiff compares every device of two sides block by block and writes the
// report (spec 12.2). The devices are paired by their host names; a device
// of input/old that has no partner is looked for by its new host name.
func (e *Env) TreeDiff(a, b string, opt textdiff.TreeOptions, now time.Time) (*TreeOutcome, error) {
	if a == "" || b == "" {
		return nil, fmt.Errorf("比べる 2 つの側を選んでください（old・new・logs・generated か、フォルダ）")
	}
	if a == b {
		return nil, fmt.Errorf("2 つの側が同じです（%s）", a)
	}
	o := &TreeOutcome{SideA: a, SideB: b, Options: opt}
	var sides [2][]*TreeFile
	done := 0
	for i, name := range []string{a, b} {
		files, failed, err := e.treeSide(name, now, func(file string) error {
			if e.Tick == nil {
				return nil
			}
			done++
			return e.Tick(done-1, 0, file)
		})
		if err != nil {
			return nil, err
		}
		sides[i] = files
		o.Failed = append(o.Failed, failed...)
	}
	// The new names of the old devices.
	renamed := map[string]string{}
	if a == "old" || b == "old" {
		if _, bk, err := e.readBook(); err != nil {
			return nil, err
		} else if bk != nil {
			for _, d := range bk.Devices {
				if d.NewHost != "" && d.NewHost != "-" {
					renamed[d.OldHost] = d.NewHost
				}
			}
		}
	}
	byHost := map[string]*TreeFile{}
	for _, f := range sides[1] {
		byHost[f.Host] = f
	}
	used := map[*TreeFile]bool{}
	partner := func(f *TreeFile) *TreeFile {
		if p := byHost[f.Host]; p != nil && !used[p] {
			return p
		}
		if a == "old" {
			if p := byHost[renamed[f.Host]]; p != nil && !used[p] {
				return p
			}
		}
		return nil
	}
	if b == "old" {
		// The other way round: the devices of a are looked for by the new
		// names of the old ones.
		for _, f := range sides[1] {
			if n := renamed[f.Host]; n != "" && byHost[n] == nil {
				byHost[n] = f
			}
		}
	}
	for _, fa := range sides[0] {
		d := TreeDevice{Device: fa.Host, A: fa}
		if fb := partner(fa); fb != nil {
			used[fb] = true
			d.B = fb
			r := textdiff.Tree(fa.Source, fb.Source, opt)
			d.Result = &r
			if fb.Host != fa.Host {
				d.Device = fa.Host + " → " + fb.Host
				if b == "old" {
					d.Device = fb.Host + " → " + fa.Host
				}
			}
		}
		o.Devices = append(o.Devices, d)
	}
	for _, fb := range sides[1] {
		if !used[fb] {
			o.Devices = append(o.Devices, TreeDevice{Device: fb.Host, B: fb})
		}
	}
	files, err := e.writeReport(now, "tree_diff", e.TreeDoc(o, now))
	if err != nil {
		return nil, err
	}
	o.Report = files
	t := o.Total()
	e.Proj.Log(now, "階層を見た比較: %s と %s、機器 %d、追加 %d、削除 %d、変更 %d、順序 %d", a, b, len(o.Devices), t.Added, t.Removed, t.Changed, t.Moved)
	return o, nil
}

// TreeKind names what a row of a structured comparison is ("" = the same).
func TreeKind(r textdiff.Row) string {
	switch {
	case r.Noise:
		return ""
	case r.Op == textdiff.Add:
		return "B だけ"
	case r.Op == textdiff.Del:
		return "A だけ"
	case r.Op == textdiff.Chg && r.Moved:
		return "変更・順序"
	case r.Op == textdiff.Chg:
		return "変更"
	case r.Moved:
		return "順序"
	}
	return ""
}

// TreeTrail gives the headings a row stands below as one text.
func TreeTrail(parents []textdiff.Row) string {
	var out []string
	for _, p := range parents {
		t := p.B
		if t == "" {
			t = p.A
		}
		out = append(out, strings.TrimSpace(t))
	}
	return strings.Join(out, " > ")
}

// TreeOptionText tells what a comparison left out.
func TreeOptionText(opt textdiff.TreeOptions) string {
	word := func(on bool) string {
		if on {
			return "無視する"
		}
		return "比べる"
	}
	return "時刻の行: " + word(opt.Stamps) + "、証明書・暗号化値: " + word(opt.Secrets)
}

// TreeSecretNote tells that the texts stand as they are.
const TreeSecretNote = "文面はそのまま出ています。どの語が秘密かをツールが見分けられないので、パスワードや鍵も伏せていません。"

// TreeDoc is the report of a comparison of every device.
func (e *Env) TreeDoc(o *TreeOutcome, now time.Time) *report.Doc {
	d := &report.Doc{Title: "階層を見た比較", Facts: e.facts(now)}
	d.Facts = append(d.Facts,
		report.Pair{Label: "A", Value: sideTitle(o.SideA)},
		report.Pair{Label: "B", Value: sideTitle(o.SideB)},
		report.Pair{Label: "比べ方", Value: TreeOptionText(o.Options)})
	t := o.Total()
	both, differ, onlyA, onlyB := 0, 0, 0, 0
	for i := range o.Devices {
		switch dev := &o.Devices[i]; {
		case dev.Result == nil && dev.A != nil:
			onlyA++
		case dev.Result == nil:
			onlyB++
		default:
			both++
			if dev.Result.Differs() {
				differ++
			}
		}
	}
	d.Counts = []report.Pair{
		{Label: "比べた機器", Value: num(both)},
		{Label: "違いのある機器", Value: num(differ)},
		{Label: "A にだけある機器", Value: num(onlyA)},
		{Label: "B にだけある機器", Value: num(onlyB)},
		{Label: "B だけの行（追加）", Value: num(t.Added)},
		{Label: "A だけの行（削除）", Value: num(t.Removed)},
		{Label: "変更の行", Value: num(t.Changed)},
		{Label: "順序が違う行", Value: num(t.Moved)},
		{Label: "比べなかったファイル", Value: num(len(o.Failed))},
	}
	d.Notes = append(d.Notes,
		"コンフィグを字下げのブロック（interface とその下の行など）ごとに突き合わせて、文面を比べています。設計データは使っていません。",
		"「順序」は、両方にあって場所だけが違う行・ブロックです。追加・削除には数えていません。",
		fmt.Sprintf("比べなかった行（A と B の合計）: 時刻の行 %d、証明書・暗号化値 %d、コメントの行 %d。", t.Stamps, t.Secrets, t.Comments),
		TreeSecretNote)
	d.Notes = append(d.Notes, o.Notes...)

	devs := report.Sheet{Name: "機器", Lead: "機器ごとの件数です。",
		Header: []string{"機器", "A のファイル", "B のファイル", "追加", "削除", "変更", "順序", "比べなかった行", "状態", "注意"}}
	lines := report.Sheet{Name: "違い", Lead: "違いのある行です。「親の行」は、その行が入っているブロックの見出しです。",
		Header: []string{"機器", "種類", "親の行", "A の行", "A", "B の行", "B"}, Code: []int{2, 4, 6}}
	for i := range o.Devices {
		dev := &o.Devices[i]
		row := []string{dev.Device, "", "", "", "", "", "", "", "", ""}
		var notes []string
		note := func(side string, f *TreeFile) {
			for _, n := range f.Notes {
				notes = append(notes, side+": "+n)
			}
		}
		mark := ""
		if dev.A != nil {
			row[1] = dev.A.Path
			note("A", dev.A)
		}
		if dev.B != nil {
			row[2] = dev.B.Path
			note("B", dev.B)
		}
		switch r := dev.Result; {
		case r == nil && dev.A != nil:
			row[8], mark = "A にだけあります", report.MarkWarn
		case r == nil:
			row[8], mark = "B にだけあります", report.MarkWarn
		default:
			row[3], row[4], row[5], row[6] = num(r.Added), num(r.Removed), num(r.Changed), num(r.Moved)
			row[7] = num(r.Stamps + r.Secrets + r.Comments)
			row[8] = "違いはありません"
			if r.Differs() {
				row[8], mark = "違いがあります", report.MarkWarn
			}
			if r.Approx {
				notes = append(notes, "違いが多すぎるので、行ごとに比べた所があります（件数は多めに出ています）")
			}
			r.Walk(func(_ int, x textdiff.Row, parents []textdiff.Row) {
				kind := TreeKind(x)
				if kind == "" {
					return
				}
				la, lb := "", ""
				if x.ALine > 0 {
					la = num(x.ALine)
				}
				if x.BLine > 0 {
					lb = num(x.BLine)
				}
				lines.Rows = append(lines.Rows, []string{dev.Device, kind, TreeTrail(parents), la, strings.TrimRight(x.A, " \t"), lb, strings.TrimRight(x.B, " \t")})
			})
		}
		row[9] = strings.Join(notes, " / ")
		devs.Rows = append(devs.Rows, row)
		devs.Marks = append(devs.Marks, mark)
	}
	d.Sheets = append(d.Sheets, devs, lines)
	if len(o.Failed) > 0 {
		s := report.Sheet{Name: "比べなかったファイル", Header: []string{"側", "ファイル", "理由"}}
		for _, f := range o.Failed {
			side := "B"
			if f.Side == o.SideA {
				side = "A"
			}
			s.Rows = append(s.Rows, []string{side, f.Path, f.Why})
			s.Marks = append(s.Marks, report.MarkUnknown)
		}
		d.Sheets = append(d.Sheets, s)
	}
	return d
}
