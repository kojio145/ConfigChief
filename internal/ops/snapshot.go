package ops

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kojio145/configchief/internal/check"
	"github.com/kojio145/configchief/internal/project"
	"github.com/kojio145/configchief/internal/report"
	"github.com/kojio145/configchief/internal/textdiff"
	"github.com/kojio145/configchief/internal/tmpl"
)

// The record of one generation, kept under state/snapshots (spec 12.4). It
// holds no secret value: a secret is kept as its seal, which still tells
// whether the value changed.
type snapFile struct {
	Version int    `json:"version"`
	At      string `json:"at"`
	Tool    string `json:"tool,omitempty"`
	// Counted tells that Errors and Warnings were kept: the findings not
	// approved at this generation (spec 14.12.1). Older records lack them.
	Counted  bool         `json:"counted,omitempty"`
	Errors   int          `json:"errors,omitempty"`
	Warnings int          `json:"warnings,omitempty"`
	Devices  []snapDevice `json:"devices"`
}

type snapDevice struct {
	Host  string `json:"host"` // "" = the rows that belong to no device
	State string `json:"state,omitempty"`
	// LinesAt is the generation the lines are of: this one, or an earlier one
	// if the device was not generated this time ("" = never generated).
	LinesAt string     `json:"lines_at,omitempty"`
	Lines   []string   `json:"lines,omitempty"`
	Cells   []snapCell `json:"cells,omitempty"`
}

type snapCell struct {
	Table  string `json:"table"`
	Key    string `json:"key,omitempty"`
	Column string `json:"column"`
	Value  string `json:"value"`
}

// CellChange is a cell of the design data that is not what it was at the
// generation before. Before "" = the cell was empty, After "" = it is now.
type CellChange struct {
	Device, Table, Key, Column string
	Before, After              string
}

// LineChange is a line of a configuration that was added or taken away since
// the generation before.
type LineChange struct {
	Device string
	Added  bool
	Line   int // in the new text for an added line, in the old one for a removed one
	Text   string
}

// Changes is what changed since the generation before (spec 12.4).
type Changes struct {
	Snapshot string // the record of this generation, from the case folder
	Before   string // the generation before ("" = this is the first)
	Cells    []CellChange
	Lines    []LineChange
	// NewDevices and GoneDevices are the devices that came and went;
	// FirstLines the devices generated for the first time; NotGenerated the
	// devices whose lines were not compared because there are none of this
	// time.
	NewDevices, GoneDevices, FirstLines, NotGenerated []string
	Report                                            Files
}

const snapshotDir = "state/snapshots"

// lastSnapshot reads the record of the generation before (nil = none).
func (e *Env) lastSnapshot() (*snapFile, string, error) {
	entries, err := os.ReadDir(e.Proj.Path(snapshotDir))
	if err != nil && !os.IsNotExist(err) {
		return nil, "", fmt.Errorf("%s を読めません: %v", snapshotDir, err)
	}
	var names []string
	for _, ent := range entries {
		if !ent.IsDir() && strings.HasSuffix(ent.Name(), ".json") {
			names = append(names, ent.Name())
		}
	}
	if len(names) == 0 {
		return nil, "", nil
	}
	sort.Strings(names)
	name := names[len(names)-1]
	data, err := os.ReadFile(e.Proj.Path(snapshotDir + "/" + name))
	if err != nil {
		return nil, "", fmt.Errorf("%s/%s を読めません: %v", snapshotDir, name, err)
	}
	var f snapFile
	if err := json.Unmarshal(data, &f); err != nil || f.Version != 1 {
		return nil, name, fmt.Errorf("%s/%s を読めません（形式が違います）", snapshotDir, name)
	}
	return &f, name, nil
}

// cells gives the cells of the design data by device, in the order of the
// table definitions.
func (e *Env) snapCells(o *GenerateOutcome) map[string][]snapCell {
	out := map[string][]snapCell{}
	for _, t := range e.Schema.Tables {
		tab := o.data.Tables[t.Name]
		if tab == nil {
			continue
		}
		type keyed struct {
			host, key string
			cells     []snapCell
		}
		var rows []keyed
		for _, r := range tab.Rows {
			k := keyed{key: check.RefOf(t, r, "").Key}
			if dc := t.DeviceColumn(); dc != "" {
				k.host = r.Get(dc)
			}
			for _, c := range t.Columns {
				if v := check.Shown(c, r.Cells[c.Name]); v != "" {
					k.cells = append(k.cells, snapCell{Table: t.Name, Key: k.key, Column: c.Name, Value: v})
				}
			}
			rows = append(rows, k)
		}
		sort.SliceStable(rows, func(i, j int) bool { return tmpl.NaturalLess(rows[i].key, rows[j].key) })
		for _, k := range rows {
			out[k.host] = append(out[k.host], k.cells...)
		}
	}
	return out
}

func cellID(c snapCell) string { return c.Table + "\x00" + c.Key + "\x00" + c.Column }

// snapshot keeps the record of this generation and compares it with the one
// before.
func (e *Env) snapshot(o *GenerateOutcome, now time.Time) (*Changes, error) {
	prev, prevName, err := e.lastSnapshot()
	if err != nil {
		if prevName == "" {
			return nil, err
		}
		// A record that cannot be read is told about and left alone.
		o.Notes = append(o.Notes, err.Error()+"。前回との違いは出していません。")
		prev = nil
	}
	stamp := project.Stamp(now)
	before := map[string]*snapDevice{}
	if prev != nil {
		for i := range prev.Devices {
			before[prev.Devices[i].Host] = &prev.Devices[i]
		}
	}

	cells := e.snapCells(o)
	c := o.Count()
	cur := snapFile{Version: 1, At: stamp, Tool: e.Version, Counted: true, Errors: c.Errors, Warnings: c.Warnings}
	if len(cells[""]) > 0 {
		cur.Devices = append(cur.Devices, snapDevice{Cells: cells[""]})
	}
	for _, d := range o.Devices {
		sd := snapDevice{Host: d.Host, State: d.State, Cells: cells[d.Host]}
		switch {
		case d.Output != nil:
			sd.LinesAt = stamp
			for i := range d.Output.Lines {
				sd.Lines = append(sd.Lines, d.Output.Sealed(i+1))
			}
		case before[d.Host] != nil:
			sd.LinesAt, sd.Lines = before[d.Host].LinesAt, before[d.Host].Lines
		}
		cur.Devices = append(cur.Devices, sd)
	}

	ch := &Changes{}
	if prev != nil {
		ch.Before = strings.TrimSuffix(prevName, ".json")
		seen := map[string]bool{}
		for i := range cur.Devices {
			d := &cur.Devices[i]
			seen[d.Host] = true
			b := before[d.Host]
			if b == nil {
				if d.Host != "" {
					ch.NewDevices = append(ch.NewDevices, d.Host)
					continue
				}
				b = &snapDevice{}
			}
			old := map[string]string{}
			for _, c := range b.Cells {
				old[cellID(c)] = c.Value
			}
			have := map[string]bool{}
			for _, c := range d.Cells {
				have[cellID(c)] = true
				if v, ok := old[cellID(c)]; !ok || v != c.Value {
					ch.Cells = append(ch.Cells, CellChange{Device: d.Host, Table: c.Table, Key: c.Key, Column: c.Column, Before: v, After: c.Value})
				}
			}
			for _, c := range b.Cells {
				if !have[cellID(c)] {
					ch.Cells = append(ch.Cells, CellChange{Device: d.Host, Table: c.Table, Key: c.Key, Column: c.Column, Before: c.Value})
				}
			}
			switch {
			case d.Host == "":
			case d.LinesAt != stamp:
				ch.NotGenerated = append(ch.NotGenerated, d.Host)
			case b.LinesAt == "":
				ch.FirstLines = append(ch.FirstLines, d.Host)
			default:
				diff := textdiff.Diff(strings.Join(b.Lines, "\n"), strings.Join(d.Lines, "\n"))
				for _, l := range diff.Lines {
					switch l.Op {
					case textdiff.Add:
						ch.Lines = append(ch.Lines, LineChange{Device: d.Host, Added: true, Line: l.BLine, Text: l.Text})
					case textdiff.Del:
						ch.Lines = append(ch.Lines, LineChange{Device: d.Host, Line: l.ALine, Text: l.Text})
					}
				}
			}
		}
		for _, d := range prev.Devices {
			if !seen[d.Host] && d.Host != "" {
				ch.GoneDevices = append(ch.GoneDevices, d.Host)
			}
		}
	}

	data, err := json.MarshalIndent(&cur, "", " ")
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(e.Proj.Path(snapshotDir), 0o755); err != nil {
		return nil, fmt.Errorf("%s を作れません: %v", snapshotDir, err)
	}
	rel := snapshotDir + "/" + stamp + ".json"
	// A second generation within the same second gets a number.
	for n := 2; ; n++ {
		if _, err := os.Stat(e.Proj.Path(rel)); err != nil {
			break
		}
		rel = fmt.Sprintf("%s/%s_%d.json", snapshotDir, stamp, n)
	}
	if err := os.WriteFile(e.Proj.Path(rel), append(data, '\n'), 0o644); err != nil {
		return nil, fmt.Errorf("%s を書けません: %v", rel, err)
	}
	ch.Snapshot = rel
	return ch, nil
}

// changesDoc builds the list of changes (spec 12.4, 13.5).
func (e *Env) changesDoc(o *GenerateOutcome, now time.Time) *report.Doc {
	ch := o.Changes
	d := &report.Doc{Title: "変更一覧（前回の生成との違い）", Facts: e.facts(now)}
	d.Facts = append(d.Facts,
		report.Pair{Label: "前回の生成", Value: ch.Before},
		report.Pair{Label: "今回の控え", Value: ch.Snapshot})
	added, removed := 0, 0
	devs := map[string]bool{}
	for _, l := range ch.Lines {
		devs[l.Device] = true
		if l.Added {
			added++
		} else {
			removed++
		}
	}
	d.Counts = []report.Pair{
		{Label: "変わったセル", Value: num(len(ch.Cells))},
		{Label: "増えたコンフィグの行", Value: num(added)},
		{Label: "減ったコンフィグの行", Value: num(removed)},
		{Label: "コンフィグの行が変わった機器", Value: num(len(devs))},
		{Label: "増えた機器", Value: num(len(ch.NewDevices))},
		{Label: "減った機器", Value: num(len(ch.GoneDevices))},
	}
	list := func(label string, hosts []string) {
		if len(hosts) > 0 {
			d.Notes = append(d.Notes, fmt.Sprintf("%s（%d 台）: %s", label, len(hosts), strings.Join(hosts, ", ")))
		}
	}
	list("前回になかった機器。セルと行は一覧に出していません", ch.NewDevices)
	list("前回あって今回ない機器", ch.GoneDevices)
	list("初めて生成した機器。コンフィグの行は一覧に出していません", ch.FirstLines)
	list("今回生成していない機器。コンフィグの行は比べていません（セルは比べています）", ch.NotGenerated)
	d.Notes = append(d.Notes,
		"セルは、パラメータブックの * と @グループ、「共通値」を機器ごとに広げた後の値で比べています。パックの既定値は入っていません。",
		"秘密の値は、控えに値そのものを残していません。「伏せ字 #…」の記号が変わっていれば、値が変わっています。")

	cells := report.Sheet{Name: "変わったセル", Lead: "前回の生成のときと値が違うセルです。",
		Header: []string{"機器", "表", "行", "列", "前回", "今回", "種別"}}
	for _, c := range ch.Cells {
		kind := "変更"
		switch {
		case c.Before == "":
			kind = "追加"
		case c.After == "":
			kind = "削除"
		}
		host := c.Device
		if host == "" {
			host = "（案件全体）"
		}
		cells.Rows = append(cells.Rows, []string{host, e.tableTitle(c.Table), e.keyLabel(c.Table, c.Key), e.columnTitle(c.Table, c.Column), c.Before, c.After, kind})
	}
	lines := report.Sheet{Name: "変わった行", Lead: "前回の生成のときと違うコンフィグの行です。行番号は、増えた行は今回の文面、減った行は前回の文面のものです。",
		Header: []string{"機器", "種別", "行番号", "文面"}, Code: []int{3}}
	for _, l := range ch.Lines {
		kind := "減った行"
		if l.Added {
			kind = "増えた行"
		}
		lines.Rows = append(lines.Rows, []string{l.Device, kind, num(l.Line), l.Text})
	}
	d.Sheets = []report.Sheet{cells, lines}
	return d
}
