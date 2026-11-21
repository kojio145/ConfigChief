package check

import (
	"fmt"
	"net/netip"
	"strconv"

	"github.com/kojio145/configchief/internal/model"
)

// The checks that set two devices of the case side by side (appendix C 1.2,
// stage M3): BGP, OSPF, the redundancy groups and the subnets they share.
// IPsec and the tunnels (stage M4) are in cross_m4.go.
// How two rows become a pair is told in spec 11.5.

// crossIDs are the checks Cross runs.
var crossIDs = []string{"X-300", "X-301", "X-302", "X-303", "X-304", "X-305", "X-306", "X-307", "X-308", "X-309",
	"X-311", "X-312", "X-313", "X-314", "X-321", "X-322", "X-323", "X-324", "X-325",
	"X-331", "X-332", "X-333", "X-334", "X-335", "X-336", "X-337", "X-338",
	"X-341", "X-342", "X-343", "X-344", "X-345", "X-346", "X-351", "X-361", "X-362",
	"X-365", "X-366", "X-381", "X-382", "X-383", "X-384", "X-385", "X-386", "X-387", "X-388"}

// xIf is an interface with the addresses written on it.
type xIf struct {
	row    *model.Row
	device string
	name   string
	vrf    string
	kind   string
	mtu    string // the IP MTU if there is one, or else the MTU
	addrs  []netip.Prefix
	// addrs6 are the fixed IPv6 addresses (2026-11-21): they only find the
	// owner of an address (has), the far end of a tunnel over IPv6.
	addrs6 []netip.Prefix
}

type cross struct {
	*values
	ifs   []*xIf
	byDev map[string]map[string]*xIf
}

// Cross runs the checks of the pairs. The values are those of in.Filled: the
// design data with the defaults of each device's vendor pack. A value that
// is empty on one side even so is not compared.
func Cross(in Input, tally *Tally) []Finding {
	if in.Filled != nil {
		in.Data = in.Filled
	}
	c := &cross{values: &values{in: in, tally: tally}, byDev: map[string]map[string]*xIf{}}
	for _, id := range append(append(append([]string{}, crossIDs...), l2tpCrossIDs...), basicsCrossIDs...) {
		tally.Look(id, "", 0)
	}
	for _, r := range c.rows("interfaces") {
		i := &xIf{row: r, device: r.Get("device"), name: r.Get("name"), vrf: r.Get("vrf"), kind: r.Get("kind"), mtu: r.Get("ip_mtu")}
		if i.mtu == "" {
			i.mtu = r.Get("mtu")
		}
		for _, col := range []string{"ipv4", "ipv4_secondary"} {
			if cell := r.Cells[col]; cell != nil {
				for _, v := range cell.Values {
					if p, err := netip.ParsePrefix(v); err == nil && p.Addr().Is4() {
						i.addrs = append(i.addrs, p)
					}
				}
			}
		}
		if cell := r.Cells["ipv6"]; cell != nil {
			for _, v := range cell.Values {
				if p, err := netip.ParsePrefix(v); err == nil && p.Addr().Is6() {
					i.addrs6 = append(i.addrs6, p)
				}
			}
		}
		c.ifs = append(c.ifs, i)
		if c.byDev[i.device] == nil {
			c.byDev[i.device] = map[string]*xIf{}
		}
		c.byDev[i.device][i.name] = i
	}
	c.bgp()
	c.ospf()
	c.fhrp()
	c.fhrp6()
	c.subnets()
	c.tunnels(c.ipsec())
	c.l2tpv3Pairs()
	c.l2tpPairs()
	c.etherIPPairs()
	c.redistLoops()
	c.l2()
	c.ipv6()
	c.l3()
	return c.out
}

// shared gives the subnet two interfaces of different devices have in
// common: the same VRF name and the same network.
func shared(a, b *xIf) (netip.Prefix, bool) {
	if a == nil || b == nil || a.device == b.device || a.vrf != b.vrf {
		return netip.Prefix{}, false
	}
	for _, pa := range a.addrs {
		for _, pb := range b.addrs {
			if pa.Masked() == pb.Masked() {
				return pa.Masked(), true
			}
		}
	}
	return netip.Prefix{}, false
}

// others gives the interfaces of other devices in a subnet of an interface.
func (c *cross) others(a *xIf) []*xIf {
	var out []*xIf
	for _, b := range c.ifs {
		if _, ok := shared(a, b); ok {
			out = append(out, b)
		}
	}
	return out
}

// has tells whether an address is written on an interface.
func (i *xIf) has(addr netip.Addr) bool {
	for _, p := range i.addrs {
		if p.Addr() == addr {
			return true
		}
	}
	for _, p := range i.addrs6 {
		if p.Addr() == addr {
			return true
		}
	}
	return false
}

// owner gives the interface of another device that has an address; one of
// the same VRF name comes first.
func (c *cross) owner(addr netip.Addr, vrf, not string) *xIf {
	var found *xIf
	for _, i := range c.ifs {
		if i.device == not || !i.has(addr) {
			continue
		}
		if i.vrf == vrf {
			return i
		}
		if found == nil {
			found = i
		}
	}
	return found
}

func where(dev, what string) string { return dev + " " + what }

// both tells that two values are there and differ.
func both(a, b string) bool { return a != "" && b != "" && a != b }

// ---- BGP ----

func (c *cross) bgp() {
	t := c.in.Schema.Table("bgp_neighbors")
	if t == nil {
		return
	}
	rows := c.rows("bgp_neighbors")
	index := map[*model.Row]int{}
	for i, r := range rows {
		index[r] = i
	}
	for _, r := range rows {
		dev, vrf := r.Get("device"), r.Get("vrf")
		addr, err := netip.ParseAddr(r.Get("neighbor"))
		if err != nil {
			continue
		}
		c.tally.Look("X-300", dev, 1)
		peer := c.owner(addr, vrf, dev)
		if peer == nil {
			c.add("X-300", t, r, "neighbor", fmt.Sprintf("BGP ネイバー %s を持つ機器が、案件の中にありません（管理外の相手）", addr), model.KV{Name: "ネイバー", Value: addr.String()})
			continue
		}
		c.tally.Look("X-321", dev, 1)
		c.tally.Look("X-322", dev, 1)
		asn := ""
		if p := c.find("bgp_processes", peer.device, "vrf", peer.vrf); p != nil {
			asn = p.Get("asn")
		}
		if asn == "" {
			c.add("X-322", t, r, "neighbor", fmt.Sprintf("相手の %s（%s）に、BGP の行がありません", peer.device, peer.name),
				model.KV{Name: "ネイバー", Value: addr.String()}, model.KV{Name: "相手", Value: where(peer.device, peer.name)})
			continue
		}
		if ras := r.Get("remote_as"); ras != "" && ras != asn {
			c.add("X-321", t, r, "remote_as", fmt.Sprintf("相手 AS 番号 %s が、相手の %s の AS 番号 %s と違います", ras, peer.device, asn),
				model.KV{Name: where(dev, "の相手 AS 番号"), Value: ras}, model.KV{Name: where(peer.device, "の AS 番号"), Value: asn})
		}
		// The row of the other side that points back to this device: at an
		// address on the same link, or at the one a side sends from.
		var rev *model.Row
		for _, o := range rows {
			if o.Get("device") != peer.device {
				continue
			}
			back, err := netip.ParseAddr(o.Get("neighbor"))
			if err != nil {
				continue
			}
			for _, i := range c.ifs {
				if i.device != dev || !i.has(back) {
					continue
				}
				_, link := shared(i, peer)
				mine := c.byDev[dev][r.Get("update_source")]
				theirs := c.byDev[peer.device][o.Get("update_source")]
				if link || (mine != nil && mine.has(back)) || (theirs != nil && theirs.has(addr)) {
					rev = o
				}
			}
			if rev != nil {
				break
			}
		}
		if rev == nil {
			c.add("X-322", t, r, "neighbor", fmt.Sprintf("相手の %s に、この機器を指すネイバーの行がありません", peer.device),
				model.KV{Name: "ネイバー", Value: addr.String()}, model.KV{Name: "相手", Value: where(peer.device, peer.name)})
			continue
		}
		c.tally.Look("X-324", dev, 1)
		back, _ := netip.ParseAddr(rev.Get("neighbor"))
		if src := r.Get("update_source"); src != "" {
			if i := c.byDev[dev][src]; i != nil && len(i.addrs) > 0 && !i.has(back) {
				c.add("X-324", t, r, "update_source", fmt.Sprintf("送信元 IF %s のアドレスが、相手の %s が指すアドレス %s と違います", src, peer.device, back),
					model.KV{Name: where(dev, src), Value: i.addrs[0].Addr().String()}, model.KV{Name: where(peer.device, "のネイバー"), Value: back.String()})
			}
		} else if out := c.owner(back, vrf, peer.device); out != nil {
			// Without a source the session leaves from the interface of the
			// link: the other side has to point there.
			if _, link := shared(out, peer); !link {
				c.add("X-324", t, r, "update_source", fmt.Sprintf("相手の %s は %s（%s）を指していますが、送信元 IF の指定がありません", peer.device, back, out.name),
					model.KV{Name: where(peer.device, "のネイバー"), Value: back.String()})
			}
		}
		c.bgpCommunities(r, rev)
		if index[r] > index[rev] {
			continue // the pair was looked at from the other side
		}
		c.tally.Look("X-323", dev, 1)
		c.tally.Look("X-325", dev, 1)
		pair := []model.KV{{Name: "こちら", Value: where(dev, "→ "+addr.String())}, {Name: "相手", Value: where(peer.device, "→ "+rev.Get("neighbor"))}}
		if a, b := r.Get("password"), rev.Get("password"); a != b {
			msg := "パスワードが両端で違います（不一致）"
			if a == "" || b == "" {
				msg = "パスワードが片側にだけあります"
			}
			c.add("X-323", t, r, "password", msg, pair...)
		}
		if a, b := r.Get("ebgp_multihop"), rev.Get("ebgp_multihop"); (a == "") != (b == "") {
			c.add("X-324", t, r, "ebgp_multihop", "eBGP マルチホップの指定が片側にだけあります",
				model.KV{Name: dev, Value: a}, model.KV{Name: peer.device, Value: b})
		}
		for _, col := range []string{"keepalive", "holdtime"} {
			if a, b := r.Get(col), rev.Get(col); both(a, b) {
				c.add("X-325", t, r, col, fmt.Sprintf("「%s」が両端で違います", t.Column(col).Title.JA),
					model.KV{Name: dev, Value: a}, model.KV{Name: peer.device, Value: b})
			}
		}
	}
}

// ---- OSPF ----

type ospfIf struct {
	row  *model.Row
	i    *xIf
	proc *model.Row
}

// area brings an area to the dotted form: "0" and "0.0.0.0" are one area.
func area(v string) string {
	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return v
	}
	return netip.AddrFrom4([4]byte{byte(n >> 24), byte(n >> 16), byte(n >> 8), byte(n)}).String()
}

func (o ospfIf) passive() bool {
	if v := o.row.Get("passive"); v != "" {
		return v == "true"
	}
	return o.proc != nil && o.proc.Get("passive_default") == "true"
}

// network gives the kind of network to compare: an empty cell on an
// Ethernet interface is "broadcast"; elsewhere it is not known.
func (o ospfIf) network() string {
	if v := o.row.Get("network_type"); v != "" {
		return v
	}
	switch o.i.kind {
	case "physical", "subif", "vlan", "lag":
		return "broadcast"
	}
	return ""
}

func auth(v string) string {
	if v == "" {
		return "none"
	}
	return v
}

func (c *cross) ospf() {
	t := c.in.Schema.Table("ospf_interfaces")
	if t == nil {
		return
	}
	var list []ospfIf
	member := map[*xIf]bool{}
	for _, r := range c.rows("ospf_interfaces") {
		i := c.byDev[r.Get("device")][r.Get("if_name")]
		if i == nil || len(i.addrs) == 0 {
			continue
		}
		list = append(list, ospfIf{r, i, c.find("ospf_processes", i.device, "process_id", r.Get("process_id"))})
		member[i] = true
	}
	areaType := func(o ospfIf) string {
		for _, a := range c.rows("ospf_areas") {
			if a.Get("device") == o.i.device && a.Get("process_id") == o.row.Get("process_id") && area(a.Get("area")) == area(o.row.Get("area")) && a.Get("type") != "" {
				return a.Get("type")
			}
		}
		return "normal"
	}
	for n, a := range list {
		dev := a.i.device
		near := c.others(a.i)
		c.tally.Look("X-300", dev, 1)
		c.tally.Look("X-338", dev, 1)
		if len(near) == 0 {
			if !a.passive() && a.i.addrs[0].Bits() < 32 {
				c.add("X-300", t, a.row, "if_name", fmt.Sprintf("%s と同じサブネットに、案件のほかの機器の IF がありません（管理外の相手）", a.i.name),
					model.KV{Name: a.i.name, Value: a.i.addrs[0].Masked().String()})
			}
			continue
		}
		for _, o := range near {
			if !member[o] {
				c.add("X-338", t, a.row, "if_name", fmt.Sprintf("同じサブネットの %s の %s は、OSPF に入っていません", o.device, o.name),
					model.KV{Name: where(dev, a.i.name), Value: "OSPF " + a.row.Get("process_id")}, model.KV{Name: where(o.device, o.name), Value: "OSPF なし"})
			}
		}
		for _, b := range list[n+1:] {
			if _, ok := shared(a.i, b.i); !ok {
				continue
			}
			for _, id := range []string{"X-331", "X-332", "X-333", "X-334", "X-335", "X-336", "X-337"} {
				c.tally.Look(id, dev, 1)
			}
			wa, wb := where(dev, a.i.name), where(b.i.device, b.i.name)
			diff := func(rule, col, msg, va, vb string) {
				c.add(rule, t, a.row, col, msg, model.KV{Name: wa, Value: va}, model.KV{Name: wb, Value: vb})
			}
			if x, y := area(a.row.Get("area")), area(b.row.Get("area")); both(x, y) {
				diff("X-331", "area", fmt.Sprintf("同じサブネットの %s とエリアが違います", wb), a.row.Get("area"), b.row.Get("area"))
			} else if x == y {
				if x, y := areaType(a), areaType(b); x != y {
					diff("X-335", "area", fmt.Sprintf("エリア %s の種類が、%s と違います", a.row.Get("area"), b.i.device), x, y)
				}
			}
			for _, col := range []string{"hello", "dead"} {
				if x, y := a.row.Get(col), b.row.Get(col); both(x, y) {
					diff("X-332", col, fmt.Sprintf("「%s」が、同じサブネットの %s と違います", t.Column(col).Title.JA, wb), x, y)
				}
			}
			if x, y := a.network(), b.network(); both(x, y) {
				diff("X-333", "network_type", fmt.Sprintf("網の種類が、同じサブネットの %s と違います", wb), x, y)
			}
			switch x, y := auth(a.row.Get("auth_type")), auth(b.row.Get("auth_type")); {
			case x != y:
				diff("X-334", "auth_type", fmt.Sprintf("認証の方式が、同じサブネットの %s と違います", wb), x, y)
			case both(a.row.Get("auth_key_id"), b.row.Get("auth_key_id")):
				diff("X-334", "auth_key_id", fmt.Sprintf("認証の鍵番号が、同じサブネットの %s と違います", wb), a.row.Get("auth_key_id"), b.row.Get("auth_key_id"))
			case x != "none" && a.row.Get("auth_key") != b.row.Get("auth_key"):
				c.add("X-334", t, a.row, "auth_key", fmt.Sprintf("認証の鍵が、同じサブネットの %s と違います（不一致）", wb))
			}
			if both(a.i.mtu, b.i.mtu) {
				diff("X-336", "if_name", fmt.Sprintf("IF の MTU が、同じサブネットの %s と違います", wb), a.i.mtu, b.i.mtu)
			}
			if a.passive() != b.passive() {
				diff("X-337", "passive", fmt.Sprintf("同じサブネットの %s と、片側だけ受動になっています", wb), strconv.FormatBool(a.passive()), strconv.FormatBool(b.passive()))
			}
		}
	}
}

// ---- VRRP and HSRP ----

type group struct {
	row    *model.Row
	i      *xIf
	subnet netip.Prefix // the subnet of the interface the virtual address lies in
}

func (c *cross) fhrp() {
	t := c.in.Schema.Table("fhrp")
	if t == nil {
		return
	}
	var list []group
	for _, r := range c.rows("fhrp") {
		i := c.byDev[r.Get("device")][r.Get("if_name")]
		if i == nil || len(i.addrs) == 0 {
			continue
		}
		g := group{r, i, i.addrs[0].Masked()}
		if vip, err := netip.ParseAddr(r.Get("virtual_ip")); err == nil {
			for _, p := range i.addrs {
				if p.Masked().Contains(vip) {
					g.subnet = p.Masked()
					break
				}
			}
		}
		list = append(list, g)
	}
	twin := func(a, b group) bool {
		return a.i.device != b.i.device && a.i.vrf == b.i.vrf && a.subnet == b.subnet && a.row.Get("group") == b.row.Get("group")
	}
	for n, a := range list {
		dev, no := a.i.device, a.row.Get("group")
		c.tally.Look("X-300", dev, 1)
		c.tally.Look("X-346", dev, 1)
		alone := true
		for _, b := range list {
			alone = alone && !twin(a, b)
		}
		if alone {
			var near []*xIf
			for _, o := range c.ifs {
				if o.device == dev || o.vrf != a.i.vrf {
					continue
				}
				for _, p := range o.addrs {
					if p.Masked() == a.subnet {
						near = append(near, o)
						break
					}
				}
			}
			if len(near) == 0 {
				c.add("X-300", t, a.row, "group", fmt.Sprintf("%s のグループ %s の相手が、案件の中にありません（同じサブネットに、ほかの機器の IF がありません。管理外の相手）", a.i.name, no),
					model.KV{Name: a.i.name, Value: a.subnet.String()})
			} else {
				c.add("X-346", t, a.row, "group", fmt.Sprintf("グループ %s に機器が 1 台しかありません（同じサブネットの %s の %s に、同じグループ番号の行がありません）", no, near[0].device, near[0].name),
					model.KV{Name: where(dev, a.i.name), Value: "グループ " + no}, model.KV{Name: where(near[0].device, near[0].name), Value: "行なし"})
			}
			continue
		}
		for _, b := range list[n+1:] {
			if !twin(a, b) {
				continue
			}
			for _, id := range []string{"X-341", "X-342", "X-343", "X-344", "X-345"} {
				c.tally.Look(id, dev, 1)
			}
			wa, wb := where(dev, a.i.name), where(b.i.device, b.i.name)
			diff := func(rule, col, msg string) {
				c.add(rule, t, a.row, col, msg, model.KV{Name: wa, Value: a.row.Get(col)}, model.KV{Name: wb, Value: b.row.Get(col)})
			}
			get := func(col string) (string, string) { return a.row.Get(col), b.row.Get(col) }
			if x, y := get("virtual_ip"); both(x, y) {
				diff("X-341", "virtual_ip", fmt.Sprintf("グループ %s の仮想 IP が、%s と違います", no, wb))
			}
			if x, y := get("protocol"); both(x, y) {
				diff("X-342", "protocol", fmt.Sprintf("グループ %s の方式が、%s と違います", no, wb))
			} else if x, y := get("version"); both(x, y) {
				diff("X-342", "version", fmt.Sprintf("グループ %s の版が、%s と違います", no, wb))
			}
			c.tally.Look("X-405", dev, 1)
			if v3, v2, ok := c.versionClash(a.row, b.row); ok {
				c.add("X-405", t, a.row, "protocol", fmt.Sprintf("グループ %s: %s は VRRP の版 3 だけで動き（「VRRP の版（IPv4）」が 3 で、「版 2 の相手とも話す」が有効でない）、%s は版 2 です。両方の機器がそろっても、版が違うのでつながりません",
					no, where(v3.Get("device"), v3.Get("if_name")), where(v2.Get("device"), v2.Get("if_name"))),
					model.KV{Name: where(v3.Get("device"), v3.Get("if_name")), Value: "版 3"}, model.KV{Name: where(v2.Get("device"), v2.Get("if_name")), Value: "版 2"})
			}
			if x, y := get("priority"); x != "" && x == y {
				diff("X-343", "priority", fmt.Sprintf("グループ %s の優先度が、%s と同じです（どちらが主になるかが決まりません）", no, wb))
			}
			switch x, y := auth(a.row.Get("auth_type")), auth(b.row.Get("auth_type")); {
			case x != y:
				c.add("X-344", t, a.row, "auth_type", fmt.Sprintf("グループ %s の認証の方式が、%s と違います", no, wb), model.KV{Name: wa, Value: x}, model.KV{Name: wb, Value: y})
			case x != "none" && a.row.Get("auth_key") != b.row.Get("auth_key"):
				c.add("X-344", t, a.row, "auth_key", fmt.Sprintf("グループ %s の認証の鍵が、%s と違います（不一致）", no, wb))
			}
			for _, col := range []string{"advert_interval", "hold"} {
				if x, y := get(col); both(x, y) {
					diff("X-345", col, fmt.Sprintf("グループ %s の「%s」が、%s と違います", no, t.Column(col).Title.JA, wb))
				}
			}
		}
	}
}

// ---- subnets ----

// subnets looks at the interfaces of different devices that share a network
// (X-351): the length of the prefix and the MTU. The MTU of two interfaces
// that both run OSPF is left to X-336.
func (c *cross) subnets() {
	t := c.in.Schema.Table("interfaces")
	if t == nil {
		return
	}
	ospf := map[*xIf]bool{}
	for _, r := range c.rows("ospf_interfaces") {
		if i := c.byDev[r.Get("device")][r.Get("if_name")]; i != nil {
			ospf[i] = true
		}
	}
	for n, a := range c.ifs {
		if len(a.addrs) == 0 {
			continue
		}
		c.tally.Look("X-351", a.device, 1)
		for _, b := range c.ifs[n+1:] {
			if b.device == a.device || b.vrf != a.vrf || len(b.addrs) == 0 {
				continue
			}
			pa, pb := a.addrs[0], b.addrs[0]
			wa, wb := where(a.device, a.name), where(b.device, b.name)
			switch {
			case pa.Masked() == pb.Masked():
				if both(a.mtu, b.mtu) && !(ospf[a] && ospf[b]) {
					col := "ip_mtu"
					if a.row.Get(col) == "" {
						col = "mtu"
					}
					c.add("X-351", t, a.row, col, fmt.Sprintf("MTU が、同じサブネットの %s と違います", wb),
						model.KV{Name: wa, Value: a.mtu}, model.KV{Name: wb, Value: b.mtu})
				}
			case pa.Bits() < 32 && pb.Bits() < 32 && pa.Masked().Overlaps(pb.Masked()):
				c.add("X-351", t, a.row, "ipv4", fmt.Sprintf("プレフィックス長が、同じネットワークの %s と違います", wb),
					model.KV{Name: wa, Value: pa.String()}, model.KV{Name: wb, Value: pb.String()})
			}
		}
	}
}
