package check

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/kojio145/configchief/internal/model"
	"github.com/kojio145/configchief/internal/schema"
)

// The checks that set the two ends of a cable side by side (appendix C 1.2):
// X-381 to X-387. Which port meets which is not in a configuration: the
// columns "peer device" and "peer interface" of the interfaces say it, and a
// port without them is not looked at. X-386 needs no pair: it looks at the
// names of the VLANs of all devices.

// l2 looks at every port that names its other end.
func (c *cross) l2() {
	t := c.in.Schema.Table("interfaces")
	sp := c.in.Schema.Table("switchports")
	lags := c.in.Schema.Table("lags")
	if t == nil || sp == nil || lags == nil {
		return
	}
	known := map[string]bool{}
	for _, r := range c.rows("devices") {
		known[r.Get("hostname")] = true
	}
	port := func(table, dev, name string) *model.Row { return c.find(table, dev, "if_name", name) }
	for _, a := range c.ifs {
		pd, pi := a.row.Get("peer_device"), a.row.Get("peer_if")
		if pd == "" || pi == "" || !known[pd] {
			continue
		}
		c.tally.Look("X-381", a.device, 1)
		b := c.byDev[pd][pi]
		if b == nil {
			c.add("X-381", t, a.row, "peer_if", fmt.Sprintf("対向の %s に、IF %s がありません", pd, pi))
			continue
		}
		if bd, bi := b.row.Get("peer_device"), b.row.Get("peer_if"); bd != "" && bi != "" && (bd != a.device || bi != a.name) {
			c.add("X-381", t, a.row, "peer_if", fmt.Sprintf("対向の %s の %s は、対向を %s の %s としています（こちらは %s）", pd, pi, bd, bi, a.name),
				model.KV{Name: where(a.device, a.name), Value: where(pd, pi)}, model.KV{Name: where(pd, pi), Value: where(bd, bi)})
			continue
		}
		// Each pair once: from the side that comes first, or from the only
		// side that names the other.
		if b.row.Get("peer_device") != "" && where(b.device, b.name) < where(a.device, a.name) {
			continue
		}
		wa, wb := where(a.device, a.name), where(b.device, b.name)

		c.tally.Look("X-387", a.device, 1)
		for _, col := range []string{"speed", "duplex"} {
			if va, vb := a.row.Get(col), b.row.Get(col); va != "" && vb != "" && va != vb {
				c.add("X-387", t, a.row, col, fmt.Sprintf("「%s」が、対向の %s と違います", t.Column(col).Title.JA, wb), model.KV{Name: wa, Value: va}, model.KV{Name: wb, Value: vb})
			}
		}

		c.udldPair(sp, a.device, b.device, port("switchports", a.device, a.name), port("switchports", b.device, b.name), wa, wb)

		la, lb := port("lags", a.device, a.name), port("lags", b.device, b.name)
		c.tally.Look("X-385", a.device, 1)
		switch {
		case la != nil && lb == nil:
			c.add("X-385", lags, la, "group", fmt.Sprintf("%s は LAG のメンバーですが、対向の %s は LAG に入っていません", a.name, wb))
		case la == nil && lb != nil:
			c.add("X-385", lags, lb, "group", fmt.Sprintf("%s は LAG のメンバーですが、対向の %s は LAG に入っていません", b.name, wa))
		case la != nil && lb != nil:
			if msg := lagPair(la.Get("mode"), lb.Get("mode")); msg != "" {
				c.add("X-385", lags, la, "mode", fmt.Sprintf("LAG の束ね方が、対向の %s と合いません: %s", wb, msg), model.KV{Name: wa, Value: la.Get("mode")}, model.KV{Name: wb, Value: lb.Get("mode")})
			}
		}

		pa, pb := port("switchports", a.device, a.name), port("switchports", b.device, b.name)
		switch {
		case pa != nil && pb != nil:
			c.switchPair(sp, pa, pb, wa, wb)
		case pa != nil:
			c.trunkToRouter(sp, pa, b, wa, wb)
		case pb != nil:
			c.trunkToRouter(sp, pb, a, wb, wa)
		}
	}
	c.vlanNames()
}

// lagPair tells why two ways of bundling do not make a LAG ("" if they do).
// An empty mode is not compared.
func lagPair(a, b string) string {
	if a == "" || b == "" {
		return ""
	}
	fa, fb := lagFamily(a), lagFamily(b)
	switch {
	case fa != fb:
		return fmt.Sprintf("%s と %s", fa, fb)
	case a == "passive" && b == "passive":
		return "両方が passive（どちらも相手を待ちます。片方を active にします）"
	case a == "auto" && b == "auto":
		return "両方が auto（どちらも相手を待ちます。片方を desirable にします）"
	}
	return ""
}

// switchPair compares two switch ports that meet: the mode and the access
// VLAN (X-384), the allowed VLANs (X-382) and the native VLAN (X-383) of a
// trunk. An empty native VLAN is 1; an empty list of allowed VLANs is all.
func (c *cross) switchPair(sp *schema.Table, pa, pb *model.Row, wa, wb string) {
	dev := pa.Get("device")
	ma, mb := pa.Get("mode"), pb.Get("mode")
	c.tally.Look("X-384", dev, 1)
	fixed := func(m string) bool { return m == "access" || m == "trunk" }
	if fixed(ma) && fixed(mb) && ma != mb {
		c.add("X-384", sp, pa, "mode", fmt.Sprintf("モードが、対向の %s と違います", wb), model.KV{Name: wa, Value: ma}, model.KV{Name: wb, Value: mb})
		return
	}
	if ma == "access" && mb == "access" {
		if va, vb := pa.Get("access_vlan"), pb.Get("access_vlan"); va != "" && vb != "" && va != vb {
			c.add("X-384", sp, pa, "access_vlan", fmt.Sprintf("アクセス VLAN が、対向の %s と違います", wb), model.KV{Name: wa, Value: va}, model.KV{Name: wb, Value: vb})
		}
		return
	}
	if ma != "trunk" || mb != "trunk" {
		return
	}
	c.tally.Look("X-383", dev, 1)
	na, nb := native(pa), native(pb)
	if na != nb {
		c.add("X-383", sp, pa, "native_vlan", fmt.Sprintf("ネイティブ VLAN が、対向の %s と違います", wb), model.KV{Name: wa, Value: na}, model.KV{Name: wb, Value: nb})
	}
	sa, alla, oka := vlanSet(pa.Cells["allowed_vlans"])
	sb, allb, okb := vlanSet(pb.Cells["allowed_vlans"])
	if !oka || !okb {
		return // V-257 tells
	}
	c.tally.Look("X-382", dev, 1)
	if alla != allb || !alla && strings.Join(nums(sa), ",") != strings.Join(nums(sb), ",") {
		c.add("X-382", sp, pa, "allowed_vlans", fmt.Sprintf("許可 VLAN が、対向の %s と違います%s", wb, onlyOn(wa, wb, sa, sb, alla, allb)),
			model.KV{Name: wa, Value: allowedText(pa)}, model.KV{Name: wb, Value: allowedText(pb)})
	}
}

func native(p *model.Row) string {
	if v := p.Get("native_vlan"); v != "" {
		return v
	}
	return "1"
}

func nums(set map[int]bool) []string {
	out := []string{}
	for _, n := range sorted(set) {
		out = append(out, strconv.Itoa(n))
	}
	return out
}

func allowedText(p *model.Row) string {
	if v := text(p, "allowed_vlans"); v != "" {
		return v
	}
	return "（空 = すべて）"
}

// onlyOn names the VLANs that one side of a trunk has and the other has not.
func onlyOn(wa, wb string, sa, sb map[int]bool, alla, allb bool) string {
	if alla || allb {
		return ""
	}
	var a, b []string
	for _, n := range sorted(sa) {
		if !sb[n] {
			a = append(a, strconv.Itoa(n))
		}
	}
	for _, n := range sorted(sb) {
		if !sa[n] {
			b = append(b, strconv.Itoa(n))
		}
	}
	var parts []string
	if len(a) > 0 {
		parts = append(parts, fmt.Sprintf("%s だけにある VLAN: %s", wa, strings.Join(a, ",")))
	}
	if len(b) > 0 {
		parts = append(parts, fmt.Sprintf("%s だけにある VLAN: %s", wb, strings.Join(b, ",")))
	}
	if len(parts) == 0 {
		return ""
	}
	return "（" + strings.Join(parts, "、") + "）"
}

// trunkToRouter compares a trunk with a port that has tagged subinterfaces
// (a router): every VLAN of a subinterface has to be allowed on the trunk
// (X-382), and the VLAN without a tag has to be the native VLAN of the trunk
// (X-383). A port with no subinterface is not compared.
func (c *cross) trunkToRouter(sp *schema.Table, p *model.Row, r *xIf, wp, wr string) {
	if p.Get("mode") != "trunk" {
		return
	}
	set, all, ok := vlanSet(p.Cells["allowed_vlans"])
	if !ok {
		return
	}
	dev := p.Get("device")
	var subs []*xIf
	for _, i := range c.ifs {
		if i.device == r.device && strings.HasPrefix(i.name, r.name+".") && i.row.Get("encap_vlan") != "" {
			subs = append(subs, i)
		}
	}
	sort.Slice(subs, func(i, j int) bool { return subs[i].name < subs[j].name })
	for _, s := range subs {
		vlan := s.row.Get("encap_vlan")
		n, err := strconv.Atoi(vlan)
		if err != nil {
			continue
		}
		ws := where(s.device, s.name)
		if s.row.Get("encap_native") == "true" {
			c.tally.Look("X-383", dev, 1)
			if native(p) != vlan {
				c.add("X-383", sp, p, "native_vlan", fmt.Sprintf("ネイティブ VLAN が、対向の %s のタグなしの VLAN と違います", ws),
					model.KV{Name: wp, Value: native(p)}, model.KV{Name: ws, Value: vlan})
			}
			continue
		}
		c.tally.Look("X-382", dev, 1)
		if !all && !set[n] {
			c.add("X-382", sp, p, "allowed_vlans", fmt.Sprintf("対向の %s の VLAN %s が、許可 VLAN に入っていません", ws, vlan),
				model.KV{Name: wp, Value: allowedText(p)}, model.KV{Name: ws, Value: vlan})
		}
	}
}

// vlanNames tells of a VLAN that has one name on a device and another name
// on another (X-386). The first device, in the order of the names, is the
// one the others are compared with.
func (c *cross) vlanNames() {
	t := c.in.Schema.Table("vlans")
	if t == nil {
		return
	}
	rows := append([]*model.Row{}, c.rows("vlans")...)
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].Get("device") < rows[j].Get("device") })
	first := map[string]*model.Row{}
	for _, r := range rows {
		id, name := r.Get("vlan_id"), r.Get("name")
		if name == "" {
			continue
		}
		c.tally.Look("X-386", r.Get("device"), 1)
		f := first[id]
		if f == nil {
			first[id] = r
			continue
		}
		if f.Get("name") != name {
			c.add("X-386", t, r, "name", fmt.Sprintf("VLAN %s の名前が、%s（%s）と違います", id, f.Get("device"), f.Get("name")),
				model.KV{Name: f.Get("device"), Value: f.Get("name")}, model.KV{Name: r.Get("device"), Value: name})
		}
	}
}

// udldPair sets the UDLD of the two ends of a cable side by side (X-388):
// it works only where both ends run it, in the same way. An end whose
// device runs UDLD as a whole is not judged (its fibre ports take that
// setting, and the data does not tell which ports are fibre).
func (c *cross) udldPair(sp *schema.Table, da, db string, pa, pb *model.Row, wa, wb string) {
	if sp.Column("udld") == nil {
		return
	}
	mode := func(r *model.Row) string {
		if r == nil || r.Get("udld") == "disable" {
			return ""
		}
		return r.Get("udld")
	}
	global := func(dev string) bool {
		for _, g := range c.rows("l2_settings") {
			if g.Get("device") == dev && g.Get("udld") != "" {
				return true
			}
		}
		return false
	}
	ma, mb := mode(pa), mode(pb)
	if ma == "" && mb == "" {
		return
	}
	c.tally.Look("X-388", da, 1)
	switch {
	case ma != "" && mb == "" && !global(db):
		c.add("X-388", sp, pa, "udld", fmt.Sprintf("%s は UDLD（%s）を使いますが、対向の %s は使っていません（UDLD は両端で動かします）", wa, ma, wb))
	case ma == "" && mb != "" && !global(da):
		c.add("X-388", sp, pb, "udld", fmt.Sprintf("%s は UDLD（%s）を使いますが、対向の %s は使っていません（UDLD は両端で動かします）", wb, mb, wa))
	case ma != "" && mb != "" && ma != mb:
		c.add("X-388", sp, pa, "udld", fmt.Sprintf("UDLD の方式が、対向の %s と違います", wb), model.KV{Name: wa, Value: ma}, model.KV{Name: wb, Value: mb})
	}
}
