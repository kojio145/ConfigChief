package check

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/tmpl"
)

// The checks of the values that came with the tables of VLANs, switch ports
// and LAGs (appendix C 1.1): V-251 to V-258.

// vlanSet is the VLANs a list such as "10,20,30-40" stands for. all tells
// that the list is empty or says "all" (every VLAN); ok is false when the
// list cannot be read.
func vlanSet(c *model.Cell) (set map[int]bool, all, ok bool) {
	if c == nil || len(c.Values) == 0 {
		return nil, true, true
	}
	words := 0
	for _, v := range c.Values {
		switch v {
		case "all":
			all = true
			words++
		case "none":
			words++
		}
	}
	if words > 0 {
		return map[int]bool{}, all, len(c.Values) == 1
	}
	nums, err := tmpl.Ranges(c.Values)
	if err != nil {
		return nil, false, false
	}
	set = map[int]bool{}
	for _, n := range nums {
		if n < 1 || n > 4094 {
			return nil, false, false
		}
		set[n] = true
	}
	return set, false, true
}

// vlans runs the checks of one device after the other.
func (x *values) vlans() {
	sp := x.in.Schema.Table("switchports")
	ifs := x.in.Schema.Table("interfaces")
	lags := x.in.Schema.Table("lags")
	if sp == nil || ifs == nil || lags == nil {
		return
	}
	devs := map[string]bool{}
	for _, name := range []string{"switchports", "interfaces", "lags"} {
		for _, r := range x.rows(name) {
			devs[r.Get("device")] = true
		}
	}
	names := make([]string, 0, len(devs))
	for d := range devs {
		names = append(names, d)
	}
	sort.Strings(names)
	for _, dev := range names {
		x.vlanDefined(dev)
		x.switchports(dev)
		x.lagMembers(dev)
		x.subifVLANs(dev)
		x.bridgeGroups(dev)
		x.stpKind(dev)
		x.stpSteps(dev)
	}
}

// vlanDefined tells of a VLAN that a port or a subinterface uses and the
// table of VLANs does not have (V-251). A device with no row in that table
// is not looked at: its VLANs may be kept outside the configuration. VLAN 1
// is there without a row.
func (x *values) vlanDefined(dev string) {
	defined := map[int]bool{1: true}
	rows := x.of("vlans", dev)
	if len(rows) == 0 {
		return
	}
	for _, r := range rows {
		if n, err := strconv.Atoi(r.Get("vlan_id")); err == nil {
			defined[n] = true
		}
	}
	sp := x.in.Schema.Table("switchports")
	for _, r := range x.of("switchports", dev) {
		for _, col := range []string{"access_vlan", "native_vlan", "voice_vlan"} {
			n, err := strconv.Atoi(r.Get(col))
			if err != nil {
				continue
			}
			x.tally.Look("V-251", dev, 1)
			if !defined[n] {
				x.add("V-251", sp, r, col, fmt.Sprintf("%s の「%s」の VLAN %d が、VLAN の表にありません", r.Get("if_name"), sp.Column(col).Title.JA, n),
					model.KV{Name: "VLAN", Value: strconv.Itoa(n)})
			}
		}
		set, all, ok := vlanSet(r.Cells["allowed_vlans"])
		if !ok || all {
			continue
		}
		x.tally.Look("V-251", dev, 1)
		var lost []string
		for _, n := range sorted(set) {
			if !defined[n] {
				lost = append(lost, strconv.Itoa(n))
			}
		}
		if len(lost) > 0 {
			x.add("V-251", sp, r, "allowed_vlans", fmt.Sprintf("%s の許可 VLAN のうち %s が、VLAN の表にありません", r.Get("if_name"), strings.Join(lost, ",")),
				model.KV{Name: "VLAN", Value: strings.Join(lost, ",")})
		}
	}
	ifs := x.in.Schema.Table("interfaces")
	for _, r := range x.of("interfaces", dev) {
		for _, col := range []string{"encap_vlan", "vlan_id"} {
			n, err := strconv.Atoi(r.Get(col))
			if err != nil {
				continue
			}
			x.tally.Look("V-251", dev, 1)
			if !defined[n] {
				x.add("V-251", ifs, r, col, fmt.Sprintf("%s の「%s」の VLAN %d が、VLAN の表にありません", r.Get("name"), ifs.Column(col).Title.JA, n),
					model.KV{Name: "VLAN", Value: strconv.Itoa(n)})
			}
		}
	}
}

func sorted(set map[int]bool) []int {
	out := make([]int, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

// switchports looks at the rows of the ports one by one: the list of the
// allowed VLANs can be read (V-257), the native VLAN is among them (V-252),
// and a routed port has no VLAN (V-258).
func (x *values) switchports(dev string) {
	sp := x.in.Schema.Table("switchports")
	for _, r := range x.of("switchports", dev) {
		name := r.Get("if_name")
		x.tally.Look("V-257", dev, 1)
		set, all, ok := vlanSet(r.Cells["allowed_vlans"])
		if !ok {
			x.add("V-257", sp, r, "allowed_vlans", fmt.Sprintf("%s の許可 VLAN「%s」が読めません（10,20,30-40 の形か、all・none だけを書きます。番号は 1〜4094）", name, strings.Join(r.Cells["allowed_vlans"].Values, ",")))
		} else if native, err := strconv.Atoi(r.Get("native_vlan")); err == nil && r.Get("mode") == "trunk" {
			x.tally.Look("V-252", dev, 1)
			if !all && !set[native] {
				x.add("V-252", sp, r, "native_vlan", fmt.Sprintf("%s のネイティブ VLAN %d が、許可 VLAN に入っていません（タグなしの通信が通りません）", name, native),
					model.KV{Name: "ネイティブ VLAN", Value: strconv.Itoa(native)}, model.KV{Name: "許可 VLAN", Value: strings.Join(r.Cells["allowed_vlans"].Values, ",")})
			}
		}
		if r.Get("mode") == "routed" {
			x.tally.Look("V-258", dev, 1)
			for _, col := range []string{"access_vlan", "native_vlan", "allowed_vlans", "voice_vlan"} {
				if c := r.Cells[col]; c != nil && len(c.Values) > 0 {
					x.add("V-258", sp, r, col, fmt.Sprintf("%s はモードが routed（L3 のポート）ですが、「%s」があります", name, sp.Column(col).Title.JA))
				}
			}
		}
	}
}

// lagFamily names the protocol a way of bundling belongs to.
func lagFamily(mode string) string {
	switch mode {
	case "on":
		return "固定（on）"
	case "active", "passive":
		return "LACP"
	case "auto", "desirable":
		return "PAgP"
	}
	return ""
}

// lagMembers looks at the members of each LAG together: one way of bundling
// (V-253), and the same settings as switch ports (V-254).
func (x *values) lagMembers(dev string) {
	lags := x.in.Schema.Table("lags")
	groups := map[string][]*model.Row{}
	var order []string
	for _, r := range x.of("lags", dev) {
		g := r.Get("group")
		if groups[g] == nil {
			order = append(order, g)
		}
		groups[g] = append(groups[g], r)
	}
	for _, g := range order {
		rows := groups[g]
		if len(rows) < 2 {
			continue
		}
		first := rows[0]
		x.tally.Look("V-253", dev, 1)
		x.tally.Look("V-254", dev, 1)
		for _, r := range rows[1:] {
			a, b := lagFamily(first.Get("mode")), lagFamily(r.Get("mode"))
			if a != "" && b != "" && a != b {
				x.add("V-253", lags, r, "mode", fmt.Sprintf("LAG %s のメンバー %s の束ね方（%s）が、%s（%s）と違います", g, r.Get("if_name"), b, first.Get("if_name"), a),
					model.KV{Name: first.Get("if_name"), Value: first.Get("mode")}, model.KV{Name: r.Get("if_name"), Value: r.Get("mode")})
			}
			pa, pb := x.find("switchports", dev, "if_name", first.Get("if_name")), x.find("switchports", dev, "if_name", r.Get("if_name"))
			if pa == nil || pb == nil {
				continue
			}
			sp := x.in.Schema.Table("switchports")
			for _, col := range []string{"mode", "access_vlan", "native_vlan", "allowed_vlans"} {
				if text(pa, col) != text(pb, col) {
					x.add("V-254", sp, pb, col, fmt.Sprintf("LAG %s のメンバー %s の「%s」が、%s と違います（束ねるポートは設定をそろえます）", g, r.Get("if_name"), sp.Column(col).Title.JA, first.Get("if_name")),
						model.KV{Name: first.Get("if_name"), Value: text(pa, col)}, model.KV{Name: r.Get("if_name"), Value: text(pb, col)})
				}
			}
		}
	}
}

// text is the value of a cell as one text; the items of a list are joined.
func text(r *model.Row, col string) string {
	if c := r.Cells[col]; c != nil {
		return strings.Join(c.Values, ",")
	}
	return ""
}

// subifVLANs looks at the subinterfaces of each port: no VLAN twice, and one
// native VLAN at most (V-255). The parent is the name before the last dot.
func (x *values) subifVLANs(dev string) {
	ifs := x.in.Schema.Table("interfaces")
	type seen struct {
		vlan   map[string]string
		native string
	}
	parents := map[string]*seen{}
	for _, r := range x.of("interfaces", dev) {
		name, vlan := r.Get("name"), r.Get("encap_vlan")
		dot := strings.LastIndex(name, ".")
		if vlan == "" || dot < 0 {
			continue
		}
		x.tally.Look("V-255", dev, 1)
		p := parents[name[:dot]]
		if p == nil {
			p = &seen{vlan: map[string]string{}}
			parents[name[:dot]] = p
		}
		if other, dup := p.vlan[vlan]; dup {
			x.add("V-255", ifs, r, "encap_vlan", fmt.Sprintf("%s の VLAN 番号 %s が、同じポートの %s と重なっています", name, vlan, other),
				model.KV{Name: other, Value: vlan}, model.KV{Name: name, Value: vlan})
		} else {
			p.vlan[vlan] = name
		}
		if r.Get("encap_native") == "true" {
			if p.native != "" {
				x.add("V-255", ifs, r, "encap_native", fmt.Sprintf("%s と %s の両方が、タグなしの VLAN になっています（1 つのポートに 1 つです）", p.native, name))
			} else {
				p.native = name
			}
		}
	}
}

// bridgeGroups tells of a bridge group that joins nothing: one interface
// only, the BVI not counted (V-256).
func (x *values) bridgeGroups(dev string) {
	ifs := x.in.Schema.Table("interfaces")
	members := map[string][]*model.Row{}
	var order []string
	for _, r := range x.of("interfaces", dev) {
		g := r.Get("bridge_group")
		if g == "" {
			continue
		}
		if _, ok := members[g]; !ok {
			order = append(order, g)
			members[g] = nil
		}
		if r.Get("kind") != "bvi" && !strings.HasPrefix(r.Get("name"), "BVI") {
			members[g] = append(members[g], r)
		}
	}
	for _, g := range order {
		x.tally.Look("V-256", dev, 1)
		switch rows := members[g]; len(rows) {
		case 0:
			for _, r := range x.of("interfaces", dev) {
				if r.Get("bridge_group") == g {
					x.add("V-256", ifs, r, "bridge_group", fmt.Sprintf("ブリッジグループ %s に入っている IF が、BVI のほかにありません", g))
					break
				}
			}
		case 1:
			x.add("V-256", ifs, rows[0], "bridge_group", fmt.Sprintf("ブリッジグループ %s に入っている IF が、%s の 1 つだけです（つなぐ相手がありません）", g, rows[0].Get("name")))
		}
	}
}
