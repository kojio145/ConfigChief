package ui

import (
	"strings"

	"github.com/kojio145/configchief/internal/fill"
	"github.com/kojio145/configchief/internal/ops"
	"github.com/kojio145/configchief/internal/report"
)

// The change commands made from the parameter book (spec 9.8, 14.12).

// DeltaSideInfo is a side the configurations now may be taken from.
type DeltaSideInfo struct {
	Side  string `json:"side"`  // old | new | logs
	Title string `json:"title"` // input/old ...
	Files int    `json:"files"`
}

// DeltaView is what the part of the screen starts from: the sides with
// their number of files, and the devices of the book.
type DeltaView struct {
	Sides   []DeltaSideInfo `json:"sides"`
	Devices []string        `json:"devices"`
}

// Delta tells the sides and the devices to choose from.
func (s *Session) Delta() (*DeltaView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	v := &DeltaView{Sides: []DeltaSideInfo{}, Devices: []string{}}
	for _, side := range ops.FillSides {
		n := 0
		if ins, err := e.Proj.Inputs(side); err == nil {
			n = len(ins)
		}
		v.Sides = append(v.Sides, DeltaSideInfo{Side: side, Title: "input/" + side, Files: n})
	}
	hosts, err := e.BookHosts()
	if err != nil {
		return nil, err
	}
	v.Devices = strs(hosts)
	return v, nil
}

// DeltaDeviceView is the outcome of one device.
type DeltaDeviceView struct {
	Name    string `json:"name"`
	Result  string `json:"result"` // made | unchanged | not
	Word    string `json:"word"`   // the result in words
	Why     string `json:"why"`
	Now     string `json:"now"`     // the file of the configuration now
	NowHost string `json:"nowHost"` // the host name in it
	Pack    string `json:"pack"`
	File    string `json:"file"`
	Lines   int    `json:"lines"`
	// Text is the commands to put in, as they are in the file (the secrets
	// as they are: it is to be copied).
	Text string `json:"text"`
	// The lines, each "parent > … > line"; a removal with "　→ the command",
	// a line replaced with "　→ the line that replaces it", a line kept with
	// "　（why）". The secrets are hidden.
	Added    []string `json:"added"`
	Removed  []string `json:"removed"`
	Replaced []string `json:"replaced"`
	Kept     []string `json:"kept"`
}

// DeltaRunView is the outcome of making the changes.
type DeltaRunView struct {
	Side    string            `json:"side"`
	Devices []DeltaDeviceView `json:"devices"`
	// Table is the devices as a list (機器・結果・理由・足す・消す・置き換わる・残る・ファイル).
	Table     Table    `json:"table"`
	Notes     []string `json:"notes"`
	Dir       string   `json:"dir"`
	Report    string   `json:"report"` // the HTML report, from the case folder
	ReportXLS string   `json:"reportXlsx"`
	Made      int      `json:"made"`
	Unchanged int      `json:"unchanged"`
	NotMade   int      `json:"notMade"`
}

func deltaItems(e *ops.Env, d *ops.DeltaDevice, list []fill.Item, after func(fill.Item) string) []string {
	out := []string{}
	for _, it := range list {
		t := e.Shown(d, strings.Join(append(append([]string{}, it.Path...), it.Text), " > "))
		if after != nil {
			if a := after(it); a != "" {
				t += a
			}
		}
		out = append(out, t)
	}
	return out
}

// DeltaRun makes the change commands of the devices (all when devices is
// empty) against the configurations now of a side (old | new | logs).
func (s *Session) DeltaRun(side string, devices []string) (*DeltaRunView, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, err := s.begin()
	if err != nil {
		return nil, err
	}
	o, err := e.Delta(ops.DeltaOptions{Side: side, Devices: devices}, s.now())
	if err != nil {
		return nil, err
	}
	v := &DeltaRunView{Side: o.Side, Devices: []DeltaDeviceView{}, Notes: strs(o.Notes), Dir: o.Dir,
		Report: o.Report.HTML, ReportXLS: o.Report.XLSX, Made: o.Made(), NotMade: o.NotMade()}
	list := report.Sheet{Name: "機器", Header: []string{"機器", "結果", "理由", "足す", "消す", "置き換わる", "残る", "ファイル"}}
	for _, d := range o.Devices {
		r, word, mark := "not", "作らなかった", "unknown"
		switch {
		case d.Made:
			r, word, mark = "made", "作った", ""
		case d.Unchanged:
			r, word, mark = "unchanged", "変更なし", "ignored"
			v.Unchanged++
		}
		dv := DeltaDeviceView{Name: d.Name, Result: r, Word: word, Why: d.Why, Now: d.Now, NowHost: d.NowHost, Pack: d.Pack,
			File: d.File, Lines: d.Lines, Text: d.Text,
			Added: deltaItems(e, d, d.Added, nil),
			Removed: deltaItems(e, d, d.Removed, func(it fill.Item) string {
				if it.Cmd == "" {
					return ""
				}
				return "　→ " + e.Shown(d, it.Cmd)
			}),
			Replaced: deltaItems(e, d, d.Replaced, func(it fill.Item) string {
				if it.Why == "" {
					return ""
				}
				return "　→ " + e.Shown(d, it.Why)
			}),
			Kept: deltaItems(e, d, d.Kept, func(it fill.Item) string {
				if it.Why == "" {
					return ""
				}
				return "　（" + it.Why + "）"
			}),
		}
		v.Devices = append(v.Devices, dv)
		n := func(x []string) string {
			if r == "not" {
				return ""
			}
			return itoa(len(x))
		}
		list.Rows = append(list.Rows, []string{d.Name, word, d.Why, n(dv.Added), n(dv.Removed), n(dv.Replaced), n(dv.Kept), d.File})
		list.Marks = append(list.Marks, mark)
	}
	v.Table = table(list)
	return v, nil
}
