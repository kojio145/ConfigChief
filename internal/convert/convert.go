package convert

import (
	"sort"
	"strings"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
)

// BasisSameName is the basis written for a proposal taken from the new
// configuration.
const BasisSameName = "新コンフィグに同じ名前の IF があります"

// Propose gives the rows for 機器対応 and IF 対応 (spec 10.2): one per old
// device, and one per interface of every old device whose new host name is
// known. have is what the book holds already; rows for the same device or
// interface may come back again and are left out when the book is written.
//
// The tool does not guess names: a new host name is proposed only when the
// new side (nil = there is none) has a device of the same name, and a new
// interface name only when that device has an interface of the same name.
func Propose(old, new *model.Dataset, have []book.DeviceMap) (dev []book.DeviceMap, ifs []book.IfMap) {
	known := map[string]book.DeviceMap{}
	for _, d := range have {
		if _, dup := known[d.OldHost]; !dup {
			known[d.OldHost] = d
		}
	}
	find := func(table string, key ...string) *model.Row {
		if new == nil {
			return nil
		}
		var kvs []model.KV
		for i := 0; i < len(key); i += 2 {
			kvs = append(kvs, model.KV{Name: key[i], Value: key[i+1]})
		}
		return new.Tables[table].Find(kvs)
	}
	for _, r := range rows(old, "devices") {
		host := r.Get("hostname")
		d, ok := known[host]
		if !ok {
			d = book.DeviceMap{OldHost: host, OldVendor: r.Get("vendor")}
			if n := find("devices", "hostname", host); n != nil {
				d.NewHost, d.NewVendor = host, n.Get("vendor")
				d.NewModel, d.NewOS = n.Get("model"), n.Get("os_version")
			}
			known[host] = d
			dev = append(dev, d)
		}
		if d.NewHost == "" {
			continue
		}
		for _, i := range rows(old, "interfaces") {
			if i.Get("device") != host {
				continue
			}
			m := book.IfMap{NewHost: d.NewHost, OldIf: i.Get("name")}
			if find("interfaces", "device", d.NewHost, "name", m.OldIf) != nil {
				m.NewIf, m.Basis = m.OldIf, BasisSameName
			}
			ifs = append(ifs, m)
		}
	}
	return dev, ifs
}

func rows(d *model.Dataset, table string) []*model.Row {
	if d == nil || d.Tables[table] == nil {
		return nil
	}
	return d.Tables[table].Rows
}

// Note is something Apply did that a person should know of.
type Note struct {
	Device string // the old host name
	Table  string
	Key    string // the key of the row as the old side has it
	Column string
	Value  string // the interface name in question
	Lines  []model.Origin
	Row    *model.Row // the row as the old side has it (nil for a note on an interface of the new side)
}

// Applied is the old design data with the mapping put on (spec 10.3).
type Applied struct {
	Data *model.Dataset
	// Hosts gives the new host name for an old one (only the devices that
	// are carried over).
	Hosts map[string]string
	// Unmapped lists the old devices that are not carried over: no row in
	// 機器対応, or no new host name. NoReason are those of them without a
	// note that tells why (M-407).
	Unmapped []string
	NoReason []string
	// Undecided lists the interface names that stayed as they were: the new
	// name is empty, or IF 対応 has no row for them (M-406).
	Undecided []Note
	// Dropped lists the rows left out because their interface is "-", and
	// Cleared the cells emptied for the same reason.
	Dropped []Note
	Cleared []Note
	// Clashes lists the rows left out because a row with the same key was
	// there already, and Shared the new interfaces that several old ones
	// point to (M-408).
	Clashes []Note
	Shared  []Note
}

// Apply rewrites the old design data for the new devices: host names by
// 機器対応, the values of every interface column by IF 対応.
func Apply(sc *schema.Set, old *model.Dataset, dev []book.DeviceMap, ifs []book.IfMap) *Applied {
	return apply(sc, old, dev, ifs, false)
}

// ApplyList is Apply for what a customer's list says by the names of the old
// devices (spec 6.5.2). A list need not hold a row for a device or for an
// interface it speaks of: every host name of every table is looked up in
// 機器対応, and every interface name without a row in IF 対応 is undecided.
// What the list says of the vendor and the model stays as it says it.
func ApplyList(sc *schema.Set, list *model.Dataset, dev []book.DeviceMap, ifs []book.IfMap) *Applied {
	return apply(sc, list, dev, ifs, true)
}

func apply(sc *schema.Set, old *model.Dataset, dev []book.DeviceMap, ifs []book.IfMap, list bool) *Applied {
	a := &Applied{Data: model.NewDataset(old.Source), Hosts: map[string]string{}}
	devs := map[string]book.DeviceMap{}
	for _, d := range dev {
		if _, dup := devs[d.OldHost]; !dup {
			devs[d.OldHost] = d
		}
	}
	// new host name → old interface → the row of IF 対応
	maps := map[string]map[string]book.IfMap{}
	users := map[string][]string{} // new host + new interface → old interfaces
	for _, m := range ifs {
		if maps[m.NewHost] == nil {
			maps[m.NewHost] = map[string]book.IfMap{}
		}
		if _, dup := maps[m.NewHost][m.OldIf]; dup {
			continue
		}
		maps[m.NewHost][m.OldIf] = m
		if m.NewIf != "" && m.NewIf != "-" {
			k := m.NewHost + "\x00" + m.NewIf
			users[k] = append(users[k], m.OldIf)
		}
	}
	var shared []string
	for k, list := range users {
		if len(list) > 1 {
			shared = append(shared, k)
		}
	}
	sort.Strings(shared)
	for _, k := range shared {
		host, name, _ := strings.Cut(k, "\x00")
		a.Shared = append(a.Shared, Note{Device: host, Value: name, Key: strings.Join(users[k], ", ")})
	}

	// The interfaces each old device has, to tell an interface name from
	// another word in an interface column.
	own := map[string]map[string]bool{}
	for _, r := range rows(old, "interfaces") {
		h := r.Get("device")
		if own[h] == nil {
			own[h] = map[string]bool{}
		}
		own[h][r.Get("name")] = true
	}

	var hosts []string
	if list {
		seen := map[string]bool{}
		for _, t := range sc.Tables {
			if devCol := t.DeviceColumn(); devCol != "" {
				for _, r := range rows(old, t.Name) {
					if h := r.Get(devCol); !seen[h] {
						seen[h] = true
						hosts = append(hosts, h)
					}
				}
			}
		}
	} else {
		for _, r := range rows(old, "devices") {
			hosts = append(hosts, r.Get("hostname"))
		}
	}
	for _, host := range hosts {
		d, ok := devs[host]
		if ok && d.NewHost != "" {
			a.Hosts[host] = d.NewHost
			continue
		}
		a.Unmapped = append(a.Unmapped, host)
		if !ok || d.Note == "" {
			a.NoReason = append(a.NoReason, host)
		}
	}

	undecided := map[string]bool{}
	for _, t := range sc.Tables {
		devCol := t.DeviceColumn()
		for _, r := range rows(old, t.Name) {
			if devCol == "" {
				a.Data.Adopt(clone(r))
				continue
			}
			host := r.Get(devCol)
			newHost, ok := a.Hosts[host]
			if !ok {
				continue
			}
			n := clone(r)
			note := func(col, val string) Note {
				return Note{Device: host, Table: t.Name, Key: r.KeyText(), Column: col, Value: val, Lines: r.Origins, Row: r}
			}
			// rename gives the new name of an interface; drop tells that the
			// interface is not carried over.
			rename := func(col, val string) (out string, drop bool) {
				m, ok := maps[newHost][val]
				switch {
				case ok && m.NewIf == "-":
					return "", true
				case ok && m.NewIf != "":
					return m.NewIf, false
				case ok || own[host][val] || list:
					if k := host + "\x00" + val; !undecided[k] {
						undecided[k] = true
						a.Undecided = append(a.Undecided, note(col, val))
					}
				}
				return val, false
			}
			left := false
			for _, c := range t.Columns {
				cell := n.Cells[c.Name]
				if cell == nil {
					continue
				}
				var vals []string
				cleared := false
				for _, v := range cell.Values {
					name, prefix := ifValue(c, v)
					if _, mapped := maps[newHost][v]; name == "" && c.OrIfname && (mapped || own[host][v]) {
						name = v
					}
					if name == "" {
						vals = append(vals, v)
						continue
					}
					out, drop := rename(c.Name, name)
					switch {
					case !drop:
						vals = append(vals, prefix+out)
					case t.IsKey(c.Name):
						left = true
					default:
						cleared = true
						a.Cleared = append(a.Cleared, note(c.Name, name))
					}
				}
				if left {
					a.Dropped = append(a.Dropped, note(c.Name, r.Get(c.Name)))
					break
				}
				if cleared {
					if len(vals) == 0 {
						delete(n.Cells, c.Name)
						continue
					}
				}
				cell.Values = vals
			}
			if left {
				continue
			}
			n.Cells[devCol].Values = []string{newHost}
			// The kind the old pack told from the old name says nothing of
			// the new name: the new pack tells it again (spec 10.3).
			if t.Name == "interfaces" && n.Get("name") != r.Get("name") {
				if k := n.Cells["kind"]; k != nil && fromDefault(k) {
					delete(n.Cells, "kind")
				}
			}
			if t.Name == "devices" && !list {
				d := devs[host]
				for col, v := range map[string]string{"vendor": d.NewVendor, "model": d.NewModel, "os_version": d.NewOS} {
					if v != "" {
						n.Cells[col] = &model.Cell{Values: []string{v}, Origins: []model.Origin{{Kind: "book"}}}
					}
				}
			}
			n.Key = n.Key[:0]
			for _, k := range t.Key {
				n.Key = append(n.Key, model.KV{Name: k, Value: n.Get(k)})
			}
			if !a.Data.Adopt(n) {
				a.Clashes = append(a.Clashes, note("", ""))
			}
		}
	}
	return a
}

// fromDefault tells that every value of a cell was put in by a pack, and not
// read from a line or written by a person.
func fromDefault(c *model.Cell) bool {
	if len(c.Origins) == 0 {
		return false
	}
	for _, o := range c.Origins {
		if o.Kind != "default" {
			return false
		}
	}
	return true
}

// ifValue tells whether a value of a column is an interface name, and gives
// the name and what stands before it. Besides the interface columns, a
// column may hold a name after a fixed word ("unnumbered:Loopback0"). A
// column that holds a name or something else (OrIfname) is left to the
// caller, who knows the interfaces of the device.
func ifValue(c *schema.Column, v string) (name, prefix string) {
	for _, a := range c.Also {
		if p, ok := strings.CutSuffix(a, "*"); ok {
			if strings.HasPrefix(v, p) && len(v) > len(p) {
				return v[len(p):], p
			}
		} else if v == a {
			return "", ""
		}
	}
	if c.Type == "ifname" {
		return v, ""
	}
	return "", ""
}

func clone(r *model.Row) *model.Row {
	n := &model.Row{Table: r.Table, Cells: map[string]*model.Cell{}}
	n.Key = append(n.Key, r.Key...)
	n.Origins = append(n.Origins, r.Origins...)
	n.Extra = append(n.Extra, r.Extra...)
	for name, c := range r.Cells {
		n.Cells[name] = &model.Cell{
			Values:  append([]string{}, c.Values...),
			List:    c.List,
			Origins: append([]model.Origin{}, c.Origins...),
		}
	}
	return n
}
