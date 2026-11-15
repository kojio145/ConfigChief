package book

import (
	"fmt"
	"sort"
	"strings"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
)

// Groups gives the groups of each device of the book, sorted by name.
func Groups(ds *model.Dataset) map[string][]string {
	out := map[string][]string{}
	t := ds.Tables["devices"]
	if t == nil {
		return out
	}
	for _, r := range t.Rows {
		host := r.Get("hostname")
		if host == "" {
			continue
		}
		var gs []string
		if c := r.Cells["group"]; c != nil {
			gs = append(gs, c.Values...)
		}
		sort.Strings(gs)
		out[host] = gs
	}
	return out
}

// Hosts gives the host names of the book, in the order of the sheet.
func Hosts(ds *model.Dataset) []string {
	var out []string
	if t := ds.Tables["devices"]; t != nil {
		for _, r := range t.Rows {
			if h := r.Get("hostname"); h != "" {
				out = append(out, h)
			}
		}
	}
	return out
}

func has(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

// pick is a row of the book chosen for a device.
type pick struct {
	row   *model.Row
	rank  int // 0 = written for the host, 1 = for a group, 2 = for every device
	group string
}

func (p pick) before(q pick) bool {
	if p.rank != q.rank {
		return p.rank < q.rank
	}
	return p.group < q.group
}

// Expand gives the design data of each device (spec 9.6): a row written for
// "*" or "@group" becomes a row of every device it stands for, where the same
// key is written in several ways one row wins (host, then group, then "*"),
// and the common values go into the cells that are still empty. The rows of
// the book are not changed.
func (b *Book) Expand(sc *schema.Set) (*model.Dataset, []Problem) {
	var problems []Problem
	problem := func(kind string, o model.Origin, format string, a ...any) {
		problems = append(problems, Problem{Sheet: o.Sheet, Addr: o.Addr, Kind: kind, Text: fmt.Sprintf(format, a...)})
	}
	out := model.NewDataset(b.Data.Source)
	hosts := Hosts(b.Data)
	groups := Groups(b.Data)
	members := map[string][]string{}
	for _, h := range hosts {
		for _, g := range groups[h] {
			members[g] = append(members[g], h)
		}
	}

	for _, t := range sc.Tables {
		src := b.Data.Tables[t.Name]
		if src == nil {
			continue
		}
		if t.DeviceColumn() != "device" {
			for _, r := range src.Rows {
				out.Adopt(clone(r, ""))
			}
			continue
		}
		type slot struct{ host, rest string }
		best := map[slot]pick{}
		var order []slot
		for _, r := range src.Rows {
			dev := r.Get("device")
			var targets []string
			p := pick{row: r}
			switch {
			case dev == "":
				continue // V-101 tells about it
			case dev == "*":
				targets, p.rank = hosts, 2
			case strings.HasPrefix(dev, "@"):
				p.rank, p.group = 1, dev[1:]
				targets = members[p.group]
				if len(targets) == 0 {
					problem("V-107", r.Origins[0], "グループ「%s」に入っている機器がありません", p.group)
				}
			default:
				if !has(hosts, dev) {
					problem("V-107", r.Origins[0], "ホスト名「%s」は機器のシートにありません", dev)
					continue
				}
				targets = []string{dev}
			}
			rest := restKey(r)
			for _, h := range targets {
				s := slot{h, rest}
				old, ok := best[s]
				if !ok {
					order = append(order, s)
				}
				if !ok || p.before(old) {
					best[s] = p
				}
			}
		}
		// The rows of a device stay together, in the order of the sheet.
		sort.SliceStable(order, func(i, j int) bool { return index(hosts, order[i].host) < index(hosts, order[j].host) })
		for _, s := range order {
			out.Adopt(clone(best[s].row, s.host))
		}
	}

	b.common(sc, out, hosts, groups, members, problem)
	return out, problems
}

func index(list []string, s string) int {
	for i, x := range list {
		if x == s {
			return i
		}
	}
	return len(list)
}

// restKey is the key of a row without its device.
func restKey(r *model.Row) string {
	var sb strings.Builder
	for _, k := range r.Key {
		if k.Name != "device" {
			sb.WriteString(k.Name + "=" + k.Value + "\x00")
		}
	}
	return sb.String()
}

// clone copies a row; with a host, the copy belongs to that device.
func clone(r *model.Row, host string) *model.Row {
	c := &model.Row{Table: r.Table, Cells: map[string]*model.Cell{}, Origins: r.Origins, Extra: r.Extra}
	for name, cell := range r.Cells {
		cp := *cell
		cp.Values = append([]string{}, cell.Values...)
		c.Cells[name] = &cp
	}
	for _, k := range r.Key {
		if k.Name == "device" && host != "" {
			k.Value = host
			c.Cells["device"].Values = []string{host}
		}
		c.Key = append(c.Key, k)
	}
	return c
}

// common puts the values of the sheet 共通値 into the empty cells: those of
// the groups of a device first (by name), then those for every device.
func (b *Book) common(sc *schema.Set, ds *model.Dataset, hosts []string, groups, members map[string][]string, problem func(string, model.Origin, string, ...any)) {
	sheetName := ""
	if info := b.Sheets["@common"]; info != nil {
		sheetName = info.Name
	}
	type entry struct {
		c      Common
		group  string // "" = every device
		table  *schema.Table
		column *schema.Column
		values []string
		origin model.Origin
	}
	var entries []entry
	for _, c := range b.Common {
		o := model.Origin{Kind: "common", File: b.File, Sheet: sheetName, Addr: addr(0, c.Row-1)}
		if c.Scope == "" && c.Table == "" && c.Column == "" && c.Value == "" {
			continue
		}
		e := entry{c: c, origin: o}
		switch {
		case c.Scope == "*":
		case strings.HasPrefix(c.Scope, "@") && len(c.Scope) > 1:
			e.group = c.Scope[1:]
			if len(members[e.group]) == 0 {
				problem("V-107", o, "グループ「%s」に入っている機器がありません", e.group)
				continue
			}
		default:
			problem("V-107", o, "範囲「%s」は読めません（* か @グループ名 を書きます）", c.Scope)
			continue
		}
		for _, t := range sc.Tables {
			if foldName(c.Table) == foldName(t.Name) || foldName(c.Table) == foldName(t.Title.JA) || foldName(c.Table) == foldName(t.Title.EN) {
				e.table = t
				break
			}
		}
		if e.table == nil || e.table.DeviceColumn() == "" {
			problem("V-106", o, "表「%s」はありません", c.Table)
			continue
		}
		for _, col := range e.table.Columns {
			if foldName(c.Column) == foldName(col.Name) || foldName(c.Column) == foldName(col.Title.JA) || foldName(c.Column) == foldName(col.Title.EN) {
				e.column = col
				break
			}
		}
		switch {
		case e.column == nil:
			problem("V-106", o, "表「%s」に列「%s」はありません", c.Table, c.Column)
			continue
		case e.table.IsKey(e.column.Name):
			problem("V-106", o, "列「%s」は行を見分ける列なので、共通値にできません", c.Column)
			continue
		}
		bad := false
		for _, item := range split(e.column, c.Value) {
			v, err := e.column.Normalize(item)
			if err != nil {
				problem("V-102", o, "%s", err)
				bad = true
				break
			}
			if v != "" {
				e.values = append(e.values, v)
			}
		}
		if bad || len(e.values) == 0 {
			continue
		}
		entries = append(entries, e)
	}
	// Groups by name, then the ones for every device; the first to reach an
	// empty cell fills it.
	sort.SliceStable(entries, func(i, j int) bool {
		gi, gj := entries[i].group, entries[j].group
		if (gi == "") != (gj == "") {
			return gi != ""
		}
		return gi < gj
	})
	for _, e := range entries {
		t := ds.Tables[e.table.Name]
		if t == nil {
			continue
		}
		devCol := e.table.DeviceColumn()
		for _, r := range t.Rows {
			if r.Get(e.column.Name) != "" {
				continue
			}
			if e.group != "" && !has(groups[r.Get(devCol)], e.group) {
				continue
			}
			r.Cells[e.column.Name] = &model.Cell{Values: append([]string{}, e.values...), List: e.column.List, Origins: []model.Origin{e.origin}}
		}
	}
}

// Targets gives the devices a "device" cell stands for: the host itself,
// the members of "@group", every host for "*". A name that is not in the
// book stands for no device.
func Targets(ds *model.Dataset, scope string) []string {
	hosts := Hosts(ds)
	switch {
	case scope == "*":
		return hosts
	case strings.HasPrefix(scope, "@"):
		var out []string
		groups := Groups(ds)
		for _, h := range hosts {
			if has(groups[h], scope[1:]) {
				out = append(out, h)
			}
		}
		return out
	case has(hosts, scope):
		return []string{scope}
	}
	return nil
}
