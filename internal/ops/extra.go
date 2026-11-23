package ops

import (
	"errors"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/tmpl"
)

// The extra configurations written on the screen (spec 4.2, 14.11.3).

// ExtraEntry is one file of extra/.
type ExtraEntry struct {
	Name string `json:"name"` // the file name, "_all.j2" and so on
	// Kind is what the name says it goes to: "all", "group", "device".
	Kind    string `json:"kind"`
	Target  string `json:"target"`  // the group or the host name ("" for all)
	Applies bool   `json:"applies"` // a device of the book takes it
}

// ExtraPlaces is what the screen of the extra configurations chooses from.
type ExtraPlaces struct {
	Files     []ExtraEntry `json:"files"`
	Devices   []string     `json:"devices"`   // the host names of the book
	Groups    []string     `json:"groups"`    // the groups of the book
	Positions []string     `json:"positions"` // the sections of the packs
}

// ExtraName gives the file name of an extra configuration for a target:
// kind "all", "group" (target = the group) or "device" (target = the host
// name).
func ExtraName(kind, target string) (string, error) {
	target = strings.TrimSpace(target)
	switch kind {
	case "all":
		return "_all.j2", nil
	case "group":
		target = strings.TrimPrefix(target, "@")
		if target == "" {
			return "", fmt.Errorf("グループを選んでください")
		}
		target = "@" + target
	case "device":
		if target == "" {
			return "", fmt.Errorf("機器を選んでください")
		}
	default:
		return "", fmt.Errorf("当てる先「%s」はありません（all・group・device）", kind)
	}
	if strings.ContainsAny(target, badNameChars) || reservedNames[strings.ToUpper(target)] || strings.HasPrefix(target, ".") || strings.HasPrefix(target, "~$") {
		return "", fmt.Errorf("「%s」は、ファイルの名前に使えません", target)
	}
	return target + ".j2", nil
}

func extraKind(name string) (kind, target string) {
	base := strings.TrimSuffix(name, ".j2")
	switch {
	case base == "_all":
		return "all", ""
	case strings.HasPrefix(base, "@"):
		return "group", base[1:]
	}
	return "device", base
}

// ExtraPlaces lists the extra configurations and what a new one can go to.
func (e *Env) ExtraPlaces() (*ExtraPlaces, error) {
	out := &ExtraPlaces{Files: []ExtraEntry{}, Devices: []string{}, Groups: []string{}, Positions: []string{}}
	_, bk, err := e.readBook()
	if err != nil {
		return nil, err
	}
	ds := bk
	names, err := e.extraNames()
	if err != nil {
		return nil, err
	}
	groups := map[string]bool{}
	if ds != nil {
		if t := ds.Data.Tables["devices"]; t != nil {
			for _, r := range t.Rows {
				h := r.Get("hostname")
				if h == "" || h == "*" || strings.HasPrefix(h, "@") {
					continue
				}
				out.Devices = append(out.Devices, h)
				for _, g := range groupsOf(r) {
					if g = strings.TrimSpace(g); g != "" && !groups[g] {
						groups[g] = true
						out.Groups = append(out.Groups, g)
					}
				}
			}
		}
	}
	sort.Strings(out.Groups)
	for _, n := range names {
		name := n + ".j2"
		kind, target := extraKind(name)
		x := ExtraEntry{Name: name, Kind: kind, Target: target}
		switch kind {
		case "all":
			x.Applies = len(out.Devices) > 0
		case "group":
			x.Applies = groups[target]
		default:
			for _, d := range out.Devices {
				if d == target {
					x.Applies = true
				}
			}
		}
		out.Files = append(out.Files, x)
	}
	seen := map[string]bool{}
	for _, id := range e.PackIDs {
		p := e.Packs[id]
		if p == nil {
			continue
		}
		for _, s := range p.Sections {
			if !seen[s.ID] && s.ID != "extra" {
				seen[s.ID] = true
				out.Positions = append(out.Positions, s.ID)
			}
		}
	}
	return out, nil
}

// checkExtraName tells whether a name is one of an extra configuration.
func checkExtraName(name string) error {
	if !strings.HasSuffix(name, ".j2") || strings.ContainsAny(name, `/\`) || name == ".j2" || strings.HasPrefix(name, ".") || strings.HasPrefix(name, "~$") {
		return fmt.Errorf("追加コンフィグの名前「%s」は使えません（_all.j2・@グループ名.j2・ホスト名.j2）", name)
	}
	return nil
}

// ReadExtra gives the text of an extra configuration.
func (e *Env) ReadExtra(name string) (string, error) {
	if err := checkExtraName(name); err != nil {
		return "", err
	}
	data, err := os.ReadFile(e.Proj.Path("extra/" + name))
	if err != nil {
		return "", fmt.Errorf("extra/%s を読めません: %v", name, err)
	}
	return strings.ReplaceAll(project.Decode(data), "\r\n", "\n"), nil
}

// ExtraSaved is what SaveExtra did.
type ExtraSaved struct {
	Name   string   `json:"name"`
	Backup string   `json:"backup"` // the copy of the file as it was ("" = it was not there)
	Notes  []string `json:"notes"`
}

// SaveExtra writes an extra configuration (spec 14.11.3). A text that is no
// template is not written: the fault comes back with the line. The file as
// it was is copied to state/backup first.
func (e *Env) SaveExtra(name, text string, now time.Time) (*ExtraSaved, error) {
	if err := checkExtraName(name); err != nil {
		return nil, err
	}
	text = strings.ReplaceAll(text, "\r\n", "\n")
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	body, offset := text, 0
	first, rest, _ := strings.Cut(text, "\n")
	out := &ExtraSaved{Name: name, Notes: []string{}}
	places, err := e.ExtraPlaces()
	if err != nil {
		return nil, err
	}
	if m := extraPosition.FindStringSubmatch(first); m != nil {
		body, offset = rest, 1
		known := false
		for _, p := range places.Positions {
			known = known || p == m[2]
		}
		if !known {
			return nil, fmt.Errorf("1 行目: 位置「%s」はどのパックにもありません（%s）", m[2], strings.Join(places.Positions, "・"))
		}
	} else if strings.Contains(first, "position:") {
		return nil, fmt.Errorf("1 行目: 位置は {# position: after <節> #} か {# position: before <節> #} と書きます")
	}
	if len(e.PackIDs) > 0 {
		if p := e.Packs[e.PackIDs[0]]; p != nil {
			if err := p.Engine().Check("extra/"+name, body); err != nil {
				msg := err.Error()
				if te, ok := asTmplError(err); ok && te.Line > 0 {
					msg = fmt.Sprintf("%d 行目: %s", te.Line+offset, te.Msg)
				} else if m := reEngineLine.FindStringSubmatchIndex(msg); m != nil {
					n, _ := strconv.Atoi(msg[m[2]:m[3]])
					msg = fmt.Sprintf("%d 行目: %s", n+offset, strings.TrimSpace(msg[:m[0]]+msg[m[1]:]))
				}
				return nil, fmt.Errorf("テンプレートとして読めないので、保存していません。%s（コンフィグの行に {{・{%%・{# があるときは、{%% raw %%} と {%% endraw %%} で囲みます）", msg)
			}
		}
	}
	rel := "extra/" + name
	if old, err := os.ReadFile(e.Proj.Path(rel)); err == nil {
		if string(old) == text {
			return out, nil
		}
		if out.Backup, err = e.copyToBackup(rel, "extra", old, now); err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(e.Proj.Path("extra"), 0o755); err != nil {
		return nil, fmt.Errorf("extra を作れません: %v", err)
	}
	if err := os.WriteFile(e.Proj.Path(rel), []byte(text), 0o644); err != nil {
		return nil, fmt.Errorf("%s を書けません: %v", rel, err)
	}
	for _, f := range places.Files {
		if f.Name == name && !f.Applies {
			out.Notes = append(out.Notes, fmt.Sprintf("%s は、パラメータブックのどの機器にも当たりません（生成でもお知らせが出ます）", rel))
		}
	}
	found := false
	for _, f := range places.Files {
		found = found || f.Name == name
	}
	if !found {
		kind, target := extraKind(name)
		if kind == "device" && !contains(places.Devices, target) || kind == "group" && !contains(places.Groups, target) {
			out.Notes = append(out.Notes, fmt.Sprintf("%s は、パラメータブックのどの機器にも当たりません（生成でもお知らせが出ます）", rel))
		}
	}
	e.Proj.Log(now, "追加コンフィグ %s を書きました（控え %s）", rel, out.Backup)
	return out, nil
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// DeleteExtra moves an extra configuration to state/backup.
func (e *Env) DeleteExtra(name string, now time.Time) (string, error) {
	if err := checkExtraName(name); err != nil {
		return "", err
	}
	to, err := e.backup("extra/"+name, "extra", now)
	if err != nil {
		return "", err
	}
	e.Proj.Log(now, "追加コンフィグ extra/%s を消しました（%s に移しました）", name, to)
	return to, nil
}

// ExtraDraft is the lines of a group of unhandled lines of one device, put
// as the end of the extra configuration of the device (spec 14.11.3).
type ExtraDraft struct {
	Kind   string `json:"kind"` // "device"
	Target string `json:"target"`
	Name   string `json:"name"`
	Text   string `json:"text"` // the file as it is, with the lines added at the end
	Lines  int    `json:"lines"`
}

// DraftExtra gives the extra configuration of the device (the new host
// name of 新旧対照表（機器）, or the host name itself) with the lines of the group
// added at its end. It writes nothing.
func (e *Env) DraftExtra(source, group, device string) (*ExtraDraft, error) {
	if source == "" {
		source = "old"
	}
	src, err := e.importSide(source, "")
	if err != nil {
		return nil, err
	}
	j, err := e.judging(src, e.sideView(src))
	if err != nil {
		return nil, err
	}
	var g *LineGroup
	for _, x := range j.Groups {
		if x.ID == group {
			g = x
		}
	}
	if g == nil {
		return nil, fmt.Errorf("読めなかった行のかたまり %s がありません（取り込み直したときは、画面を読み直してください）", group)
	}
	var gd *GroupDevice
	for _, d := range g.Devices {
		if d.Device == device {
			gd = d
		}
	}
	if gd == nil {
		return nil, fmt.Errorf("かたまり %s に、機器 %s の行がありません", group, device)
	}
	f := src.File(device)
	if f == nil || f.Result == nil {
		return nil, fmt.Errorf("機器 %s のコンフィグを読めません", device)
	}
	target, oldVendor, newVendor := device, f.Pack, f.Pack
	if source == "old" {
		if _, bk, err := e.readBook(); err == nil && bk != nil {
			for _, d := range bk.Devices {
				if d.OldHost == device && d.NewHost != "" {
					target = d.NewHost
					if d.NewVendor != "" {
						newVendor = d.NewVendor
					}
				}
			}
		}
	}
	name, err := ExtraName("device", target)
	if err != nil {
		return nil, err
	}
	text := ""
	if old, err := os.ReadFile(e.Proj.Path("extra/" + name)); err == nil {
		text = strings.ReplaceAll(project.Decode(old), "\r\n", "\n")
		if text != "" && !strings.HasSuffix(text, "\n") {
			text += "\n"
		}
	}
	lines := f.Result.Lines
	byNo := map[int]int{}
	for i, l := range lines {
		byNo[l.Line] = i
	}
	var b strings.Builder
	n := 0
	for _, hd := range gd.Heads {
		note := fmt.Sprintf("{# %s %s の読めなかった行（%s %d 行目〜）", map[string]string{"old": "旧", "new": "新"}[source], device, f.Path, hd.Line)
		if oldVendor != newVendor && newVendor != "" {
			note += "。新しいメーカーの書き方に直してください"
		}
		i, ok := byNo[hd.Line]
		if !ok {
			continue
		}
		// The lines above it (an interface, say), top first.
		var above []string
		for p := lines[i].Parent; p > 0; {
			k, ok := byNo[p]
			if !ok {
				break
			}
			above = append([]string{lines[k].Text}, above...)
			p = lines[k].Parent
		}
		if len(above) > 0 {
			note += "。上の行の名前（IF 名など）は、新しい機器の名前に直してください"
		}
		b.WriteString(note + " #}\n")
		for _, a := range above {
			b.WriteString(a + "\n")
		}
		in := map[int]bool{hd.Line: true}
		for _, l := range lines[i:] {
			if l.Line == hd.Line || in[l.Parent] {
				in[l.Line] = true
				b.WriteString(l.Text + "\n")
				n++
			}
		}
	}
	if text != "" && !strings.HasSuffix(text, "\n\n") {
		text += "\n"
	}
	return &ExtraDraft{Kind: "device", Target: target, Name: name, Text: text + b.String(), Lines: n}, nil
}

// extraPosition is the first line that tells where an extra configuration
// goes (spec 9.4), as the renderer reads it.
var extraPosition = regexp.MustCompile(`^\{#\s*position:\s*(after|before)\s+(\S+)\s*#\}\s*$`)

func asTmplError(err error) (*tmpl.Error, bool) {
	var te *tmpl.Error
	ok := errors.As(err, &te)
	return te, ok
}

// groupsOf gives the groups a row of the devices names.
func groupsOf(r *model.Row) []string {
	if c := r.Cells["group"]; c != nil {
		return c.Values
	}
	return nil
}

// reEngineLine is where the template engine tells the line of a fault.
var reEngineLine = regexp.MustCompile(` ?\(line (\d+)\)`)
