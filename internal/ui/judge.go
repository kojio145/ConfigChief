package ui

import (
	"fmt"
	"strings"

	"github.com/kojio145/configchief/internal/ops"
)

// headCap is how many lines of a device in a group the page gets.
const headCap = 100

// JudgeLine is an unhandled line a judgment is put on.
type JudgeLine struct {
	No    int    `json:"no"`
	Text  string `json:"text"`
	Below int    `json:"below"` // the lines that follow it
}

// JudgeDevice is what one device has of a group of unhandled lines.
type JudgeDevice struct {
	Device string      `json:"device"`
	File   string      `json:"file"`
	Lines  int         `json:"lines"` // with the lines that follow
	Heads  []JudgeLine `json:"heads"`
	More   int         `json:"more"` // the lines that are not in Heads
	// Verdict: "" (not judged) | "hold" | "manual" | "not_needed". Own tells
	// that the judgment names the device.
	Verdict string `json:"verdict"`
	Word    string `json:"word"`
	Reason  string `json:"reason"`
	By      string `json:"by"`
	At      string `json:"at"`
	Own     bool   `json:"own"`
}

// JudgeGroup is the unhandled lines of one shape (spec 8.5).
type JudgeGroup struct {
	ID      string        `json:"id"`
	Pack    string        `json:"pack"`
	Path    string        `json:"path"` // the lines above ("" = the top level)
	Shape   string        `json:"shape"`
	Heads   int           `json:"heads"`
	Below   int           `json:"below"`
	Open    bool          `json:"open"`
	Verdict string        `json:"verdict"` // in words
	Reason  string        `json:"reason"`
	By      string        `json:"by"`
	Devices []JudgeDevice `json:"devices"`
	// The example: the first line of the group with the lines below it.
	Example     JudgeLine `json:"example"`
	ExampleFile string    `json:"exampleFile"`
	ExampleMore int       `json:"exampleMore"`
	ExampleText []string  `json:"exampleText"`
}

// JudgeView is the unhandled lines of a side with their judgments (spec 14.2
// 読めなかった行).
type JudgeView struct {
	Source  string       `json:"source"`
	Title   string       `json:"title"`
	Message string       `json:"message"` // what the last step did
	Counts  []Pair       `json:"counts"`
	Notes   []string     `json:"notes"`
	Groups  []JudgeGroup `json:"groups"`
	Left    int          `json:"left"` // the lines not judged or on hold
	By      string       `json:"by"`   // the name the last judgment was given with
}

// Unhandled reads the configurations of a side again and gives their
// unhandled lines by shape, with the judgments. It writes nothing.
func (s *Session) Unhandled(source string) (*JudgeView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := checkSource(source); err != nil {
		return nil, err
	}
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	j, err := e.Review(source)
	if err != nil {
		return nil, err
	}
	s.judged(source, j)
	return s.judgeView(j, ""), nil
}

// Judge puts a judgment on a group of unhandled lines: verdict "manual",
// "not_needed" or "hold", or "none" to take it away. devices: "*" or "" =
// every device, else host names with commas between.
func (s *Session) Judge(source, group, devices, verdict, reason, by string) (*JudgeView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := checkSource(source); err != nil {
		return nil, err
	}
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	o, err := e.JudgeLines(ops.JudgeRequest{Source: source, Group: group, Devices: devices, Verdict: verdict, Reason: reason, By: by}, s.now())
	if err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(by); name != "" && name != s.name {
		s.name = name
		s.save()
	}
	s.judged(source, o.Judging)
	id := group
	if o.Group != nil {
		id = o.Group.ID
	}
	done := fmt.Sprintf("かたまり %s に「%s」を付けました（機器 %d 台、%d 行）", id, ops.VerdictWord(o.Verdict), o.Devices, o.Lines)
	if o.Verdict == ops.VerdictNone {
		done = fmt.Sprintf("かたまり %s の判定を取り消しました（機器 %d 台、%d 行）", id, o.Devices, o.Lines)
	}
	if o.Own > 0 {
		done += fmt.Sprintf("。機器を指定した判定が %d 台に残っています（その機器は変わりません）", o.Own)
	}
	return s.judgeView(o.Judging, done), nil
}

// judged lets the lines of the last import show the judgments as they are
// now. The counts and the report of that import stay as they were written.
func (s *Session) judged(source string, j *ops.Judging) {
	if o := s.imports[source]; o != nil {
		o.Judging = j
	}
}

func (s *Session) judgeView(j *ops.Judging, done string) *JudgeView {
	v := &JudgeView{
		Source: j.Source, Title: "読めなかった行（" + sourceWord(j.Source) + "）", Message: done,
		Notes: []string{}, Groups: []JudgeGroup{}, Left: j.Open + j.Hold, By: s.name,
	}
	v.Counts = []Pair{
		{"未判定", fmt.Sprint(j.Open) + " 行"},
		{"保留", fmt.Sprint(j.Hold) + " 行"},
		{"追加コンフィグで書く", fmt.Sprint(j.Manual) + " 行"},
		{"要らない", fmt.Sprint(j.NotNeeded) + " 行"},
		{"かたまり", fmt.Sprint(len(j.Groups)) + " 個"},
	}
	for _, m := range j.NoExtra {
		who := m.Device
		if m.Host != m.Device {
			who = fmt.Sprintf("%s（新ホスト名 %s）", m.Device, m.Host)
		}
		v.Notes = append(v.Notes, fmt.Sprintf("M-402: %s に「追加コンフィグで書く」の行が %d 行ありますが、extra に、この機器に当たる追加コンフィグ（_all.j2・@グループ名.j2・%s.j2）がありません", who, m.Lines, m.Host))
	}
	if j.HostsUnknown {
		v.Notes = append(v.Notes, "パラメータブックがまだ無いので、「追加コンフィグで書く」の機器に追加コンフィグがあるか（M-402）は見ていません")
	}
	if j.BookErr != "" {
		v.Notes = append(v.Notes, "パラメータブックを読めないため、グループ向けの追加コンフィグは見ていません: "+j.BookErr)
	}
	for _, x := range j.Stale {
		v.Notes = append(v.Notes, fmt.Sprintf("判定「%s」（パック %s、形 %s、範囲 %s）は、当たる行がありません（文面が変わったか、パックが対応しました）", ops.VerdictWord(x.Verdict), x.Pack, x.Shape, x.Devices))
	}
	for _, f := range j.Unread {
		v.Notes = append(v.Notes, f+" は取り込めなかったので、ここに入っていません（「取り込み」で理由が出ます）")
	}
	for _, g := range j.Groups {
		heads, below := g.Count()
		gv := JudgeGroup{
			ID: g.ID, Pack: g.Key.Pack, Path: g.PathText(), Shape: g.Key.Shape, Heads: heads, Below: below,
			Open: g.Open(), Verdict: g.VerdictText(), Reason: g.ReasonText(), By: g.ByText(),
			Devices:     []JudgeDevice{},
			Example:     JudgeLine{No: g.Example.Line, Text: g.Example.Text, Below: g.Example.Below},
			ExampleFile: g.Devices[0].File, ExampleMore: g.ExampleMore, ExampleText: strs(g.ExampleBelow),
		}
		for _, d := range g.Devices {
			dv := JudgeDevice{
				Device: d.Device, File: d.File, Lines: d.Lines(), Heads: []JudgeLine{},
				Verdict: d.Verdict, Word: ops.VerdictWord(d.Verdict), Reason: d.Reason, By: d.By, At: d.At, Own: d.Own,
			}
			for i, h := range d.Heads {
				if i == headCap {
					dv.More = len(d.Heads) - headCap
					break
				}
				dv.Heads = append(dv.Heads, JudgeLine{No: h.Line, Text: h.Text, Below: h.Below})
			}
			gv.Devices = append(gv.Devices, dv)
		}
		v.Groups = append(v.Groups, gv)
	}
	return v
}

// JudgeMany puts one judgment on several groups for every device (spec
// 14.19.8): the groups the list shows after it is narrowed.
func (s *Session) JudgeMany(source string, groups []string, verdict, reason, by string) (*JudgeView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := checkSource(source); err != nil {
		return nil, err
	}
	e, err := s.beginWrite()
	if err != nil {
		return nil, err
	}
	o, err := e.JudgeMany(source, groups, verdict, reason, by, s.now())
	if err != nil {
		return nil, err
	}
	if name := strings.TrimSpace(by); name != "" && name != s.name {
		s.name = name
		s.save()
	}
	s.judged(source, o.Judging)
	return s.judgeView(o.Judging, fmt.Sprintf("かたまり %d 個に「%s」を付けました（%d 行）", o.Groups, ops.VerdictWord(o.Verdict), o.Lines)), nil
}
