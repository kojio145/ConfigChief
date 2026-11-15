package check

import (
	"fmt"
	"net/netip"

	"github.com/kojio145/configchief/internal/model"
)

// The checks of the values that came with the routing and redundancy tables
// (appendix C 1.1, stage M3): V-205, V-207, V-210, V-211 and V-214.

func (x *values) rows(table string) []*model.Row {
	if tab := x.in.Data.Tables[table]; tab != nil {
		return tab.Rows
	}
	return nil
}

// find gives the row of a device with a value in a column, or nil.
func (x *values) find(table, dev, column, val string) *model.Row {
	for _, r := range x.of(table, dev) {
		if r.Get(column) == val {
			return r
		}
	}
	return nil
}

// redundancy looks at the virtual addresses (V-205): one has to lie in a
// subnet of its interface. An interface without a written address is left
// alone.
func (x *values) redundancy() {
	t := x.in.Schema.Table("fhrp")
	if t == nil {
		return
	}
	list, _ := x.addrs()
	for _, r := range x.rows("fhrp") {
		dev, ifn := r.Get("device"), r.Get("if_name")
		vip, err := netip.ParseAddr(r.Get("virtual_ip"))
		if err != nil {
			continue
		}
		var nets []string
		inside := false
		for _, a := range list {
			if a.device != dev || a.name != ifn {
				continue
			}
			nets = append(nets, a.prefix.Masked().String())
			inside = inside || a.prefix.Masked().Contains(vip)
		}
		if len(nets) == 0 {
			continue
		}
		x.tally.Look("V-205", dev, 1)
		if !inside {
			x.add("V-205", t, r, "virtual_ip", fmt.Sprintf("仮想 IP %s は、%s のサブネットに入りません", vip, ifn),
				model.KV{Name: "仮想 IP", Value: vip.String()}, model.KV{Name: ifn, Value: join(nets)})
		}
	}
}

func join(vals []string) string {
	out := ""
	for i, v := range vals {
		if i > 0 {
			out += ", "
		}
		out += v
	}
	return out
}

// routerIDs looks for a router ID two devices share (V-207): OSPF among
// OSPF, BGP among BGP. The later row gets the finding.
func (x *values) routerIDs() {
	for _, name := range []string{"ospf_processes", "bgp_processes"} {
		t := x.in.Schema.Table(name)
		if t == nil {
			continue
		}
		first := map[string]*model.Row{}
		for _, r := range x.rows(name) {
			id, dev := r.Get("router_id"), r.Get("device")
			if id == "" {
				continue
			}
			x.tally.Look("V-207", dev, 1)
			f, ok := first[id]
			switch {
			case !ok:
				first[id] = r
			case f.Get("device") != dev:
				x.add("V-207", t, r, "router_id", fmt.Sprintf("ルータ ID %s が、%s の%sと同じです", id, f.Get("device"), t.Title.JA),
					model.KV{Name: dev, Value: id}, model.KV{Name: f.Get("device"), Value: id})
			}
		}
	}
}

// unused looks for an access list, a prefix list, a watch, a QoS class or a
// QoS policy that no column of the design data names (V-210). The entries of
// a list and the conditions of a class do not count as a use of it.
func (x *values) unused() {
	for _, target := range []struct{ table, column string }{{"acls", "name"}, {"prefix_lists", "name"}, {"tracks", "id"}, {"qos_classes", "name"}, {"qos_policies", "policy"}} {
		t := x.in.Schema.Table(target.table)
		if t == nil {
			continue
		}
		used := map[string]bool{}
		for _, st := range x.in.Schema.Tables {
			if (st.Name == "acl_entries" && target.table == "acls") || (st.Name == "qos_class_matches" && target.table == "qos_classes") || st.Name == target.table {
				continue
			}
			for _, c := range st.Columns {
				if c.Ref != target.table+"."+target.column {
					continue
				}
				for _, r := range x.rows(st.Name) {
					if cell := r.Cells[c.Name]; cell != nil {
						for _, v := range cell.Values {
							used[device(st, r)+"\x00"+v] = true
						}
					}
				}
			}
		}
		told := map[string]bool{}
		for _, r := range x.rows(target.table) {
			dev, name := device(t, r), r.Get(target.column)
			key := dev + "\x00" + name
			if name == "" || told[key] {
				continue
			}
			told[key] = true
			x.tally.Look("V-210", dev, 1)
			if !used[key] {
				x.add("V-210", t, r, target.column, fmt.Sprintf("%s「%s」は、どの表からも使われていません（追加コンフィグなど、表の外で使っているなら承認します）", t.Title.JA, name),
					model.KV{Name: "値", Value: name})
			}
		}
	}
}

// emptyLists looks for an access list without an entry, or without an entry
// that lets anything through (V-211).
func (x *values) emptyLists() {
	t := x.in.Schema.Table("acls")
	if t == nil {
		return
	}
	for _, r := range x.rows("acls") {
		dev, name := r.Get("device"), r.Get("name")
		entries, permits := 0, 0
		for _, e := range x.of("acl_entries", dev) {
			if e.Get("acl") != name || e.Get("action") == "remark" {
				continue
			}
			entries++
			if e.Get("action") == "permit" {
				permits++
			}
		}
		x.tally.Look("V-211", dev, 1)
		switch {
		case entries == 0:
			x.add("V-211", t, r, "name", fmt.Sprintf("ACL「%s」に行が 1 つもありません", name), model.KV{Name: "値", Value: name})
		case permits == 0:
			x.add("V-211", t, r, "name", fmt.Sprintf("ACL「%s」に permit の行が 1 つもありません（当てた所の通信をすべて止めます）", name), model.KV{Name: "値", Value: name})
		}
	}
}

// trackSources looks at the watches the routes and the redundancy groups
// rely on (V-214): one that sends from an interface that is switched off
// never sees its target.
func (x *values) trackSources() {
	for _, name := range []string{"static_routes", "fhrp"} {
		t := x.in.Schema.Table(name)
		if t == nil {
			continue
		}
		for _, r := range x.rows(name) {
			dev, id := r.Get("device"), r.Get("track")
			if id == "" {
				continue
			}
			w := x.find("tracks", dev, "id", id)
			if w == nil || w.Get("source_if") == "" {
				continue
			}
			x.tally.Look("V-214", dev, 1)
			src := w.Get("source_if")
			if i := x.find("interfaces", dev, "name", src); i != nil && i.Get("enabled") == "false" {
				x.add("V-214", t, r, "track", fmt.Sprintf("回線監視 %s の送信元 IF %s が無効です", id, src),
					model.KV{Name: "回線監視", Value: id}, model.KV{Name: "送信元 IF", Value: src})
			}
		}
	}
}
