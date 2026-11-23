package ops

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/convert"
	"github.com/kojio145/configchief/internal/importer"
	"github.com/kojio145/configchief/internal/judge"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/project"
)

// VerdictNone asks to take a judgment away (it is never written down).
const VerdictNone = "none"

var verdictWords = map[string]string{
	"":                       notJudged,
	project.VerdictHold:      "保留",
	project.VerdictManual:    "追加コンフィグで書く",
	project.VerdictNotNeeded: "要らない",
}

// The order the verdicts are told in: what is open first.
var verdictOrder = []string{"", project.VerdictHold, project.VerdictManual, project.VerdictNotNeeded}

// VerdictWord gives the word for a verdict ("" = not judged yet).
func VerdictWord(v string) string { return verdictWords[v] }

// open tells whether a verdict leaves the line to be decided (M-401).
func open(verdict string) bool { return verdict == "" || verdict == project.VerdictHold }

// LineVerdict is what is decided about one unhandled line.
type LineVerdict struct {
	group *LineGroup
	// Head is false for a line that follows the unhandled line above it.
	Head                    bool
	Verdict, Reason, By, At string
}

// Open tells whether the line is still to be decided: not judged, or on hold.
func (v LineVerdict) Open() bool { return open(v.Verdict) }

// Group gives the ID of the group the line (or the line it follows) is in.
func (v LineVerdict) Group() string { return v.group.ID }

// Word gives the verdict as the reports show it.
func (v LineVerdict) Word() string {
	if v.Head {
		return VerdictWord(v.Verdict)
	}
	return VerdictWord(v.Verdict) + "（上の行に従う）"
}

// HeadLine is an unhandled line a judgment is put on: the top-most one.
type HeadLine struct {
	Line  int
	Text  string
	Below int // the lines that follow it
}

// GroupDevice is what one device has of a group.
type GroupDevice struct {
	Device, File string
	Heads        []HeadLine
	Below        int
	// The judgment that holds for the device ("" = none). Own tells that it
	// names the device, and is not the one for every device.
	Verdict, Reason, By, At string
	Own                     bool
}

// Lines counts the lines of the device in the group.
func (d *GroupDevice) Lines() int { return len(d.Heads) + d.Below }

// LineGroup is the unhandled lines of one shape (spec 8.5).
type LineGroup struct {
	ID      string
	Key     judge.Key
	Devices []*GroupDevice
	// An example: the first line of the group, with the lines below it.
	Example      HeadLine
	ExampleBelow []string
	ExampleMore  int // the lines below that are not in ExampleBelow
}

// exampleCap is how many of the lines below the example are kept.
const exampleCap = 30

// Count gives the top-most lines of the group and the lines below them.
func (g *LineGroup) Count() (heads, below int) {
	for _, d := range g.Devices {
		heads += len(d.Heads)
		below += d.Below
	}
	return heads, below
}

// Open tells whether a device of the group is not judged or on hold.
func (g *LineGroup) Open() bool {
	for _, d := range g.Devices {
		if open(d.Verdict) {
			return true
		}
	}
	return false
}

// Device finds a device of the group by its host name (nil = not there).
func (g *LineGroup) Device(host string) *GroupDevice {
	for _, d := range g.Devices {
		if d.Device == host && host != "" {
			return d
		}
	}
	return nil
}

// VerdictText tells the judgment of the group: the word, or the devices of
// each verdict when they differ.
func (g *LineGroup) VerdictText() string {
	n := map[string]int{}
	for _, d := range g.Devices {
		n[d.Verdict]++
	}
	var parts []string
	for _, v := range verdictOrder {
		if n[v] == len(g.Devices) {
			return VerdictWord(v)
		}
		if n[v] > 0 {
			parts = append(parts, fmt.Sprintf("%s %d 台", VerdictWord(v), n[v]))
		}
	}
	return strings.Join(parts, "、")
}

// differs is shown where the devices of a group do not say the same.
const differs = "（機器によって違う）"

// ReasonText and ByText tell the reason and the person of the judgments of
// the group ("" = no device is judged).
func (g *LineGroup) ReasonText() string {
	return g.same(func(d *GroupDevice) string { return d.Reason })
}

func (g *LineGroup) ByText() string {
	return g.same(func(d *GroupDevice) string { return d.By })
}

func (g *LineGroup) same(of func(*GroupDevice) string) string {
	text, first := "", true
	for _, d := range g.Devices {
		if d.Verdict == "" {
			continue
		}
		if !first && of(d) != text {
			return differs
		}
		text, first = of(d), false
	}
	return text
}

// PathText gives the lines above the group in one line ("" = the top level).
func (g *LineGroup) PathText() string { return strings.Join(g.Key.Path, " > ") }

// NoExtra is a device with lines judged "by hand" that no extra configuration
// is for (appendix C M-402).
type NoExtra struct {
	Device, Host string
	Lines        int
}

// Judging is the judgments put on the unhandled lines of one side.
type Judging struct {
	Source string
	Groups []*LineGroup // in the order of their first lines
	lines  map[string]map[int]LineVerdict

	// The unhandled lines by verdict (the lines that follow one count too).
	Open, Hold, Manual, NotNeeded int
	NoExtra                       []NoExtra
	// HostsUnknown: the old side has lines judged "by hand", but there is no
	// book that tells the new host names, so M-402 was not looked at.
	HostsUnknown bool
	// BookErr: the book could not be read, so the extra configurations of
	// groups were not looked at.
	BookErr string
	// Stale are the judgments of this side that no line takes any more.
	Stale []project.Judgment
	// Unread are the files that were not imported: their lines are not here.
	Unread []string
}

// Line gives the verdict of a line (false = the line is not unhandled).
func (j *Judging) Line(file string, line int) (LineVerdict, bool) {
	if j == nil {
		return LineVerdict{}, false
	}
	v, ok := j.lines[file][line]
	return v, ok
}

// Findings counts what a person still has to do: the lines that are open
// (M-401) and the devices of M-402.
func (j *Judging) Findings() int { return j.Open + j.Hold + len(j.NoExtra) }

// OpenLines counts the lines of a file that are not judged or on hold.
func (j *Judging) OpenLines(file string) int {
	n := 0
	for _, v := range j.lines[file] {
		if open(v.Verdict) {
			n++
		}
	}
	return n
}

// Group finds a group by its ID, or by enough of its beginning.
func (j *Judging) Group(id string) (*LineGroup, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	if id == "" {
		return nil, fmt.Errorf("かたまりの ID を指定してください")
	}
	var hit []*LineGroup
	for _, g := range j.Groups {
		if g.ID == id {
			return g, nil
		}
		if strings.HasPrefix(g.ID, id) {
			hit = append(hit, g)
		}
	}
	switch len(hit) {
	case 0:
		return nil, fmt.Errorf("ID が %s のかたまりは、%sの読めなかった行にありません", id, sourceLabel(j.Source))
	case 1:
		return hit[0], nil
	}
	return nil, fmt.Errorf("%s で始まるかたまりが %d つあります。ID をもう少し長く書いてください", id, len(hit))
}

// hostView tells how the devices of a side are called on the new side.
type hostView struct {
	// known is false on the old side when no book tells the new host names.
	known bool
	// hosts gives the new host name of an old device. nil on the new side:
	// the device is the host.
	hosts   map[string]string
	book    *model.Dataset // the data of the book (nil = no book)
	bookErr string
}

// sideView reads what the book says about the devices of a side. It writes
// nothing.
func (e *Env) sideView(src *Source) hostView {
	_, bk, err := e.readBook()
	v := hostView{known: true}
	switch {
	case err != nil:
		v.bookErr = err.Error()
		v.known = src.Name != "old"
	case bk == nil:
		v.known = src.Name != "old"
	default:
		v.book = bk.Data
		if src.Name == "old" {
			v.hosts = convert.Apply(e.Schema, src.Data, bk.Devices, bk.Ifs).Hosts
		}
	}
	return v
}

// extraFile tells whether an entry of extra/ is an extra configuration.
func extraFile(ent os.DirEntry) bool {
	name := ent.Name()
	return !ent.IsDir() && !strings.HasPrefix(name, ".") && !strings.HasPrefix(name, "~$") && strings.HasSuffix(name, ".j2")
}

// extraNames gives the names of the extra configurations, without ".j2".
func (e *Env) extraNames() ([]string, error) {
	entries, err := os.ReadDir(e.Proj.Path("extra"))
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("extra を読めません: %v", err)
	}
	var out []string
	for _, ent := range entries {
		if extraFile(ent) {
			out = append(out, strings.TrimSuffix(ent.Name(), ".j2"))
		}
	}
	return out, nil
}

// linePath gives the shapes of the lines a line hangs below, the outermost
// first.
func linePath(r *importer.Result, l importer.LineState) []string {
	var path []string
	for p := l.Parent; p > 0; p = r.Lines[p-1].Parent {
		path = append(path, judge.Shape(r.Lines[p-1].Text))
	}
	for i, k := 0, len(path)-1; i < k; i, k = i+1, k-1 {
		path[i], path[k] = path[k], path[i]
	}
	return path
}

// judging puts the judgments of the case on the unhandled lines of a side
// (spec 8.4, 8.5). It writes nothing.
func (e *Env) judging(src *Source, view hostView) (*Judging, error) {
	list, err := e.Proj.Judgments()
	if err != nil {
		return nil, err
	}
	out := &Judging{Source: src.Name, lines: map[string]map[int]LineVerdict{}, BookErr: view.bookErr}
	groups := map[string]*LineGroup{} // by the hash of the key
	used := make([]bool, len(list))
	type manual struct {
		device string
		lines  int
	}
	var byHand []manual

	for _, f := range src.Files {
		r := f.Result
		if r == nil {
			out.Unread = append(out.Unread, f.Path)
			continue
		}
		verdicts := map[int]LineVerdict{}
		type place struct {
			of   *GroupDevice
			head int // the place in of.Heads
		}
		headOf := map[int]place{}
		devOf := map[*LineGroup]*GroupDevice{}
		hand := 0
		for _, l := range r.Lines {
			if l.State != importer.Unknown {
				continue
			}
			var v LineVerdict
			if l.Parent > 0 && r.Lines[l.Parent-1].State == importer.Unknown {
				// The line follows the unhandled line above it.
				p := headOf[l.Parent]
				headOf[l.Line] = p
				p.of.Heads[p.head].Below++
				p.of.Below++
				v = verdicts[p.of.Heads[p.head].Line]
				v.Head = false
				if g := v.group; g.Example.Line == p.of.Heads[p.head].Line && g.Devices[0] == p.of {
					if len(g.ExampleBelow) < exampleCap {
						g.ExampleBelow = append(g.ExampleBelow, l.Text)
					} else {
						g.ExampleMore++
					}
				}
			} else {
				k := judge.Key{Source: src.Name, Pack: f.Pack, Path: linePath(r, l), Shape: judge.Shape(l.Text)}
				hash := k.Hash()
				g := groups[hash]
				if g == nil {
					g = &LineGroup{Key: k, Example: HeadLine{Line: l.Line, Text: l.Text}}
					groups[hash] = g
					out.Groups = append(out.Groups, g)
				}
				d := devOf[g]
				if d == nil {
					d = &GroupDevice{Device: r.Device, File: f.Path}
					if at, own := judge.Find(list, k, r.Device); at >= 0 {
						j := list[at]
						d.Verdict, d.Reason, d.By, d.At, d.Own = j.Verdict, j.Reason, j.By, j.At, own
						used[at] = true
					}
					devOf[g] = d
					g.Devices = append(g.Devices, d)
				}
				d.Heads = append(d.Heads, HeadLine{Line: l.Line, Text: l.Text})
				headOf[l.Line] = place{of: d, head: len(d.Heads) - 1}
				v = LineVerdict{group: g, Head: true, Verdict: d.Verdict, Reason: d.Reason, By: d.By, At: d.At}
			}
			verdicts[l.Line] = v
			switch v.Verdict {
			case "":
				out.Open++
			case project.VerdictHold:
				out.Hold++
			case project.VerdictManual:
				out.Manual++
				hand++
			case project.VerdictNotNeeded:
				out.NotNeeded++
			}
		}
		if len(verdicts) > 0 {
			out.lines[f.Path] = verdicts
		}
		if hand > 0 {
			byHand = append(byHand, manual{r.Device, hand})
		}
	}

	keys := make([]judge.Key, len(out.Groups))
	for i, g := range out.Groups {
		keys[i] = g.Key
		// The example of the group keeps its count of lines below.
		g.Example.Below = g.Devices[0].Heads[0].Below
	}
	for i, id := range judge.IDs(keys) {
		out.Groups[i].ID = id
	}
	for i, j := range list {
		if !used[i] && judge.KeyOf(j).Source == src.Name {
			out.Stale = append(out.Stale, j)
		}
	}

	// M-402: a device with lines judged "by hand" needs an extra configuration.
	if len(byHand) == 0 {
		return out, nil
	}
	if !view.known {
		out.HostsUnknown = true
		return out, nil
	}
	extra, err := e.extraNames()
	if err != nil {
		return nil, err
	}
	var groupsOf map[string][]string
	if view.book != nil {
		groupsOf = book.Groups(view.book)
	}
	for _, m := range byHand {
		host := m.device
		if view.hosts != nil {
			host = view.hosts[m.device]
		}
		if host == "" {
			continue
		}
		found := has(extra, "_all") || has(extra, host)
		for _, g := range groupsOf[host] {
			found = found || has(extra, "@"+g)
		}
		if !found {
			out.NoExtra = append(out.NoExtra, NoExtra{Device: m.device, Host: host, Lines: m.lines})
		}
	}
	return out, nil
}

// Review reads the configurations of a side again and puts the judgments on
// their unhandled lines. It writes nothing (spec 14.2).
func (e *Env) Review(source string) (*Judging, error) {
	src, err := e.importSide(source, "")
	if err != nil {
		return nil, err
	}
	return e.judging(src, e.sideView(src))
}

// JudgeRequest asks to put a judgment on a group of unhandled lines, or to
// take it away (Verdict "none").
type JudgeRequest struct {
	Source  string // "old" (also when empty) | "new"
	Group   string // the ID, or enough of its beginning
	Devices string // "*" (also when empty), or host names with commas between
	Verdict string // manual | not_needed | hold | none
	Reason  string
	By      string
}

// JudgeOutcome is what JudgeLines did.
type JudgeOutcome struct {
	Judging *Judging   // the judgments after the step
	Group   *LineGroup // the group, as it is after the step
	Verdict string
	Devices int // the devices the judgment was put on (or taken from)
	Lines   int // their lines in the group
	// Own are the devices of the group that keep a judgment of their own
	// after one for every device was put or taken away.
	Own int
}

// JudgeLines puts a judgment into state/judgments.yaml (spec 8.4, 8.5). The
// configurations are read first: a judgment is for lines that are there.
func (e *Env) JudgeLines(req JudgeRequest, now time.Time) (*JudgeOutcome, error) {
	req.Reason, req.By, req.Verdict = strings.TrimSpace(req.Reason), strings.TrimSpace(req.By), strings.TrimSpace(req.Verdict)
	if req.Source == "" {
		req.Source = "old"
	}
	switch req.Verdict {
	case project.VerdictManual, project.VerdictHold, VerdictNone:
	case project.VerdictNotNeeded:
		if req.Reason == "" {
			return nil, fmt.Errorf("「要らない」には理由を書いてください")
		}
	default:
		return nil, fmt.Errorf("判定は manual（追加コンフィグで書く）・not_needed（要らない）・hold（保留）・none（取り消し）のどれかです（「%s」）", req.Verdict)
	}
	if req.By == "" && req.Verdict != VerdictNone {
		return nil, fmt.Errorf("判定した人の名前を書いてください")
	}
	src, err := e.importSide(req.Source, "")
	if err != nil {
		return nil, err
	}
	view := e.sideView(src)
	before, err := e.judging(src, view)
	if err != nil {
		return nil, err
	}
	g, err := before.Group(req.Group)
	if err != nil {
		return nil, err
	}
	devices, all := judge.Devices(req.Devices)
	if strings.TrimSpace(req.Devices) == "" {
		all = true
	}
	if !all {
		var missing []string
		for _, d := range devices {
			if g.Device(d) == nil {
				missing = append(missing, d)
			}
		}
		if len(missing) > 0 {
			return nil, fmt.Errorf("次の機器には、このかたまり（%s）の行がありません: %s", g.ID, strings.Join(missing, ", "))
		}
		devices = uniq(devices)
	}
	list, err := e.Proj.Judgments()
	if err != nil {
		return nil, err
	}
	own := 0
	for _, d := range g.Devices {
		if d.Own {
			own++
		}
	}
	if req.Verdict == VerdictNone {
		var missing []string
		if list, missing = judge.Clear(list, g.Key, devices); len(missing) > 0 {
			at, _ := judge.Find(list, g.Key, "")
			switch {
			case all && own > 0:
				return nil, fmt.Errorf("このかたまり（%s）に、全台への判定はありません。機器を指定した判定が %d 台にあります。取り消すには、機器を指定してください", g.ID, own)
			case all:
				return nil, fmt.Errorf("このかたまり（%s）に、判定はありません", g.ID)
			case at >= 0:
				return nil, fmt.Errorf("次の機器には、機器を指定した判定がありません（全台への判定が当たっています。全台の判定を取り消すか、その機器に別の判定を付けてください）: %s", strings.Join(missing, ", "))
			}
			return nil, fmt.Errorf("次の機器には、判定がありません: %s", strings.Join(missing, ", "))
		}
	} else {
		list = judge.Set(list, g.Key, devices, project.Judgment{
			Verdict: req.Verdict, Reason: req.Reason, By: req.By, At: now.Format(time.RFC3339),
		})
	}
	if err := e.Proj.SaveJudgments(list); err != nil {
		return nil, err
	}

	after, err := e.judging(src, view)
	if err != nil {
		return nil, err
	}
	out := &JudgeOutcome{Judging: after, Verdict: req.Verdict}
	for _, x := range after.Groups {
		if x.Key.Same(g.Key) {
			out.Group = x
		}
	}
	for _, d := range g.Devices {
		switch {
		case all && d.Own:
			out.Own++
		case all || has(devices, d.Device):
			out.Devices++
			out.Lines += d.Lines()
		}
	}
	// The shape of the lines stays out of the log (spec 6.7): the ID tells
	// the group.
	word := VerdictWord(req.Verdict)
	if req.Verdict == VerdictNone {
		word = "取り消し"
	}
	scope := "全台"
	if !all {
		scope = "機器を指定"
	}
	e.Proj.Log(now, "judge: %s %s %s（%s、機器 %d 台、%d 行、判定した人 %s）", req.Source, g.ID, word, scope, out.Devices, out.Lines, req.By)
	return out, nil
}

// sortedGroups gives the groups with what is open first; the order of the
// first lines stays within each half.
func sortedGroups(groups []*LineGroup) []*LineGroup {
	out := append([]*LineGroup(nil), groups...)
	sort.SliceStable(out, func(i, k int) bool { return out[i].Open() && !out[k].Open() })
	return out
}
