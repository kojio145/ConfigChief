package convert

import (
	"sort"
	"strconv"
	"strings"

	"github.com/kojio145/configchief/internal/book"
	"github.com/kojio145/configchief/internal/model"
)

// Shape is what a pack knows of a model (appendix B 6): its ports in the
// order they are handed out, and how its logical interfaces are named.
type Shape struct {
	ID      string
	Ports   []ShapePort
	Logical map[string]string
}

// ShapePort is a physical port of a model.
type ShapePort struct {
	Name, Role, Sub, VlanIf string
}

// Shapes finds the model a row of 機器対応 names (nil: not known).
type Shapes func(vendor, model string) *Shape

// BasisShape is the basis written for a proposal from a model.
func BasisShape(id string) string { return "機種の定義（" + id + "）による割り当て" }

// natLess orders names with their numbers as numbers: Fa2 before Fa10.
func natLess(a, b string) bool {
	for a != "" && b != "" {
		da, db := digits(a), digits(b)
		if da > 0 && db > 0 {
			x, _ := strconv.Atoi(a[:da])
			y, _ := strconv.Atoi(b[:db])
			if x != y {
				return x < y
			}
			a, b = a[da:], b[db:]
			continue
		}
		if a[0] != b[0] {
			return a[0] < b[0]
		}
		a, b = a[1:], b[1:]
	}
	return len(a) < len(b)
}

func digits(s string) int {
	n := 0
	for n < len(s) && s[n] >= 48 && s[n] <= 57 {
		n++
	}
	return n
}

// number gives the first number of a name: Tunnel12.0 → 12.
func number(name string) string {
	i := strings.IndexAny(name, "0123456789")
	if i < 0 {
		return ""
	}
	return name[i : i+digits(name[i:])]
}

// byShape proposes the new names of the interfaces of one old device from
// the model of the new one (appendix B 6.3). An interface it has no answer
// for is not in the result; "-" stands for a port that is not in use.
func byShape(old *model.Dataset, host string, sh *Shape) map[string]string {
	of := func(table string) []*model.Row {
		var out []*model.Row
		for _, r := range rows(old, table) {
			if r.Get("device") == host {
				out = append(out, r)
			}
		}
		return out
	}
	ifs := of("interfaces")
	byName := map[string]*model.Row{}
	hasVlanIf := false
	for _, i := range ifs {
		byName[i.Get("name")] = i
		if isVlanIf(i) {
			hasVlanIf = true
		}
	}
	sw := map[string]*model.Row{}
	for _, s := range of("switchports") {
		sw[s.Get("if_name")] = s
	}
	// The parent of a subinterface: "X.10" is under "X", or under "X.0".
	parentOf := func(name string) string {
		i := strings.LastIndex(name, ".")
		if i < 0 {
			return ""
		}
		for _, p := range []string{name[:i], name[:i] + ".0"} {
			if r := byName[p]; r != nil && p != name && r.Get("kind") == "physical" {
				return p
			}
		}
		return ""
	}
	hasSub, pppPhys, pppLogic := map[string]bool{}, map[string]bool{}, map[string]string{}
	for _, i := range ifs {
		if i.Get("kind") == "subif" {
			hasSub[parentOf(i.Get("name"))] = true
		}
	}
	for _, p := range of("pppoe") {
		ph, lo := p.Get("physical_if"), p.Get("logical_if")
		pppPhys[ph] = true
		if lo != "" && lo != ph {
			pppLogic[lo] = ph
		}
	}

	out := map[string]string{}
	used := map[string]bool{}
	take := func(old, name string) {
		out[old] = name
		used[name] = true
	}

	// 1. The physical ports: those of the switch to the ports of the
	// switch, the others to the others, each in the order of their names.
	var l2, l3 []string
	for _, i := range ifs {
		name := i.Get("name")
		if i.Get("kind") != "physical" || isVlanIf(i) {
			continue
		}
		if i.Get("enabled") == "false" {
			out[name] = "-"
			continue
		}
		s := sw[name]
		switch {
		case s != nil && s.Get("mode") != "routed":
			l2 = append(l2, name)
		case s == nil && hasVlanIf && i.Get("ipv4") == "" && i.Get("ipv6") == "" && i.Get("bridge_group") == "" && !hasSub[name] && !pppPhys[name]:
			l2 = append(l2, name)
		default:
			l3 = append(l3, name)
		}
	}
	sort.SliceStable(l2, func(a, b int) bool { return natLess(l2[a], l2[b]) })
	sort.SliceStable(l3, func(a, b int) bool { return natLess(l3[a], l3[b]) })
	port := map[string]*ShapePort{} // new name → the port
	var freeSw, freeL3 []*ShapePort
	for i := range sh.Ports {
		p := &sh.Ports[i]
		port[p.Name] = p
		if p.Role == "switch" {
			freeSw = append(freeSw, p)
		} else {
			freeL3 = append(freeL3, p)
		}
	}
	// A switch has no ports apart from those of the switch: a routed port of
	// the old device is one of them, in the order of all the names.
	if len(freeL3) == 0 {
		l2 = append(l2, l3...)
		l3 = nil
		sort.SliceStable(l2, func(a, b int) bool { return natLess(l2[a], l2[b]) })
	}
	// A port the new model has under the same name keeps its name; the
	// others take the ports that are left, in their order.
	hand := func(olds []string, free []*ShapePort) {
		var rest []string
		for _, name := range olds {
			same := false
			for _, p := range free {
				if p.Name == name {
					same = true
				}
			}
			if same {
				take(name, name)
			} else {
				rest = append(rest, name)
			}
		}
		n := 0
		for _, name := range rest {
			for n < len(free) && used[free[n].Name] {
				n++
			}
			if n < len(free) {
				take(name, free[n].Name)
			}
		}
	}
	hand(l2, freeSw)
	hand(l3, freeL3)
	sub := func(oldParent string) string {
		if p := port[out[oldParent]]; p != nil {
			return p.Sub
		}
		return ""
	}
	// fill puts the values on a form of "logical"; "" if a mark has no
	// value, or if the name is taken.
	fill := func(form string, marks map[string]string) string {
		if form == "" {
			return ""
		}
		for k, v := range marks {
			if strings.Contains(form, k) {
				if v == "" {
					return ""
				}
				form = strings.ReplaceAll(form, k, v)
			}
		}
		for _, c := range []struct {
			mark string
			from int
		}{{"{i}", 1}, {"{j}", 0}} {
			if !strings.Contains(form, c.mark) {
				continue
			}
			for n := c.from; n < c.from+5000; n++ {
				if name := strings.ReplaceAll(form, c.mark, strconv.Itoa(n)); !used[name] {
					form = name
					break
				}
			}
		}
		if strings.Contains(form, "{") || used[form] {
			return ""
		}
		return form
	}

	// The VLANs a trunk of the old device carries, and the access ports of
	// each VLAN (a port of the switch without a row is in VLAN 1).
	trunkAll, trunked, access := false, map[string]bool{}, map[string][]string{}
	for _, name := range l2 {
		s := sw[name]
		mode, vlan := "", ""
		if s != nil {
			mode, vlan = s.Get("mode"), s.Get("access_vlan")
		}
		if mode == "trunk" {
			list := s.Cells["allowed_vlans"]
			if list == nil || len(list.Values) == 0 {
				trunkAll = true
				continue
			}
			for _, v := range list.Values {
				if v == "all" {
					trunkAll = true
				}
				for _, n := range expand(v) {
					trunked[n] = true
				}
			}
			continue
		}
		if mode == "" || mode == "access" {
			if vlan == "" {
				vlan = "1"
			}
			access[vlan] = append(access[vlan], name)
		}
	}

	for _, i := range ifs {
		name, kind := i.Get("name"), i.Get("kind")
		if _, done := out[name]; done {
			continue
		}
		var got string
		switch {
		case pppLogic[name] != "":
			got = fill(sh.Logical["pppoe"], map[string]string{"{parent}": sub(pppLogic[name]), "{n}": number(name)})
		case isVlanIf(i):
			vid := i.Get("vlan_id")
			if vid == "" {
				vid = number(name)
			}
			if !trunkAll && !trunked[vid] {
				var best *ShapePort
				for _, a := range access[vid] {
					if p := port[out[a]]; p != nil && p.VlanIf != "" && (best == nil || natLess(p.Name, best.Name)) {
						best = p
					}
				}
				if best != nil && !used[best.VlanIf] {
					got = best.VlanIf
				}
			}
			if got == "" {
				got = fill(sh.Logical["vlan"], map[string]string{"{vlan}": vid, "{n}": number(name)})
			}
		case kind == "subif":
			at := strings.LastIndex(name, ".")
			got = fill(sh.Logical["subif"], map[string]string{"{parent}": sub(parentOf(name)), "{vlan}": i.Get("encap_vlan"), "{n}": name[at+1:]})
		case kind == "loopback" || kind == "tunnel" || kind == "lag":
			got = fill(sh.Logical[kind], map[string]string{"{n}": number(name)})
		}
		if got != "" {
			take(name, got)
		}
	}
	return out
}

func isVlanIf(i *model.Row) bool {
	return i.Get("vlan_id") != "" || i.Get("kind") == "vlan"
}

// expand gives the numbers of "10,20,30-32" as texts.
func expand(list string) []string {
	var out []string
	for _, part := range strings.Split(list, ",") {
		a, b, isRange := strings.Cut(strings.TrimSpace(part), "-")
		from, err := strconv.Atoi(a)
		if err != nil {
			continue
		}
		to := from
		if isRange {
			if to, err = strconv.Atoi(b); err != nil || to-from > 4094 {
				continue
			}
		}
		for n := from; n <= to; n++ {
			out = append(out, strconv.Itoa(n))
		}
	}
	return out
}

// ProposeWith is Propose with the models of the packs: the new names of the
// interfaces of a device whose new vendor and model are known come from
// the model first (appendix B 6.3).
func ProposeWith(old, new *model.Dataset, have []book.DeviceMap, shapes Shapes) (dev []book.DeviceMap, ifs []book.IfMap) {
	dev, ifs = Propose(old, new, have)
	if shapes == nil {
		return dev, ifs
	}
	byHost := map[string]book.DeviceMap{} // new host → the row
	for _, d := range append(append([]book.DeviceMap{}, have...), dev...) {
		if _, dup := byHost[d.NewHost]; !dup && d.NewHost != "" {
			byHost[d.NewHost] = d
		}
	}
	names := map[string]map[string]string{}
	for i := range ifs {
		m := &ifs[i]
		d := byHost[m.NewHost]
		sh := shapes(d.NewVendor, d.NewModel)
		if sh == nil {
			continue
		}
		if names[d.OldHost] == nil {
			names[d.OldHost] = byShape(old, d.OldHost, sh)
		}
		if name := names[d.OldHost][m.OldIf]; name != "" {
			m.NewIf, m.Basis = name, BasisShape(sh.ID)
		}
	}
	return dev, ifs
}
